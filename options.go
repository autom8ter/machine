package machine

import (
	"github.com/autom8ter/machine/pubsub"
	"sort"
	"time"
)

// goOpts holds options for creating a goroutine. It is configured via GoOpt functions.
type goOpts struct {
	id          string
	tags        []string
	timeout     *time.Duration
	deadline    *time.Time
	middlewares []Middleware
	key         interface{}
	val         interface{}
}

// GoOpt is a function that configures GoOpts
type GoOpt func(o *goOpts)

// GoWithTags is a GoOpt that adds an array of strings as "tags" to the Routine.
func GoWithTags(tags ...string) GoOpt {
	return func(o *goOpts) {
		o.tags = append(o.tags, tags...)
		sort.Strings(o.tags)
	}
}

// GoWithPID is a GoOpt that sets/overrides the process ID of the Routine. A random id is assigned if this option is not used.
func GoWithPID(id string) GoOpt {
	return func(o *goOpts) {
		o.id = id
	}
}

// GoWithTimeout is a GoOpt that creates the Routine's context with the given timeout value
func GoWithTimeout(to time.Duration) GoOpt {
	return func(o *goOpts) {
		o.timeout = &to
	}
}

// GoWithDeadline is a GoOpt that creates the Routine's context with the given deadline.
func GoWithDeadline(deadline time.Time) GoOpt {
	return func(o *goOpts) {
		o.deadline = &deadline
	}
}

// GoWithMiddlewares wraps the gived function with the input middlewares.
func GoWithMiddlewares(middlewares ...Middleware) GoOpt {
	return func(o *goOpts) {
		o.middlewares = append(o.middlewares, middlewares...)
	}
}

// GoWithValues adds the k/v to the routine's root context. It can be retrieved with routine.Context().Value()
func GoWithValues(key, val interface{}) GoOpt {
	return func(o *goOpts) {
		o.key = key
		o.val = val
	}
}

// opts are options when creating a machine instance
type option struct {
	id string
	// MaxRoutines throttles goroutines at the given count
	maxRoutines int
	children    []*Machine
	middlewares []Middleware
	pubsub      pubsub.PubSub
	tags        []string
	key         interface{}
	val         interface{}
	timeout     time.Duration
	deadline    time.Time
	closers     []func()
}

// Opt is a single option when creating a machine instance with New
type Opt func(o *option)

// WithMaxRoutines throttles goroutines at the input number. It will panic if <= zero.
func WithMaxRoutines(max int) Opt {
	return func(o *option) {
		if max <= 0 {
			panic("max routines must be greater than zero!")
		}
		o.maxRoutines = max
	}
}

// WithPubSub sets the pubsub implementation for the machine instance. An inmemory implementation is used if none is provided.
func WithPubSub(pubsub pubsub.PubSub) Opt {
	return func(o *option) {
		o.pubsub = pubsub
	}
}

// WithChildren sets the machine instances children
func WithChildren(children ...*Machine) Opt {
	return func(o *option) {
		o.children = append(o.children, children...)
	}
}

// WithMiddlewares wraps every goroutine function executed by the machine with the given middlewares.
// Middlewares can be added to individual goroutines with GoWithMiddlewares
func WithMiddlewares(middlewares ...Middleware) Opt {
	return func(o *option) {
		o.middlewares = append(o.middlewares, middlewares...)
	}
}

// WithTags sets the machine instances tags
func WithTags(tags ...string) Opt {
	return func(o *option) {
		o.tags = append(o.tags, tags...)
		sort.Strings(o.tags)
	}
}

// WithValue adds the k/v to the Machine's root context. It can be retrieved with context.Value() in all sub routine contexts
func WithValue(key, val interface{}) Opt {
	return func(o *option) {
		o.key = key
		o.val = val
	}
}

// WithTimeout is an Opt that creates the Machine's context with the given timeout value
func WithTimeout(to time.Duration) Opt {
	return func(o *option) {
		o.timeout = to
	}
}

// WithDeadline is an Opt that creates the Machine's context with the given deadline.
func WithDeadline(deadline time.Time) Opt {
	return func(o *option) {
		o.deadline = deadline
	}
}

// WithID sets the machine instances unique id. If one isn't provided, a unique id will be assigned
func WithID(id string) Opt {
	return func(o *option) {
		o.id = id
	}
}

// WithClosers makes the Machine instance execute the given closers before it closes
func WithClosers(closers ...func()) Opt {
	return func(o *option) {
		o.closers = append(o.closers, closers...)
	}
}
