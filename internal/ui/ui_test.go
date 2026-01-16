package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/doridoridoriand/surveiller/internal/config"
	"github.com/doridoridoriand/surveiller/internal/state"
	"github.com/gdamore/tcell/v2"
)

func styledRunesToString(parts []styledRune) string {
	var b strings.Builder
	for _, part := range parts {
		b.WriteString(string(part.r))
	}
	return b.String()
}

func TestFormatTargetLineShowsLatestRTTAndAverage(t *testing.T) {
	u := &UI{cfg: config.GlobalOptions{UIScale: 10}}
	target := state.TargetStatus{
		Name:    "example",
		Address: "192.0.2.10",
		Status:  state.StatusOK,
		LastRTT: 30 * time.Millisecond,
		History: []state.RTTPoint{
			{RTT: 10 * time.Millisecond},
			{RTT: 20 * time.Millisecond},
			{RTT: 30 * time.Millisecond},
		},
		TotalSuccess: 3,
	}

	line := styledRunesToString(u.formatTargetLine(120, target))
	rttIndex := strings.Index(line, "RTT:30ms")
	avgIndex := strings.Index(line, "AVG:20ms")
	if rttIndex == -1 {
		t.Fatalf("expected latest RTT to be displayed, got %q", line)
	}
	if avgIndex == -1 {
		t.Fatalf("expected average RTT to be displayed, got %q", line)
	}
	if rttIndex > avgIndex {
		t.Fatalf("expected RTT before AVG, got %q", line)
	}
}

// 5.1 TUI レンダリングの単体テスト

func TestFormatTargetLine_DisplaysTargetInfo(t *testing.T) {
	u := &UI{cfg: config.GlobalOptions{UIScale: 10}}
	target := state.TargetStatus{
		Name:    "testhost",
		Address: "192.168.1.1",
		Status:  state.StatusOK,
		LastRTT: 15 * time.Millisecond,
	}

	line := styledRunesToString(u.formatTargetLine(120, target))

	if !strings.Contains(line, "testhost") {
		t.Errorf("expected target name 'testhost' in line, got %q", line)
	}
	if !strings.Contains(line, "192.168.1.1") {
		t.Errorf("expected address '192.168.1.1' in line, got %q", line)
	}
	if !strings.Contains(line, "OK") {
		t.Errorf("expected status 'OK' in line, got %q", line)
	}
}

func TestFormatRTT_VariousDurations(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"zero", 0, "-"},
		{"negative", -1 * time.Millisecond, "-"},
		{"microseconds", 500 * time.Microsecond, "500us"},
		{"milliseconds", 25 * time.Millisecond, "25ms"},
		{"seconds", 2500 * time.Millisecond, "2.5s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRTT(tt.duration)
			if result != tt.expected {
				t.Errorf("formatRTT(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestStatusStyle_ReturnsCorrectColors(t *testing.T) {
	tests := []struct {
		status        state.Status
		expectedColor tcell.Color
	}{
		{state.StatusOK, tcell.ColorGreen},
		{state.StatusWarn, tcell.ColorYellow},
		{state.StatusDown, tcell.ColorRed},
		{state.StatusUnknown, tcell.ColorGray},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			style := statusStyle(tt.status)
			fg, _, _ := style.Decompose()
			if fg != tt.expectedColor {
				t.Errorf("statusStyle(%v) foreground = %v, want %v", tt.status, fg, tt.expectedColor)
			}
		})
	}
}

func TestFormatConfigInfo_DisplaysAllSettings(t *testing.T) {
	cfg := config.GlobalOptions{
		Interval:       2 * time.Second,
		Timeout:        1 * time.Second,
		MaxConcurrency: 50,
		UIScale:        15,
	}

	result := formatConfigInfo(cfg)

	if !strings.Contains(result, "interval=2.0s") {
		t.Errorf("expected interval in config info, got %q", result)
	}
	if !strings.Contains(result, "timeout=1.0s") {
		t.Errorf("expected timeout in config info, got %q", result)
	}
	if !strings.Contains(result, "max_concurrency=50") {
		t.Errorf("expected max_concurrency in config info, got %q", result)
	}
	if !strings.Contains(result, "ui.scale=15") {
		t.Errorf("expected ui.scale in config info, got %q", result)
	}
}

func TestCalculateAvgRTT_WithHistory(t *testing.T) {
	target := state.TargetStatus{
		LastRTT: 50 * time.Millisecond,
		History: []state.RTTPoint{
			{RTT: 10 * time.Millisecond},
			{RTT: 20 * time.Millisecond},
			{RTT: 30 * time.Millisecond},
		},
	}

	avg := calculateAvgRTT(target)
	expected := 20 * time.Millisecond
	if avg != expected {
		t.Errorf("calculateAvgRTT() = %v, want %v", avg, expected)
	}
}

func TestCalculateAvgRTT_NoHistory(t *testing.T) {
	target := state.TargetStatus{
		LastRTT: 50 * time.Millisecond,
		History: []state.RTTPoint{},
	}

	avg := calculateAvgRTT(target)
	if avg != target.LastRTT {
		t.Errorf("calculateAvgRTT() = %v, want %v", avg, target.LastRTT)
	}
}

func TestCalculateLossPercent(t *testing.T) {
	tests := []struct {
		name         string
		totalSuccess int
		totalFailure int
		expected     float64
	}{
		{"no pings", 0, 0, 0.0},
		{"all success", 10, 0, 0.0},
		{"all failure", 0, 10, 100.0},
		{"50% loss", 5, 5, 50.0},
		{"25% loss", 75, 25, 25.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := state.TargetStatus{
				TotalSuccess: tt.totalSuccess,
				TotalFailure: tt.totalFailure,
			}
			result := calculateLossPercent(target)
			if result != tt.expected {
				t.Errorf("calculateLossPercent() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPadOrTrim(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		width    int
		expected string
	}{
		{"exact fit", "hello", 5, "hello"},
		{"needs padding", "hi", 5, "hi   "},
		{"needs trimming", "hello world", 5, "hello"},
		{"zero width", "test", 0, ""},
		{"negative width", "test", -1, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := padOrTrim(tt.value, tt.width)
			if result != tt.expected {
				t.Errorf("padOrTrim(%q, %d) = %q, want %q", tt.value, tt.width, result, tt.expected)
			}
		})
	}
}

// 5.2 グループ表示機能の単体テスト

func TestGroupTargets_EmptySnapshot(t *testing.T) {
	result := groupTargets([]state.TargetStatus{})
	if result != nil {
		t.Errorf("groupTargets([]) = %v, want nil", result)
	}
}

func TestGroupTargets_SingleGroup(t *testing.T) {
	snapshot := []state.TargetStatus{
		{Name: "host1", Group: "web"},
		{Name: "host2", Group: "web"},
	}

	groups := groupTargets(snapshot)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Name != "web" {
		t.Errorf("expected group name 'web', got %q", groups[0].Name)
	}
	if len(groups[0].Targets) != 2 {
		t.Errorf("expected 2 targets in group, got %d", len(groups[0].Targets))
	}
}

func TestGroupTargets_MultipleGroups(t *testing.T) {
	snapshot := []state.TargetStatus{
		{Name: "web1", Group: "web"},
		{Name: "db1", Group: "database"},
		{Name: "web2", Group: "web"},
	}

	groups := groupTargets(snapshot)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}

	// Verify groups are sorted (database before web alphabetically)
	groupNames := make([]string, len(groups))
	for i, g := range groups {
		groupNames[i] = g.Name
	}

	// Check that we have both groups
	hasWeb := false
	hasDB := false
	for _, name := range groupNames {
		if name == "web" {
			hasWeb = true
		}
		if name == "database" {
			hasDB = true
		}
	}
	if !hasWeb || !hasDB {
		t.Errorf("expected both 'web' and 'database' groups, got %v", groupNames)
	}
}

func TestGroupTargets_DefaultGroup(t *testing.T) {
	snapshot := []state.TargetStatus{
		{Name: "host1", Group: ""},
		{Name: "host2", Group: "  "},
	}

	groups := groupTargets(snapshot)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Name != "default" {
		t.Errorf("expected group name 'default', got %q", groups[0].Name)
	}
}

func TestGroupTargets_DefaultGroupFirst(t *testing.T) {
	snapshot := []state.TargetStatus{
		{Name: "host1", Group: "web"},
		{Name: "host2", Group: ""},
	}

	groups := groupTargets(snapshot)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	// Default group should come first
	if groups[0].Name != "default" {
		t.Errorf("expected 'default' group first, got %q", groups[0].Name)
	}
}

func TestGroupTargets_TargetsSortedWithinGroup(t *testing.T) {
	snapshot := []state.TargetStatus{
		{Name: "zebra", Group: "web"},
		{Name: "alpha", Group: "web"},
		{Name: "beta", Group: "web"},
	}

	groups := groupTargets(snapshot)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}

	targets := groups[0].Targets
	if len(targets) != 3 {
		t.Fatalf("expected 3 targets, got %d", len(targets))
	}

	// Verify alphabetical sorting
	if targets[0].Name != "alpha" {
		t.Errorf("expected first target 'alpha', got %q", targets[0].Name)
	}
	if targets[1].Name != "beta" {
		t.Errorf("expected second target 'beta', got %q", targets[1].Name)
	}
	if targets[2].Name != "zebra" {
		t.Errorf("expected third target 'zebra', got %q", targets[2].Name)
	}
}

// 5.4 RTT バーグラフ表示の単体テスト

func TestBuildBar_ZeroWidth(t *testing.T) {
	target := state.TargetStatus{LastRTT: 50 * time.Millisecond}
	bar := buildBar(target, 10, 0)
	if bar != "" {
		t.Errorf("buildBar with width 0 should return empty string, got %q", bar)
	}
}

func TestBuildBar_NegativeWidth(t *testing.T) {
	target := state.TargetStatus{LastRTT: 50 * time.Millisecond}
	bar := buildBar(target, 10, -5)
	if bar != "" {
		t.Errorf("buildBar with negative width should return empty string, got %q", bar)
	}
}

func TestBuildBar_ZeroRTT(t *testing.T) {
	target := state.TargetStatus{LastRTT: 0}
	bar := buildBar(target, 10, 20)
	expected := strings.Repeat(" ", 20)
	if bar != expected {
		t.Errorf("buildBar with zero RTT should return all spaces, got %q", bar)
	}
}

func TestBuildBar_NegativeRTT(t *testing.T) {
	target := state.TargetStatus{LastRTT: -10 * time.Millisecond}
	bar := buildBar(target, 10, 20)
	expected := strings.Repeat(" ", 20)
	if bar != expected {
		t.Errorf("buildBar with negative RTT should return all spaces, got %q", bar)
	}
}

func TestBuildBar_ScaleCalculation(t *testing.T) {
	tests := []struct {
		name          string
		rtt           time.Duration
		scale         int
		width         int
		expectedUnits int
	}{
		{"10ms with scale 10", 10 * time.Millisecond, 10, 20, 1},
		{"50ms with scale 10", 50 * time.Millisecond, 10, 20, 5},
		{"100ms with scale 10", 100 * time.Millisecond, 10, 20, 10},
		{"25ms with scale 5", 25 * time.Millisecond, 5, 20, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := state.TargetStatus{LastRTT: tt.rtt}
			bar := buildBar(target, tt.scale, tt.width)

			hashCount := strings.Count(bar, "#")
			if hashCount != tt.expectedUnits {
				t.Errorf("buildBar() hash count = %d, want %d (bar: %q)", hashCount, tt.expectedUnits, bar)
			}

			totalLen := len(bar)
			if totalLen != tt.width {
				t.Errorf("buildBar() length = %d, want %d", totalLen, tt.width)
			}
		})
	}
}

func TestBuildBar_RTTExceedsWidth(t *testing.T) {
	target := state.TargetStatus{LastRTT: 500 * time.Millisecond}
	bar := buildBar(target, 10, 20)

	// With 500ms and scale 10, we'd want 50 units, but width is only 20
	// So it should be capped at 20 hashes
	hashCount := strings.Count(bar, "#")
	if hashCount != 20 {
		t.Errorf("buildBar() should cap at width, got %d hashes, want 20", hashCount)
	}

	if len(bar) != 20 {
		t.Errorf("buildBar() length = %d, want 20", len(bar))
	}
}

func TestBuildBar_ZeroScale(t *testing.T) {
	target := state.TargetStatus{LastRTT: 50 * time.Millisecond}
	bar := buildBar(target, 0, 20)

	// Zero scale should default to 10
	hashCount := strings.Count(bar, "#")
	if hashCount != 5 {
		t.Errorf("buildBar() with zero scale should use default 10, got %d hashes, want 5", hashCount)
	}
}

func TestBuildBar_NegativeScale(t *testing.T) {
	target := state.TargetStatus{LastRTT: 50 * time.Millisecond}
	bar := buildBar(target, -5, 20)

	// Negative scale should default to 10
	hashCount := strings.Count(bar, "#")
	if hashCount != 5 {
		t.Errorf("buildBar() with negative scale should use default 10, got %d hashes, want 5", hashCount)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"microseconds", 500 * time.Microsecond, "500us"},
		{"milliseconds", 250 * time.Millisecond, "250ms"},
		{"seconds", 3500 * time.Millisecond, "3.5s"},
		{"minutes", 90 * time.Second, "1.5m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}
