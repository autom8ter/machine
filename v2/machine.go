package machine

import (
	"context"
	"log"
	"math/rand"
	"os"
	"strings"
	"sync"
	"sync/atomic"
)

// Handler is the function executed against the inbound message in a subscription. If
type Handler func(ctx context.Context, msg Message) (done bool, err error)

// Filter is a function to filter out messages before they reach a subscriptions primary Handler
type Filter func(ctx context.Context, msg Message) (match bool, err error)

// Machine is an interface
type Machine interface {
	// Publish publishes the Message
	Publish(ctx context.Context, msg Message)
	// Subscribe subscribes to messages on a given channel,  executing the given Handler it returns done OR until the context is cancelled
	Subscribe(ctx context.Context, channel string, handler Handler, opts ...SubOpt)
	// Go asynchronously executes the given Func
	Go(ctx context.Context, fn Func)
	// Loop asynchronously executes the given function UNTIL the context cancels OR an error is returned by Func
	Loop(ctx context.Context, fn Func)
	// Wait blocks until all active routine's exit, using the closure fn to handle runtime errors from Handler's, Filter's & Func's
	Wait()
	// Close closes all subscriptions
	Close()
}

type SubOptions struct {
	filter Filter
}

type SubOpt func(options *SubOptions)

func WithFilter(filter Filter) SubOpt {
	return func(options *SubOptions) {
		options.filter = filter
	}
}

type machine struct {
	errHandler func(err error)
	started       int64
	finished      int64
	max           int
	subscriptions map[string]map[int]chan Message
	subMu         sync.RWMutex
	errChan       chan error
	wg            sync.WaitGroup
}

type Options struct {
	maxRoutines int
	errHandler func(err error)
}

type Opt func(o *Options)

func WithThrottledRoutines(max int) Opt {
	return func(o *Options) {
		o.maxRoutines = max
	}
}

func WithErrHandler(errHandler func(err error)) Opt {
	return func(o *Options) {
		o.errHandler = errHandler
	}
}

func New(opts ...Opt) Machine {
	options := &Options{}
	for _, o := range opts {
		o(options)
	}
	if options.errHandler != nil {
		logger := log.New(os.Stderr, "machine", log.Flags())
		options.errHandler = func(err error) {
			logger.Printf("runtime error: %v", err)
		}
	}
	return &machine{
		errHandler: options.errHandler,
		started:       0,
		finished:      0,
		max:           options.maxRoutines,
		subscriptions: map[string]map[int]chan Message{},
		subMu:         sync.RWMutex{},
		errChan:       make(chan error),
		wg:            sync.WaitGroup{},
	}
}

// Func is a first class function that is asynchronously executed against a machine instance.
type Func func(ctx context.Context) error

func (m *machine) Go(ctx context.Context, fn Func) {
	if m.max > 0 {
		for x := m.activeRoutines(); x >= m.max; x = m.activeRoutines() {
			if ctx.Err() != nil {
				return
			}
		}
	}
	atomic.AddInt64(&m.started, 1)
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer atomic.AddInt64(&m.finished, 1)
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		if err := fn(ctx); err != nil {
			m.errChan <- err
		}
	}()
}

func (m *machine) activeRoutines() int {
	return int(atomic.LoadInt64(&m.started) - atomic.LoadInt64(&m.finished))
}

type Msg struct {
	Channel string
	Body    interface{}
}

func (m Msg) GetChannel() string {
	return m.Channel
}

func (m Msg) GetBody() interface{} {
	return m.Body
}

type Message interface {
	GetChannel() string
	GetBody() interface{}
}

func (p *machine) Subscribe(ctx context.Context, channel string, handler Handler, options ...SubOpt) {
	opts := &SubOptions{}
	for _, o := range options {
		o(opts)
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ch, closer := p.setupSubscription(channel)
	defer closer()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch:
			if opts.filter != nil {
				res, err := opts.filter(ctx, msg)
				if err != nil {
					p.errChan <- err
					continue
				}
				if !res {
					continue
				}
			}
			done, err := handler(ctx, msg)
			if err != nil {
				p.errChan <- err
				continue
			}
			if !done {
				return
			}
		}
	}
}

func (p *machine) Publish(ctx context.Context, msg Message) {
	p.subMu.RLock()
	subscribers := p.subscriptions
	p.subMu.RUnlock()
	for k, channelSubscribers := range subscribers {
		if globMatch(k, msg.GetChannel()) {
			for _, ch := range channelSubscribers {
				if ctx.Err() != nil {
					return
				}
				ch <- msg
			}
		}
	}
	return
}

func (m *machine) Wait() {
	done := make(chan struct{}, 1)
	go func() {
		for {
			select {
			case <-done:
				return
			case err := <-m.errChan:
				if m.errHandler != nil {
					m.errHandler(err)
				}
			}
		}
	}()
	m.wg.Wait()
	done <- struct{}{}
}

func (p *machine) Close() {
	p.wg.Wait()
	p.subMu.Lock()
	defer p.subMu.Unlock()
	for k, _ := range p.subscriptions {
		delete(p.subscriptions, k)
	}
	close(p.errChan)
}

func (m *machine) loop(ctx context.Context, fn Func) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			if err := fn(ctx); err != nil {
				m.errChan <- err
			}
		}
	}

}

func (m *machine) Loop(ctx context.Context, fn Func) {
	m.Go(ctx, func(ctx context.Context) error {
		return m.loop(ctx, fn)
	})
}

func (p *machine) setupSubscription(channel string) (chan Message, func()) {
	subId := rand.Int()
	ch := make(chan Message, 10)
	p.subMu.Lock()
	if p.subscriptions[channel] == nil {
		p.subscriptions[channel] = map[int]chan Message{}
	}
	p.subscriptions[channel][subId] = ch
	p.subMu.Unlock()
	return ch, func() {
		p.subMu.Lock()
		delete(p.subscriptions[channel], subId)
		p.subMu.Unlock()
		close(ch)
	}
}

func globMatch(pattern string, subj string) bool {
	const matchAll = "*"
	if pattern == subj {
		return true
	}
	if pattern == matchAll {
		return true
	}

	parts := strings.Split(pattern, matchAll)
	if len(parts) == 1 {
		return subj == pattern
	}
	leadingGlob := strings.HasPrefix(pattern, matchAll)
	trailingGlob := strings.HasSuffix(pattern, matchAll)
	end := len(parts) - 1
	for i := 0; i < end; i++ {
		idx := strings.Index(subj, parts[i])

		switch i {
		case 0:
			if !leadingGlob && idx != 0 {
				return false
			}
		default:
			if idx < 0 {
				return false
			}
		}
		subj = subj[idx+len(parts[i]):]
	}
	return trailingGlob || strings.HasSuffix(subj, parts[end])
}
