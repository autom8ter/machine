package machine_test

import (
	"context"
	"fmt"
	"github.com/autom8ter/machine/v2"
	"testing"
	"time"
)

func Test(t *testing.T) {
	var (
		m = machine.New(machine.WithErrHandler(func(err error) {
			t.Fatal(err)
		}))
		count = 0
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
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
	var published = 0
	m.Cron(ctx, 250*time.Millisecond, func(ctx context.Context) (bool, error) {
		m.Publish(ctx, machine.Msg{
			Channel: fmt.Sprintf("testing.%v", published),
			Body:    "hello world",
		})
		published++
		return published < 3, nil
	})
	m.Wait()
	if published < 3 {
		t.Fatal("published < 3")
	}
	if count < 3 {
		t.Fatal("count < 3")
	}

}
