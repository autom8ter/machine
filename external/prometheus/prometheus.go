package prometheus

import (
	"github.com/autom8ter/machine"
	"github.com/prometheus/client_golang/prometheus"
	"strings"
)

func init() {
	prometheus.MustRegister(
		startedRoutines,
		routineDuration,
		finishedRoutines,
	)
}

var (
	routineDuration = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "machine_routine_duration_gauge",
			Help: "gauge the execution time of each routine",
		},
		[]string{"machine_id", "machine_tags", "routine_tags"})
	startedRoutines = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "machine_started_routine_counter",
		Help: "total routines started",
	},
		[]string{"machine_id", "machine_tags", "routine_tags"})
	finishedRoutines = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "machine_finished_routine_counter",
		Help: "total routines finished",
	},
		[]string{"machine_id", "machine_tags", "routine_tags"})
)

// Middleware is a Machine.Middleware that wraps goroutine's with prometheus metrics
/*
Name: "machine_routine_duration_gauge",
Help: "gauge the execution time of each routine",d",
Tags: "machine_id", "machine_tags", "routine_tags"

Name: "machine_started_routine_counter",
Help: "total routines started",
Tags: "machine_id", "machine_tags", "routine_tags"

Name: "machine_finished_routine_counter",
Help: "total routines finished",
Tags: "machine_id", "machine_tags", "routine_tags"
*/
func Middleware() machine.Middleware {
	return func(fn machine.Func) machine.Func {
		return func(routine machine.Routine) {
			mjoined := strings.Join(routine.Machine().Tags(), ",")
			joined := strings.Join(routine.Tags(), ",")
			defer routineDuration.WithLabelValues(routine.Machine().ID(), mjoined, joined).Set(float64(routine.Duration().Nanoseconds()))
			defer finishedRoutines.WithLabelValues(routine.Machine().ID(), mjoined, joined).Inc()
			startedRoutines.WithLabelValues(routine.Machine().ID(), mjoined, joined).Inc()
			fn(routine)
		}
	}
}
