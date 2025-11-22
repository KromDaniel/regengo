package compiler

import (
	"os"
	"os/exec"
	"regexp"
	"testing"
)

// TestFindAllBehavior tests that FindAll behaves like stdlib
func TestFindAllBehavior(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		input   string
		n       int
	}{
		{
			name:    "multiple matches unlimited",
			pattern: `(\d+)`,
			input:   "123 456 789",
			n:       -1,
		},
		{
			name:    "limited matches",
			pattern: `(\w+)`,
			input:   "one two three four five",
			n:       3,
		},
		{
			name:    "no matches",
			pattern: `\d+`,
			input:   "no numbers here",
			n:       -1,
		},
		{
			name:    "n=0 returns nil",
			pattern: `\d+`,
			input:   "123 456",
			n:       0,
		},
		{
			name:    "single match",
			pattern: `(\w+)`,
			input:   "hello",
			n:       -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdlibRe := regexp.MustCompile(tt.pattern)
			stdlibMatches := stdlibRe.FindAllStringSubmatch(tt.input, tt.n)

			// Verify expected behavior
			if tt.n == 0 {
				if stdlibMatches != nil {
					t.Errorf("expected nil for n=0, got %d matches", len(stdlibMatches))
				}
			} else if tt.n > 0 {
				if len(stdlibMatches) > tt.n {
					t.Errorf("expected at most %d matches, got %d", tt.n, len(stdlibMatches))
				}
			}

			t.Logf("Pattern: %s, Input: %q, n: %d, Matches: %d", tt.pattern, tt.input, tt.n, len(stdlibMatches))
		})
	}
}

// TestGeneratedFindAll tests that generated code has FindAll functions
func TestGeneratedFindAll(t *testing.T) {
	// Run make bench-gen and verify output
	cmd := exec.Command("make", "bench-gen")
	cmd.Dir = "../../"
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run make bench-gen: %v\nOutput: %s", err, output)
	}

	// Check that FindAll functions exist in generated code
	dateCapturePath := "../../benchmarks/generated/DateCapture.go"
	content, err := os.ReadFile(dateCapturePath)
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}

	contentStr := string(content)

	// Verify FindAllString exists
	if !contains(contentStr, "func (r DateCapture) FindAllString(input string, n int)") {
		t.Error("DateCapture.FindAllString not found in generated code")
	}

	// Verify FindAllBytes exists
	if !contains(contentStr, "func (r DateCapture) FindAllBytes(input []byte, n int)") {
		t.Error("DateCapture.FindAllBytes not found in generated code")
	}

	t.Log("✓ FindAll methods generated successfully")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && hasSubstring(s, substr))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
