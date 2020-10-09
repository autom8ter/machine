package cron_test

import (
	"context"
	"github.com/autom8ter/machine"
	"github.com/autom8ter/machine/cron"
	"testing"
	"time"
)

func runTest(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer cancel()
	m := machine.New(ctx,
		machine.WithMaxRoutines(3),
	)
	var seen = false

	m.Go(cron.Every(time.NewTicker(1*time.Second), func(routine machine.Routine) {
		seen = true
		t.Logf("cron1")
	}), machine.WithTags("cron1"))
	m.Go(cron.Every(time.NewTicker(1*time.Second), func(routine machine.Routine) {
		seen = true
		t.Logf("cron2")
	}), machine.WithTags("cron2"))
	m.Go(cron.Every(time.NewTicker(1*time.Second), func(routine machine.Routine) {
		seen = true
		t.Logf("cron3")
	}), machine.WithTags("cron3"))
	m.Wait()
	if m.Current() != 0 {
		t.Fatalf("expected current to be zero, got: %v", m.Current())
	}
	if !seen {
		t.Fatalf("expected to have received subscription msg")
	}
}

func Test(t *testing.T) {
	runTest(t)
}
