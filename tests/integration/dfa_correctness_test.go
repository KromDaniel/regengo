package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/KromDaniel/regengo"
)

// Test patterns that should use DFA (ASCII-only, no dot, no word boundary)
var dfaTestCases = []struct {
	name    string
	pattern string
	inputs  []string
}{
	{
		name:    "simple_literal",
		pattern: "hello",
		inputs:  []string{"hello", "world", "hello world", "say hello please", "HELLO", ""},
	},
	{
		name:    "char_class_lower",
		pattern: "[a-z]+",
		inputs:  []string{"abc", "ABC", "abc123", "123", "", "hello"},
	},
	{
		name:    "char_class_digits",
		pattern: "[0-9]+",
		inputs:  []string{"123", "abc", "abc123", "", "42"},
	},
	{
		name:    "alternation",
		pattern: "cat|dog|fish",
		inputs:  []string{"cat", "dog", "fish", "bird", "catfish", ""},
	},
	{
		name:    "anchored_start",
		pattern: "^hello",
		inputs:  []string{"hello world", "say hello", "hello", "HELLO", ""},
	},
	{
		name:    "anchored_end",
		pattern: "world$",
		inputs:  []string{"hello world", "world tour", "world", "WORLD", ""},
	},
	{
		name:    "anchored_both",
		pattern: "^exact$",
		inputs:  []string{"exact", "not exact", "exactness", "", "exact\n"},
	},
	{
		name:    "quantifiers",
		pattern: "a+b*c?",
		inputs:  []string{"a", "ab", "abc", "aabbcc", "bc", "aaaa", ""},
	},
	{
		name:    "nested_quantifiers",
		pattern: "(ab)+",
		inputs:  []string{"ab", "abab", "ababab", "a", "b", "abc", ""},
	},
	{
		name:    "complex_alternation",
		pattern: "(one|two|three)(a|b|c)",
		inputs:  []string{"onea", "twob", "threec", "one", "a", "foura", ""},
	},
}

func TestDFACorrectnessVsStdlib(t *testing.T) {
	for _, tc := range dfaTestCases {
		t.Run(tc.name, func(t *testing.T) {
			// Compile stdlib regex for comparison
			stdRe := regexp.MustCompile(tc.pattern)

			// Create temp directory for generated code
			tmpDir := t.TempDir()
			outputFile := filepath.Join(tmpDir, "regex.go")

			// Generate with ForceDFA option
			opts := regengo.Options{
				Pattern:    tc.pattern,
				Name:       "TestRegex",
				OutputFile: outputFile,
				Package:    "main",
				ForceDFA:   true,
			}

			if err := regengo.Compile(opts); err != nil {
				t.Fatalf("compilation failed: %v", err)
			}

			// Create test program that compares results
			var testCases strings.Builder
			for _, input := range tc.inputs {
				expected := stdRe.MatchString(input)
				testCases.WriteString("\t{`")
				testCases.WriteString(strings.ReplaceAll(input, "`", "` + \"`\" + `"))
				testCases.WriteString("`, ")
				if expected {
					testCases.WriteString("true")
				} else {
					testCases.WriteString("false")
				}
				testCases.WriteString("},\n")
			}

			testProgram := `package main

import (
	"fmt"
	"os"
)

func main() {
	testCases := []struct {
		input    string
		expected bool
	}{
` + testCases.String() + `	}

	failed := false
	for _, tc := range testCases {
		got := CompiledTestRegex.MatchString(tc.input)
		if got != tc.expected {
			fmt.Printf("FAIL: input=%q expected=%v got=%v\n", tc.input, tc.expected, got)
			failed = true
		}
	}

	if failed {
		os.Exit(1)
	}
	fmt.Println("PASS")
}
`

			testFile := filepath.Join(tmpDir, "main.go")
			if err := os.WriteFile(testFile, []byte(testProgram), 0644); err != nil {
				t.Fatalf("failed to write test program: %v", err)
			}

			// Get project root
			wd, _ := os.Getwd()
			projectRoot := filepath.Join(wd, "..", "..")

			// Create go.mod
			goMod := `module test

go 1.21

require github.com/KromDaniel/regengo v0.0.0

replace github.com/KromDaniel/regengo => ` + projectRoot + `
`
			if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
				t.Fatalf("failed to write go.mod: %v", err)
			}

			// Run go mod tidy first
			tidyCmd := exec.Command("go", "mod", "tidy")
			tidyCmd.Dir = tmpDir
			if output, err := tidyCmd.CombinedOutput(); err != nil {
				t.Fatalf("go mod tidy failed: %v\nOutput:\n%s", err, output)
			}

			// Run the test program
			cmd := exec.Command("go", "run", ".")
			cmd.Dir = tmpDir
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("test program failed: %v\nOutput:\n%s", err, output)
			}

			if !strings.Contains(string(output), "PASS") {
				t.Fatalf("unexpected output:\n%s", output)
			}
		})
	}
}

// Test patterns that should NOT use DFA (should fall back)
var dfaFallbackCases = []struct {
	name    string
	pattern string
	reason  string
}{
	{
		name:    "dot",
		pattern: "a.b",
		reason:  "dot matches non-ASCII",
	},
	{
		name:    "word_boundary",
		pattern: `\bword\b`,
		reason:  "word boundary not supported",
	},
	{
		name:    "non_ascii_char",
		pattern: "caf\u00e9",
		reason:  "non-ASCII character",
	},
	{
		name:    "unicode_char_class",
		pattern: `\p{L}+`,
		reason:  "Unicode character class",
	},
}

func TestDFAFallback(t *testing.T) {
	for _, tc := range dfaFallbackCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := regengo.Analyze(tc.pattern)
			if err != nil {
				t.Fatalf("analysis failed: %v", err)
			}

			// Verify DFA is NOT in engine labels
			for _, label := range result.EngineLabels {
				if label == "DFA" {
					t.Errorf("DFA should not be used for pattern %q (%s), got labels: %v",
						tc.pattern, tc.reason, result.EngineLabels)
				}
			}
		})
	}
}

// Test DFA vs Thompson equivalence
// This ensures our internal engines agree with each other
// Note: Anchored patterns (^, $) are not included because Thompson NFA
// has a known limitation with handling EmptyWidth instructions.
var dfaThompsonPatterns = []struct {
	name    string
	pattern string
	inputs  []string
}{
	{
		name:    "simple_literal",
		pattern: "hello",
		inputs:  []string{"hello", "world", "hello world", "HELLO", ""},
	},
	{
		name:    "char_class",
		pattern: "[a-z]+",
		inputs:  []string{"abc", "ABC", "abc123", "test"},
	},
	{
		name:    "alternation",
		pattern: "cat|dog|fish",
		inputs:  []string{"cat", "dog", "fish", "bird", "catfish"},
	},
	{
		name:    "quantifiers",
		pattern: "a+b*",
		inputs:  []string{"a", "ab", "aabb", "b", "aaaa"},
	},
}

func TestDFAvsThompsonEquivalence(t *testing.T) {
	for _, tc := range dfaThompsonPatterns {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Generate DFA version
			dfaFile := filepath.Join(tmpDir, "dfa_regex.go")
			dfaOpts := regengo.Options{
				Pattern:    tc.pattern,
				Name:       "DFARegex",
				OutputFile: dfaFile,
				Package:    "main",
				ForceDFA:   true,
			}
			if err := regengo.Compile(dfaOpts); err != nil {
				t.Fatalf("DFA compilation failed: %v", err)
			}

			// Generate Thompson version
			thompsonFile := filepath.Join(tmpDir, "thompson_regex.go")
			thompsonOpts := regengo.Options{
				Pattern:       tc.pattern,
				Name:          "ThompsonRegex",
				OutputFile:    thompsonFile,
				Package:       "main",
				ForceThompson: true,
			}
			if err := regengo.Compile(thompsonOpts); err != nil {
				t.Fatalf("Thompson compilation failed: %v", err)
			}

			// Create test program comparing both
			var inputsCode strings.Builder
			for _, input := range tc.inputs {
				inputsCode.WriteString("\t\t`")
				inputsCode.WriteString(strings.ReplaceAll(input, "`", "` + \"`\" + `"))
				inputsCode.WriteString("`,\n")
			}

			testProgram := `package main

import (
	"fmt"
	"os"
)

func main() {
	inputs := []string{
` + inputsCode.String() + `	}

	failed := false
	for _, input := range inputs {
		dfaResult := CompiledDFARegex.MatchString(input)
		thompsonResult := CompiledThompsonRegex.MatchString(input)
		if dfaResult != thompsonResult {
			fmt.Printf("MISMATCH: input=%q DFA=%v Thompson=%v\n", input, dfaResult, thompsonResult)
			failed = true
		}
	}

	if failed {
		os.Exit(1)
	}
	fmt.Println("PASS: DFA and Thompson produce identical results")
}
`

			testFile := filepath.Join(tmpDir, "main.go")
			if err := os.WriteFile(testFile, []byte(testProgram), 0644); err != nil {
				t.Fatalf("failed to write test program: %v", err)
			}

			// Get project root
			wd, _ := os.Getwd()
			projectRoot := filepath.Join(wd, "..", "..")

			// Create go.mod
			goMod := `module test

go 1.21

require github.com/KromDaniel/regengo v0.0.0

replace github.com/KromDaniel/regengo => ` + projectRoot + `
`
			if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
				t.Fatalf("failed to write go.mod: %v", err)
			}

			// Run go mod tidy
			tidyCmd := exec.Command("go", "mod", "tidy")
			tidyCmd.Dir = tmpDir
			if output, err := tidyCmd.CombinedOutput(); err != nil {
				t.Fatalf("go mod tidy failed: %v\nOutput:\n%s", err, output)
			}

			// Run the test program
			cmd := exec.Command("go", "run", ".")
			cmd.Dir = tmpDir
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("test program failed: %v\nOutput:\n%s", err, output)
			}

			if !strings.Contains(string(output), "PASS") {
				t.Fatalf("unexpected output:\n%s", output)
			}
		})
	}
}

// Test that ForceDFA falls back gracefully when DFA is not possible
func TestForceDFAFallback(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "regex.go")

	// Pattern with dot - can't use DFA
	opts := regengo.Options{
		Pattern:    "a.b",
		Name:       "DotRegex",
		OutputFile: outputFile,
		Package:    "main",
		ForceDFA:   true, // Force DFA, but pattern is incompatible
	}

	// Should compile without error (falls back to Thompson)
	if err := regengo.Compile(opts); err != nil {
		t.Fatalf("compilation should succeed with fallback: %v", err)
	}

	// Verify file was generated
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatal("output file was not generated")
	}
}
