package machine

import (
	"context"
	"fmt"
	"sync"
	"time"
)

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
	// PublishTo starts a stream that may be published to from the routine. It listens on the returned channel.
	PublishTo(channel string) chan interface{}
	// SubscribeTo subscribes to a channel & returns a go channel
	SubscribeTo(channel string) chan interface{}
	// Subscriptions returns the channels that this goroutine is subscribed to
	Subscriptions() []string
	// AddedAt returns the goroutine count before the goroutine was added
	AddedAt() int
	//Done cancels the context of the current goroutine & kills any of it's subscriptions
	Done()
}

type goRoutine struct {
	machine       *Machine
	addedAt       int
	ctx           context.Context
	id            string
	tags          []string
	start         time.Time
	subscriptions []string
	doneOnce      sync.Once
	cancel        func()
}

func (r *goRoutine) Context() context.Context {
	return r.ctx
}

func (r *goRoutine) ID() string {
	return r.id
}

func (r *goRoutine) AddedAt() int {
	return r.addedAt
}

func (r *goRoutine) Tags() []string {
	return r.tags
}

func (r *goRoutine) Start() time.Time {
	return r.start
}

func (r *goRoutine) Duration() time.Duration {
	return time.Since(r.start)
}

func (g *goRoutine) PublishTo(channel string) chan interface{} {
	ch := make(chan interface{}, g.machine.pubChanLength)
	g.machine.Go(func(routine Routine) error {
		for {
			select {
			case <-routine.Context().Done():
				return nil
			case obj := <-ch:
				if g.machine.subscriptions[channel] == nil {
					g.machine.subMu.Lock()
					g.machine.subscriptions[channel] = map[string]chan interface{}{}
					g.machine.subMu.Unlock()
				}
				channelSubscribers := g.machine.subscriptions[channel]
				for _, input := range channelSubscribers {
					input <- obj
				}
			}
		}
	}, fmt.Sprintf("stream-to-%s", channel))
	return ch
}

func (g *goRoutine) SubscribeTo(channel string) chan interface{} {
	g.machine.subMu.Lock()
	defer g.machine.subMu.Unlock()
	ch := make(chan interface{}, g.machine.subChanLength)
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

func (g *goRoutine) Done() {
	g.doneOnce.Do(func() {
		g.cancel()
		g.machine.mu.Lock()
		defer g.machine.mu.Unlock()
		g.machine.subMu.Lock()
		defer g.machine.subMu.Unlock()
		for _, sub := range g.subscriptions {
			if _, ok := g.machine.subscriptions[sub]; ok {
				delete(g.machine.subscriptions[sub], g.id)
			}
		}
		delete(g.machine.routines, g.id)
	})
}
