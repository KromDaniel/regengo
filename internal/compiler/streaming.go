package compiler

import (
	"fmt"

	"github.com/dave/jennifer/jen"
)

// generateStreamingMethods generates the FindReader and related streaming methods.
// These methods enable matching on io.Reader with constant memory usage.
func (c *Compiler) generateStreamingMethods() {
	if !c.config.WithCaptures {
		// Streaming only works with capture functions (FindBytes)
		return
	}

	c.generateDefaultMaxLeftover()
	c.generateFindReader()
	c.generateFindReaderCount()
	c.generateFindReaderFirst()
}

// generateDefaultMaxLeftover generates the DefaultMaxLeftover method
// that returns the pattern-specific default for MaxLeftover.
func (c *Compiler) generateDefaultMaxLeftover() {
	// Calculate default max leftover based on pattern analysis
	defaultLeftover := 1 << 20 // 1MB for unbounded patterns
	if c.matchLength.MaxMatchLen != -1 {
		defaultLeftover = c.matchLength.MaxMatchLen * 10
		if defaultLeftover < 1024 {
			defaultLeftover = 1024 // minimum 1KB
		}
		if defaultLeftover > 1<<20 {
			defaultLeftover = 1 << 20 // cap at 1MB
		}
	}

	c.file.Comment("DefaultMaxLeftover returns the recommended MaxLeftover value for streaming.")
	c.file.Comment("For bounded patterns, this is 10 * MaxMatchLen.")
	c.file.Comment("For unbounded patterns, this returns 1MB as a safety limit.")
	c.method("DefaultMaxLeftover").
		Params().
		Params(jen.Int()).
		Block(
			jen.Return(jen.Lit(defaultLeftover)),
		)
	c.file.Line()
}

// generateFindReader generates the main FindReader streaming method.
func (c *Compiler) generateFindReader() {
	bytesStructName := fmt.Sprintf("%sBytesResult", c.config.Name)
	streamPkg := "github.com/KromDaniel/regengo/stream"

	// Calculate minimum buffer size (2 * max match len, or 64KB for unbounded)
	minBuffer := 64 * 1024
	if c.matchLength.MaxMatchLen > 0 {
		minBuffer = c.matchLength.MaxMatchLen * 2
		if minBuffer < 64*1024 {
			minBuffer = 64 * 1024
		}
	}

	c.file.Comment("FindReader streams matches from an io.Reader.")
	c.file.Comment("")
	c.file.Comment("The onMatch callback is called for each match found. Return false to stop")
	c.file.Comment("processing early. The StreamMatch.Result points into an internal buffer")
	c.file.Comment("and is only valid during the callback - copy if needed.")
	c.file.Comment("")
	c.file.Comment("Returns nil on success (including early termination via callback).")
	c.file.Comment("Returns error on read failure.")
	c.method("FindReader").
		Params(
			jen.Id("r").Qual("io", "Reader"),
			jen.Id("cfg").Qual(streamPkg, "Config"),
			jen.Id("onMatch").Func().Params(jen.Qual(streamPkg, "Match").Types(jen.Op("*").Id(bytesStructName))).Bool(),
		).
		Params(jen.Error()).
		Block(c.generateFindReaderBody(minBuffer, bytesStructName, streamPkg)...)
	c.file.Line()
}

// generateFindReaderBody generates the body of the FindReader method.
// This uses a simpler approach: iterate through matches by searching incrementally.
func (c *Compiler) generateFindReaderBody(minBuffer int, bytesStructName, streamPkg string) []jen.Code {
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

	code := []jen.Code{
		// Validate and apply defaults to config
		jen.If(jen.Err().Op(":=").Id("cfg").Dot("Validate").Call(jen.Lit(minBuffer)), jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Err()),
		),
		jen.Id("cfg").Op("=").Id("cfg").Dot("ApplyDefaults").Call(
			jen.Lit(minBuffer),
			jen.Lit(defaultLeftover),
		),
		jen.Line(),

		// Initialize buffer
		jen.Id("buf").Op(":=").Make(jen.Index().Byte(), jen.Id("cfg").Dot("BufferSize")),
		jen.Id("leftover").Op(":=").Lit(0),
		jen.Id("streamOffset").Op(":=").Int64().Call(jen.Lit(0)),
		jen.Id("chunkIndex").Op(":=").Lit(0),
		jen.Line(),

		// Pre-allocate result struct for reuse (zero-allocation optimization)
		jen.Id("reuseResult").Op(":=").Op("&").Id(bytesStructName).Values(),
		jen.Line(),

		// Main read loop
		jen.For().Block(
			// Read into buffer after leftover
			jen.List(jen.Id("n"), jen.Id("err")).Op(":=").Id("r").Dot("Read").Call(
				jen.Id("buf").Index(jen.Id("leftover").Op(":")),
			),

			// Handle read errors
			jen.If(jen.Id("n").Op("==").Lit(0).Op("&&").Id("err").Op("!=").Nil()).Block(
				jen.If(jen.Id("err").Op("==").Qual("io", "EOF")).Block(
					jen.Comment("Process any remaining data in leftover"),
					jen.If(jen.Id("leftover").Op(">").Lit(0)).Block(
						jen.Id("chunk").Op(":=").Id("buf").Index(jen.Empty().Op(":").Id("leftover")),
						jen.Id("searchPos").Op(":=").Lit(0),
						jen.For(jen.Id("searchPos").Op("<").Len(jen.Id("chunk"))).Block(
							jen.List(jen.Id("result"), jen.Id("ok")).Op(":=").Id(c.config.Name).Values().Dot("FindBytesReuse").Call(
								jen.Id("chunk").Index(jen.Id("searchPos").Op(":")),
								jen.Id("reuseResult"),
							),
							jen.If(jen.Op("!").Id("ok")).Block(
								jen.Break(),
							),
							// Find where match starts relative to chunk
							jen.Id("matchIdx").Op(":=").Qual("bytes", "Index").Call(
								jen.Id("chunk").Index(jen.Id("searchPos").Op(":")),
								jen.Id("result").Dot("Match"),
							),
							jen.If(jen.Id("matchIdx").Op("<").Lit(0)).Block(
								jen.Break(),
							),
							jen.Id("matchStart").Op(":=").Id("searchPos").Op("+").Id("matchIdx"),
							jen.Line(),
							jen.Id("m").Op(":=").Qual(streamPkg, "Match").Types(jen.Op("*").Id(bytesStructName)).Values(jen.Dict{
								jen.Id("Result"):       jen.Id("result"),
								jen.Id("StreamOffset"): jen.Id("streamOffset").Op("+").Int64().Call(jen.Id("matchStart")),
								jen.Id("ChunkIndex"):   jen.Id("chunkIndex"),
							}),
							jen.If(jen.Op("!").Id("onMatch").Call(jen.Id("m"))).Block(
								jen.Return(jen.Nil()),
							),
							// Move past this match
							jen.If(jen.Len(jen.Id("result").Dot("Match")).Op(">").Lit(0)).Block(
								jen.Id("searchPos").Op("=").Id("matchStart").Op("+").Len(jen.Id("result").Dot("Match")),
							).Else().Block(
								jen.Id("searchPos").Op("++"),
							),
						),
					),
					jen.Return(jen.Nil()),
				),
				jen.Return(jen.Id("err")),
			),
			jen.Line(),

			// Calculate total data length
			jen.Id("dataLen").Op(":=").Id("leftover").Op("+").Id("n"),
			jen.Id("chunk").Op(":=").Id("buf").Index(jen.Empty().Op(":").Id("dataLen")),
			jen.Id("isFull").Op(":=").Id("n").Op("==").Id("cfg").Dot("BufferSize").Op("-").Id("leftover"),
			jen.Line(),

			// Search for matches, keeping track of position
			jen.Id("searchPos").Op(":=").Lit(0),
			jen.Id("committed").Op(":=").Lit(0),
			jen.For(jen.Id("searchPos").Op("<").Len(jen.Id("chunk"))).Block(
				jen.List(jen.Id("result"), jen.Id("ok")).Op(":=").Id(c.config.Name).Values().Dot("FindBytesReuse").Call(
					jen.Id("chunk").Index(jen.Id("searchPos").Op(":")),
					jen.Id("reuseResult"),
				),
				jen.If(jen.Op("!").Id("ok")).Block(
					jen.Break(),
				),
				// Find where match starts relative to chunk
				jen.Id("matchIdx").Op(":=").Qual("bytes", "Index").Call(
					jen.Id("chunk").Index(jen.Id("searchPos").Op(":")),
					jen.Id("result").Dot("Match"),
				),
				jen.If(jen.Id("matchIdx").Op("<").Lit(0)).Block(
					jen.Break(),
				),
				jen.Id("matchStart").Op(":=").Id("searchPos").Op("+").Id("matchIdx"),
				jen.Id("matchEnd").Op(":=").Id("matchStart").Op("+").Len(jen.Id("result").Dot("Match")),
				jen.Line(),

				// If match ends near chunk boundary and buffer was full, defer
				jen.If(jen.Id("isFull").Op("&&").Id("matchEnd").Op(">").Id("dataLen").Op("-").Id("cfg").Dot("MaxLeftover")).Block(
					jen.Comment("Match too close to boundary, defer to next chunk"),
					jen.Break(),
				),
				jen.Line(),

				// Call the callback
				jen.Id("m").Op(":=").Qual(streamPkg, "Match").Types(jen.Op("*").Id(bytesStructName)).Values(jen.Dict{
					jen.Id("Result"):       jen.Id("result"),
					jen.Id("StreamOffset"): jen.Id("streamOffset").Op("+").Int64().Call(jen.Id("matchStart")),
					jen.Id("ChunkIndex"):   jen.Id("chunkIndex"),
				}),
				jen.If(jen.Op("!").Id("onMatch").Call(jen.Id("m"))).Block(
					jen.Return(jen.Nil()),
				),
				jen.Id("committed").Op("=").Id("matchEnd"),

				// Move past this match
				jen.If(jen.Len(jen.Id("result").Dot("Match")).Op(">").Lit(0)).Block(
					jen.Id("searchPos").Op("=").Id("matchEnd"),
				).Else().Block(
					jen.Id("searchPos").Op("++"),
				),
			),
			jen.Line(),

			// Manage leftover
			jen.Comment("Prepare leftover for next iteration"),
			jen.If(jen.Id("isFull")).Block(
				jen.Comment("Buffer was full, need to keep leftover"),
				jen.Id("keepFrom").Op(":=").Id("dataLen").Op("-").Id("cfg").Dot("MaxLeftover"),
				jen.If(jen.Id("keepFrom").Op("<").Id("committed")).Block(
					jen.Id("keepFrom").Op("=").Id("committed"),
				),
				jen.Id("leftover").Op("=").Id("dataLen").Op("-").Id("keepFrom"),
				jen.Id("streamOffset").Op("+=").Int64().Call(jen.Id("keepFrom")),
				jen.Copy(jen.Id("buf").Index(jen.Empty().Op(":").Id("leftover")), jen.Id("buf").Index(jen.Id("keepFrom").Op(":").Id("dataLen"))),
			).Else().Block(
				jen.Comment("Reached EOF or short read, no more data"),
				jen.Id("leftover").Op("=").Lit(0),
			),
			jen.Id("chunkIndex").Op("++"),

			// Handle EOF on partial read
			jen.If(jen.Id("err").Op("==").Qual("io", "EOF")).Block(
				jen.Return(jen.Nil()),
			),
		),
	}

	return code
}

// generateFindReaderCount generates FindReaderCount for efficient counting.
func (c *Compiler) generateFindReaderCount() {
	streamPkg := "github.com/KromDaniel/regengo/stream"
	bytesStructName := fmt.Sprintf("%sBytesResult", c.config.Name)

	c.file.Comment("FindReaderCount counts matches without allocating result structs.")
	c.file.Comment("More efficient when you only need the count.")
	c.method("FindReaderCount").
		Params(
			jen.Id("r").Qual("io", "Reader"),
			jen.Id("cfg").Qual(streamPkg, "Config"),
		).
		Params(jen.Int64(), jen.Error()).
		Block(
			jen.Var().Id("count").Int64(),
			jen.Id("err").Op(":=").Id(c.config.Name).Values().Dot("FindReader").Call(
				jen.Id("r"),
				jen.Id("cfg"),
				jen.Func().Params(jen.Id("_").Qual(streamPkg, "Match").Types(jen.Op("*").Id(bytesStructName))).Bool().Block(
					jen.Id("count").Op("++"),
					jen.Return(jen.True()),
				),
			),
			jen.Return(jen.Id("count"), jen.Id("err")),
		)
	c.file.Line()
}

// generateFindReaderFirst generates FindReaderFirst for finding just the first match.
func (c *Compiler) generateFindReaderFirst() {
	streamPkg := "github.com/KromDaniel/regengo/stream"
	bytesStructName := fmt.Sprintf("%sBytesResult", c.config.Name)

	c.file.Comment("FindReaderFirst returns the first match, or nil if none found.")
	c.file.Comment("The returned result is a copy (safe to use after call returns).")
	c.method("FindReaderFirst").
		Params(
			jen.Id("r").Qual("io", "Reader"),
			jen.Id("cfg").Qual(streamPkg, "Config"),
		).
		Params(jen.Op("*").Id(bytesStructName), jen.Int64(), jen.Error()).
		Block(
			jen.Var().Id("result").Op("*").Id(bytesStructName),
			jen.Var().Id("offset").Int64(),
			jen.Id("err").Op(":=").Id(c.config.Name).Values().Dot("FindReader").Call(
				jen.Id("r"),
				jen.Id("cfg"),
				jen.Func().Params(jen.Id("m").Qual(streamPkg, "Match").Types(jen.Op("*").Id(bytesStructName))).Bool().Block(
					jen.Comment("Copy the result since buffer will be reused"),
					jen.Id("result").Op("=").Id(c.config.Name).Values().Dot("copyBytesResult").Call(jen.Id("m").Dot("Result")),
					jen.Id("offset").Op("=").Id("m").Dot("StreamOffset"),
					jen.Return(jen.False()), // Stop after first match
				),
			),
			jen.Return(jen.Id("result"), jen.Id("offset"), jen.Id("err")),
		)
	c.file.Line()

	// Generate the copyBytesResult helper
	c.generateCopyBytesResultHelper(bytesStructName)
}

// generateCopyBytesResultHelper generates a helper to deep-copy a BytesResult.
func (c *Compiler) generateCopyBytesResultHelper(bytesStructName string) {
	copyBody := []jen.Code{
		jen.If(jen.Id("src").Op("==").Nil()).Block(
			jen.Return(jen.Nil()),
		),
		jen.Id("dst").Op(":=").Op("&").Id(bytesStructName).Values(),
		jen.Comment("Copy Match slice"),
		jen.Id("dst").Dot("Match").Op("=").Append(jen.Index().Byte().Values(), jen.Id("src").Dot("Match").Op("...")),
	}

	// Add copy statements for each named capture group
	for _, name := range c.captureNames[1:] { // Skip group 0 (full match)
		if name == "" {
			continue // Skip unnamed groups
		}
		// Capitalize the first letter to match Go's exported field convention
		fieldName := capitalizeFirst(name)
		copyBody = append(copyBody,
			jen.If(jen.Id("src").Dot(fieldName).Op("!=").Nil()).Block(
				jen.Id("dst").Dot(fieldName).Op("=").Append(jen.Index().Byte().Values(), jen.Id("src").Dot(fieldName).Op("...")),
			),
		)
	}

	copyBody = append(copyBody, jen.Return(jen.Id("dst")))

	c.file.Comment("copyBytesResult creates a deep copy of a BytesResult.")
	c.file.Comment("This is needed because the original slices point into the stream buffer.")
	c.method("copyBytesResult").
		Params(jen.Id("src").Op("*").Id(bytesStructName)).
		Params(jen.Op("*").Id(bytesStructName)).
		Block(copyBody...)
	c.file.Line()
}

// capitalizeFirst capitalizes the first letter of a string.
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	// Handle first rune properly for Unicode
	for i, r := range s {
		if i == 0 {
			if r >= 'a' && r <= 'z' {
				return string(r-32) + s[1:]
			}
			return s
		}
	}
	return s
}
