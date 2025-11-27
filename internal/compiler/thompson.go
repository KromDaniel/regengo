package compiler

import (
	"regexp/syntax"

	"github.com/KromDaniel/regengo/internal/codegen"
	"github.com/dave/jennifer/jen"
)

// ThompsonGenerator generates Thompson NFA simulation code.
// Thompson's algorithm simulates all possible NFA states simultaneously,
// guaranteeing O(n*m) time complexity where n = input length, m = states.
type ThompsonGenerator struct {
	compiler     *Compiler
	prog         *syntax.Prog
	closures     []uint64 // Precomputed epsilon closures
	stateCount   int
	acceptMask   uint64 // Bitset of accepting states
	startClosure uint64 // Epsilon closure of start state
	charStates   []int  // States that consume characters
	isBytes      bool   // Generating bytes version
}

// NewThompsonGenerator creates a new Thompson NFA generator.
func NewThompsonGenerator(c *Compiler) *ThompsonGenerator {
	prog := c.config.Program
	if prog == nil {
		return nil
	}

	gen := &ThompsonGenerator{
		compiler:   c,
		prog:       prog,
		stateCount: len(prog.Inst),
	}

	// Precompute epsilon closures
	gen.closures = computeEpsilonClosures(prog)

	// Compute start state epsilon closure
	if prog.Start < len(prog.Inst) {
		gen.startClosure = gen.closures[prog.Start]
	}

	// Find accepting states and character-consuming states
	for i, inst := range prog.Inst {
		if inst.Op == syntax.InstMatch {
			if i < 64 {
				gen.acceptMask |= (1 << i)
			}
		}
		// Character-consuming instructions
		switch inst.Op {
		case syntax.InstRune, syntax.InstRune1, syntax.InstRuneAny, syntax.InstRuneAnyNotNL:
			gen.charStates = append(gen.charStates, i)
		}
	}

	return gen
}

// CanUseThompson returns true if the pattern can use Thompson NFA.
// Currently supports patterns with <= 64 states.
func (g *ThompsonGenerator) CanUseThompson() bool {
	return g.stateCount <= 64
}

// GenerateMatchFunction generates the Thompson NFA match function.
func (g *ThompsonGenerator) GenerateMatchFunction(isBytes bool) ([]jen.Code, error) {
	g.isBytes = isBytes
	g.compiler.logger.Section("Code Generation")
	g.compiler.logger.Log("Generating Thompson NFA match function (states: %d)", g.stateCount)

	code := []jen.Code{
		// Initialize length
		jen.Id(codegen.InputLenName).Op(":=").Len(jen.Id(codegen.InputName)),
	}

	// Generate constants as variables for clarity
	code = append(code,
		jen.Comment("Thompson NFA state sets (bitset representation)"),
		jen.Var().Id("current").Uint64(),
		jen.Var().Id("next").Uint64(),
		jen.Line(),
		jen.Comment("Precomputed constants"),
		jen.Id("startClosure").Op(":=").Uint64().Call(jen.Lit(g.startClosure)),
		jen.Id("acceptMask").Op(":=").Uint64().Call(jen.Lit(g.acceptMask)),
	)

	// Generate epsilon closure lookup table
	code = append(code, g.generateEpsilonClosureLookup()...)

	// Main matching loop - try from each position
	if g.compiler.isAnchored {
		// Anchored pattern - only try from start
		code = append(code,
			jen.Line(),
			jen.Comment("Anchored pattern - only try from start"),
			jen.Id("current").Op("=").Id("startClosure"),
			jen.Line(),
			jen.For(jen.Id("offset").Op(":=").Lit(0), jen.Id("offset").Op("<").Id(codegen.InputLenName), jen.Id("offset").Op("++")).Block(
				g.generateTransitionBlock()...,
			),
			jen.Line(),
			jen.Comment("Check if any accepting state is active"),
			jen.Return(jen.Id("current").Op("&").Id("acceptMask").Op("!=").Lit(0)),
		)
	} else {
		// Unanchored pattern - try from each position
		code = append(code,
			jen.Line(),
			jen.Comment("Unanchored pattern - try from each starting position"),
			jen.For(jen.Id("searchStart").Op(":=").Lit(0), jen.Id("searchStart").Op("<=").Id(codegen.InputLenName), jen.Id("searchStart").Op("++")).Block(
				jen.Id("current").Op("=").Id("startClosure"),
				jen.Line(),
				jen.Comment("Check immediate match (empty pattern or already at accept)"),
				jen.If(jen.Id("current").Op("&").Id("acceptMask").Op("!=").Lit(0)).Block(
					jen.Return(jen.True()),
				),
				jen.Line(),
				jen.For(jen.Id("offset").Op(":=").Id("searchStart"), jen.Id("offset").Op("<").Id(codegen.InputLenName), jen.Id("offset").Op("++")).Block(
					g.generateTransitionBlock()...,
				),
			),
			jen.Line(),
			jen.Return(jen.False()),
		)
	}

	return code, nil
}

// generateEpsilonClosureLookup generates the epsilon closure lookup table.
func (g *ThompsonGenerator) generateEpsilonClosureLookup() []jen.Code {
	// Only generate for states that need it (character-consuming states)
	closureEntries := make([]jen.Code, 0, len(g.charStates))
	for _, state := range g.charStates {
		inst := g.prog.Inst[state]
		nextState := int(inst.Out)
		if nextState < len(g.closures) {
			closureEntries = append(closureEntries,
				jen.Lit(state).Op(":").Uint64().Call(jen.Lit(g.closures[nextState])),
			)
		}
	}

	if len(closureEntries) == 0 {
		return nil
	}

	return []jen.Code{
		jen.Line(),
		jen.Comment("Epsilon closure lookup for next states after character transitions"),
		jen.Id("epsilonClosures").Op(":=").Index(jen.Lit(g.stateCount)).Uint64().Values(closureEntries...),
	}
}

// generateTransitionBlock generates the inner transition logic.
func (g *ThompsonGenerator) generateTransitionBlock() []jen.Code {
	var inputAccess *jen.Statement
	if g.isBytes {
		inputAccess = jen.Id(codegen.InputName).Index(jen.Id("offset"))
	} else {
		inputAccess = jen.Id(codegen.InputName).Index(jen.Id("offset"))
	}

	block := []jen.Code{
		jen.Id("c").Op(":=").Add(inputAccess),
		jen.Id("next").Op("=").Lit(0),
		jen.Line(),
	}

	// Generate transition code for each character-consuming state
	for _, stateIdx := range g.charStates {
		inst := g.prog.Inst[stateIdx]
		block = append(block, g.generateStateTransition(stateIdx, inst)...)
	}

	// Apply epsilon closures and check for match
	block = append(block,
		jen.Line(),
		jen.Comment("Update current state set"),
		jen.Id("current").Op("=").Id("next"),
		jen.Line(),
		jen.Comment("Check for dead end"),
		jen.If(jen.Id("current").Op("==").Lit(0)).Block(
			jen.Break(),
		),
		jen.Line(),
		jen.Comment("Check for match"),
		jen.If(jen.Id("current").Op("&").Id("acceptMask").Op("!=").Lit(0)).Block(
			jen.Return(jen.True()),
		),
	)

	return block
}

// generateStateTransition generates transition code for a single state.
func (g *ThompsonGenerator) generateStateTransition(stateIdx int, inst syntax.Inst) []jen.Code {
	stateBit := jen.Uint64().Call(jen.Lit(uint64(1) << stateIdx))

	var condition jen.Code
	switch inst.Op {
	case syntax.InstRune1:
		// Single character match
		if len(inst.Rune) > 0 {
			r := inst.Rune[0]
			if r < 128 {
				condition = jen.Id("c").Op("==").Lit(byte(r))
			} else {
				// Unicode character - need rune handling
				condition = jen.Id("c").Op("==").Lit(byte(r))
			}
		}
	case syntax.InstRune:
		// Character class
		condition = g.generateRuneCondition(inst)
	case syntax.InstRuneAny:
		// Match any character
		condition = jen.True()
	case syntax.InstRuneAnyNotNL:
		// Match any except newline
		condition = jen.Id("c").Op("!=").Lit(byte('\n'))
	}

	if condition == nil {
		return nil
	}

	return []jen.Code{
		jen.Comment("State " + string(rune('0'+stateIdx%10))),
		jen.If(
			jen.Id("current").Op("&").Add(stateBit).Op("!=").Lit(0).Op("&&").Add(condition),
		).Block(
			jen.Id("next").Op("|=").Id("epsilonClosures").Index(jen.Lit(stateIdx)),
		),
	}
}

// generateRuneCondition generates the condition for matching a character class.
func (g *ThompsonGenerator) generateRuneCondition(inst syntax.Inst) jen.Code {
	runes := inst.Rune
	if len(runes) == 0 {
		return jen.False()
	}

	// Check for case-insensitive flag
	foldCase := syntax.Flags(inst.Arg)&syntax.FoldCase != 0

	// Simple single character or range
	if len(runes) == 2 && runes[0] == runes[1] {
		// Single character
		r := runes[0]
		if foldCase && r < 128 {
			// Case insensitive ASCII
			lower := byte(r | 0x20)
			return jen.Parens(jen.Id("c").Op("|").Lit(byte(0x20))).Op("==").Lit(lower)
		}
		if r < 128 {
			return jen.Id("c").Op("==").Lit(byte(r))
		}
		return jen.Id("c").Op("==").Lit(byte(r))
	}

	// Build OR conditions for ranges
	var conditions []jen.Code
	for i := 0; i < len(runes); i += 2 {
		lo, hi := runes[i], runes[i+1]
		if lo == hi {
			if lo < 128 {
				conditions = append(conditions, jen.Id("c").Op("==").Lit(byte(lo)))
			}
		} else {
			// For ranges that start in ASCII range, include them
			// Clamp hi to 127 for ranges that extend beyond ASCII (e.g., [^"] generates [35, 1114111])
			if lo < 128 {
				clampedHi := hi
				if clampedHi > 127 {
					clampedHi = 127
				}
				conditions = append(conditions,
					jen.Parens(jen.Id("c").Op(">=").Lit(byte(lo)).Op("&&").Id("c").Op("<=").Lit(byte(clampedHi))),
				)
			}
		}
	}

	if len(conditions) == 0 {
		return jen.False()
	}
	if len(conditions) == 1 {
		return conditions[0]
	}

	// Combine with OR and wrap in parentheses to ensure correct precedence
	// when used with && in state condition
	result := conditions[0]
	for i := 1; i < len(conditions); i++ {
		result = jen.Parens(result).Op("||").Add(conditions[i])
	}
	return jen.Parens(result)
}
