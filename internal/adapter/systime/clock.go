// Package systime adapts the system clock to the Clock port.
package systime

import (
	"time"

	"github.com/mateuszmierzwinski/basalt/internal/port"
)

type clock struct{}

// New builds the real clock.
func New() port.Clock {
	return &clock{}
}

// Now returns the current wall-clock time.
func (c *clock) Now() time.Time {
	return time.Now()
}
