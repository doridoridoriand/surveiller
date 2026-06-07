package config

import "testing"

func TestTargetConfigCopyDeepCopiesOptions(t *testing.T) {
	original := TargetConfig{
		Name:    "api",
		Address: "192.0.2.10",
		Group:   "edge",
		Options: map[string]string{
			"relay": "jump1",
			"user":  "alice",
		},
	}

	copied := original.Copy()
	copied.Options["relay"] = "jump2"
	copied.Options["new"] = "value"

	if original.Options["relay"] != "jump1" {
		t.Fatalf("original option was mutated: %q", original.Options["relay"])
	}
	if _, ok := original.Options["new"]; ok {
		t.Fatalf("new option leaked into original: %+v", original.Options)
	}
}

func TestTargetConfigCopyPreservesNilOptions(t *testing.T) {
	original := TargetConfig{Name: "api", Address: "192.0.2.10"}
	copied := original.Copy()

	if copied.Options != nil {
		t.Fatalf("expected nil options to remain nil, got %+v", copied.Options)
	}
}
