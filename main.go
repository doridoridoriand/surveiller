package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/doridoridoriand/deadman-go/internal/cli"
	"github.com/doridoridoriand/deadman-go/internal/config"
	"github.com/doridoridoriand/deadman-go/internal/metrics"
	"github.com/doridoridoriand/deadman-go/internal/ping"
	"github.com/doridoridoriand/deadman-go/internal/scheduler"
	"github.com/doridoridoriand/deadman-go/internal/state"
	"github.com/doridoridoriand/deadman-go/internal/ui"
)

const version = "0.0.1"

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

	icmpPinger, err := ping.NewICMPPinger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize pinger: %v\n", err)
		os.Exit(1)
	}
	pinger := ping.NewFallbackPinger(icmpPinger, ping.NewExternalPinger())

	store := state.NewStore(cfg.Targets, cfg.Global.Timeout)
	sched := scheduler.NewScheduler(cfg.Global, cfg.Targets, pinger, store)

	ctx, cancel := signalContext()
	defer cancel()

	var wg sync.WaitGroup
	if cfg.Global.MetricsListen != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := metrics.Serve(ctx, cfg.Global.MetricsListen, cfg.Global.MetricsMode, store); err != nil && !errors.Is(err, context.Canceled) {
				fmt.Fprintf(os.Stderr, "metrics error: %v\n", err)
				cancel()
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := sched.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			fmt.Fprintf(os.Stderr, "scheduler error: %v\n", err)
			cancel()
		}
	}()

	if cfg.Global.UIDisable {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runTextReporter(ctx, store)
		}()
		<-ctx.Done()
	} else {
		ui := ui.New(cfg.Global, store)
		if err := ui.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			fmt.Fprintf(os.Stderr, "ui error: %v\n", err)
			cancel()
		}
	}

	wg.Wait()
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

func signalContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		cancel()
	}()
	return ctx, cancel
}

func runTextReporter(ctx context.Context, store state.Store) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			snapshot := store.GetSnapshot()
			if len(snapshot) == 0 {
				continue
			}
			sort.Slice(snapshot, func(i, j int) bool {
				return snapshot[i].Name < snapshot[j].Name
			})
			fmt.Fprintf(os.Stdout, "[%s] targets=%d\n", time.Now().Format(time.RFC3339), len(snapshot))
			for _, target := range snapshot {
				fmt.Fprintf(
					os.Stdout,
					"- %s (%s) status=%s rtt=%s ok=%d ng=%d\n",
					target.Name,
					target.Address,
					target.Status,
					target.LastRTT,
					target.ConsecutiveOK,
					target.ConsecutiveNG,
				)
			}
		}
	}
}
