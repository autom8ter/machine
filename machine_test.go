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
			i := x
			t.Logf("id = %v current = %v\n", i, m.Current())
			time.Sleep(200 * time.Millisecond)
			t.Logf("duration = %v\n", routine.Duration())
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
	t.Logf("after: %v", m.Current())

}
