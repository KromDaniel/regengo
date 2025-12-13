package compiler

import (
	"github.com/KromDaniel/regengo/internal/codegen"
	"github.com/dave/jennifer/jen"
)

// generateBacktracking generates the backtracking logic.
func (c *Compiler) generateBacktracking(prefix byte, hasPrefix bool, isBytes bool) []jen.Code {
	retryBlock := []jen.Code{}
	// Optimization: If anchored, we don't retry at next offset
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

			retryBlock = append(retryBlock,
				jen.Id(codegen.OffsetName).Op("++"),
				jen.If(jen.Id(codegen.InputLenName).Op(">").Id(codegen.OffsetName)).Block(
					jen.Id("idx").Op(":=").Add(indexCall),
					jen.If(jen.Id("idx").Op("==").Lit(-1)).Block(jen.Return(jen.False())),
					jen.Id(codegen.OffsetName).Op("+=").Id("idx"),
					// Clear visited bit-vector if memoization is used
					func() jen.Code {
						if c.useMemoization {
							return jen.For(jen.Id("i").Op(":=").Range().Id(codegen.VisitedName)).Block(
								jen.Id(codegen.VisitedName).Index(jen.Id("i")).Op("=").Lit(0),
							)
						}
						return jen.Null()
					}(),
					jen.Id(codegen.NextInstructionName).Op("=").Lit(int(c.config.Program.Start)),
					jen.Goto().Id(codegen.StepSelectName),
				),
			)
		} else {
			retryBlock = append(retryBlock,
				jen.If(jen.Id(codegen.InputLenName).Op(">").Id(codegen.OffsetName)).Block(
					jen.Id(codegen.NextInstructionName).Op("=").Lit(int(c.config.Program.Start)),
					jen.Id(codegen.OffsetName).Op("++"),
					// Clear visited bit-vector if memoization is used
					func() jen.Code {
						if c.useMemoization {
							return jen.For(jen.Id("i").Op(":=").Range().Id(codegen.VisitedName)).Block(
								jen.Id(codegen.VisitedName).Index(jen.Id("i")).Op("=").Lit(0),
							)
						}
						return jen.Null()
					}(),
					jen.Goto().Id(codegen.StepSelectName),
				),
			)
		}
	}
	retryBlock = append(retryBlock, jen.Return(jen.False()))

	return []jen.Code{
		jen.Id(codegen.TryFallbackName).Op(":"),
		jen.If(jen.Len(jen.Id(codegen.StackName)).Op(">").Lit(0)).Block(
			jen.Id("last").Op(":=").Id(codegen.StackName).Index(jen.Len(jen.Id(codegen.StackName)).Op("-").Lit(1)),
			jen.Id(codegen.OffsetName).Op("=").Id("last").Index(jen.Lit(0)),
			jen.Id(codegen.NextInstructionName).Op("=").Id("last").Index(jen.Lit(1)),
			jen.Id(codegen.StackName).Op("=").Id(codegen.StackName).Index(jen.Empty(), jen.Len(jen.Id(codegen.StackName)).Op("-").Lit(1)),
			jen.Goto().Id(codegen.StepSelectName),
		).Else().Block(retryBlock...),
	}
}

// generateBacktrackingWithCaptures generates the backtracking logic for capture functions.
// Supports two modes:
// - Per-capture checkpointing (type=2): restores individual captures, loops until finding Alt backtrack
// - Array checkpointing (type=1): restores entire capture array from checkpoint stack
func (c *Compiler) generateBacktrackingWithCaptures() []jen.Code {
	retryBlock := []jen.Code{}
	// Optimization: If anchored, we don't retry at next offset
	if !c.isAnchored {
		var clearCaptureStack jen.Code
		if c.usePerCaptureCheckpointing {
			// Per-capture mode doesn't use captureStack
			clearCaptureStack = jen.Null()
		} else {
			clearCaptureStack = jen.Id("captureStack").Op("=").Id("captureStack").Index(jen.Empty(), jen.Lit(0))
		}

		retryBlock = append(retryBlock,
			jen.If(jen.Id(codegen.InputLenName).Op(">").Id(codegen.OffsetName)).Block(
				jen.Id(codegen.OffsetName).Op("++"),
				// Reset captures array for new match attempt
				jen.For(jen.Id("i").Op(":=").Range().Id(codegen.CapturesName)).Block(
					jen.Id(codegen.CapturesName).Index(jen.Id("i")).Op("=").Lit(0),
				),
				// Clear capture checkpoint stack (only for array mode)
				clearCaptureStack,
				// Clear visited bit-vector if memoization is used
				func() jen.Code {
					if c.useMemoization {
						return jen.For(jen.Id("i").Op(":=").Range().Id(codegen.VisitedName)).Block(
							jen.Id(codegen.VisitedName).Index(jen.Id("i")).Op("=").Lit(0),
						)
					}
					return jen.Null()
				}(),
				// Set capture[0] to mark start of match attempt (after incrementing offset)
				jen.Id(codegen.CapturesName).Index(jen.Lit(0)).Op("=").Id(codegen.OffsetName),
				jen.Id(codegen.NextInstructionName).Op("=").Lit(int(c.config.Program.Start)),
				jen.Goto().Id(codegen.StepSelectName),
			),
		)
	}
	retryBlock = append(retryBlock, jen.Return(jen.Nil(), jen.False()))

	// Per-capture checkpointing mode: loop to process capture restores (type=2)
	// until we find an Alt backtrack point (type=0)
	if c.usePerCaptureCheckpointing {
		return []jen.Code{
			jen.Id(codegen.TryFallbackName).Op(":"),
			jen.For(jen.Len(jen.Id(codegen.StackName)).Op(">").Lit(0)).Block(
				jen.Id("last").Op(":=").Id(codegen.StackName).Index(jen.Len(jen.Id(codegen.StackName)).Op("-").Lit(1)),
				jen.Id(codegen.StackName).Op("=").Id(codegen.StackName).Index(jen.Empty(), jen.Len(jen.Id(codegen.StackName)).Op("-").Lit(1)),
				// Check entry type: StackEntryPerCaptureRestore = capture restore, StackEntryAlt = Alt backtrack
				jen.If(jen.Id("last").Index(jen.Lit(2)).Op("==").Lit(StackEntryPerCaptureRestore)).Block(
					// Per-capture restore: captures[index] = oldValue
					jen.Id(codegen.CapturesName).Index(jen.Id("last").Index(jen.Lit(1))).Op("=").Id("last").Index(jen.Lit(0)),
					jen.Continue(), // Keep processing stack
				),
				// Alt backtrack point (type=0): set offset and instruction, jump to selector
				jen.Id(codegen.OffsetName).Op("=").Id("last").Index(jen.Lit(0)),
				jen.Id(codegen.NextInstructionName).Op("=").Id("last").Index(jen.Lit(1)),
				jen.Goto().Id(codegen.StepSelectName),
			),
			// Stack empty - try next position or return no match
			jen.Block(retryBlock...),
		}
	}

	// Array checkpointing mode: restore entire array when type=1
	return []jen.Code{
		jen.Id(codegen.TryFallbackName).Op(":"),
		jen.If(jen.Len(jen.Id(codegen.StackName)).Op(">").Lit(0)).Block(
			jen.Id("last").Op(":=").Id(codegen.StackName).Index(jen.Len(jen.Id(codegen.StackName)).Op("-").Lit(1)),
			jen.Id(codegen.OffsetName).Op("=").Id("last").Index(jen.Lit(0)),
			jen.Id(codegen.NextInstructionName).Op("=").Id("last").Index(jen.Lit(1)),
			jen.Id(codegen.StackName).Op("=").Id(codegen.StackName).Index(jen.Empty(), jen.Len(jen.Id(codegen.StackName)).Op("-").Lit(1)),
			// Restore capture state from checkpoint stack only if this backtrack point had a checkpoint
			// (indicated by last[2] == StackEntryCheckpoint, selective checkpointing optimization)
			jen.If(jen.Id("last").Index(jen.Lit(2)).Op("==").Lit(StackEntryCheckpoint).Op("&&").Len(jen.Id("captureStack")).Op(">").Lit(0)).Block(
				jen.Id("n").Op(":=").Len(jen.Id(codegen.CapturesName)),
				jen.Id("top").Op(":=").Len(jen.Id("captureStack")).Op("-").Id("n"),
				jen.Copy(jen.Id(codegen.CapturesName).Index(jen.Empty(), jen.Empty()), jen.Id("captureStack").Index(jen.Id("top"), jen.Empty())),
				jen.Id("captureStack").Op("=").Id("captureStack").Index(jen.Empty(), jen.Id("top")),
			),
			jen.Goto().Id(codegen.StepSelectName),
		).Else().Block(retryBlock...),
	}
}
