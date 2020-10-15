package machine

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

const DefaultMaxRoutines = 1000

// Machine is a zero dependency runtime for managed goroutines. It is inspired by errgroup.Group with extra bells & whistles:
type Machine struct {
	parent      *Machine
	children    []*Machine
	childMu     sync.RWMutex
	done        chan struct{}
	cancel      func()
	middlewares []Middleware
	ctx         context.Context
	workQueue   chan *work
	mu          sync.RWMutex
	routines    map[string]Routine
	tags        []string
	max         int
	closeOnce   sync.Once
	doneOnce    sync.Once
	pubsub      PubSub
	total       int64
	timeout     *time.Duration
	deadline    *time.Time
}

// New Creates a new machine instance with the given root context & options
func New(ctx context.Context, options ...Opt) *Machine {
	opts := &option{}
	for _, o := range options {
		o(opts)
	}
	if opts.maxRoutines <= 0 {
		opts.maxRoutines = DefaultMaxRoutines
	}
	if opts.pubsub == nil {
		opts.pubsub = &pubSub{
			subscriptions: map[string]map[int]chan interface{}{},
			subMu:         sync.RWMutex{},
		}
	}
	if opts.data != nil {
		for k, v := range opts.data {
			ctx = context.WithValue(ctx, k, v)
		}
	}
	ctx, cancel := context.WithCancel(ctx)
	if opts.timeout != nil {
		ctx, cancel = context.WithTimeout(ctx, *opts.timeout)
	}
	if opts.deadline != nil {
		ctx, cancel = context.WithDeadline(ctx, *opts.deadline)
	}
	m := &Machine{
		parent:      opts.parent,
		children:    opts.children,
		childMu:     sync.RWMutex{},
		done:        make(chan struct{}, 1),
		cancel:      cancel,
		middlewares: opts.middlewares,
		ctx:         ctx,
		workQueue:   make(chan *work),
		mu:          sync.RWMutex{},
		routines:    map[string]Routine{},
		tags:        opts.tags,
		max:         opts.maxRoutines,
		closeOnce:   sync.Once{},
		doneOnce:    sync.Once{},
		pubsub:      opts.pubsub,
		total:       0,
		timeout:     opts.timeout,
		deadline:    opts.deadline,
	}
	go m.serve()
	return m
}

// Active returns current active managed goroutine count
func (p *Machine) Active() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.routines)
}

// Total returns total goroutines that have been executed by the machine
func (p *Machine) Total() int {
	return int(atomic.LoadInt64(&p.total))
}

// Tags returns the machine's tags
func (p *Machine) Tags() []string {
	return p.tags
}

// Go calls the given function in a new goroutine.
// it is passed information about the goroutine at runtime via the Routine interface
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

func (m *Machine) serve() {
	for {
		select {
		case <-m.done:
			return
		case w := <-m.workQueue:
			for _, ware := range w.opts.middlewares {
				w.fn = ware(w.fn)
			}
			for _, ware := range m.middlewares {
				w.fn = ware(w.fn)
			}
			for x := m.Active(); x >= m.max; x = m.Active() {

			}
			if w.opts.id == "" {
				w.opts.id = genUUID()
			}
			ctx, cancel := context.WithCancel(m.ctx)
			if w.opts.data != nil {
				for k, v := range w.opts.data {
					ctx = context.WithValue(ctx, k, v)
				}
			}
			if w.opts.timeout != nil {
				ctx, cancel = context.WithTimeout(ctx, *w.opts.timeout)
			}
			if w.opts.deadline != nil {
				ctx, cancel = context.WithDeadline(ctx, *w.opts.deadline)
			}
			routine := &goRoutine{
				machine:  m,
				ctx:      ctx,
				id:       w.opts.id,
				tags:     w.opts.tags,
				start:    time.Now(),
				doneOnce: sync.Once{},
				cancel:   cancel,
			}
			m.mu.Lock()
			m.routines[w.opts.id] = routine
			m.mu.Unlock()
			atomic.AddInt64(&m.total, 1)
			go func() {
				defer routine.done()
				w.fn(routine)
			}()
		}
	}
}

// Wait blocks until total active goroutine count reaches zero for the instance and all of it's children.
// At least one goroutine must have finished in order for wait to un-block
func (m *Machine) Wait() {
	for m.Total() < 1 {

	}
	for m.Active() > 0 {
		for len(m.workQueue) > 0 {
		}
		for _, child := range m.children {
			child.Wait()
		}
	}
}

// Cancel cancels every goroutines context within the machine instance & it's children
func (p *Machine) Cancel() {
	p.closeOnce.Do(func() {
		if p.cancel != nil {
			p.cancel()
			for _, child := range p.children {
				child.Cancel()
			}
		}
	})
}

// Stats returns Goroutine information from the machine
func (m *Machine) Stats() *Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	copied := []RoutineStats{}
	for _, v := range m.routines {
		if v != nil {
			copied = append(copied, RoutineStats{
				PID:      v.PID(),
				Start:    v.Start(),
				Duration: v.Duration(),
				Tags:     v.Tags(),
			})
		}
	}
	return &Stats{
		Tags:             m.tags,
		TotalRoutines:    m.Total(),
		ActiveRoutines:   len(copied),
		Routines:         copied,
		TotalChildren:    len(m.children),
		HasParent:        m.parent != nil,
		TotalMiddlewares: len(m.middlewares),
		Timeout:          m.timeout,
		Deadline:         m.deadline,
	}
}

// Close completely closes the machine instance & all of it's children
func (m *Machine) Close() {
	m.doneOnce.Do(func() {
		m.Cancel()
		m.done <- struct{}{}
		m.pubsub.Close()
		for _, child := range m.children {
			child.Close()
		}
	})
}

// Sub returns a nested Machine instance that is dependent on the parent machine's context.
func (m *Machine) Sub(opts ...Opt) *Machine {
	opts = append(opts, WithParent(m))
	sub := New(m.ctx, opts...)
	m.childMu.Lock()
	m.children = append(m.children, sub)
	m.childMu.Unlock()
	return sub
}

// Parent returns the parent Machine instance if it exists and nil if not.
func (m *Machine) Parent() *Machine {
	return m.parent
}

// HasRoutine returns true if the machine has a active routine with the given id
func (m *Machine) HasRoutine(id string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.routines[id] != nil
}

// CancelRoutine cancels the context of the active routine with the given id if it exists.
func (m *Machine) CancelRoutine(id string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if r, ok := m.routines[id]; ok {
		r.Cancel()
	}
}
