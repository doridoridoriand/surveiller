package main

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/doridoridoriand/surveiller/internal/config"
	"github.com/doridoridoriand/surveiller/internal/log"
	"github.com/doridoridoriand/surveiller/internal/ping"
	"github.com/doridoridoriand/surveiller/internal/scheduler"
	"github.com/doridoridoriand/surveiller/internal/state"
	"github.com/doridoridoriand/surveiller/internal/ui"
)

// MockPinger is a mock implementation of ping.Pinger for testing.
type MockPinger struct {
	mu          sync.Mutex
	pingCount   sync.Map // map[string]*int64
	success     bool
	rtt         time.Duration
	pingDelay   time.Duration
	pingResults map[string]ping.Result
}

// NewMockPinger creates a new MockPinger.
func NewMockPinger() *MockPinger {
	return &MockPinger{
		pingCount:   sync.Map{},
		success:     true,
		rtt:         10 * time.Millisecond,
		pingResults: make(map[string]ping.Result),
	}
}

// SetResult sets the result for a specific address.
func (m *MockPinger) SetResult(addr string, result ping.Result) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pingResults[addr] = result
}

// SetDefaultResult sets the default result for all addresses.
func (m *MockPinger) SetDefaultResult(success bool, rtt time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.success = success
	m.rtt = rtt
}

// SetPingDelay sets a delay before returning ping results.
func (m *MockPinger) SetPingDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pingDelay = delay
}

// Ping implements ping.Pinger interface.
func (m *MockPinger) Ping(ctx context.Context, addr string, timeout time.Duration) ping.Result {
	// Increment ping count
	val, _ := m.pingCount.LoadOrStore(addr, new(int64))
	countPtr := val.(*int64)
	atomic.AddInt64(countPtr, 1)

	m.mu.Lock()
	delay := m.pingDelay
	result, ok := m.pingResults[addr]
	m.mu.Unlock()

	if delay > 0 {
		select {
		case <-ctx.Done():
			return ping.Result{Success: false, Error: ctx.Err()}
		case <-time.After(delay):
		}
	}

	if ok {
		return result
	}

	m.mu.Lock()
	success := m.success
	rtt := m.rtt
	m.mu.Unlock()

	return ping.Result{
		Success: success,
		RTT:     rtt,
	}
}

// GetPingCount returns the number of pings for an address.
func (m *MockPinger) GetPingCount(addr string) int64 {
	val, ok := m.pingCount.Load(addr)
	if !ok {
		return 0
	}
	return atomic.LoadInt64(val.(*int64))
}

// WaitForPings waits until the specified address has been pinged at least count times.
func (m *MockPinger) WaitForPings(t *testing.T, addr string, count int64, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for %d pings to %s (got %d)", count, addr, m.GetPingCount(addr))
		}
		if m.GetPingCount(addr) >= count {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// createTempConfig creates a temporary config file for testing.
func createTempConfig(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "surveiller.conf")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
	return configPath
}

// waitForCondition waits until the condition function returns true or timeout.
func waitForCondition(t *testing.T, condition func() bool, timeout time.Duration, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if condition() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for condition: %s", msg)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// 7.1 設定ファイルから監視開始までの統合テスト
func TestE2E_ConfigToMonitoring(t *testing.T) {
	// 1. 一時的な設定ファイルを作成
	configContent := `# surveiller: interval=100ms timeout=50ms max_concurrency=10
target1 192.0.2.1
target2 192.0.2.2
`
	configPath := createTempConfig(t, configContent)

	// 2. 設定ファイルを読み込む
	parser := config.SurveillerParser{}
	cfg, err := parser.LoadConfig(configPath, config.CLIOverrides{})
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(cfg.Targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(cfg.Targets))
	}
	if cfg.Global.Interval != 100*time.Millisecond {
		t.Errorf("expected interval 100ms, got %v", cfg.Global.Interval)
	}
	if cfg.Global.Timeout != 50*time.Millisecond {
		t.Errorf("expected timeout 50ms, got %v", cfg.Global.Timeout)
	}

	// 3. モックpinger作成
	mockPinger := NewMockPinger()
	mockPinger.SetDefaultResult(true, 10*time.Millisecond)

	// 4. スケジューラー起動
	store := state.NewStore(cfg.Targets, cfg.Global.Timeout)
	logger := log.NewLogger(log.LevelInfo)
	sched := scheduler.NewScheduler(cfg.Global, cfg.Targets, mockPinger, store, logger)

	// 5. 監視開始と検証
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		if err := sched.Run(ctx); err != nil && err != context.Canceled && err != context.DeadlineExceeded {
			t.Errorf("scheduler error: %v", err)
		}
	}()

	// pingが実行されることを確認
	mockPinger.WaitForPings(t, "192.0.2.1", 1, 2*time.Second)
	mockPinger.WaitForPings(t, "192.0.2.2", 1, 2*time.Second)

	// 状態が更新されることを確認
	waitForCondition(t, func() bool {
		snapshot := store.GetSnapshot()
		if len(snapshot) != 2 {
			return false
		}
		for _, target := range snapshot {
			if target.Status == state.StatusUnknown {
				return false
			}
		}
		return true
	}, 2*time.Second, "state should be updated")

	snapshot := store.GetSnapshot()
	if len(snapshot) != 2 {
		t.Fatalf("expected 2 targets in snapshot, got %d", len(snapshot))
	}

	// 各ターゲットの状態を確認
	targetMap := make(map[string]state.TargetStatus)
	for _, target := range snapshot {
		targetMap[target.Name] = target
	}

	target1, ok := targetMap["target1"]
	if !ok {
		t.Fatal("target1 not found in snapshot")
	}
	if target1.Address != "192.0.2.1" {
		t.Errorf("target1 address: expected 192.0.2.1, got %s", target1.Address)
	}
	if target1.Status != state.StatusOK && target1.Status != state.StatusWarn {
		t.Errorf("target1 status: expected OK or WARN, got %s", target1.Status)
	}
	if target1.LastRTT <= 0 {
		t.Errorf("target1 LastRTT: expected > 0, got %v", target1.LastRTT)
	}

	target2, ok := targetMap["target2"]
	if !ok {
		t.Fatal("target2 not found in snapshot")
	}
	if target2.Address != "192.0.2.2" {
		t.Errorf("target2 address: expected 192.0.2.2, got %s", target2.Address)
	}
}

// 7.2 TUI とバックエンドの統合テスト
func TestE2E_TUIAndBackend(t *testing.T) {
	// 1. 状態管理ストア作成
	targets := []config.TargetConfig{
		{Name: "target1", Address: "192.0.2.1", Group: "group1"},
		{Name: "target2", Address: "192.0.2.2", Group: "group1"},
	}
	timeout := 100 * time.Millisecond
	store := state.NewStore(targets, timeout)

	// 2. TUI初期化（実際のスクリーンは使用しない）
	cfg := config.GlobalOptions{
		Interval:       1 * time.Second,
		Timeout:        timeout,
		MaxConcurrency: 10,
		UIScale:        10,
		UIDisable:      false,
	}
	reloadCh := make(chan struct{}, 1)
	ui := ui.New(cfg, store, reloadCh)

	// UIが正しく初期化されたことを確認
	if ui == nil {
		t.Fatal("UI should not be nil")
	}

	// 3. 状態更新
	store.UpdateResult("target1", ping.Result{
		Success: true,
		RTT:     10 * time.Millisecond,
	})
	store.UpdateResult("target2", ping.Result{
		Success: true,
		RTT:     20 * time.Millisecond,
	})

	// 4. TUI表示の検証（スナップショットを取得して検証）
	snapshot := store.GetSnapshot()
	if len(snapshot) != 2 {
		t.Fatalf("expected 2 targets in snapshot, got %d", len(snapshot))
	}

	// 状態が正しく更新されていることを確認
	targetMap := make(map[string]state.TargetStatus)
	for _, target := range snapshot {
		targetMap[target.Name] = target
	}

	target1, ok := targetMap["target1"]
	if !ok {
		t.Fatal("target1 not found in snapshot")
	}
	if target1.Status != state.StatusOK {
		t.Errorf("target1 status: expected OK, got %s", target1.Status)
	}
	if target1.LastRTT != 10*time.Millisecond {
		t.Errorf("target1 LastRTT: expected 10ms, got %v", target1.LastRTT)
	}
	if target1.ConsecutiveOK != 1 {
		t.Errorf("target1 ConsecutiveOK: expected 1, got %d", target1.ConsecutiveOK)
	}

	target2, ok := targetMap["target2"]
	if !ok {
		t.Fatal("target2 not found in snapshot")
	}
	if target2.Status != state.StatusOK {
		t.Errorf("target2 status: expected OK, got %s", target2.Status)
	}
	if target2.LastRTT != 20*time.Millisecond {
		t.Errorf("target2 LastRTT: expected 20ms, got %v", target2.LastRTT)
	}

	// 5. キーボード入力のシミュレート（リロードチャネル経由）
	// 'r'キーでリロードをトリガー
	select {
	case reloadCh <- struct{}{}:
		// リロードリクエストが送信された
	default:
		t.Error("failed to send reload request")
	}

	// リロードチャネルが正しく動作することを確認
	select {
	case <-reloadCh:
		// リロードリクエストが受信された
	default:
		t.Error("reload channel should have a message")
	}
}

// 7.3 設定リロードの統合テスト
func TestE2E_ConfigReload(t *testing.T) {
	// 1. 初期設定ファイル作成
	initialConfig := `# surveiller: interval=100ms timeout=50ms max_concurrency=10
target1 192.0.2.1
target2 192.0.2.2
`
	configPath := createTempConfig(t, initialConfig)

	// 2. アプリケーション初期化
	parser := config.SurveillerParser{}
	cfg, err := parser.LoadConfig(configPath, config.CLIOverrides{})
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	mockPinger := NewMockPinger()
	mockPinger.SetDefaultResult(true, 10*time.Millisecond)

	store := state.NewStore(cfg.Targets, cfg.Global.Timeout)
	logger := log.NewLogger(log.LevelInfo)
	sched := scheduler.NewScheduler(cfg.Global, cfg.Targets, mockPinger, store, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 3. 監視開始
	go func() {
		if err := sched.Run(ctx); err != nil && err != context.Canceled && err != context.DeadlineExceeded {
			t.Errorf("scheduler error: %v", err)
		}
	}()

	// 初期ターゲットがpingされることを確認
	mockPinger.WaitForPings(t, "192.0.2.1", 1, 2*time.Second)
	mockPinger.WaitForPings(t, "192.0.2.2", 1, 2*time.Second)

	// 初期状態を確認
	snapshot := store.GetSnapshot()
	if len(snapshot) != 2 {
		t.Fatalf("expected 2 targets initially, got %d", len(snapshot))
	}

	// 4. 設定ファイルを変更
	updatedConfig := `# surveiller: interval=100ms timeout=50ms max_concurrency=10
target1 192.0.2.1
target3 192.0.2.3
target4 192.0.2.4
`
	if err := os.WriteFile(configPath, []byte(updatedConfig), 0644); err != nil {
		t.Fatalf("failed to update config file: %v", err)
	}

	// 5. リロードをトリガー（リロードチャネル経由）
	reloadCh := make(chan struct{}, 1)
	reloadCh <- struct{}{}

	// リロード処理をシミュレート
	newCfg, err := parser.LoadConfig(configPath, config.CLIOverrides{})
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}

	// 6. 設定が更新されることを確認
	if len(newCfg.Targets) != 3 {
		t.Fatalf("expected 3 targets after reload, got %d", len(newCfg.Targets))
	}

	// スケジューラーとストアを更新
	sched.UpdateConfig(newCfg.Global, newCfg.Targets)
	store.UpdateTargets(newCfg.Targets)
	store.UpdateTimeout(newCfg.Global.Timeout)

	// 新しいターゲットが追加されることを確認
	waitForCondition(t, func() bool {
		snapshot := store.GetSnapshot()
		return len(snapshot) == 3
	}, 2*time.Second, "new targets should be added")

	// 新しいターゲットがpingされることを確認
	mockPinger.WaitForPings(t, "192.0.2.3", 1, 2*time.Second)
	mockPinger.WaitForPings(t, "192.0.2.4", 1, 2*time.Second)

	// 最終状態を確認
	finalSnapshot := store.GetSnapshot()
	if len(finalSnapshot) != 3 {
		t.Fatalf("expected 3 targets after reload, got %d", len(finalSnapshot))
	}

	// ターゲット名を確認
	targetMap := make(map[string]state.TargetStatus)
	for _, target := range finalSnapshot {
		targetMap[target.Name] = target
	}

	if _, ok := targetMap["target1"]; !ok {
		t.Error("target1 should still exist")
	}
	if _, ok := targetMap["target3"]; !ok {
		t.Error("target3 should be added")
	}
	if _, ok := targetMap["target4"]; !ok {
		t.Error("target4 should be added")
	}
	if _, ok := targetMap["target2"]; ok {
		t.Error("target2 should be removed")
	}
}
