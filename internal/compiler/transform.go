package compiler

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"
)

// generateTransformMethods generates the streaming transform methods:
// - NewTransformReader: low-level method with emit callback
// - ReplaceReader: convenience method for template-based replacement
// - SelectReader: keep matches satisfying predicate
// - RejectReader: remove matches satisfying predicate
func (c *Compiler) generateTransformMethods() {
	if !c.config.WithCaptures {
		// Transform methods require capture functions (FindBytes)
		return
	}

	c.generateNewTransformReader()
	c.generateReplaceReader()
	c.generateSelectReader()
	c.generateRejectReader()
}

// generateNewTransformReader generates the low-level transform method that returns an io.Reader.
func (c *Compiler) generateNewTransformReader() {
	bytesStructName := fmt.Sprintf("%sBytesResult", c.config.Name)
	streamPkg := "github.com/KromDaniel/regengo/stream"

	// Calculate default leftover based on pattern analysis
	defaultLeftover := 1 << 20 // 1MB for unbounded patterns
	if c.matchLength.MaxMatchLen != -1 {
		defaultLeftover = c.matchLength.MaxMatchLen * 10
		if defaultLeftover < 1024 {
			defaultLeftover = 1024
		}
		if defaultLeftover > 1<<20 {
			defaultLeftover = 1 << 20
		}
	}

	c.file.Comment("NewTransformReader returns an io.Reader that transforms matches in the input stream.")
	c.file.Comment("")
	c.file.Comment("The onMatch callback is called for each match. Use emit to output replacement bytes.")
	c.file.Comment("If emit is not called, the match is dropped (filter behavior).")
	c.file.Comment("Non-matching segments are automatically passed through to output.")
	c.file.Comment("")
	c.file.Comment("Example usage:")
	c.file.Comment("  r := pattern.NewTransformReader(input, stream.DefaultTransformConfig(),")
	c.file.Comment("      func(m *PatternBytesResult, emit func([]byte)) {")
	c.file.Comment("          emit([]byte(\"REDACTED\"))")
	c.file.Comment("      })")
	c.file.Comment("  io.Copy(os.Stdout, r)")
	c.method("NewTransformReader").
		Params(
			jen.Id("r").Qual("io", "Reader"),
			jen.Id("cfg").Qual(streamPkg, "TransformConfig"),
			jen.Id("onMatch").Func().Params(
				jen.Id("match").Op("*").Id(bytesStructName),
				jen.Id("emit").Func().Params(jen.Index().Byte()),
			),
		).
		Params(jen.Qual("io", "Reader")).
		Block(
			// Apply config defaults
			jen.If(jen.Id("cfg").Dot("BufferSize").Op("==").Lit(0)).Block(
				jen.Id("cfg").Dot("BufferSize").Op("=").Lit(64*1024),
			),
			jen.If(jen.Id("cfg").Dot("MaxLeftover").Op("==").Lit(0)).Block(
				jen.Id("cfg").Dot("MaxLeftover").Op("=").Lit(defaultLeftover),
			),
			jen.Line(),
			// Create the processor function
			jen.Id("processor").Op(":=").Func().Params(
				jen.Id("data").Index().Byte(),
				jen.Id("isEOF").Bool(),
				jen.Id("_").Qual(streamPkg, "TransformFunc"),
				jen.Id("emitOut").Func().Params(jen.Index().Byte()),
			).Int().Block(
				jen.Return(jen.Id(c.config.Name).Values().Dot("processTransform").Call(
					jen.Id("data"),
					jen.Id("isEOF"),
					jen.Id("onMatch"),
					jen.Id("emitOut"),
				)),
			),
			jen.Line(),
			// Create the transformer with a placeholder transformFn (unused, processor handles it)
			jen.Return(jen.Qual(streamPkg, "NewTransformer").Call(
				jen.Id("r"),
				jen.Id("cfg"),
				jen.Id("processor"),
				jen.Func().Params(
					jen.Id("_").Index().Byte(),
					jen.Id("emit").Func().Params(jen.Index().Byte()),
				).Block(),
			)),
		)
	c.file.Line()

	// Generate the internal processTransform helper
	c.generateProcessTransform(bytesStructName, defaultLeftover)
}

// generateProcessTransform generates the internal processing function.
func (c *Compiler) generateProcessTransform(bytesStructName string, defaultLeftover int) {
	c.file.Comment("processTransform processes data for transformation.")
	c.method("processTransform").
		Params(
			jen.Id("data").Index().Byte(),
			jen.Id("isEOF").Bool(),
			jen.Id("onMatch").Func().Params(
				jen.Id("match").Op("*").Id(bytesStructName),
				jen.Id("emit").Func().Params(jen.Index().Byte()),
			),
			jen.Id("emitOut").Func().Params(jen.Index().Byte()),
		).
		Params(jen.Int()).
		Block(
			jen.Id("processed").Op(":=").Lit(0),
			jen.Id("reuseResult").Op(":=").Op("&").Id(bytesStructName).Values(),
			jen.Line(),
			jen.For().Block(
				// Search for next match
				jen.List(jen.Id("result"), jen.Id("ok")).Op(":=").Id(c.config.Name).Values().Dot("FindBytesReuse").Call(
					jen.Id("data").Index(jen.Id("processed").Op(":")),
					jen.Id("reuseResult"),
				),
				jen.If(jen.Op("!").Id("ok")).Block(
					jen.Comment("No more matches"),
					jen.If(jen.Id("isEOF")).Block(
						jen.If(jen.Id("processed").Op("<").Len(jen.Id("data"))).Block(
							jen.Id("emitOut").Call(jen.Id("data").Index(jen.Id("processed").Op(":"))),
						),
						jen.Return(jen.Len(jen.Id("data"))),
					),
					jen.Comment("Keep potential partial match data"),
					jen.Id("safePoint").Op(":=").Len(jen.Id("data")).Op("-").Lit(defaultLeftover/10),
					jen.If(jen.Id("safePoint").Op("<").Id("processed")).Block(
						jen.Id("safePoint").Op("=").Id("processed"),
					),
					jen.If(jen.Id("safePoint").Op(">").Id("processed")).Block(
						jen.Id("emitOut").Call(jen.Id("data").Index(jen.Id("processed").Op(":").Id("safePoint"))),
					),
					jen.Return(jen.Id("safePoint")),
				),
				jen.Line(),
				// Find match position
				jen.Id("matchIdx").Op(":=").Qual("bytes", "Index").Call(
					jen.Id("data").Index(jen.Id("processed").Op(":")),
					jen.Id("result").Dot("Match"),
				),
				jen.If(jen.Id("matchIdx").Op("<").Lit(0)).Block(
					jen.Break(),
				),
				jen.Line(),
				jen.Id("matchStart").Op(":=").Id("processed").Op("+").Id("matchIdx"),
				jen.Id("matchEnd").Op(":=").Id("matchStart").Op("+").Len(jen.Id("result").Dot("Match")),
				jen.Line(),
				// Emit non-match segment before this match
				jen.If(jen.Id("matchStart").Op(">").Id("processed")).Block(
					jen.Id("emitOut").Call(jen.Id("data").Index(jen.Id("processed").Op(":").Id("matchStart"))),
				),
				jen.Line(),
				// Call onMatch to handle the match
				jen.Id("onMatch").Call(jen.Id("result"), jen.Id("emitOut")),
				jen.Line(),
				// Move past this match
				jen.If(jen.Len(jen.Id("result").Dot("Match")).Op(">").Lit(0)).Block(
					jen.Id("processed").Op("=").Id("matchEnd"),
				).Else().Block(
					jen.Id("processed").Op("++"),
				),
			),
			jen.Return(jen.Id("processed")),
		)
	c.file.Line()
}

// generateReplaceReader generates the ReplaceReader convenience method.
func (c *Compiler) generateReplaceReader() {
	bytesStructName := fmt.Sprintf("%sBytesResult", c.config.Name)
	streamPkg := "github.com/KromDaniel/regengo/stream"
	replacePkg := "github.com/KromDaniel/regengo/replace"

	c.file.Comment("ReplaceReader returns an io.Reader that replaces all matches with the template.")
	c.file.Comment("")
	c.file.Comment("Template syntax:")
	c.file.Comment("  $0 or ${0}: full match")
	c.file.Comment("  $1, $2, ...: capture group by index")
	c.file.Comment("  $name or ${name}: capture group by name")
	c.file.Comment("  $$: literal dollar sign")
	c.file.Comment("")
	c.file.Comment("Example:")
	c.file.Comment("  r := pattern.ReplaceReader(input, \"[$1]\")")
	c.file.Comment("  io.Copy(os.Stdout, r)")
	c.method("ReplaceReader").
		Params(
			jen.Id("r").Qual("io", "Reader"),
			jen.Id("template").String(),
		).
		Params(jen.Qual("io", "Reader")).
		Block(
			// Parse template once
			jen.List(jen.Id("parsed"), jen.Id("err")).Op(":=").Qual(replacePkg, "Parse").Call(jen.Id("template")),
			jen.If(jen.Id("err").Op("!=").Nil()).Block(
				jen.Return(jen.Op("&").Id(c.transformErrReaderName()).Values(jen.Id("err").Op(":").Id("err"))),
			),
			jen.Line(),
			// Build capture names map for validation
			jen.Id("captureNames").Op(":=").Map(jen.String()).Int().Values(c.generateCaptureNamesMap()...),
			jen.List(jen.Id("resolved"), jen.Id("err")).Op(":=").Id("parsed").Dot("ValidateAndResolve").Call(
				jen.Id("captureNames"),
				jen.Lit(len(c.captureNames)-1),
			),
			jen.If(jen.Id("err").Op("!=").Nil()).Block(
				jen.Return(jen.Op("&").Id(c.transformErrReaderName()).Values(jen.Id("err").Op(":").Id("err"))),
			),
			jen.Line(),
			// Create transform reader with replacement logic
			jen.Return(jen.Id(c.config.Name).Values().Dot("NewTransformReader").Call(
				jen.Id("r"),
				jen.Qual(streamPkg, "DefaultTransformConfig").Call(),
				jen.Func().Params(
					jen.Id("m").Op("*").Id(bytesStructName),
					jen.Id("emit").Func().Params(jen.Index().Byte()),
				).Block(
					jen.Id("result").Op(":=").Make(jen.Index().Byte(), jen.Lit(0), jen.Len(jen.Id("template")).Op("+").Lit(64)),
					jen.For(jen.List(jen.Id("_"), jen.Id("seg")).Op(":=").Range().Id("resolved")).Block(
						jen.Switch(jen.Id("seg").Dot("Type")).Block(
							jen.Case(jen.Qual(replacePkg, "SegmentLiteral")).Block(
								jen.Id("result").Op("=").Append(jen.Id("result"), jen.Id("seg").Dot("Literal").Op("..."))),
							jen.Case(jen.Qual(replacePkg, "SegmentFullMatch")).Block(
								jen.Id("result").Op("=").Append(jen.Id("result"), jen.Id("m").Dot("Match").Op("..."))),
							jen.Case(jen.Qual(replacePkg, "SegmentCaptureIndex")).Block(
								jen.Id("capture").Op(":=").Parens(jen.Id(c.config.Name).Values()).Dot("getCaptureByIndex").Call(jen.Id("m"), jen.Id("seg").Dot("CaptureIndex")),
								jen.If(jen.Id("capture").Op("!=").Nil()).Block(
									jen.Id("result").Op("=").Append(jen.Id("result"), jen.Id("capture").Op("..."))),
							),
						),
					),
					jen.Id("emit").Call(jen.Id("result")),
				),
			)),
		)
	c.file.Line()

	// Generate helper types and functions
	c.generateTransformErrReader()
	c.generateGetCaptureByIndex(bytesStructName)
}

// generateCaptureNamesMap generates the capture names map literal.
func (c *Compiler) generateCaptureNamesMap() []jen.Code {
	var entries []jen.Code
	for i, name := range c.captureNames {
		if i == 0 || name == "" {
			continue
		}
		entries = append(entries, jen.Lit(name).Op(":").Lit(i))
	}
	return entries
}

// generateTransformErrReader generates the error reader helper type.
// Uses pattern-specific name to avoid conflicts when multiple patterns are in the same package.
func (c *Compiler) generateTransformErrReader() {
	errReaderName := c.transformErrReaderName()
	c.file.Commentf("%s is a reader that always returns an error.", errReaderName)
	c.file.Type().Id(errReaderName).Struct(
		jen.Id("err").Error(),
	)
	c.file.Line()
	c.file.Func().Params(jen.Id("r").Op("*").Id(errReaderName)).Id("Read").Params(
		jen.Id("p").Index().Byte(),
	).Params(jen.Int(), jen.Error()).Block(
		jen.Return(jen.Lit(0), jen.Id("r").Dot("err")),
	)
	c.file.Line()
}

// transformErrReaderName returns the pattern-specific error reader type name.
func (c *Compiler) transformErrReaderName() string {
	return strings.ToLower(c.config.Name[:1]) + c.config.Name[1:] + "TransformErrReader"
}

// generateGetCaptureByIndex generates a helper to get capture group by index.
func (c *Compiler) generateGetCaptureByIndex(bytesStructName string) {
	var cases []jen.Code
	for i, name := range c.captureNames {
		if i == 0 {
			cases = append(cases, jen.Case(jen.Lit(0)).Block(
				jen.Return(jen.Id("m").Dot("Match")),
			))
			continue
		}
		if name == "" {
			continue
		}
		fieldName := capitalizeFirst(name)
		cases = append(cases, jen.Case(jen.Lit(i)).Block(
			jen.Return(jen.Id("m").Dot(fieldName)),
		))
	}
	cases = append(cases, jen.Default().Block(jen.Return(jen.Nil())))

	c.file.Comment("getCaptureByIndex returns the capture group value by its index.")
	c.method("getCaptureByIndex").
		Params(
			jen.Id("m").Op("*").Id(bytesStructName),
			jen.Id("index").Int(),
		).
		Params(jen.Index().Byte()).
		Block(
			jen.Switch(jen.Id("index")).Block(cases...),
		)
	c.file.Line()
}

// generateSelectReader generates the SelectReader method.
func (c *Compiler) generateSelectReader() {
	bytesStructName := fmt.Sprintf("%sBytesResult", c.config.Name)
	streamPkg := "github.com/KromDaniel/regengo/stream"

	// Calculate default leftover
	defaultLeftover := 1 << 20
	if c.matchLength.MaxMatchLen != -1 {
		defaultLeftover = c.matchLength.MaxMatchLen * 10
		if defaultLeftover < 1024 {
			defaultLeftover = 1024
		}
		if defaultLeftover > 1<<20 {
			defaultLeftover = 1 << 20
		}
	}

	c.file.Comment("SelectReader returns an io.Reader that outputs only matches satisfying the predicate.")
	c.file.Comment("Non-matching segments are dropped. Only matches where pred returns true are output.")
	c.file.Comment("")
	c.file.Comment("Example - extract all email addresses:")
	c.file.Comment("  r := emailPattern.SelectReader(input, func(m *EmailBytesResult) bool {")
	c.file.Comment("      return true // Keep all matches")
	c.file.Comment("  })")
	c.method("SelectReader").
		Params(
			jen.Id("r").Qual("io", "Reader"),
			jen.Id("pred").Func().Params(jen.Op("*").Id(bytesStructName)).Bool(),
		).
		Params(jen.Qual("io", "Reader")).
		Block(
			jen.Id("cfg").Op(":=").Qual(streamPkg, "DefaultTransformConfig").Call(),
			jen.Id("cfg").Dot("MaxLeftover").Op("=").Lit(defaultLeftover),
			jen.Line(),
			jen.Id("processor").Op(":=").Func().Params(
				jen.Id("data").Index().Byte(),
				jen.Id("isEOF").Bool(),
				jen.Id("_").Qual(streamPkg, "TransformFunc"),
				jen.Id("emitOut").Func().Params(jen.Index().Byte()),
			).Int().Block(
				jen.Return(jen.Id(c.config.Name).Values().Dot("processSelect").Call(
					jen.Id("data"),
					jen.Id("isEOF"),
					jen.Id("pred"),
					jen.Id("emitOut"),
				)),
			),
			jen.Line(),
			jen.Return(jen.Qual(streamPkg, "NewTransformer").Call(
				jen.Id("r"),
				jen.Id("cfg"),
				jen.Id("processor"),
				jen.Func().Params(
					jen.Id("_").Index().Byte(),
					jen.Id("emit").Func().Params(jen.Index().Byte()),
				).Block(),
			)),
		)
	c.file.Line()

	// Generate the internal processSelect helper
	c.generateProcessSelect(bytesStructName, defaultLeftover)
}

// generateProcessSelect generates the internal processing function for Select.
func (c *Compiler) generateProcessSelect(bytesStructName string, defaultLeftover int) {
	c.file.Comment("processSelect processes data for select filtering.")
	c.method("processSelect").
		Params(
			jen.Id("data").Index().Byte(),
			jen.Id("isEOF").Bool(),
			jen.Id("pred").Func().Params(jen.Op("*").Id(bytesStructName)).Bool(),
			jen.Id("emitOut").Func().Params(jen.Index().Byte()),
		).
		Params(jen.Int()).
		Block(
			jen.Id("processed").Op(":=").Lit(0),
			jen.Id("reuseResult").Op(":=").Op("&").Id(bytesStructName).Values(),
			jen.Line(),
			jen.For().Block(
				jen.List(jen.Id("result"), jen.Id("ok")).Op(":=").Id(c.config.Name).Values().Dot("FindBytesReuse").Call(
					jen.Id("data").Index(jen.Id("processed").Op(":")),
					jen.Id("reuseResult"),
				),
				jen.If(jen.Op("!").Id("ok")).Block(
					jen.If(jen.Id("isEOF")).Block(
						jen.Return(jen.Len(jen.Id("data"))),
					),
					jen.Return(jen.Id("processed")),
				),
				jen.Line(),
				jen.Id("matchIdx").Op(":=").Qual("bytes", "Index").Call(
					jen.Id("data").Index(jen.Id("processed").Op(":")),
					jen.Id("result").Dot("Match"),
				),
				jen.If(jen.Id("matchIdx").Op("<").Lit(0)).Block(
					jen.Break(),
				),
				jen.Line(),
				jen.Id("matchEnd").Op(":=").Id("processed").Op("+").Id("matchIdx").Op("+").Len(jen.Id("result").Dot("Match")),
				jen.Line(),
				// For Select: emit match if predicate is true
				jen.If(jen.Id("pred").Call(jen.Id("result"))).Block(
					jen.Id("emitOut").Call(jen.Id("result").Dot("Match")),
				),
				jen.Line(),
				jen.If(jen.Len(jen.Id("result").Dot("Match")).Op(">").Lit(0)).Block(
					jen.Id("processed").Op("=").Id("matchEnd"),
				).Else().Block(
					jen.Id("processed").Op("++"),
				),
			),
			jen.Return(jen.Id("processed")),
		)
	c.file.Line()
}

// generateRejectReader generates the RejectReader method.
func (c *Compiler) generateRejectReader() {
	bytesStructName := fmt.Sprintf("%sBytesResult", c.config.Name)
	streamPkg := "github.com/KromDaniel/regengo/stream"

	// Calculate default leftover
	defaultLeftover := 1 << 20
	if c.matchLength.MaxMatchLen != -1 {
		defaultLeftover = c.matchLength.MaxMatchLen * 10
		if defaultLeftover < 1024 {
			defaultLeftover = 1024
		}
		if defaultLeftover > 1<<20 {
			defaultLeftover = 1 << 20
		}
	}

	c.file.Comment("RejectReader returns an io.Reader that removes matches satisfying the predicate.")
	c.file.Comment("Non-matching segments pass through. Matches where pred returns true are dropped.")
	c.file.Comment("")
	c.file.Comment("Example - remove all sensitive data:")
	c.file.Comment("  r := sensitivePattern.RejectReader(input, func(m *SensitiveBytesResult) bool {")
	c.file.Comment("      return true // Remove all matches")
	c.file.Comment("  })")
	c.method("RejectReader").
		Params(
			jen.Id("r").Qual("io", "Reader"),
			jen.Id("pred").Func().Params(jen.Op("*").Id(bytesStructName)).Bool(),
		).
		Params(jen.Qual("io", "Reader")).
		Block(
			jen.Id("cfg").Op(":=").Qual(streamPkg, "DefaultTransformConfig").Call(),
			jen.Id("cfg").Dot("MaxLeftover").Op("=").Lit(defaultLeftover),
			jen.Line(),
			jen.Id("processor").Op(":=").Func().Params(
				jen.Id("data").Index().Byte(),
				jen.Id("isEOF").Bool(),
				jen.Id("_").Qual(streamPkg, "TransformFunc"),
				jen.Id("emitOut").Func().Params(jen.Index().Byte()),
			).Int().Block(
				jen.Return(jen.Id(c.config.Name).Values().Dot("processReject").Call(
					jen.Id("data"),
					jen.Id("isEOF"),
					jen.Id("pred"),
					jen.Id("emitOut"),
				)),
			),
			jen.Line(),
			jen.Return(jen.Qual(streamPkg, "NewTransformer").Call(
				jen.Id("r"),
				jen.Id("cfg"),
				jen.Id("processor"),
				jen.Func().Params(
					jen.Id("_").Index().Byte(),
					jen.Id("emit").Func().Params(jen.Index().Byte()),
				).Block(),
			)),
		)
	c.file.Line()

	// Generate the internal processReject helper
	c.generateProcessReject(bytesStructName, defaultLeftover)
}

// generateProcessReject generates the internal processing function for Reject.
func (c *Compiler) generateProcessReject(bytesStructName string, defaultLeftover int) {
	c.file.Comment("processReject processes data for reject filtering.")
	c.method("processReject").
		Params(
			jen.Id("data").Index().Byte(),
			jen.Id("isEOF").Bool(),
			jen.Id("pred").Func().Params(jen.Op("*").Id(bytesStructName)).Bool(),
			jen.Id("emitOut").Func().Params(jen.Index().Byte()),
		).
		Params(jen.Int()).
		Block(
			jen.Id("processed").Op(":=").Lit(0),
			jen.Id("reuseResult").Op(":=").Op("&").Id(bytesStructName).Values(),
			jen.Line(),
			jen.For().Block(
				jen.List(jen.Id("result"), jen.Id("ok")).Op(":=").Id(c.config.Name).Values().Dot("FindBytesReuse").Call(
					jen.Id("data").Index(jen.Id("processed").Op(":")),
					jen.Id("reuseResult"),
				),
				jen.If(jen.Op("!").Id("ok")).Block(
					jen.If(jen.Id("isEOF")).Block(
						jen.If(jen.Id("processed").Op("<").Len(jen.Id("data"))).Block(
							jen.Id("emitOut").Call(jen.Id("data").Index(jen.Id("processed").Op(":"))),
						),
						jen.Return(jen.Len(jen.Id("data"))),
					),
					// Keep some data for boundary matches
					jen.Id("safePoint").Op(":=").Len(jen.Id("data")).Op("-").Lit(defaultLeftover/10),
					jen.If(jen.Id("safePoint").Op("<").Id("processed")).Block(
						jen.Id("safePoint").Op("=").Id("processed"),
					),
					jen.If(jen.Id("safePoint").Op(">").Id("processed")).Block(
						jen.Id("emitOut").Call(jen.Id("data").Index(jen.Id("processed").Op(":").Id("safePoint"))),
					),
					jen.Return(jen.Id("safePoint")),
				),
				jen.Line(),
				jen.Id("matchIdx").Op(":=").Qual("bytes", "Index").Call(
					jen.Id("data").Index(jen.Id("processed").Op(":")),
					jen.Id("result").Dot("Match"),
				),
				jen.If(jen.Id("matchIdx").Op("<").Lit(0)).Block(
					jen.Break(),
				),
				jen.Line(),
				jen.Id("matchStart").Op(":=").Id("processed").Op("+").Id("matchIdx"),
				jen.Id("matchEnd").Op(":=").Id("matchStart").Op("+").Len(jen.Id("result").Dot("Match")),
				jen.Line(),
				// Emit non-match segment before this match
				jen.If(jen.Id("matchStart").Op(">").Id("processed")).Block(
					jen.Id("emitOut").Call(jen.Id("data").Index(jen.Id("processed").Op(":").Id("matchStart"))),
				),
				jen.Line(),
				// For Reject: emit match only if predicate is false
				jen.If(jen.Op("!").Id("pred").Call(jen.Id("result"))).Block(
					jen.Id("emitOut").Call(jen.Id("result").Dot("Match")),
				),
				jen.Line(),
				jen.If(jen.Len(jen.Id("result").Dot("Match")).Op(">").Lit(0)).Block(
					jen.Id("processed").Op("=").Id("matchEnd"),
				).Else().Block(
					jen.Id("processed").Op("++"),
				),
			),
			jen.Return(jen.Id("processed")),
		)
	c.file.Line()
}
