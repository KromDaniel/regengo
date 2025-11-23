package compiler

import (
	"github.com/KromDaniel/regengo/internal/codegen"
	"github.com/dave/jennifer/jen"
)

// createBitmap creates a 256-bit bitmap for the given runes.
func createBitmap(runes []rune) [32]byte {
	var bitmap [32]byte
	for i := 0; i < len(runes); i += 2 {
		lo, hi := runes[i], runes[i+1]
		for c := lo; c <= hi; c++ {
			if c < 256 {
				bitmap[c/8] |= 1 << (c % 8)
			}
		}
	}
	return bitmap
}

// generateBitmapCheck generates a bitmap-based check for character classes.
// Returns the condition for FAILURE (i.e., char is NOT in the class).
func (c *Compiler) generateBitmapCheck(runes []rune) *jen.Statement {
	bitmap := createBitmap(runes)
	var values []jen.Code
	for _, b := range bitmap {
		values = append(values, jen.Lit(b))
	}

	// We define the bitmap inline. Go compiler handles this efficiently.
	// if [32]byte{...}[char/8] & (1 << (char%8)) == 0
	return jen.Index(jen.Lit(32)).Byte().Values(values...).Index(
		jen.Id(codegen.InputName).Index(jen.Id(codegen.OffsetName)).Op("/").Lit(8),
	).Op("&").Parens(
		jen.Lit(1).Op("<<").Parens(
			jen.Id(codegen.InputName).Index(jen.Id(codegen.OffsetName)).Op("%").Lit(8),
		),
	).Op("==").Lit(0)
}

// generateRuneCheck generates the condition for checking if a rune matches.
func (c *Compiler) generateRuneCheck(runes []rune) *jen.Statement {
	if len(runes) == 0 {
		return jen.True()
	}

	// Check for common character classes and use optimized checks
	if charClass := detectCharacterClass(runes); charClass != "" {
		return c.generateOptimizedCharClassCheck(charClass)
	}

	// For small sets (3 or fewer distinct values), use switch-like OR conditions
	if len(runes) <= 6 && allSingleChars(runes) {
		return c.generateSmallSetCheck(runes)
	}

	// For everything else, use bitmap-based check (O(1))
	return c.generateBitmapCheck(runes)
}

// detectCharacterClass checks if runes match a common character class pattern.
func detectCharacterClass(runes []rune) string {
	// \w: [0-9A-Za-z_]
	if len(runes) == 8 &&
		runes[0] == '0' && runes[1] == '9' &&
		runes[2] == 'A' && runes[3] == 'Z' &&
		runes[4] == '_' && runes[5] == '_' &&
		runes[6] == 'a' && runes[7] == 'z' {
		return "word"
	}

	// \d: [0-9]
	if len(runes) == 2 && runes[0] == '0' && runes[1] == '9' {
		return "digit"
	}

	// \s: [ \t\n\r\f\v]
	if len(runes) == 12 &&
		runes[0] == '\t' && runes[1] == '\n' &&
		runes[2] == '\f' && runes[3] == '\r' &&
		runes[4] == ' ' && runes[5] == ' ' {
		return "space"
	}

	// [a-z]
	if len(runes) == 2 && runes[0] == 'a' && runes[1] == 'z' {
		return "lowercase"
	}

	// [A-Z]
	if len(runes) == 2 && runes[0] == 'A' && runes[1] == 'Z' {
		return "uppercase"
	}

	// [a-zA-Z]
	if len(runes) == 4 &&
		runes[0] == 'A' && runes[1] == 'Z' &&
		runes[2] == 'a' && runes[3] == 'z' {
		return "alpha"
	}

	return ""
}

// generateOptimizedCharClassCheck generates optimized code for common character classes.
func (c *Compiler) generateOptimizedCharClassCheck(charClass string) *jen.Statement {
	// Helper to create input[offset] expression
	inputAt := func() *jen.Statement {
		return jen.Id(codegen.InputName).Index(jen.Id(codegen.OffsetName))
	}

	switch charClass {
	case "word":
		// \w: [0-9A-Za-z_] - NOT (< '0' || (> '9' && < 'A') || (> 'Z' && < '_') || (> '_' && < 'a') || > 'z')
		part1 := inputAt().Op("<").Lit(byte('0'))
		part2 := jen.Parens(inputAt().Op(">").Lit(byte('9')).Op("&&").Add(inputAt()).Op("<").Lit(byte('A')))
		part3 := jen.Parens(inputAt().Op(">").Lit(byte('Z')).Op("&&").Add(inputAt()).Op("<").Lit(byte('_')))
		part4 := jen.Parens(inputAt().Op(">").Lit(byte('_')).Op("&&").Add(inputAt()).Op("<").Lit(byte('a')))
		part5 := inputAt().Op(">").Lit(byte('z'))
		return jen.Parens(part1.Op("||").Add(part2).Op("||").Add(part3).Op("||").Add(part4).Op("||").Add(part5))

	case "digit":
		// \d: [0-9] - NOT (< '0' || > '9')
		return jen.Parens(inputAt().Op("<").Lit(byte('0')).Op("||").Add(inputAt()).Op(">").Lit(byte('9')))

	case "space":
		// \s: whitespace - NOT (== ' ' || == '\t' || ... )
		// Using != for all since we want "not in set"
		part1 := inputAt().Op("!=").Lit(byte(' '))
		part2 := inputAt().Op("!=").Lit(byte('\t'))
		part3 := inputAt().Op("!=").Lit(byte('\n'))
		part4 := inputAt().Op("!=").Lit(byte('\r'))
		part5 := inputAt().Op("!=").Lit(byte('\f'))
		return jen.Parens(part1.Op("&&").Add(part2).Op("&&").Add(part3).Op("&&").Add(part4).Op("&&").Add(part5))

	case "lowercase":
		// [a-z] - NOT (< 'a' || > 'z')
		return jen.Parens(inputAt().Op("<").Lit(byte('a')).Op("||").Add(inputAt()).Op(">").Lit(byte('z')))

	case "uppercase":
		// [A-Z] - NOT (< 'A' || > 'Z')
		return jen.Parens(inputAt().Op("<").Lit(byte('A')).Op("||").Add(inputAt()).Op(">").Lit(byte('Z')))

	case "alpha":
		// [a-zA-Z] - NOT ((< 'A' || > 'Z') && (< 'a' || > 'z'))
		upper := jen.Parens(inputAt().Op("<").Lit(byte('A')).Op("||").Add(inputAt()).Op(">").Lit(byte('Z')))
		lower := jen.Parens(inputAt().Op("<").Lit(byte('a')).Op("||").Add(inputAt()).Op(">").Lit(byte('z')))
		return jen.Parens(upper.Op("&&").Add(lower))
	}

	return jen.True()
}

// allSingleChars checks if all ranges are single characters.
func allSingleChars(runes []rune) bool {
	for i := 0; i < len(runes); i += 2 {
		if runes[i] != runes[i+1] {
			return false
		}
	}
	return true
}

// generateSmallSetCheck generates optimized code for small character sets using OR conditions.
func (c *Compiler) generateSmallSetCheck(runes []rune) *jen.Statement {
	ch := jen.Id(codegen.InputName).Index(jen.Id(codegen.OffsetName))

	var stmt *jen.Statement
	for i := 0; i < len(runes); i += 2 {
		condition := ch.Clone().Op("!=").Lit(byte(runes[i]))
		if stmt == nil {
			stmt = condition
		} else {
			stmt = stmt.Op("&&").Add(condition)
		}
	}

	return stmt
}
