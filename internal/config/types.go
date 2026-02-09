package config

import (
	"fmt"
	"time"
)

// ConfigError is an error that includes config file path and optional line number.
type ConfigError struct {
	Path string
	Line int // 1-based; 0 means unknown or global-level
	Err  error
}

func (e *ConfigError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s:%d: %s", e.Path, e.Line, e.Err.Error())
	}
	return fmt.Sprintf("%s: %s", e.Path, e.Err.Error())
}

func (e *ConfigError) Unwrap() error { return e.Err }

// MetricsMode describes the granularity of metrics (future use).
type MetricsMode string

const (
	MetricsModePerTarget  MetricsMode = "per-target"
	MetricsModeAggregated MetricsMode = "aggregated"
	MetricsModeBoth       MetricsMode = "both"
)

// GlobalOptions holds global settings parsed from config and CLI overrides.
type GlobalOptions struct {
	Interval       time.Duration
	Timeout        time.Duration
	MaxConcurrency int
	MetricsMode    MetricsMode
	MetricsListen  string
	UIScale        int
	UIDisable      bool
}

// TargetConfig represents a single target definition.
type TargetConfig struct {
	Name    string
	Address string
	Group   string
	Options map[string]string
}

// Config is the parsed configuration file with global settings.
type Config struct {
	Targets []TargetConfig
	Global  GlobalOptions
}

// CLIOverrides holds optional CLI values that override config file values.
type CLIOverrides struct {
	Interval       *time.Duration
	Timeout        *time.Duration
	MaxConcurrency *int
	MetricsMode    *MetricsMode
	MetricsListen  *string
	UIDisable      *bool
}

// Parser defines config parsing behavior.
type Parser interface {
	LoadConfig(path string, overrides CLIOverrides) (*Config, error)
	ParseSurveillerDirective(line string) (map[string]string, error)
	ParseTargetLine(line string, group string) (TargetConfig, error)
}
