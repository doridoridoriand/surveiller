package config

import (
	"bufio"
	"fmt"
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

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#") {
			if strings.HasPrefix(line, "# surveiller:") {
				pairs, err := p.ParseSurveillerDirective(line)
				if err != nil {
					return nil, err
				}
				if err := applyDirective(&cfg.Global, pairs); err != nil {
					return nil, err
				}
			}
			continue
		}

		if strings.HasPrefix(line, "surveiller:") {
			pairs, err := p.ParseSurveillerDirective(line)
			if err != nil {
				return nil, err
			}
			if err := applyDirective(&cfg.Global, pairs); err != nil {
				return nil, err
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
			return nil, err
		}
		cfg.Targets = append(cfg.Targets, target)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	applyCLIOverrides(&cfg.Global, overrides)
	return cfg, nil
}

// ParseSurveillerDirective extracts key=value pairs from a directive line.
func (p SurveillerParser) ParseSurveillerDirective(line string) (map[string]string, error) {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") {
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
	} else if !strings.HasPrefix(trimmed, "surveiller:") {
		return nil, fmt.Errorf("directive line must start with '# surveiller:' or 'surveiller:': %q", line)
	}
	payload := strings.TrimSpace(strings.TrimPrefix(trimmed, "surveiller:"))
	if payload == "" {
		return map[string]string{}, nil
	}

	pairs := make(map[string]string)
	for _, token := range strings.Fields(payload) {
		kv := strings.SplitN(token, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid directive token: %q", token)
		}
		pairs[kv[0]] = kv[1]
	}
	return pairs, nil
}

// ParseTargetLine parses a single target definition.
func (p SurveillerParser) ParseTargetLine(line string, group string) (TargetConfig, error) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return TargetConfig{}, fmt.Errorf("invalid target line: %q", line)
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
				return TargetConfig{}, fmt.Errorf("invalid target option: %q", field)
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
				return fmt.Errorf("invalid interval: %w", err)
			}
			global.Interval = d
		case "timeout":
			d, err := time.ParseDuration(val)
			if err != nil {
				return fmt.Errorf("invalid timeout: %w", err)
			}
			global.Timeout = d
		case "max_concurrency":
			n, err := strconv.Atoi(val)
			if err != nil {
				return fmt.Errorf("invalid max_concurrency: %w", err)
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
				return fmt.Errorf("invalid metrics.mode: %q", val)
			}
		case "metrics.listen":
			if isDigits(val) {
				global.MetricsListen = ":" + val
			} else {
				global.MetricsListen = val
			}
		case "ui.scale":
			n, err := strconv.Atoi(val)
			if err != nil {
				return fmt.Errorf("invalid ui.scale: %w", err)
			}
			global.UIScale = n
		case "ui.disable":
			b, err := strconv.ParseBool(val)
			if err != nil {
				return fmt.Errorf("invalid ui.disable: %w", err)
			}
			global.UIDisable = b
		default:
			// Ignore unknown keys for forward compatibility.
		}
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
		val := *overrides.MetricsListen
		if isDigits(val) {
			val = ":" + val
		}
		global.MetricsListen = val
	}
	if overrides.UIDisable != nil {
		global.UIDisable = *overrides.UIDisable
	}
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
