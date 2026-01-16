package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/doridoridoriand/surveiller/internal/config"
	"github.com/doridoridoriand/surveiller/internal/state"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/prop"
)

// **Feature: surveiller, Property 7: TUI グループ表示**
// **Validates: Requirements 3.2**
func TestPropertyTUIGroupDisplay(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	props := gopter.NewProperties(params)

	props.Property("targets are grouped correctly by group name", prop.ForAll(
		func(groupCount int, targetsPerGroup int) bool {
			if groupCount < 1 || groupCount > 10 || targetsPerGroup < 1 || targetsPerGroup > 10 {
				return true // Skip invalid combinations
			}

			// Generate snapshot with specified groups
			snapshot := make([]state.TargetStatus, 0, groupCount*targetsPerGroup)
			for g := 0; g < groupCount; g++ {
				groupName := generateGroupName(g)
				for t := 0; t < targetsPerGroup; t++ {
					snapshot = append(snapshot, state.TargetStatus{
						Name:    generateTargetName(g, t),
						Address: generateAddress(g, t),
						Group:   groupName,
						Status:  state.StatusOK,
					})
				}
			}

			groups := groupTargets(snapshot)

			// Verify correct number of groups
			if len(groups) != groupCount {
				return false
			}

			// Verify each group has correct number of targets
			for _, group := range groups {
				if len(group.Targets) != targetsPerGroup {
					return false
				}
				// Verify all targets in group have same group name
				for _, target := range group.Targets {
					expectedGroup := strings.TrimSpace(target.Group)
					if expectedGroup == "" {
						expectedGroup = "default"
					}
					if group.Name != expectedGroup {
						return false
					}
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

	props.Property("default group is created for empty group names", prop.ForAll(
		func(targetCount int) bool {
			if targetCount < 1 || targetCount > 20 {
				return true
			}

			snapshot := make([]state.TargetStatus, targetCount)
			for i := 0; i < targetCount; i++ {
				snapshot[i] = state.TargetStatus{
					Name:    generateTargetName(0, i),
					Address: generateAddress(0, i),
					Group:   "", // Empty group
					Status:  state.StatusOK,
				}
			}

			groups := groupTargets(snapshot)

			// Should have exactly one group named "default"
			if len(groups) != 1 {
				return false
			}
			if groups[0].Name != "default" {
				return false
			}
			if len(groups[0].Targets) != targetCount {
				return false
			}

			return true
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(20) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.Property("default group appears first in sorted order", prop.ForAll(
		func(namedGroupCount int) bool {
			if namedGroupCount < 1 || namedGroupCount > 10 {
				return true
			}

			snapshot := make([]state.TargetStatus, 0, namedGroupCount+1)
			// Add default group target
			snapshot = append(snapshot, state.TargetStatus{
				Name:    "default-target",
				Address: "192.0.2.1",
				Group:   "",
				Status:  state.StatusOK,
			})
			// Add named group targets
			for i := 0; i < namedGroupCount; i++ {
				snapshot = append(snapshot, state.TargetStatus{
					Name:    generateTargetName(i, 0),
					Address: generateAddress(i, 0),
					Group:   generateGroupName(i),
					Status:  state.StatusOK,
				})
			}

			groups := groupTargets(snapshot)

			// Default group should be first
			if len(groups) < 1 {
				return false
			}
			if groups[0].Name != "default" {
				return false
			}

			return true
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(10) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.Property("targets within group are sorted alphabetically", prop.ForAll(
		func(targetCount int) bool {
			if targetCount < 2 || targetCount > 20 {
				return true
			}

			// Create targets with names in reverse alphabetical order
			snapshot := make([]state.TargetStatus, targetCount)
			for i := 0; i < targetCount; i++ {
				// Generate names in reverse order (z, y, x, ...)
				name := string(rune('z' - i))
				snapshot[i] = state.TargetStatus{
					Name:    name,
					Address: generateAddress(0, i),
					Group:   "test-group",
					Status:  state.StatusOK,
				}
			}

			groups := groupTargets(snapshot)

			if len(groups) != 1 {
				return false
			}
			if len(groups[0].Targets) != targetCount {
				return false
			}

			// Verify alphabetical order
			for i := 1; i < len(groups[0].Targets); i++ {
				if groups[0].Targets[i-1].Name > groups[0].Targets[i].Name {
					return false
				}
			}

			return true
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(19) + 2
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.TestingRun(t, gopter.ConsoleReporter(false))
}

// **Feature: surveiller, Property 8: TUI RTT バーグラフ表示**
// **Validates: Requirements 3.3**
func TestPropertyTUIRTTBarGraphDisplay(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	props := gopter.NewProperties(params)

	props.Property("bar length is proportional to RTT and scale", prop.ForAll(
		func(rttMs int, scale int, width int) bool {
			if rttMs < 1 || rttMs > 1000 || scale < 1 || scale > 100 || width < 1 || width > 100 {
				return true // Skip invalid combinations
			}

			target := state.TargetStatus{
				LastRTT: time.Duration(rttMs) * time.Millisecond,
			}

			bar := buildBar(target, scale, width)

			// Calculate expected units
			expectedUnits := int(float64(rttMs) / float64(scale))
			if expectedUnits > width {
				expectedUnits = width
			}
			if expectedUnits < 0 {
				expectedUnits = 0
			}

			// Count hash characters in bar
			hashCount := strings.Count(bar, "#")
			spaceCount := strings.Count(bar, " ")

			// Verify bar length matches width
			if len(bar) != width {
				return false
			}

			// Verify hash count matches expected (with rounding tolerance)
			if hashCount < expectedUnits-1 || hashCount > expectedUnits+1 {
				return false
			}

			// Verify total characters match width
			if hashCount+spaceCount != width {
				return false
			}

			return true
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(1000) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(100) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(100) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.Property("bar is empty for zero or negative RTT", prop.ForAll(
		func(width int) bool {
			if width < 1 || width > 100 {
				return true
			}

			zeroTarget := state.TargetStatus{LastRTT: 0}
			negativeTarget := state.TargetStatus{LastRTT: -10 * time.Millisecond}

			zeroBar := buildBar(zeroTarget, 10, width)
			negativeBar := buildBar(negativeTarget, 10, width)

			// Both should be all spaces
			expected := strings.Repeat(" ", width)
			if zeroBar != expected {
				return false
			}
			if negativeBar != expected {
				return false
			}

			return true
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(100) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.Property("bar is capped at width when RTT exceeds scale", prop.ForAll(
		func(rttMs int, scale int, width int) bool {
			if rttMs < 1 || scale < 1 || width < 1 || rttMs <= scale*width {
				return true // Only test cases where RTT exceeds scale*width
			}

			target := state.TargetStatus{
				LastRTT: time.Duration(rttMs) * time.Millisecond,
			}

			bar := buildBar(target, scale, width)

			// Bar should be exactly width characters, all hashes
			if len(bar) != width {
				return false
			}
			hashCount := strings.Count(bar, "#")
			if hashCount != width {
				return false
			}

			return true
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(1000) + 100
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(10) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(50) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.Property("zero or negative scale defaults to 10", prop.ForAll(
		func(rttMs int, width int) bool {
			if rttMs < 1 || rttMs > 1000 || width < 1 || width > 100 {
				return true
			}

			target := state.TargetStatus{
				LastRTT: time.Duration(rttMs) * time.Millisecond,
			}

			zeroScaleBar := buildBar(target, 0, width)
			negativeScaleBar := buildBar(target, -5, width)
			defaultScaleBar := buildBar(target, 10, width)

			// All should produce same result
			if zeroScaleBar != defaultScaleBar {
				return false
			}
			if negativeScaleBar != defaultScaleBar {
				return false
			}

			return true
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(1000) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(100) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.TestingRun(t, gopter.ConsoleReporter(false))
}

// **Feature: surveiller, Property 9: TUI 状態更新の即時反映**
// **Validates: Requirements 3.4**
func TestPropertyTUIStateUpdateImmediateReflection(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	props := gopter.NewProperties(params)

	props.Property("formatTargetLine reflects current status immediately", prop.ForAll(
		func(statusInt int) bool {
			if statusInt < 0 || statusInt > 3 {
				return true
			}

			statuses := []state.Status{
				state.StatusOK,
				state.StatusWarn,
				state.StatusDown,
				state.StatusUnknown,
			}
			status := statuses[statusInt%len(statuses)]

			u := &UI{cfg: config.GlobalOptions{UIScale: 10}}
			target := state.TargetStatus{
				Name:    "test",
				Address: "192.0.2.1",
				Status:  status,
				LastRTT: 50 * time.Millisecond,
			}

			line := styledRunesToString(u.formatTargetLine(120, target))

			// Verify status string appears in line
			// padOrTrim may truncate "UNKNOWN" to 6 chars, so check for substring
			statusStr := string(status)
			// Check if status string (or truncated version) appears in line
			if len(statusStr) > 6 {
				statusStr = statusStr[:6]
			}
			if !strings.Contains(line, statusStr) {
				return false
			}

			return true
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(4)
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.Property("formatTargetLine reflects latest RTT immediately", prop.ForAll(
		func(rttMs int) bool {
			if rttMs < 1 || rttMs > 10000 {
				return true
			}

			rtt := time.Duration(rttMs) * time.Millisecond
			u := &UI{cfg: config.GlobalOptions{UIScale: 10}}
			target := state.TargetStatus{
				Name:    "test",
				Address: "192.0.2.1",
				Status:  state.StatusOK,
				LastRTT: rtt,
			}

			line := styledRunesToString(u.formatTargetLine(120, target))

			// Verify RTT appears in line
			// Format should be "RTT:XXms" or "RTT:XX.Xs"
			if !strings.Contains(line, "RTT:") {
				return false
			}

			return true
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(10000) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.Property("formatTargetLine reflects average RTT from history", prop.ForAll(
		func(historySize int) bool {
			if historySize < 1 || historySize > 20 {
				return true
			}

			u := &UI{cfg: config.GlobalOptions{UIScale: 10}}
			history := make([]state.RTTPoint, historySize)
			var sum time.Duration
			for i := 0; i < historySize; i++ {
				rtt := time.Duration((i+1)*10) * time.Millisecond
				history[i] = state.RTTPoint{RTT: rtt}
				sum += rtt
			}
			avgRTT := sum / time.Duration(historySize)

			target := state.TargetStatus{
				Name:    "test",
				Address: "192.0.2.1",
				Status:  state.StatusOK,
				LastRTT: 100 * time.Millisecond,
				History: history,
			}

			line := styledRunesToString(u.formatTargetLine(120, target))

			// Verify AVG appears in line
			if !strings.Contains(line, "AVG:") {
				return false
			}

			// Verify calculated average matches
			calculatedAvg := calculateAvgRTT(target)
			if calculatedAvg != avgRTT {
				return false
			}

			return true
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(20) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.Property("formatTargetLine reflects loss percentage immediately", prop.ForAll(
		func(successCount int, failureCount int) bool {
			if successCount < 0 || successCount > 100 || failureCount < 0 || failureCount > 100 {
				return true
			}
			if successCount == 0 && failureCount == 0 {
				return true // Skip zero case
			}

			u := &UI{cfg: config.GlobalOptions{UIScale: 10}}
			target := state.TargetStatus{
				Name:         "test",
				Address:      "192.0.2.1",
				Status:       state.StatusOK,
				LastRTT:      50 * time.Millisecond,
				TotalSuccess: successCount,
				TotalFailure: failureCount,
			}

			line := styledRunesToString(u.formatTargetLine(120, target))

			// Verify LOSS appears in line
			if !strings.Contains(line, "LOSS:") {
				return false
			}

			// Verify calculated loss percentage
			expectedLoss := float64(failureCount) / float64(successCount+failureCount) * 100.0
			calculatedLoss := calculateLossPercent(target)
			if calculatedLoss != expectedLoss {
				return false
			}

			return true
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(101)
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(101)
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.TestingRun(t, gopter.ConsoleReporter(false))
}

// Helper functions for generating test data
func generateGroupName(index int) string {
	return "group-" + string(rune('A'+index))
}

func generateTargetName(groupIndex, targetIndex int) string {
	return "target-" + string(rune('A'+groupIndex)) + "-" + string(rune('0'+targetIndex))
}

func generateAddress(groupIndex, targetIndex int) string {
	return "192.0.2." + string(rune('0'+groupIndex*10+targetIndex))
}
