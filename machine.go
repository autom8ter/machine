package machine

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
)

// Handler is a first class that is executed against the inbound message in a subscription.
// Return false to indicate that the subscription should end
type MessageHandlerFunc func(ctx context.Context, msg Message) (bool, error)

// Filter is a first class function to filter out messages before they reach a subscriptions primary Handler
// Return true to indicate that a message passes the filter
type MessageFilterFunc func(ctx context.Context, msg Message) (bool, error)

// Func is a first class function that is asynchronously executed.
type Func func(ctx context.Context) error

// Machine is an interface for highly asynchronous Go applications
type Machine interface {
	// Publish synchronously publishes the Message
	Publish(ctx context.Context, msg Message)
	// Subscribe synchronously subscribes to messages on a given channel,  executing the given HandlerFunc UNTIL the context cancels OR false is returned by the HandlerFunc.
	// Glob matching IS supported for subscribing to multiple channels at once.
	Subscribe(ctx context.Context, channel string, handler MessageHandlerFunc, options ...SubscriptionOpt) error
	// Go asynchronously executes the given Func
	Go(ctx context.Context, fn Func)
	// Current returns the number of active jobs that are running concurrently
	Current() int
	// Wait blocks until all active async functions(Go) exit
	Wait() error
	// Close blocks until all active routine's exit(calls Wait) then closes all active subscriptions
	Close()
}

// SubscriptionOptions holds config options for a subscription
type SubscriptionOptions struct {
	filter         MessageFilterFunc
	subscriptionID string
}

// SubscriptionOpt configures a subscription
type SubscriptionOpt func(options *SubscriptionOptions)

// WithFilter is a subscription option that filters messages
func WithFilter(filter MessageFilterFunc) SubscriptionOpt {
	return func(options *SubscriptionOptions) {
		options.filter = filter
	}
}

// WithSubscriptionID is a subscription option that sets the subscription id - if multiple consumers have the same subscritpion id,
// a message will be broadcasted to just one of the consumers
func WithSubscriptionID(id string) SubscriptionOpt {
	return func(options *SubscriptionOptions) {
		options.subscriptionID = id
	}
}

type machine struct {
	max           int
	subscriptions *sync.Map
	current       chan struct{}
	errChan       chan error
	wg            sync.WaitGroup
}

// Options holds config options for a machine instance
type Options struct {
	maxRoutines int
}

// Opt configures a machine instance
type Opt func(o *Options)

// WithThrottledRoutines throttles the max number of active routine's spawned by the Machine.
func WithThrottledRoutines(max int) Opt {
	return func(o *Options) {
		o.maxRoutines = max
	}
}

// New creates a new Machine instance with the given options(if present)
func New(opts ...Opt) Machine {
	options := &Options{}
	for _, o := range opts {
		o(options)
	}
	if options.maxRoutines <= 0 {
		options.maxRoutines = 1000
	}
	return &machine{
		max:           options.maxRoutines,
		current:       make(chan struct{}, options.maxRoutines),
		subscriptions: &sync.Map{},
		errChan:       make(chan error, 100),
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

func (p *machine) Subscribe(ctx context.Context, channel string, handler MessageHandlerFunc, options ...SubscriptionOpt) error {
	opts := &SubscriptionOptions{}
	for _, o := range options {
		o(opts)
	}
	var subID string
	if opts.subscriptionID != "" {
		subID = opts.subscriptionID
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ch, closer := p.setupSubscription(channel, subID)
	defer closer()
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-ch:
			if opts.filter != nil {
				shouldHandle, err := opts.filter(ctx, msg)
				if err != nil {
					return err
				}
				if !shouldHandle {
					continue
				}
			}
			contin, err := handler(ctx, msg)
			if err != nil {
				return err
			}
			if !contin {
				return nil
			}
		}
	}
}

func (p *machine) Publish(ctx context.Context, msg Message) {
	p.subscriptions.Range(func(key, value any) bool {

		split := strings.Split(key.(string), "|||")

		if globMatch(split[0], msg.Channel) {
			if ctx.Err() != nil {
				return false
			}
			value.(chan Message) <- msg
		}
		return true
	})
	return
}

func (m *machine) Wait() error {
	m.wg.Wait()
	select {
	case err := <-m.errChan:
		return err
	default:
		return nil
	}
}

func (p *machine) Close() {
	p.wg.Wait()
	p.subscriptions.Range(func(key, value any) bool {
		p.subscriptions.Delete(key)
		return true
	})
	close(p.errChan)
}

func (p *machine) setupSubscription(channel, subId string) (chan Message, func()) {
	if subId == "" {
		subId = fmt.Sprintf("%v", rand.Int())
	}
	ch := make(chan Message, 1)
	key := fmt.Sprintf("%s|||%s", channel, subId)
	p.subscriptions.Store(key, ch)
	return ch, func() {
		p.subscriptions.Delete(key)
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
