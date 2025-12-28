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
	}, 100*time.Millisecond)

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
	store := NewStore([]config.TargetConfig{{Name: "example"}}, 100*time.Millisecond)
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
	store := NewStore([]config.TargetConfig{{Name: "example", Address: "192.0.2.1"}}, 100*time.Millisecond)
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

func TestStoreUpdateResultRTTThresholds(t *testing.T) {
	timeout := 100 * time.Millisecond
	store := NewStore([]config.TargetConfig{
		{Name: "example", Address: "192.0.2.1"},
	}, timeout)

	// 直近10個のデータポイントで平均を計算するため、同じRTT値を10回送信して
	// Historyを満たしてからテストを実行する
	// OK: timeoutの25%以内（25ms以内）
	for i := 0; i < 10; i++ {
		store.UpdateResult("example", ping.Result{Success: true, RTT: 20 * time.Millisecond})
	}
	status, _ := store.GetTargetStatus("example")
	if status.Status != StatusOK {
		t.Fatalf("expected OK for avg RTT 20ms (within 25%% of 100ms), got %s", status.Status)
	}

	// OK: 境界値 - timeoutの25%ちょうど（25ms）
	for i := 0; i < 10; i++ {
		store.UpdateResult("example", ping.Result{Success: true, RTT: 25 * time.Millisecond})
	}
	status, _ = store.GetTargetStatus("example")
	if status.Status != StatusOK {
		t.Fatalf("expected OK for avg RTT 25ms (exactly 25%% of 100ms), got %s", status.Status)
	}

	// WARN: timeoutの25%超、50%以内（25ms超、50ms以内）
	for i := 0; i < 10; i++ {
		store.UpdateResult("example", ping.Result{Success: true, RTT: 26 * time.Millisecond})
	}
	status, _ = store.GetTargetStatus("example")
	if status.Status != StatusWarn {
		t.Fatalf("expected WARN for avg RTT 26ms (just over 25%% of 100ms), got %s", status.Status)
	}

	for i := 0; i < 10; i++ {
		store.UpdateResult("example", ping.Result{Success: true, RTT: 40 * time.Millisecond})
	}
	status, _ = store.GetTargetStatus("example")
	if status.Status != StatusWarn {
		t.Fatalf("expected WARN for avg RTT 40ms (between 25%% and 50%% of 100ms), got %s", status.Status)
	}

	// WARN: 境界値 - timeoutの50%ちょうど（50ms）
	for i := 0; i < 10; i++ {
		store.UpdateResult("example", ping.Result{Success: true, RTT: 50 * time.Millisecond})
	}
	status, _ = store.GetTargetStatus("example")
	if status.Status != StatusWarn {
		t.Fatalf("expected WARN for avg RTT 50ms (exactly 50%% of 100ms), got %s", status.Status)
	}

	// WARN: timeoutの50%超（50ms超）
	for i := 0; i < 10; i++ {
		store.UpdateResult("example", ping.Result{Success: true, RTT: 80 * time.Millisecond})
	}
	status, _ = store.GetTargetStatus("example")
	if status.Status != StatusWarn {
		t.Fatalf("expected WARN for avg RTT 80ms (over 50%% of 100ms), got %s", status.Status)
	}
}

type errSentinel struct{}

func (errSentinel) Error() string {
	return "sentinel"
}
