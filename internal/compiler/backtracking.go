package compiler

import (
	"github.com/KromDaniel/regengo/internal/codegen"
	"github.com/dave/jennifer/jen"
)

// retryOpts configures generateRetryBlock.
type retryOpts struct {
	withCaptures      bool // Find: reset captures and set captures[0] = attemptStart
	clearCaptureStack bool // Find with array checkpointing: captureStack = captureStack[:0]
	hasPrefix         bool // Match only: skip to the next required prefix byte with IndexByte
	prefix            byte
	isBytes           bool
}

// attemptStartInit returns the declaration of attemptStart (the start offset of
// the current unanchored match attempt). Anchored programs never retry, so they
// get no declaration (Go rejects unused variables).
func (c *Compiler) attemptStartInit() []jen.Code {
	if c.isAnchored {
		return nil
	}
	return []jen.Code{
		jen.Id(codegen.AttemptStartName).Op(":=").Id(codegen.OffsetName),
	}
}

// generateAdvanceRune emits code that advances variable v by one UTF-8 rune
// (one byte for ASCII). The caller guarantees v < l.
func (c *Compiler) generateAdvanceRune(v string, isBytes bool) jen.Code {
	decode := "DecodeRuneInString"
	if isBytes {
		decode = "DecodeRune"
	}
	return jen.If(jen.Id(codegen.InputName).Index(jen.Id(v)).Op("<").Qual("unicode/utf8", "RuneSelf")).Block(
		jen.Id(v).Op("++"),
	).Else().Block(
		jen.List(jen.Id("_"), jen.Id("attemptWidth")).Op(":=").Qual("unicode/utf8", decode).Call(
			jen.Id(codegen.InputName).Index(jen.Id(v).Op(":")),
		),
		jen.Id(v).Op("+=").Id("attemptWidth"),
	)
}

// generateRetryBlock emits the unanchored retry: the next attempt starts one
// rune after attemptStart (or at the next prefix byte) instead of at the offset
// where the last path died. It returns nil when the pattern is anchored. The
// caller appends the failure return.
func (c *Compiler) generateRetryBlock(o retryOpts) []jen.Code {
	if c.isAnchored {
		return nil
	}
	if o.withCaptures && o.hasPrefix {
		panic("generateRetryBlock: prefix skipping is only supported for Match")
	}

	attempt := jen.Id(codegen.AttemptStartName)
	clearVisited := func() jen.Code {
		if c.useMemoization {
			return jen.For(jen.Id("i").Op(":=").Range().Id(codegen.VisitedName)).Block(
				jen.Id(codegen.VisitedName).Index(jen.Id("i")).Op("=").Lit(0),
			)
		}
		return jen.Null()
	}
	start := jen.Id(codegen.NextInstructionName).Op("=").Lit(int(c.config.Program.Start))
	goStep := jen.Goto().Id(codegen.StepSelectName)

	if o.hasPrefix {
		pkg := "strings"
		if o.isBytes {
			pkg = "bytes"
		}
		indexCall := jen.Qual(pkg, "IndexByte").Call(
			jen.Id(codegen.InputName).Index(attempt.Clone().Op(":")),
			jen.Lit(o.prefix),
		)
		return []jen.Code{
			attempt.Clone().Op("++"),
			jen.If(attempt.Clone().Op("<").Id(codegen.InputLenName)).Block(
				jen.Id("idx").Op(":=").Add(indexCall),
				jen.If(jen.Id("idx").Op("==").Lit(-1)).Block(jen.Return(jen.False())),
				attempt.Clone().Op("+=").Id("idx"),
				jen.Id(codegen.OffsetName).Op("=").Add(attempt.Clone()),
				clearVisited(),
				start,
				goStep,
			),
		}
	}

	block := []jen.Code{
		c.generateAdvanceRune(codegen.AttemptStartName, o.isBytes),
		jen.Id(codegen.OffsetName).Op("=").Add(attempt.Clone()),
	}
	if o.withCaptures {
		block = append(block,
			// Reset captures array for the new match attempt
			jen.For(jen.Id("i").Op(":=").Range().Id(codegen.CapturesName)).Block(
				jen.Id(codegen.CapturesName).Index(jen.Id("i")).Op("=").Lit(0),
			),
		)
	}
	if o.clearCaptureStack {
		block = append(block, jen.Id("captureStack").Op("=").Id("captureStack").Index(jen.Empty(), jen.Lit(0)))
	}
	block = append(block, clearVisited())
	if o.withCaptures {
		block = append(block, jen.Id(codegen.CapturesName).Index(jen.Lit(0)).Op("=").Add(attempt.Clone()))
	}
	block = append(block, start, goStep)

	return []jen.Code{
		jen.If(attempt.Clone().Op("<").Id(codegen.InputLenName)).Block(block...),
	}
}

// generateBacktracking generates the backtracking logic.
func (c *Compiler) generateBacktracking(prefix byte, hasPrefix bool, isBytes bool) []jen.Code {
	retryBlock := c.generateRetryBlock(retryOpts{hasPrefix: hasPrefix, prefix: prefix, isBytes: isBytes})
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
func (c *Compiler) generateBacktrackingWithCaptures(isBytes bool) []jen.Code {
	retryBlock := c.generateRetryBlock(retryOpts{
		withCaptures:      true,
		clearCaptureStack: !c.usePerCaptureCheckpointing, // per-capture mode doesn't use captureStack
		isBytes:           isBytes,
	})
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
