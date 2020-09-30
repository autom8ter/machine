//go:generate godocdown -o README.md

package machine

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"sync"
	"sync/atomic"
	"time"
)

var Cancel = errors.New("machine: cancel")

type Storage interface {
	Get(id string) (map[string]interface{}, error)
	Set(id string, data map[string]interface{}) error
	Range(fn func(key string, data map[string]interface{}) bool)
	Sync() error
	Del(id string) error
	Close() error
}

type inMem struct {
	data *sync.Map
}

func NewInMemStorage() Storage {
	return &inMem{data: &sync.Map{}}
}

func (i *inMem) Get(id string) (map[string]interface{}, error) {
	val, ok := i.data.Load(id)
	if !ok {
		return nil, fmt.Errorf("%s does not exist", id)
	}
	return val.(map[string]interface{}), nil
}

func (i *inMem) Sync() error {
	i.Range(func(id string, data map[string]interface{}) bool {
		if data["expired"] == true {
			if err := i.Del(id); err != nil {
				i.Del(id)
			}
		}
		if data["stale"] == true {
			i.Del(id)
		}
		return true
	})
	return nil
}

func (i *inMem) Set(id string, data map[string]interface{}) error {
	i.data.Store(id, data)
	return nil
}

func (i *inMem) Range(fn func(id string, data map[string]interface{}) bool) {
	i.data.Range(func(key, value interface{}) bool {
		return fn(key.(string), value.(map[string]interface{}))
	})
}

func (i *inMem) Del(id string) error {
	i.data.Delete(id)
	return nil
}

func (i *inMem) Close() error {
	return nil
}

// Machine is just like sync.WaitGroup, except it lets you throttle max goroutines.
type Machine struct {
	Storage
	cancel     func()
	ctx        context.Context
	errs       []error
	current    *uint64
	max        uint64
	closeOnce  sync.Once
	workerChan chan func(ctx context.Context) error
}

type Opts struct {
	MaxRoutines     uint64
	StorageProvider Storage
	SyncInterval    time.Duration
}

func New(ctx context.Context, opts *Opts) (*Machine, error) {
	if opts == nil {
		opts = &Opts{}
	}
	if opts.MaxRoutines == 0 {
		opts.MaxRoutines = 10000
	}
	if opts.SyncInterval == 0 {
		opts.SyncInterval = 5 * time.Minute
	}
	if opts.StorageProvider == nil {
		opts.StorageProvider = NewInMemStorage()
	}
	ctx, cancel := context.WithCancel(ctx)
	current := uint64(0)
	m := &Machine{
		Storage:    opts.StorageProvider,
		cancel:     cancel,
		ctx:        ctx,
		errs:       nil,
		current:    &current,
		max:        opts.MaxRoutines,
		closeOnce:  sync.Once{},
		workerChan: make(chan func(ctx context.Context) error, opts.MaxRoutines),
	}
	m.add(1)
	go func() {
		defer m.done()
		child1, cancel1 := context.WithCancel(m.ctx)
		defer cancel1()
		gcTicker := time.NewTicker(opts.SyncInterval)
		for {
			select {
			case <-gcTicker.C:
				if err := m.Sync(); err != nil {
					m.errs = append(m.errs, err)
				}
			case <-child1.Done():
				gcTicker.Stop()
				if err := m.Sync(); err != nil {
					m.errs = append(m.errs, err)
				}
				if err := m.Close(); err != nil {
					m.errs = append(m.errs, err)
				}
			case f := <-m.workerChan:
				m.add(1)
				go func() {
					defer m.done()
					child2, cancel2 := context.WithCancel(child1)
					if err := f(child2); err != nil {
						if errors.Cause(err) == Cancel {
							m.Cancel()
						} else {
							m.addErr(err)
						}
					}
					cancel2()
				}()
			}
		}
	}()
	return m, nil
}

func (p *Machine) Current() uint64 {
	return atomic.LoadUint64(p.current)
}

func (p *Machine) add(delta uint64) {
	for p.Current() >= p.max {
		time.Sleep(50 * time.Nanosecond)
	}
	atomic.AddUint64(p.current, delta)
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

func (p *Machine) done() {
	atomic.AddUint64(p.current, ^uint64(0))
}

func (p *Machine) Wait() []error {
	gcTicker := time.NewTicker(1 * time.Second)
	for !p.Finished() {
		select {
		case <-gcTicker.C:
			fmt.Printf("current = %v\n", p.Current())
		case <-p.ctx.Done():
			p.Cancel()
		}
	}
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
