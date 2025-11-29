package compiler

import (
	"fmt"

	"github.com/KromDaniel/regengo/internal/codegen"
	"github.com/dave/jennifer/jen"
)

// generateReplaceMethods generates runtime Replace methods for patterns with captures.
// These methods accept a template string at runtime and perform replacements.
func (c *Compiler) generateReplaceMethods() error {
	// Generate helper method to get capture by index
	c.generateCaptureByIndexHelper()

	// Generate ReplaceAllString
	c.generateReplaceAllString()

	// Generate ReplaceAllBytes
	c.generateReplaceAllBytes()

	// Generate ReplaceAllBytesAppend
	c.generateReplaceAllBytesAppend()

	// Generate ReplaceFirstString
	c.generateReplaceFirstString()

	// Generate ReplaceFirstBytes
	c.generateReplaceFirstBytes()

	return nil
}

// generateCaptureByIndexHelper generates a helper method on the result struct
// to access captures by index (needed for runtime template expansion).
func (c *Compiler) generateCaptureByIndexHelper() {
	structName := fmt.Sprintf("%sResult", c.config.Name)
	bytesStructName := fmt.Sprintf("%sBytesResult", c.config.Name)

	// Generate for string result
	cases := []jen.Code{
		jen.Case(jen.Lit(0)).Block(jen.Return(jen.Id("r").Dot("Match"))),
	}

	for i := 1; i < len(c.captureNames); i++ {
		fieldName := c.captureNames[i]
		if fieldName == "" {
			fieldName = fmt.Sprintf("Group%d", i)
		} else {
			fieldName = codegen.UpperFirst(fieldName)
		}
		cases = append(cases, jen.Case(jen.Lit(i)).Block(jen.Return(jen.Id("r").Dot(fieldName))))
	}

	cases = append(cases, jen.Default().Block(jen.Return(jen.Lit(""))))

	c.file.Comment("CaptureByIndex returns the capture group value by its 0-based index.")
	c.file.Comment("Index 0 returns the full match, 1+ returns capture groups.")
	c.file.Func().Params(jen.Id("r").Op("*").Id(structName)).Id("CaptureByIndex").
		Params(jen.Id("idx").Int()).
		Params(jen.String()).
		Block(
			jen.Switch(jen.Id("idx")).Block(cases...),
		)
	c.file.Line()

	// Generate for bytes result
	bytesCases := []jen.Code{
		jen.Case(jen.Lit(0)).Block(jen.Return(jen.Id("r").Dot("Match"))),
	}

	for i := 1; i < len(c.captureNames); i++ {
		fieldName := c.captureNames[i]
		if fieldName == "" {
			fieldName = fmt.Sprintf("Group%d", i)
		} else {
			fieldName = codegen.UpperFirst(fieldName)
		}
		bytesCases = append(bytesCases, jen.Case(jen.Lit(i)).Block(jen.Return(jen.Id("r").Dot(fieldName))))
	}

	bytesCases = append(bytesCases, jen.Default().Block(jen.Return(jen.Nil())))

	c.file.Comment("CaptureByIndex returns the capture group value by its 0-based index.")
	c.file.Comment("Index 0 returns the full match, 1+ returns capture groups.")
	c.file.Func().Params(jen.Id("r").Op("*").Id(bytesStructName)).Id("CaptureByIndex").
		Params(jen.Id("idx").Int()).
		Params(jen.Index().Byte()).
		Block(
			jen.Switch(jen.Id("idx")).Block(bytesCases...),
		)
	c.file.Line()
}

// generateReplaceAllString generates ReplaceAllString method.
func (c *Compiler) generateReplaceAllString() {
	structName := fmt.Sprintf("%sResult", c.config.Name)

	c.file.Comment("ReplaceAllString replaces all matches in input with the template expansion.")
	c.file.Comment("Template syntax: $0 (full match), $1/$2 (by index), $name (by name), $$ (literal $)")
	c.method("ReplaceAllString").
		Params(
			jen.Id("input").String(),
			jen.Id("template").String(),
		).
		Params(jen.String()).
		Block(
			// Parse template
			jen.List(jen.Id("tmpl"), jen.Id("err")).Op(":=").Qual("github.com/KromDaniel/regengo/pkg/regengo/replace", "Parse").Call(jen.Id("template")),
			jen.If(jen.Id("err").Op("!=").Nil()).Block(
				jen.Return(jen.Id("input")), // On parse error, return input unchanged (consistent with stdlib)
			),
			jen.Line(),

			// Initialize result builder
			jen.Var().Id("result").Qual("strings", "Builder"),
			jen.Id("lastEnd").Op(":=").Lit(0),
			jen.Var().Id("r").Id(structName),
			jen.Line(),

			// Find all matches
			jen.Id("remaining").Op(":=").Id("input"),
			jen.Id("offset").Op(":=").Lit(0),
			jen.Line(),

			jen.For().Block(
				// Find next match
				jen.List(jen.Id("match"), jen.Id("ok")).Op(":=").Id(c.config.Name).Values().Dot("FindStringReuse").Call(
					jen.Id("remaining"),
					jen.Op("&").Id("r"),
				),
				jen.If(jen.Op("!").Id("ok")).Block(
					jen.Break(),
				),
				jen.Line(),

				// Find match position in remaining string
				jen.Id("matchIdx").Op(":=").Qual("strings", "Index").Call(jen.Id("remaining"), jen.Id("match").Dot("Match")),
				jen.If(jen.Id("matchIdx").Op("<").Lit(0)).Block(
					jen.Break(), // Should not happen
				),
				jen.Line(),

				// Calculate absolute positions
				jen.Id("matchStart").Op(":=").Id("offset").Op("+").Id("matchIdx"),
				jen.Id("matchEnd").Op(":=").Id("matchStart").Op("+").Len(jen.Id("match").Dot("Match")),
				jen.Line(),

				// Write prefix (non-matching part)
				jen.Id("result").Dot("WriteString").Call(jen.Id("input").Index(jen.Id("lastEnd").Op(":").Id("matchStart"))),
				jen.Line(),

				// Expand template
				c.generateTemplateExpansionString(jen.Id("match")),
				jen.Line(),

				// Update positions
				jen.Id("lastEnd").Op("=").Id("matchEnd"),
				jen.If(jen.Len(jen.Id("match").Dot("Match")).Op(">").Lit(0)).Block(
					jen.Id("remaining").Op("=").Id("input").Index(jen.Id("matchEnd").Op(":")),
					jen.Id("offset").Op("=").Id("matchEnd"),
				).Else().Block(
					// Empty match - advance by 1 to avoid infinite loop
					jen.If(jen.Id("matchEnd").Op("<").Len(jen.Id("input"))).Block(
						jen.Id("remaining").Op("=").Id("input").Index(jen.Id("matchEnd").Op("+").Lit(1).Op(":")),
						jen.Id("offset").Op("=").Id("matchEnd").Op("+").Lit(1),
					).Else().Block(
						jen.Break(),
					),
				),
			),
			jen.Line(),

			// Write suffix
			jen.Id("result").Dot("WriteString").Call(jen.Id("input").Index(jen.Id("lastEnd").Op(":"))),
			jen.Return(jen.Id("result").Dot("String").Call()),
		)
	c.file.Line()
}

// generateReplaceAllBytes generates ReplaceAllBytes method.
func (c *Compiler) generateReplaceAllBytes() {
	c.file.Comment("ReplaceAllBytes replaces all matches in input with the template expansion.")
	c.file.Comment("Template syntax: $0 (full match), $1/$2 (by index), $name (by name), $$ (literal $)")
	c.method("ReplaceAllBytes").
		Params(
			jen.Id("input").Index().Byte(),
			jen.Id("template").String(),
		).
		Params(jen.Index().Byte()).
		Block(
			jen.Return(jen.Id(c.config.Name).Values().Dot("ReplaceAllBytesAppend").Call(
				jen.Id("input"),
				jen.Id("template"),
				jen.Nil(),
			)),
		)
	c.file.Line()
}

// generateReplaceAllBytesAppend generates ReplaceAllBytesAppend method.
func (c *Compiler) generateReplaceAllBytesAppend() {
	bytesStructName := fmt.Sprintf("%sBytesResult", c.config.Name)

	c.file.Comment("ReplaceAllBytesAppend replaces all matches and appends to buf.")
	c.file.Comment("If buf has sufficient capacity, no allocation occurs.")
	c.file.Comment("Template syntax: $0 (full match), $1/$2 (by index), $name (by name), $$ (literal $)")
	c.method("ReplaceAllBytesAppend").
		Params(
			jen.Id("input").Index().Byte(),
			jen.Id("template").String(),
			jen.Id("buf").Index().Byte(),
		).
		Params(jen.Index().Byte()).
		Block(
			// Parse template
			jen.List(jen.Id("tmpl"), jen.Id("err")).Op(":=").Qual("github.com/KromDaniel/regengo/pkg/regengo/replace", "Parse").Call(jen.Id("template")),
			jen.If(jen.Id("err").Op("!=").Nil()).Block(
				jen.Return(jen.Append(jen.Id("buf"), jen.Id("input").Op("..."))),
			),
			jen.Line(),

			// Initialize
			jen.Id("result").Op(":=").Id("buf"),
			jen.Id("lastEnd").Op(":=").Lit(0),
			jen.Var().Id("r").Id(bytesStructName),
			jen.Line(),

			// Find all matches
			jen.Id("remaining").Op(":=").Id("input"),
			jen.Id("offset").Op(":=").Lit(0),
			jen.Line(),

			jen.For().Block(
				jen.List(jen.Id("match"), jen.Id("ok")).Op(":=").Id(c.config.Name).Values().Dot("FindBytesReuse").Call(
					jen.Id("remaining"),
					jen.Op("&").Id("r"),
				),
				jen.If(jen.Op("!").Id("ok")).Block(
					jen.Break(),
				),
				jen.Line(),

				jen.Id("matchIdx").Op(":=").Qual("bytes", "Index").Call(jen.Id("remaining"), jen.Id("match").Dot("Match")),
				jen.If(jen.Id("matchIdx").Op("<").Lit(0)).Block(
					jen.Break(),
				),
				jen.Line(),

				jen.Id("matchStart").Op(":=").Id("offset").Op("+").Id("matchIdx"),
				jen.Id("matchEnd").Op(":=").Id("matchStart").Op("+").Len(jen.Id("match").Dot("Match")),
				jen.Line(),

				// Append prefix
				jen.Id("result").Op("=").Append(jen.Id("result"), jen.Id("input").Index(jen.Id("lastEnd").Op(":").Id("matchStart")).Op("...")),
				jen.Line(),

				// Expand template
				c.generateTemplateExpansionBytes(jen.Id("match")),
				jen.Line(),

				jen.Id("lastEnd").Op("=").Id("matchEnd"),
				jen.If(jen.Len(jen.Id("match").Dot("Match")).Op(">").Lit(0)).Block(
					jen.Id("remaining").Op("=").Id("input").Index(jen.Id("matchEnd").Op(":")),
					jen.Id("offset").Op("=").Id("matchEnd"),
				).Else().Block(
					jen.If(jen.Id("matchEnd").Op("<").Len(jen.Id("input"))).Block(
						jen.Id("remaining").Op("=").Id("input").Index(jen.Id("matchEnd").Op("+").Lit(1).Op(":")),
						jen.Id("offset").Op("=").Id("matchEnd").Op("+").Lit(1),
					).Else().Block(
						jen.Break(),
					),
				),
			),
			jen.Line(),

			// Append suffix
			jen.Id("result").Op("=").Append(jen.Id("result"), jen.Id("input").Index(jen.Id("lastEnd").Op(":")).Op("...")),
			jen.Return(jen.Id("result")),
		)
	c.file.Line()
}

// generateReplaceFirstString generates ReplaceFirstString method.
func (c *Compiler) generateReplaceFirstString() {
	structName := fmt.Sprintf("%sResult", c.config.Name)

	c.file.Comment("ReplaceFirstString replaces only the first match in input with the template expansion.")
	c.file.Comment("Template syntax: $0 (full match), $1/$2 (by index), $name (by name), $$ (literal $)")
	c.method("ReplaceFirstString").
		Params(
			jen.Id("input").String(),
			jen.Id("template").String(),
		).
		Params(jen.String()).
		Block(
			jen.List(jen.Id("tmpl"), jen.Id("err")).Op(":=").Qual("github.com/KromDaniel/regengo/pkg/regengo/replace", "Parse").Call(jen.Id("template")),
			jen.If(jen.Id("err").Op("!=").Nil()).Block(
				jen.Return(jen.Id("input")),
			),
			jen.Line(),

			jen.Var().Id("r").Id(structName),
			jen.List(jen.Id("match"), jen.Id("ok")).Op(":=").Id(c.config.Name).Values().Dot("FindStringReuse").Call(
				jen.Id("input"),
				jen.Op("&").Id("r"),
			),
			jen.If(jen.Op("!").Id("ok")).Block(
				jen.Return(jen.Id("input")),
			),
			jen.Line(),

			jen.Id("matchIdx").Op(":=").Qual("strings", "Index").Call(jen.Id("input"), jen.Id("match").Dot("Match")),
			jen.If(jen.Id("matchIdx").Op("<").Lit(0)).Block(
				jen.Return(jen.Id("input")),
			),
			jen.Line(),

			jen.Var().Id("result").Qual("strings", "Builder"),
			jen.Id("result").Dot("WriteString").Call(jen.Id("input").Index(jen.Op(":").Id("matchIdx"))),
			jen.Line(),

			c.generateTemplateExpansionString(jen.Id("match")),
			jen.Line(),

			jen.Id("result").Dot("WriteString").Call(jen.Id("input").Index(jen.Id("matchIdx").Op("+").Len(jen.Id("match").Dot("Match")).Op(":"))),
			jen.Return(jen.Id("result").Dot("String").Call()),
		)
	c.file.Line()
}

// generateReplaceFirstBytes generates ReplaceFirstBytes method.
func (c *Compiler) generateReplaceFirstBytes() {
	bytesStructName := fmt.Sprintf("%sBytesResult", c.config.Name)

	c.file.Comment("ReplaceFirstBytes replaces only the first match in input with the template expansion.")
	c.file.Comment("Template syntax: $0 (full match), $1/$2 (by index), $name (by name), $$ (literal $)")
	c.method("ReplaceFirstBytes").
		Params(
			jen.Id("input").Index().Byte(),
			jen.Id("template").String(),
		).
		Params(jen.Index().Byte()).
		Block(
			jen.List(jen.Id("tmpl"), jen.Id("err")).Op(":=").Qual("github.com/KromDaniel/regengo/pkg/regengo/replace", "Parse").Call(jen.Id("template")),
			jen.If(jen.Id("err").Op("!=").Nil()).Block(
				jen.Return(jen.Append(jen.Index().Byte().Values(), jen.Id("input").Op("..."))),
			),
			jen.Line(),

			jen.Var().Id("r").Id(bytesStructName),
			jen.List(jen.Id("match"), jen.Id("ok")).Op(":=").Id(c.config.Name).Values().Dot("FindBytesReuse").Call(
				jen.Id("input"),
				jen.Op("&").Id("r"),
			),
			jen.If(jen.Op("!").Id("ok")).Block(
				jen.Return(jen.Append(jen.Index().Byte().Values(), jen.Id("input").Op("..."))),
			),
			jen.Line(),

			jen.Id("matchIdx").Op(":=").Qual("bytes", "Index").Call(jen.Id("input"), jen.Id("match").Dot("Match")),
			jen.If(jen.Id("matchIdx").Op("<").Lit(0)).Block(
				jen.Return(jen.Append(jen.Index().Byte().Values(), jen.Id("input").Op("..."))),
			),
			jen.Line(),

			jen.Var().Id("result").Index().Byte(),
			jen.Id("result").Op("=").Append(jen.Id("result"), jen.Id("input").Index(jen.Op(":").Id("matchIdx")).Op("...")),
			jen.Line(),

			c.generateTemplateExpansionBytes(jen.Id("match")),
			jen.Line(),

			jen.Id("result").Op("=").Append(jen.Id("result"), jen.Id("input").Index(jen.Id("matchIdx").Op("+").Len(jen.Id("match").Dot("Match")).Op(":")).Op("...")),
			jen.Return(jen.Id("result")),
		)
	c.file.Line()
}

// generateTemplateExpansionString generates code to expand a template into result (strings.Builder).
func (c *Compiler) generateTemplateExpansionString(matchVar *jen.Statement) jen.Code {
	return jen.For(jen.List(jen.Id("_"), jen.Id("seg")).Op(":=").Range().Id("tmpl").Dot("Segments")).Block(
		jen.Switch(jen.Id("seg").Dot("Type")).Block(
			jen.Case(jen.Qual("github.com/KromDaniel/regengo/pkg/regengo/replace", "SegmentLiteral")).Block(
				jen.Id("result").Dot("WriteString").Call(jen.Id("seg").Dot("Literal")),
			),
			jen.Case(jen.Qual("github.com/KromDaniel/regengo/pkg/regengo/replace", "SegmentFullMatch")).Block(
				jen.Id("result").Dot("WriteString").Call(matchVar.Clone().Dot("Match")),
			),
			jen.Case(jen.Qual("github.com/KromDaniel/regengo/pkg/regengo/replace", "SegmentCaptureIndex")).Block(
				jen.Id("result").Dot("WriteString").Call(matchVar.Clone().Dot("CaptureByIndex").Call(jen.Id("seg").Dot("CaptureIndex"))),
			),
			jen.Case(jen.Qual("github.com/KromDaniel/regengo/pkg/regengo/replace", "SegmentCaptureName")).Block(
				// For named captures, we need to look up the index
				// This is handled by generating a name->index map or switch
				c.generateNamedCaptureLookupString(matchVar),
			),
		),
	)
}

// generateTemplateExpansionBytes generates code to expand a template into result ([]byte).
func (c *Compiler) generateTemplateExpansionBytes(matchVar *jen.Statement) jen.Code {
	return jen.For(jen.List(jen.Id("_"), jen.Id("seg")).Op(":=").Range().Id("tmpl").Dot("Segments")).Block(
		jen.Switch(jen.Id("seg").Dot("Type")).Block(
			jen.Case(jen.Qual("github.com/KromDaniel/regengo/pkg/regengo/replace", "SegmentLiteral")).Block(
				jen.Id("result").Op("=").Append(jen.Id("result"), jen.Id("seg").Dot("Literal").Op("...")),
			),
			jen.Case(jen.Qual("github.com/KromDaniel/regengo/pkg/regengo/replace", "SegmentFullMatch")).Block(
				jen.Id("result").Op("=").Append(jen.Id("result"), matchVar.Clone().Dot("Match").Op("...")),
			),
			jen.Case(jen.Qual("github.com/KromDaniel/regengo/pkg/regengo/replace", "SegmentCaptureIndex")).Block(
				jen.Id("result").Op("=").Append(jen.Id("result"), matchVar.Clone().Dot("CaptureByIndex").Call(jen.Id("seg").Dot("CaptureIndex")).Op("...")),
			),
			jen.Case(jen.Qual("github.com/KromDaniel/regengo/pkg/regengo/replace", "SegmentCaptureName")).Block(
				c.generateNamedCaptureLookupBytes(matchVar),
			),
		),
	)
}

// generateNamedCaptureLookupString generates code to look up a named capture.
func (c *Compiler) generateNamedCaptureLookupString(matchVar *jen.Statement) jen.Code {
	// Generate a switch statement over capture names
	cases := []jen.Code{}

	for i := 1; i < len(c.captureNames); i++ {
		name := c.captureNames[i]
		if name != "" {
			fieldName := codegen.UpperFirst(name)
			cases = append(cases, jen.Case(jen.Lit(name)).Block(
				jen.Id("result").Dot("WriteString").Call(matchVar.Clone().Dot(fieldName)),
			))
		}
	}

	// If no named captures, just ignore unknown names
	if len(cases) == 0 {
		return jen.Comment("No named captures in pattern")
	}

	return jen.Switch(jen.Id("seg").Dot("CaptureName")).Block(cases...)
}

// generateNamedCaptureLookupBytes generates code to look up a named capture for bytes.
func (c *Compiler) generateNamedCaptureLookupBytes(matchVar *jen.Statement) jen.Code {
	cases := []jen.Code{}

	for i := 1; i < len(c.captureNames); i++ {
		name := c.captureNames[i]
		if name != "" {
			fieldName := codegen.UpperFirst(name)
			cases = append(cases, jen.Case(jen.Lit(name)).Block(
				jen.Id("result").Op("=").Append(jen.Id("result"), matchVar.Clone().Dot(fieldName).Op("...")),
			))
		}
	}

	if len(cases) == 0 {
		return jen.Comment("No named captures in pattern")
	}

	return jen.Switch(jen.Id("seg").Dot("CaptureName")).Block(cases...)
}
