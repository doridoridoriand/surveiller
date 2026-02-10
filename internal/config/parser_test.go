package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "surveiller.conf")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}

func TestLoadConfigParsesTargetsAndGroups(t *testing.T) {
	configText := "" +
		"# surveiller: interval=2s timeout=1500ms max_concurrency=50 ui.scale=25 ui.disable=true\n" +
		"google 216.58.197.174\n" +
		"googleDNS 8.8.8.8\n" +
		"---\n" +
		"kame 203.178.141.194\n"

	path := writeTempConfig(t, configText)
	parser := SurveillerParser{}

	cfg, err := parser.LoadConfig(path, CLIOverrides{})
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	if len(cfg.Targets) != 3 {
		t.Fatalf("expected 3 targets, got %d", len(cfg.Targets))
	}

	if cfg.Targets[0].Group != "" {
		t.Fatalf("expected empty group for first target, got %q", cfg.Targets[0].Group)
	}
	if cfg.Targets[2].Group != "group-1" {
		t.Fatalf("expected group-1 for third target, got %q", cfg.Targets[2].Group)
	}

	if cfg.Global.Interval != 2*time.Second {
		t.Fatalf("expected interval 2s, got %v", cfg.Global.Interval)
	}
	if cfg.Global.Timeout != 1500*time.Millisecond {
		t.Fatalf("expected timeout 1500ms, got %v", cfg.Global.Timeout)
	}
	if cfg.Global.MaxConcurrency != 50 {
		t.Fatalf("expected max_concurrency 50, got %d", cfg.Global.MaxConcurrency)
	}
	if cfg.Global.UIScale != 25 {
		t.Fatalf("expected ui.scale 25, got %d", cfg.Global.UIScale)
	}
	if !cfg.Global.UIDisable {
		t.Fatalf("expected ui.disable true")
	}
}

func TestLoadConfigParsesNamedGroup(t *testing.T) {
	configText := "" +
		"resolver 8.8.8.8\n" +
		"--- DNS\n" +
		"public 1.1.1.1\n"

	path := writeTempConfig(t, configText)
	parser := SurveillerParser{}

	cfg, err := parser.LoadConfig(path, CLIOverrides{})
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if len(cfg.Targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(cfg.Targets))
	}
	if cfg.Targets[1].Group != "DNS" {
		t.Fatalf("expected group DNS, got %q", cfg.Targets[1].Group)
	}
}

func TestLoadConfigParsesDirectiveWithoutComment(t *testing.T) {
	configText := "" +
		"surveiller: interval=3s metrics.listen=9100\n" +
		"example 192.0.2.1\n"

	path := writeTempConfig(t, configText)
	parser := SurveillerParser{}

	cfg, err := parser.LoadConfig(path, CLIOverrides{})
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if cfg.Global.Interval != 3*time.Second {
		t.Fatalf("expected interval 3s, got %v", cfg.Global.Interval)
	}
	if cfg.Global.MetricsListen != ":9100" {
		t.Fatalf("expected metrics.listen :9100, got %q", cfg.Global.MetricsListen)
	}
}

func TestLoadConfigIgnoresComments(t *testing.T) {
	configText := "" +
		"# normal comment\n" +
		"\n" +
		"example 192.0.2.1\n"

	path := writeTempConfig(t, configText)
	parser := SurveillerParser{}

	cfg, err := parser.LoadConfig(path, CLIOverrides{})
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if len(cfg.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(cfg.Targets))
	}
}

func TestLoadConfigRejectsInvalidTargetLine(t *testing.T) {
	configText := "" +
		"invalidline\n"

	path := writeTempConfig(t, configText)
	parser := SurveillerParser{}

	if _, err := parser.LoadConfig(path, CLIOverrides{}); err == nil {
		t.Fatalf("expected error for invalid target line")
	}
}

func TestLoadConfigRejectsInvalidDirective(t *testing.T) {
	configText := "" +
		"# surveiller: interval=notaduration\n" +
		"example 192.0.2.1\n"

	path := writeTempConfig(t, configText)
	parser := SurveillerParser{}

	if _, err := parser.LoadConfig(path, CLIOverrides{}); err == nil {
		t.Fatalf("expected error for invalid directive")
	}
}

func TestLoadConfigAppliesCLIOverrides(t *testing.T) {
	configText := "" +
		"# surveiller: interval=2s timeout=1500ms max_concurrency=50 ui.disable=false\n" +
		"example 192.0.2.1\n"

	path := writeTempConfig(t, configText)
	parser := SurveillerParser{}

	interval := 5 * time.Second
	timeout := 500 * time.Millisecond
	maxConc := 10
	overrides := CLIOverrides{
		Interval:       &interval,
		Timeout:        &timeout,
		MaxConcurrency: &maxConc,
	}

	cfg, err := parser.LoadConfig(path, overrides)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	if cfg.Global.Interval != interval {
		t.Fatalf("expected interval override %v, got %v", interval, cfg.Global.Interval)
	}
	if cfg.Global.Timeout != timeout {
		t.Fatalf("expected timeout override %v, got %v", timeout, cfg.Global.Timeout)
	}
	if cfg.Global.MaxConcurrency != maxConc {
		t.Fatalf("expected max_concurrency override %d, got %d", maxConc, cfg.Global.MaxConcurrency)
	}
}

func TestParseTargetLineOptions(t *testing.T) {
	parser := SurveillerParser{}
	target, err := parser.ParseTargetLine("relay1 192.0.2.10 relay=jump1 user=foo", "group-1")
	if err != nil {
		t.Fatalf("ParseTargetLine error: %v", err)
	}
	if target.Options["relay"] != "jump1" || target.Options["user"] != "foo" {
		t.Fatalf("expected options parsed, got %+v", target.Options)
	}
}

// TestLoadConfigErrorIncludesLineNumber verifies that parse errors include file path and line number (task 13.1).
func TestLoadConfigErrorIncludesLineNumber(t *testing.T) {
	// Error on line 2: invalid directive on line 1, then invalid target on line 2
	configText := "# surveiller: interval=1s\ninvalidline\n"
	path := writeTempConfig(t, configText)
	parser := SurveillerParser{}

	_, err := parser.LoadConfig(path, CLIOverrides{})
	if err == nil {
		t.Fatal("expected error for invalid target line")
	}
	msg := err.Error()
	if !strings.Contains(msg, path) {
		t.Errorf("error message should contain path %q, got %q", path, msg)
	}
	if !strings.Contains(msg, ":2:") {
		t.Errorf("error message should contain line number :2:, got %q", msg)
	}
	// ConfigError unwraps to the underlying error
	var cfgErr *ConfigError
	if !errors.As(err, &cfgErr) {
		t.Errorf("expected *ConfigError, got %T", err)
	} else if cfgErr.Line != 2 {
		t.Errorf("expected Line 2, got %d", cfgErr.Line)
	}
}

// TestLoadConfigRejectsInvalidDirective_LineNumber verifies directive error includes line number.
func TestLoadConfigRejectsInvalidDirective_LineNumber(t *testing.T) {
	configText := "# surveiller: interval=notaduration\n"
	path := writeTempConfig(t, configText)
	parser := SurveillerParser{}

	_, err := parser.LoadConfig(path, CLIOverrides{})
	if err == nil {
		t.Fatal("expected error for invalid directive")
	}
	if !strings.Contains(err.Error(), ":1:") {
		t.Errorf("error message should contain :1:, got %q", err.Error())
	}
}

// TestLoadConfigValidatesGlobalOptions verifies semantic validation for global options (task 13.2).
func TestLoadConfigValidatesGlobalOptions(t *testing.T) {
	t.Run("interval_zero_via_cli", func(t *testing.T) {
		configText := "surveiller: interval=1s\nhost 8.8.8.8\n"
		path := writeTempConfig(t, configText)
		zero := time.Duration(0)
		overrides := CLIOverrides{Interval: &zero}

		_, err := SurveillerParser{}.LoadConfig(path, overrides)
		if err == nil {
			t.Fatal("expected error for interval 0")
		}
		if !strings.Contains(err.Error(), "interval must be positive") {
			t.Errorf("expected message about interval, got %q", err.Error())
		}
	})

	t.Run("timeout_zero_via_cli", func(t *testing.T) {
		configText := "surveiller: timeout=1s\nhost 8.8.8.8\n"
		path := writeTempConfig(t, configText)
		zero := time.Duration(0)
		overrides := CLIOverrides{Timeout: &zero}

		_, err := SurveillerParser{}.LoadConfig(path, overrides)
		if err == nil {
			t.Fatal("expected error for timeout 0")
		}
		if !strings.Contains(err.Error(), "timeout must be positive") {
			t.Errorf("expected message about timeout, got %q", err.Error())
		}
	})

	t.Run("ui_scale_zero_in_config", func(t *testing.T) {
		configText := "surveiller: ui.scale=0\nhost 8.8.8.8\n"
		path := writeTempConfig(t, configText)

		_, err := SurveillerParser{}.LoadConfig(path, CLIOverrides{})
		if err == nil {
			t.Fatal("expected error for ui.scale 0")
		}
		if !strings.Contains(err.Error(), "ui.scale must be positive") {
			t.Errorf("expected message about ui.scale, got %q", err.Error())
		}
	})

	t.Run("max_concurrency_zero_via_cli", func(t *testing.T) {
		configText := "surveiller: max_concurrency=10\nhost 8.8.8.8\n"
		path := writeTempConfig(t, configText)
		zero := 0
		overrides := CLIOverrides{MaxConcurrency: &zero}

		_, err := SurveillerParser{}.LoadConfig(path, overrides)
		if err == nil {
			t.Fatal("expected error for max_concurrency 0")
		}
		if !strings.Contains(err.Error(), "max_concurrency must be positive") {
			t.Errorf("expected message about max_concurrency, got %q", err.Error())
		}
	})
}

// TestLoadConfigRejectsDuplicateTargetName verifies duplicate target names are rejected (task 13.3).
func TestLoadConfigRejectsDuplicateTargetName(t *testing.T) {
	configText := "same 8.8.8.8\nsame 1.1.1.1\n"
	path := writeTempConfig(t, configText)
	parser := SurveillerParser{}

	_, err := parser.LoadConfig(path, CLIOverrides{})
	if err == nil {
		t.Fatal("expected error for duplicate target name")
	}
	if !strings.Contains(err.Error(), "duplicate target name") {
		t.Errorf("expected message about duplicate name, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), ":2:") {
		t.Errorf("error should reference line 2 (second definition), got %q", err.Error())
	}
}
