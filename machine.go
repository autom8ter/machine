package machine

import (
	"context"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const DefaultMaxRoutines = 1000

// Machine is a zero dependency runtime for managed goroutines. It is inspired by errgroup.Group with extra bells & whistles:
type Machine struct {
	id          string
	parent      *Machine
	children    map[string]*Machine
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
	cache       Cache
	total       int64
	timeout     time.Duration
	deadline    time.Time
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
	if opts.key != nil && opts.val != nil {
		ctx = context.WithValue(ctx, opts.key, opts.val)
	}
	ctx, cancel := context.WithCancel(ctx)
	if opts.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, opts.timeout)
	}
	if opts.deadline.Unix() > time.Now().Unix() {
		ctx, cancel = context.WithDeadline(ctx, opts.deadline)
	}
	if opts.id == "" {
		opts.id = genUUID()
	}
	children := map[string]*Machine{}
	for _, c := range opts.children {
		children[c.id] = c
	}
	if opts.cache == nil {
		opts.cache = newCache(ctx, time.NewTicker(5*time.Minute))
	}
	m := &Machine{
		id:          opts.id,
		children:    children,
		childMu:     sync.RWMutex{},
		done:        make(chan struct{}, 1),
		cancel:      cancel,
		middlewares: opts.middlewares,
		ctx:         ctx,
		workQueue:   make(chan *work, 1),
		mu:          sync.RWMutex{},
		routines:    map[string]Routine{},
		tags:        opts.tags,
		max:         opts.maxRoutines,
		closeOnce:   sync.Once{},
		doneOnce:    sync.Once{},
		pubsub:      opts.pubsub,
		cache:       opts.cache,
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

// Cache returns the machine's cache implementation. One is automatically set if not provided as an Opt on machine instance creation.
func (m *Machine) Cache() Cache {
	return m.cache
}

// Go calls the given function in a new goroutine.
// it is passed information about the goroutine at runtime via the Routine interface
func (m *Machine) Go(fn Func, opts ...GoOpt) {
	if m.ctx.Err() == nil {
		w := workPool.allocateWork()
		for _, opt := range opts {
			opt(w.opts)
		}
		w.fn = fn
		m.workQueue <- w
	}
}

func (m *Machine) serve() {
	interupt := make(chan os.Signal, 1)
	signal.Notify(interupt, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interupt)
	for {
		select {
		case <-interupt:
			m.Cancel()
			return
		case <-m.done:
			m.Cancel()
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
			if w.opts.key != nil && w.opts.val != nil {
				ctx = context.WithValue(ctx, w.opts.key, w.opts.val)
			}
			if w.opts.timeout != nil {
				ctx, cancel = context.WithTimeout(ctx, *w.opts.timeout)
			}
			if w.opts.deadline != nil {
				ctx, cancel = context.WithDeadline(ctx, *w.opts.deadline)
			}
			now := time.Now()
			ctx = pprof.WithLabels(ctx, pprof.Labels(
				"id", w.opts.id,
				"tags", strings.Join(w.opts.tags, " "),
				"machine_id", m.id,
				"start", now.String(),
			))
			pprof.Do()
			routine := routinePool.allocateRoutine()
			routine.machine = m
			routine.ctx = ctx
			routine.id = w.opts.id
			routine.tags = w.opts.tags
			routine.start = now
			routine.doneOnce = sync.Once{}
			routine.cancel = cancel
			m.mu.Lock()
			m.routines[w.opts.id] = routine
			m.mu.Unlock()
			atomic.AddInt64(&m.total, 1)
			go func(r *goRoutine) {
				defer workPool.deallocateWork(w)
				pprof.SetGoroutineLabels(ctx)
				w.fn(routine)
				r.done()
			}(routine)
		}
	}
}

// Wait blocks until total active goroutine count reaches zero for the instance and all of it's children.
// At least one goroutine must have finished in order for wait to un-block
func (m *Machine) Wait() {
	for m.Total() < 1 {

	}
	for _, child := range m.children {
		child.Wait()
	}
	for m.Active() > 0 {
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

// Stats returns Goroutine information from the machine and all of it's children
func (m *Machine) Stats() *Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.childMu.RLock()
	defer m.childMu.RUnlock()
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
	stats := &Stats{
		ID:               m.id,
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

	for _, child := range m.children {
		stats.Children = append(stats.Children, child.Stats())
	}
	return stats
}

// Close completely closes the machine instance & all of it's children
func (m *Machine) Close() {
	m.doneOnce.Do(func() {
		m.Cancel()
		for _, child := range m.children {
			child.Close()
		}
		m.done <- struct{}{}
		m.pubsub.Close()
		m.cache.Close()
	})
}

// Sub returns a nested Machine instance that is dependent on the parent machine's context.
// It inherits the parent's pubsub/cache implementation & middlewares if none are provided
// Sub machine's do not inherit their parents max routine setting
func (m *Machine) Sub(opts ...Opt) *Machine {
	opts = append([]Opt{WithMiddlewares(m.middlewares...), WithCache(m.cache), WithPubSub(m.pubsub)}, opts...)
	sub := New(m.ctx, opts...)
	sub.parent = m
	m.childMu.Lock()
	m.children[sub.ID()] = sub
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

// ID returns the machine instance's unique id.
func (m *Machine) ID() string {
	return m.id
}
