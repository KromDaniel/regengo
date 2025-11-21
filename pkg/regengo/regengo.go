// Package regengo provides regex-to-Go code generation functionality.
// It compiles regular expressions into optimized Go functions at build time.
package regengo

import (
	"fmt"
	"regexp/syntax"

	"github.com/KromDaniel/regengo/internal/compiler"
)

// Options configures the regex compilation process.
type Options struct {
	// Pattern is the regular expression to compile
	Pattern string

	// Name is the prefix for generated function names (e.g., "Email" generates "EmailMatchString")
	Name string

	// OutputFile is the path where generated code will be written
	OutputFile string

	// Package is the Go package name for the generated code
	Package string

	// NoPool disables sync.Pool for stack reuse (pool is enabled by default for better performance)
	NoPool bool

	// GenerateTestFile generates a test file with tests and benchmarks against standard regexp (default: true if TestFileInputs provided)
	GenerateTestFile bool

	// TestFileInputs is a list of test inputs for the generated test file. If empty and GenerateTestFile is true, defaults to []string{"example"}
	TestFileInputs []string
}

// Validate checks if the options are valid.
func (o Options) Validate() error {
	if o.Pattern == "" {
		return fmt.Errorf("pattern cannot be empty")
	}
	if o.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if o.OutputFile == "" {
		return fmt.Errorf("output file cannot be empty")
	}
	if o.Package == "" {
		return fmt.Errorf("package cannot be empty")
	}
	return nil
}

// Compile generates optimized Go code for the given regex pattern.
// It returns an error if the pattern is invalid or code generation fails.
func Compile(opts Options) error {
	if err := opts.Validate(); err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	// Parse the regex pattern
	regexAST, err := syntax.Parse(opts.Pattern, syntax.Perl)
	if err != nil {
		return fmt.Errorf("failed to parse pattern: %w", err)
	}

	// Simplify the regex syntax tree
	regexAST = regexAST.Simplify()

	// Optimization #2: Unroll small bounded repetitions ({2}, {3})
	regexAST = unrollSmallRepetitions(regexAST)

	// Compile to instruction program
	prog, err := syntax.Compile(regexAST)
	if err != nil {
		return fmt.Errorf("failed to compile pattern: %w", err)
	}

	// Auto-detect capture groups
	hasCaptures := prog.NumCap > 2 // NumCap > 2 means there are user-defined capture groups (0 and 1 are full match)

	// Set default for GenerateTestFile
	generateTestFile := opts.GenerateTestFile
	testInputs := opts.TestFileInputs
	if len(testInputs) > 0 {
		// If TestFileInputs provided, enable GenerateTestFile by default
		if !opts.GenerateTestFile {
			generateTestFile = true
		}
	} else if !opts.GenerateTestFile {
		// No test inputs and test file not explicitly requested - don't generate
		generateTestFile = false
	} else {
		// Test file explicitly requested but no inputs - use default
		testInputs = []string{"example"}
	}

	// Generate Go code
	config := compiler.Config{
		Pattern:          opts.Pattern,
		Program:          prog,
		Name:             opts.Name,
		Package:          opts.Package,
		UsePool:          !opts.NoPool, // Invert: NoPool flag disables pool
		WithCaptures:     hasCaptures,
		RegexAST:         regexAST,
		GenerateTestFile: generateTestFile,
		TestFileInputs:   testInputs,
	}

	c := compiler.NewCompiler(config)
	c.SetOutputFile(opts.OutputFile)

	if err := c.Generate(); err != nil {
		return fmt.Errorf("failed to generate code: %w", err)
	}

	return nil
}

// unrollSmallRepetitions transforms small bounded repetitions into explicit sequences.
// Optimization #2: Converts {2}, {3} into repeated sequences to reduce goto complexity.
// Example: (?:foo|bar){2} becomes (?:foo|bar)(?:foo|bar)
func unrollSmallRepetitions(re *syntax.Regexp) *syntax.Regexp {
	if re == nil {
		return nil
	}

	// Recursively process children first
	for i, sub := range re.Sub {
		re.Sub[i] = unrollSmallRepetitions(sub)
	}

	// Check if this is a small bounded repetition that should be unrolled
	if re.Op == syntax.OpRepeat {
		// Only unroll if Min == Max and it's a small number (2 or 3)
		if re.Min == re.Max && re.Min >= 2 && re.Min <= 3 {
			// Only unroll if the sub-expression is reasonably simple
			// (not too complex to avoid code explosion)
			if shouldUnrollExpression(re.Sub[0]) {
				// Create a concatenation of N copies
				copies := make([]*syntax.Regexp, re.Min)
				for i := 0; i < re.Min; i++ {
					// Deep copy the sub-expression
					copies[i] = copyRegexp(re.Sub[0])
				}

				// Return a concatenation node
				return &syntax.Regexp{
					Op:    syntax.OpConcat,
					Sub:   copies,
					Flags: re.Flags,
				}
			}
		}
	}

	return re
}

// shouldUnrollExpression determines if an expression is simple enough to unroll.
// We avoid unrolling very complex expressions to prevent code bloat.
func shouldUnrollExpression(re *syntax.Regexp) bool {
	// Count complexity recursively
	complexity := countComplexity(re)
	// Unroll if complexity is reasonable (less than 10 nodes)
	// For {2}: max 20 nodes, for {3}: max 30 nodes - both acceptable
	return complexity < 10
}

// countComplexity returns a rough measure of expression complexity.
func countComplexity(re *syntax.Regexp) int {
	if re == nil {
		return 0
	}

	count := 1 // Count this node
	for _, sub := range re.Sub {
		count += countComplexity(sub)
	}

	// Weighted by operation type
	switch re.Op {
	case syntax.OpCapture, syntax.OpConcat, syntax.OpAlternate:
		count += 1
	case syntax.OpStar, syntax.OpPlus, syntax.OpQuest, syntax.OpRepeat:
		count += 2 // Repetitions are more complex
	}

	return count
}

// copyRegexp creates a deep copy of a Regexp node.
func copyRegexp(re *syntax.Regexp) *syntax.Regexp {
	if re == nil {
		return nil
	}

	copied := &syntax.Regexp{
		Op:    re.Op,
		Flags: re.Flags,
		Min:   re.Min,
		Max:   re.Max,
		Cap:   re.Cap,
		Name:  re.Name,
	}

	// Copy runes if present
	if len(re.Rune) > 0 {
		copied.Rune = make([]rune, len(re.Rune))
		copy(copied.Rune, re.Rune)
	}

	// Deep copy sub-expressions
	if len(re.Sub) > 0 {
		copied.Sub = make([]*syntax.Regexp, len(re.Sub))
		for i, sub := range re.Sub {
			copied.Sub[i] = copyRegexp(sub)
		}
	}

	return copied
}
