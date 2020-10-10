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
		machine.WithSubscribeChannelBuffer(10),
	)
	channelName := "acme.com"
	var seen = false
	m.Go(func(routine machine.Routine) {
		routine.Subscribe(channelName, func(obj interface{}) {
			seen = true
			t.Logf("subscription msg received! channel = %v msg = %v stats= %s\n", channelName, obj, m.Stats().String())
		})
	}, machine.WithTags("subscribe"))
	m.Go(func(routine machine.Routine) {
		tick := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-routine.Context().Done():
				tick.Stop()
				return
			case <-tick.C:
				msg := "hey there bud!"
				t.Logf("streaming msg to channel = %v msg = %v stats= %s\n", channelName, msg, m.Stats().String())
				routine.Publish(channelName, msg)
				time.Sleep(1 * time.Second)
			}
		}
	}, machine.WithTags("publish"))
	var seenCron = false

	m.Go(func(routine machine.Routine) {
		seenCron = true
		t.Logf("cron1")
	},
		machine.WithTags("cron1"),
		machine.WithMiddlewares(machine.Cron(time.NewTicker(1*time.Second))),
	)
	m.Go(func(routine machine.Routine) {
		seenCron = true
		t.Logf("cron2")
	},
		machine.WithTags("cron2"),
		machine.WithMiddlewares(machine.Cron(time.NewTicker(1*time.Second))),
	)
	m.Go(func(routine machine.Routine) {
		seenCron = true
		t.Logf("cron3")
	},
		machine.WithTags("cron3"),
		machine.WithMiddlewares(machine.Cron(time.NewTicker(1*time.Second))),
	)
	m.Go(func(routine machine.Routine) {
		panic("panic!")
	})
	m.Wait()
	if m.Current() != 0 {
		t.Fatalf("expected current to be zero, got: %v", m.Current())
	}
	if !seenCron {
		t.Fatalf("expected to have received cron msg")
	}
	if !seen {
		t.Fatalf("expected to have received subscription msg")
	}
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
	m := machine.New(context.Background(), machine.WithMaxRoutines(100))
	for n := 0; n < b.N; n++ {
		m.Go(func(routine machine.Routine) {
			return
		})
	}
	m.Wait()
}
