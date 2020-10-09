package cron

import (
	"github.com/autom8ter/machine"
	"time"
)

// Every executes the function every time the ticker ticks until the context cancels
func Every(ticker *time.Ticker, fn machine.Func) machine.Func {
	return func(routine machine.Routine) {
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

// HourOfDay executes the function on a give hour of the day
func HourOfDay(hourOfDay int, fn machine.Func) machine.Func {
	return func(routine machine.Routine) {
		for {
			select {
			case <-routine.Context().Done():
				return
			default:
				if time.Now().Hour() == hourOfDay {
					fn(routine)
				}
				time.Sleep(15 * time.Minute)
			}
		}
	}
}
