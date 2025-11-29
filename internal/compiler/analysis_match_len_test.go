package compiler

import (
	"regexp/syntax"
	"testing"
)

func TestMinMatchLen(t *testing.T) {
	tests := []struct {
		pattern string
		want    int
	}{
		// Literals
		{"abc", 3},
		{"a", 1},
		{"", 0},
		{"hello world", 11},

		// Unicode literals
		{"日本", 6},   // 3 bytes per character
		{"café", 5}, // 4 ASCII + 1 two-byte char

		// Character classes
		{"[a-z]", 1},
		{"[0-9]", 1},
		{"\\d", 1},
		{"\\w", 1},

		// Quantifiers
		{"a*", 0},     // Zero or more
		{"a+", 1},     // One or more
		{"a?", 0},     // Zero or one
		{"a{3}", 3},   // Exactly 3
		{"a{2,5}", 2}, // 2 to 5
		{"a{0,5}", 0}, // 0 to 5
		{"a{3,}", 3},  // 3 or more

		// Concatenation
		{"abc", 3},
		{"a.b", 3},       // dot is at least 1 byte
		{"\\d\\d\\d", 3}, // Three digits

		// Alternation
		{"a|bc", 1},  // min of "a" (1) and "bc" (2)
		{"abc|d", 1}, // min of "abc" (3) and "d" (1)
		{"a|b|c", 1},

		// Groups
		{"(abc)", 3},
		{"(a)(b)", 2},
		{"(a|bc)", 1},

		// Complex patterns
		{"\\d{4}-\\d{2}-\\d{2}", 10},  // Date pattern
		{"[a-z]+@[a-z]+\\.[a-z]+", 5}, // Email-ish (a@b.c = 5 chars minimum)

		// Anchors (zero-width)
		{"^abc", 3},
		{"abc$", 3},
		{"^abc$", 3},
		{"\\b\\w+\\b", 1}, // Word boundary is zero-width

		// Named captures
		{"(?P<year>\\d{4})", 4},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			re, err := syntax.Parse(tt.pattern, syntax.Perl)
			if err != nil {
				t.Fatalf("failed to parse pattern: %v", err)
			}
			re = re.Simplify()

			got := minMatchLen(re)
			if got != tt.want {
				t.Errorf("minMatchLen(%q) = %d, want %d", tt.pattern, got, tt.want)
			}
		})
	}
}

func TestMaxMatchLen(t *testing.T) {
	tests := []struct {
		pattern string
		want    int // -1 means unbounded
	}{
		// Literals
		{"abc", 3},
		{"a", 1},
		{"", 0},

		// Unicode literals
		{"日本", 6},

		// Character classes (max UTF-8 byte length in class)
		{"[a-z]", 1},
		{"[a-z日]", 3}, // 日 is 3 bytes

		// Unbounded quantifiers
		{"a*", -1},
		{"a+", -1},
		{".+", -1},
		{".*", -1},

		// Bounded quantifiers
		{"a?", 1},
		{"a{3}", 3},
		{"a{2,5}", 5},
		{"a{0,5}", 5},
		{"a{3,}", -1}, // Unbounded

		// Concatenation
		{"abc", 3},
		{"a.b", 6}, // dot can be up to 4 bytes, so a(4)b = 6

		// Alternation
		{"a|bc", 2},  // max of "a" (1) and "bc" (2)
		{"abc|d", 3}, // max of "abc" (3) and "d" (1)
		{"a|b|cde", 3},

		// Groups
		{"(abc)", 3},
		{"(a)(b)", 2},

		// Complex patterns - bounded
		{"\\d{4}-\\d{2}-\\d{2}", 10}, // Date pattern (fixed)

		// Complex patterns - unbounded
		{"[a-z]+@[a-z]+\\.[a-z]+", -1}, // Email has + so unbounded

		// Anchors (zero-width)
		{"^abc", 3},
		{"abc$", 3},
		{"\\bword\\b", 4},

		// Named captures
		{"(?P<year>\\d{4})", 4},
		{"(?P<name>\\w+)", -1}, // Unbounded due to +
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			re, err := syntax.Parse(tt.pattern, syntax.Perl)
			if err != nil {
				t.Fatalf("failed to parse pattern: %v", err)
			}
			re = re.Simplify()

			got := maxMatchLen(re)
			if got != tt.want {
				t.Errorf("maxMatchLen(%q) = %d, want %d", tt.pattern, got, tt.want)
			}
		})
	}
}

func TestAnalyzeMatchLength(t *testing.T) {
	tests := []struct {
		pattern string
		wantMin int
		wantMax int
	}{
		{"\\d{4}-\\d{2}-\\d{2}", 10, 10},
		{"[a-z]+", 1, -1},
		{"a{2,5}", 2, 5},
		{"(a|bb|ccc)", 1, 3},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			re, err := syntax.Parse(tt.pattern, syntax.Perl)
			if err != nil {
				t.Fatalf("failed to parse pattern: %v", err)
			}
			re = re.Simplify()

			analysis := AnalyzeMatchLength(re)
			if analysis.MinMatchLen != tt.wantMin {
				t.Errorf("MinMatchLen = %d, want %d", analysis.MinMatchLen, tt.wantMin)
			}
			if analysis.MaxMatchLen != tt.wantMax {
				t.Errorf("MaxMatchLen = %d, want %d", analysis.MaxMatchLen, tt.wantMax)
			}
		})
	}
}

func TestDefaultMaxLeftover(t *testing.T) {
	tests := []struct {
		name     string
		analysis MatchLengthAnalysis
		want     int
	}{
		{
			name:     "bounded small",
			analysis: MatchLengthAnalysis{MaxMatchLen: 10},
			want:     1024, // Minimum 1KB
		},
		{
			name:     "bounded medium",
			analysis: MatchLengthAnalysis{MaxMatchLen: 500},
			want:     5000, // 10 * 500
		},
		{
			name:     "bounded large",
			analysis: MatchLengthAnalysis{MaxMatchLen: 200000},
			want:     1 << 20, // Cap at 1MB
		},
		{
			name:     "unbounded",
			analysis: MatchLengthAnalysis{MaxMatchLen: -1},
			want:     1 << 20, // 1MB default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DefaultMaxLeftover(tt.analysis)
			if got != tt.want {
				t.Errorf("DefaultMaxLeftover() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMinBufferSize(t *testing.T) {
	tests := []struct {
		name     string
		analysis MatchLengthAnalysis
		want     int
	}{
		{
			name:     "small pattern",
			analysis: MatchLengthAnalysis{MaxMatchLen: 10},
			want:     64 * 1024, // Minimum 64KB
		},
		{
			name:     "large pattern",
			analysis: MatchLengthAnalysis{MaxMatchLen: 50000},
			want:     100000, // 2 * 50000
		},
		{
			name:     "unbounded",
			analysis: MatchLengthAnalysis{MaxMatchLen: -1},
			want:     64 * 1024, // 64KB default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MinBufferSize(tt.analysis)
			if got != tt.want {
				t.Errorf("MinBufferSize() = %d, want %d", got, tt.want)
			}
		})
	}
}
