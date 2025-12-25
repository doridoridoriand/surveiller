package ping

import (
	"context"
	"time"
)

// Result captures a single ping result.
type Result struct {
	RTT     time.Duration
	Success bool
	Error   error
}

// Pinger sends a single ping and returns the result.
type Pinger interface {
	Ping(ctx context.Context, addr string, timeout time.Duration) Result
}
