//go:generate godocdown -o README.md

package machine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/pkg/errors"
	"sync"
	"time"
)

var Cancel = errors.New("[machine] cancel")

// Machine is just like sync.WaitGroup, except it lets you throttle max goroutines.
type Machine struct {
	cache      Cache
	cancel     func()
	ctx        context.Context
	errs       []error
	routineMu  sync.RWMutex
	routines   map[string][]string
	max        int
	closeOnce  sync.Once
	debug      bool
	workerChan chan *worker
}

type worker struct {
	tags []string
	fn   func(ctx context.Context) error
}

type Opts struct {
	MaxRoutines   int
	Debug         bool
	CacheProvider Cache
	SyncInterval  time.Duration
}

func New(ctx context.Context, opts *Opts) (*Machine, error) {
	if opts == nil {
		opts = &Opts{}
	}
	if opts.MaxRoutines <= 0 {
		opts.MaxRoutines = 10000
	}
	if opts.SyncInterval <= 0 {
		opts.SyncInterval = 5 * time.Minute
	}
	if opts.CacheProvider == nil {
		opts.CacheProvider = NewInMemStorage()
	}
	ctx, cancel := context.WithCancel(ctx)
	m := &Machine{
		cache:      opts.CacheProvider,
		cancel:     cancel,
		ctx:        ctx,
		errs:       nil,
		routineMu:  sync.RWMutex{},
		routines:   map[string][]string{},
		max:        opts.MaxRoutines,
		closeOnce:  sync.Once{},
		debug:      false,
		workerChan: make(chan *worker, opts.MaxRoutines),
	}
	workerLisId := m.addRoutine([]string{"worker queue"})
	go func() {
		defer m.closeRoutine(workerLisId)
		child1, cancel1 := context.WithCancel(m.ctx)
		defer cancel1()
		gcTicker := time.NewTicker(opts.SyncInterval)
		for {
			select {
			case <-gcTicker.C:
				if err := m.cache.Sync(); err != nil {
					m.errs = append(m.errs, err)
				}
			case <-child1.Done():
				gcTicker.Stop()
				if err := m.cache.Sync(); err != nil {
					m.errs = append(m.errs, err)
				}
				if err := m.cache.Close(); err != nil {
					m.errs = append(m.errs, err)
				}
				return
			case work := <-m.workerChan:
				id := m.addRoutine([]string{"worker"})
				go func() {
					defer m.closeRoutine(id)
					child2, cancel2 := context.WithCancel(child1)
					defer cancel2()
					if err := work.fn(child2); err != nil {
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

func (p *Machine) Current() int {
	p.routineMu.Lock()
	defer p.routineMu.Unlock()
	return len(p.routines)
}

func (p *Machine) addRoutine(tags []string) string {
	var x int
	for x = len(p.routines); x >= p.max && p.ctx.Err() == nil; x = p.Current() {
	}
	id := hash(x)
	p.routineMu.Lock()
	p.routines[id] = tags
	p.routineMu.Unlock()
	return id
}

// Go calls the given function in a new goroutine.
//
// The first call to return a non-nil error who's cause is CancelGroup cancels the context of every job.
// All errors that are not CancelGroup will be returned by Wait.
func (p *Machine) Go(f func(ctx context.Context) error, tags ...string) {
	p.workerChan <- &worker{
		tags: tags,
		fn:   f,
	}
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
	for !p.Finished() {
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

func (p *Machine) Finished() bool {
	return p.Current() == 0
}

type Stats struct {
	Count    int
	Routines map[string][]string
}

func (m *Machine) Stats() Stats {
	copied := map[string][]string{}
	m.routineMu.RLock()
	defer m.routineMu.RUnlock()
	for k, v := range m.routines {
		copied[k] = v
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

func hash(this int) string {
	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%v", this)))
	return hex.EncodeToString(hasher.Sum(nil))
}
