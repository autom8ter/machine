package prometheus_test

import (
	"context"
	"fmt"
	"github.com/autom8ter/machine"
	prom "github.com/autom8ter/machine/external/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"testing"
	"time"
)

func Test(t *testing.T) {
	m := machine.New(
		context.Background(),
		machine.WithTimeout(5*time.Second),
		machine.WithMiddlewares(prom.Middleware()),
	)
	for x := 0; x < 100; x++ {
		i := x
		m.Go(func(routine machine.Routine) {
			t.Log("here")
		}, machine.GoWithPID(fmt.Sprint(i)))
	}
	if err := prometheus.WriteToTextfile("testing.metrics", prometheus.DefaultGatherer); err != nil {
		t.Fatal(err)
	}
}
