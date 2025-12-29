package ping

import (
	"reflect"
	"runtime"
	"strconv"
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
