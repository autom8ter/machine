//go:generate godocdown -o README.md

package machine

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/pkg/errors"
	"log"
	"sync"
	"time"
)

var Cancel = errors.New("[machine] cancel")

// Machine is just like sync.WaitGroup, except it lets you throttle max goroutines.
type Machine struct {
	cancel     func()
	ctx        context.Context
	errs       []error
	routineMu  sync.RWMutex
	routines   map[string]*Routine
	max        int
	closeOnce  sync.Once
	debug      bool
	workerChan chan func(ctx context.Context) error
}

type Routine struct {
	ID       string
	Metadata map[string]string
	Start    time.Time
	Duration time.Duration
	dataChan chan interface{}
}

type Opts struct {
	MaxRoutines int
	Debug       bool
}

func New(ctx context.Context, opts *Opts) (*Machine, error) {
	if opts == nil {
		opts = &Opts{}
	}
	if opts.MaxRoutines <= 0 {
		opts.MaxRoutines = 10000
	}
	ctx, cancel := context.WithCancel(ctx)
	m := &Machine{
		cancel:     cancel,
		ctx:        ctx,
		errs:       nil,
		routineMu:  sync.RWMutex{},
		routines:   map[string]*Routine{},
		max:        opts.MaxRoutines,
		closeOnce:  sync.Once{},
		debug:      false,
		workerChan: make(chan func(ctx context.Context) error, opts.MaxRoutines),
	}
	workerLisId := m.addRoutine()
	go func() {
		defer m.closeRoutine(workerLisId)
		child1, cancel1 := context.WithCancel(m.ctx)
		defer cancel1()
		for {
			select {
			case <-child1.Done():
				return
			case work := <-m.workerChan:
				id := m.addRoutine()
				go func() {
					defer m.closeRoutine(id)
					child2, cancel2 := context.WithCancel(child1)
					defer cancel2()
					if err := work(child2); err != nil {
						if errors.Cause(err) == Cancel {
							m.Cancel()
						} else {
							m.addErr(err)
						}
					}
				}()
			}
		}
	}()
	return m, nil
}

// Current returns current managed goroutine count
func (p *Machine) Current() int {
	p.routineMu.Lock()
	defer p.routineMu.Unlock()
	return len(p.routines)
}

func (p *Machine) addRoutine() string {
	var x int
	for x = len(p.routines); x >= p.max && p.ctx.Err() == nil; x = p.Current() {
	}
	id := uuid()
	p.routineMu.Lock()
	p.routines[id] = &Routine{
		ID:       id,
		Metadata: map[string]string{},
		Start:    time.Now(),
		dataChan: make(chan interface{}),
	}
	p.routineMu.Unlock()
	return id
}

// Go calls the given function in a new goroutine.
//
// The first call to return a non-nil error who's cause is CancelGroup cancels the context of every job.
// All errors that are not CancelGroup will be returned by Wait.
func (p *Machine) Go(f func(ctx context.Context) error) {
	p.workerChan <- f
}

func (p *Machine) addErr(err error) {
	p.errs = append(p.errs, err)
}

func (p *Machine) closeRoutine(id string) {
	p.routineMu.Lock()
	delete(p.routines, id)
	p.routineMu.Unlock()
}

func (p *Machine) Wait() []error {
	for !(p.Current() > 0) {
	}
	p.Cancel()
	return p.errs
}

// Cancel cancels every functions context
func (p *Machine) Cancel() {
	p.closeOnce.Do(func() {
		if p.cancel != nil {
			p.cancel()
		}
	})
}

type Stats struct {
	Count    int
	Routines map[string]*Routine
}

func (m *Machine) Stats() Stats {
	copied := map[string]*Routine{}
	m.routineMu.RLock()
	defer m.routineMu.RUnlock()
	for k, v := range m.routines {
		if v != nil {
			v.Duration = time.Since(v.Start)
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
