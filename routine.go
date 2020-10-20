package machine

import (
	"context"
	"fmt"
	"runtime/trace"
	"strings"
	"sync"
	"time"
)

// Routine is an interface representing a goroutine
type Routine interface {
	// Context returns the goroutines unique context that may be used for cancellation
	Context() context.Context
	// Cancel cancels the context returned from Context()
	Cancel()
	// PID() is the goroutines unique process id
	PID() string
	// Tags() are the tags associated with the goroutine
	Tags() []string
	// Start is when the goroutine started
	Start() time.Time
	// Duration is the duration since the goroutine started
	Duration() time.Duration
	// Publish publishes the object to the given channel
	Publish(channel string, obj interface{}) error
	// Subscribe subscribes to a channel and executes the function on every message passed to it. It exits if the goroutines context is cancelled.
	Subscribe(channel string, handler func(obj interface{})) error
	// SubscribeN subscribes to the given channel until it receives N messages or its context is cancelled
	SubscribeN(channel string, n int, handler func(msg interface{})) error
	// TraceLog logs a message within the goroutine execution tracer. ref: https://golang.org/pkg/runtime/trace/#example_
	TraceLog(message string)
	// Machine returns the underlying routine's machine instance
	Machine() *Machine
}

type goRoutine struct {
	machine  *Machine
	ctx      context.Context
	id       string
	tags     []string
	start    time.Time
	doneOnce sync.Once
	cancel   func()
}

func (r *goRoutine) Context() context.Context {
	return r.ctx
}

func (r *goRoutine) PID() string {
	return r.id
}

func (r *goRoutine) Tags() []string {
	return r.tags
}

func (r *goRoutine) Cancel() {
	r.cancel()
}

func (r *goRoutine) Start() time.Time {
	return r.start
}

func (r *goRoutine) Duration() time.Duration {
	return time.Since(r.start)
}

func (g *goRoutine) Publish(channel string, obj interface{}) error {
	return g.machine.pubsub.Publish(channel, obj)
}

func (g *goRoutine) Subscribe(channel string, handler func(obj interface{})) error {
	return g.machine.pubsub.Subscribe(g.ctx, channel, handler)
}

func (g *goRoutine) SubscribeN(channel string, n int, handler func(obj interface{})) error {
	return g.machine.pubsub.SubscribeN(g.ctx, channel, n, handler)
}

func (g *goRoutine) Machine() *Machine {
	return g.machine
}

func (g *goRoutine) done() {
	g.doneOnce.Do(func() {
		g.cancel()
		g.machine.mu.Lock()
		delete(g.machine.routines, g.id)
		g.machine.mu.Unlock()
	})
	routinePool.deallocateRoutine(g)
}

func (g *goRoutine) TraceLog(message string) {
	trace.Logf(g.ctx, strings.Join(g.tags, " "), fmt.Sprintf("%s %s", g.PID(), message))
}
