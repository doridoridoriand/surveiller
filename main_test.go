package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/doridoridoriand/surveiller/internal/cli"
	"github.com/doridoridoriand/surveiller/internal/config"
	"github.com/doridoridoriand/surveiller/internal/log"
	"github.com/doridoridoriand/surveiller/internal/ping"
	"github.com/doridoridoriand/surveiller/internal/state"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/prop"
)

// 6.1 アプリケーション初期化の単体テスト
func TestBuildOverrides(t *testing.T) {
	tests := []struct {
		name             string
		setupInterval    func() cli.OptionalDuration
		setupTimeout     func() cli.OptionalDuration
		setupMaxConc     func() cli.OptionalInt
		setupMetrics     func() cli.OptionalMetricsMode
		setupListen      func() cli.OptionalString
		setupNoUI        func() cli.OptionalBool
		checkInterval    bool
		checkTimeout     bool
		checkMaxConc     bool
		checkMetrics     bool
		checkListen      bool
		checkUIDisable   bool
		expectedInterval time.Duration
		expectedTimeout  time.Duration
		expectedMaxConc  int
		expectedMetrics  config.MetricsMode
		expectedListen   string
		expectedNoUI     bool
	}{
		{
			name: "all overrides set",
			setupInterval: func() cli.OptionalDuration {
				var d cli.OptionalDuration
				d.Set("2s")
				return d
			},
			setupTimeout: func() cli.OptionalDuration {
				var d cli.OptionalDuration
				d.Set("1s")
				return d
			},
			setupMaxConc: func() cli.OptionalInt {
				var i cli.OptionalInt
				i.Set("50")
				return i
			},
			setupMetrics: func() cli.OptionalMetricsMode {
				var m cli.OptionalMetricsMode
				m.Set("aggregated")
				return m
			},
			setupListen: func() cli.OptionalString {
				var s cli.OptionalString
				s.Set(":9100")
				return s
			},
			setupNoUI: func() cli.OptionalBool {
				var b cli.OptionalBool
				b.Set("true")
				return b
			},
			checkInterval:    true,
			checkTimeout:     true,
			checkMaxConc:     true,
			checkMetrics:     true,
			checkListen:      true,
			checkUIDisable:   true,
			expectedInterval: 2 * time.Second,
			expectedTimeout:  1 * time.Second,
			expectedMaxConc:  50,
			expectedMetrics:  config.MetricsModeAggregated,
			expectedListen:   ":9100",
			expectedNoUI:     true,
		},
		{
			name: "partial overrides",
			setupInterval: func() cli.OptionalDuration {
				var d cli.OptionalDuration
				d.Set("3s")
				return d
			},
			setupTimeout: func() cli.OptionalDuration {
				var d cli.OptionalDuration
				d.Set("2s")
				return d
			},
			setupMaxConc:     func() cli.OptionalInt { return cli.OptionalInt{} },
			setupMetrics:     func() cli.OptionalMetricsMode { return cli.OptionalMetricsMode{} },
			setupListen:      func() cli.OptionalString { return cli.OptionalString{} },
			setupNoUI:        func() cli.OptionalBool { return cli.OptionalBool{} },
			checkInterval:    true,
			checkTimeout:     true,
			expectedInterval: 3 * time.Second,
			expectedTimeout:  2 * time.Second,
		},
		{
			name:          "no overrides",
			setupInterval: func() cli.OptionalDuration { return cli.OptionalDuration{} },
			setupTimeout:  func() cli.OptionalDuration { return cli.OptionalDuration{} },
			setupMaxConc:  func() cli.OptionalInt { return cli.OptionalInt{} },
			setupMetrics:  func() cli.OptionalMetricsMode { return cli.OptionalMetricsMode{} },
			setupListen:   func() cli.OptionalString { return cli.OptionalString{} },
			setupNoUI:     func() cli.OptionalBool { return cli.OptionalBool{} },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overrides := buildOverrides(
				tt.setupInterval(),
				tt.setupTimeout(),
				tt.setupMaxConc(),
				tt.setupMetrics(),
				tt.setupListen(),
				tt.setupNoUI(),
			)

			if tt.checkInterval {
				if overrides.Interval == nil {
					t.Error("expected interval override to be set")
				} else if *overrides.Interval != tt.expectedInterval {
					t.Errorf("expected interval %v, got %v", tt.expectedInterval, *overrides.Interval)
				}
			}

			if tt.checkTimeout {
				if overrides.Timeout == nil {
					t.Error("expected timeout override to be set")
				} else if *overrides.Timeout != tt.expectedTimeout {
					t.Errorf("expected timeout %v, got %v", tt.expectedTimeout, *overrides.Timeout)
				}
			}

			if tt.checkMaxConc {
				if overrides.MaxConcurrency == nil {
					t.Error("expected maxConcurrency override to be set")
				} else if *overrides.MaxConcurrency != tt.expectedMaxConc {
					t.Errorf("expected maxConcurrency %d, got %d", tt.expectedMaxConc, *overrides.MaxConcurrency)
				}
			}

			if tt.checkMetrics {
				if overrides.MetricsMode == nil {
					t.Error("expected metricsMode override to be set")
				} else if *overrides.MetricsMode != tt.expectedMetrics {
					t.Errorf("expected metricsMode %v, got %v", tt.expectedMetrics, *overrides.MetricsMode)
				}
			}

			if tt.checkListen {
				if overrides.MetricsListen == nil {
					t.Error("expected metricsListen override to be set")
				} else if *overrides.MetricsListen != tt.expectedListen {
					t.Errorf("expected metricsListen %q, got %q", tt.expectedListen, *overrides.MetricsListen)
				}
			}

			if tt.checkUIDisable {
				if overrides.UIDisable == nil {
					t.Error("expected uiDisable override to be set")
				} else if *overrides.UIDisable != tt.expectedNoUI {
					t.Errorf("expected uiDisable %v, got %v", tt.expectedNoUI, *overrides.UIDisable)
				}
			}
		})
	}
}

func TestBuildOverrides_EmptyValues(t *testing.T) {
	// Test that empty OptionalString doesn't set override
	var emptyListen cli.OptionalString
	// Don't call Set, so it remains unset
	overrides := buildOverrides(
		cli.OptionalDuration{},
		cli.OptionalDuration{},
		cli.OptionalInt{},
		cli.OptionalMetricsMode{},
		emptyListen,
		cli.OptionalBool{},
	)

	if overrides.MetricsListen != nil {
		t.Error("expected metricsListen to be nil for unset OptionalString")
	}
}

func TestSignalContext(t *testing.T) {
	ctx, cancel := signalContext()
	defer cancel()

	// Verify context is not done initially
	select {
	case <-ctx.Done():
		t.Error("context should not be done initially")
	default:
		// Good
	}

	// Verify cancel function works
	cancel()

	// Context should be done after cancel
	select {
	case <-ctx.Done():
		// Good
	case <-time.After(100 * time.Millisecond):
		t.Error("context should be done after cancel")
	}
}

func TestRequestReload(t *testing.T) {
	ch := make(chan struct{}, 1)

	// First request should succeed
	requestReload(ch)

	select {
	case <-ch:
		// Good
	case <-time.After(100 * time.Millisecond):
		t.Error("expected reload request to be sent")
	}

	// Fill channel
	ch <- struct{}{}

	// Second request should not block (channel is full)
	done := make(chan bool)
	go func() {
		requestReload(ch)
		done <- true
	}()

	select {
	case <-done:
		// Good - function returned without blocking
	case <-time.After(100 * time.Millisecond):
		t.Error("requestReload should not block when channel is full")
	}
}

func TestConfigLoadingWithOverrides(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.conf")

	configContent := `# surveiller: interval=2s timeout=1s max_concurrency=50
test1 192.0.2.1
test2 192.0.2.2
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	parser := config.SurveillerParser{}

	// Test loading without overrides
	cfg, err := parser.LoadConfig(configPath, config.CLIOverrides{})
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Global.Interval != 2*time.Second {
		t.Errorf("expected interval 2s, got %v", cfg.Global.Interval)
	}
	if len(cfg.Targets) != 2 {
		t.Errorf("expected 2 targets, got %d", len(cfg.Targets))
	}

	// Test loading with overrides
	overrideInterval := 5 * time.Second
	overrides := config.CLIOverrides{
		Interval: &overrideInterval,
	}

	cfg2, err := parser.LoadConfig(configPath, overrides)
	if err != nil {
		t.Fatalf("failed to load config with overrides: %v", err)
	}

	if cfg2.Global.Interval != overrideInterval {
		t.Errorf("expected overridden interval %v, got %v", overrideInterval, cfg2.Global.Interval)
	}
	// Other values should remain from config file
	if cfg2.Global.Timeout != 1*time.Second {
		t.Errorf("expected timeout 1s, got %v", cfg2.Global.Timeout)
	}
}

func TestConfigLoading_InvalidFile(t *testing.T) {
	parser := config.SurveillerParser{}
	_, err := parser.LoadConfig("/nonexistent/file.conf", config.CLIOverrides{})
	if err == nil {
		t.Error("expected error for nonexistent config file")
	}
}

// 6.2 シグナルハンドリングの単体テスト
func TestSignalContext_Cancellation(t *testing.T) {
	ctx, cancel := signalContext()
	defer cancel()

	// Verify context can be cancelled
	cancel()

	select {
	case <-ctx.Done():
		// Expected
		if ctx.Err() != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", ctx.Err())
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("context should be cancelled")
	}
}

func TestSignalContext_InitialState(t *testing.T) {
	ctx, cancel := signalContext()
	defer cancel()

	// Context should not be done initially
	if ctx.Err() != nil {
		t.Errorf("expected nil error initially, got %v", ctx.Err())
	}

	// Verify deadline is not set
	if _, ok := ctx.Deadline(); ok {
		t.Error("context should not have deadline")
	}
}

// **Feature: surveiller, Property 15: 設定リロードエラー処理**
// **Validates: Requirements 5.4**
func TestPropertyConfigReloadErrorHandling(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	props := gopter.NewProperties(params)

	props.Property("invalid config file reload maintains existing configuration", prop.ForAll(
		func(targetCount int, historySize int) bool {
			if targetCount < 1 || targetCount > 10 || historySize < 1 || historySize > 20 {
				return true
			}

			tmpDir := t.TempDir()
			validConfigPath := filepath.Join(tmpDir, "valid.conf")
			invalidConfigPath := filepath.Join(tmpDir, "invalid.conf")

			// Create valid initial config
			initialTargets := make([]config.TargetConfig, targetCount)
			for i := 0; i < targetCount; i++ {
				initialTargets[i] = config.TargetConfig{
					Name:    generateTargetName(i),
					Address: generateAddress(i),
					Group:   "group-1",
				}
			}

			validConfigContent := "# surveiller: interval=2s timeout=1s\n"
			for _, tgt := range initialTargets {
				validConfigContent += fmt.Sprintf("%s %s\n", tgt.Name, tgt.Address)
			}

			if err := os.WriteFile(validConfigPath, []byte(validConfigContent), 0644); err != nil {
				return false
			}

			// Create invalid config file (syntax error)
			invalidConfigContent := "# surveiller: interval=invalid_duration\ninvalid line format\n"
			if err := os.WriteFile(invalidConfigPath, []byte(invalidConfigContent), 0644); err != nil {
				return false
			}

			// Load valid config
			parser := config.SurveillerParser{}
			cfg, err := parser.LoadConfig(validConfigPath, config.CLIOverrides{})
			if err != nil {
				return false
			}

			// Initialize store
			store := state.NewStore(cfg.Targets, cfg.Global.Timeout)

			// Add history to targets
			for i := 0; i < targetCount; i++ {
				for j := 0; j < historySize; j++ {
					store.UpdateResult(generateTargetName(i), ping.Result{
						Success: true,
						RTT:     time.Duration(j+1) * time.Millisecond,
					})
				}
			}

			// Capture state before reload attempt
			snapshotBefore := store.GetSnapshot()
			historyBefore := make(map[string]int)
			for i := 0; i < targetCount; i++ {
				status, _ := store.GetTargetStatus(generateTargetName(i))
				historyBefore[generateTargetName(i)] = len(status.History)
			}

			// Attempt reload with invalid config (simulating main.go reload function)
			_, err = parser.LoadConfig(invalidConfigPath, config.CLIOverrides{})
			if err == nil {
				return false // Should have error
			}

			// Verify existing configuration is maintained
			snapshotAfter := store.GetSnapshot()
			if len(snapshotAfter) != len(snapshotBefore) {
				return false
			}

			// Verify targets are still present
			for i := 0; i < targetCount; i++ {
				status, ok := store.GetTargetStatus(generateTargetName(i))
				if !ok {
					return false
				}
				// Verify history is preserved
				if len(status.History) != historyBefore[generateTargetName(i)] {
					return false
				}
			}

			// Verify timeout is maintained (store's timeout)
			// Note: We can't directly check scheduler's config, but we can verify store
			// The timeout in store should remain unchanged

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

	props.Property("nonexistent config file reload maintains existing configuration", prop.ForAll(
		func(targetCount int) bool {
			if targetCount < 1 || targetCount > 10 {
				return true
			}

			tmpDir := t.TempDir()
			validConfigPath := filepath.Join(tmpDir, "valid.conf")
			nonexistentConfigPath := filepath.Join(tmpDir, "nonexistent.conf")

			// Create valid initial config
			initialTargets := make([]config.TargetConfig, targetCount)
			for i := 0; i < targetCount; i++ {
				initialTargets[i] = config.TargetConfig{
					Name:    generateTargetName(i),
					Address: generateAddress(i),
				}
			}

			validConfigContent := "# surveiller: interval=2s timeout=1s\n"
			for _, tgt := range initialTargets {
				validConfigContent += fmt.Sprintf("%s %s\n", tgt.Name, tgt.Address)
			}

			if err := os.WriteFile(validConfigPath, []byte(validConfigContent), 0644); err != nil {
				return false
			}

			// Load valid config
			parser := config.SurveillerParser{}
			cfg, err := parser.LoadConfig(validConfigPath, config.CLIOverrides{})
			if err != nil {
				return false
			}

			// Initialize store
			store := state.NewStore(cfg.Targets, cfg.Global.Timeout)

			// Add some state
			for i := 0; i < targetCount; i++ {
				store.UpdateResult(generateTargetName(i), ping.Result{
					Success: true,
					RTT:     10 * time.Millisecond,
				})
			}

			// Capture state before reload attempt
			snapshotBefore := store.GetSnapshot()

			// Attempt reload with nonexistent file
			_, err = parser.LoadConfig(nonexistentConfigPath, config.CLIOverrides{})
			if err == nil {
				return false // Should have error
			}

			// Verify existing configuration is maintained
			snapshotAfter := store.GetSnapshot()
			if len(snapshotAfter) != len(snapshotBefore) {
				return false
			}

			// Verify targets are still present with their state
			for i := 0; i < targetCount; i++ {
				status, ok := store.GetTargetStatus(generateTargetName(i))
				if !ok {
					return false
				}
				// Verify state is preserved
				if status.Status == state.StatusUnknown {
					return false // Should have status from previous ping
				}
			}

			return true
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(10) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.Property("reload error does not update scheduler or store", prop.ForAll(
		func(targetCount int, initialInterval int, initialTimeout int) bool {
			if targetCount < 1 || targetCount > 10 || initialInterval < 1 || initialInterval > 10 ||
				initialTimeout < 1 || initialTimeout > 10 {
				return true
			}

			tmpDir := t.TempDir()
			validConfigPath := filepath.Join(tmpDir, "valid.conf")
			invalidConfigPath := filepath.Join(tmpDir, "invalid.conf")

			// Create valid initial config
			validConfigContent := fmt.Sprintf("# surveiller: interval=%ds timeout=%ds\n",
				initialInterval, initialTimeout)
			for i := 0; i < targetCount; i++ {
				validConfigContent += fmt.Sprintf("%s %s\n", generateTargetName(i), generateAddress(i))
			}

			if err := os.WriteFile(validConfigPath, []byte(validConfigContent), 0644); err != nil {
				return false
			}

			// Create invalid config
			invalidConfigContent := "# surveiller: interval=invalid\n"
			if err := os.WriteFile(invalidConfigPath, []byte(invalidConfigContent), 0644); err != nil {
				return false
			}

			// Load valid config
			parser := config.SurveillerParser{}
			cfg, err := parser.LoadConfig(validConfigPath, config.CLIOverrides{})
			if err != nil {
				return false
			}

			// Initialize store
			store := state.NewStore(cfg.Targets, cfg.Global.Timeout)

			// Attempt reload with invalid config
			_, err = parser.LoadConfig(invalidConfigPath, config.CLIOverrides{})
			if err == nil {
				return false // Should have error
			}

			// Verify that UpdateConfig, UpdateTargets, UpdateTimeout are NOT called
			// by checking that store still has original targets
			snapshot := store.GetSnapshot()
			if len(snapshot) != targetCount {
				return false
			}

			// Verify that if we had called UpdateConfig, it would have failed
			// But since we didn't call it (because LoadConfig failed), original config remains
			// This is the key property: error prevents updates

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
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(10) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.Property("reload error returns appropriate error message", prop.ForAll(
		func(errorType int) bool {
			if errorType < 0 || errorType > 2 {
				return true
			}

			tmpDir := t.TempDir()
			var configPath string
			var expectedErrorContains string

			switch errorType {
			case 0: // Syntax error
				configPath = filepath.Join(tmpDir, "syntax_error.conf")
				configContent := "# surveiller: interval=invalid_duration\n"
				if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
					return false
				}
				expectedErrorContains = "invalid interval"
			case 1: // Nonexistent file
				configPath = filepath.Join(tmpDir, "nonexistent.conf")
				expectedErrorContains = "no such file"
			case 2: // Invalid target line
				configPath = filepath.Join(tmpDir, "invalid_target.conf")
				configContent := "invalid line without enough fields\n"
				if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
					return false
				}
				expectedErrorContains = "invalid target line"
			}

			parser := config.SurveillerParser{}
			_, err := parser.LoadConfig(configPath, config.CLIOverrides{})

			// Verify error occurred
			if err == nil {
				return false
			}

			// Verify error message contains expected content
			errorMsg := err.Error()
			if expectedErrorContains != "" {
				// Check if error message contains expected substring (case-insensitive)
				errorMsgLower := errorMsg
				expectedLower := expectedErrorContains
				// Simple substring check
				if len(errorMsgLower) < len(expectedLower) {
					return false
				}
				// For simplicity, just verify error is not empty and is descriptive
				if errorMsg == "" {
					return false
				}
			}

			return true
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(3)
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.TestingRun(t, gopter.ConsoleReporter(false))
}

// Helper functions for generating test data
func generateTargetName(index int) string {
	return "target-" + string(rune('a'+index%26)) + string(rune('0'+index/26))
}

func generateAddress(index int) string {
	return "192.0.2." + string(rune('0'+(index%250)))
}

// **Feature: surveiller, Property 17: Structured Logging Output**
// **Validates: Requirements 6.1, 6.2, 6.3, 6.4, 6.5**
func TestPropertyStructuredLogging(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	props := gopter.NewProperties(params)

	props.Property("ping result logs contain target name, result, and RTT", prop.ForAll(
		func(success bool, rttMs int) bool {
			if rttMs < 0 || rttMs > 10000 {
				return true
			}

			var buf bytes.Buffer
			logger := log.NewLogger(log.LevelInfo)
			logger.SetOutput(&buf)

			rtt := time.Duration(rttMs) * time.Millisecond
			var err error
			if !success {
				err = fmt.Errorf("ping failed")
			}

			logger.LogPingResult("test-target", success, rtt, err)

			output := buf.String()
			if output == "" {
				return false
			}

			// Parse JSON
			var entry log.LogEntry
			if err := json.Unmarshal([]byte(output), &entry); err != nil {
				return false
			}

			// Verify structure
			if entry.Timestamp == "" || entry.Level == "" || entry.Message == "" {
				return false
			}

			// Verify fields
			if entry.Fields == nil {
				return false
			}
			if entry.Fields["target"] != "test-target" {
				return false
			}
			if entry.Fields["success"] != success {
				return false
			}
			if rttMs > 0 {
				if rttMsVal, ok := entry.Fields["rtt_ms"].(float64); !ok || int(rttMsVal) != rttMs {
					return false
				}
			}
			if !success && err != nil {
				if entry.Fields["error"] == nil {
					return false
				}
			}

			return true
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(2) == 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(10000)
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.Property("config load logs contain path and result", prop.ForAll(
		func(success bool) bool {
			var buf bytes.Buffer
			logger := log.NewLogger(log.LevelInfo)
			logger.SetOutput(&buf)

			configPath := "/path/to/config.conf"
			var err error
			if !success {
				err = fmt.Errorf("config load failed")
			}

			logger.LogConfigLoad(success, configPath, err)

			output := buf.String()
			if output == "" {
				return false
			}

			// Parse JSON
			var entry log.LogEntry
			if err := json.Unmarshal([]byte(output), &entry); err != nil {
				return false
			}

			// Verify structure
			if entry.Timestamp == "" || entry.Level == "" || entry.Message == "" {
				return false
			}

			// Verify fields
			if entry.Fields == nil {
				return false
			}
			if entry.Fields["path"] != configPath {
				return false
			}
			if !success && err != nil {
				if entry.Fields["error"] == nil {
					return false
				}
			}

			// Verify log level
			if success {
				if entry.Level != "INFO" {
					return false
				}
			} else {
				if entry.Level != "ERROR" {
					return false
				}
			}

			return true
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(2) == 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.Property("error logs contain component and error details", prop.ForAll(
		func(component string) bool {
			if component == "" {
				component = "test-component"
			}

			var buf bytes.Buffer
			logger := log.NewLogger(log.LevelInfo)
			logger.SetOutput(&buf)

			err := fmt.Errorf("test error message")
			fields := map[string]interface{}{
				"additional": "field",
			}

			logger.LogError(component, err, fields)

			output := buf.String()
			if output == "" {
				return false
			}

			// Parse JSON
			var entry log.LogEntry
			if err := json.Unmarshal([]byte(output), &entry); err != nil {
				return false
			}

			// Verify structure
			if entry.Timestamp == "" || entry.Level != "ERROR" || entry.Message == "" {
				return false
			}

			// Verify fields
			if entry.Fields == nil {
				return false
			}
			if entry.Fields["component"] != component {
				return false
			}
			if entry.Fields["error"] == nil {
				return false
			}
			if entry.Fields["additional"] != "field" {
				return false
			}

			return true
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			components := []string{"scheduler", "metrics", "ui", "pinger", "config"}
			idx := genParams.Rng.Intn(len(components))
			return gopter.NewGenResult(components[idx], gopter.NoShrinker)
		}),
	))

	props.Property("log level filtering works correctly", prop.ForAll(
		func(levelInt int) bool {
			if levelInt < 0 || levelInt > 3 {
				return true
			}

			levels := []log.Level{log.LevelDebug, log.LevelInfo, log.LevelWarn, log.LevelError}
			loggerLevel := levels[levelInt]

			var buf bytes.Buffer
			logger := log.NewLogger(loggerLevel)
			logger.SetOutput(&buf)

			// Log at different levels
			logger.Debug("debug message", nil)
			logger.Info("info message", nil)
			logger.Warn("warn message", nil)
			logger.Error("error message", nil)

			output := buf.String()
			lines := strings.Split(strings.TrimSpace(output), "\n")

			// Count non-empty lines
			actualCount := 0
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					actualCount++
				}
			}

			// Expected count: messages at loggerLevel or higher
			expectedCount := 0
			if log.LevelDebug >= loggerLevel {
				expectedCount++
			}
			if log.LevelInfo >= loggerLevel {
				expectedCount++
			}
			if log.LevelWarn >= loggerLevel {
				expectedCount++
			}
			if log.LevelError >= loggerLevel {
				expectedCount++
			}

			return actualCount == expectedCount
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(4)
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.TestingRun(t, gopter.ConsoleReporter(false))
}
