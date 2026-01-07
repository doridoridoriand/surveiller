package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/doridoridoriand/surveiller/internal/config"
	"github.com/doridoridoriand/surveiller/internal/state"
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
