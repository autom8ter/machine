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
}

func Benchmark(b *testing.B) {
	b.Run("empty routine", func(b *testing.B) {
		/*
			MC02CG684LVDL:machine Coleman.Word$ go test -bench=.
			goos: darwin
			goarch: amd64
			pkg: github.com/autom8ter/machine
			Benchmark-8       860584              1366 ns/op             272 B/op          5 allocs/op
		*/
		benchmarkEmpty(b)
	})
}

func benchmarkEmpty(b *testing.B) {
	b.ReportAllocs()
	m := New(context.Background(), WithMaxRoutines(3))
	defer m.Close()
	for n := 0; n < b.N; n++ {
		m.Go(func(routine Routine) {
			return
		})
	}
	m.Wait()
}

func runE2ETest(t *testing.T) {
	t.Parallel()
	m := New(context.Background(),
		WithMaxRoutines(10),
		WithMiddlewares(PanicRecover()),
		WithValues(map[interface{}]interface{}{
			"testing": true,
		}),
		WithDeadline(time.Now().Add(5*time.Second)),
	)
	defer m.Close()
	channelName := "acme.com"
	var seen = false
	m.Go(func(routine Routine) {
		if routine.Context().Value("testing").(bool) != true {
			t.Fatal("expected testing = true in context")
		}
		routine.Subscribe(channelName, func(obj interface{}) {

			seen = true
			t.Logf("subscription msg received! channel = %v msg = %v stats= %s\n", channelName, obj, m.Stats().String())
		})
	}, GoWithTags("subscribe"))
	m.Go(func(routine Routine) {
		if routine.Context().Value("testing").(bool) != true {
			t.Fatal("expected testing = true in context")
		}
		msg := "hey there bud!"
		t.Logf("streaming msg to channel = %v msg = %v stats= %s\n", channelName, msg, routine.Machine().Stats().String())
		routine.Publish(channelName, msg)
	},
		GoWithTags("publish"),
		GoWithTimeout(5*time.Second),
		GoWithMiddlewares(
			Cron(time.NewTicker(1*time.Second)),
		),
	)
	m2 := m.Sub(WithMaxRoutines(3))
	defer m2.Close()
	var seenCron = false

	m2.Go(func(routine Routine) {
		if routine.Context().Value("testing").(bool) != true {
			t.Fatal("expected testing = true in context")
		}
		seenCron = true
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
		seenCron = true
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
		seenCron = true
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
	if !seenCron {
		t.Fatalf("expected to have received cron msg")
	}
	if !seen {
		t.Fatalf("expected to have received subscription msg")
	}
	t.Logf("total = %v", m.Total())
}

func runStatsTest(t *testing.T) {
	t.Parallel()
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
