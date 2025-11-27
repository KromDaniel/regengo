package compiler

import (
	"fmt"

	"github.com/KromDaniel/regengo/internal/codegen"
	"github.com/dave/jennifer/jen"
)

// generateStackPool generates a sync.Pool for stack reuse (for Match functions).
func (c *Compiler) generateStackPool() {
	poolName := fmt.Sprintf("%sStackPool", codegen.LowerFirst(c.config.Name))

	c.file.Var().Id(poolName).Op("=").Qual("sync", "Pool").Values(jen.Dict{
		jen.Id("New"): jen.Func().Params().Interface().Block(
			jen.Id("stack").Op(":=").Make(jen.Index().Index(jen.Lit(2)).Int(), jen.Lit(0), jen.Lit(32)),
			jen.Return(jen.Op("&").Id("stack")),
		),
	})
	c.file.Line()
}

// generateStackPoolWithCaptures generates a sync.Pool for capture function stacks.
// Uses [3]int entries: [offset, instruction, hasCheckpoint] for selective checkpointing.
func (c *Compiler) generateStackPoolWithCaptures() {
	poolName := fmt.Sprintf("%sStackPool", codegen.LowerFirst(c.config.Name))

	c.file.Var().Id(poolName).Op("=").Qual("sync", "Pool").Values(jen.Dict{
		jen.Id("New"): jen.Func().Params().Interface().Block(
			jen.Id("stack").Op(":=").Make(jen.Index().Index(jen.Lit(3)).Int(), jen.Lit(0), jen.Lit(32)),
			jen.Return(jen.Op("&").Id("stack")),
		),
	})
	c.file.Line()
}

// generateCaptureStackPool generates the sync.Pool for capture stacks.
func (c *Compiler) generateCaptureStackPool() {
	poolName := fmt.Sprintf("%sCaptureStackPool", codegen.LowerFirst(c.config.Name))

	c.file.Var().Id(poolName).Op("=").Qual("sync", "Pool").Values(jen.Dict{
		jen.Id("New"): jen.Func().Params().Interface().Block(
			// Initial capacity: 16 checkpoints * numCaptures
			jen.Id("stack").Op(":=").Make(jen.Index().Int(), jen.Lit(0), jen.Lit(16*c.config.Program.NumCap)),
			jen.Return(jen.Op("&").Id("stack")),
		),
	})
	c.file.Line()
}

// generatePooledStackInit generates code to get a stack from the pool (for Match functions).
// Uses the same stack size as the pool (which may be [3]int if captures are enabled).
func (c *Compiler) generatePooledStackInit() []jen.Code {
	poolName := fmt.Sprintf("%sStackPool", codegen.LowerFirst(c.config.Name))

	// Determine stack entry size based on whether captures are enabled
	// When captures are enabled (including TNFA), we use [3]int for selective checkpointing
	// Only TDFA uses a completely different approach and doesn't need this
	stackSize := 2
	if c.config.WithCaptures && !c.useTDFAForCaptures {
		stackSize = 3
	}

	// Build the zero value for clearing
	var zeroValues []jen.Code
	for i := 0; i < stackSize; i++ {
		zeroValues = append(zeroValues, jen.Lit(0))
	}

	return []jen.Code{
		// Get stack from pool
		jen.Id("stackPtr").Op(":=").Id(poolName).Dot("Get").Call().Assert(jen.Op("*").Index().Index(jen.Lit(stackSize)).Int()),
		jen.Id(codegen.StackName).Op(":=").Parens(jen.Op("*").Id("stackPtr")).Index(jen.Empty(), jen.Lit(0)),
		// Defer return to pool
		jen.Defer().Func().Params().Block(
			// Clear references to prevent memory leaks
			jen.For(jen.Id("i").Op(":=").Range().Id(codegen.StackName)).Block(
				jen.Id(codegen.StackName).Index(jen.Id("i")).Op("=").Index(jen.Lit(stackSize)).Int().Values(zeroValues...),
			),
			jen.Op("*").Id("stackPtr").Op("=").Id(codegen.StackName).Index(jen.Empty(), jen.Lit(0)),
			jen.Id(poolName).Dot("Put").Call(jen.Id("stackPtr")),
		).Call(),
	}
}

// generatePooledStackInitWithCaptures generates code to get a stack from the pool (for Find functions).
// Uses [3]int entries for selective checkpointing.
func (c *Compiler) generatePooledStackInitWithCaptures() []jen.Code {
	poolName := fmt.Sprintf("%sStackPool", codegen.LowerFirst(c.config.Name))

	return []jen.Code{
		// Get stack from pool
		jen.Id("stackPtr").Op(":=").Id(poolName).Dot("Get").Call().Assert(jen.Op("*").Index().Index(jen.Lit(3)).Int()),
		jen.Id(codegen.StackName).Op(":=").Parens(jen.Op("*").Id("stackPtr")).Index(jen.Empty(), jen.Lit(0)),
		// Defer return to pool
		jen.Defer().Func().Params().Block(
			// Clear references to prevent memory leaks
			jen.For(jen.Id("i").Op(":=").Range().Id(codegen.StackName)).Block(
				jen.Id(codegen.StackName).Index(jen.Id("i")).Op("=").Index(jen.Lit(3)).Int().Values(jen.Lit(0), jen.Lit(0), jen.Lit(0)),
			),
			jen.Op("*").Id("stackPtr").Op("=").Id(codegen.StackName).Index(jen.Empty(), jen.Lit(0)),
			jen.Id(poolName).Dot("Put").Call(jen.Id("stackPtr")),
		).Call(),
	}
}

// generatePooledCaptureStackInit generates code to get a capture stack from the pool.
func (c *Compiler) generatePooledCaptureStackInit() []jen.Code {
	poolName := fmt.Sprintf("%sCaptureStackPool", codegen.LowerFirst(c.config.Name))

	return []jen.Code{
		// Get stack from pool
		jen.Id("captureStackPtr").Op(":=").Id(poolName).Dot("Get").Call().Assert(jen.Op("*").Index().Int()),
		jen.Id("captureStack").Op(":=").Parens(jen.Op("*").Id("captureStackPtr")).Index(jen.Empty(), jen.Lit(0)),
		// Defer return to pool
		jen.Defer().Func().Params().Block(
			// Clear references to prevent memory leaks (not strictly needed for []int but good practice if we change types later)
			// For []int we just need to reset length
			jen.Op("*").Id("captureStackPtr").Op("=").Id("captureStack").Index(jen.Empty(), jen.Lit(0)),
			jen.Id(poolName).Dot("Put").Call(jen.Id("captureStackPtr")),
		).Call(),
	}
}

// generateVisitedPool generates a sync.Pool for visited bit-vectors (for memoization).
func (c *Compiler) generateVisitedPool() {
	poolName := fmt.Sprintf("%sVisitedPool", codegen.LowerFirst(c.config.Name))

	c.file.Var().Id(poolName).Op("=").Qual("sync", "Pool").Values(jen.Dict{
		jen.Id("New"): jen.Func().Params().Interface().Block(
			// Return *[]uint32
			jen.Return(jen.New(jen.Index().Uint32())),
		),
	})
	c.file.Line()
}

// generatePooledVisitedInit generates code to get a visited bit-vector from the pool.
func (c *Compiler) generatePooledVisitedInit() []jen.Code {
	poolName := fmt.Sprintf("%sVisitedPool", codegen.LowerFirst(c.config.Name))
	numInst := len(c.config.Program.Inst)

	return []jen.Code{
		// Calculate required size: numInst * (l + 1) bits
		jen.Id("visitedSize").Op(":=").Lit(numInst).Op("*").Parens(jen.Id(codegen.InputLenName).Op("+").Lit(1)),
		jen.Id("visitedWords").Op(":=").Parens(jen.Id("visitedSize").Op("+").Lit(31)).Op("/").Lit(32),

		// Get from pool
		jen.Id("visitedPtr").Op(":=").Id(poolName).Dot("Get").Call().Assert(jen.Op("*").Index().Uint32()),
		jen.Id(codegen.VisitedName).Op(":=").Op("*").Id("visitedPtr"),

		// Resize if necessary
		jen.If(jen.Cap(jen.Id(codegen.VisitedName)).Op("<").Id("visitedWords")).Block(
			jen.Id(codegen.VisitedName).Op("=").Make(jen.Index().Uint32(), jen.Id("visitedWords")),
		).Else().Block(
			jen.Id(codegen.VisitedName).Op("=").Id(codegen.VisitedName).Index(jen.Empty(), jen.Id("visitedWords")),
			// Zero out the slice
			jen.For(jen.Id("i").Op(":=").Range().Id(codegen.VisitedName)).Block(
				jen.Id(codegen.VisitedName).Index(jen.Id("i")).Op("=").Lit(0),
			),
		),

		// Defer return to pool
		jen.Defer().Func().Params().Block(
			jen.Op("*").Id("visitedPtr").Op("=").Id(codegen.VisitedName),
			jen.Id(poolName).Dot("Put").Call(jen.Id("visitedPtr")),
		).Call(),
	}
}
