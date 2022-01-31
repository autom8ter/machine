package machine

import (
	"context"
	"log"
	"math/rand"
	"os"
	"strings"
	"sync"
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
	// Subscribe synchronously subscribes to messages on a given channel,  executing the given HandlerFunc UNTIL the context cancels OR false is returned by the HandlerFunc.
	// Glob matching IS supported for subscribing to multiple channels at once.
	Subscribe(ctx context.Context, channel string, handler MessageHandlerFunc, opts ...SubscriptionOpt)
	// Subscribers returns total number of subscribers to the given channel
	Subscribers(channel string) int
	// Subscriptions returns the channel names/patterns that subscribers are listening on
	Subscriptions() []string
	// Go asynchronously executes the given Func
	Go(ctx context.Context, fn Func)
	// Cron asynchronously executes the given function on a timed interval UNTIL the context cancels OR false is returned by the CronFunc
	Cron(ctx context.Context, interval time.Duration, fn CronFunc)
	// Current returns the number of active jobs that are running concurrently
	Current() int
	// Wait blocks until all active async functions(Go, Cron) exit
	Wait()
	// Close blocks until all active routine's exit(calls Wait) then closes all active subscriptions
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
	max           int
	subscriptions map[string]map[int]chan Message
	current       chan struct{}
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
	if options.errHandler == nil {
		logger := log.New(os.Stderr, "machine/v2", log.Flags())
		options.errHandler = func(err error) {
			logger.Printf("runtime error: %v", err)
		}
	}
	if options.maxRoutines <= 0 {
		options.maxRoutines = 1000
	}
	return &machine{
		errHandler:    options.errHandler,
		max:           options.maxRoutines,
		current:       make(chan struct{}, options.maxRoutines),
		subscriptions: map[string]map[int]chan Message{},
		subMu:         sync.RWMutex{},
		errChan:       make(chan error),
		wg:            sync.WaitGroup{},
	}
}

func (m *machine) Current() int {
	return len(m.current)
}

func (m *machine) Go(ctx context.Context, fn Func) {
	if ctx.Err() != nil {
		return
	}
	m.wg.Add(1)
	go func() {
		defer func() {
			m.wg.Done()
			// remove element from "current" channel
			<-m.current
		}()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		select {
		// return context error if context is closed before slot becomes available
		case <-ctx.Done():
			return
		case m.current <- struct{}{}: // add element to current channel
		}
		if err := fn(ctx); err != nil {
			m.errChan <- err
		}
	}()
}

type Message struct {
	Channel string
	Body    interface{}
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

func (m *machine) Subscribers(channel string) int {
	m.subMu.RLock()
	defer m.subMu.RUnlock()
	return len(m.subscriptions[channel])
}

func (m *machine) Subscriptions() []string {
	m.subMu.RLock()
	defer m.subMu.RUnlock()
	var channels []string
	for ch, _ := range m.subscriptions {
		channels = append(channels, ch)
	}
	return channels
}

func (p *machine) Publish(ctx context.Context, msg Message) {
	p.subMu.RLock()
	defer p.subMu.RUnlock()
	for k, channelSubscribers := range p.subscriptions {
		if globMatch(k, msg.Channel) {
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
	ch := make(chan Message, 1)
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
