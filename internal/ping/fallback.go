package ping

import (
	"context"
	"errors"
	"os"
	"strings"
	"syscall"
	"time"
)

// FallbackPinger delegates to primary, then secondary when permission errors occur.
type FallbackPinger struct {
	primary   Pinger
	secondary Pinger
}

// NewFallbackPinger wraps primary with a secondary fallback.
func NewFallbackPinger(primary, secondary Pinger) *FallbackPinger {
	return &FallbackPinger{primary: primary, secondary: secondary}
}

// Ping uses the primary pinger and falls back on permission-related errors.
func (p *FallbackPinger) Ping(ctx context.Context, addr string, timeout time.Duration) Result {
	result := p.primary.Ping(ctx, addr, timeout)
	if result.Success || !isPermissionError(result.Error) {
		return result
	}
	return p.secondary.Ping(ctx, addr, timeout)
}

func isPermissionError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrPermission) || errors.Is(err, syscall.EPERM) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "operation not permitted") || strings.Contains(msg, "permission denied")
}
