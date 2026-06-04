package state

import (
	"testing"
	"time"

	"github.com/doridoridoriand/surveiller/internal/config"
	"github.com/doridoridoriand/surveiller/internal/ping"
)

func TestStoreUpdateResultSuccessAndFailure(t *testing.T) {
	store := NewStore([]config.TargetConfig{
		{Name: "example", Address: "192.0.2.1", Group: "group-1"},
	}, 100*time.Millisecond)

	tests := []struct {
		name                  string
		result                ping.Result
		wantStatus            Status
		wantConsecutiveOK     int
		wantConsecutiveFailures int
		wantHistoryLen        int
	}{
		{
			name:                  "first_failure → WARN",
			result:                ping.Result{Success: false, Error: errSentinel{}},
			wantStatus:            StatusWarn,
			wantConsecutiveOK:     0,
			wantConsecutiveFailures: 1,
			wantHistoryLen:        0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store.UpdateResult("example", tt.result)
			status, ok := store.GetTargetStatus("example")
			if !ok {
				t.Fatal("expected target status")
			}
			if status.Status != tt.wantStatus {
				t.Errorf("status = %s, want %s", status.Status, tt.wantStatus)
			}
			if status.ConsecutiveOK != tt.wantConsecutiveOK {
				t.Errorf("ConsecutiveOK = %d, want %d", status.ConsecutiveOK, tt.wantConsecutiveOK)
			}
			if status.ConsecutiveFailures != tt.wantConsecutiveFailures {
				t.Errorf("ConsecutiveFailures = %d, want %d", status.ConsecutiveFailures, tt.wantConsecutiveFailures)
			}
			if len(status.History) != tt.wantHistoryLen {
				t.Errorf("History len = %d, want %d", len(status.History), tt.wantHistoryLen)
			}
		})
	}

	// Sequential test: two more failures push to DOWN, then success resets
	store.UpdateResult("example", ping.Result{Success: false, Error: errSentinel{}})
	store.UpdateResult("example", ping.Result{Success: false, Error: errSentinel{}})
	status, _ := store.GetTargetStatus("example")
	if status.Status != StatusDown {
		t.Errorf("expected DOWN after threshold, got %s", status.Status)
	}

	store.UpdateResult("example", ping.Result{Success: true, RTT: 12 * time.Millisecond})
	status, _ = store.GetTargetStatus("example")
	if status.Status != StatusOK {
		t.Errorf("expected OK after success, got %s", status.Status)
	}
	if status.ConsecutiveFailures != 0 || status.ConsecutiveOK != 1 {
		t.Errorf("unexpected counters: ok=%d fail=%d", status.ConsecutiveOK, status.ConsecutiveFailures)
	}
	if len(status.History) != 1 {
		t.Errorf("expected history length 1, got %d", len(status.History))
	}
}

func TestStoreHistorySize(t *testing.T) {
	store := NewStore([]config.TargetConfig{{Name: "example"}}, 100*time.Millisecond)
	store.historySize = 2

	for _, rtt := range []time.Duration{10, 11, 12} {
		store.UpdateResult("example", ping.Result{Success: true, RTT: rtt * time.Millisecond})
	}

	status, ok := store.GetTargetStatus("example")
	if !ok {
		t.Fatal("expected target status")
	}
	if len(status.History) > 2 {
		t.Errorf("history should not exceed limit: got %d", len(status.History))
	}
}

func TestStoreGroups(t *testing.T) {
	store := NewStore([]config.TargetConfig{
		{Name: "a", Address: "192.0.2.1", Group: "group-1"},
		{Name: "b", Address: "192.0.2.2", Group: "group-1"},
		{Name: "c", Address: "192.0.2.3", Group: "group-2"},
	}, 100*time.Millisecond)

	groups := store.GetGroups()
	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}
}

func TestStoreNoTargets(t *testing.T) {
	store := NewStore(nil, 100*time.Millisecond)
	_, ok := store.GetTargetStatus("missing")
	if ok {
		t.Error("expected target not found")
	}
	groups := store.GetGroups()
	if len(groups) != 0 {
		t.Errorf("expected no groups, got %d", len(groups))
	}
}
