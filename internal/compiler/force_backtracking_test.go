package compiler

import (
	"os"
	"path/filepath"
	"regexp/syntax"
	"strings"
	"testing"
)

// newForceBacktrackingCompiler builds a compiler for pattern with the given
// ForceBacktracking setting.
func newForceBacktrackingCompiler(t *testing.T, pattern string, force bool) *Compiler {
	t.Helper()

	regexAST, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	regexAST = regexAST.Simplify()
	prog, err := syntax.Compile(regexAST)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	return New(Config{
		Pattern:           pattern,
		Name:              "Test",
		Package:           "test",
		Program:           prog,
		RegexAST:          regexAST,
		WithCaptures:      prog.NumCap > ImplicitCaptureCount,
		ForceBacktracking: force,
	})
}

// funcBody returns the text of the generated method that starts at the line
// beginning with header, up to the first closing "\n}\n".
func funcBody(t *testing.T, src, header string) string {
	t.Helper()

	start := strings.Index(src, header)
	if start < 0 {
		t.Fatalf("function %q not found in generated code", header)
	}
	rest := src[start:]
	end := strings.Index(rest, "\n}\n")
	if end < 0 {
		t.Fatalf("end of function %q not found in generated code", header)
	}
	return rest[:end]
}

func TestForceBacktrackingRouting(t *testing.T) {
	tests := []struct {
		name         string
		pattern      string
		wantMemo     bool
		checkTDFA    bool
		wantTDFAFind bool
	}{
		{"dfa_pattern", `0x[0-9a-fA-F]+`, false, false, false},
		{"thompson_pattern", `(?:[^,]+,?)+;`, true, false, false},
		{"captures_unchanged_find_routing", `((?:é+x?)+)`, true, true, true},
		{"word_boundary_catastrophic", `(?:\w+\s?)+\bend\b`, true, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newForceBacktrackingCompiler(t, tt.pattern, true)

			if c.useDFAForMatch {
				t.Error("useDFAForMatch = true, want false")
			}
			if c.useThompsonForMatch {
				t.Error("useThompsonForMatch = true, want false")
			}
			if c.useMemoization != tt.wantMemo {
				t.Errorf("useMemoization = %v, want %v", c.useMemoization, tt.wantMemo)
			}
			if tt.checkTDFA && c.useTDFAForCaptures != tt.wantTDFAFind {
				t.Errorf("useTDFAForCaptures = %v, want %v", c.useTDFAForCaptures, tt.wantTDFAFind)
			}
		})
	}

	generateMatchBody := func(t *testing.T, force bool) string {
		t.Helper()
		c := newForceBacktrackingCompiler(t, `0x[0-9a-fA-F]+`, force)
		out := filepath.Join(t.TempDir(), "r.go")
		c.SetOutputFile(out)
		if err := c.Generate(); err != nil {
			t.Fatalf("Generate failed: %v", err)
		}
		src, err := os.ReadFile(out)
		if err != nil {
			t.Fatalf("read generated file: %v", err)
		}
		return funcBody(t, string(src), "func (Test) MatchString(")
	}

	t.Run("generated_match_uses_backtracking", func(t *testing.T) {
		body := generateMatchBody(t, true)
		if !strings.Contains(body, "StepSelect") {
			t.Error("MatchString body does not contain StepSelect (backtracking marker)")
		}
		if strings.Contains(body, "dfaTransitions") {
			t.Error("MatchString body contains dfaTransitions with ForceBacktracking")
		}
		if strings.Contains(body, "epsilonClosures") {
			t.Error("MatchString body contains epsilonClosures with ForceBacktracking")
		}
	})

	t.Run("control_without_flag_uses_dfa", func(t *testing.T) {
		body := generateMatchBody(t, false)
		if !strings.Contains(body, "dfaTransitions") {
			t.Error("control: MatchString body does not contain dfaTransitions without the flag")
		}
	})
}
