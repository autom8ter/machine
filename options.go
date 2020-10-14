package machine

import (
	"time"
)

// goOpts holds options for creating a goroutine. It is configured via GoOpt functions.
type goOpts struct {
	id          int
	tags        []string
	timeout     *time.Duration
	middlewares []Middleware
	data        map[interface{}]interface{}
}

// GoOpt is a function that configures GoOpts
type GoOpt func(o *goOpts)

// GoWithTags is a GoOpt that adds an array of strings as "tags" to the Routine.
func GoWithTags(tags ...string) GoOpt {
	return func(o *goOpts) {
		o.tags = append(o.tags, tags...)
	}
}

// GoWithPID is a GoOpt that sets/overrides the process ID of the Routine. A random id is assigned if this option is not used.
func GoWithPID(id int) GoOpt {
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

// GoWithMiddlewares wraps the gived function with the input middlewares.
func GoWithMiddlewares(middlewares ...Middleware) GoOpt {
	return func(o *goOpts) {
		o.middlewares = append(o.middlewares, middlewares...)
	}
}

// GoWithValues adds the data to the machine's root context. It can be retrieved with
func GoWithValues(data map[interface{}]interface{}) GoOpt {
	return func(o *goOpts) {
		o.data = data
	}
}

// opts are options when creating a machine instance
type option struct {
	// MaxRoutines throttles goroutines at the given count
	maxRoutines int
	parent      *Machine
	children    []*Machine
	middlewares []Middleware
	pubsub      PubSub
	tags        []string
	data        map[interface{}]interface{}
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
func WithPubSub(pubsub PubSub) Opt {
	return func(o *option) {
		o.pubsub = pubsub
	}
}

func WithParent(parent *Machine) Opt {
	return func(o *option) {
		o.parent = parent
	}
}

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
func WithTags(tags []string) Opt {
	return func(o *option) {
		o.tags = append(o.tags, tags...)
	}
}

// WithValues adds the data to the machine's root context. It can be retrieved with
func WithValues(data map[interface{}]interface{}) Opt {
	return func(o *option) {
		o.data = data
	}
}
