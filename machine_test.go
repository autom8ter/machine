package machine_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/autom8ter/machine/v3"
)

func Test(t *testing.T) {
	var (
		m     = machine.New()
		count = 0
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	defer m.Close()
	m.Go(ctx, func(ctx context.Context) error {
		return m.Subscribe(ctx, "testing.*", func(ctx context.Context, msg machine.Message) (bool, error) {
			t.Logf("(%s) got message: %v", msg.Channel, msg.Body)
			if !strings.Contains(msg.Channel, "testing") {
				t.Fatal("expected channel to contain 'testing'")
			}
			count++
			return count < 3, nil
		})
	})
	time.Sleep(1 * time.Second)
	m.Wait()
	if count < 3 {
		t.Fatal("count < 3")
	}
	if channels := len(m.Subscriptions()); channels != 1 {
		t.Fatalf("expected 1 total channel, got %v", m.Subscriptions())
	}
	if subscribers := m.Subscribers("testing.*"); subscribers != 0 {
		t.Fatalf("expected 0 total subscriber, got %v", subscribers)
	}

}

func TestWithThrottledRoutines(t *testing.T) {
	max := 3
	m := machine.New(machine.WithThrottledRoutines(max))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	defer m.Close()
	for i := 0; i < 100; i++ {
		i := i
		m.Go(ctx, func(ctx context.Context) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if current := m.Current(); current > max {
				t.Fatalf("more routines running %v than max threshold: %v", current, max)
			}
			t.Logf("(%v)", i)
			time.Sleep(50 * time.Millisecond)
			return nil
		})
	}
	m.Wait()
}

func Benchmark(b *testing.B) {
	b.Run("publish", func(b *testing.B) {
		m := machine.New()
		go func() {
			m.Subscribe(context.Background(), "testing.*", func(ctx context.Context, _ machine.Message) (bool, error) {
				return true, nil
			})
		}()
		for i := 0; i < b.N; i++ {
			m.Publish(context.Background(), machine.Message{
				Channel: "testing",
				Body:    i,
			})
		}
	})
}
