package state

import (
	"testing"
	"time"

	"github.com/doridoridoriand/deadman-go/internal/config"
	"github.com/doridoridoriand/deadman-go/internal/ping"
)

func TestStoreUpdateResultSuccessAndFailure(t *testing.T) {
	store := NewStore([]config.TargetConfig{
		{Name: "example", Address: "192.0.2.1", Group: "group-1"},
	})

	store.UpdateResult("example", ping.Result{Success: false, Error: errSentinel{}})
	status, ok := store.GetTargetStatus("example")
	if !ok {
		t.Fatalf("expected target status")
	}
	if status.Status != StatusWarn {
		t.Fatalf("expected WARN after first failure, got %s", status.Status)
	}
	if status.ConsecutiveNG != 1 {
		t.Fatalf("expected 1 consecutive NG, got %d", status.ConsecutiveNG)
	}

	store.UpdateResult("example", ping.Result{Success: false, Error: errSentinel{}})
	store.UpdateResult("example", ping.Result{Success: false, Error: errSentinel{}})
	status, _ = store.GetTargetStatus("example")
	if status.Status != StatusDown {
		t.Fatalf("expected DOWN after threshold, got %s", status.Status)
	}

	store.UpdateResult("example", ping.Result{Success: true, RTT: 12 * time.Millisecond})
	status, _ = store.GetTargetStatus("example")
	if status.Status != StatusOK {
		t.Fatalf("expected OK after success, got %s", status.Status)
	}
	if status.ConsecutiveNG != 0 || status.ConsecutiveOK != 1 {
		t.Fatalf("unexpected counters: ok=%d ng=%d", status.ConsecutiveOK, status.ConsecutiveNG)
	}
	if len(status.History) != 1 {
		t.Fatalf("expected history length 1, got %d", len(status.History))
	}
}

func TestStoreHistorySize(t *testing.T) {
	store := NewStore([]config.TargetConfig{{Name: "example"}})
	store.historySize = 2

	store.UpdateResult("example", ping.Result{Success: true, RTT: 10 * time.Millisecond})
	store.UpdateResult("example", ping.Result{Success: true, RTT: 11 * time.Millisecond})
	store.UpdateResult("example", ping.Result{Success: true, RTT: 12 * time.Millisecond})

	status, _ := store.GetTargetStatus("example")
	if len(status.History) != 2 {
		t.Fatalf("expected history size 2, got %d", len(status.History))
	}
	if status.History[0].RTT != 11*time.Millisecond || status.History[1].RTT != 12*time.Millisecond {
		t.Fatalf("unexpected history values: %+v", status.History)
	}
}

func TestStoreUpdateTargetsKeepsHistory(t *testing.T) {
	store := NewStore([]config.TargetConfig{{Name: "example", Address: "192.0.2.1"}})
	store.UpdateResult("example", ping.Result{Success: true, RTT: 10 * time.Millisecond})

	store.UpdateTargets([]config.TargetConfig{
		{Name: "example", Address: "192.0.2.2", Group: "group-2"},
		{Name: "new", Address: "192.0.2.3"},
	})

	status, ok := store.GetTargetStatus("example")
	if !ok {
		t.Fatalf("expected existing target status")
	}
	if status.Address != "192.0.2.2" || status.Group != "group-2" {
		t.Fatalf("expected updated address/group, got %s/%s", status.Address, status.Group)
	}
	if len(status.History) != 1 {
		t.Fatalf("expected history preserved, got %d", len(status.History))
	}

	_, ok = store.GetTargetStatus("new")
	if !ok {
		t.Fatalf("expected new target status")
	}
}

type errSentinel struct{}

func (errSentinel) Error() string {
	return "sentinel"
}
