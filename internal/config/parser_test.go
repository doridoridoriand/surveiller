package config

import (
	"os"
	"path/filepath"
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
