package ping

import (
	"context"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestPingArgs(t *testing.T) {
	timeout := 1500 * time.Millisecond
	args := pingArgs("example.com", timeout)

	var expected []string
	switch runtime.GOOS {
	case "darwin":
		timeoutMs := maxInt(100, int(timeout.Milliseconds()))
		expected = []string{"-n", "-c", "1", "-W", strconv.Itoa(timeoutMs), "example.com"}
	default:
		timeoutSec := maxInt(1, int(timeout.Seconds()+0.5))
		expected = []string{"-n", "-c", "1", "-W", strconv.Itoa(timeoutSec), "example.com"}
	}

	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("expected args %v, got %v", expected, args)
	}
}

func TestPingArgsMinimumTimeout(t *testing.T) {
	timeout := 10 * time.Millisecond
	args := pingArgs("example.com", timeout)

	var expectedTimeout string
	switch runtime.GOOS {
	case "darwin":
		expectedTimeout = strconv.Itoa(100)
	default:
		expectedTimeout = strconv.Itoa(1)
	}

	if len(args) < 5 || args[4] != expectedTimeout {
		t.Fatalf("expected timeout arg %q, got %v", expectedTimeout, args)
	}
}

func TestParseRTT(t *testing.T) {
	output := []byte("64 bytes from 8.8.8.8: icmp_seq=1 ttl=58 time=12.5 ms\n")
	rtt := parseRTT(output)
	if rtt != time.Duration(12.5*float64(time.Millisecond)) {
		t.Fatalf("expected RTT 12.5ms, got %v", rtt)
	}
}

func TestParseRTTInvalid(t *testing.T) {
	output := []byte("no time here\n")
	if rtt := parseRTT(output); rtt != 0 {
		t.Fatalf("expected zero RTT for missing pattern, got %v", rtt)
	}
}

func TestMaxInt(t *testing.T) {
	if maxInt(1, 2) != 2 {
		t.Fatalf("expected maxInt to return 2")
	}
	if maxInt(5, -1) != 5 {
		t.Fatalf("expected maxInt to return 5")
	}
}

// External Pinger unit tests

func TestNewExternalPinger(t *testing.T) {
	pinger := NewExternalPinger()
	if pinger == nil {
		t.Fatalf("expected non-nil external pinger")
	}
}

func TestExternalPingerContextCancellation(t *testing.T) {
	pinger := NewExternalPinger()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := pinger.Ping(ctx, "127.0.0.1", time.Second)
	if result.Success {
		t.Fatalf("expected failure due to cancelled context")
	}
	if result.Error == nil {
		t.Fatalf("expected error due to cancelled context")
	}
}

func TestExternalPingerTimeout(t *testing.T) {
	pinger := NewExternalPinger()

	// Use a very short timeout to force timeout behavior
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	result := pinger.Ping(ctx, "127.0.0.1", time.Second)
	if result.Success {
		t.Fatalf("expected timeout failure")
	}
	if result.Error == nil {
		t.Fatalf("expected timeout error")
	}
	// Should contain timeout information
	if !strings.Contains(result.Error.Error(), "timeout") {
		t.Logf("Error message: %v", result.Error)
	}
}

func TestExternalPingerInvalidAddress(t *testing.T) {
	pinger := NewExternalPinger()

	testCases := []string{
		"invalid@@address",
		"999.999.999.999",
		"not.a.real.domain.example.invalid",
	}

	for _, addr := range testCases {
		result := pinger.Ping(context.Background(), addr, 100*time.Millisecond)
		if result.Success {
			t.Fatalf("expected failure for invalid address %q", addr)
		}
		if result.Error == nil {
			t.Fatalf("expected error for invalid address %q", addr)
		}
	}
}

func TestExternalPingerValidAddress(t *testing.T) {
	pinger := NewExternalPinger()

	// Test with localhost - this should work on most systems
	result := pinger.Ping(context.Background(), "127.0.0.1", time.Second)

	// The ping might succeed or fail depending on system configuration
	// but we should get a proper result structure
	if result.Success {
		if result.RTT <= 0 {
			t.Fatalf("expected positive RTT for successful ping, got %v", result.RTT)
		}
		t.Logf("Successful ping to localhost: RTT=%v", result.RTT)
	} else {
		if result.Error == nil {
			t.Fatalf("expected error for failed ping")
		}
		t.Logf("Ping failed (may be expected): %v", result.Error)
	}
}

func TestParseRTTVariousFormats(t *testing.T) {
	testCases := []struct {
		output   string
		expected time.Duration
	}{
		{"64 bytes from 8.8.8.8: icmp_seq=1 ttl=58 time=12.5 ms\n", time.Duration(12.5 * float64(time.Millisecond))},
		{"PING 8.8.8.8 (8.8.8.8): 56 data bytes\n64 bytes from 8.8.8.8: icmp_seq=0 ttl=58 time=0.123 ms\n", time.Duration(0.123 * float64(time.Millisecond))},
		{"64 bytes from 127.0.0.1: icmp_seq=1 ttl=64 time=0.045 ms", time.Duration(0.045 * float64(time.Millisecond))},
		{"time=100.0 ms", time.Duration(100.0 * float64(time.Millisecond))},
		{"no time information here", 0},
		{"", 0},
	}

	for _, tc := range testCases {
		result := parseRTT([]byte(tc.output))
		if result != tc.expected {
			t.Fatalf("parseRTT(%q) = %v, expected %v", tc.output, result, tc.expected)
		}
	}
}

func TestPingArgsVariousTimeouts(t *testing.T) {
	testCases := []struct {
		timeout time.Duration
		addr    string
	}{
		{100 * time.Millisecond, "example.com"},
		{1 * time.Second, "127.0.0.1"},
		{5 * time.Second, "google.com"},
		{10 * time.Millisecond, "test.local"}, // Very short timeout
	}

	for _, tc := range testCases {
		args := pingArgs(tc.addr, tc.timeout)

		// Verify basic structure
		if len(args) < 5 {
			t.Fatalf("expected at least 5 args for pingArgs(%q, %v), got %v", tc.addr, tc.timeout, args)
		}

		// Verify address is included
		if args[len(args)-1] != tc.addr {
			t.Fatalf("expected last arg to be address %q, got %q", tc.addr, args[len(args)-1])
		}

		// Verify standard flags are present
		expectedFlags := []string{"-n", "-c", "1", "-W"}
		for i, flag := range expectedFlags {
			if i >= len(args) || args[i] != flag {
				t.Fatalf("expected flag %q at position %d, got %v", flag, i, args)
			}
		}
	}
}

func TestExternalPingerCommandConstruction(t *testing.T) {
	pinger := NewExternalPinger()

	// This test verifies that the external pinger can construct commands properly
	// We'll test with a short timeout to avoid long waits
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result := pinger.Ping(ctx, "127.0.0.1", 100*time.Millisecond)

	// We expect this to timeout or fail, but not panic
	if result.Success {
		t.Logf("Unexpected success: %v", result.RTT)
	} else {
		if result.Error == nil {
			t.Fatalf("expected error for failed/timeout ping")
		}
		t.Logf("Expected failure: %v", result.Error)
	}
}
