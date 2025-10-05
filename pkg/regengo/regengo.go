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

	// BytesView generates a separate struct for FindBytes that uses []byte instead of string for zero-copy captures
	// Only applies when pattern has capture groups. Generates both *Match (string fields) and *BytesMatch ([]byte fields) structs.
	BytesView bool

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
		BytesView:        opts.BytesView && hasCaptures, // Only apply if captures exist
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
