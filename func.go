package machine

// Func is the function passed into machine.Go. The Routine is passed into this function at runtime.
type Func func(routine Routine)

// Middleware is a function that wraps/modifies the behavior of a machine.Func.
type Middleware func(fn Func) Func

type work struct {
	opts *goOpts
	fn   Func
}
