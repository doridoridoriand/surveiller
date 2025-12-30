package config

import "time"

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
