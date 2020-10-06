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
	// PID() is the goroutines unique process id
	PID() int
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
	//Done cancels the context of the current goroutine & kills any of it's subscriptions
	Done()
}

type goRoutine struct {
	machine       *Machine
	ctx           context.Context
	id            int
	tags          []string
	start         time.Time
	subscriptions []string
	doneOnce      sync.Once
	cancel        func()
}

func (r *goRoutine) Context() context.Context {
	return r.ctx
}

func (r *goRoutine) PID() int {
	return r.id
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
	g.machine.Go(func(routine Routine) {
		for {
			select {
			case <-routine.Context().Done():
				return
			case obj := <-ch:
				if g.machine.subscriptions[channel] == nil {
					g.machine.subMu.Lock()
					g.machine.subscriptions[channel] = map[int]chan interface{}{}
					g.machine.subMu.Unlock()
				}
				channelSubscribers := g.machine.subscriptions[channel]
				for _, input := range channelSubscribers {
					input <- obj
				}
			}
		}
	}, WithTags(fmt.Sprintf("stream-to-%s", channel)))
	return ch
}

func (g *goRoutine) SubscribeTo(channel string) chan interface{} {
	g.machine.subMu.Lock()
	defer g.machine.subMu.Unlock()
	ch := make(chan interface{}, g.machine.subChanLength)
	if g.machine.subscriptions[channel] == nil {
		g.machine.subscriptions[channel] = map[int]chan interface{}{}
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
		g.machine.subMu.Lock()
		for _, sub := range g.subscriptions {
			if _, ok := g.machine.subscriptions[sub]; ok {
				delete(g.machine.subscriptions[sub], g.id)
			}
		}
		g.machine.subMu.Unlock()
		g.machine.mu.Lock()
		delete(g.machine.routines, g.id)
		g.machine.mu.Unlock()
	})
}
