package machine

import "time"

// goOpts holds options for creating a goroutine. It is configured via GoOpt functions.
type goOpts struct {
	id          int
	tags        []string
	timeout     *time.Duration
	middlewares []Middleware
}

// GoOpt is a function that configures GoOpts
type GoOpt func(o *goOpts)

// WithTags is a GoOpt that adds an array of strings as "tags" to the Routine.
func WithTags(tags ...string) GoOpt {
	return func(o *goOpts) {
		o.tags = append(o.tags, tags...)
	}
}

// WithPID is a GoOpt that sets/overrides the process ID of the Routine. A random id is assigned if this option is not used.
func WithPID(id int) GoOpt {
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

// WithMiddlewares wraps the gived function with the input middlewares.
func WithMiddlewares(middlewares ...Middleware) GoOpt {
	return func(o *goOpts) {
		o.middlewares = append(o.middlewares, middlewares...)
	}
}

// opts are options when creating a machine instance
type option struct {
	// MaxRoutines throttles goroutines at the given count
	maxRoutines      int
	subChannelLength int
	cache            Cache
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

// WithSubscribeChannelBuffer sets the buffer length of the channel returned from a Routine subscribeTo
func WithSubscribeChannelBuffer(length int) Opt {
	return func(o *option) {
		o.subChannelLength = length
	}
}

// WithCache sets the in memory, concurrency safe cache. If not set, a default sync.Map implementation is used.
func WithCache(cache Cache) Opt {
	return func(o *option) {
		o.cache = cache
	}
}
