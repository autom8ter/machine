package machine

import "time"

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
