package state

import (
	"testing"
	"time"

	"github.com/doridoridoriand/surveiller/internal/config"
	"github.com/doridoridoriand/surveiller/internal/ping"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/prop"
)

// **Feature: surveiller, Property 5: ping 結果の状態更新**
// **Validates: Requirements 2.2, 2.3, 2.4**
func TestPropertyPingResultStateUpdate(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	props := gopter.NewProperties(params)

	props.Property("ping success updates RTT and status correctly", prop.ForAll(
		func(rttMs int, timeoutMs int) bool {
			if rttMs < 1 || timeoutMs < 1 || rttMs > timeoutMs {
				return true // Skip invalid combinations
			}

			timeout := time.Duration(timeoutMs) * time.Millisecond
			rtt := time.Duration(rttMs) * time.Millisecond

			store := NewStore([]config.TargetConfig{
				{Name: "test", Address: "192.0.2.1"},
			}, timeout)

			// Send 10 successful pings with the same RTT to fill history
			for i := 0; i < 10; i++ {
				store.UpdateResult("test", ping.Result{Success: true, RTT: rtt})
			}

			status, ok := store.GetTargetStatus("test")
			if !ok {
				return false
			}

			// Verify RTT is recorded
			if status.LastRTT != rtt {
				return false
			}

			// Verify consecutive counters
			if status.ConsecutiveOK != 10 || status.ConsecutiveNG != 0 {
				return false
			}

			// Verify status based on RTT thresholds
			okThreshold := timeout / 4   // 25%
			warnThreshold := timeout / 2 // 50%

			if rtt <= okThreshold {
				return status.Status == StatusOK
			} else if rtt <= warnThreshold {
				return status.Status == StatusWarn
			} else {
				// Over 50% should also be WARN
				return status.Status == StatusWarn
			}
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(500) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(500) + 100
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.Property("ping failure updates counters and status correctly", prop.ForAll(
		func(failureCount int) bool {
			if failureCount < 1 || failureCount > 10 {
				return true
			}

			timeout := 100 * time.Millisecond
			store := NewStore([]config.TargetConfig{
				{Name: "test", Address: "192.0.2.1"},
			}, timeout)

			// Send consecutive failures
			for i := 0; i < failureCount; i++ {
				store.UpdateResult("test", ping.Result{Success: false, Error: errSentinel{}})
			}

			status, ok := store.GetTargetStatus("test")
			if !ok {
				return false
			}

			// Verify failure counters
			if status.ConsecutiveNG != failureCount || status.ConsecutiveOK != 0 {
				return false
			}

			// Verify status based on failure threshold (default is 3)
			if failureCount >= defaultDownThreshold {
				return status.Status == StatusDown
			} else {
				return status.Status == StatusWarn
			}
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(10) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.Property("success after failure resets counters", prop.ForAll(
		func(failureCount int, rttMs int) bool {
			if failureCount < 1 || failureCount > 10 || rttMs < 1 || rttMs > 100 {
				return true
			}

			timeout := 100 * time.Millisecond
			rtt := time.Duration(rttMs) * time.Millisecond
			store := NewStore([]config.TargetConfig{
				{Name: "test", Address: "192.0.2.1"},
			}, timeout)

			// Send failures
			for i := 0; i < failureCount; i++ {
				store.UpdateResult("test", ping.Result{Success: false, Error: errSentinel{}})
			}

			// Send 10 successes to fill history
			for i := 0; i < 10; i++ {
				store.UpdateResult("test", ping.Result{Success: true, RTT: rtt})
			}

			status, ok := store.GetTargetStatus("test")
			if !ok {
				return false
			}

			// Verify counters are reset
			if status.ConsecutiveNG != 0 || status.ConsecutiveOK != 10 {
				return false
			}

			// Verify status is based on RTT, not DOWN
			return status.Status != StatusDown
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(10) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(100) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.Property("history is recorded for successful pings", prop.ForAll(
		func(successCount int) bool {
			if successCount < 1 || successCount > 150 {
				return true
			}

			timeout := 100 * time.Millisecond
			store := NewStore([]config.TargetConfig{
				{Name: "test", Address: "192.0.2.1"},
			}, timeout)

			// Send successes
			for i := 0; i < successCount; i++ {
				store.UpdateResult("test", ping.Result{Success: true, RTT: time.Duration(i+1) * time.Millisecond})
			}

			status, ok := store.GetTargetStatus("test")
			if !ok {
				return false
			}

			// Verify history size is capped at defaultHistorySize
			expectedSize := successCount
			if expectedSize > defaultHistorySize {
				expectedSize = defaultHistorySize
			}

			return len(status.History) == expectedSize
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(150) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.TestingRun(t, gopter.ConsoleReporter(false))
}

// **Feature: surveiller, Property 14: 動的設定更新**
// **Validates: Requirements 5.2, 5.3**
func TestPropertyDynamicConfigUpdate(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	props := gopter.NewProperties(params)

	props.Property("adding targets starts monitoring for new targets", prop.ForAll(
		func(initialCount int, addCount int) bool {
			if initialCount < 1 || initialCount > 10 || addCount < 1 || addCount > 10 {
				return true
			}

			timeout := 100 * time.Millisecond

			// Create initial targets
			initialTargets := make([]config.TargetConfig, initialCount)
			for i := 0; i < initialCount; i++ {
				initialTargets[i] = config.TargetConfig{
					Name:    generateTargetName(i),
					Address: generateAddress(i),
					Group:   "group-1",
				}
			}

			store := NewStore(initialTargets, timeout)

			// Add some history to initial targets
			for i := 0; i < initialCount; i++ {
				store.UpdateResult(generateTargetName(i), ping.Result{Success: true, RTT: 10 * time.Millisecond})
			}

			// Create updated target list with new targets
			updatedTargets := make([]config.TargetConfig, initialCount+addCount)
			copy(updatedTargets, initialTargets)
			for i := 0; i < addCount; i++ {
				updatedTargets[initialCount+i] = config.TargetConfig{
					Name:    generateTargetName(initialCount + i),
					Address: generateAddress(initialCount + i),
					Group:   "group-2",
				}
			}

			// Update targets
			store.UpdateTargets(updatedTargets)

			// Verify all targets exist
			snapshot := store.GetSnapshot()
			if len(snapshot) != initialCount+addCount {
				return false
			}

			// Verify new targets are initialized with UNKNOWN status
			for i := initialCount; i < initialCount+addCount; i++ {
				status, ok := store.GetTargetStatus(generateTargetName(i))
				if !ok {
					return false
				}
				if status.Status != StatusUnknown {
					return false
				}
			}

			return true
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(10) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(10) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.Property("removing targets stops monitoring for deleted targets", prop.ForAll(
		func(initialCount int, removeCount int) bool {
			if initialCount < 2 || initialCount > 10 || removeCount < 1 || removeCount >= initialCount {
				return true
			}

			timeout := 100 * time.Millisecond

			// Create initial targets
			initialTargets := make([]config.TargetConfig, initialCount)
			for i := 0; i < initialCount; i++ {
				initialTargets[i] = config.TargetConfig{
					Name:    generateTargetName(i),
					Address: generateAddress(i),
				}
			}

			store := NewStore(initialTargets, timeout)

			// Create updated target list with some targets removed
			remainingCount := initialCount - removeCount
			updatedTargets := make([]config.TargetConfig, remainingCount)
			copy(updatedTargets, initialTargets[:remainingCount])

			// Update targets
			store.UpdateTargets(updatedTargets)

			// Verify only remaining targets exist
			snapshot := store.GetSnapshot()
			if len(snapshot) != remainingCount {
				return false
			}

			// Verify removed targets are gone
			for i := remainingCount; i < initialCount; i++ {
				_, ok := store.GetTargetStatus(generateTargetName(i))
				if ok {
					return false // Should not exist
				}
			}

			return true
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(9) + 2
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(5) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.Property("updating existing targets preserves history", prop.ForAll(
		func(targetCount int, historySize int) bool {
			if targetCount < 1 || targetCount > 10 || historySize < 1 || historySize > 20 {
				return true
			}

			timeout := 100 * time.Millisecond

			// Create initial targets
			initialTargets := make([]config.TargetConfig, targetCount)
			for i := 0; i < targetCount; i++ {
				initialTargets[i] = config.TargetConfig{
					Name:    generateTargetName(i),
					Address: generateAddress(i),
					Group:   "group-1",
				}
			}

			store := NewStore(initialTargets, timeout)

			// Add history to targets
			for i := 0; i < targetCount; i++ {
				for j := 0; j < historySize; j++ {
					store.UpdateResult(generateTargetName(i), ping.Result{
						Success: true,
						RTT:     time.Duration(j+1) * time.Millisecond,
					})
				}
			}

			// Get history before update
			historyBefore := make(map[string]int)
			for i := 0; i < targetCount; i++ {
				status, _ := store.GetTargetStatus(generateTargetName(i))
				historyBefore[generateTargetName(i)] = len(status.History)
			}

			// Update targets with modified addresses/groups
			updatedTargets := make([]config.TargetConfig, targetCount)
			for i := 0; i < targetCount; i++ {
				updatedTargets[i] = config.TargetConfig{
					Name:    generateTargetName(i),
					Address: generateAddress(i + 100), // Different address
					Group:   "group-2",                // Different group
				}
			}

			store.UpdateTargets(updatedTargets)

			// Verify history is preserved
			for i := 0; i < targetCount; i++ {
				status, ok := store.GetTargetStatus(generateTargetName(i))
				if !ok {
					return false
				}

				// History should be preserved
				if len(status.History) != historyBefore[generateTargetName(i)] {
					return false
				}

				// Address and group should be updated
				if status.Address != generateAddress(i+100) || status.Group != "group-2" {
					return false
				}
			}

			return true
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(10) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(20) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.TestingRun(t, gopter.ConsoleReporter(false))
}

func generateTargetName(index int) string {
	return "target-" + string(rune('a'+index%26)) + string(rune('0'+index/26))
}

func generateAddress(index int) string {
	return "192.0.2." + string(rune('0'+(index%250)))
}

// **Feature: surveiller, Property 16: 履歴保持**
// **Validates: Requirements 5.5**
func TestPropertyHistoryPreservation(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	props := gopter.NewProperties(params)

	props.Property("config reload preserves history for common targets", prop.ForAll(
		func(commonCount int, addCount int, removeCount int, historySize int) bool {
			if commonCount < 1 || commonCount > 10 || addCount < 0 || addCount > 5 ||
				removeCount < 0 || removeCount > 5 || historySize < 1 || historySize > 20 {
				return true
			}

			timeout := 100 * time.Millisecond

			// Create initial targets: common + to-be-removed
			totalInitial := commonCount + removeCount
			initialTargets := make([]config.TargetConfig, totalInitial)
			for i := 0; i < totalInitial; i++ {
				initialTargets[i] = config.TargetConfig{
					Name:    generateTargetName(i),
					Address: generateAddress(i),
					Group:   "group-1",
				}
			}

			store := NewStore(initialTargets, timeout)

			// Add history to all initial targets
			for i := 0; i < totalInitial; i++ {
				for j := 0; j < historySize; j++ {
					store.UpdateResult(generateTargetName(i), ping.Result{
						Success: true,
						RTT:     time.Duration(j+1) * time.Millisecond,
					})
				}
			}

			// Capture history for common targets before reload
			historyBefore := make(map[string][]RTTPoint)
			for i := 0; i < commonCount; i++ {
				status, _ := store.GetTargetStatus(generateTargetName(i))
				historyBefore[generateTargetName(i)] = append([]RTTPoint(nil), status.History...)
			}

			// Create new target list: common + new (removing some)
			newTargets := make([]config.TargetConfig, commonCount+addCount)
			// Keep common targets
			for i := 0; i < commonCount; i++ {
				newTargets[i] = config.TargetConfig{
					Name:    generateTargetName(i),
					Address: generateAddress(i),
					Group:   "group-1",
				}
			}
			// Add new targets
			for i := 0; i < addCount; i++ {
				newTargets[commonCount+i] = config.TargetConfig{
					Name:    generateTargetName(totalInitial + i),
					Address: generateAddress(totalInitial + i),
					Group:   "group-2",
				}
			}

			// Reload configuration
			store.UpdateTargets(newTargets)

			// Verify history is preserved for common targets
			for i := 0; i < commonCount; i++ {
				targetName := generateTargetName(i)
				status, ok := store.GetTargetStatus(targetName)
				if !ok {
					return false
				}

				// History should be preserved
				if len(status.History) != len(historyBefore[targetName]) {
					return false
				}

				// Verify history content matches
				for j := 0; j < len(status.History); j++ {
					if status.History[j].RTT != historyBefore[targetName][j].RTT {
						return false
					}
				}
			}

			// Verify removed targets are gone
			for i := commonCount; i < totalInitial; i++ {
				_, ok := store.GetTargetStatus(generateTargetName(i))
				if ok {
					return false // Should not exist
				}
			}

			// Verify new targets have no history
			for i := 0; i < addCount; i++ {
				status, ok := store.GetTargetStatus(generateTargetName(totalInitial + i))
				if !ok {
					return false
				}
				if len(status.History) != 0 {
					return false
				}
			}

			return true
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(10) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(6)
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(6)
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(20) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.Property("multiple reloads preserve history for persistent targets", prop.ForAll(
		func(targetCount int, reloadCount int, historySize int) bool {
			if targetCount < 1 || targetCount > 5 || reloadCount < 1 || reloadCount > 5 ||
				historySize < 1 || historySize > 10 {
				return true
			}

			timeout := 100 * time.Millisecond

			// Create initial targets
			initialTargets := make([]config.TargetConfig, targetCount)
			for i := 0; i < targetCount; i++ {
				initialTargets[i] = config.TargetConfig{
					Name:    generateTargetName(i),
					Address: generateAddress(i),
					Group:   "group-1",
				}
			}

			store := NewStore(initialTargets, timeout)

			// Add initial history
			for i := 0; i < targetCount; i++ {
				for j := 0; j < historySize; j++ {
					store.UpdateResult(generateTargetName(i), ping.Result{
						Success: true,
						RTT:     time.Duration(j+1) * time.Millisecond,
					})
				}
			}

			// Capture initial history
			initialHistory := make(map[string]int)
			for i := 0; i < targetCount; i++ {
				status, _ := store.GetTargetStatus(generateTargetName(i))
				initialHistory[generateTargetName(i)] = len(status.History)
			}

			// Perform multiple reloads with same targets
			for reload := 0; reload < reloadCount; reload++ {
				reloadTargets := make([]config.TargetConfig, targetCount)
				for i := 0; i < targetCount; i++ {
					reloadTargets[i] = config.TargetConfig{
						Name:    generateTargetName(i),
						Address: generateAddress(i + reload*100), // Change address
						Group:   "group-" + string(rune('1'+reload)),
					}
				}
				store.UpdateTargets(reloadTargets)
			}

			// Verify history is still preserved after multiple reloads
			for i := 0; i < targetCount; i++ {
				status, ok := store.GetTargetStatus(generateTargetName(i))
				if !ok {
					return false
				}

				// History should be preserved
				if len(status.History) != initialHistory[generateTargetName(i)] {
					return false
				}
			}

			return true
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(5) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(5) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(10) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.TestingRun(t, gopter.ConsoleReporter(false))
}
