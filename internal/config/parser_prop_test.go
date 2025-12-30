package config

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/prop"
)

type targetSpec struct {
	Name    string
	Address string
}

type configSpec struct {
	Groups [][]targetSpec
}

type directiveSpec struct {
	IntervalMs     int
	TimeoutMs      int
	MaxConcurrency int
	MetricsMode    MetricsMode
	MetricsListen  string
	UIScale        int
	UIDisable      bool
}

func TestPropertyConfigParsing(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 25
	props := gopter.NewProperties(params)

	props.Property("targets and groups are parsed accurately", prop.ForAll(
		func(spec configSpec) bool {
			configText, expected := buildConfigFromSpec(spec)
			parser := SurveillerParser{}
			path := writeTempConfig(t, configText)
			cfg, err := parser.LoadConfig(path, CLIOverrides{})
			if err != nil {
				return false
			}
			if len(cfg.Targets) != len(expected) {
				return false
			}
			for i, tgt := range expected {
				got := cfg.Targets[i]
				if got.Name != tgt.Name || got.Address != tgt.Address || got.Group != tgt.Group {
					return false
				}
			}
			return true
		},
		genConfigSpec(),
	))

	props.TestingRun(t)
}

func TestPropertyDirectiveParsing(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 25
	props := gopter.NewProperties(params)

	props.Property("surveiller directives map to GlobalOptions", prop.ForAll(
		func(spec directiveSpec) bool {
			directive := fmt.Sprintf(
				"# surveiller: interval=%dms timeout=%dms max_concurrency=%d metrics.mode=%s metrics.listen=%s ui.scale=%d ui.disable=%t\n",
				spec.IntervalMs,
				spec.TimeoutMs,
				spec.MaxConcurrency,
				spec.MetricsMode,
				spec.MetricsListen,
				spec.UIScale,
				spec.UIDisable,
			)
			parser := SurveillerParser{}
			path := writeTempConfig(t, directive)
			cfg, err := parser.LoadConfig(path, CLIOverrides{})
			if err != nil {
				return false
			}
			if cfg.Global.Interval != time.Duration(spec.IntervalMs)*time.Millisecond {
				return false
			}
			if cfg.Global.Timeout != time.Duration(spec.TimeoutMs)*time.Millisecond {
				return false
			}
			if cfg.Global.MaxConcurrency != spec.MaxConcurrency {
				return false
			}
			if cfg.Global.MetricsMode != spec.MetricsMode {
				return false
			}
			if cfg.Global.MetricsListen != spec.MetricsListen {
				return false
			}
			if cfg.Global.UIScale != spec.UIScale {
				return false
			}
			if cfg.Global.UIDisable != spec.UIDisable {
				return false
			}
			return true
		},
		genDirectiveSpec(),
	))

	props.TestingRun(t)
}

func TestPropertyCommentHandling(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 25
	props := gopter.NewProperties(params)

	props.Property("comment-only files produce no targets", prop.ForAll(
		func(count int) bool {
			if count < 1 {
				return true
			}
			lines := make([]string, 0, count)
			for i := 0; i < count; i++ {
				lines = append(lines, "# comment")
			}
			parser := SurveillerParser{}
			path := writeTempConfig(t, strings.Join(lines, "\n"))
			cfg, err := parser.LoadConfig(path, CLIOverrides{})
			if err != nil {
				return false
			}
			return len(cfg.Targets) == 0
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			count := genParams.Rng.Intn(10) + 1
			return gopter.NewGenResult(count, gopter.NoShrinker)
		}),
	))

	props.Property("invalid target lines are rejected", prop.ForAll(
		func(token string) bool {
			token = strings.TrimSpace(token)
			if token == "" {
				token = "invalid"
			}
			parser := SurveillerParser{}
			path := writeTempConfig(t, token+"\n")
			_, err := parser.LoadConfig(path, CLIOverrides{})
			return err != nil
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			token := randomToken(genParams.Rng)
			return gopter.NewGenResult(token, gopter.NoShrinker)
		}),
	))

	props.TestingRun(t)
}

func TestPropertyCLIPriority(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 25
	props := gopter.NewProperties(params)

	props.Property("CLI overrides config values", prop.ForAll(
		func(intervalMs, timeoutMs, maxConc int) bool {
			if intervalMs < 1 || timeoutMs < 1 || maxConc < 1 {
				return true
			}
			configText := fmt.Sprintf(
				"# surveiller: interval=%dms timeout=%dms max_concurrency=%d ui.disable=false\n",
				intervalMs,
				timeoutMs,
				maxConc,
			)
			parser := SurveillerParser{}
			path := writeTempConfig(t, configText)

			overrideInterval := time.Duration(intervalMs+1) * time.Millisecond
			overrideTimeout := time.Duration(timeoutMs+1) * time.Millisecond
			overrideMaxConc := maxConc + 1
			overrideNoUI := true
			overrides := CLIOverrides{
				Interval:       &overrideInterval,
				Timeout:        &overrideTimeout,
				MaxConcurrency: &overrideMaxConc,
				UIDisable:      &overrideNoUI,
			}

			cfg, err := parser.LoadConfig(path, overrides)
			if err != nil {
				return false
			}

			return cfg.Global.Interval == overrideInterval &&
				cfg.Global.Timeout == overrideTimeout &&
				cfg.Global.MaxConcurrency == overrideMaxConc &&
				cfg.Global.UIDisable == overrideNoUI
		},
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(500) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(500) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
		gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
			value := genParams.Rng.Intn(50) + 1
			return gopter.NewGenResult(value, gopter.NoShrinker)
		}),
	))

	props.TestingRun(t)
}

func genConfigSpec() gopter.Gen {
	return gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
		groupCount := genParams.Rng.Intn(3) + 1
		groups := make([][]targetSpec, groupCount)
		for i := 0; i < groupCount; i++ {
			targetCount := genParams.Rng.Intn(3) + 1
			group := make([]targetSpec, targetCount)
			for j := 0; j < targetCount; j++ {
				group[j] = targetSpec{
					Name:    randomToken(genParams.Rng),
					Address: randomToken(genParams.Rng),
				}
			}
			groups[i] = group
		}
		spec := configSpec{Groups: groups}
		return gopter.NewGenResult(spec, gopter.NoShrinker)
	})
}

func genDirectiveSpec() gopter.Gen {
	return gopter.Gen(func(genParams *gopter.GenParameters) *gopter.GenResult {
		modes := []MetricsMode{MetricsModePerTarget, MetricsModeAggregated, MetricsModeBoth}
		spec := directiveSpec{
			IntervalMs:     genParams.Rng.Intn(4000) + 1,
			TimeoutMs:      genParams.Rng.Intn(4000) + 1,
			MaxConcurrency: genParams.Rng.Intn(200) + 1,
			MetricsMode:    modes[genParams.Rng.Intn(len(modes))],
			MetricsListen:  fmt.Sprintf(":%d", genParams.Rng.Intn(60000)+1024),
			UIScale:        genParams.Rng.Intn(100) + 1,
			UIDisable:      genParams.Rng.Intn(2) == 0,
		}
		return gopter.NewGenResult(spec, gopter.NoShrinker)
	})
}

func buildConfigFromSpec(spec configSpec) (string, []TargetConfig) {
	var lines []string
	var expected []TargetConfig

	for groupIndex, group := range spec.Groups {
		if groupIndex > 0 {
			lines = append(lines, "---")
		}
		groupName := ""
		if groupIndex > 0 {
			groupName = fmt.Sprintf("group-%d", groupIndex)
		}
		for _, tgt := range group {
			lines = append(lines, fmt.Sprintf("%s %s", tgt.Name, tgt.Address))
			expected = append(expected, TargetConfig{
				Name:    tgt.Name,
				Address: tgt.Address,
				Group:   groupName,
				Options: map[string]string{},
			})
		}
	}

	return strings.Join(lines, "\n"), expected
}

func randomToken(rng *rand.Rand) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	length := rng.Intn(8) + 1
	buf := make([]byte, length)
	for i := range buf {
		buf[i] = letters[rng.Intn(len(letters))]
	}
	return string(buf)
}
