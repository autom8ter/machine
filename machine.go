//go:generate godocdown -o README.md

package machine

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

// if a goroutine returns this error, every goroutines context will be cancelled
var Cancel = errors.New("[machine] cancel")

// Machine is just like sync.WaitGroup, except it lets you throttle max goroutines.
type Machine struct {
	cancel        func()
	ctx           context.Context
	errs          []error
	routineMu     sync.RWMutex
	routines      map[string]Routine
	max           int
	closeOnce     sync.Once
	debug         bool
	workerChan    chan *worker
	publishChan   chan *object
	subMu         sync.RWMutex
	subscriptions map[string]map[string]chan interface{}
}

type object struct {
	channel string
	obj     interface{}
}

type worker struct {
	fn   func(routine Routine) error
	tags []string
}

// Routine is an interface representing a goroutine
type Routine interface {
	// Context returns the goroutines unique context that may be used for cancellation
	Context() context.Context
	// ID() is the goroutines unique id
	ID() string
	// Tags() are the tags associated with the goroutine
	Tags() []string
	// Start is when the goroutine started
	Start() time.Time
	// Duration is the duration since the goroutine started
	Duration() time.Duration
	// Publish publishes the object to the given channel
	Publish(channel string, obj interface{})
	// Subscribe subscribes to a channel & returns a go channel
	Subscribe(channel string) chan interface{}
	// Subscriptions returns the channels that this goroutine is subscribed to
	Subscriptions() []string
	Done()
}

type goRoutine struct {
	machine       *Machine
	ctx           context.Context
	id            string
	tags          []string
	start         time.Time
	subscriptions []string
	doneOnce sync.Once
	cancel func()
}

func (r *goRoutine) Context() context.Context {
	return r.ctx
}

func (r *goRoutine) ID() string {
	return r.id
}

func (r *goRoutine) Tags() []string {
	return r.Tags()
}

func (r *goRoutine) Start() time.Time {
	return r.start
}

func (r *goRoutine) Duration() time.Duration {
	return time.Since(r.start)
}

func (g *goRoutine) Publish(channel string, obj interface{}) {
	if g.machine.publishChan == nil {
		g.machine.publishChan = make(chan *object, 1000)
	}
	g.machine.publishChan <- &object{
		channel: channel,
		obj:     obj,
	}
}

func (g *goRoutine) Subscribe(channel string) chan interface{} {
	g.machine.subMu.Lock()
	defer g.machine.subMu.Unlock()
	ch := make(chan interface{}, 1000)
	if g.machine.subscriptions == nil {
		g.machine.subscriptions = map[string]map[string]chan interface{}{}
	}
	if g.machine.subscriptions[channel] == nil {
		g.machine.subscriptions[channel] = map[string]chan interface{}{}
	}
	g.machine.subscriptions[channel][g.id] = ch
	g.subscriptions = append(g.subscriptions, channel)
	return ch
}

func (g *goRoutine) Subscriptions() []string {
	return g.subscriptions
}

// Opts are options when creating a machine instance
type Opts struct {
	// MaxRoutines throttles goroutines at the given count
	MaxRoutines int
	// Debug enables debug logs
	Debug bool
}

// New Creates a new machine instance
func New(ctx context.Context, opts *Opts) (*Machine, error) {
	if opts == nil {
		opts = &Opts{}
	}
	if opts.MaxRoutines <= 0 {
		opts.MaxRoutines = 10000
	}
	ctx, cancel := context.WithCancel(ctx)
	m := &Machine{
		cancel:        cancel,
		ctx:           ctx,
		errs:          nil,
		routineMu:     sync.RWMutex{},
		routines:      map[string]Routine{},
		max:           opts.MaxRoutines,
		closeOnce:     sync.Once{},
		debug:         false,
		workerChan:    make(chan *worker, opts.MaxRoutines),
		publishChan:   make(chan *object, 1000),
		subMu:         sync.RWMutex{},
		subscriptions: map[string]map[string]chan interface{}{},
	}

	{
		workerLisRoutine := m.addRoutine("machine.worker.queue")
		go func() {
			defer workerLisRoutine.Done()
			for {
				select {
				case <-workerLisRoutine.Context().Done():
					return
				case obj := <-m.publishChan:
					if m.subscriptions[obj.channel] == nil {
						m.subscriptions[obj.channel] = map[string]chan interface{}{}
					}
					channelSubscribers := m.subscriptions[obj.channel]
					for _, input := range channelSubscribers {
						input <- obj.obj
					}
				case work := <-m.workerChan:
					workerRoutine := m.addRoutine(work.tags...)
					go func() {
						defer workerRoutine.Done()
						if err := work.fn(workerRoutine); err != nil {
							if errors.Unwrap(err) == Cancel {
								m.Cancel()
							} else {
								m.addErr(err)
							}
						}
					}()
				}
			}
		}()
	}
	return m, nil
}

// Current returns current managed goroutine count
func (p *Machine) Current() int {
	p.routineMu.Lock()
	defer p.routineMu.Unlock()
	return len(p.routines)
}

func (m *Machine) addRoutine(tags ...string) Routine {
	child, cancel := context.WithCancel(m.ctx)
	var x int
	for x = m.Current(); x >= m.max; x = m.Current() {
		if m.ctx.Err() != nil {
			return nil
		}
	}
	id := uuid()
	routine := &goRoutine{
		machine: m,
		ctx:     child,
		id:      id,
		tags:    tags,
		start:   time.Now(),
		doneOnce: sync.Once{},
		cancel: cancel,
	}
	m.routineMu.Lock()
	m.routines[id] = routine
	m.routineMu.Unlock()
	return routine
}

// Go calls the given function in a new goroutine.
//
// The first call to return a non-nil error who's cause is CancelGroup cancels the context of every job.
// All errors that are not CancelGroup will be returned by Wait.
func (p *Machine) Go(f func(routine Routine) error, tags ...string) {
	p.workerChan <- &worker{
		fn:   f,
		tags: tags,
	}
}

func (p *Machine) addErr(err error) {
	p.errs = append(p.errs, err)
}

func (g *goRoutine) Done() {
	g.doneOnce.Do(func() {
		g.cancel()
		g.machine.subMu.Lock()
		defer g.machine.subMu.Unlock()
		g.machine.routineMu.Lock()
		defer g.machine.routineMu.Unlock()
		for _, sub := range g.subscriptions {
			if _, ok := g.machine.subscriptions[sub]; ok {
				delete(g.machine.subscriptions[sub], g.id)
			}
		}
		delete(g.machine.routines, g.id)
	})
}

// Wait waites for all goroutines to exit
func (p *Machine) Wait() []error {
	for !(p.Current() != 0) {
	}
	p.Cancel()
	for !(p.Current() != 0) {
	}
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

// Stats holds information about goroutines
type Stats struct {
	Count    int
	Routines map[string]Routine
}

// Stats returns Goroutine information
func (m *Machine) Stats() Stats {
	copied := map[string]Routine{}
	m.routineMu.RLock()
	defer m.routineMu.RUnlock()
	for k, v := range m.routines {
		if v != nil {
			copied[k] = v
		}
	}
	return Stats{
		Routines: copied,
		Count:    len(copied),
	}
}

func (p *Machine) debugf(format string, a ...interface{}) {
	if p.debug {
		fmt.Printf(format, a...)
	}
}

func uuid() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal(err)
	}
	uuid := fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return uuid
}
