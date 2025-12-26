package ping

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"time"
)

var timePattern = regexp.MustCompile(`time=([0-9.]+)\s*ms`)

// ExternalPinger invokes the system ping command for environments without raw socket access.
type ExternalPinger struct{}

// NewExternalPinger returns a ping implementation that shells out to ping.
func NewExternalPinger() *ExternalPinger {
	return &ExternalPinger{}
}

// Ping runs the system ping command and parses the RTT from stdout.
func (p *ExternalPinger) Ping(ctx context.Context, addr string, timeout time.Duration) Result {
	args := pingArgs(addr, timeout)
	start := time.Now()
	cmd := exec.CommandContext(ctx, "ping", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Result{Success: false, Error: fmt.Errorf("external ping failed: %w", err)}
	}

	rtt := parseRTT(out)
	if rtt == 0 {
		rtt = time.Since(start)
	}
	return Result{Success: true, RTT: rtt}
}

func pingArgs(addr string, timeout time.Duration) []string {
	switch runtime.GOOS {
	case "darwin":
		timeoutMs := maxInt(100, int(timeout.Milliseconds()))
		return []string{"-n", "-c", "1", "-W", strconv.Itoa(timeoutMs), addr}
	default:
		timeoutSec := maxInt(1, int(timeout.Seconds()+0.5))
		return []string{"-n", "-c", "1", "-W", strconv.Itoa(timeoutSec), addr}
	}
}

func parseRTT(output []byte) time.Duration {
	matches := timePattern.FindSubmatch(output)
	if len(matches) < 2 {
		return 0
	}
	value, err := strconv.ParseFloat(string(matches[1]), 64)
	if err != nil {
		return 0
	}
	return time.Duration(value * float64(time.Millisecond))
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
