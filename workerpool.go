//go:generate godocdown -o README.md

package sync

import (
	"context"
	"github.com/pkg/errors"
	"sync/atomic"
	"time"
)

var Cancel = errors.New("sync: cancel")

// WorkerPool is just like sync.WaitGroup, except it lets you throttle max goroutines.
type WorkerPool struct {
	cancel  func()
	ctx     context.Context
	errs    []error
	current *int64
	max     int64
}

func NewWorkerPool(ctx context.Context, max int64) *WorkerPool {
	ctx, cancel := context.WithCancel(ctx)
	current := int64(0)
	return &WorkerPool{
		cancel:  cancel,
		ctx:     ctx,
		errs:    nil,
		current: &current,
		max:     max,
	}
}

func (p *WorkerPool) Current() int64 {
	return atomic.LoadInt64(p.current)
}

func (p *WorkerPool) Add(delta int64) {
	for p.Current() >= p.max {
		time.Sleep(50 * time.Nanosecond)
	}
	atomic.AddInt64(p.current, delta)
}


// Go calls the given function in a new goroutine.
//
// The first call to return a non-nil error who's cause is CancelGroup cancels the context of every job.
// All errors that are not CancelGroup will be returned by Wait.
func (p *WorkerPool) Go(f func(ctx context.Context) error) {
	p.Add(1)
	go func() {
		defer p.Done()
		child, cancel := context.WithCancel(p.ctx)
		defer cancel()
		if err := f(child); err != nil {
			if errors.Cause(err) == Cancel {
				p.Cancel()
			} else {
				p.AddErr(err)
			}
		}
	}()
}

func (p *WorkerPool) AddErr(err error) {
	p.errs = append(p.errs, err)
}

func (p *WorkerPool) Done() {
	atomic.AddInt64(p.current,  -1)
}

func (p *WorkerPool) Wait() []error {
	for {
		if p.Current() == 0 {
			p.Cancel()
			return p.errs
		}
	}
}

// Cancel cancels every functions context
func (p *WorkerPool) Cancel() {
	if p.cancel != nil {
		p.cancel()
	}
}