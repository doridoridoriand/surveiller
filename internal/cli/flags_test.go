package cli

import (
	"testing"
	"time"

	"github.com/doridoridoriand/surveiller/internal/config"
)

func TestOptionalDuration(t *testing.T) {
	var d OptionalDuration
	if d.String() != "" {
		t.Fatalf("expected empty string for unset duration")
	}
	if _, ok := d.Value(); ok {
		t.Fatalf("expected unset duration to report false")
	}
	if err := d.Set("250ms"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.String() != "250ms" {
		t.Fatalf("expected duration string to be 250ms, got %q", d.String())
	}
	if v, ok := d.Value(); !ok || v != 250*time.Millisecond {
		t.Fatalf("expected duration value 250ms, got %v (ok=%v)", v, ok)
	}
}

func TestOptionalDurationInvalid(t *testing.T) {
	var d OptionalDuration
	if err := d.Set("bad"); err == nil {
		t.Fatalf("expected error for invalid duration")
	}
	if _, ok := d.Value(); ok {
		t.Fatalf("expected invalid duration to remain unset")
	}
}

func TestOptionalInt(t *testing.T) {
	var i OptionalInt
	if i.String() != "" {
		t.Fatalf("expected empty string for unset int")
	}
	if _, ok := i.Value(); ok {
		t.Fatalf("expected unset int to report false")
	}
	if err := i.Set("42"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if i.String() != "42" {
		t.Fatalf("expected int string to be 42, got %q", i.String())
	}
	if v, ok := i.Value(); !ok || v != 42 {
		t.Fatalf("expected int value 42, got %v (ok=%v)", v, ok)
	}
}

func TestOptionalIntInvalid(t *testing.T) {
	var i OptionalInt
	if err := i.Set("bad"); err == nil {
		t.Fatalf("expected error for invalid int")
	}
	if _, ok := i.Value(); ok {
		t.Fatalf("expected invalid int to remain unset")
	}
}

func TestOptionalString(t *testing.T) {
	var s OptionalString
	if s.String() != "" {
		t.Fatalf("expected empty string for unset string")
	}
	if _, ok := s.Value(); ok {
		t.Fatalf("expected unset string to report false")
	}
	if err := s.Set("hello"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.String() != "hello" {
		t.Fatalf("expected string value to be hello, got %q", s.String())
	}
	if v, ok := s.Value(); !ok || v != "hello" {
		t.Fatalf("expected string value hello, got %q (ok=%v)", v, ok)
	}
}

func TestOptionalBool(t *testing.T) {
	var b OptionalBool
	if b.String() != "" {
		t.Fatalf("expected empty string for unset bool")
	}
	if _, ok := b.Value(); ok {
		t.Fatalf("expected unset bool to report false")
	}
	if !b.IsBoolFlag() {
		t.Fatalf("expected IsBoolFlag to return true")
	}
	if err := b.Set("true"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.String() != "true" {
		t.Fatalf("expected bool string to be true, got %q", b.String())
	}
	if v, ok := b.Value(); !ok || v != true {
		t.Fatalf("expected bool value true, got %v (ok=%v)", v, ok)
	}
}

func TestOptionalBoolInvalid(t *testing.T) {
	var b OptionalBool
	if err := b.Set("bad"); err == nil {
		t.Fatalf("expected error for invalid bool")
	}
	if _, ok := b.Value(); ok {
		t.Fatalf("expected invalid bool to remain unset")
	}
}

// TestOptionalMetricsMode tests MetricsMode flag parsing
func TestOptionalMetricsMode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected config.MetricsMode
		wantErr  bool
	}{
		{
			name:     "per-target mode",
			input:    "per-target",
			expected: config.MetricsModePerTarget,
			wantErr:  false,
		},
		{
			name:     "aggregated mode",
			input:    "aggregated",
			expected: config.MetricsModeAggregated,
			wantErr:  false,
		},
		{
			name:     "both mode",
			input:    "both",
			expected: config.MetricsModeBoth,
			wantErr:  false,
		},
		{
			name:     "invalid mode",
			input:    "invalid",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m OptionalMetricsMode

			// Test initial state
			if m.String() != "" {
				t.Fatalf("expected empty string for unset MetricsMode")
			}
			if _, ok := m.Value(); ok {
				t.Fatalf("expected unset MetricsMode to report false")
			}

			// Test setting value
			err := m.Set(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for input %q", tt.input)
				}
				// Should remain unset after error
				if _, ok := m.Value(); ok {
					t.Fatalf("expected MetricsMode to remain unset after error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error for input %q: %v", tt.input, err)
			}

			// Test string representation
			if m.String() != tt.input {
				t.Fatalf("expected string to be %q, got %q", tt.input, m.String())
			}

			// Test value retrieval
			if v, ok := m.Value(); !ok || v != tt.expected {
				t.Fatalf("expected MetricsMode value %q, got %q (ok=%v)", tt.expected, v, ok)
			}
		})
	}
}

// TestOptionalMetricsModeErrorMessages tests specific error messages
func TestOptionalMetricsModeErrorMessages(t *testing.T) {
	var m OptionalMetricsMode
	err := m.Set("invalid-mode")
	if err == nil {
		t.Fatalf("expected error for invalid metrics mode")
	}

	expectedMsg := `invalid metrics mode: "invalid-mode" (valid values: per-target, aggregated, both)`
	if err.Error() != expectedMsg {
		t.Fatalf("expected error message %q, got %q", expectedMsg, err.Error())
	}
}

// TestAllFlagTypesErrorHandling tests error handling across all flag types
func TestAllFlagTypesErrorHandling(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func() error
	}{
		{
			name: "OptionalDuration invalid",
			testFunc: func() error {
				var d OptionalDuration
				return d.Set("invalid-duration")
			},
		},
		{
			name: "OptionalInt invalid",
			testFunc: func() error {
				var i OptionalInt
				return i.Set("not-a-number")
			},
		},
		{
			name: "OptionalBool invalid",
			testFunc: func() error {
				var b OptionalBool
				return b.Set("not-a-bool")
			},
		},
		{
			name: "OptionalMetricsMode invalid",
			testFunc: func() error {
				var m OptionalMetricsMode
				return m.Set("invalid-mode")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFunc()
			if err == nil {
				t.Fatalf("expected error for %s", tt.name)
			}
		})
	}
}

// TestFlagTypesBoolFlagInterface tests IsBoolFlag interface
func TestFlagTypesBoolFlagInterface(t *testing.T) {
	var b OptionalBool
	if !b.IsBoolFlag() {
		t.Fatalf("expected OptionalBool to implement IsBoolFlag() returning true")
	}
}

// TestFlagTypesStringRepresentation tests String() method for all types
func TestFlagTypesStringRepresentation(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func() (unsetStr, setStr string)
	}{
		{
			name: "OptionalDuration",
			testFunc: func() (string, string) {
				var d OptionalDuration
				unset := d.String()
				d.Set("1s")
				set := d.String()
				return unset, set
			},
		},
		{
			name: "OptionalInt",
			testFunc: func() (string, string) {
				var i OptionalInt
				unset := i.String()
				i.Set("42")
				set := i.String()
				return unset, set
			},
		},
		{
			name: "OptionalString",
			testFunc: func() (string, string) {
				var s OptionalString
				unset := s.String()
				s.Set("test")
				set := s.String()
				return unset, set
			},
		},
		{
			name: "OptionalBool",
			testFunc: func() (string, string) {
				var b OptionalBool
				unset := b.String()
				b.Set("true")
				set := b.String()
				return unset, set
			},
		},
		{
			name: "OptionalMetricsMode",
			testFunc: func() (string, string) {
				var m OptionalMetricsMode
				unset := m.String()
				m.Set("per-target")
				set := m.String()
				return unset, set
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unsetStr, setStr := tt.testFunc()
			if unsetStr != "" {
				t.Fatalf("expected empty string for unset %s, got %q", tt.name, unsetStr)
			}
			if setStr == "" {
				t.Fatalf("expected non-empty string for set %s", tt.name)
			}
		})
	}
}
