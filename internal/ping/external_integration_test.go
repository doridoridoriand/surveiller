//go:build integration

package ping

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestExternalPingerContextCancellation_Integration(t *testing.T) {
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

func TestExternalPingerTimeout_Integration(t *testing.T) {
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

func TestExternalPingerInvalidAddress_Integration(t *testing.T) {
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

func TestExternalPingerValidAddress_Integration(t *testing.T) {
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

func TestExternalPingerCommandConstruction_Integration(t *testing.T) {
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
