package main

import (
	"testing"
)

func TestArrayFlagsString(t *testing.T) {
	tests := []struct {
		name     string
		flags    arrayFlags
		expected string
	}{
		{
			name:     "empty",
			flags:    arrayFlags{},
			expected: "",
		},
		{
			name:     "single",
			flags:    arrayFlags{"$user@REDACTED"},
			expected: "$user@REDACTED",
		},
		{
			name:     "multiple",
			flags:    arrayFlags{"$user@REDACTED", "[$0]", "FILTERED"},
			expected: "$user@REDACTED, [$0], FILTERED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.flags.String()
			if result != tt.expected {
				t.Errorf("String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestArrayFlagsSet(t *testing.T) {
	var flags arrayFlags

	// Test adding multiple values
	if err := flags.Set("$user@REDACTED"); err != nil {
		t.Errorf("Set() returned error: %v", err)
	}
	if len(flags) != 1 || flags[0] != "$user@REDACTED" {
		t.Errorf("Set() = %v, want [\"$user@REDACTED\"]", flags)
	}

	if err := flags.Set("[$0]"); err != nil {
		t.Errorf("Set() returned error: %v", err)
	}
	if len(flags) != 2 || flags[1] != "[$0]" {
		t.Errorf("Set() = %v, want [\"$user@REDACTED\", \"[$0]\"]", flags)
	}
}
