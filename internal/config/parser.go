package config

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// SurveillerParser implements the Parser interface.
type SurveillerParser struct{}

// DefaultGlobalOptions returns baseline settings used before config overrides.
func DefaultGlobalOptions() GlobalOptions {
	return GlobalOptions{
		Interval:       1 * time.Second,
		Timeout:        1 * time.Second,
		MaxConcurrency: 100,
		MetricsMode:    MetricsModePerTarget,
		MetricsListen:  "",
		UIScale:        10,
		UIDisable:      false,
	}
}

// LoadConfig parses a surveiller.conf file with CLI overrides applied.
// Errors include file path and line number when applicable (ConfigError).
func (p SurveillerParser) LoadConfig(path string, overrides CLIOverrides) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cfg := &Config{Global: DefaultGlobalOptions()}

	scanner := bufio.NewScanner(file)
	groupIndex := 0
	currentGroup := ""
	seenNames := make(map[string]int) // name -> line number (1-based)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#") {
			if strings.HasPrefix(line, "# surveiller:") {
				pairs, err := p.ParseSurveillerDirective(line)
				if err != nil {
					return nil, &ConfigError{Path: path, Line: lineNum, Err: err}
				}
				if err := applyDirective(&cfg.Global, pairs); err != nil {
					return nil, &ConfigError{Path: path, Line: lineNum, Err: err}
				}
			}
			continue
		}

		if strings.HasPrefix(line, "surveiller:") {
			pairs, err := p.ParseSurveillerDirective(line)
			if err != nil {
				return nil, &ConfigError{Path: path, Line: lineNum, Err: err}
			}
			if err := applyDirective(&cfg.Global, pairs); err != nil {
				return nil, &ConfigError{Path: path, Line: lineNum, Err: err}
			}
			continue
		}

		if strings.HasPrefix(line, "---") {
			groupIndex++
			groupName := strings.TrimSpace(strings.TrimPrefix(line, "---"))
			if groupName == "" {
				groupName = fmt.Sprintf("group-%d", groupIndex)
			}
			currentGroup = groupName
			continue
		}

		target, err := p.ParseTargetLine(line, currentGroup)
		if err != nil {
			return nil, &ConfigError{Path: path, Line: lineNum, Err: err}
		}
		if err := validateTargetLine(path, lineNum, target, seenNames); err != nil {
			return nil, err
		}
		seenNames[target.Name] = lineNum
		cfg.Targets = append(cfg.Targets, target)
	}

	if err := scanner.Err(); err != nil {
		return nil, &ConfigError{Path: path, Line: 0, Err: err}
	}

	applyCLIOverrides(&cfg.Global, overrides)
	if err := validateGlobal(path, &cfg.Global); err != nil {
		return nil, err
	}
	return cfg, nil
}

// ParseSurveillerDirective extracts key=value pairs from a directive line.
func (p SurveillerParser) ParseSurveillerDirective(line string) (map[string]string, error) {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") {
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
	} else if !strings.HasPrefix(trimmed, "surveiller:") {
		return nil, fmt.Errorf("invalid directive: line must start with \"# surveiller:\" or \"surveiller:\", got %q", line)
	}
	payload := strings.TrimSpace(strings.TrimPrefix(trimmed, "surveiller:"))
	if payload == "" {
		return map[string]string{}, nil
	}

	pairs := make(map[string]string)
	for _, token := range strings.Fields(payload) {
		kv := strings.SplitN(token, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid directive token %q (expected: key=value)", token)
		}
		pairs[kv[0]] = kv[1]
	}
	return pairs, nil
}

// ParseTargetLine parses a single target definition.
func (p SurveillerParser) ParseTargetLine(line string, group string) (TargetConfig, error) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return TargetConfig{}, fmt.Errorf("invalid target line %q (expected: name address [key=value ...])", line)
	}

	target := TargetConfig{
		Name:    fields[0],
		Address: fields[1],
		Group:   group,
		Options: map[string]string{},
	}

	if len(fields) > 2 {
		for _, field := range fields[2:] {
			kv := strings.SplitN(field, "=", 2)
			if len(kv) != 2 {
				return TargetConfig{}, fmt.Errorf("invalid target option %q (expected: key=value)", field)
			}
			target.Options[kv[0]] = kv[1]
		}
	}

	return target, nil
}

func applyDirective(global *GlobalOptions, pairs map[string]string) error {
	for key, val := range pairs {
		switch key {
		case "interval":
			d, err := time.ParseDuration(val)
			if err != nil {
				return fmt.Errorf("invalid interval %q: %w (expected: positive duration, e.g. 1s, 500ms)", val, err)
			}
			global.Interval = d
		case "timeout":
			d, err := time.ParseDuration(val)
			if err != nil {
				return fmt.Errorf("invalid timeout %q: %w (expected: positive duration, e.g. 1s, 2s)", val, err)
			}
			global.Timeout = d
		case "max_concurrency":
			n, err := strconv.Atoi(val)
			if err != nil {
				return fmt.Errorf("invalid max_concurrency %q: %w (expected: positive integer)", val, err)
			}
			global.MaxConcurrency = n
		case "metrics.mode":
			switch val {
			case string(MetricsModePerTarget):
				global.MetricsMode = MetricsModePerTarget
			case string(MetricsModeAggregated):
				global.MetricsMode = MetricsModeAggregated
			case string(MetricsModeBoth):
				global.MetricsMode = MetricsModeBoth
			default:
				return fmt.Errorf("invalid metrics.mode %q (valid values: per-target, aggregated, both)", val)
			}
		case "metrics.listen":
			listen := normalizeMetricsListen(val)
			if err := validateMetricsListen(listen); err != nil {
				return err
			}
			global.MetricsListen = listen
		case "ui.scale":
			n, err := strconv.Atoi(val)
			if err != nil {
				return fmt.Errorf("invalid ui.scale %q: %w (expected: positive integer)", val, err)
			}
			global.UIScale = n
		case "ui.disable":
			b, err := strconv.ParseBool(val)
			if err != nil {
				return fmt.Errorf("invalid ui.disable %q: %w (expected: true or false)", val, err)
			}
			global.UIDisable = b
		default:
			// Ignore unknown keys for forward compatibility.
		}
	}
	return nil
}

func validateGlobal(path string, global *GlobalOptions) error {
	if global.Interval <= 0 {
		return &ConfigError{Path: path, Line: 0, Err: fmt.Errorf("interval must be positive, got %v (expected: e.g. 1s, 500ms)", global.Interval)}
	}
	if global.Timeout <= 0 {
		return &ConfigError{Path: path, Line: 0, Err: fmt.Errorf("timeout must be positive, got %v (expected: e.g. 1s, 2s)", global.Timeout)}
	}
	if global.MaxConcurrency <= 0 {
		return &ConfigError{Path: path, Line: 0, Err: fmt.Errorf("max_concurrency must be positive, got %d", global.MaxConcurrency)}
	}
	if global.UIScale <= 0 {
		return &ConfigError{Path: path, Line: 0, Err: fmt.Errorf("ui.scale must be positive, got %d", global.UIScale)}
	}
	if err := validateMetricsListen(global.MetricsListen); err != nil {
		return &ConfigError{Path: path, Line: 0, Err: err}
	}
	return nil
}

func validateTargetLine(path string, lineNum int, target TargetConfig, seenNames map[string]int) error {
	if target.Name == "" {
		return &ConfigError{Path: path, Line: lineNum, Err: fmt.Errorf("target name must be non-empty")}
	}
	if target.Address == "" {
		return &ConfigError{Path: path, Line: lineNum, Err: fmt.Errorf("target address must be non-empty")}
	}
	if prev, ok := seenNames[target.Name]; ok {
		return &ConfigError{Path: path, Line: lineNum, Err: fmt.Errorf("duplicate target name %q (already defined at line %d)", target.Name, prev)}
	}
	return nil
}

func applyCLIOverrides(global *GlobalOptions, overrides CLIOverrides) {
	if overrides.Interval != nil {
		global.Interval = *overrides.Interval
	}
	if overrides.Timeout != nil {
		global.Timeout = *overrides.Timeout
	}
	if overrides.MaxConcurrency != nil {
		global.MaxConcurrency = *overrides.MaxConcurrency
	}
	if overrides.MetricsMode != nil {
		global.MetricsMode = *overrides.MetricsMode
	}
	if overrides.MetricsListen != nil {
		global.MetricsListen = normalizeMetricsListen(*overrides.MetricsListen)
	}
	if overrides.UIDisable != nil {
		global.UIDisable = *overrides.UIDisable
	}
}

func normalizeMetricsListen(value string) string {
	if isDigits(value) {
		return ":" + value
	}
	return value
}

func validateMetricsListen(addr string) error {
	if addr == "" {
		return nil
	}
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid metrics.listen address %q: %w", addr, err)
	}
	portNum, err := strconv.Atoi(port)
	if err != nil || portNum < 1 || portNum > 65535 {
		return fmt.Errorf("invalid metrics.listen port %q in address %q", port, addr)
	}
	return nil
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
