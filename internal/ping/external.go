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
	isIPv6Addr := resolveIPv6(ctx, addr)
	args := pingArgsFor(addr, timeout, isIPv6Addr)
	cmdName := pingCommandFor(isIPv6Addr)
	cmd := exec.CommandContext(ctx, cmdName, args...)
	out, err := cmd.CombinedOutput()
	return resultFromExternalPing(ctx, out, err)
}

func resultFromExternalPing(ctx context.Context, output []byte, err error) Result {
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			if ctxErr == context.DeadlineExceeded {
				return Result{Success: false, Error: fmt.Errorf("ping timeout: %w", ctxErr)}
			}
			return Result{Success: false, Error: ctxErr}
		}
		return Result{Success: false, Error: fmt.Errorf("external ping failed: %w", err)}
	}

	rtt, ok := parseRTTValue(output)
	if !ok {
		return Result{Success: false, Error: fmt.Errorf("external ping output did not contain RTT")}
	}
	return Result{Success: true, RTT: rtt}
}

// pingCommandFor returns the appropriate ping command name for an address family.
// On macOS, IPv6 addresses require ping6 command.
func pingCommandFor(isIPv6Addr bool) string {
	if runtime.GOOS == "darwin" && isIPv6Addr {
		return "ping6"
	}
	return "ping"
}

// isIPv6 checks if the given address is an IPv6 address.
func isIPv6(addr string) bool {
	return resolveIPv6(context.Background(), addr)
}

func resolveIPv6(ctx context.Context, addr string) bool {
	ip := net.ParseIP(addr)
	if ip != nil {
		return ip.To4() == nil
	}

	lookupCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	resolver := &net.Resolver{PreferGo: true}
	ips, err := resolver.LookupIPAddr(lookupCtx, addr)
	if err != nil || len(ips) == 0 {
		return false
	}
	return ips[0].IP.To4() == nil
}

func pingArgsFor(addr string, timeout time.Duration, isIPv6Addr bool) []string {
	switch runtime.GOOS {
	case "darwin":
		if isIPv6Addr {
			// macOS ping6 doesn't support -W but supports -t (per-packet timeout in seconds)
			timeoutSec := max(1, int(timeout.Seconds()+0.5))
			return []string{"-n", "-c", "1", "-t", strconv.Itoa(timeoutSec), addr}
		}
		timeoutMs := max(100, int(timeout.Milliseconds()))
		return []string{"-n", "-c", "1", "-W", strconv.Itoa(timeoutMs), addr}
	default:
		timeoutSec := max(1, int(timeout.Seconds()+0.5))
		return []string{"-n", "-c", "1", "-W", strconv.Itoa(timeoutSec), addr}
	}
}

func parseRTT(output []byte) time.Duration {
	rtt, _ := parseRTTValue(output)
	return rtt
}

func parseRTTValue(output []byte) (time.Duration, bool) {
	matches := timePattern.FindSubmatch(output)
	if len(matches) < 2 {
		return 0, false
	}
	value, err := strconv.ParseFloat(string(matches[1]), 64)
	if err != nil {
		return 0, false
	}
	return time.Duration(value * float64(time.Millisecond)), true
}
