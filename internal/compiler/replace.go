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
		fieldName := c.getCaptureFieldName(i)
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
		fieldName := c.getCaptureFieldName(i)
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
			jen.List(jen.Id("tmpl"), jen.Id("err")).Op(":=").Qual("github.com/KromDaniel/regengo/replace", "Parse").Call(jen.Id("template")),
			jen.If(jen.Id("err").Op("!=").Nil()).Block(
				jen.Panic(jen.Qual("fmt", "Sprintf").Call(jen.Lit("regengo: invalid replace template: %v"), jen.Id("err"))),
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
			jen.List(jen.Id("tmpl"), jen.Id("err")).Op(":=").Qual("github.com/KromDaniel/regengo/replace", "Parse").Call(jen.Id("template")),
			jen.If(jen.Id("err").Op("!=").Nil()).Block(
				jen.Panic(jen.Qual("fmt", "Sprintf").Call(jen.Lit("regengo: invalid replace template: %v"), jen.Id("err"))),
			),
			jen.Line(),

			// Initialize - use buf[:0] to reset length while keeping capacity for zero-alloc
			jen.Id("result").Op(":=").Id("buf").Index(jen.Op(":").Lit(0)),
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
			jen.List(jen.Id("tmpl"), jen.Id("err")).Op(":=").Qual("github.com/KromDaniel/regengo/replace", "Parse").Call(jen.Id("template")),
			jen.If(jen.Id("err").Op("!=").Nil()).Block(
				jen.Panic(jen.Qual("fmt", "Sprintf").Call(jen.Lit("regengo: invalid replace template: %v"), jen.Id("err"))),
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
			jen.List(jen.Id("tmpl"), jen.Id("err")).Op(":=").Qual("github.com/KromDaniel/regengo/replace", "Parse").Call(jen.Id("template")),
			jen.If(jen.Id("err").Op("!=").Nil()).Block(
				jen.Panic(jen.Qual("fmt", "Sprintf").Call(jen.Lit("regengo: invalid replace template: %v"), jen.Id("err"))),
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
			jen.Case(jen.Qual("github.com/KromDaniel/regengo/replace", "SegmentLiteral")).Block(
				jen.Id("result").Dot("WriteString").Call(jen.Id("seg").Dot("Literal")),
			),
			jen.Case(jen.Qual("github.com/KromDaniel/regengo/replace", "SegmentFullMatch")).Block(
				jen.Id("result").Dot("WriteString").Call(matchVar.Clone().Dot("Match")),
			),
			jen.Case(jen.Qual("github.com/KromDaniel/regengo/replace", "SegmentCaptureIndex")).Block(
				jen.Id("result").Dot("WriteString").Call(matchVar.Clone().Dot("CaptureByIndex").Call(jen.Id("seg").Dot("CaptureIndex"))),
			),
			jen.Case(jen.Qual("github.com/KromDaniel/regengo/replace", "SegmentCaptureName")).Block(
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
			jen.Case(jen.Qual("github.com/KromDaniel/regengo/replace", "SegmentLiteral")).Block(
				jen.Id("result").Op("=").Append(jen.Id("result"), jen.Id("seg").Dot("Literal").Op("...")),
			),
			jen.Case(jen.Qual("github.com/KromDaniel/regengo/replace", "SegmentFullMatch")).Block(
				jen.Id("result").Op("=").Append(jen.Id("result"), matchVar.Clone().Dot("Match").Op("...")),
			),
			jen.Case(jen.Qual("github.com/KromDaniel/regengo/replace", "SegmentCaptureIndex")).Block(
				jen.Id("result").Op("=").Append(jen.Id("result"), matchVar.Clone().Dot("CaptureByIndex").Call(jen.Id("seg").Dot("CaptureIndex")).Op("...")),
			),
			jen.Case(jen.Qual("github.com/KromDaniel/regengo/replace", "SegmentCaptureName")).Block(
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
			fieldName := c.getCaptureFieldNameByName(name)
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
			fieldName := c.getCaptureFieldNameByName(name)
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

// generatePrecompiledReplaceMethods generates optimized Replace methods for pre-compiled templates.
// These have zero runtime template parsing and use direct struct field access.
func (c *Compiler) generatePrecompiledReplaceMethods() error {
	for i, tmpl := range c.parsedReplacers {
		// Generate ReplaceAllString{N}
		c.generatePrecompiledReplaceAllString(i, tmpl)

		// Generate ReplaceAllBytes{N}
		c.generatePrecompiledReplaceAllBytes(i, tmpl)

		// Generate ReplaceAllBytesAppend{N}
		c.generatePrecompiledReplaceAllBytesAppend(i, tmpl)

		// Generate ReplaceFirstString{N}
		c.generatePrecompiledReplaceFirstString(i, tmpl)

		// Generate ReplaceFirstBytes{N}
		c.generatePrecompiledReplaceFirstBytes(i, tmpl)
	}

	return nil
}

// generatePrecompiledReplaceAllString generates an optimized ReplaceAllString{N} method.
func (c *Compiler) generatePrecompiledReplaceAllString(index int, tmpl *ParsedTemplate) {
	structName := fmt.Sprintf("%sResult", c.config.Name)
	methodName := fmt.Sprintf("ReplaceAllString%d", index)

	c.file.Comment(fmt.Sprintf("%s replaces all matches with pre-compiled template: %s", methodName, tmpl.Original))
	c.method(methodName).
		Params(jen.Id("input").String()).
		Params(jen.String()).
		Block(
			jen.Var().Id("result").Qual("strings", "Builder"),
			jen.Id("lastEnd").Op(":=").Lit(0),
			jen.Var().Id("r").Id(structName),
			jen.Line(),

			jen.Id("remaining").Op(":=").Id("input"),
			jen.Id("offset").Op(":=").Lit(0),
			jen.Line(),

			jen.For().Block(
				jen.List(jen.Id("match"), jen.Id("ok")).Op(":=").Id(c.config.Name).Values().Dot("FindStringReuse").Call(
					jen.Id("remaining"),
					jen.Op("&").Id("r"),
				),
				jen.If(jen.Op("!").Id("ok")).Block(
					jen.Break(),
				),
				jen.Line(),

				jen.Id("matchIdx").Op(":=").Qual("strings", "Index").Call(jen.Id("remaining"), jen.Id("match").Dot("Match")),
				jen.If(jen.Id("matchIdx").Op("<").Lit(0)).Block(
					jen.Break(),
				),
				jen.Line(),

				jen.Id("matchStart").Op(":=").Id("offset").Op("+").Id("matchIdx"),
				jen.Id("matchEnd").Op(":=").Id("matchStart").Op("+").Len(jen.Id("match").Dot("Match")),
				jen.Line(),

				jen.Id("result").Dot("WriteString").Call(jen.Id("input").Index(jen.Id("lastEnd").Op(":").Id("matchStart"))),
				jen.Line(),

				// Inlined template expansion
				c.generateInlinedTemplateExpansionString(tmpl, jen.Id("match")),

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

			jen.Id("result").Dot("WriteString").Call(jen.Id("input").Index(jen.Id("lastEnd").Op(":"))),
			jen.Return(jen.Id("result").Dot("String").Call()),
		)
	c.file.Line()
}

// generatePrecompiledReplaceAllBytes generates an optimized ReplaceAllBytes{N} method.
func (c *Compiler) generatePrecompiledReplaceAllBytes(index int, tmpl *ParsedTemplate) {
	methodName := fmt.Sprintf("ReplaceAllBytes%d", index)
	appendMethodName := fmt.Sprintf("ReplaceAllBytesAppend%d", index)

	c.file.Comment(fmt.Sprintf("%s replaces all matches with pre-compiled template: %s", methodName, tmpl.Original))
	c.method(methodName).
		Params(jen.Id("input").Index().Byte()).
		Params(jen.Index().Byte()).
		Block(
			jen.Return(jen.Id(c.config.Name).Values().Dot(appendMethodName).Call(
				jen.Id("input"),
				jen.Nil(),
			)),
		)
	c.file.Line()
}

// generatePrecompiledReplaceAllBytesAppend generates an optimized ReplaceAllBytesAppend{N} method.
func (c *Compiler) generatePrecompiledReplaceAllBytesAppend(index int, tmpl *ParsedTemplate) {
	bytesStructName := fmt.Sprintf("%sBytesResult", c.config.Name)
	methodName := fmt.Sprintf("ReplaceAllBytesAppend%d", index)

	c.file.Comment(fmt.Sprintf("%s replaces all matches and appends to buf.", methodName))
	c.file.Comment(fmt.Sprintf("Pre-compiled template: %s", tmpl.Original))
	c.method(methodName).
		Params(
			jen.Id("input").Index().Byte(),
			jen.Id("buf").Index().Byte(),
		).
		Params(jen.Index().Byte()).
		Block(
			// Use buf[:0] to reset length while keeping capacity for zero-alloc
			jen.Id("result").Op(":=").Id("buf").Index(jen.Op(":").Lit(0)),
			jen.Id("lastEnd").Op(":=").Lit(0),
			jen.Var().Id("r").Id(bytesStructName),
			jen.Line(),

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

				jen.Id("result").Op("=").Append(jen.Id("result"), jen.Id("input").Index(jen.Id("lastEnd").Op(":").Id("matchStart")).Op("...")),
				jen.Line(),

				// Inlined template expansion
				c.generateInlinedTemplateExpansionBytes(tmpl, jen.Id("match")),

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

			jen.Id("result").Op("=").Append(jen.Id("result"), jen.Id("input").Index(jen.Id("lastEnd").Op(":")).Op("...")),
			jen.Return(jen.Id("result")),
		)
	c.file.Line()
}

// generatePrecompiledReplaceFirstString generates an optimized ReplaceFirstString{N} method.
func (c *Compiler) generatePrecompiledReplaceFirstString(index int, tmpl *ParsedTemplate) {
	structName := fmt.Sprintf("%sResult", c.config.Name)
	methodName := fmt.Sprintf("ReplaceFirstString%d", index)

	c.file.Comment(fmt.Sprintf("%s replaces only the first match with pre-compiled template: %s", methodName, tmpl.Original))
	c.method(methodName).
		Params(jen.Id("input").String()).
		Params(jen.String()).
		Block(
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

			// Inlined template expansion
			c.generateInlinedTemplateExpansionString(tmpl, jen.Id("match")),

			jen.Id("result").Dot("WriteString").Call(jen.Id("input").Index(jen.Id("matchIdx").Op("+").Len(jen.Id("match").Dot("Match")).Op(":"))),
			jen.Return(jen.Id("result").Dot("String").Call()),
		)
	c.file.Line()
}

// generatePrecompiledReplaceFirstBytes generates an optimized ReplaceFirstBytes{N} method.
func (c *Compiler) generatePrecompiledReplaceFirstBytes(index int, tmpl *ParsedTemplate) {
	bytesStructName := fmt.Sprintf("%sBytesResult", c.config.Name)
	methodName := fmt.Sprintf("ReplaceFirstBytes%d", index)

	c.file.Comment(fmt.Sprintf("%s replaces only the first match with pre-compiled template: %s", methodName, tmpl.Original))
	c.method(methodName).
		Params(jen.Id("input").Index().Byte()).
		Params(jen.Index().Byte()).
		Block(
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

			// Inlined template expansion
			c.generateInlinedTemplateExpansionBytes(tmpl, jen.Id("match")),

			jen.Id("result").Op("=").Append(jen.Id("result"), jen.Id("input").Index(jen.Id("matchIdx").Op("+").Len(jen.Id("match").Dot("Match")).Op(":")).Op("...")),
			jen.Return(jen.Id("result")),
		)
	c.file.Line()
}

// generateInlinedTemplateExpansionString generates inlined code to expand a pre-compiled template.
// Unlike runtime expansion, this doesn't loop - it generates direct code for each segment.
// Optimizations:
//   - Literal-only templates: combines all literals into a single WriteString call
//   - Full-match-only templates: only accesses match.Match, no capture field access
func (c *Compiler) generateInlinedTemplateExpansionString(tmpl *ParsedTemplate, matchVar *jen.Statement) jen.Code {
	// Optimization: literal-only templates use a single combined string
	if tmpl.IsLiteralOnly() {
		combined := tmpl.CombinedLiteral()
		if combined == "" {
			return jen.Comment("Empty template - literal only (optimized)")
		}
		return jen.Block(
			jen.Comment("Optimized: literal-only template"),
			jen.Id("result").Dot("WriteString").Call(jen.Lit(combined)),
		)
	}

	var stmts []jen.Code

	// Add optimization comment for full-match-only templates
	if tmpl.UsesOnlyFullMatch() {
		stmts = append(stmts, jen.Comment("Optimized: uses only full match, no capture field access"))
	}

	for _, seg := range tmpl.Segments {
		switch seg.Type {
		case SegmentLiteral:
			stmts = append(stmts, jen.Id("result").Dot("WriteString").Call(jen.Lit(seg.Literal)))
		case SegmentFullMatch:
			stmts = append(stmts, jen.Id("result").Dot("WriteString").Call(matchVar.Clone().Dot("Match")))
		case SegmentCaptureIndex:
			fieldName := c.getCaptureFieldName(seg.CaptureIndex)
			stmts = append(stmts, jen.Id("result").Dot("WriteString").Call(matchVar.Clone().Dot(fieldName)))
		case SegmentCaptureName:
			// Named captures were resolved to indices in parseAndValidateReplacers
			// This shouldn't happen, but handle it anyway
			fieldName := c.getCaptureFieldNameByName(seg.CaptureName)
			stmts = append(stmts, jen.Id("result").Dot("WriteString").Call(matchVar.Clone().Dot(fieldName)))
		}
	}

	if len(stmts) == 0 {
		return jen.Comment("Empty template")
	}

	// Return as a block of statements
	return jen.Block(stmts...)
}

// generateInlinedTemplateExpansionBytes generates inlined code to expand a pre-compiled template for bytes.
// Optimizations:
//   - Literal-only templates: combines all literals into a single append call
//   - Full-match-only templates: only accesses match.Match, no capture field access
func (c *Compiler) generateInlinedTemplateExpansionBytes(tmpl *ParsedTemplate, matchVar *jen.Statement) jen.Code {
	// Optimization: literal-only templates use a single combined string
	if tmpl.IsLiteralOnly() {
		combined := tmpl.CombinedLiteral()
		if combined == "" {
			return jen.Comment("Empty template - literal only (optimized)")
		}
		return jen.Block(
			jen.Comment("Optimized: literal-only template"),
			jen.Id("result").Op("=").Append(jen.Id("result"), jen.Lit(combined).Op("...")),
		)
	}

	var stmts []jen.Code

	// Add optimization comment for full-match-only templates
	if tmpl.UsesOnlyFullMatch() {
		stmts = append(stmts, jen.Comment("Optimized: uses only full match, no capture field access"))
	}

	for _, seg := range tmpl.Segments {
		switch seg.Type {
		case SegmentLiteral:
			stmts = append(stmts, jen.Id("result").Op("=").Append(jen.Id("result"), jen.Lit(seg.Literal).Op("...")))
		case SegmentFullMatch:
			stmts = append(stmts, jen.Id("result").Op("=").Append(jen.Id("result"), matchVar.Clone().Dot("Match").Op("...")))
		case SegmentCaptureIndex:
			fieldName := c.getCaptureFieldName(seg.CaptureIndex)
			stmts = append(stmts, jen.Id("result").Op("=").Append(jen.Id("result"), matchVar.Clone().Dot(fieldName).Op("...")))
		case SegmentCaptureName:
			fieldName := c.getCaptureFieldNameByName(seg.CaptureName)
			stmts = append(stmts, jen.Id("result").Op("=").Append(jen.Id("result"), matchVar.Clone().Dot(fieldName).Op("...")))
		}
	}

	if len(stmts) == 0 {
		return jen.Comment("Empty template")
	}

	return jen.Block(stmts...)
}

// getCaptureFieldName returns the struct field name for a capture group by index.
// This uses the same collision resolution logic as struct generation in find.go.
func (c *Compiler) getCaptureFieldName(captureIndex int) string {
	if captureIndex == 0 {
		return "Match"
	}

	// Need to compute field names with collision resolution
	usedNames := make(map[string]bool)
	usedNames["Match"] = true // Reserved for full match

	for i := 1; i < len(c.captureNames); i++ {
		fieldName := c.captureNames[i]
		if fieldName == "" {
			fieldName = fmt.Sprintf("Group%d", i)
		} else {
			fieldName = codegen.UpperFirst(fieldName)
		}
		// Handle collisions by adding group number suffix
		if usedNames[fieldName] {
			fieldName = fmt.Sprintf("%s%d", fieldName, i)
		}
		usedNames[fieldName] = true

		if i == captureIndex {
			return fieldName
		}
	}

	return fmt.Sprintf("Group%d", captureIndex)
}

// getCaptureFieldNameByName returns the struct field name for a named capture group.
// This uses the same collision resolution logic as struct generation in find.go.
func (c *Compiler) getCaptureFieldNameByName(captureName string) string {
	// Need to compute field names with collision resolution
	usedNames := make(map[string]bool)
	usedNames["Match"] = true // Reserved for full match

	for i := 1; i < len(c.captureNames); i++ {
		name := c.captureNames[i]
		if name == "" {
			fieldName := fmt.Sprintf("Group%d", i)
			usedNames[fieldName] = true
			continue
		}

		fieldName := codegen.UpperFirst(name)
		// Handle collisions by adding group number suffix
		if usedNames[fieldName] {
			fieldName = fmt.Sprintf("%s%d", fieldName, i)
		}
		usedNames[fieldName] = true

		if name == captureName {
			return fieldName
		}
	}

	// Fallback: just uppercase the name (shouldn't happen if validation is correct)
	return codegen.UpperFirst(captureName)
}

// generateCompiledTemplateAPI generates the compiled template struct and methods.
// This allows users to compile a template once and reuse it for multiple replacements.
func (c *Compiler) generateCompiledTemplateAPI() {
	c.generateCompiledTemplateStruct()
	c.generateCompileReplaceTemplateMethod()
	c.generateCompiledTemplateStringMethod()
	c.generateCompiledReplaceAllString()
	c.generateCompiledReplaceAllBytes()
	c.generateCompiledReplaceAllBytesAppend()
	c.generateCompiledReplaceFirstString()
	c.generateCompiledReplaceFirstBytes()
}

// generateCompiledTemplateStruct generates the struct to hold a compiled replace template.
func (c *Compiler) generateCompiledTemplateStruct() {
	structName := fmt.Sprintf("%sReplaceTemplate", c.config.Name)

	c.file.Comment(fmt.Sprintf("%s holds a pre-compiled replace template for the %s pattern.", structName, c.config.Name))
	c.file.Comment("Use CompileReplaceTemplate to create one, then call its Replace methods.")
	c.file.Type().Id(structName).Struct(
		jen.Id("original").String(),
		jen.Id("segments").Index().Qual("github.com/KromDaniel/regengo/replace", "Segment"),
	)
	c.file.Line()
}

// generateCompileReplaceTemplateMethod generates the method to compile a replace template.
func (c *Compiler) generateCompileReplaceTemplateMethod() {
	structName := fmt.Sprintf("%sReplaceTemplate", c.config.Name)

	// Build capture names map literal
	captureNamesMap := jen.Dict{}
	for i := 1; i < len(c.captureNames); i++ {
		name := c.captureNames[i]
		if name != "" {
			captureNamesMap[jen.Lit(name)] = jen.Lit(i)
		}
	}

	numCaptures := len(c.captureNames) - 1 // exclude full match

	c.file.Comment("CompileReplaceTemplate parses and validates a replace template.")
	c.file.Comment("Returns an error if the template syntax is invalid or references non-existent captures.")
	c.file.Comment("The compiled template can be reused for multiple replacements without re-parsing.")
	c.file.Comment("")
	c.file.Comment("Template syntax: $0 (full match), $1/$2 (by index), $name (by name), $$ (literal $)")
	c.method("CompileReplaceTemplate").
		Params(jen.Id("template").String()).
		Params(
			jen.Op("*").Id(structName),
			jen.Error(),
		).
		Block(
			// Parse template
			jen.List(jen.Id("tmpl"), jen.Id("err")).Op(":=").
				Qual("github.com/KromDaniel/regengo/replace", "Parse").Call(jen.Id("template")),
			jen.If(jen.Id("err").Op("!=").Nil()).Block(
				jen.Return(jen.Nil(), jen.Id("err")),
			),
			jen.Line(),

			// Build capture names map
			jen.Id("captureNames").Op(":=").Map(jen.String()).Int().Values(captureNamesMap),
			jen.Line(),

			// Validate and resolve
			jen.List(jen.Id("resolved"), jen.Id("err")).Op(":=").
				Id("tmpl").Dot("ValidateAndResolve").Call(
				jen.Id("captureNames"),
				jen.Lit(numCaptures),
			),
			jen.If(jen.Id("err").Op("!=").Nil()).Block(
				jen.Return(jen.Nil(), jen.Id("err")),
			),
			jen.Line(),

			jen.Return(
				jen.Op("&").Id(structName).Values(jen.Dict{
					jen.Id("original"): jen.Id("template"),
					jen.Id("segments"): jen.Id("resolved"),
				}),
				jen.Nil(),
			),
		)
	c.file.Line()
}

// generateCompiledTemplateStringMethod generates the String() method.
func (c *Compiler) generateCompiledTemplateStringMethod() {
	structName := fmt.Sprintf("%sReplaceTemplate", c.config.Name)

	c.file.Comment("String returns the original template string.")
	c.file.Func().Params(jen.Id("t").Op("*").Id(structName)).Id("String").
		Params().
		Params(jen.String()).
		Block(
			jen.Return(jen.Id("t").Dot("original")),
		)
	c.file.Line()
}

// generateCompiledReplaceAllString generates ReplaceAllString on the compiled template.
func (c *Compiler) generateCompiledReplaceAllString() {
	structName := fmt.Sprintf("%sReplaceTemplate", c.config.Name)
	resultStructName := fmt.Sprintf("%sResult", c.config.Name)

	c.file.Comment("ReplaceAllString replaces all matches in input using this compiled template.")
	c.file.Func().Params(jen.Id("t").Op("*").Id(structName)).Id("ReplaceAllString").
		Params(jen.Id("input").String()).
		Params(jen.String()).
		Block(
			jen.Var().Id("result").Qual("strings", "Builder"),
			jen.Id("lastEnd").Op(":=").Lit(0),
			jen.Var().Id("r").Id(resultStructName),
			jen.Line(),

			jen.Id("remaining").Op(":=").Id("input"),
			jen.Id("offset").Op(":=").Lit(0),
			jen.Line(),

			jen.For().Block(
				jen.List(jen.Id("match"), jen.Id("ok")).Op(":=").Id(c.config.Name).Values().Dot("FindStringReuse").Call(
					jen.Id("remaining"),
					jen.Op("&").Id("r"),
				),
				jen.If(jen.Op("!").Id("ok")).Block(
					jen.Break(),
				),
				jen.Line(),

				jen.Id("matchIdx").Op(":=").Qual("strings", "Index").Call(jen.Id("remaining"), jen.Id("match").Dot("Match")),
				jen.If(jen.Id("matchIdx").Op("<").Lit(0)).Block(
					jen.Break(),
				),
				jen.Line(),

				jen.Id("absMatchStart").Op(":=").Id("offset").Op("+").Id("matchIdx"),
				jen.Id("absMatchEnd").Op(":=").Id("absMatchStart").Op("+").Len(jen.Id("match").Dot("Match")),
				jen.Line(),

				jen.Id("result").Dot("WriteString").Call(jen.Id("input").Index(jen.Id("lastEnd").Op(":").Id("absMatchStart"))),
				jen.Line(),

				// Expand template using segments
				jen.For(jen.List(jen.Id("_"), jen.Id("seg")).Op(":=").Range().Id("t").Dot("segments")).Block(
					jen.Switch(jen.Id("seg").Dot("Type")).Block(
						jen.Case(jen.Qual("github.com/KromDaniel/regengo/replace", "SegmentLiteral")).Block(
							jen.Id("result").Dot("WriteString").Call(jen.Id("seg").Dot("Literal")),
						),
						jen.Case(jen.Qual("github.com/KromDaniel/regengo/replace", "SegmentFullMatch")).Block(
							jen.Id("result").Dot("WriteString").Call(jen.Id("match").Dot("Match")),
						),
						jen.Case(jen.Qual("github.com/KromDaniel/regengo/replace", "SegmentCaptureIndex")).Block(
							jen.Id("result").Dot("WriteString").Call(jen.Id("match").Dot("CaptureByIndex").Call(jen.Id("seg").Dot("CaptureIndex"))),
						),
					),
				),
				jen.Line(),

				jen.Id("lastEnd").Op("=").Id("absMatchEnd"),
				jen.Id("remaining").Op("=").Id("input").Index(jen.Id("absMatchEnd").Op(":")),
				jen.Id("offset").Op("=").Id("absMatchEnd"),
			),
			jen.Line(),

			jen.Id("result").Dot("WriteString").Call(jen.Id("input").Index(jen.Id("lastEnd").Op(":"))),
			jen.Return(jen.Id("result").Dot("String").Call()),
		)
	c.file.Line()
}

// generateCompiledReplaceAllBytes generates ReplaceAllBytes on the compiled template.
func (c *Compiler) generateCompiledReplaceAllBytes() {
	structName := fmt.Sprintf("%sReplaceTemplate", c.config.Name)
	resultStructName := fmt.Sprintf("%sBytesResult", c.config.Name)

	c.file.Comment("ReplaceAllBytes replaces all matches in input using this compiled template.")
	c.file.Func().Params(jen.Id("t").Op("*").Id(structName)).Id("ReplaceAllBytes").
		Params(jen.Id("input").Index().Byte()).
		Params(jen.Index().Byte()).
		Block(
			jen.Return(jen.Id("t").Dot("ReplaceAllBytesAppend").Call(jen.Id("input"), jen.Nil())),
		)
	c.file.Line()

	// Also need to generate ReplaceAllBytesAppend
	c.file.Comment("ReplaceAllBytesAppend replaces all matches and appends to buf.")
	c.file.Comment("If buf has sufficient capacity, no allocation occurs.")
	c.file.Func().Params(jen.Id("t").Op("*").Id(structName)).Id("ReplaceAllBytesAppend").
		Params(
			jen.Id("input").Index().Byte(),
			jen.Id("buf").Index().Byte(),
		).
		Params(jen.Index().Byte()).
		Block(
			jen.Id("result").Op(":=").Id("buf").Index(jen.Op(":").Lit(0)),
			jen.Id("lastEnd").Op(":=").Lit(0),
			jen.Var().Id("r").Id(resultStructName),
			jen.Line(),

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

				jen.Id("absMatchStart").Op(":=").Id("offset").Op("+").Id("matchIdx"),
				jen.Id("absMatchEnd").Op(":=").Id("absMatchStart").Op("+").Len(jen.Id("match").Dot("Match")),
				jen.Line(),

				jen.Id("result").Op("=").Append(jen.Id("result"), jen.Id("input").Index(jen.Id("lastEnd").Op(":").Id("absMatchStart")).Op("...")),
				jen.Line(),

				// Expand template using segments
				jen.For(jen.List(jen.Id("_"), jen.Id("seg")).Op(":=").Range().Id("t").Dot("segments")).Block(
					jen.Switch(jen.Id("seg").Dot("Type")).Block(
						jen.Case(jen.Qual("github.com/KromDaniel/regengo/replace", "SegmentLiteral")).Block(
							jen.Id("result").Op("=").Append(jen.Id("result"), jen.Id("seg").Dot("Literal").Op("...")),
						),
						jen.Case(jen.Qual("github.com/KromDaniel/regengo/replace", "SegmentFullMatch")).Block(
							jen.Id("result").Op("=").Append(jen.Id("result"), jen.Id("match").Dot("Match").Op("...")),
						),
						jen.Case(jen.Qual("github.com/KromDaniel/regengo/replace", "SegmentCaptureIndex")).Block(
							jen.Id("result").Op("=").Append(jen.Id("result"), jen.Id("match").Dot("CaptureByIndex").Call(jen.Id("seg").Dot("CaptureIndex")).Op("...")),
						),
					),
				),
				jen.Line(),

				jen.Id("lastEnd").Op("=").Id("absMatchEnd"),
				jen.Id("remaining").Op("=").Id("input").Index(jen.Id("absMatchEnd").Op(":")),
				jen.Id("offset").Op("=").Id("absMatchEnd"),
			),
			jen.Line(),

			jen.Id("result").Op("=").Append(jen.Id("result"), jen.Id("input").Index(jen.Id("lastEnd").Op(":")).Op("...")),
			jen.Return(jen.Id("result")),
		)
	c.file.Line()
}

// generateCompiledReplaceAllBytesAppend is included in generateCompiledReplaceAllBytes.
func (c *Compiler) generateCompiledReplaceAllBytesAppend() {
	// Already generated in generateCompiledReplaceAllBytes
}

// generateCompiledReplaceFirstString generates ReplaceFirstString on the compiled template.
func (c *Compiler) generateCompiledReplaceFirstString() {
	structName := fmt.Sprintf("%sReplaceTemplate", c.config.Name)
	resultStructName := fmt.Sprintf("%sResult", c.config.Name)

	c.file.Comment("ReplaceFirstString replaces only the first match using this compiled template.")
	c.file.Func().Params(jen.Id("t").Op("*").Id(structName)).Id("ReplaceFirstString").
		Params(jen.Id("input").String()).
		Params(jen.String()).
		Block(
			jen.Var().Id("r").Id(resultStructName),
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

			// Expand template
			jen.For(jen.List(jen.Id("_"), jen.Id("seg")).Op(":=").Range().Id("t").Dot("segments")).Block(
				jen.Switch(jen.Id("seg").Dot("Type")).Block(
					jen.Case(jen.Qual("github.com/KromDaniel/regengo/replace", "SegmentLiteral")).Block(
						jen.Id("result").Dot("WriteString").Call(jen.Id("seg").Dot("Literal")),
					),
					jen.Case(jen.Qual("github.com/KromDaniel/regengo/replace", "SegmentFullMatch")).Block(
						jen.Id("result").Dot("WriteString").Call(jen.Id("match").Dot("Match")),
					),
					jen.Case(jen.Qual("github.com/KromDaniel/regengo/replace", "SegmentCaptureIndex")).Block(
						jen.Id("result").Dot("WriteString").Call(jen.Id("match").Dot("CaptureByIndex").Call(jen.Id("seg").Dot("CaptureIndex"))),
					),
				),
			),
			jen.Line(),

			jen.Id("result").Dot("WriteString").Call(jen.Id("input").Index(jen.Id("matchIdx").Op("+").Len(jen.Id("match").Dot("Match")).Op(":"))),
			jen.Return(jen.Id("result").Dot("String").Call()),
		)
	c.file.Line()
}

// generateCompiledReplaceFirstBytes generates ReplaceFirstBytes on the compiled template.
func (c *Compiler) generateCompiledReplaceFirstBytes() {
	structName := fmt.Sprintf("%sReplaceTemplate", c.config.Name)
	resultStructName := fmt.Sprintf("%sBytesResult", c.config.Name)

	c.file.Comment("ReplaceFirstBytes replaces only the first match using this compiled template.")
	c.file.Func().Params(jen.Id("t").Op("*").Id(structName)).Id("ReplaceFirstBytes").
		Params(jen.Id("input").Index().Byte()).
		Params(jen.Index().Byte()).
		Block(
			jen.Var().Id("r").Id(resultStructName),
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

			// Expand template
			jen.For(jen.List(jen.Id("_"), jen.Id("seg")).Op(":=").Range().Id("t").Dot("segments")).Block(
				jen.Switch(jen.Id("seg").Dot("Type")).Block(
					jen.Case(jen.Qual("github.com/KromDaniel/regengo/replace", "SegmentLiteral")).Block(
						jen.Id("result").Op("=").Append(jen.Id("result"), jen.Id("seg").Dot("Literal").Op("...")),
					),
					jen.Case(jen.Qual("github.com/KromDaniel/regengo/replace", "SegmentFullMatch")).Block(
						jen.Id("result").Op("=").Append(jen.Id("result"), jen.Id("match").Dot("Match").Op("...")),
					),
					jen.Case(jen.Qual("github.com/KromDaniel/regengo/replace", "SegmentCaptureIndex")).Block(
						jen.Id("result").Op("=").Append(jen.Id("result"), jen.Id("match").Dot("CaptureByIndex").Call(jen.Id("seg").Dot("CaptureIndex")).Op("...")),
					),
				),
			),
			jen.Line(),

			jen.Id("result").Op("=").Append(jen.Id("result"), jen.Id("input").Index(jen.Id("matchIdx").Op("+").Len(jen.Id("match").Dot("Match")).Op(":")).Op("...")),
			jen.Return(jen.Id("result")),
		)
	c.file.Line()
}
