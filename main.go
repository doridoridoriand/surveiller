package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/doridoridoriand/deadman-go/internal/cli"
	"github.com/doridoridoriand/deadman-go/internal/config"
)

const version = "0.1.0"

func main() {
	var (
		flagInterval       cli.OptionalDuration
		flagTimeout        cli.OptionalDuration
		flagMaxConcurrency cli.OptionalInt
		flagMetricsMode    cli.OptionalString
		flagMetricsListen  cli.OptionalString
		flagNoUI           cli.OptionalBool
		flagVersion        bool
		flagVersionShort   bool
	)

	flag.Var(&flagInterval, "interval", "ping interval per target (override config)")
	flag.Var(&flagInterval, "i", "ping interval per target (override config)")
	flag.Var(&flagTimeout, "timeout", "ping timeout (override config)")
	flag.Var(&flagTimeout, "t", "ping timeout (override config)")
	flag.Var(&flagMaxConcurrency, "max-concurrency", "max concurrent pings (override config)")
	flag.Var(&flagMetricsMode, "metrics-mode", "metrics mode: per-target|aggregated|both")
	flag.Var(&flagMetricsListen, "metrics-listen", "metrics listen address (e.g. :9100)")
	flag.Var(&flagNoUI, "no-ui", "disable TUI (log only)")
	flag.BoolVar(&flagVersion, "version", false, "show version")
	flag.BoolVar(&flagVersionShort, "v", false, "show version")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [options] <config-file>\n\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
	}

	flag.Parse()

	if flagVersion || flagVersionShort {
		fmt.Fprintf(os.Stdout, "deadman-go version %s\n", version)
		return
	}

	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}
	configPath := args[0]

	overrides := buildOverrides(flagInterval, flagTimeout, flagMaxConcurrency, flagMetricsMode, flagMetricsListen, flagNoUI)

	parser := config.DeadmanParser{}
	cfg, err := parser.LoadConfig(configPath, overrides)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "loaded %d targets\n", len(cfg.Targets))
}

func buildOverrides(
	interval cli.OptionalDuration,
	timeout cli.OptionalDuration,
	maxConcurrency cli.OptionalInt,
	metricsMode cli.OptionalString,
	metricsListen cli.OptionalString,
	noUI cli.OptionalBool,
) config.CLIOverrides {
	overrides := config.CLIOverrides{}

	if v, ok := interval.Value(); ok {
		value := v
		overrides.Interval = &value
	}
	if v, ok := timeout.Value(); ok {
		value := v
		overrides.Timeout = &value
	}
	if v, ok := maxConcurrency.Value(); ok {
		value := v
		overrides.MaxConcurrency = &value
	}
	if v, ok := metricsMode.Value(); ok && v != "" {
		value := config.MetricsMode(v)
		overrides.MetricsMode = &value
	}
	if v, ok := metricsListen.Value(); ok && v != "" {
		value := v
		overrides.MetricsListen = &value
	}
	if v, ok := noUI.Value(); ok {
		value := v
		overrides.UIDisable = &value
	}

	return overrides
}
