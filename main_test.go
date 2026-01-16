package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/doridoridoriand/surveiller/internal/cli"
	"github.com/doridoridoriand/surveiller/internal/config"
)

// 6.1 アプリケーション初期化の単体テスト
func TestBuildOverrides(t *testing.T) {
	tests := []struct {
		name            string
		setupInterval   func() cli.OptionalDuration
		setupTimeout    func() cli.OptionalDuration
		setupMaxConc    func() cli.OptionalInt
		setupMetrics    func() cli.OptionalMetricsMode
		setupListen     func() cli.OptionalString
		setupNoUI       func() cli.OptionalBool
		checkInterval   bool
		checkTimeout    bool
		checkMaxConc    bool
		checkMetrics    bool
		checkListen     bool
		checkUIDisable  bool
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
			checkInterval:  true,
			checkTimeout:   true,
			checkMaxConc:   true,
			checkMetrics:   true,
			checkListen:    true,
			checkUIDisable: true,
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
			setupMaxConc: func() cli.OptionalInt { return cli.OptionalInt{} },
			setupMetrics: func() cli.OptionalMetricsMode { return cli.OptionalMetricsMode{} },
			setupListen:  func() cli.OptionalString { return cli.OptionalString{} },
			setupNoUI:    func() cli.OptionalBool { return cli.OptionalBool{} },
			checkInterval: true,
			checkTimeout:  true,
			expectedInterval: 3 * time.Second,
			expectedTimeout:  2 * time.Second,
		},
		{
			name: "no overrides",
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
