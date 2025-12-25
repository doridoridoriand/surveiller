package ui

import (
	"context"
	"time"

	"github.com/doridoridoriand/deadman-go/internal/config"
	"github.com/doridoridoriand/deadman-go/internal/state"
)

// UI provides a placeholder TUI loop for future implementation.
type UI struct {
	cfg   config.GlobalOptions
	state state.Store
}

// New returns a UI instance.
func New(cfg config.GlobalOptions, store state.Store) *UI {
	return &UI{cfg: cfg, state: store}
}

// Run blocks until the context is cancelled, periodically polling state.
func (u *UI) Run(ctx context.Context) error {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			_ = u.state.GetSnapshot()
		}
	}
}
