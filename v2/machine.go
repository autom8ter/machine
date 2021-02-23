package machine

import (
	"context"
	"log"
	"math/rand"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Handler is a first class that is executed against the inbound message in a subscription.
// Return false to indicate that the subscription should end
type MessageHandlerFunc func(ctx context.Context, msg Message) (bool, error)

// Filter is a first class function to filter out messages before they reach a subscriptions primary Handler
// Return true to indicate that a message passes the filter
type MessageFilterFunc func(ctx context.Context, msg Message) (bool, error)

// Func is a first class function that is asynchronously executed.
type Func func(ctx context.Context) error

// CronFunc is a first class function that is asynchronously executed on a timed interval.
// Return false to indicate that the cron should end
type CronFunc func(ctx context.Context) (bool, error)

// Machine is an interface for highly asynchronous Go applications
type Machine interface {
	// Publish synchronously publishes the Message
	Publish(ctx context.Context, msg Message)
	// Subscribe synchronously subscribes to messages on a given channel,  executing the given Handler UNTIL the context cancels OR false is returned by the Handler function.
	// Glob matching IS supported for subscribing to multiple channels at once.
	Subscribe(ctx context.Context, channel string, handler MessageHandlerFunc, opts ...SubscriptionOpt)
	// Go asynchronously executes the given Func
	Go(ctx context.Context, fn Func)
	// Cron asynchronously executes the given function on a timed interval UNTIL the context cancels OR false is returned by the Cron function
	Cron(ctx context.Context, interval time.Duration, fn CronFunc)
	// Wait blocks until all active routine's exit
	Wait()
	// Close blocks until all active routine's exit then closes all subscriptions
	Close()
}

// SubscriptionOptions holds config options for a subscription
type SubscriptionOptions struct {
	filter MessageFilterFunc
}

// SubscriptionOpt configures a subscription
type SubscriptionOpt func(options *SubscriptionOptions)

// WithFilter is a subscription option that filters messages
func WithFilter(filter MessageFilterFunc) SubscriptionOpt {
	return func(options *SubscriptionOptions) {
		options.filter = filter
	}
}

type machine struct {
	errHandler    func(err error)
	started       int64
	finished      int64
	max           int
	subscriptions map[string]map[int]chan Message
	subMu         sync.RWMutex
	errChan       chan error
	wg            sync.WaitGroup
}

// Options holds config options for a machine instance
type Options struct {
	maxRoutines int
	errHandler  func(err error)
}

// Opt configures a machine instance
type Opt func(o *Options)

// WithThrottledRoutines throttles the max number of active routine's spawned by the Machine.
func WithThrottledRoutines(max int) Opt {
	return func(o *Options) {
		o.maxRoutines = max
	}
}

// WithErrHandler overrides the default machine error handler
func WithErrHandler(errHandler func(err error)) Opt {
	return func(o *Options) {
		o.errHandler = errHandler
	}
}

// New creates a new Machine instance with the given options(if present)
func New(opts ...Opt) Machine {
	options := &Options{}
	for _, o := range opts {
		o(options)
	}
	if options.errHandler != nil {
		logger := log.New(os.Stderr, "machine/v2", log.Flags())
		options.errHandler = func(err error) {
			logger.Printf("runtime error: %v", err)
		}
	}
	return &machine{
		errHandler:    options.errHandler,
		started:       0,
		finished:      0,
		max:           options.maxRoutines,
		subscriptions: map[string]map[int]chan Message{},
		subMu:         sync.RWMutex{},
		errChan:       make(chan error),
		wg:            sync.WaitGroup{},
	}
}

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

func (p *machine) Subscribe(ctx context.Context, channel string, handler MessageHandlerFunc, options ...SubscriptionOpt) {
	opts := &SubscriptionOptions{}
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
				}
				if !res {
					continue
				}
			}
			contin, err := handler(ctx, msg)
			if err != nil {
				p.errChan <- err
				continue
			}
			if !contin {
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

func (m *machine) Cron(ctx context.Context, timeout time.Duration, fn CronFunc) {
	m.Go(ctx, func(ctx context.Context) error {
		ticker := time.NewTicker(timeout)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				contin, err := fn(ctx)
				if err != nil {
					m.errChan <- err
				}
				if !contin {
					return nil
				}
			}
		}
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
