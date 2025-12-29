package ping

import (
	"context"
	"errors"
	"net"
	"os"
	"syscall"
	"testing"
	"time"
)

type stubPinger struct {
	result Result
	calls  int
}

func (s *stubPinger) Ping(ctx context.Context, addr string, timeout time.Duration) Result {
	s.calls++
	return s.result
}

func TestResolveIPValid(t *testing.T) {
	ipAddr, ip, err := resolveIP("127.0.0.1")
	if err != nil {
		t.Fatalf("expected valid IP, got error: %v", err)
	}
	if ipAddr == nil || ip == nil {
		t.Fatalf("expected resolved IP address, got nil")
	}
	if ip.To4() == nil {
		t.Fatalf("expected IPv4 address, got %v", ip)
	}
}

func TestResolveIPInvalid(t *testing.T) {
	_, _, err := resolveIP("invalid@@")
	if err == nil {
		t.Fatalf("expected error for invalid address")
	}
}

func TestICMPSettings(t *testing.T) {
	ipv4 := net.ParseIP("127.0.0.1")
	network, _, _, _ := icmpSettings(ipv4)
	if network != "ip4:icmp" {
		t.Fatalf("expected ipv4 network, got %q", network)
	}

	ipv6 := net.ParseIP("2001:db8::1")
	network, _, _, _ = icmpSettings(ipv6)
	if network != "ip6:ipv6-icmp" {
		t.Fatalf("expected ipv6 network, got %q", network)
	}
}

func TestEffectiveDeadlineUsesContextDeadline(t *testing.T) {
	ctxDeadline := time.Now().Add(50 * time.Millisecond)
	ctx, cancel := context.WithDeadline(context.Background(), ctxDeadline)
	defer cancel()

	deadline := effectiveDeadline(ctx, time.Second)
	if !deadline.Equal(ctxDeadline) {
		t.Fatalf("expected context deadline %v, got %v", ctxDeadline, deadline)
	}
}

func TestEffectiveDeadlineUsesTimeout(t *testing.T) {
	start := time.Now()
	deadline := effectiveDeadline(context.Background(), 25*time.Millisecond)
	if deadline.Before(start) {
		t.Fatalf("expected deadline after start, got %v", deadline)
	}
	if deadline.After(start.Add(75 * time.Millisecond)) {
		t.Fatalf("expected deadline within timeout window, got %v", deadline)
	}
}

func TestIsPermissionError(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{err: nil, want: false},
		{err: os.ErrPermission, want: true},
		{err: syscall.EPERM, want: true},
		{err: errors.New("operation not permitted"), want: true},
		{err: errors.New("permission denied"), want: true},
		{err: errors.New("other failure"), want: false},
	}

	for _, tc := range cases {
		if got := isPermissionError(tc.err); got != tc.want {
			t.Fatalf("isPermissionError(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}

func TestFallbackPingerUsesPrimaryOnSuccess(t *testing.T) {
	primary := &stubPinger{result: Result{Success: true}}
	secondary := &stubPinger{result: Result{Success: true}}
	pinger := NewFallbackPinger(primary, secondary)

	result := pinger.Ping(context.Background(), "127.0.0.1", time.Second)
	if !result.Success {
		t.Fatalf("expected success result")
	}
	if primary.calls != 1 || secondary.calls != 0 {
		t.Fatalf("expected primary called once and secondary not called, got %d/%d", primary.calls, secondary.calls)
	}
}

func TestFallbackPingerFallsBackOnPermissionError(t *testing.T) {
	primary := &stubPinger{result: Result{Success: false, Error: os.ErrPermission}}
	secondary := &stubPinger{result: Result{Success: true}}
	pinger := NewFallbackPinger(primary, secondary)

	result := pinger.Ping(context.Background(), "127.0.0.1", time.Second)
	if !result.Success {
		t.Fatalf("expected fallback success result")
	}
	if primary.calls != 1 || secondary.calls != 1 {
		t.Fatalf("expected both pingers called, got %d/%d", primary.calls, secondary.calls)
	}
}

func TestFallbackPingerSkipsFallbackOnOtherErrors(t *testing.T) {
	primary := &stubPinger{result: Result{Success: false, Error: errors.New("network down")}}
	secondary := &stubPinger{result: Result{Success: true}}
	pinger := NewFallbackPinger(primary, secondary)

	result := pinger.Ping(context.Background(), "127.0.0.1", time.Second)
	if result.Success {
		t.Fatalf("expected primary error result")
	}
	if primary.calls != 1 || secondary.calls != 0 {
		t.Fatalf("expected only primary called, got %d/%d", primary.calls, secondary.calls)
	}
}
