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
					jen.Id(codegen.NextInstructionName).Op("=").Lit(int(c.config.Program.Start)),
					jen.Goto().Id(codegen.StepSelectName),
				),
			)
		} else {
			retryBlock = append(retryBlock,
				jen.If(jen.Id(codegen.InputLenName).Op(">").Id(codegen.OffsetName)).Block(
					jen.Id(codegen.NextInstructionName).Op("=").Lit(int(c.config.Program.Start)),
					jen.Id(codegen.OffsetName).Op("++"),
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
// Uses capture checkpointing to avoid resetting all captures on every backtrack.
func (c *Compiler) generateBacktrackingWithCaptures() []jen.Code {
	retryBlock := []jen.Code{}
	// Optimization: If anchored, we don't retry at next offset
	if !c.isAnchored {
		retryBlock = append(retryBlock,
			jen.If(jen.Id(codegen.InputLenName).Op(">").Id(codegen.OffsetName)).Block(
				jen.Id(codegen.OffsetName).Op("++"),
				// Reset captures array for new match attempt
				jen.For(jen.Id("i").Op(":=").Range().Id(codegen.CapturesName)).Block(
					jen.Id(codegen.CapturesName).Index(jen.Id("i")).Op("=").Lit(0),
				),
				// Clear capture checkpoint stack
				jen.Id("captureStack").Op("=").Id("captureStack").Index(jen.Empty(), jen.Lit(0)),
				// Set capture[0] to mark start of match attempt (after incrementing offset)
				jen.Id(codegen.CapturesName).Index(jen.Lit(0)).Op("=").Id(codegen.OffsetName),
				jen.Id(codegen.NextInstructionName).Op("=").Lit(int(c.config.Program.Start)),
				jen.Goto().Id(codegen.StepSelectName),
			),
		)
	}
	retryBlock = append(retryBlock, jen.Return(jen.Nil(), jen.False()))

	return []jen.Code{
		jen.Id(codegen.TryFallbackName).Op(":"),
		jen.If(jen.Len(jen.Id(codegen.StackName)).Op(">").Lit(0)).Block(
			jen.Id("last").Op(":=").Id(codegen.StackName).Index(jen.Len(jen.Id(codegen.StackName)).Op("-").Lit(1)),
			jen.Id(codegen.OffsetName).Op("=").Id("last").Index(jen.Lit(0)),
			jen.Id(codegen.NextInstructionName).Op("=").Id("last").Index(jen.Lit(1)),
			jen.Id(codegen.StackName).Op("=").Id(codegen.StackName).Index(jen.Empty(), jen.Len(jen.Id(codegen.StackName)).Op("-").Lit(1)),
			// Restore capture state from checkpoint stack
			jen.If(jen.Len(jen.Id("captureStack")).Op(">").Lit(0)).Block(
				jen.Id("n").Op(":=").Len(jen.Id(codegen.CapturesName)),
				jen.Id("top").Op(":=").Len(jen.Id("captureStack")).Op("-").Id("n"),
				jen.Copy(jen.Id(codegen.CapturesName).Index(jen.Empty(), jen.Empty()), jen.Id("captureStack").Index(jen.Id("top"), jen.Empty())),
				jen.Id("captureStack").Op("=").Id("captureStack").Index(jen.Empty(), jen.Id("top")),
			),
			jen.Goto().Id(codegen.StepSelectName),
		).Else().Block(retryBlock...),
	}
}
