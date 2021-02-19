package machine

import (
	"context"
	"fmt"
	"os"
	"runtime/pprof"
	"testing"
	"time"
)

func Test(t *testing.T) {
	t.Run("e2e", runE2ETest)
	t.Run("stats", runStatsTest)
}

func runE2ETest(t *testing.T) {
	cpu, err := os.Create("testing.cpu.prof")
	if err != nil {
		t.Fatal(err)
	}
	cpu.Truncate(0)
	defer cpu.Close()
	pprof.StartCPUProfile(cpu)
	defer pprof.StopCPUProfile()
	m := New(context.Background(),
		WithMaxRoutines(10),
		WithMiddlewares(PanicRecover()),
		WithValue("testing", true),
		WithDeadline(time.Now().Add(5*time.Second)),
		WithTags("root"),
	)
	defer m.Close()
	channelName := "acme.com"
	var seen = false
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
		GoWithPID("publisher"),
		GoWithTimeout(5*time.Second),
		GoWithMiddlewares(
			Cron(time.NewTicker(1*time.Second)),
		),
	)
	m.Go(func(routine Routine) {
		if routine.Context().Value("testing").(bool) != true {
			t.Fatal("expected testing = true in context")
		}
		if err := routine.Subscribe(channelName, func(obj interface{}) bool {
			seen = true

			t.Logf("subscription msg received! channel = %v msg = %v stats= %s\n", channelName, obj, m.Stats().String())
			return true
		}); err != nil {
			t.Fatal(err)
		}
	}, GoWithTags("subscribe"))
	// start a goroutine that subscribes to just the channel until the publishing goroutine exits
	m.Go(func(routine Routine) {
		routine.Subscribe(channelName, func(obj interface{}) bool {
			fmt.Printf("%v | subscriptionUntil msg received! channel = %v msg = %v stats = %s\n", routine.PID(), channelName, obj, m.Stats().String())
			return m.HasRoutine("publisher")
		})
	}, GoWithTags("subscribeUntil"),
		GoWithTimeout(5*time.Second),
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
