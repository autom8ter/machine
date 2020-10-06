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
		machine.WithMaxRoutines(3),
		machine.WithPublishChannelBuffer(10),
		machine.WithSubscribeChannelBuffer(10),
	)
	for x := 0; x < 100; x++ {
		m.Go(func(routine machine.Routine) {
			time.Sleep(50 * time.Millisecond)
			return
		})
	}
	channelName := "acme.com"
	var seen = false
	m.Go(func(routine machine.Routine) {
		channel := routine.SubscribeTo(channelName)
		for {
			select {
			case <-routine.Context().Done():
				return
			case msg := <-channel:
				seen = true
				t.Logf("subscription msg received! channel = %v msg = %v stats= %s\n", channelName, msg, m.Stats().String())
			}
		}
	}, machine.WithTags("subscribe"))
	m.Go(func(routine machine.Routine) {
		channel := routine.PublishTo(channelName)
		tick := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-routine.Context().Done():
				tick.Stop()
				return
			case <-tick.C:
				msg := "hey there bud!"
				t.Logf("streaming msg to channel = %v msg = %v stats= %s\n", channelName, msg, m.Stats().String())
				channel <- msg
				time.Sleep(1 * time.Second)
			}
		}
	}, machine.WithTags("publish"))
	time.Sleep(1 * time.Second)
	stats := m.Stats()
	t.Logf("stats = %s\n", stats)
	m.Wait()
	if m.Current() != 0 {
		t.Fatalf("expected current to be zero, got: %v", m.Current())
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
