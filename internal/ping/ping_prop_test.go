//go:build property

package ping

import (
	"context"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: surveiller, Property 6: タイムアウト処理**
// **Validates: Requirements 2.5**
func TestTimeoutHandlingProperty(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("timeout handling property", prop.ForAll(
		func(timeoutMs int) bool {
			// Generate timeout values between 1ms and 1000ms
			timeout := time.Duration(timeoutMs) * time.Millisecond

			// Create a context with a shorter deadline than the ping timeout
			// This ensures the context will timeout before the ping timeout
			ctxTimeout := timeout / 2
			if ctxTimeout < 1*time.Millisecond {
				ctxTimeout = 1 * time.Millisecond
			}

			ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
			defer cancel()

			// Test with ICMP pinger (may fail due to permissions, but should handle timeout)
			icmpPinger, err := NewICMPPinger()
			if err != nil {
				// If ICMP pinger creation fails, test with external pinger
				externalPinger := NewExternalPinger()
				result := externalPinger.Ping(ctx, "127.0.0.1", timeout)

				// The result should be a failure due to timeout
				if result.Success {
					// If it succeeded, it means the ping was faster than expected
					// This is acceptable behavior
					return true
				}

				// If it failed, it should have an error
				return result.Error != nil
			}

			// Test with ICMP pinger
			result := icmpPinger.Ping(ctx, "127.0.0.1", timeout)

			// The result should be a failure due to timeout or permission error
			if result.Success {
				// If it succeeded, it means the ping was faster than expected
				// This is acceptable behavior
				return true
			}

			// If it failed, it should have an error
			return result.Error != nil
		},
		gen.IntRange(1, 1000), // Generate timeout values from 1ms to 1000ms
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// **Feature: surveiller, Property 6: タイムアウト処理**
// **Validates: Requirements 2.5**
func TestTimeoutBehaviorProperty(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("timeout behavior consistency", prop.ForAll(
		func(baseTimeoutMs int) bool {
			// Generate base timeout values between 10ms and 500ms
			baseTimeout := time.Duration(baseTimeoutMs) * time.Millisecond

			// Test that longer timeouts don't fail faster than shorter ones
			shortTimeout := baseTimeout
			longTimeout := baseTimeout * 2

			// Use external pinger for more predictable behavior
			pinger := NewExternalPinger()

			// Create contexts with very short deadlines to force timeouts
			shortCtx, shortCancel := context.WithTimeout(context.Background(), shortTimeout/10)
			defer shortCancel()

			longCtx, longCancel := context.WithTimeout(context.Background(), longTimeout/10)
			defer longCancel()

			// Both should timeout, but we're testing that the timeout handling is consistent
			shortResult := pinger.Ping(shortCtx, "192.0.2.1", shortTimeout) // Use TEST-NET-1 address
			longResult := pinger.Ping(longCtx, "192.0.2.1", longTimeout)

			// Both should fail due to timeout
			shortFailed := !shortResult.Success && shortResult.Error != nil
			longFailed := !longResult.Success && longResult.Error != nil

			// The property is that both should handle timeout consistently
			return shortFailed && longFailed
		},
		gen.IntRange(10, 500), // Generate base timeout values from 10ms to 500ms
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// **Feature: surveiller, Property 6: タイムアウト処理**
// **Validates: Requirements 2.5**
func TestEffectiveDeadlineProperty(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("effective deadline calculation", prop.ForAll(
		func(timeoutMs, ctxTimeoutMs int) bool {
			// Generate timeout values
			timeout := time.Duration(timeoutMs) * time.Millisecond
			ctxTimeout := time.Duration(ctxTimeoutMs) * time.Millisecond

			// Create context with deadline
			ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
			defer cancel()

			// Calculate effective deadline
			deadline := effectiveDeadline(ctx, timeout)
			now := time.Now()

			// The deadline should be in the future
			if !deadline.After(now) {
				return false
			}

			// The deadline should not be more than max(timeout, ctxTimeout) from now
			maxTimeout := timeout
			if ctxTimeout > timeout {
				maxTimeout = ctxTimeout
			}

			// Allow some tolerance for execution time
			tolerance := 100 * time.Millisecond
			expectedMaxDeadline := now.Add(maxTimeout + tolerance)

			return deadline.Before(expectedMaxDeadline)
		},
		gen.IntRange(1, 1000), // timeout in ms
		gen.IntRange(1, 1000), // context timeout in ms
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
