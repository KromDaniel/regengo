package compiler

import (
	"fmt"

	"github.com/KromDaniel/regengo/internal/codegen"
	"github.com/dave/jennifer/jen"
)

// generateStackPool generates a sync.Pool for stack reuse.
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

// generatePooledStackInit generates code to get a stack from the pool.
func (c *Compiler) generatePooledStackInit() []jen.Code {
	poolName := fmt.Sprintf("%sStackPool", codegen.LowerFirst(c.config.Name))

	return []jen.Code{
		// Get stack from pool
		jen.Id("stackPtr").Op(":=").Id(poolName).Dot("Get").Call().Assert(jen.Op("*").Index().Index(jen.Lit(2)).Int()),
		jen.Id(codegen.StackName).Op(":=").Parens(jen.Op("*").Id("stackPtr")).Index(jen.Empty(), jen.Lit(0)),
		// Defer return to pool
		jen.Defer().Func().Params().Block(
			// Clear references to prevent memory leaks
			jen.For(jen.Id("i").Op(":=").Range().Id(codegen.StackName)).Block(
				jen.Id(codegen.StackName).Index(jen.Id("i")).Op("=").Index(jen.Lit(2)).Int().Values(jen.Lit(0), jen.Lit(0)),
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
