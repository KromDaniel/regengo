package compiler

import (
	"bytes"
	"regexp/syntax"
	"testing"
)

func TestDetectNestedQuantifiers(t *testing.T) {
	tests := []struct {
		pattern     string
		hasNested   bool
		description string
	}{
		// Patterns WITH nested quantifiers (catastrophic backtracking risk)
		{`(a+)+`, true, "plus inside plus"},
		{`(a+)+b`, true, "plus inside plus with suffix"},
		{`(a*)*`, true, "star inside star"},
		{`(a*)*b`, true, "star inside star with suffix"},
		{`(a?)+`, true, "optional inside plus"},
		{`(a+)*`, true, "plus inside star"},
		{`(a{2,})+`, true, "repeat inside plus"},
		{`((a+)+)`, true, "nested groups with nested quantifiers"},
		{`(a|b+)+`, true, "alternation with nested quantifiers"},
		{`(x+x+)+y`, true, "multiple plus with outer plus"},

		// Patterns WITHOUT nested quantifiers
		{`a+b`, false, "simple plus"},
		{`a*b`, false, "simple star"},
		{`(a+)b`, false, "capture with plus, no nesting"},
		{`(ab)+`, false, "capture repeated, no nested quantifier"},
		{`a+b+c+`, false, "sequential quantifiers"},
		{`(a)(b)+`, false, "capture followed by repeated capture"},
		{`\d{4}-\d{2}-\d{2}`, false, "date pattern"},
		{`\w+@\w+\.\w+`, false, "simple email pattern"},
		{`(foo|bar)+`, false, "alternation repeated, no nested quantifier"},
		{`(?:a|b)+`, false, "non-capturing group repeated"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			re, err := syntax.Parse(tt.pattern, syntax.Perl)
			if err != nil {
				t.Fatalf("failed to parse pattern %q: %v", tt.pattern, err)
			}

			hasNested := detectNestedQuantifiers(re)
			if hasNested != tt.hasNested {
				t.Errorf("pattern %q: detectNestedQuantifiers = %v, want %v",
					tt.pattern, hasNested, tt.hasNested)
			}
		})
	}
}

func TestAnalyzeComplexity(t *testing.T) {
	tests := []struct {
		pattern              string
		wantCatastrophicRisk bool
		wantUseThompson      bool
		description          string
	}{
		// Simple patterns - no Thompson needed
		{`abc`, false, false, "literal string"},
		{`a+b`, false, false, "simple quantifier"},
		{`\d{4}-\d{2}-\d{2}`, false, false, "date pattern"},

		// Catastrophic patterns - Thompson recommended
		{`(a+)+b`, true, true, "nested plus"},
		{`(a*)*b`, true, true, "nested star"},
		{`(a|a)+b`, false, false, "ambiguous alternation (no nested quantifier)"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			re, err := syntax.Parse(tt.pattern, syntax.Perl)
			if err != nil {
				t.Fatalf("failed to parse pattern %q: %v", tt.pattern, err)
			}

			re = re.Simplify()
			prog, err := syntax.Compile(re)
			if err != nil {
				t.Fatalf("failed to compile pattern %q: %v", tt.pattern, err)
			}

			analysis := analyzeComplexity(prog, re)

			if analysis.HasCatastrophicRisk != tt.wantCatastrophicRisk {
				t.Errorf("pattern %q: HasCatastrophicRisk = %v, want %v",
					tt.pattern, analysis.HasCatastrophicRisk, tt.wantCatastrophicRisk)
			}

			if analysis.UseThompsonNFA != tt.wantUseThompson {
				t.Errorf("pattern %q: UseThompsonNFA = %v, want %v",
					tt.pattern, analysis.UseThompsonNFA, tt.wantUseThompson)
			}
		})
	}
}

func TestComputeEpsilonClosures(t *testing.T) {
	tests := []struct {
		pattern     string
		description string
	}{
		{`abc`, "simple literal"},
		{`a|b`, "simple alternation"},
		{`a+`, "simple plus"},
		{`(a)`, "simple capture"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			re, err := syntax.Parse(tt.pattern, syntax.Perl)
			if err != nil {
				t.Fatalf("failed to parse pattern %q: %v", tt.pattern, err)
			}

			re = re.Simplify()
			prog, err := syntax.Compile(re)
			if err != nil {
				t.Fatalf("failed to compile pattern %q: %v", tt.pattern, err)
			}

			closures := computeEpsilonClosures(prog)

			// Basic sanity checks
			if len(closures) != len(prog.Inst) {
				t.Errorf("closures length = %d, want %d", len(closures), len(prog.Inst))
			}

			// Each state should at least include itself in its closure
			for i := 0; i < len(prog.Inst) && i < 64; i++ {
				if closures[i]&(1<<i) == 0 {
					t.Errorf("state %d not in its own epsilon closure", i)
				}
			}
		})
	}
}

func TestLogger(t *testing.T) {
	t.Run("disabled logger produces no output", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewLogger(false)
		logger.SetOutput(&buf)

		logger.Log("test message")
		logger.Section("test section")

		if buf.Len() != 0 {
			t.Errorf("disabled logger produced output: %s", buf.String())
		}
	})

	t.Run("enabled logger produces output", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewLogger(true)
		logger.SetOutput(&buf)

		logger.Log("test message")
		logger.Section("test section")

		output := buf.String()
		if output == "" {
			t.Error("enabled logger produced no output")
		}
		if !bytes.Contains([]byte(output), []byte("test message")) {
			t.Errorf("output missing 'test message': %s", output)
		}
		if !bytes.Contains([]byte(output), []byte("test section")) {
			t.Errorf("output missing 'test section': %s", output)
		}
	})
}

func TestCompilerVerboseLogging(t *testing.T) {
	// Test that verbose mode produces expected log output
	t.Run("verbose mode logs analysis", func(t *testing.T) {
		re, err := syntax.Parse(`(a+)+b`, syntax.Perl)
		if err != nil {
			t.Fatalf("failed to parse pattern: %v", err)
		}
		re = re.Simplify()
		prog, err := syntax.Compile(re)
		if err != nil {
			t.Fatalf("failed to compile pattern: %v", err)
		}

		var buf bytes.Buffer
		config := Config{
			Pattern:  `(a+)+b`,
			Name:     "Test",
			Package:  "test",
			Program:  prog,
			RegexAST: re,
			Verbose:  true,
		}

		compiler := New(config)
		compiler.logger.SetOutput(&buf)

		// Re-run analysis to capture output
		compiler.analyzeAndLog()

		output := buf.String()
		if !bytes.Contains([]byte(output), []byte("Pattern Analysis")) {
			t.Errorf("missing Pattern Analysis section in verbose output")
		}
		if !bytes.Contains([]byte(output), []byte("Engine Selection")) {
			t.Errorf("missing Engine Selection section in verbose output")
		}
	})
}
