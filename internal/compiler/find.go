package compiler

import (
	"fmt"
	"regexp/syntax"

	"github.com/KromDaniel/regengo/internal/codegen"
	"github.com/dave/jennifer/jen"
)

// generateFindStringFunction generates the FindString method with captures.
func (c *Compiler) generateFindStringFunction(structName string) error {
	code, err := c.generateFindFunction(structName, false)
	if err != nil {
		return err
	}

	c.method("FindString").
		Params(jen.Id(codegen.InputName).String()).
		Params(jen.Op("*").Id(structName), jen.Bool()).
		Block(code...)

	return nil
}

// generateFindBytesFunction generates the FindBytes method with captures.
func (c *Compiler) generateFindBytesFunction(structName string) error {
	code, err := c.generateFindFunction(structName, true)
	if err != nil {
		return err
	}

	c.method("FindBytes").
		Params(jen.Id(codegen.InputName).Index().Byte()).
		Params(jen.Op("*").Id(structName), jen.Bool()).
		Block(code...)

	return nil
}

// generateFindAllStringFunction generates the FindAllString method with captures.
func (c *Compiler) generateFindAllStringFunction(structName string) error {
	// Generate FindAllStringAppend first
	code, err := c.generateFindAllAppendFunction(structName, false)
	if err != nil {
		return err
	}

	c.method("FindAllStringAppend").
		Params(jen.Id(codegen.InputName).String(), jen.Id("n").Int(), jen.Id("s").Index().Op("*").Id(structName)).
		Params(jen.Index().Op("*").Id(structName)).
		Block(code...)

	// Generate FindAllString that calls FindAllStringAppend
	c.file.Func().
		Params(jen.Id("r").Id(c.config.Name)).
		Id("FindAllString").
		Params(jen.Id(codegen.InputName).String(), jen.Id("n").Int()).
		Params(jen.Index().Op("*").Id(structName)).
		Block(
			jen.Return(jen.Id("r").Dot("FindAllStringAppend").Call(
				jen.Id(codegen.InputName),
				jen.Id("n"),
				jen.Nil(),
			)),
		)

	return nil
}

// generateFindAllBytesFunction generates the FindAllBytes method with captures.
func (c *Compiler) generateFindAllBytesFunction(structName string) error {
	// Generate FindAllBytesAppend first
	code, err := c.generateFindAllAppendFunction(structName, true)
	if err != nil {
		return err
	}

	c.method("FindAllBytesAppend").
		Params(jen.Id(codegen.InputName).Index().Byte(), jen.Id("n").Int(), jen.Id("s").Index().Op("*").Id(structName)).
		Params(jen.Index().Op("*").Id(structName)).
		Block(code...)

	// Generate FindAllBytes that calls FindAllBytesAppend
	c.file.Func().
		Params(jen.Id("r").Id(c.config.Name)).
		Id("FindAllBytes").
		Params(jen.Id(codegen.InputName).Index().Byte(), jen.Id("n").Int()).
		Params(jen.Index().Op("*").Id(structName)).
		Block(
			jen.Return(jen.Id("r").Dot("FindAllBytesAppend").Call(
				jen.Id(codegen.InputName),
				jen.Id("n"),
				jen.Nil(),
			)),
		)

	return nil
}

// generateFindAllAppendFunction generates the FindAllAppend logic with slice reuse.
func (c *Compiler) generateFindAllAppendFunction(structName string, isBytes bool) ([]jen.Code, error) {
	numCaptures := c.config.Program.NumCap

	// Enable capture checkpoint optimization
	c.generatingCaptures = true
	defer func() { c.generatingCaptures = false }()

	code := []jen.Code{
		// Handle n parameter
		jen.If(jen.Id("n").Op("==").Lit(0)).Block(
			jen.Return(jen.Id("s")),
		),
		// Use provided slice
		jen.Id("result").Op(":=").Id("s"),
		// Initialize length
		jen.Id(codegen.InputLenName).Op(":=").Len(jen.Id(codegen.InputName)),
		// Initialize search start position
		jen.Id("searchStart").Op(":=").Lit(0),
	}

	// Build the loop body statements
	loopBody := []jen.Code{
		// Check if we've found enough matches
		jen.If(jen.Id("n").Op(">").Lit(0).Op("&&").Len(jen.Id("result")).Op(">=").Id("n")).Block(
			jen.Break(),
		),
	}

	// Optimization: If anchored, we only run once at offset 0
	if c.isAnchored {
		loopBody = append(loopBody,
			jen.If(jen.Id("searchStart").Op(">").Lit(0)).Block(
				jen.Break(),
			),
		)
	}

	loopBody = append(loopBody,
		// Check if we've reached end of input
		jen.If(jen.Id("searchStart").Op(">=").Id(codegen.InputLenName)).Block(
			jen.Break(),
		),
		// Initialize offset
		jen.Id(codegen.OffsetName).Op(":=").Id("searchStart"),
		// Initialize captures array
		jen.Id(codegen.CapturesName).Op(":=").Make(jen.Index().Int(), jen.Lit(numCaptures)),
	)

	// Initialize capture checkpoint stack only if we have alternations (Optimization #1)
	if c.needsBacktracking {
		if c.config.UsePool {
			loopBody = append(loopBody, c.generatePooledCaptureStackInit()...)
		} else {
			loopBody = append(loopBody, jen.Id("captureStack").Op(":=").Make(jen.Index().Int(), jen.Lit(0), jen.Lit(16*numCaptures)))
		}
	}

	// Only add stack initialization if backtracking is needed
	if c.needsBacktracking {
		// Add stack initialization
		if c.config.UsePool {
			poolName := fmt.Sprintf("%sStackPool", codegen.LowerFirst(c.config.Name))
			loopBody = append(loopBody,
				jen.Id("stackPtr").Op(":=").Id(poolName).Dot("Get").Call().Assert(jen.Op("*").Index().Index(jen.Lit(2)).Int()),
				jen.Id(codegen.StackName).Op(":=").Parens(jen.Op("*").Id("stackPtr")).Index(jen.Empty(), jen.Lit(0)),
				jen.Defer().Func().Params().Block(
					jen.For(jen.Id("i").Op(":=").Range().Id(codegen.StackName)).Block(
						jen.Id(codegen.StackName).Index(jen.Id("i")).Op("=").Index(jen.Lit(2)).Int().Values(jen.Lit(0), jen.Lit(0)),
					),
					jen.Op("*").Id("stackPtr").Op("=").Id(codegen.StackName).Index(jen.Empty(), jen.Lit(0)),
					jen.Id(poolName).Dot("Put").Call(jen.Id("stackPtr")),
				).Call(),
			)
		} else {
			loopBody = append(loopBody,
				jen.Id(codegen.StackName).Op(":=").Make(jen.Index().Index(jen.Lit(2)).Int(), jen.Lit(0), jen.Lit(32)),
			)
		}
	}

	// Continue with matching logic
	loopBody = append(loopBody,
		// Set captures[0] to mark start of match attempt
		jen.Id(codegen.CapturesName).Index(jen.Lit(0)).Op("=").Id("searchStart"),
		// Initialize next instruction
		jen.Id(codegen.NextInstructionName).Op(":=").Lit(int(c.config.Program.Start)),
		jen.Goto().Id(codegen.StepSelectName),
	)

	// Add backtracking logic that continues to next position on failure
	// Only if backtracking is needed
	if c.needsBacktracking {
		loopBody = append(loopBody,
			jen.Id(codegen.TryFallbackName).Op(":"),
			jen.If(jen.Len(jen.Id(codegen.StackName)).Op(">").Lit(0)).Block(
				jen.Id("last").Op(":=").Id(codegen.StackName).Index(jen.Len(jen.Id(codegen.StackName)).Op("-").Lit(1)),
				jen.Id(codegen.OffsetName).Op("=").Id("last").Index(jen.Lit(0)),
				jen.Id(codegen.NextInstructionName).Op("=").Id("last").Index(jen.Lit(1)),
				jen.Id(codegen.StackName).Op("=").Id(codegen.StackName).Index(jen.Empty(), jen.Len(jen.Id(codegen.StackName)).Op("-").Lit(1)),
				jen.Goto().Id(codegen.StepSelectName),
			).Else().Block(
				// No match at this position, try next
				jen.Id("searchStart").Op("++"),
				jen.Continue(),
			),
		)
	} else {
		// For patterns without backtracking, just move to next position on failure
		loopBody = append(loopBody,
			jen.Id(codegen.TryFallbackName).Op(":"),
			jen.Id("searchStart").Op("++"),
			jen.Continue(),
		)
	}

	// Add step selector
	loopBody = append(loopBody, c.generateStepSelector()...)

	// Generate instructions with captures - with slice reuse
	instructions, err := c.generateInstructionsForFindAllAppend(structName, isBytes)
	if err != nil {
		return nil, err
	}
	loopBody = append(loopBody, instructions...)

	// Add the loop
	code = append(code,
		jen.For(jen.True()).Block(loopBody...),
		jen.Return(jen.Id("result")),
	)

	return code, nil
}

// generateInstructionsForFindAllAppend generates instructions with slice reuse for FindAllAppend.
func (c *Compiler) generateInstructionsForFindAllAppend(structName string, isBytes bool) ([]jen.Code, error) {
	var code []jen.Code

	for i, inst := range c.config.Program.Inst {
		var instCode []jen.Code
		var err error

		if inst.Op == syntax.InstMatch {
			// Special handling for Match instruction with slice reuse
			instCode, err = c.generateMatchInstForFindAllAppend(uint32(i), structName, isBytes)
		} else if inst.Op == syntax.InstFail {
			// Special handling for Fail instruction - goto fallback instead of return
			label := jen.Id(codegen.InstructionName(uint32(i))).Op(":")
			instCode = []jen.Code{
				label,
				jen.Block(jen.Goto().Id(codegen.TryFallbackName)),
			}
		} else {
			// Use regular instruction generation for other instructions
			instCode, err = c.generateInstructionWithCaptures(uint32(i), &inst, structName, isBytes)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to generate instruction %d: %w", i, err)
		}
		code = append(code, instCode...)
	}

	return code, nil
}

// generateMatchInstForFindAllAppend generates Match instruction with slice reuse.
func (c *Compiler) generateMatchInstForFindAllAppend(id uint32, structName string, isBytes bool) ([]jen.Code, error) {
	label := jen.Id(codegen.InstructionName(id)).Op(":")

	// Build field assignments for reusing existing element
	var fieldAssignments []jen.Code

	// Match field
	if isBytes {
		fieldAssignments = append(fieldAssignments,
			jen.Id("item").Dot("Match").Op("=").Id(codegen.InputName).Index(
				jen.Id(codegen.CapturesName).Index(jen.Lit(0)).Op(":").Id(codegen.CapturesName).Index(jen.Lit(1)),
			),
		)
	} else {
		fieldAssignments = append(fieldAssignments,
			jen.Id("item").Dot("Match").Op("=").String().Call(
				jen.Id(codegen.InputName).Index(
					jen.Id(codegen.CapturesName).Index(jen.Lit(0)).Op(":").Id(codegen.CapturesName).Index(jen.Lit(1)),
				),
			),
		)
	}

	// Add capture group fields
	for i := 1; i < len(c.captureNames); i++ {
		fieldName := c.captureNames[i]
		if fieldName == "" {
			fieldName = fmt.Sprintf("Group%d", i)
		} else {
			fieldName = codegen.UpperFirst(fieldName)
		}

		captureStart := i * 2
		captureEnd := i*2 + 1

		if isBytes {
			fieldAssignments = append(fieldAssignments,
				jen.If(
					jen.Id(codegen.CapturesName).Index(jen.Lit(captureStart)).Op("<=").Id(codegen.CapturesName).Index(jen.Lit(captureEnd)).
						Op("&&").Id(codegen.CapturesName).Index(jen.Lit(captureEnd)).Op("<=").Len(jen.Id(codegen.InputName)),
				).Block(
					jen.Id("item").Dot(fieldName).Op("=").Id(codegen.InputName).Index(
						jen.Id(codegen.CapturesName).Index(jen.Lit(captureStart)).Op(":").Id(codegen.CapturesName).Index(jen.Lit(captureEnd)),
					),
				).Else().Block(
					jen.Id("item").Dot(fieldName).Op("=").Nil(),
				),
			)
		} else {
			fieldAssignments = append(fieldAssignments,
				jen.If(
					jen.Id(codegen.CapturesName).Index(jen.Lit(captureStart)).Op("<=").Id(codegen.CapturesName).Index(jen.Lit(captureEnd)).
						Op("&&").Id(codegen.CapturesName).Index(jen.Lit(captureEnd)).Op("<=").Len(jen.Id(codegen.InputName)),
				).Block(
					jen.Id("item").Dot(fieldName).Op("=").String().Call(jen.Id(codegen.InputName).Index(
						jen.Id(codegen.CapturesName).Index(jen.Lit(captureStart)).Op(":").Id(codegen.CapturesName).Index(jen.Lit(captureEnd)),
					)),
				).Else().Block(
					jen.Id("item").Dot(fieldName).Op("=").Lit(""),
				),
			)
		}
	}

	// Build the block with slice reuse logic
	blockCode := []jen.Code{
		// Set captures[1] to mark end of match
		jen.Id(codegen.CapturesName).Index(jen.Lit(1)).Op("=").Id(codegen.OffsetName),
		// Declare item variable
		jen.Var().Id("item").Op("*").Id(structName),
		// Check if we can reuse from capacity
		jen.If(jen.Len(jen.Id("result")).Op("<").Cap(jen.Id("result"))).Block(
			// Extend slice and get existing pointer
			jen.Id("result").Op("=").Id("result").Index(jen.Empty(), jen.Len(jen.Id("result")).Op("+").Lit(1)),
			jen.Id("item").Op("=").Id("result").Index(jen.Len(jen.Id("result")).Op("-").Lit(1)),
			// If nil, allocate new
			jen.If(jen.Id("item").Op("==").Nil()).Block(
				jen.Id("item").Op("=").Op("&").Id(structName).Values(),
				jen.Id("result").Index(jen.Len(jen.Id("result")).Op("-").Lit(1)).Op("=").Id("item"),
			),
		).Else().Block(
			// Allocate new and append
			jen.Id("item").Op("=").Op("&").Id(structName).Values(),
			jen.Id("result").Op("=").Append(jen.Id("result"), jen.Id("item")),
		),
	}

	// Add field assignments
	blockCode = append(blockCode, fieldAssignments...)

	// Add search position update and continue
	blockCode = append(blockCode,
		// Move search position past this match
		jen.If(jen.Id(codegen.CapturesName).Index(jen.Lit(1)).Op(">").Id("searchStart")).Block(
			jen.Id("searchStart").Op("=").Id(codegen.CapturesName).Index(jen.Lit(1)),
		).Else().Block(
			// Prevent infinite loop on zero-width matches
			jen.Id("searchStart").Op("++"),
		),
		// Continue searching
		jen.Continue(),
	)

	return []jen.Code{
		label,
		jen.Block(blockCode...),
	}, nil
}

// generateFindFunction generates the main Find logic with captures.
func (c *Compiler) generateFindFunction(structName string, isBytes bool) ([]jen.Code, error) {
	numCaptures := c.config.Program.NumCap

	// Enable capture checkpoint optimization
	c.generatingCaptures = true
	defer func() { c.generatingCaptures = false }()

	code := []jen.Code{
		// Initialize length
		jen.Id(codegen.InputLenName).Op(":=").Len(jen.Id(codegen.InputName)),
		// Initialize offset
		jen.Id(codegen.OffsetName).Op(":=").Lit(0),
		// Initialize captures array
		jen.Id(codegen.CapturesName).Op(":=").Make(jen.Index().Int(), jen.Lit(numCaptures)),
	}

	// Initialize capture checkpoint stack only if we have alternations (Optimization #1)
	if c.needsBacktracking {
		if c.config.UsePool {
			code = append(code, c.generatePooledCaptureStackInit()...)
		} else {
			// Initial capacity: 16 checkpoints * numCaptures
			code = append(code, jen.Id("captureStack").Op(":=").Make(jen.Index().Int(), jen.Lit(0), jen.Lit(16*numCaptures)))
		}
	}

	// Only add stack initialization if backtracking is needed
	if c.needsBacktracking {
		// Add stack initialization (pooled or regular)
		if c.config.UsePool {
			code = append(code, c.generatePooledStackInit()...)
		} else {
			code = append(code,
				jen.Id(codegen.StackName).Op(":=").Make(jen.Index().Index(jen.Lit(2)).Int(), jen.Lit(0), jen.Lit(32)),
			)
		}
	}

	code = append(code,
		// Set captures[0] to mark start of first match attempt
		jen.Id(codegen.CapturesName).Index(jen.Lit(0)).Op("=").Lit(0),
		// Initialize next instruction
		jen.Id(codegen.NextInstructionName).Op(":=").Lit(int(c.config.Program.Start)),
		// Jump to step selector
		jen.Goto().Id(codegen.StepSelectName),
	)

	// Add backtracking logic (with nil return for captures)
	// Only if backtracking is needed
	if c.needsBacktracking {
		code = append(code, c.generateBacktrackingWithCaptures()...)
	} else {
		// For patterns without backtracking:
		fallback := []jen.Code{jen.Id(codegen.TryFallbackName).Op(":")}

		// If not anchored, we must retry at next offset
		if !c.isAnchored {
			fallback = append(fallback,
				jen.If(jen.Id(codegen.InputLenName).Op(">").Id(codegen.OffsetName)).Block(
					jen.Id(codegen.OffsetName).Op("++"),
					// Reset captures array for new match attempt
					jen.For(jen.Id("i").Op(":=").Range().Id(codegen.CapturesName)).Block(
						jen.Id(codegen.CapturesName).Index(jen.Id("i")).Op("=").Lit(0),
					),
					// Set capture[0] to mark start of match attempt
					jen.Id(codegen.CapturesName).Index(jen.Lit(0)).Op("=").Id(codegen.OffsetName),
					jen.Id(codegen.NextInstructionName).Op("=").Lit(int(c.config.Program.Start)),
					jen.Goto().Id(codegen.StepSelectName),
				),
			)
		}

		fallback = append(fallback, jen.Return(jen.Nil(), jen.False()))
		code = append(code, fallback...)
	}

	// Add step selector
	code = append(code, c.generateStepSelector()...)

	// Generate instructions with captures
	instructions, err := c.generateInstructionsWithCaptures(structName, isBytes)
	if err != nil {
		return nil, err
	}
	code = append(code, instructions...)

	return code, nil
}

// generateInstructionsWithCaptures generates code for all instructions with capture support.
func (c *Compiler) generateInstructionsWithCaptures(structName string, isBytes bool) ([]jen.Code, error) {
	var code []jen.Code

	for i, inst := range c.config.Program.Inst {
		instCode, err := c.generateInstructionWithCaptures(uint32(i), &inst, structName, isBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to generate instruction %d: %w", i, err)
		}
		code = append(code, instCode...)
	}

	return code, nil
}

// generateInstructionWithCaptures generates code for a single instruction with capture support.
func (c *Compiler) generateInstructionWithCaptures(id uint32, inst *syntax.Inst, structName string, isBytes bool) ([]jen.Code, error) {
	label := jen.Id(codegen.InstructionName(id)).Op(":")

	switch inst.Op {
	case syntax.InstMatch:
		return c.generateMatchInstWithCaptures(label, structName, isBytes)

	case syntax.InstCapture:
		return c.generateCaptureInst(label, inst)

	case syntax.InstFail:
		return []jen.Code{
			label,
			jen.Block(jen.Return(jen.Nil(), jen.False())),
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

// generateMatchInstWithCaptures generates code for InstMatch with capture extraction.
func (c *Compiler) generateMatchInstWithCaptures(label *jen.Statement, structName string, isBytes bool) ([]jen.Code, error) {
	// Build struct fields from captures array
	structFields := jen.Dict{}

	if isBytes {
		// Zero-copy []byte view
		structFields[jen.Id("Match")] = jen.Id(codegen.InputName).Index(
			jen.Id(codegen.CapturesName).Index(jen.Lit(0)).Op(":").Id(codegen.CapturesName).Index(jen.Lit(1)),
		)
	} else {
		// String (may need conversion from []byte)
		structFields[jen.Id("Match")] = jen.String().Call(
			jen.Id(codegen.InputName).Index(
				jen.Id(codegen.CapturesName).Index(jen.Lit(0)).Op(":").Id(codegen.CapturesName).Index(jen.Lit(1)),
			),
		)
	}

	// Add capture group fields (skip group 0)
	for i := 1; i < len(c.captureNames); i++ {
		fieldName := c.captureNames[i]
		if fieldName == "" {
			fieldName = fmt.Sprintf("Group%d", i)
		} else {
			fieldName = codegen.UpperFirst(fieldName)
		}

		captureStart := i * 2
		captureEnd := i*2 + 1

		// Generate a ternary-like expression using function call to handle invalid captures
		// If start > end (invalid capture), use empty string/slice
		if isBytes {
			structFields[jen.Id(fieldName)] = jen.Func().Params().Index().Byte().Block(
				jen.If(
					jen.Id(codegen.CapturesName).Index(jen.Lit(captureStart)).Op("<=").Id(codegen.CapturesName).Index(jen.Lit(captureEnd)).
						Op("&&").Id(codegen.CapturesName).Index(jen.Lit(captureEnd)).Op("<=").Len(jen.Id(codegen.InputName)),
				).Block(
					jen.Return(jen.Id(codegen.InputName).Index(
						jen.Id(codegen.CapturesName).Index(jen.Lit(captureStart)).Op(":").Id(codegen.CapturesName).Index(jen.Lit(captureEnd)),
					)),
				),
				jen.Return(jen.Nil()),
			).Call()
		} else {
			structFields[jen.Id(fieldName)] = jen.Func().Params().String().Block(
				jen.If(
					jen.Id(codegen.CapturesName).Index(jen.Lit(captureStart)).Op("<=").Id(codegen.CapturesName).Index(jen.Lit(captureEnd)).
						Op("&&").Id(codegen.CapturesName).Index(jen.Lit(captureEnd)).Op("<=").Len(jen.Id(codegen.InputName)),
				).Block(
					jen.Return(jen.String().Call(jen.Id(codegen.InputName).Index(
						jen.Id(codegen.CapturesName).Index(jen.Lit(captureStart)).Op(":").Id(codegen.CapturesName).Index(jen.Lit(captureEnd)),
					))),
				),
				jen.Return(jen.Lit("")),
			).Call()
		}
	}

	return []jen.Code{
		label,
		jen.Block(
			// Set captures[1] to mark end of match
			jen.Id(codegen.CapturesName).Index(jen.Lit(1)).Op("=").Id(codegen.OffsetName),
			jen.Return(
				jen.Op("&").Id(structName).Values(structFields),
				jen.True(),
			),
		),
	}, nil
}
