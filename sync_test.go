package sync_test

import (
	"context"
	"gitlab.com/ftdr/sync"
	"testing"
	"time"
)

func Test(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10 *time.Second))
	defer cancel()
	workerPool := sync.NewWorkerPool(ctx, 5)
	for x := 0; x < 10; x++ {
		y := x
		workerPool.Go(func(ctx context.Context) error {
			t.Logf("current: %v val: %v", workerPool.Current(), y)
			time.Sleep(1 *time.Second)
			return nil
		})
	}
	t.Logf("current: %v", workerPool.Current())
	if errs := workerPool.Wait(); len(errs) > 0 {
		for _, err := range errs {
			t.Logf("workerPool error: %s", err)
		}
	}
	t.Logf("after: %v", workerPool.Current())

}
