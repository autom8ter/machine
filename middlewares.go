package machine

import (
	"fmt"
	"time"
)

// Cron is a middleware that execute the function every time the ticker ticks until the goroutine's context cancels
func Cron(ticker *time.Ticker) Middleware {
	return func(fn Func) Func {
		return func(routine Routine) {
			defer ticker.Stop()
			for {
				select {
				case <-routine.Context().Done():
					return
				case <-ticker.C:
					fn(routine)
				}
			}
		}
	}
}

// While is a middleware that will continue to execute the Func while deciderFunc() returns true.
// The loop breaks the first time deciderFunc() returns false or the routine's context cancels
func While(deciderFunc func(routine Routine) bool) Middleware {
	return func(fn Func) Func {
		return func(routine Routine) {
			for {
				select {
				case <-routine.Context().Done():
					return
				default:
					if !deciderFunc(routine) {
						return
					}
					fn(routine)
				}
			}
		}
	}
}

// After exectues the afterFunc after the main goroutine exits.
func After(afterFunc func(routine Routine)) Middleware {
	return func(fn Func) Func {
		return func(routine Routine) {
			fn(routine)
			afterFunc(routine)
		}
	}
}

// Before exectues the beforeFunc before the main goroutine is executed.
func Before(beforeFunc func(routine Routine)) Middleware {
	return func(fn Func) Func {
		return func(routine Routine) {
			beforeFunc(routine)
			fn(routine)
		}
	}
}

// Decider exectues the deciderFunc before the main goroutine is executed.
// If it returns false, the goroutine won't be executed.
func Decider(deciderFunc func(routine Routine) bool) Middleware {
	return func(fn Func) Func {
		return func(routine Routine) {
			if deciderFunc(routine) {
				fn(routine)
			}
		}
	}
}

// PanicRecover wraps a goroutine with a middleware the recovers from panics.
func PanicRecover() Middleware {
	return func(fn Func) Func {
		return func(routine Routine) {
			defer func() {
				r := recover()
				if _, ok := r.(error); ok {
					fmt.Println("machine: panic recovered")
				}
			}()
			fn(routine)
		}
	}
}
