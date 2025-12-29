package cli

import (
	"testing"
	"time"
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
