package machine_test

import (
	"context"
	"github.com/autom8ter/machine"
	"testing"
	"time"
)

func runTest(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer cancel()
	m := machine.New(ctx,
		machine.WithMaxRoutines(10),
	)
	defer m.Close()
	channelName := "acme.com"
	var seen = false
	m.Go(func(routine machine.Routine) {
		routine.Subscribe(channelName, func(obj interface{}) {
			seen = true
			t.Logf("subscription msg received! channel = %v msg = %v stats= %s\n", channelName, obj, m.Stats().String())
		})
	}, machine.GoWithTags("subscribe"))
	m.Go(func(routine machine.Routine) {
		msg := "hey there bud!"
		t.Logf("streaming msg to channel = %v msg = %v stats= %s\n", channelName, msg, routine.Machine().Stats().String())
		routine.Publish(channelName, msg)
	},
		machine.GoWithTags("publish"),
		machine.GoWithMiddlewares(
			machine.Cron(time.NewTicker(1*time.Second)),
		),
	)
	m2 := m.Sub(machine.WithMaxRoutines(3))
	var seenCron = false

	m2.Go(func(routine machine.Routine) {
		seenCron = true
		t.Logf("cron1 stats= %s\n", routine.Machine().Stats().String())
	},
		machine.GoWithTags("cron1"),
		machine.GoWithTimeout(3*time.Second),
		machine.GoWithMiddlewares(
			machine.Cron(time.NewTicker(1*time.Second)),
		),
	)
	m2.Go(func(routine machine.Routine) {
		seenCron = true
		t.Logf("cron2 stats= %s\n", routine.Machine().Stats().String())
	},
		machine.GoWithTags("cron2"),
		machine.GoWithTimeout(3*time.Second),
		machine.GoWithMiddlewares(
			machine.Cron(time.NewTicker(1*time.Second)),
		),
	)
	m2.Go(func(routine machine.Routine) {
		seenCron = true
		t.Logf("cron3 stats= %s\n", routine.Machine().Stats().String())
	},
		machine.GoWithTags("cron3"),
		machine.GoWithTimeout(3*time.Second),
		machine.GoWithMiddlewares(
			machine.Cron(time.NewTicker(1*time.Second)),
		),
	)
	m2.Wait()
	t.Logf("here")
	m.Go(func(routine machine.Routine) {
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

func Test(t *testing.T) {
	runTest(t)
}

/*
MC02CG684LVDL:machine Coleman.Word$ go test -bench=.
goos: darwin
goarch: amd64
pkg: github.com/autom8ter/machine
Benchmark-8       860584              1366 ns/op             272 B/op          5 allocs/op
*/
func Benchmark(b *testing.B) {
	b.ReportAllocs()
	m := machine.New(context.Background(), machine.WithMaxRoutines(3))
	defer m.Close()
	for n := 0; n < b.N; n++ {
		m.Go(func(routine machine.Routine) {
			return
		})
	}
	m.Wait()
}
