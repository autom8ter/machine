package machine

import (
	"encoding/json"
	"fmt"
	"time"
)

// Stats holds information about goroutines
type Stats struct {
	TotalRoutines int            `json:"totalRoutines"`
	Routines      []RoutineStats `json:"routines"`
	TotalChildren int            `json:"totalChildren"`
	HasParent     bool           `json:"hasParent"`
}

// String prints a pretty json string of the stats
func (s Stats) String() string {
	bits, _ := json.MarshalIndent(&s, "", "    ")
	return fmt.Sprintf("%s", string(bits))
}

// RoutineStats holds information about a single goroutine
type RoutineStats struct {
	PID      int           `json:"pid"`
	Start    time.Time     `json:"start"`
	Duration time.Duration `json:"duration"`
	Tags     []string      `json:"tags"`
}
