package machine

import "time"

// goOpts holds options for creating a goroutine. It is configured via GoOpt functions.
type goOpts struct {
	id      string
	tags    []string
	timeout *time.Duration
}

// GoOpt is a function that configures GoOpts
type GoOpt func(o *goOpts)

// WithTags is a GoOpt that adds an array of strings as "tags" to the Routine.
func WithTags(tags ...string) GoOpt {
	return func(o *goOpts) {
		o.tags = append(o.tags, tags...)
	}
}

// WithID is a GoOpt that sets/overrides the ID of the Routine. A random uuid is assigned if this option is not used.
func WithID(id string) GoOpt {
	return func(o *goOpts) {
		o.id = id
	}
}

// WithTimeout is a GoOpt that creates the Routine's context with the given timeout value
func WithTimeout(to time.Duration) GoOpt {
	return func(o *goOpts) {
		o.timeout = &to
	}
}

// opts are options when creating a machine instance
type option struct {
	middlewares []Middleware
	// MaxRoutines throttles goroutines at the given count
	maxRoutines int
	// Debug enables debug logs
	debug            bool
	pubChannelLength int
	subChannelLength int
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

// WithDebug enables debug mode
func WithDebug(debug bool) Opt {
	return func(o *option) {
		o.debug = debug
	}
}

// WithPublishChannelBuffer sets the buffer length of the channel returned from a goroutine publishTo
func WithPublishChannelBuffer(length int) Opt {
	return func(o *option) {
		o.pubChannelLength = length
	}
}

// WithSubscribeChannelBuffer sets the buffer length of the channel returned from a Routine subscribeTo
func WithSubscribeChannelBuffer(length int) Opt {
	return func(o *option) {
		o.subChannelLength = length
	}
}

// WithMiddlewares adds middlewares to the machine that will wrap every machine.Go Func that is executed by the machine instance.
func WithMiddlewares(middlewares ...Middleware) Opt {
	return func(o *option) {
		o.middlewares = append(o.middlewares, middlewares...)
	}
}
