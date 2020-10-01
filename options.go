package machine

// opts are options when creating a machine instance
type option struct {
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
