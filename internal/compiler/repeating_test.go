package compiler

import (
	"regexp"
	"regexp/syntax"
	"testing"
)

func TestRepeatingCaptureDetection(t *testing.T) {
	tests := []struct {
		pattern      string
		hasRepeating bool
		description  string
	}{
		{`(\w)*`, true, "star quantifier on capture"},
		{`(\w)+`, true, "plus quantifier on capture"},
		{`(\w)?`, true, "optional capture"},
		{`(\w){3}`, true, "fixed repeat on capture"},
		{`(\w){2,5}`, true, "range repeat on capture"},
		{`(?P<word>\w)+`, true, "named repeating capture"},
		{`(\w)(\d)+`, true, "one normal, one repeating"},
		{`(\w+)`, false, "non-repeating capture with repeating content"},
		{`((\w)+)`, true, "nested repeating capture"},
		{`(\w)(\d)`, false, "two normal captures"},
		{`a(\w)b`, false, "single capture no repeat"},
		{`(?:(\w)+)`, true, "repeating capture in non-capturing group"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			re, err := syntax.Parse(tt.pattern, syntax.Perl)
			if err != nil {
				t.Fatalf("failed to parse pattern %q: %v", tt.pattern, err)
			}

			hasRepeating := hasRepeatingCaptures(re)
			if hasRepeating != tt.hasRepeating {
				t.Errorf("pattern %q: hasRepeatingCaptures = %v, want %v",
					tt.pattern, hasRepeating, tt.hasRepeating)
			}
		})
	}
}

func TestRepeatingCapturesBehavior(t *testing.T) {
	// Test that stdlib behavior matches our expectations
	tests := []struct {
		pattern      string
		input        string
		wantFull     string
		wantCaptures []string
		description  string
	}{
		{
			pattern:      `(\w)*`,
			input:        "abc",
			wantFull:     "abc",
			wantCaptures: []string{"c"}, // Only last match
			description:  "star quantifier captures last",
		},
		{
			pattern:      `(\w)+`,
			input:        "abc",
			wantFull:     "abc",
			wantCaptures: []string{"c"}, // Only last match
			description:  "plus quantifier captures last",
		},
		{
			pattern:      `(?P<word>\w)+`,
			input:        "abc",
			wantFull:     "abc",
			wantCaptures: []string{"c"}, // Only last match
			description:  "named repeating captures last",
		},
		{
			pattern:      `(\w)(\d)+`,
			input:        "a123",
			wantFull:     "a123",
			wantCaptures: []string{"a", "3"}, // First normal, second last
			description:  "mixed normal and repeating",
		},
		{
			pattern:      `(\w+)`,
			input:        "abc",
			wantFull:     "abc",
			wantCaptures: []string{"abc"}, // Not repeating, captures all
			description:  "non-repeating capture",
		},
		{
			pattern:      `((\w)+)`,
			input:        "abc",
			wantFull:     "abc",
			wantCaptures: []string{"abc", "c"}, // Outer captures all, inner captures last
			description:  "nested captures",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			re := regexp.MustCompile(tt.pattern)
			matches := re.FindStringSubmatch(tt.input)

			if len(matches) == 0 {
				t.Fatalf("no matches found for pattern %q on input %q", tt.pattern, tt.input)
			}

			if matches[0] != tt.wantFull {
				t.Errorf("full match: got %q, want %q", matches[0], tt.wantFull)
			}

			if len(matches)-1 != len(tt.wantCaptures) {
				t.Fatalf("capture count: got %d, want %d", len(matches)-1, len(tt.wantCaptures))
			}

			for i, want := range tt.wantCaptures {
				if matches[i+1] != want {
					t.Errorf("capture[%d]: got %q, want %q", i, matches[i+1], want)
				}
			}
		})
	}
}

// Note: hasRepeatingCaptures and walkCheckRepeating are now defined in compiler.go
