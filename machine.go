//go:generate godocdown -o README.md

package machine

import (
	"context"
	"errors"
	"fmt"
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
	middlewares   []Middleware
	subChanLength int
	pubChanLength int
	cancel        func()
	ctx           context.Context
	errs          []error
	mu            sync.RWMutex
	routines      map[string]Routine
	max           int
	closeOnce     sync.Once
	subscriptions map[string]map[string]chan interface{}
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
	return &Machine{
		middlewares:   opts.middlewares,
		subChanLength: opts.subChannelLength,
		pubChanLength: opts.pubChannelLength,
		cancel:        cancel,
		ctx:           ctx,
		errs:          nil,
		mu:            sync.RWMutex{},
		routines:      map[string]Routine{},
		max:           opts.maxRoutines,
		closeOnce:     sync.Once{},
		subscriptions: map[string]map[string]chan interface{}{},
		subMu:         sync.RWMutex{},
	}
}

// Current returns current managed goroutine count
func (p *Machine) Current() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.routines)
}

func (m *Machine) addRoutine(opts *goOpts) Routine {
	var (
		child  context.Context
		cancel func()
	)
	if opts.timeout != nil {
		child, cancel = context.WithTimeout(m.ctx, *opts.timeout)
	} else {
		child, cancel = context.WithCancel(m.ctx)
	}
	var x int
	for x = m.Current(); x >= m.max; x = m.Current() {
		if m.ctx.Err() != nil {
			cancel()
			return nil
		}
	}
	if opts.id == "" {
		opts.id = fmt.Sprintf("%v-%v", x, time.Now().UnixNano())
	}
	routine := &goRoutine{
		machine:       m,
		ctx:           child,
		id:            opts.id,
		tags:          opts.tags,
		start:         time.Now(),
		subscriptions: []string{},
		doneOnce:      sync.Once{},
		cancel:        cancel,
	}
	m.mu.Lock()
	m.routines[opts.id] = routine
	m.mu.Unlock()
	return routine
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
	routine := m.addRoutine(o)
	if len(m.middlewares) > 0 {
		for _, ware := range m.middlewares {
			fn = ware(fn)
		}
	}
	go func() {
		defer routine.Done()
		if err := fn(routine); err != nil {
			if errors.Unwrap(err) == Cancel {
				m.Cancel()
			} else {
				m.addErr(err)
			}
		}
	}()
}

func (p *Machine) addErr(err error) {
	p.errs = append(p.errs, err)
}

// Wait waites for all goroutines to exit
func (p *Machine) Wait() []error {
	for p.Current() != 0 {
	}
	p.Cancel()
	return p.errs
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
                    "subscriptions": null
                },
                "8afa3f85-b8a6-2708-caeb-bac880b5b89b": {
                    "id": "8afa3f85-b8a6-2708-caeb-bac880b5b89b",
                    "start": "2020-10-04T20:00:21.011062-06:00",
                    "duration": 3051375565,
                    "tags": [
                        "subscribe"
                    ],
                    "subscriptions": [
                        "acme.com"
                    ]
                },
                "93da5381-0164-4021-04e6-48b6226a1b78": {
                    "id": "93da5381-0164-4021-04e6-48b6226a1b78",
                    "start": "2020-10-04T20:00:21.01107-06:00",
                    "duration": 3051367098,
                    "tags": [
                        "publish"
                    ],
                    "subscriptions": null
                }
     }
}
*/
func (m *Machine) Stats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.subMu.RLock()
	defer m.subMu.RUnlock()
	copied := map[string]RoutineStats{}
	for k, v := range m.routines {
		if v != nil {
			copied[k] = RoutineStats{
				ID:            v.ID(),
				Start:         v.Start(),
				Duration:      v.Duration(),
				Tags:          v.Tags(),
				Subscriptions: v.Subscriptions(),
			}
		}
	}
	return Stats{
		Count:    len(copied),
		Routines: copied,
	}
}
