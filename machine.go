//go:generate godocdown -o README.md

package machine

import (
	"context"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

/*
Machine is a zero dependency runtime for managed goroutines. It is inspired by errgroup.Group with extra bells & whistles:

- throttled goroutines

- self-cancellable goroutines with context

- global-cancellable goroutines with context (see Cancel)

- goroutines have IDs and optional tags for easy debugging(see Stats)

- publish/subscribe to channels for passing messages between goroutines

*/
type Machine struct {
	done          chan struct{}
	middlewares   []Middleware
	subChanLength int
	pubChanLength int
	cancel        func()
	ctx           context.Context
	workQueue     chan *work
	mu            sync.RWMutex
	routines      map[int]Routine
	max           int
	closeOnce     sync.Once
	subscriptions map[string]map[int]chan interface{}
	subMu         sync.RWMutex
}

// New Creates a new machine instance with the given root context & options
func New(ctx context.Context, options ...Opt) *Machine {
	opts := &option{}
	for _, o := range options {
		o(opts)
	}
	if opts.maxRoutines <= 0 {
		opts.maxRoutines = 10000
	}
	ctx, cancel := context.WithCancel(ctx)
	m := &Machine{
		done:          make(chan struct{}, 1),
		middlewares:   opts.middlewares,
		subChanLength: opts.subChannelLength,
		pubChanLength: opts.pubChannelLength,
		cancel:        cancel,
		ctx:           ctx,
		workQueue:     make(chan *work),
		mu:            sync.RWMutex{},
		routines:      map[int]Routine{},
		max:           opts.maxRoutines,
		closeOnce:     sync.Once{},
		subscriptions: map[string]map[int]chan interface{}{},
		subMu:         sync.RWMutex{},
	}
	go m.serve()
	return m
}

// Current returns current managed goroutine count
func (p *Machine) Current() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.routines)
}

// Go calls the given function in a new goroutine.
//
// The first call to return a non-nil error who's cause is machine.Cancel cancels the context of every job.
// All errors that are not of type machine.Cancel will be returned by Wait.
func (m *Machine) Go(fn Func, opts ...GoOpt) {
	o := &goOpts{}
	for _, opt := range opts {
		opt(o)
	}
	if m.ctx.Err() == nil {
		m.workQueue <- &work{
			opts: o,
			fn:   fn,
		}
	}
}

// serve waites for all goroutines to exit
func (m *Machine) serve() {
	for {
		select {
		case <-m.done:
			return
		case w := <-m.workQueue:
			if len(m.middlewares) > 0 {
				for _, ware := range m.middlewares {
					w.fn = ware(w.fn)
				}
			}
			for x := m.Current(); x >= m.max; x = m.Current() {
				if m.ctx.Err() != nil {
					return
				}
			}
			if w.opts.id == 0 {
				w.opts.id = rand.Int()
			}
			var (
				child  context.Context
				cancel func()
			)
			if w.opts.timeout != nil {
				child, cancel = context.WithTimeout(m.ctx, *w.opts.timeout)
			} else {
				child, cancel = context.WithCancel(m.ctx)
			}
			routine := &goRoutine{
				machine:  m,
				ctx:      child,
				id:       w.opts.id,
				tags:     w.opts.tags,
				start:    time.Now(),
				doneOnce: sync.Once{},
				cancel:   cancel,
			}
			m.mu.Lock()
			m.routines[w.opts.id] = routine
			m.mu.Unlock()
			go func() {
				defer routine.done()
				w.fn(routine)
			}()
		}
	}
}

// Wait blocks until all goroutines exit
func (m *Machine) Wait() {
	for m.Current() > 0 {
		for len(m.workQueue) > 0 {
		}
	}
	m.Cancel()
	m.done <- struct{}{}
}

// Cancel cancels every goroutines context
func (p *Machine) Cancel() {
	p.closeOnce.Do(func() {
		if p.cancel != nil {
			p.cancel()
		}
	})
}

/*
Stats returns Goroutine information from the machine
example:

{
            "count": 3,
            "routines": {
                "021851f5-d9ac-0f31-3a89-ddfc454c5f8f": {
                    "id": "021851f5-d9ac-0f31-3a89-ddfc454c5f8f",
                    "start": "2020-10-04T20:00:21.061072-06:00",
                    "duration": 3001366067,
                    "tags": [
                        "stream-to-acme.com"
                    ],
                },
                "8afa3f85-b8a6-2708-caeb-bac880b5b89b": {
                    "id": "8afa3f85-b8a6-2708-caeb-bac880b5b89b",
                    "start": "2020-10-04T20:00:21.011062-06:00",
                    "duration": 3051375565,
                    "tags": [
                        "subscribe"
                    ],
                },
                "93da5381-0164-4021-04e6-48b6226a1b78": {
                    "id": "93da5381-0164-4021-04e6-48b6226a1b78",
                    "start": "2020-10-04T20:00:21.01107-06:00",
                    "duration": 3051367098,
                    "tags": [
                        "publish"
                    ],
                }
     }
}
*/
func (m *Machine) Stats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	copied := map[string]RoutineStats{}
	for k, v := range m.routines {
		if v != nil {
			copied[strconv.Itoa(k)] = RoutineStats{
				PID:      v.PID(),
				Start:    v.Start(),
				Duration: v.Duration(),
				Tags:     v.Tags(),
			}
		}
	}
	return Stats{
		Count:    len(copied),
		Routines: copied,
	}
}
