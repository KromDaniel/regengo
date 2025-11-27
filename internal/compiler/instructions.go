package compiler

import (
	"fmt"
	"regexp/syntax"

	"github.com/KromDaniel/regengo/internal/codegen"
	"github.com/dave/jennifer/jen"
)

// boolToInt converts a bool to int for use in generated code
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// generateStepSelector generates the instruction dispatch switch.
func (c *Compiler) generateStepSelector() []jen.Code {
	cases := []jen.Code{}
	for i := range c.config.Program.Inst {
		cases = append(cases,
			jen.Case(jen.Lit(i)).Block(jen.Goto().Id(codegen.InstructionName(uint32(i)))),
		)
	}

	return []jen.Code{
		jen.Id(codegen.StepSelectName).Op(":"),
		jen.Switch(jen.Id(codegen.NextInstructionName)).Block(cases...),
	}
}

// generateInstructions generates code for all instructions.
func (c *Compiler) generateInstructions() ([]jen.Code, error) {
	var code []jen.Code

	for i, inst := range c.config.Program.Inst {
		instCode, err := c.generateInstruction(uint32(i), &inst)
		if err != nil {
			return nil, fmt.Errorf("failed to generate instruction %d: %w", i, err)
		}
		code = append(code, instCode...)
	}

	return code, nil
}

// generateInstruction generates code for a single instruction.
func (c *Compiler) generateInstruction(id uint32, inst *syntax.Inst) ([]jen.Code, error) {
	label := jen.Id(codegen.InstructionName(id)).Op(":")

	switch inst.Op {
	case syntax.InstMatch:
		return []jen.Code{
			label,
			jen.Block(jen.Return(jen.True())),
		}, nil

	case syntax.InstFail:
		return []jen.Code{
			label,
			jen.Block(jen.Return(jen.False())),
		}, nil

	case syntax.InstCapture:
		// For Match functions without captures, just skip capture instructions
		return []jen.Code{
			label,
			jen.Block(
				jen.Goto().Id(codegen.InstructionName(inst.Out)),
			),
		}, nil

	case syntax.InstRune:
		return c.generateRuneInst(label, inst)

	case syntax.InstRune1:
		return c.generateRune1Inst(label, inst)

	case syntax.InstRuneAny:
		return c.generateRuneAnyInst(label, inst)

	case syntax.InstRuneAnyNotNL:
		return c.generateRuneAnyNotNLInst(label, inst)

	case syntax.InstAlt:
		return c.generateAltInst(label, inst, id)

	case syntax.InstAltMatch:
		return c.generateAltMatchInst(label, inst)

	case syntax.InstEmptyWidth:
		return c.generateEmptyWidthInst(label, inst)

	case syntax.InstNop:
		return c.generateNopInst(label, inst)

	default:
		return nil, fmt.Errorf("unsupported instruction type: %v", inst.Op)
	}
}

// generateRune1Inst generates code for InstRune1 (single rune match).
func (c *Compiler) generateRune1Inst(label *jen.Statement, inst *syntax.Inst) ([]jen.Code, error) {
	return []jen.Code{
		label,
		jen.Block(
			jen.If(jen.Id(codegen.InputLenName).Op("<=").Id(codegen.OffsetName)).Block(
				jen.Goto().Id(codegen.TryFallbackName),
			),
		),
		jen.Block(
			jen.If(jen.Id(codegen.InputName).Index(jen.Id(codegen.OffsetName)).Op("!=").Lit(byte(inst.Rune[0]))).Block(
				jen.Goto().Id(codegen.TryFallbackName),
			),
			jen.Id(codegen.OffsetName).Op("++"),
			jen.Goto().Id(codegen.InstructionName(inst.Out)),
		),
	}, nil
}

// generateRuneInst generates code for InstRune (character class match).
func (c *Compiler) generateRuneInst(label *jen.Statement, inst *syntax.Inst) ([]jen.Code, error) {
	return []jen.Code{
		label,
		jen.Block(
			jen.If(jen.Id(codegen.InputLenName).Op("<=").Id(codegen.OffsetName)).Block(
				jen.Goto().Id(codegen.TryFallbackName),
			),
		),
		jen.Block(
			jen.If(c.generateRuneCheck(inst.Rune)).Block(
				jen.Goto().Id(codegen.TryFallbackName),
			),
			jen.Id(codegen.OffsetName).Op("++"),
			jen.Goto().Id(codegen.InstructionName(inst.Out)),
		),
	}, nil
}

// generateRuneAnyInst generates code for InstRuneAny (match any character).
func (c *Compiler) generateRuneAnyInst(label *jen.Statement, inst *syntax.Inst) ([]jen.Code, error) {
	return []jen.Code{
		label,
		jen.Block(
			jen.If(jen.Id(codegen.InputLenName).Op("<=").Id(codegen.OffsetName)).Block(
				jen.Goto().Id(codegen.TryFallbackName),
			),
			jen.Id(codegen.OffsetName).Op("++"),
			jen.Goto().Id(codegen.InstructionName(inst.Out)),
		),
	}, nil
}

// generateRuneAnyNotNLInst generates code for InstRuneAnyNotNL (match any character except newline).
func (c *Compiler) generateRuneAnyNotNLInst(label *jen.Statement, inst *syntax.Inst) ([]jen.Code, error) {
	// Both string[i] and []byte[i] return byte in Go, so always use byte('\n')
	newlineCheck := jen.Id(codegen.InputName).Index(jen.Id(codegen.OffsetName)).Op("==").Lit(byte('\n'))

	return []jen.Code{
		label,
		jen.Block(
			jen.If(
				jen.Id(codegen.InputLenName).Op("<=").Id(codegen.OffsetName).Op("||").Add(newlineCheck),
			).Block(
				jen.Goto().Id(codegen.TryFallbackName),
			),
			jen.Id(codegen.OffsetName).Op("++"),
			jen.Goto().Id(codegen.InstructionName(inst.Out)),
		),
	}, nil
}

// generateAltInst generates code for InstAlt (alternation with backtracking).
func (c *Compiler) generateAltInst(label *jen.Statement, inst *syntax.Inst, id uint32) ([]jen.Code, error) {
	// Detect greedy loop: inst.Out points backward (to earlier instruction)
	// This indicates patterns like +, *, {n,} quantifiers
	isGreedyLoop := inst.Out < id

	// Prepare memoization code using bit-vector
	// idx = stateId * (l + 1) + offset; word = idx/32; bit = 1 << (idx%32)
	var memoCheck []jen.Code
	if c.useMemoization {
		numInst := len(c.config.Program.Inst)
		memoCheck = []jen.Code{
			// idx := stateId * (l + 1) + offset
			jen.Id("idx").Op(":=").Lit(int(id)).Op("*").Parens(jen.Id(codegen.InputLenName).Op("+").Lit(1)).Op("+").Id(codegen.OffsetName),
			// word, bit := idx/32, uint32(1)<<(idx%32)
			jen.List(jen.Id("word"), jen.Id("bit")).Op(":=").
				Id("idx").Op("/").Lit(32).Op(",").
				Uint32().Call(jen.Lit(1)).Op("<<").Parens(jen.Id("idx").Op("%").Lit(32)),
			// if visited[word] & bit != 0 { goto TryFallback }
			jen.If(jen.Id(codegen.VisitedName).Index(jen.Id("word")).Op("&").Id("bit").Op("!=").Lit(0)).Block(
				jen.Goto().Id(codegen.TryFallbackName),
			),
			// visited[word] |= bit
			jen.Id(codegen.VisitedName).Index(jen.Id("word")).Op("|=").Id("bit"),
		}
		_ = numInst // numInst available for potential future optimizations
	}

	// Optimization #1: When generating Find* functions with captures
	if c.generatingCaptures {
		block := append([]jen.Code{}, memoCheck...)

		// Per-capture checkpointing mode: captures are saved at InstCapture instructions,
		// not at Alt instructions. This is the stdlib-style approach with zero array copies.
		if c.usePerCaptureCheckpointing {
			block = append(block,
				// Push simple backtrack entry (type=0 for Alt backtrack)
				jen.Id(codegen.StackName).Op("=").Append(
					jen.Id(codegen.StackName),
					jen.Index(jen.Lit(3)).Int().Values(jen.Id(codegen.OffsetName), jen.Lit(int(inst.Arg)), jen.Lit(0)),
				),
				jen.Goto().Id(codegen.InstructionName(inst.Out)),
			)
			return []jen.Code{
				label,
				jen.Block(block...),
			}, nil
		}

		// Array-copy checkpointing mode (for patterns with few checkpoint-needing Alts)
		// Only checkpoint if this Alt's Out branch can modify captures (selective checkpointing)
		needsCheckpoint := c.altsNeedingCheckpoint[int(id)]
		if needsCheckpoint {
			block = append(block,
				// Save current capture state as checkpoint (flattened)
				jen.Id("captureStack").Op("=").Append(jen.Id("captureStack"), jen.Id(codegen.CapturesName).Index(jen.Empty(), jen.Empty()).Op("...")),
			)
		}

		block = append(block,
			// Push to backtracking stack with checkpoint flag
			jen.Id(codegen.StackName).Op("=").Append(
				jen.Id(codegen.StackName),
				jen.Index(jen.Lit(3)).Int().Values(jen.Id(codegen.OffsetName), jen.Lit(int(inst.Arg)), jen.Lit(boolToInt(needsCheckpoint))),
			),
			jen.Goto().Id(codegen.InstructionName(inst.Out)),
		)
		return []jen.Code{
			label,
			jen.Block(block...),
		}, nil
	}

	// Determine stack entry size - must match the pool type
	// When captures are enabled (not TDFA), we use [3]int even for Match functions
	// because they share the same pool
	stackSize := 2
	if c.config.WithCaptures && !c.useTDFAForCaptures {
		stackSize = 3
	}

	// Optimization #2: Greedy loop optimization (auto-detected)
	// Only optimize simple greedy loops where the target is a character-matching instruction
	// Avoid optimizing nested alternations (like in complex patterns) to prevent regressions
	if isGreedyLoop && c.isSimpleGreedyLoop(inst) {
		block := append([]jen.Code{}, memoCheck...)
		var stackEntry jen.Code
		if stackSize == 3 {
			stackEntry = jen.Index(jen.Lit(3)).Int().Values(jen.Id(codegen.OffsetName), jen.Lit(int(inst.Out)), jen.Lit(0))
		} else {
			stackEntry = jen.Index(jen.Lit(2)).Int().Values(jen.Id(codegen.OffsetName), jen.Lit(int(inst.Out)))
		}
		block = append(block,
			// For greedy loops, push to stack with the loop start (for backtracking)
			// Then try the continuation instruction first
			jen.Id(codegen.StackName).Op("=").Append(
				jen.Id(codegen.StackName),
				stackEntry,
			),
			// Try continuation first (greedy behavior: we've matched, now try what comes next)
			jen.Goto().Id(codegen.InstructionName(inst.Arg)),
		)
		return []jen.Code{
			label,
			jen.Block(block...),
		}, nil
	}

	// Standard alternation: Just use append - Go's runtime handles capacity efficiently
	block := append([]jen.Code{}, memoCheck...)
	var stackEntry jen.Code
	if stackSize == 3 {
		stackEntry = jen.Index(jen.Lit(3)).Int().Values(jen.Id(codegen.OffsetName), jen.Lit(int(inst.Arg)), jen.Lit(0))
	} else {
		stackEntry = jen.Index(jen.Lit(2)).Int().Values(jen.Id(codegen.OffsetName), jen.Lit(int(inst.Arg)))
	}
	block = append(block,
		jen.Id(codegen.StackName).Op("=").Append(
			jen.Id(codegen.StackName),
			stackEntry,
		),
		jen.Goto().Id(codegen.InstructionName(inst.Out)),
	)
	return []jen.Code{
		label,
		jen.Block(block...),
	}, nil
}

// isSimpleGreedyLoop checks if a greedy loop is safe to optimize.
// Returns true for simple character-matching loops (like [\w]+ or a+)
// Returns false for complex nested alternations to avoid regressions.
func (c *Compiler) isSimpleGreedyLoop(inst *syntax.Inst) bool {
	// Check what instruction the loop jumps back to
	loopTarget := c.config.Program.Inst[inst.Out]

	// Optimize only if the loop target is a simple character-matching instruction
	// Don't optimize if it's another alternation (nested structure)
	switch loopTarget.Op {
	case syntax.InstRune, syntax.InstRune1, syntax.InstRuneAny, syntax.InstRuneAnyNotNL:
		// Simple character matching - safe to optimize
		return true
	case syntax.InstAlt:
		// Nested alternation - not safe to optimize (causes regression in complex patterns)
		return false
	default:
		// Other instruction types - conservatively don't optimize
		return false
	}
}

// generateAltMatchInst generates code for InstAltMatch (alternation without backtracking).
func (c *Compiler) generateAltMatchInst(label *jen.Statement, inst *syntax.Inst) ([]jen.Code, error) {
	return []jen.Code{
		label,
		jen.Block(
			jen.Goto().Id(codegen.InstructionName(inst.Out)),
		),
	}, nil
}

// generateEmptyWidthInst generates code for InstEmptyWidth (position assertions).
func (c *Compiler) generateEmptyWidthInst(label *jen.Statement, inst *syntax.Inst) ([]jen.Code, error) {
	emptyOp := syntax.EmptyOp(inst.Arg)

	var checks []jen.Code

	// Check for beginning of text (^)
	if emptyOp&syntax.EmptyBeginText != 0 {
		// offset must be 0
		checks = append(checks,
			jen.If(jen.Id(codegen.OffsetName).Op("!=").Lit(0)).Block(
				jen.Goto().Id(codegen.TryFallbackName),
			),
		)
	}

	// Check for end of text ($)
	if emptyOp&syntax.EmptyEndText != 0 {
		// offset must equal length
		checks = append(checks,
			jen.If(jen.Id(codegen.OffsetName).Op("!=").Id(codegen.InputLenName)).Block(
				jen.Goto().Id(codegen.TryFallbackName),
			),
		)
	}

	// Check for beginning of line
	if emptyOp&syntax.EmptyBeginLine != 0 {
		// offset must be 0 OR previous character must be newline
		// Both string[i] and []byte[i] return byte in Go
		prevNewlineCheck := jen.Id(codegen.InputName).Index(jen.Id(codegen.OffsetName).Op("-").Lit(1)).Op("!=").Lit(byte('\n'))
		checks = append(checks,
			jen.If(
				jen.Id(codegen.OffsetName).Op("!=").Lit(0).Op("&&").
					Parens(
						jen.Id(codegen.OffsetName).Op("==").Lit(0).Op("||").Add(prevNewlineCheck),
					),
			).Block(
				jen.Goto().Id(codegen.TryFallbackName),
			),
		)
	}

	// Check for end of line
	if emptyOp&syntax.EmptyEndLine != 0 {
		// offset must equal length OR current character must be newline
		// Both string[i] and []byte[i] return byte in Go
		currNewlineCheck := jen.Id(codegen.InputName).Index(jen.Id(codegen.OffsetName)).Op("!=").Lit(byte('\n'))
		checks = append(checks,
			jen.If(
				jen.Id(codegen.OffsetName).Op("!=").Id(codegen.InputLenName).Op("&&").Add(currNewlineCheck),
			).Block(
				jen.Goto().Id(codegen.TryFallbackName),
			),
		)
	}

	// Word boundary check (\b)
	// A word boundary exists when the character before and after the position
	// have different "word" status. Word characters are [a-zA-Z0-9_].
	// At start of text, "before" is considered non-word.
	// At end of text, "after" is considered non-word.
	if emptyOp&syntax.EmptyWordBoundary != 0 {
		// Generate: prevIsWord := offset > 0 && isWordChar(input[offset-1])
		// Generate: currIsWord := offset < inputLen && isWordChar(input[offset])
		// Generate: if prevIsWord == currIsWord { goto TryFallback } // no boundary
		checks = append(checks,
			jen.Id("prevIsWord").Op(":=").Id(codegen.OffsetName).Op(">").Lit(0).Op("&&").
				Id(codegen.IsWordCharName).Call(jen.Id(codegen.InputName).Index(jen.Id(codegen.OffsetName).Op("-").Lit(1))),
			jen.Id("currIsWord").Op(":=").Id(codegen.OffsetName).Op("<").Id(codegen.InputLenName).Op("&&").
				Id(codegen.IsWordCharName).Call(jen.Id(codegen.InputName).Index(jen.Id(codegen.OffsetName))),
			jen.If(jen.Id("prevIsWord").Op("==").Id("currIsWord")).Block(
				jen.Goto().Id(codegen.TryFallbackName),
			),
		)
	}

	// Non-word boundary check (\B)
	// Matches when NOT at a word boundary (both sides are word chars or both are non-word)
	if emptyOp&syntax.EmptyNoWordBoundary != 0 {
		checks = append(checks,
			jen.Id("prevIsWord").Op(":=").Id(codegen.OffsetName).Op(">").Lit(0).Op("&&").
				Id(codegen.IsWordCharName).Call(jen.Id(codegen.InputName).Index(jen.Id(codegen.OffsetName).Op("-").Lit(1))),
			jen.Id("currIsWord").Op(":=").Id(codegen.OffsetName).Op("<").Id(codegen.InputLenName).Op("&&").
				Id(codegen.IsWordCharName).Call(jen.Id(codegen.InputName).Index(jen.Id(codegen.OffsetName))),
			jen.If(jen.Id("prevIsWord").Op("!=").Id("currIsWord")).Block(
				jen.Goto().Id(codegen.TryFallbackName),
			),
		)
	}

	// Add the checks and continue to next instruction
	code := []jen.Code{label}
	if len(checks) > 0 {
		code = append(code, jen.Block(append(checks,
			jen.Goto().Id(codegen.InstructionName(inst.Out)),
		)...))
	} else {
		code = append(code, jen.Block(
			jen.Goto().Id(codegen.InstructionName(inst.Out)),
		))
	}

	return code, nil
}

// generateNopInst generates code for InstNop (no operation).
func (c *Compiler) generateNopInst(label *jen.Statement, inst *syntax.Inst) ([]jen.Code, error) {
	return []jen.Code{
		label,
		jen.Block(
			jen.Goto().Id(codegen.InstructionName(inst.Out)),
		),
	}, nil
}
