package compiler

import (
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
