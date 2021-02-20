package machine_test

import (
	"context"
	"github.com/autom8ter/machine/v2"
	"testing"
	"time"
)

func Test(t *testing.T) {
	var (
		m     = machine.New(machine.WithErrHandler(func(err error) {
			t.Fatal(err)
		}))
		ctx   = context.Background()
		count = 0
	)
	defer m.Close()
	m.Go(ctx, func(ctx context.Context) error {
		m.Subscribe(ctx, "testing.*", func(ctx context.Context, msg machine.Message) (bool, error) {
			t.Logf("(%s) got message: %v", msg.GetChannel(), msg.GetBody())
			count++
			return count < 3, nil
		})
		return nil
	})
	time.Sleep(1 * time.Second)
	m.Publish(ctx, machine.Msg{
		Channel: "testing.0",
		Body:    "hello world",
	})
	m.Publish(ctx, machine.Msg{
		Channel: "testing.1",
		Body:    "hello world",
	})
	m.Publish(ctx, machine.Msg{
		Channel: "testing.2",
		Body:    "hello world",
	})
	m.Wait()
}
