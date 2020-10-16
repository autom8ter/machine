package machine

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func Test(t *testing.T) {
	t.Run("e2e", runE2ETest)
	t.Run("stats", runStatsTest)
	t.Run("cache", runCacheTest)
}

func Benchmark(b *testing.B) {
	b.Run("benchmarkGoSleep", func(b *testing.B) {
		benchmarkGoSleep(b, 100*time.Nanosecond)
	})
	b.Run("benchSetCache", benchSetCache)
}

//
func benchmarkGoSleep(b *testing.B, sleep time.Duration) {
	b.ReportAllocs()
	m := New(context.Background(), WithMaxRoutines(100))
	defer m.Close()
	for n := 0; n < b.N; n++ {
		m.Go(func(routine Routine) {
			time.Sleep(sleep)
			return
		})
	}
	m.Wait()
}

func runE2ETest(t *testing.T) {
	m := New(context.Background(),
		WithMaxRoutines(10),
		WithMiddlewares(PanicRecover()),
		WithValue("testing", true),
		WithDeadline(time.Now().Add(5*time.Second)),
		WithTags([]string{"root"}),
	)
	defer m.Close()
	channelName := "acme.com"
	var seen = false
	m.Go(func(routine Routine) {
		if routine.Context().Value("testing").(bool) != true {
			t.Fatal("expected testing = true in context")
		}
		if err := routine.Subscribe(channelName, func(obj interface{}) {
			seen = true
			t.Logf("subscription msg received! channel = %v msg = %v stats= %s\n", channelName, obj, m.Stats().String())
		}); err != nil {
			t.Fatal(err)
		}
	}, GoWithTags("subscribe"))
	m.Go(func(routine Routine) {
		if routine.Context().Value("testing").(bool) != true {
			t.Fatal("expected testing = true in context")
		}
		msg := "hey there bud!"
		t.Logf("streaming msg to channel = %v msg = %v stats= %s\n", channelName, msg, routine.Machine().Stats().String())
		if err := routine.Publish(channelName, msg); err != nil {
			t.Fatal(err)
		}
	},
		GoWithTags("publish"),
		GoWithTimeout(5*time.Second),
		GoWithMiddlewares(
			Cron(time.NewTicker(1*time.Second)),
		),
	)
	m2 := m.Sub(WithMaxRoutines(3))
	defer m2.Close()

	m2.Go(func(routine Routine) {
		if routine.Context().Value("testing").(bool) != true {
			t.Fatal("expected testing = true in context")
		}
		t.Logf("cron1 stats= %s\n", routine.Machine().Stats().String())
	},
		GoWithTags("cron1"),
		GoWithTimeout(3*time.Second),
		GoWithMiddlewares(
			Cron(time.NewTicker(1*time.Second)),
		),
	)
	m2.Go(func(routine Routine) {
		if routine.Context().Value("testing").(bool) != true {
			t.Fatal("expected testing = true in context")
		}
		t.Logf("cron2 stats= %s\n", routine.Machine().Stats().String())
	},
		GoWithTags("cron2"),
		GoWithTimeout(3*time.Second),
		GoWithMiddlewares(
			Cron(time.NewTicker(1*time.Second)),
		),
	)
	m2.Go(func(routine Routine) {
		if routine.Context().Value("testing").(bool) != true {
			t.Fatal("expected testing = true in context")
		}
		t.Logf("cron3 stats= %s\n", routine.Machine().Stats().String())
	},
		GoWithTags("cron3"),
		GoWithTimeout(3*time.Second),
		GoWithMiddlewares(
			Cron(time.NewTicker(1*time.Second)),
		),
	)
	m2.Wait()
	t.Logf("here")
	m.Go(func(routine Routine) {
		panic("panic!")
	})
	m.Wait()
	if m.Active() != 0 {
		t.Fatalf("expected active to be zero, got: %v", m.Active())
	}
	if !seen {
		t.Fatalf("expected to have received subscription msg")
	}
	t.Logf("total = %v", m.Total())
}

func runStatsTest(t *testing.T) {
	m := New(
		context.Background(),
		WithTimeout(3*time.Second),
	)
	defer m.Close()
	for x := 0; x < 100; x++ {
		m.Go(func(routine Routine) {
			fmt.Printf("cron pid = %v\n", routine.PID())
		},
			GoWithMiddlewares(Cron(time.NewTicker(100*time.Millisecond))),
			GoWithTags(fmt.Sprintf("x = %v", x)),
		)
	}
	time.Sleep(1 * time.Second)
	stats := m.Stats()
	if stats.ActiveRoutines != 100 {
		t.Fatalf("expected 100 active routines, got: %v\n", stats.ActiveRoutines)
	}
	m.Wait()

	total := m.Stats().TotalRoutines
	if total != 100 {
		t.Fatalf("expected 100 total routines, got: %v\n", total)
	}
}

func runCacheTest(t *testing.T) {
	m := New(context.Background())
	defer m.Close()
	m.Cache().Set("config", "env", "testing", 5*time.Second)
	val, ok := m.Cache().Get("config", "env")
	if !ok {
		t.Fatal("key not found")
	}
	if val != "testing" {
		t.Fatal("incorrect cache value")
	}
	m.Cache().Delete("config", "env")
	_, ok = m.Cache().Get("config", "env")
	if ok {
		t.Fatal("key found after deletion")
	}
	m.Cache().Set("config", "env", "testing", 500*time.Millisecond)
	time.Sleep(1 * time.Second)
	val, ok = m.Cache().Get("config", "env")
	if ok {
		t.Fatal("key found after expiration!")
	}
}

func benchSetCache(b *testing.B) {
	b.ReportAllocs()
	m := New(context.Background())
	defer m.Close()
	for n := 0; n < b.N; n++ {
		m.Go(func(routine Routine) {
			m.Cache().Set("testing", fmt.Sprintf("%v", time.Now().UnixNano()), 1, 5*time.Minute)
		})
	}
	b.Logf("active = %v\n", m.Active())
	b.Logf("cached items = %v\n", m.Cache().Len("testing"))
	m.Wait()
}
