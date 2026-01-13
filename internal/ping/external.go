package ping

import (
	"context"
	"fmt"
	"net"
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
	cmdName := pingCommand(addr)
	start := time.Now()
	cmd := exec.CommandContext(ctx, cmdName, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// contextがtimeoutでキャンセルされた場合をチェック
		if ctx.Err() == context.DeadlineExceeded {
			return Result{Success: false, Error: fmt.Errorf("ping timeout: %w", ctx.Err())}
		}
		return Result{Success: false, Error: fmt.Errorf("external ping failed: %w", err)}
	}

	rtt := parseRTT(out)
	if rtt == 0 {
		rtt = time.Since(start)
	}
	return Result{Success: true, RTT: rtt}
}

// pingCommand returns the appropriate ping command name for the given address.
// On macOS, IPv6 addresses require ping6 command.
func pingCommand(addr string) string {
	if runtime.GOOS == "darwin" && isIPv6(addr) {
		return "ping6"
	}
	return "ping"
}

// isIPv6 checks if the given address is an IPv6 address.
func isIPv6(addr string) bool {
	ip := net.ParseIP(addr)
	if ip == nil {
		// If parsing fails, try to resolve it
		ipAddr, err := net.ResolveIPAddr("ip", addr)
		if err != nil {
			return false
		}
		ip = ipAddr.IP
	}
	return ip != nil && ip.To4() == nil
}

func pingArgs(addr string, timeout time.Duration) []string {
	isIPv6Addr := isIPv6(addr)
	
	switch runtime.GOOS {
	case "darwin":
		if isIPv6Addr {
			// macOS ping6 doesn't support -W option, timeout is handled by context
			return []string{"-n", "-c", "1", addr}
		}
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
