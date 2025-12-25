package state

import (
	"time"

	"github.com/doridoridoriand/deadman-go/internal/config"
	"github.com/doridoridoriand/deadman-go/internal/ping"
)

// Status represents target health.
type Status string

const (
	StatusUnknown Status = "UNKNOWN"
	StatusOK      Status = "OK"
	StatusWarn    Status = "WARN"
	StatusDown    Status = "DOWN"
)

// RTTPoint records a single RTT measurement.
type RTTPoint struct {
	Time time.Time
	RTT  time.Duration
}

// TargetStatus captures the current state and history for a target.
type TargetStatus struct {
	Name          string
	Address       string
	Group         string
	LastRTT       time.Duration
	LastSuccessAt time.Time
	LastFailureAt time.Time
	ConsecutiveOK int
	ConsecutiveNG int
	Status        Status
	History       []RTTPoint
}

// Store defines operations for tracking target state.
type Store interface {
	UpdateResult(name string, result ping.Result)
	GetSnapshot() []TargetStatus
	UpdateTargets(targets []config.TargetConfig)
	GetTargetStatus(name string) (TargetStatus, bool)
}
