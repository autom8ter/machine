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
	workerPool := machine.New(ctx, 100)
	for x := 0; x < 10000; x++ {
		workerPool.Go(func(ctx context.Context) error {
			time.Sleep(200 * time.Millisecond)
			return nil
		})
	}
	if errs := workerPool.Wait(); len(errs) > 0 {
		for _, err := range errs {
			t.Logf("workerPool error: %s", err)
		}
	}
	t.Logf("after: %v", workerPool.Current())

}
