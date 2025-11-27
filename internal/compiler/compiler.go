// Package compiler implements the core regex compilation logic.
package compiler

import (
	"fmt"
	"go/format"
	"os"
	"regexp/syntax"

	"github.com/KromDaniel/regengo/internal/codegen"
	"github.com/dave/jennifer/jen"
)

// Config holds the configuration for code generation.
type Config struct {
	Pattern          string
	Name             string
	OutputFile       string
	Package          string
	Program          *syntax.Prog
	RegexAST         *syntax.Regexp // For extracting capture group names
	UsePool          bool           // Enable sync.Pool for stack reuse
	WithCaptures     bool           // Generate capture group functions
	GenerateTestFile bool           // Generate test file with tests and benchmarks
	TestFileInputs   []string       // Test inputs for generated test file
	ForceThompson    bool           // Force Thompson NFA for MatchString/MatchBytes
	ForceTNFA        bool           // Force Tagged NFA for FindString with captures
	ForceTDFA        bool           // Force Tagged DFA for FindString with captures
	TDFAThreshold    int            // Max DFA states before falling back (0 = use default 500)
	Verbose          bool           // Enable verbose logging of analysis decisions
}

// Compiler generates optimized Go code from regex patterns.
type Compiler struct {
	config                     Config
	file                       *jen.File
	captureNames               []string           // Capture group names (empty string for unnamed groups)
	hasRepeatingCaptures       bool               // True if any capture groups are in repeating context
	needsBacktracking          bool               // True if the program contains alternation instructions
	generatingCaptures         bool               // True when generating Find* functions (needs capture checkpoints)
	isAnchored                 bool               // True if the pattern is anchored to the start of text
	generatingBytes            bool               // True when generating Bytes functions (affects type generation)
	useMemoization             bool               // True if complexity analysis suggests using memoization
	logger                     *Logger            // Verbose logger for analysis decisions
	complexity                 ComplexityAnalysis // Results of pattern complexity analysis
	useThompsonForMatch        bool               // True if Thompson NFA should be used for Match functions
	useTNFAForCaptures         bool               // True if TNFA/memoization should be used for Find functions
	useTDFAForCaptures         bool               // True if Tagged DFA should be used for Find functions
	altsNeedingCheckpoint      map[int]bool       // Alt instructions that need capture checkpointing
	usePerCaptureCheckpointing bool               // True to use stdlib-style per-capture saves instead of array copying
}

// New creates a new compiler instance.
func New(config Config) *Compiler {
	compiler := &Compiler{
		config: config,
		file:   jen.NewFile(config.Package),
		logger: NewLogger(config.Verbose),
	}

	// Extract capture group names if WithCaptures is enabled
	if config.WithCaptures && config.RegexAST != nil {
		compiler.captureNames = extractCaptureNames(config.RegexAST)
		compiler.hasRepeatingCaptures = hasRepeatingCaptures(config.RegexAST)
	}

	// Check if the program needs backtracking
	if config.Program != nil {
		compiler.needsBacktracking = needsBacktracking(config.Program)
		compiler.isAnchored = isAnchored(config.Program)
		compiler.useMemoization = detectComplexity(config.Program)
		// Compute which Alt instructions need capture checkpointing (optimization)
		compiler.altsNeedingCheckpoint = computeAltsNeedingCheckpoint(config.Program)
		// Use per-capture checkpointing for patterns with many Alts that need checkpoints
		// This uses stdlib-style individual capture saves instead of array copying
		compiler.usePerCaptureCheckpointing = shouldUsePerCaptureCheckpointing(compiler.altsNeedingCheckpoint)
	}

	// Perform comprehensive complexity analysis
	compiler.analyzeAndLog()

	return compiler
}

// analyzeAndLog performs complexity analysis and logs the results if verbose mode is enabled.
func (c *Compiler) analyzeAndLog() {
	c.logger.Section("Pattern Analysis")
	c.logger.Log("Pattern: %s", c.config.Pattern)

	if c.config.Program != nil {
		c.logger.Log("NFA states: %d", len(c.config.Program.Inst))
	}

	// Perform comprehensive analysis
	c.complexity = analyzeComplexity(c.config.Program, c.config.RegexAST)

	c.logger.Log("Has nested quantifiers: %v", c.complexity.HasCatastrophicRisk)
	c.logger.Log("Has nested loops: %v", c.complexity.HasNestedLoops)
	c.logger.Log("Has end anchor ($): %v", c.complexity.HasEndAnchor)
	c.logger.Log("Has captures: %v", c.config.WithCaptures)
	c.logger.Log("Needs backtracking: %v", c.needsBacktracking)
	c.logger.Log("Is anchored: %v", c.isAnchored)

	// Determine engine selection based on analysis and user overrides
	c.logger.Section("Engine Selection")

	// Determine if Thompson NFA should be used for Match functions
	// Note: UseThompsonNFA already accounts for end anchor ($ prevents Thompson)
	c.useThompsonForMatch = c.config.ForceThompson || c.complexity.UseThompsonNFA

	// If catastrophic risk but can't use Thompson NFA (e.g., has end anchor), enable memoization
	if c.complexity.HasCatastrophicRisk && !c.useThompsonForMatch {
		c.useMemoization = true
	}

	// Determine capture engine selection
	// Priority: TDFA > TNFA/memoization > simple backtracking
	if c.config.WithCaptures && (c.complexity.HasCatastrophicRisk || c.config.ForceTDFA) {
		// Try TDFA first for patterns with catastrophic risk or when forced
		tdfaGen := NewTDFAGenerator(c)
		if tdfaGen != nil && tdfaGen.CanUseTDFA() {
			c.useTDFAForCaptures = true
			c.complexity.UseTDFA = true
		} else if c.config.ForceTDFA {
			// TDFA was forced but can't be used - warn and fall back
			c.logger.Log("Warning: TDFA forced but pattern exceeds state threshold, falling back")
			c.useTNFAForCaptures = true
		} else {
			// Fall back to TNFA/memoization
			c.useTNFAForCaptures = true
		}
	} else if c.config.ForceTNFA {
		c.useTNFAForCaptures = true
	}

	// Log engine selection decisions
	if c.config.ForceThompson {
		c.logger.Log("Match engine: Thompson NFA (forced by user)")
	} else if c.complexity.HasEndAnchor && (c.complexity.HasCatastrophicRisk || c.complexity.HasNestedLoops) {
		c.logger.Log("Match engine: Backtracking NFA with memoization (end anchor $ requires backtracking)")
	} else if c.complexity.HasCatastrophicRisk || c.complexity.HasNestedLoops {
		c.logger.Log("Match engine: Thompson NFA (catastrophic backtracking risk detected)")
	} else {
		c.logger.Log("Match engine: Backtracking NFA (simple pattern)")
	}

	if c.config.WithCaptures {
		if c.useTDFAForCaptures {
			c.logger.Log("Capture engine: Tagged DFA (O(n) guaranteed, catastrophic risk mitigated)")
		} else if c.config.ForceTNFA {
			c.logger.Log("Capture engine: TNFA/memoization (forced by user)")
		} else if c.useTNFAForCaptures {
			c.logger.Log("Capture engine: TNFA/memoization (TDFA infeasible, catastrophic risk)")
		} else {
			c.logger.Log("Capture engine: Backtracking with checkpoints")
		}
		// Log per-capture checkpointing decision
		c.logger.Log("Alts needing checkpoint: %d (threshold: %d)", len(c.altsNeedingCheckpoint), PerCaptureCheckpointThreshold)
		if c.usePerCaptureCheckpointing {
			c.logger.Log("Using per-capture checkpointing (stdlib-style, reduces allocations)")
		} else {
			c.logger.Log("Using array checkpointing (simple, few checkpoints needed)")
		}
	}
}

// NewCompiler is an alias for New for backward compatibility.
func NewCompiler(config Config) *Compiler {
	return New(config)
}

// SetOutputFile sets the output file path.
func (c *Compiler) SetOutputFile(path string) {
	c.config.OutputFile = path
}

// method returns a jen.Statement for declaring a method on the generated struct.
func (c *Compiler) method(name string) *jen.Statement {
	return c.file.Func().
		Params(jen.Id(c.config.Name)).
		Id(name)
}

// Generate generates the Go code and writes it to the output file.
func (c *Compiler) Generate() error {
	c.file.Comment(fmt.Sprintf("Code generated by regengo for pattern: %s", c.config.Pattern))
	c.file.Comment("DO NOT EDIT.")
	c.file.Line()

	// Add sync.Pool if enabled and backtracking is needed
	// Stack pool is needed for:
	// 1. Non-Thompson Match functions that use backtracking
	// 2. Capture functions that use backtracking (even when Thompson is used for Match)
	needsStackPool := c.config.UsePool && c.needsBacktracking &&
		(!c.useThompsonForMatch || c.config.WithCaptures)
	if needsStackPool {
		// Use [3]int stack pool when captures are enabled (for selective checkpointing)
		// This includes TNFA which uses backtracking-with-captures under the hood
		// Only TDFA doesn't need this as it uses a completely different approach
		if c.config.WithCaptures && !c.useTDFAForCaptures {
			c.generateStackPoolWithCaptures()
		} else {
			c.generateStackPool()
		}
		// Capture stack pool is needed for backtracking capture functions
		// (both regular and TNFA use the same capture stack)
		if c.config.WithCaptures && !c.useTDFAForCaptures {
			c.generateCaptureStackPool()
		}
	}

	// Generate the main struct type
	c.file.Type().Id(c.config.Name).Struct()
	c.file.Line()

	// Generate convenience variable for direct usage
	c.file.Var().Id(fmt.Sprintf("Compiled%s", c.config.Name)).Op("=").Id(c.config.Name).Values()
	c.file.Line()

	// Generate Match functions - use Thompson NFA if recommended
	var matchStringCode, matchBytesCode []jen.Code
	var err error

	if c.useThompsonForMatch {
		thompsonGen := NewThompsonGenerator(c)
		if thompsonGen != nil && thompsonGen.CanUseThompson() {
			c.logger.Log("Using Thompson NFA for MatchString/MatchBytes")
			matchStringCode, err = thompsonGen.GenerateMatchFunction(false)
			if err != nil {
				return fmt.Errorf("failed to generate Thompson match string function: %w", err)
			}
			matchBytesCode, err = thompsonGen.GenerateMatchFunction(true)
			if err != nil {
				return fmt.Errorf("failed to generate Thompson match bytes function: %w", err)
			}
		} else {
			// Fall back to backtracking if Thompson can't be used
			c.logger.Log("Thompson NFA not available (too many states), falling back to backtracking with memoization")
			matchStringCode, err = c.generateMatchFunction(false)
			if err != nil {
				return fmt.Errorf("failed to generate match string function: %w", err)
			}
			matchBytesCode, err = c.generateMatchFunction(true)
			if err != nil {
				return fmt.Errorf("failed to generate match bytes function: %w", err)
			}
		}
	} else {
		matchStringCode, err = c.generateMatchFunction(false)
		if err != nil {
			return fmt.Errorf("failed to generate match string function: %w", err)
		}
		matchBytesCode, err = c.generateMatchFunction(true)
		if err != nil {
			return fmt.Errorf("failed to generate match bytes function: %w", err)
		}
	}

	// Add MatchString method
	c.method("MatchString").
		Params(jen.Id(codegen.InputName).String()).
		Params(jen.Bool()).
		Block(matchStringCode...)

	// Add MatchBytes method
	c.method("MatchBytes").
		Params(jen.Id(codegen.InputName).Index().Byte()).
		Params(jen.Bool()).
		Block(matchBytesCode...)

	// Generate capture group functions if pattern has captures
	if c.config.WithCaptures {
		if c.useTDFAForCaptures {
			c.logger.Log("Using Tagged DFA for capture functions")
			if err := c.generateTDFACaptureFunctions(); err != nil {
				return fmt.Errorf("failed to generate TDFA capture functions: %w", err)
			}
		} else if c.useTNFAForCaptures {
			c.logger.Log("Using TNFA for capture functions")
			if err := c.generateTNFACaptureFunctions(); err != nil {
				return fmt.Errorf("failed to generate TNFA capture functions: %w", err)
			}
		} else {
			if err := c.generateCaptureFunctions(); err != nil {
				return fmt.Errorf("failed to generate capture functions: %w", err)
			}
		}
	}

	// Save to file
	if err := c.file.Save(c.config.OutputFile); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	// Format the generated file
	if err := formatFile(c.config.OutputFile); err != nil {
		return fmt.Errorf("failed to format file: %w", err)
	}

	// Generate test file if requested
	if c.config.GenerateTestFile {
		if err := c.generateTestFile(); err != nil {
			return fmt.Errorf("failed to generate test file: %w", err)
		}
	}

	return nil
}

// generateTNFACaptureFunctions generates capture functions using Tagged NFA.
// For patterns with catastrophic backtracking risk, we use backtracking with memoization
// which provides polynomial time guarantees. Full TNFA can be added in a future phase.
func (c *Compiler) generateTNFACaptureFunctions() error {
	// Enable memoization for capture functions to ensure polynomial time
	c.useMemoization = true
	c.logger.Log("Using backtracking with memoization for captures (polynomial time guarantee)")

	// Note: Stack pools (both regular and capture) are now generated by Generate()
	// when captures are enabled, so we don't need to generate them here

	return c.generateCaptureFunctions()
}

// generateTDFACaptureFunctions generates capture functions using Tagged DFA.
// This provides O(n) matching with captures for patterns that would cause
// exponential backtracking with traditional approaches.
func (c *Compiler) generateTDFACaptureFunctions() error {
	c.logger.Log("Generating Tagged DFA capture functions")

	tdfaGen := NewTDFAGenerator(c)
	if tdfaGen == nil {
		return fmt.Errorf("failed to create TDFA generator")
	}

	// Build TDFA (already done in CanUseTDFA, but rebuild to ensure fresh state)
	if err := tdfaGen.buildTDFA(); err != nil {
		return fmt.Errorf("failed to build TDFA: %w", err)
	}

	c.logger.Log("TDFA built with %d states", len(tdfaGen.states))

	// Generate result struct types
	structName := fmt.Sprintf("%sResult", c.config.Name)
	c.generateCaptureStruct(structName)
	bytesStructName := fmt.Sprintf("%sBytesResult", c.config.Name)
	c.generateCaptureStructBytes(bytesStructName)

	// Generate FindStringReuse function using TDFA (returns pointer for API compatibility)
	findStringCode, err := tdfaGen.GenerateFindFunction(false)
	if err != nil {
		return fmt.Errorf("failed to generate TDFA find string function: %w", err)
	}

	// The TDFA code ends with return statements inside the loop, we need to wrap for pointer return
	c.method("FindStringReuse").
		Params(jen.Id(codegen.InputName).String(), jen.Id("r").Op("*").Id(structName)).
		Params(jen.Op("*").Id(structName), jen.Bool()).
		Block(
			jen.List(jen.Id("result"), jen.Id("ok")).Op(":=").Id(c.config.Name).Values().Dot("findStringInternal").Call(jen.Id(codegen.InputName)),
			jen.If(jen.Op("!").Id("ok")).Block(
				jen.Return(jen.Nil(), jen.False()),
			),
			jen.Return(jen.Op("&").Id("result"), jen.True()),
		)

	// Generate internal FindString function (returns value)
	c.method("findStringInternal").
		Params(jen.Id(codegen.InputName).String()).
		Params(jen.Id(c.config.Name+"Result"), jen.Bool()).
		Block(findStringCode...)

	// Generate FindString that calls FindStringReuse
	c.method("FindString").
		Params(jen.Id(codegen.InputName).String()).
		Params(jen.Op("*").Id(structName), jen.Bool()).
		Block(
			jen.Return(jen.Id(c.config.Name).Values().Dot("FindStringReuse").Call(
				jen.Id(codegen.InputName),
				jen.Nil(),
			)),
		)

	// Generate FindBytesReuse function using TDFA
	findBytesCode, err := tdfaGen.GenerateFindFunction(true)
	if err != nil {
		return fmt.Errorf("failed to generate TDFA find bytes function: %w", err)
	}

	c.method("FindBytesReuse").
		Params(jen.Id(codegen.InputName).Index().Byte(), jen.Id("r").Op("*").Id(bytesStructName)).
		Params(jen.Op("*").Id(bytesStructName), jen.Bool()).
		Block(
			jen.List(jen.Id("result"), jen.Id("ok")).Op(":=").Id(c.config.Name).Values().Dot("findBytesInternal").Call(jen.Id(codegen.InputName)),
			jen.If(jen.Op("!").Id("ok")).Block(
				jen.Return(jen.Nil(), jen.False()),
			),
			jen.Return(jen.Op("&").Id("result"), jen.True()),
		)

	// Generate internal FindBytes function (returns value)
	c.method("findBytesInternal").
		Params(jen.Id(codegen.InputName).Index().Byte()).
		Params(jen.Id(c.config.Name+"BytesResult"), jen.Bool()).
		Block(findBytesCode...)

	// Generate FindBytes that calls FindBytesReuse
	c.method("FindBytes").
		Params(jen.Id(codegen.InputName).Index().Byte()).
		Params(jen.Op("*").Id(bytesStructName), jen.Bool()).
		Block(
			jen.Return(jen.Id(c.config.Name).Values().Dot("FindBytesReuse").Call(
				jen.Id(codegen.InputName),
				jen.Nil(),
			)),
		)

	// Generate FindAllString using TDFA (iterate with FindString)
	c.generateFindAllStringTDFA()

	// Generate FindAllBytes using TDFA
	c.generateFindAllBytesTDFA()

	return nil
}

// generateFindAllStringTDFA generates FindAllString using TDFA-based FindString.
func (c *Compiler) generateFindAllStringTDFA() {
	structName := fmt.Sprintf("%sResult", c.config.Name)
	c.method("FindAllString").
		Params(
			jen.Id(codegen.InputName).String(),
			jen.Id("n").Int(),
		).
		Params(jen.Index().Op("*").Id(structName)).
		Block(
			jen.If(jen.Id("n").Op("==").Lit(0)).Block(
				jen.Return(jen.Nil()),
			),
			jen.Var().Id("results").Index().Op("*").Id(structName),
			jen.Id("offset").Op(":=").Lit(0),
			jen.For(jen.Id("offset").Op("<").Len(jen.Id(codegen.InputName))).Block(
				jen.List(jen.Id("result"), jen.Id("ok")).Op(":=").Id(c.config.Name).Values().Dot("FindString").Call(
					jen.Id(codegen.InputName).Index(jen.Id("offset").Op(":")),
				),
				jen.If(jen.Op("!").Id("ok")).Block(
					jen.Break(),
				),
				jen.Id("results").Op("=").Append(jen.Id("results"), jen.Id("result")),
				jen.If(jen.Id("n").Op(">").Lit(0).Op("&&").Len(jen.Id("results")).Op(">=").Id("n")).Block(
					jen.Break(),
				),
				jen.Comment("Move past this match"),
				jen.Id("matchLen").Op(":=").Len(jen.Id("result").Dot("Match")),
				jen.If(jen.Id("matchLen").Op(">").Lit(0)).Block(
					jen.Id("offset").Op("+=").Id("matchLen"),
				).Else().Block(
					jen.Id("offset").Op("++"),
				),
			),
			jen.Return(jen.Id("results")),
		)
}

// generateFindAllBytesTDFA generates FindAllBytes using TDFA-based FindBytes.
func (c *Compiler) generateFindAllBytesTDFA() {
	bytesStructName := fmt.Sprintf("%sBytesResult", c.config.Name)
	c.method("FindAllBytes").
		Params(
			jen.Id(codegen.InputName).Index().Byte(),
			jen.Id("n").Int(),
		).
		Params(jen.Index().Op("*").Id(bytesStructName)).
		Block(
			jen.If(jen.Id("n").Op("==").Lit(0)).Block(
				jen.Return(jen.Nil()),
			),
			jen.Var().Id("results").Index().Op("*").Id(bytesStructName),
			jen.Id("offset").Op(":=").Lit(0),
			jen.For(jen.Id("offset").Op("<").Len(jen.Id(codegen.InputName))).Block(
				jen.List(jen.Id("result"), jen.Id("ok")).Op(":=").Id(c.config.Name).Values().Dot("FindBytes").Call(
					jen.Id(codegen.InputName).Index(jen.Id("offset").Op(":")),
				),
				jen.If(jen.Op("!").Id("ok")).Block(
					jen.Break(),
				),
				jen.Id("results").Op("=").Append(jen.Id("results"), jen.Id("result")),
				jen.If(jen.Id("n").Op(">").Lit(0).Op("&&").Len(jen.Id("results")).Op(">=").Id("n")).Block(
					jen.Break(),
				),
				jen.Comment("Move past this match"),
				jen.Id("matchLen").Op(":=").Len(jen.Id("result").Dot("Match")),
				jen.If(jen.Id("matchLen").Op(">").Lit(0)).Block(
					jen.Id("offset").Op("+=").Id("matchLen"),
				).Else().Block(
					jen.Id("offset").Op("++"),
				),
			),
			jen.Return(jen.Id("results")),
		)
}

// formatFile reads a file, formats it with go/format, and writes it back.
func formatFile(path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	formatted, err := format.Source(src)
	if err != nil {
		return err
	}

	return os.WriteFile(path, formatted, 0644)
}

// findRequiredPrefix scans the program to find a required starting byte.
// Returns the byte and true if found, 0 and false otherwise.
func (c *Compiler) findRequiredPrefix() (byte, bool) {
	pc := c.config.Program.Start
	for {
		inst := c.config.Program.Inst[pc]
		switch inst.Op {
		case syntax.InstNop, syntax.InstCapture:
			pc = int(inst.Out)
			continue
		case syntax.InstRune1:
			// Found a single rune literal
			if len(inst.Rune) == 1 && inst.Rune[0] < 128 {
				return byte(inst.Rune[0]), true
			}
			return 0, false
		default:
			return 0, false
		}
	}
}

// generateMatchFunction generates the main matching logic.
func (c *Compiler) generateMatchFunction(isBytes bool) ([]jen.Code, error) {
	// Set the generatingBytes flag for instruction generation
	c.generatingBytes = isBytes

	prefix, hasPrefix := c.findRequiredPrefix()

	code := []jen.Code{
		// Initialize length
		jen.Id(codegen.InputLenName).Op(":=").Len(jen.Id(codegen.InputName)),
	}

	if hasPrefix && !c.isAnchored {
		var indexCall *jen.Statement
		if isBytes {
			indexCall = jen.Qual("bytes", "IndexByte").Call(jen.Id(codegen.InputName), jen.Lit(byte(prefix)))
		} else {
			indexCall = jen.Qual("strings", "IndexByte").Call(jen.Id(codegen.InputName), jen.Lit(byte(prefix)))
		}

		code = append(code,
			// Fast forward to first occurrence
			jen.Id("idx").Op(":=").Add(indexCall),
			jen.If(jen.Id("idx").Op("==").Lit(-1)).Block(jen.Return(jen.False())),
			jen.Id(codegen.OffsetName).Op(":=").Id("idx"),
		)
	} else {
		code = append(code,
			// Initialize offset
			jen.Id(codegen.OffsetName).Op(":=").Lit(0),
		)
	}

	// Only add stack initialization if backtracking is needed
	if c.needsBacktracking {
		// Determine stack entry size - must match pool if shared
		stackSize := 2
		if c.config.WithCaptures && !c.useTDFAForCaptures {
			stackSize = 3
		}

		// Add stack initialization (pooled or regular)
		if c.config.UsePool {
			code = append(code, c.generatePooledStackInit()...)
		} else {
			code = append(code,
				jen.Id(codegen.StackName).Op(":=").Make(jen.Index().Index(jen.Lit(stackSize)).Int(), jen.Lit(0), jen.Lit(32)),
			)
		}
	}

	// Initialize memoization bit-vector if needed (Optimization: Avoid exponential backtracking)
	// Uses bit-vector instead of map for O(1) check/set with zero allocations per operation
	if c.useMemoization {
		numInst := len(c.config.Program.Inst)
		code = append(code,
			// visitedSize = numInst * (l + 1) bits, rounded up to uint32 words
			jen.Id("visitedSize").Op(":=").Lit(numInst).Op("*").Parens(jen.Id(codegen.InputLenName).Op("+").Lit(1)),
			jen.Id(codegen.VisitedName).Op(":=").Make(jen.Index().Uint32(), jen.Parens(jen.Id("visitedSize").Op("+").Lit(31)).Op("/").Lit(32)),
		)
	}

	code = append(code,
		// Initialize next instruction
		jen.Id(codegen.NextInstructionName).Op(":=").Lit(int(c.config.Program.Start)),
		// Jump to step selector
		jen.Goto().Id(codegen.StepSelectName),
	)

	// Only add backtracking logic if needed
	if c.needsBacktracking {
		code = append(code, c.generateBacktracking(prefix, hasPrefix, isBytes)...)
	} else {
		// For patterns without backtracking:
		fallback := []jen.Code{jen.Id(codegen.TryFallbackName).Op(":")}

		// If not anchored, we must retry at next offset
		if !c.isAnchored {
			if hasPrefix {
				var indexCall *jen.Statement
				if isBytes {
					indexCall = jen.Qual("bytes", "IndexByte").Call(
						jen.Id(codegen.InputName).Index(jen.Id(codegen.OffsetName).Op(":")),
						jen.Lit(byte(prefix)),
					)
				} else {
					indexCall = jen.Qual("strings", "IndexByte").Call(
						jen.Id(codegen.InputName).Index(jen.Id(codegen.OffsetName).Op(":")),
						jen.Lit(byte(prefix)),
					)
				}

				fallback = append(fallback,
					jen.Id(codegen.OffsetName).Op("++"),
					jen.If(jen.Id(codegen.InputLenName).Op(">").Id(codegen.OffsetName)).Block(
						jen.Id("idx").Op(":=").Add(indexCall),
						jen.If(jen.Id("idx").Op("==").Lit(-1)).Block(jen.Return(jen.False())),
						jen.Id(codegen.OffsetName).Op("+=").Id("idx"),
						jen.Id(codegen.NextInstructionName).Op("=").Lit(int(c.config.Program.Start)),
						jen.Goto().Id(codegen.StepSelectName),
					),
				)
			} else {
				fallback = append(fallback,
					jen.If(jen.Id(codegen.InputLenName).Op(">").Id(codegen.OffsetName)).Block(
						jen.Id(codegen.NextInstructionName).Op("=").Lit(int(c.config.Program.Start)),
						jen.Id(codegen.OffsetName).Op("++"),
						jen.Goto().Id(codegen.StepSelectName),
					),
				)
			}
		}

		fallback = append(fallback, jen.Return(jen.False()))
		code = append(code, fallback...)
	}

	// Add step selector
	code = append(code, c.generateStepSelector()...)

	// Generate instructions
	instructions, err := c.generateInstructions()
	if err != nil {
		return nil, err
	}
	code = append(code, instructions...)

	return code, nil
}
