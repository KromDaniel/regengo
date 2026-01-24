package compiler

import (
	"regexp/syntax"
	"testing"
)

// mustCompilePattern compiles a pattern to a syntax.Prog for testing.
func mustCompilePattern(t *testing.T, pattern string) *syntax.Prog {
	t.Helper()
	ast, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		t.Fatalf("failed to parse pattern %q: %v", pattern, err)
	}
	ast = ast.Simplify()
	prog, err := syntax.Compile(ast)
	if err != nil {
		t.Fatalf("failed to compile pattern %q: %v", pattern, err)
	}
	return prog
}

// mustParsePattern parses a pattern to a syntax.Regexp AST for testing.
func mustParsePattern(t *testing.T, pattern string) *syntax.Regexp {
	t.Helper()
	ast, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		t.Fatalf("failed to parse pattern %q: %v", pattern, err)
	}
	return ast.Simplify()
}

func TestAnalyzePattern(t *testing.T) {
	tests := []struct {
		name          string
		pattern       string
		tdfaThreshold int
		dfaThreshold  int
		wantEngine    []string // expected engine labels
		wantFeatures  []string // expected feature labels (subset)
		wantCaptures  bool
		wantRisk      bool
		wantEndAnchor bool
		wantErr       bool
	}{
		{
			name:          "simple literal",
			pattern:       "hello",
			tdfaThreshold: 500,
			dfaThreshold:  500,
			wantEngine:    []string{"Backtracking"},
			wantFeatures:  []string{"Simple"},
			wantCaptures:  false,
			wantRisk:      false,
			wantErr:       false,
		},
		{
			name:          "catastrophic pattern uses DFA",
			pattern:       "(a+)+b",
			tdfaThreshold: 500,
			dfaThreshold:  500,
			wantEngine:    []string{"DFA", "TDFA"}, // Has capture group (a+), so uses TDFA too
			wantCaptures:  true,                    // (a+) is a capture group
			wantRisk:      true,
			wantErr:       false,
		},
		{
			name:          "catastrophic with captures uses DFA and TDFA",
			pattern:       "(?P<x>(a+)+)b",
			tdfaThreshold: 500,
			dfaThreshold:  500,
			wantEngine:    []string{"DFA", "TDFA"},
			wantCaptures:  true,
			wantRisk:      true,
			wantErr:       false,
		},
		{
			name:          "simple pattern with captures",
			pattern:       "(?P<user>\\w+)@(?P<domain>\\w+)",
			tdfaThreshold: 500,
			dfaThreshold:  500,
			wantEngine:    []string{"Backtracking"},
			wantFeatures:  []string{"Captures", "CharClass", "Quantifiers"},
			wantCaptures:  true,
			wantRisk:      false,
			wantErr:       false,
		},
		{
			name:          "pattern with end anchor and catastrophic risk uses DFA and TDFA",
			pattern:       "(a+)+$",
			tdfaThreshold: 500,
			dfaThreshold:  500,
			wantEngine:    []string{"DFA", "TDFA"}, // DFA handles match, TDFA handles capture
			wantCaptures:  true,                    // (a+) is a capture group
			wantRisk:      true,
			wantEndAnchor: true,
			wantErr:       false,
		},
		{
			name:          "word boundary pattern cannot use DFA",
			pattern:       "\\b(a+)+\\b",
			tdfaThreshold: 500,
			dfaThreshold:  500,
			wantEngine:    []string{"Thompson"}, // Falls back to Thompson since DFA doesn't support \b
			wantCaptures:  true,                 // (a+) is a capture group
			wantRisk:      true,
			wantErr:       false,
		},
		{
			name:          "alternation pattern",
			pattern:       "cat|dog|fish",
			tdfaThreshold: 500,
			dfaThreshold:  500,
			wantEngine:    []string{"Backtracking"},
			wantFeatures:  []string{"Alternation"},
			wantCaptures:  false,
			wantRisk:      false,
			wantErr:       false,
		},
		{
			name:          "multibyte pattern",
			pattern:       "héllo",
			tdfaThreshold: 500,
			dfaThreshold:  500,
			wantEngine:    []string{"Backtracking"},
			wantFeatures:  []string{"Multibyte"},
			wantCaptures:  false,
			wantRisk:      false,
			wantErr:       false,
		},
		{
			name:          "anchored pattern",
			pattern:       "^hello$",
			tdfaThreshold: 500,
			dfaThreshold:  500,
			wantEngine:    []string{"Backtracking"},
			wantFeatures:  []string{"Anchored"},
			wantCaptures:  false,
			wantRisk:      false,
			wantEndAnchor: true,
			wantErr:       false,
		},
		{
			name:          "non-capturing group",
			pattern:       "(?:abc)+",
			tdfaThreshold: 500,
			dfaThreshold:  500,
			wantEngine:    []string{"Backtracking"},
			wantFeatures:  []string{"NonCapturing", "Quantifiers"},
			wantCaptures:  false,
			wantRisk:      false,
			wantErr:       false,
		},
		{
			name:          "character class pattern",
			pattern:       "[a-z]+",
			tdfaThreshold: 500,
			dfaThreshold:  500,
			wantEngine:    []string{"Backtracking"},
			wantFeatures:  []string{"CharClass", "Quantifiers"},
			wantCaptures:  false,
			wantRisk:      false,
			wantErr:       false,
		},
		{
			name:          "dot metacharacter is char class",
			pattern:       ".*",
			tdfaThreshold: 500,
			dfaThreshold:  500,
			wantEngine:    []string{"Backtracking"},
			wantFeatures:  []string{"CharClass", "Quantifiers"},
			wantCaptures:  false,
			wantRisk:      false,
			wantErr:       false,
		},
		{
			name:          "invalid pattern",
			pattern:       "[invalid",
			tdfaThreshold: 500,
			dfaThreshold:  500,
			wantErr:       true,
		},
		{
			name:          "zero dfaThreshold uses default",
			pattern:       "(a+)+b",
			tdfaThreshold: 500,
			dfaThreshold:  0, // Should use default 500
			wantEngine:    []string{"DFA", "TDFA"},
			wantCaptures:  true, // (a+) is a capture group
			wantRisk:      true,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := AnalyzePattern(tt.pattern, tt.tdfaThreshold, tt.dfaThreshold)
			if (err != nil) != tt.wantErr {
				t.Errorf("AnalyzePattern() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if result == nil {
				t.Fatal("AnalyzePattern() returned nil result")
			}

			// Check engine labels
			for _, want := range tt.wantEngine {
				found := false
				for _, got := range result.EngineLabels {
					if got == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected engine label %q not found in %v", want, result.EngineLabels)
				}
			}

			// Check feature labels (subset)
			for _, want := range tt.wantFeatures {
				found := false
				for _, got := range result.FeatureLabels {
					if got == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected feature label %q not found in %v", want, result.FeatureLabels)
				}
			}

			// Check flags
			if result.HasCaptures != tt.wantCaptures {
				t.Errorf("HasCaptures = %v, want %v", result.HasCaptures, tt.wantCaptures)
			}
			if result.HasCatastrophicRisk != tt.wantRisk {
				t.Errorf("HasCatastrophicRisk = %v, want %v", result.HasCatastrophicRisk, tt.wantRisk)
			}
			if result.HasEndAnchor != tt.wantEndAnchor {
				t.Errorf("HasEndAnchor = %v, want %v", result.HasEndAnchor, tt.wantEndAnchor)
			}
		})
	}
}

func TestCanUseTDFAStandalone(t *testing.T) {
	tests := []struct {
		name          string
		pattern       string
		tdfaThreshold int
		want          bool
	}{
		{
			name:          "simple pattern",
			pattern:       "hello",
			tdfaThreshold: 500,
			want:          true,
		},
		{
			name:          "pattern with word boundary",
			pattern:       "\\bhello\\b",
			tdfaThreshold: 500,
			want:          false,
		},
		{
			name:          "very low threshold",
			pattern:       "(?P<a>\\w+)(?P<b>\\w+)(?P<c>\\w+)",
			tdfaThreshold: 1,
			want:          false,
		},
		{
			name:          "zero threshold uses default",
			pattern:       "hello",
			tdfaThreshold: 0,
			want:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := mustCompilePattern(t, tt.pattern)
			got := canUseTDFAStandalone(prog, tt.tdfaThreshold)
			if got != tt.want {
				t.Errorf("canUseTDFAStandalone() = %v, want %v", got, tt.want)
			}
		})
	}

	// Test nil program
	t.Run("nil program", func(t *testing.T) {
		if canUseTDFAStandalone(nil, 500) {
			t.Error("canUseTDFAStandalone(nil) should return false")
		}
	})
}

func TestCanUseDFAForPattern(t *testing.T) {
	tests := []struct {
		name         string
		pattern      string
		dfaThreshold int
		want         bool
	}{
		{
			name:         "simple ASCII pattern",
			pattern:      "hello",
			dfaThreshold: 500,
			want:         true,
		},
		{
			name:         "pattern with word boundary",
			pattern:      "\\bhello\\b",
			dfaThreshold: 500,
			want:         false,
		},
		{
			name:         "pattern with dot",
			pattern:      "a.b",
			dfaThreshold: 500,
			want:         false,
		},
		{
			name:         "multibyte pattern",
			pattern:      "héllo",
			dfaThreshold: 500,
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := mustCompilePattern(t, tt.pattern)
			got := canUseDFAForPattern(prog, tt.dfaThreshold)
			if got != tt.want {
				t.Errorf("canUseDFAForPattern() = %v, want %v", got, tt.want)
			}
		})
	}

	// Test nil program
	t.Run("nil program", func(t *testing.T) {
		if canUseDFAForPattern(nil, 500) {
			t.Error("canUseDFAForPattern(nil) should return false")
		}
	})
}

func TestHasAlternation(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    bool
	}{
		{"no alternation", "hello", false},
		{"simple alternation", "cat|dog", true},
		{"nested alternation", "(a|b)+", false}, // Simplify() converts [ab]+ which is OpCharClass, not OpAlternate
		{"real nested alternation", "(cat|dog)+", true},
		{"no alternation with groups", "(abc)+", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast := mustParsePattern(t, tt.pattern)
			if got := hasAlternation(ast); got != tt.want {
				t.Errorf("hasAlternation() = %v, want %v", got, tt.want)
			}
		})
	}

	// Test nil AST
	t.Run("nil AST", func(t *testing.T) {
		if hasAlternation(nil) {
			t.Error("hasAlternation(nil) should return false")
		}
	})
}

func TestHasCharClass(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    bool
	}{
		{"no char class", "hello", false},
		{"bracket char class", "[a-z]", true},
		{"digit escape", "\\d+", true},
		{"word escape", "\\w+", true},
		{"space escape", "\\s+", true},
		{"negated digit", "\\D+", true},
		{"negated word", "\\W+", true},
		{"negated space", "\\S+", true},
		{"dot metachar", "a.b", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast := mustParsePattern(t, tt.pattern)
			if got := hasCharClass(tt.pattern, ast); got != tt.want {
				t.Errorf("hasCharClass() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasCharClassInAST(t *testing.T) {
	// Test nil AST
	t.Run("nil AST", func(t *testing.T) {
		if hasCharClassInAST(nil) {
			t.Error("hasCharClassInAST(nil) should return false")
		}
	})
}

func TestHasMultibyte(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    bool
	}{
		{"ASCII only", "hello", false},
		{"accented char", "héllo", true},
		{"emoji", "hello😀", true},
		{"unicode escape", "\\x{1F600}", false}, // This is just ASCII chars in the pattern string
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasMultibyte(tt.pattern); got != tt.want {
				t.Errorf("hasMultibyte() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasQuantifiers(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    bool
	}{
		{"no quantifier", "hello", false},
		{"star", "a*", true},
		{"plus", "a+", true},
		{"question", "a?", true},
		{"repeat", "a{2,3}", true},
		{"nested quantifier", "(a+)+", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast := mustParsePattern(t, tt.pattern)
			if got := hasQuantifiers(ast); got != tt.want {
				t.Errorf("hasQuantifiers() = %v, want %v", got, tt.want)
			}
		})
	}

	// Test nil AST
	t.Run("nil AST", func(t *testing.T) {
		if hasQuantifiers(nil) {
			t.Error("hasQuantifiers(nil) should return false")
		}
	})
}
