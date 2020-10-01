package machine_test

import (
	"context"
	"github.com/autom8ter/machine"
	"testing"
	"time"
)

func Test(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()
	m, err := machine.New(ctx, machine.WithMaxRoutines(100))
	if err != nil {
		t.Fatalf(err.Error())
	}

	for x := 0; x < 1000; x++ {
		m.Go(func(routine machine.Routine) error {
			time.Sleep(200 * time.Millisecond)
			return nil
		})
	}
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
}

func TestPubSub(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()
	m, err := machine.New(ctx, machine.WithMaxRoutines(100))
	if err != nil {
		t.Fatalf(err.Error())
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
	if errs := m.Wait(); len(errs) > 0 {
		for _, err := range errs {
			t.Fatalf(err.Error())
		}
	}
	if !seen {
		t.Fatal("expected to have received a subscription msg")
	}
}
