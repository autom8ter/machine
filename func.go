package machine

// Func is the function passed into machine.Go. The Routine is passed into this function at runtime.
type Func func(routine Routine) error

// Middleware is a function that wraps/modifies the behavior of a machine.Func.
type Middleware func(fn Func) Func
