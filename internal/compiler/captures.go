package compiler

import (
	"fmt"
	"regexp/syntax"

	"github.com/KromDaniel/regengo/internal/codegen"
	"github.com/dave/jennifer/jen"
)

// generateCaptureFunctions generates Find and FindAll methods with capture groups.
func (c *Compiler) generateCaptureFunctions() error {
	// Generate capture result struct
	structName := fmt.Sprintf("%sResult", c.config.Name)
	c.generateCaptureStruct(structName)

	// Generate FindString method
	if err := c.generateFindStringFunction(structName); err != nil {
		return fmt.Errorf("failed to generate FindString: %w", err)
	}

	// Generate FindAllString method
	if err := c.generateFindAllStringFunction(structName); err != nil {
		return fmt.Errorf("failed to generate FindAllString: %w", err)
	}

	// Generate FindBytes method
	if c.config.BytesView {
		bytesStructName := fmt.Sprintf("%sResultBytes", c.config.Name)
		c.generateCaptureStructBytes(bytesStructName)
		if err := c.generateFindBytesFunction(bytesStructName); err != nil {
			return fmt.Errorf("failed to generate FindBytes: %w", err)
		}
		if err := c.generateFindAllBytesFunction(bytesStructName); err != nil {
			return fmt.Errorf("failed to generate FindAllBytes: %w", err)
		}
	} else {
		// Use same struct, convert []byte to string
		if err := c.generateFindBytesFunction(structName); err != nil {
			return fmt.Errorf("failed to generate FindBytes: %w", err)
		}
		if err := c.generateFindAllBytesFunction(structName); err != nil {
			return fmt.Errorf("failed to generate FindAllBytes: %w", err)
		}
	}

	return nil
}

// generateCaptureStruct generates the Match struct with string fields.
func (c *Compiler) generateCaptureStruct(structName string) {
	// Add warning comment if there are repeating captures
	if c.hasRepeatingCaptures {
		c.file.Comment("Note: This pattern contains capture groups in repeating/optional context.")
		c.file.Comment("Go's regex engine captures only the LAST match from repeating groups (* + {n,m}).")
		c.file.Comment("For example: (\\w)+ matching 'abc' captures 'c', not ['a','b','c'].")
		c.file.Comment("Optional groups (?) return empty string when not matched.")
		c.file.Line()
	}

	fields := []jen.Code{
		jen.Id("Match").String().Comment("Full match"),
	}

	// Add fields for each capture group (skip group 0 which is the full match)
	for i := 1; i < len(c.captureNames); i++ {
		fieldName := c.captureNames[i]
		if fieldName == "" {
			fieldName = fmt.Sprintf("Group%d", i)
		} else {
			fieldName = codegen.UpperFirst(fieldName)
		}
		fields = append(fields, jen.Id(fieldName).String())
	}

	c.file.Type().Id(structName).Struct(fields...)
	c.file.Line()
}

// generateCaptureStructBytes generates the Match struct with []byte fields for BytesView.
func (c *Compiler) generateCaptureStructBytes(structName string) {
	// Add warning comment if there are repeating captures
	if c.hasRepeatingCaptures {
		c.file.Comment("Note: This pattern contains capture groups in repeating/optional context.")
		c.file.Comment("Go's regex engine captures only the LAST match from repeating groups (* + {n,m}).")
		c.file.Comment("For example: (\\w)+ matching 'abc' captures 'c', not ['a','b','c'].")
		c.file.Comment("Optional groups (?) return empty slice when not matched.")
		c.file.Line()
	}

	fields := []jen.Code{
		jen.Id("Match").Index().Byte().Comment("Full match"),
	}

	// Add fields for each capture group (skip group 0 which is the full match)
	for i := 1; i < len(c.captureNames); i++ {
		fieldName := c.captureNames[i]
		if fieldName == "" {
			fieldName = fmt.Sprintf("Group%d", i)
		} else {
			fieldName = codegen.UpperFirst(fieldName)
		}
		fields = append(fields, jen.Id(fieldName).Index().Byte())
	}

	c.file.Type().Id(structName).Struct(fields...)
	c.file.Line()
}

// generateCaptureInst generates code for InstCapture (record capture position).
func (c *Compiler) generateCaptureInst(label *jen.Statement, inst *syntax.Inst) ([]jen.Code, error) {
	return []jen.Code{
		label,
		jen.Block(
			jen.Id(codegen.CapturesName).Index(jen.Lit(int(inst.Arg))).Op("=").Id(codegen.OffsetName),
			jen.Id(codegen.NextInstructionName).Op("=").Lit(int(inst.Out)),
			jen.Goto().Id(codegen.StepSelectName),
		),
	}, nil
}
