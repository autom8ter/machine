package machine

import (
	"encoding/json"
	"fmt"
	"time"
)

// Stats holds information about goroutines
type Stats struct {
	ID               string         `json:"id"`
	Tags             []string       `json:"tags"`
	TotalRoutines    int            `json:"totalRoutines"`
	ActiveRoutines   int            `json:"activeRoutines"`
	Routines         []RoutineStats `json:"routines"`
	TotalChildren    int            `json:"totalChildren"`
	HasParent        bool           `json:"hasParent"`
	TotalMiddlewares int            `json:"totalMiddlewares"`
	Timeout          time.Duration  `json:"timeout"`
	Deadline         time.Time      `json:"deadline"`
	Children         []*Stats       `json:"children"`
}

// String prints a pretty json string of the stats
func (s Stats) String() string {
	bits, _ := json.MarshalIndent(&s, "", "    ")
	return fmt.Sprintf("%s", string(bits))
}

// RoutineStats holds information about a single goroutine
type RoutineStats struct {
	PID      string        `json:"pid"`
	Start    time.Time     `json:"start"`
	Duration time.Duration `json:"duration"`
	Tags     []string      `json:"tags"`
}
