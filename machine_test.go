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
	m, err := machine.New(ctx,
		machine.WithMaxRoutines(3),
		machine.WithPublishChannelBuffer(10),
		machine.WithSubscribeChannelBuffer(10),
	)
	if err != nil {
		t.Fatalf(err.Error())
	}

	for x := 0; x < 100; x++ {
		m.Go(func(routine machine.Routine) error {
			time.Sleep(50 * time.Millisecond)
			return nil
		})
	}
	channelName := "acme.com"
	var seen = false
	m.Go(func(routine machine.Routine) error {
		channel := routine.SubscribeTo(channelName)
		for {
			select {
			case <-routine.Context().Done():
				close(channel)
				return nil
			case msg := <-channel:
				seen = true
				t.Logf("subscription msg received! channel = %v msg = %v stats= %s\n", channelName, msg, m.Stats().String())
			}
		}
	}, "subscribe")
	m.Go(func(routine machine.Routine) error {
		channel := routine.PublishTo(channelName)
		tick := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-routine.Context().Done():
				tick.Stop()
				close(channel)
				return nil
			case <-tick.C:
				msg := "hey there bud!"
				t.Logf("streaming msg to channel = %v msg = %v stats= %s\n", channelName, msg, m.Stats().String())
				channel <- msg
				time.Sleep(1 * time.Second)
			}
		}
	}, "publish")
	time.Sleep(1 * time.Second)
	stats := m.Stats()
	t.Logf("stats = %s\n", stats)
	if errs := m.Wait(); len(errs) > 0 {
		for _, err := range errs {
			t.Logf("workerPool error: %s", err)
		}
	}
	if m.Current() != 0 {
		t.Fatalf("expected current to be zero")
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
Benchmark-8       555478              2163 ns/op
*/
func Benchmark(b *testing.B) {
	for n := 0; n < b.N; n++ {
		m, err := machine.New(context.Background(), machine.WithMaxRoutines(5))
		if err != nil {
			b.Fatal(err.Error())
		}
		m.Go(func(routine machine.Routine) error {
			return nil
		})
	}
}
