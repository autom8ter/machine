package machine_test

import (
	"context"
	"encoding/json"
	"github.com/autom8ter/machine"
	"testing"
	"time"
)

func Test(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()
	m, err := machine.New(ctx, &machine.Opts{
		MaxRoutines: 100,
		Debug:       true,
	})
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
	bits, _ := json.MarshalIndent(&stats, "", "    ")
	t.Logf("stats = %v\n", string(bits))
	if errs := m.Wait(); len(errs) > 0 {
		for _, err := range errs {
			t.Logf("workerPool error: %s", err)
		}
	}
	select {
	case <-ctx.Done():
		 break
	}
	if m.Current() != 0 {
		t.Fatalf("expected current to be zero")
	}
}

func TestPubSub(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()
	m, err := machine.New(ctx, &machine.Opts{
		MaxRoutines: 100,
		Debug:       true,
	})
	if err != nil {
		t.Fatalf(err.Error())
	}
	channelName := "acme.com"
	var seen = false
	m.Go(func(routine machine.Routine) error {
		channel := routine.Subscribe(channelName)
		for {
			select {
			case <-routine.Context().Done():
				return nil
			case msg := <-channel:
				seen = true
				t.Logf("subscription msg received! channel = %v msg = %v\n", channelName, msg)
			}
		}
	})
	published := "hello there"
	m.Go(func(routine machine.Routine) error {
		ticker := time.NewTicker(1 *time.Second)
		for {
			select {
			case <-routine.Context().Done():
				return nil
			case <-ticker.C:
				t.Logf("publishing message to channel = %v msg = %v\n", channelName, published)
				routine.Publish(channelName, published)
			}
		}
	})
	if errs := m.Wait(); len(errs) > 0 {
		for _, err := range errs {
			t.Fatalf(err.Error())
		}
	}
	if !seen {
		t.Fatal("expected to have received a subscription msg")
	}
}
