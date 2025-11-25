package compiler

import (
	"fmt"
	"regexp/syntax"

	"github.com/KromDaniel/regengo/internal/codegen"
	"github.com/dave/jennifer/jen"
)

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

	// Optimization #1: When generating Find* functions with captures, save capture checkpoint
	if c.generatingCaptures {
		return []jen.Code{
			label,
			jen.Block(
				// Save current capture state as checkpoint (flattened)
				jen.Id("captureStack").Op("=").Append(jen.Id("captureStack"), jen.Id(codegen.CapturesName).Index(jen.Empty(), jen.Empty()).Op("...")),
				// Push to backtracking stack
				jen.Id(codegen.StackName).Op("=").Append(
					jen.Id(codegen.StackName),
					jen.Index(jen.Lit(2)).Int().Values(jen.Id(codegen.OffsetName), jen.Lit(int(inst.Arg))),
				),
				jen.Goto().Id(codegen.InstructionName(inst.Out)),
			),
		}, nil
	}

	// Optimization #2: Greedy loop optimization (auto-detected)
	// Only optimize simple greedy loops where the target is a character-matching instruction
	// Avoid optimizing nested alternations (like in complex patterns) to prevent regressions
	if isGreedyLoop && c.isSimpleGreedyLoop(inst) {
		return []jen.Code{
			label,
			jen.Block(
				// For greedy loops, push to stack with the loop start (for backtracking)
				// Then try the continuation instruction first
				jen.Id(codegen.StackName).Op("=").Append(
					jen.Id(codegen.StackName),
					jen.Index(jen.Lit(2)).Int().Values(jen.Id(codegen.OffsetName), jen.Lit(int(inst.Out))),
				),
				// Try continuation first (greedy behavior: we've matched, now try what comes next)
				jen.Goto().Id(codegen.InstructionName(inst.Arg)),
			),
		}, nil
	}

	// Standard alternation: Just use append - Go's runtime handles capacity efficiently
	return []jen.Code{
		label,
		jen.Block(
			jen.Id(codegen.StackName).Op("=").Append(
				jen.Id(codegen.StackName),
				jen.Index(jen.Lit(2)).Int().Values(jen.Id(codegen.OffsetName), jen.Lit(int(inst.Arg))),
			),
			jen.Goto().Id(codegen.InstructionName(inst.Out)),
		),
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

	// Word boundary checks would go here if needed
	// For now, we'll leave them unimplemented
	if emptyOp&(syntax.EmptyWordBoundary|syntax.EmptyNoWordBoundary) != 0 {
		return nil, fmt.Errorf("word boundary assertions (\\b, \\B) are not yet implemented")
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
