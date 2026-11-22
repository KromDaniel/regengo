package compiler

import (
	"fmt"
	"regexp/syntax"

	"github.com/KromDaniel/regengo/internal/codegen"
	"github.com/dave/jennifer/jen"
)

// ewState is an InstEmptyWidth state whose outgoing closure is added at run time.
type ewState struct {
	idx  int    // state index
	need uint8  // required syntax.EmptyOp flags
	out  uint64 // epsilon closure of the state's Out
}

// ThompsonGenerator generates Thompson NFA simulation code.
// Thompson's algorithm simulates all possible NFA states simultaneously,
// guaranteeing O(n*m) time complexity where n = input length, m = states.
type ThompsonGenerator struct {
	compiler      *Compiler
	prog          *syntax.Prog
	closures      []uint64 // Precomputed epsilon closures
	stateCount    int
	acceptMask    uint64    // Bitset of accepting states
	startClosure  uint64    // Epsilon closure of start state
	charStates    []int     // States that consume characters
	isBytes       bool      // Generating bytes version
	runeAware     bool      // Program can match non-ASCII runes: step by rune
	ewStates      []ewState // Reachable assertion states, expanded at run time
	ewMask        uint64    // Bitset of ewStates
	ewChained     bool      // An assertion closure reaches another assertion
	ewOps         uint8     // OR of the assertions required by ewStates
	tablesEmitted bool      // Large class range tables already written to the file
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
		runeAware:  !isASCIIProgram(prog),
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

	gen.computeAssertions()

	return gen
}

// computeAssertions collects the InstEmptyWidth states that can become active.
// The static closures stop at those states (see computeEpsilonClosureForState);
// the generated code adds their closure at run time from the position context.
func (g *ThompsonGenerator) computeAssertions() {
	reach := g.startClosure
	for _, s := range g.charStates {
		if out := int(g.prog.Inst[s].Out); out < len(g.closures) {
			reach |= g.closures[out]
		}
	}
	for changed := true; changed; {
		changed = false
		for i, inst := range g.prog.Inst {
			if i >= 64 || inst.Op != syntax.InstEmptyWidth || reach&(1<<i) == 0 {
				continue
			}
			if out := int(inst.Out); out < len(g.closures) {
				if next := reach | g.closures[out]; next != reach {
					reach = next
					changed = true
				}
			}
		}
	}

	for i, inst := range g.prog.Inst {
		if i >= 64 || inst.Op != syntax.InstEmptyWidth || reach&(1<<i) == 0 {
			continue
		}
		st := ewState{idx: i, need: uint8(syntax.EmptyOp(inst.Arg))}
		if int(inst.Out) < len(g.closures) {
			st.out = g.closures[inst.Out]
		}
		g.ewStates = append(g.ewStates, st)
		g.ewMask |= 1 << i
		g.ewOps |= st.need
	}
	for _, st := range g.ewStates {
		if st.out&g.ewMask != 0 {
			g.ewChained = true
		}
	}
}

// CanUseThompson returns true if the pattern can use Thompson NFA.
// Currently supports patterns with <= 64 states.
func (g *ThompsonGenerator) CanUseThompson() bool {
	return g.stateCount <= 64
}

func (g *ThompsonGenerator) decodeCall(offsetVar string) jen.Code {
	name := "DecodeRuneInString"
	if g.isBytes {
		name = "DecodeRune"
	}
	return jen.Qual("unicode/utf8", name).Call(jen.Id(codegen.InputName).Index(jen.Id(offsetVar).Op(":")))
}

// scanLoop returns the loop that steps through the input from start.
func (g *ThompsonGenerator) scanLoop(start jen.Code) jen.Code {
	l := jen.Id(codegen.InputLenName)
	if g.runeAware {
		return jen.For(
			jen.List(jen.Id("offset"), jen.Id("w")).Op(":=").List(start, jen.Lit(1)),
			jen.Id("offset").Op("<").Add(l),
			jen.Id("offset").Op("+=").Id("w"),
		).Block(g.generateTransitionBlock()...)
	}
	return jen.For(
		jen.Id("offset").Op(":=").Add(start),
		jen.Id("offset").Op("<").Add(l),
		jen.Id("offset").Op("++"),
	).Block(g.generateTransitionBlock()...)
}

// GenerateMatchFunction generates the Thompson NFA match function.
func (g *ThompsonGenerator) GenerateMatchFunction(isBytes bool) ([]jen.Code, error) {
	g.isBytes = isBytes
	g.compiler.logger.Section("Code Generation")
	g.compiler.logger.Log("Generating Thompson NFA match function (states: %d)", g.stateCount)

	if g.runeAware && !g.tablesEmitted {
		g.emitClassTables()
		g.tablesEmitted = true
	}

	l := jen.Id(codegen.InputLenName)
	hasChars := len(g.charStates) > 0

	code := []jen.Code{
		// Initialize length
		jen.Id(codegen.InputLenName).Op(":=").Len(jen.Id(codegen.InputName)),
	}

	// Generate constants as variables for clarity
	code = append(code,
		jen.Comment("Thompson NFA state sets (bitset representation)"),
		jen.Var().Id("current").Uint64(),
	)
	if hasChars {
		code = append(code, jen.Var().Id("next").Uint64())
	}
	code = append(code,
		jen.Line(),
		jen.Comment("Precomputed constants"),
		jen.Id("startClosure").Op(":=").Uint64().Call(jen.Lit(g.startClosure)),
		jen.Id("acceptMask").Op(":=").Uint64().Call(jen.Lit(g.acceptMask)),
	)
	if len(g.ewStates) > 0 {
		code = append(code, jen.Id("ewMask").Op(":=").Uint64().Call(jen.Lit(g.ewMask)))
	}

	// Generate epsilon closure lookup table
	code = append(code, g.generateEpsilonClosureLookup()...)

	// Main matching loop - try from each position
	if g.compiler.isAnchored {
		// Anchored pattern - only try from start
		code = append(code,
			jen.Line(),
			jen.Comment("Anchored pattern - only try from start"),
			jen.Id("current").Op("=").Id("startClosure"),
		)
		code = append(code, g.generateExpand(jen.Lit(0))...)
		code = append(code,
			jen.Line(),
			jen.Comment("Check immediate match (pattern matches the empty string)"),
			jen.If(jen.Id("current").Op("&").Id("acceptMask").Op("!=").Lit(0)).Block(
				jen.Return(jen.True()),
			),
		)
		if hasChars {
			code = append(code, jen.Line(), g.scanLoop(jen.Lit(0)))
		} else {
			code = append(code, jen.Id("_").Op("=").Add(l))
		}
		code = append(code,
			jen.Line(),
			jen.Comment("Check if any accepting state is active"),
			jen.Return(jen.Id("current").Op("&").Id("acceptMask").Op("!=").Lit(0)),
		)
	} else {
		// Unanchored pattern - try from each position
		var loopBody []jen.Code
		var loop *jen.Statement
		if g.runeAware {
			// Advance by rune width so no attempt starts mid-rune.
			loop = jen.For(
				jen.List(jen.Id("searchStart"), jen.Id("sw")).Op(":=").List(jen.Lit(0), jen.Lit(1)),
				jen.Id("searchStart").Op("<=").Add(l),
				jen.Id("searchStart").Op("+=").Id("sw"),
			)
			loopBody = append(loopBody,
				jen.Id("sw").Op("=").Lit(1),
				jen.If(jen.Id("searchStart").Op("<").Add(l).Op("&&").
					Id(codegen.InputName).Index(jen.Id("searchStart")).Op(">=").Qual("unicode/utf8", "RuneSelf")).Block(
					jen.Id("_").Op(",").Id("sw").Op("=").Add(g.decodeCall("searchStart")),
				),
			)
		} else {
			loop = jen.For(jen.Id("searchStart").Op(":=").Lit(0), jen.Id("searchStart").Op("<=").Add(l), jen.Id("searchStart").Op("++"))
			if len(g.ewStates) > 0 {
				// An ASCII program never has a thread inside a rune, except at
				// the attempt start: skip continuation bytes so that attempts
				// start only on rune boundaries (for valid UTF-8 input).
				loopBody = append(loopBody,
					jen.Comment("Attempts start on rune boundaries only (valid UTF-8 input)"),
					jen.If(jen.Id("searchStart").Op("<").Add(l).Op("&&").
						Id(codegen.InputName).Index(jen.Id("searchStart")).Op("&").Lit(byte(0xC0)).Op("==").Lit(byte(0x80))).Block(
						jen.Continue(),
					),
				)
			}
		}
		loopBody = append(loopBody, jen.Id("current").Op("=").Id("startClosure"))
		loopBody = append(loopBody, g.generateExpand(jen.Id("searchStart"))...)
		loopBody = append(loopBody,
			jen.Line(),
			jen.Comment("Check immediate match (empty pattern or already at accept)"),
			jen.If(jen.Id("current").Op("&").Id("acceptMask").Op("!=").Lit(0)).Block(
				jen.Return(jen.True()),
			),
		)
		if hasChars {
			loopBody = append(loopBody, jen.Line(), g.scanLoop(jen.Id("searchStart")))
		}
		code = append(code,
			jen.Line(),
			jen.Comment("Unanchored pattern - try from each starting position"),
			loop.Block(loopBody...),
			jen.Line(),
			jen.Return(jen.False()),
		)
	}

	return code, nil
}

// generateExpand emits the run-time assertion expansion: it adds the closure of
// every active assertion state whose requirements hold at input position pos.
// It returns nil when the program has no reachable assertion. pos is assigned
// to a variable first so a constant position never forms input[0-1].
func (g *ThompsonGenerator) generateExpand(pos *jen.Statement) []jen.Code {
	if len(g.ewStates) == 0 {
		return nil
	}
	input := func() *jen.Statement { return jen.Id(codegen.InputName) }
	l := func() *jen.Statement { return jen.Id(codegen.InputLenName) }
	ewPos := func() *jen.Statement { return jen.Id("ewPos") }

	beginOps := uint8(syntax.EmptyBeginText | syntax.EmptyBeginLine)
	endOps := uint8(syntax.EmptyEndText | syntax.EmptyEndLine)
	wordOps := uint8(syntax.EmptyWordBoundary | syntax.EmptyNoWordBoundary)

	block := []jen.Code{
		ewPos().Op(":=").Add(pos),
		jen.Var().Id("ctx").Uint8(),
	}
	if g.ewOps&beginOps != 0 {
		block = append(block,
			jen.If(ewPos().Op("==").Lit(0)).Block(
				jen.Id("ctx").Op("|=").Lit(uint8(syntax.EmptyBeginText|syntax.EmptyBeginLine)),
			).Else().If(input().Index(ewPos().Op("-").Lit(1)).Op("==").Lit(byte('\n'))).Block(
				jen.Id("ctx").Op("|=").Lit(uint8(syntax.EmptyBeginLine)),
			),
		)
	}
	if g.ewOps&endOps != 0 {
		block = append(block,
			jen.If(ewPos().Op("==").Add(l())).Block(
				jen.Id("ctx").Op("|=").Lit(uint8(syntax.EmptyEndText|syntax.EmptyEndLine)),
			).Else().If(input().Index(ewPos()).Op("==").Lit(byte('\n'))).Block(
				jen.Id("ctx").Op("|=").Lit(uint8(syntax.EmptyEndLine)),
			),
		)
	}
	if g.ewOps&wordOps != 0 {
		block = append(block,
			jen.Id("pw").Op(":=").Add(ewPos()).Op(">").Lit(0).Op("&&").
				Id(codegen.IsWordCharName).Call(input().Index(ewPos().Op("-").Lit(1))),
			jen.Id("nw").Op(":=").Add(ewPos()).Op("<").Add(l()).Op("&&").
				Id(codegen.IsWordCharName).Call(input().Index(ewPos())),
			jen.If(jen.Id("pw").Op("!=").Id("nw")).Block(
				jen.Id("ctx").Op("|=").Lit(uint8(syntax.EmptyWordBoundary)),
			).Else().Block(
				jen.Id("ctx").Op("|=").Lit(uint8(syntax.EmptyNoWordBoundary)),
			),
		)
	}

	var lines []jen.Code
	for _, st := range g.ewStates {
		need := jen.Lit(st.need)
		lines = append(lines,
			jen.If(
				jen.Id("current").Op("&").Uint64().Call(jen.Lit(uint64(1)<<st.idx)).Op("!=").Lit(0).Op("&&").
					Id("ctx").Op("&").Add(need).Op("==").Add(jen.Lit(st.need)),
			).Block(
				jen.Id("current").Op("|=").Uint64().Call(jen.Lit(st.out)),
			),
		)
	}
	if g.ewChained {
		// One assertion's closure can reach another assertion: iterate.
		inner := append([]jen.Code{jen.Id("prev").Op(":=").Id("current")}, lines...)
		inner = append(inner, jen.If(jen.Id("current").Op("==").Id("prev")).Block(jen.Break()))
		block = append(block, jen.For().Block(inner...))
	} else {
		block = append(block, lines...)
	}

	return []jen.Code{
		jen.Line(),
		jen.Comment("Expand assertions (^ $ \\b ...) at this position"),
		jen.If(jen.Id("current").Op("&").Id("ewMask").Op("!=").Lit(0)).Block(block...),
	}
}

// emitClassTables writes a package-level unicode.RangeTable for every large
// class (more than 8 ranges) of a rune-aware program.
func (g *ThompsonGenerator) emitClassTables() {
	for _, stateIdx := range g.charStates {
		inst := g.prog.Inst[stateIdx]
		if !g.usesClassTable(inst) {
			continue
		}
		var r16, r32 []jen.Code
		add := func(lo, hi rune) {
			switch {
			case hi <= 0xFFFF:
				r16 = append(r16, jen.Values(jen.Dict{
					jen.Id("Lo"): jen.Lit(uint16(lo)), jen.Id("Hi"): jen.Lit(uint16(hi)), jen.Id("Stride"): jen.Lit(uint16(1)),
				}))
			default:
				r32 = append(r32, jen.Values(jen.Dict{
					jen.Id("Lo"): jen.Lit(uint32(lo)), jen.Id("Hi"): jen.Lit(uint32(hi)), jen.Id("Stride"): jen.Lit(uint32(1)),
				}))
			}
		}
		for i := 0; i < len(inst.Rune); i += 2 {
			lo, hi := inst.Rune[i], inst.Rune[i+1]
			if lo <= 0xFFFF && hi > 0xFFFF {
				add(lo, 0xFFFF)
				add(0x10000, hi)
			} else {
				add(lo, hi)
			}
		}
		fields := jen.Dict{}
		if len(r16) > 0 {
			fields[jen.Id("R16")] = jen.Index().Qual("unicode", "Range16").Values(r16...)
		}
		if len(r32) > 0 {
			fields[jen.Id("R32")] = jen.Index().Qual("unicode", "Range32").Values(r32...)
		}
		g.compiler.file.Comment("Rune ranges of NFA state " + fmt.Sprint(stateIdx) + " (Thompson NFA).")
		g.compiler.file.Var().Id(g.classTableName(stateIdx)).Op("=").Op("&").Qual("unicode", "RangeTable").Values(fields)
		g.compiler.file.Line()
	}
}

func (g *ThompsonGenerator) classTableName(stateIdx int) string {
	return fmt.Sprintf("thompsonClass%s%d", g.compiler.config.Name, stateIdx)
}

// usesClassTable reports whether a rune-aware condition for inst is a
// unicode.Is lookup (a multi-range class with more than 8 ranges).
func (g *ThompsonGenerator) usesClassTable(inst syntax.Inst) bool {
	return inst.Op == syntax.InstRune && len(inst.Rune) > 2*8 && len(inst.Rune)%2 == 0
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
	inputAccess := jen.Id(codegen.InputName).Index(jen.Id("offset"))

	// Transition code for each character-consuming state
	var trans []jen.Code
	usesR := false
	for _, stateIdx := range g.charStates {
		code, u := g.generateStateTransition(stateIdx, g.prog.Inst[stateIdx])
		trans = append(trans, code...)
		usesR = usesR || u
	}

	block := []jen.Code{jen.Id("c").Op(":=").Add(inputAccess)}
	var width jen.Code = jen.Lit(1)
	if g.runeAware {
		width = jen.Id("w")
		decode := jen.Id("_").Op(",").Id("w").Op("=").Add(g.decodeCall("offset"))
		if usesR {
			block = append(block, jen.Id("r").Op(":=").Id("rune").Call(jen.Id("c")))
			decode = jen.Id("r").Op(",").Id("w").Op("=").Add(g.decodeCall("offset"))
		}
		block = append(block,
			jen.Id("w").Op("=").Lit(1),
			jen.If(jen.Id("c").Op(">=").Qual("unicode/utf8", "RuneSelf")).Block(decode),
		)
	}
	block = append(block, jen.Id("next").Op("=").Lit(0), jen.Line())
	block = append(block, trans...)

	// Apply epsilon closures and check for match
	block = append(block,
		jen.Line(),
		jen.Comment("Update current state set"),
		jen.Id("current").Op("=").Id("next"),
	)
	block = append(block, g.generateExpand(jen.Id("offset").Op("+").Add(width))...)
	block = append(block,
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

// generateStateTransition generates transition code for a single state. The
// second result reports whether the condition reads the decoded rune r.
func (g *ThompsonGenerator) generateStateTransition(stateIdx int, inst syntax.Inst) ([]jen.Code, bool) {
	stateBit := jen.Uint64().Call(jen.Lit(uint64(1) << stateIdx))

	var condition jen.Code
	usesR := false
	if g.runeAware {
		condition, usesR = g.generateRuneModeCondition(stateIdx, inst)
	} else {
		switch inst.Op {
		case syntax.InstRune1:
			// Single character match (ASCII program: Rune[0] < 128)
			if len(inst.Rune) > 0 {
				condition = jen.Id("c").Op("==").Lit(byte(inst.Rune[0]))
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
	}

	if condition == nil {
		return nil, false
	}

	return []jen.Code{
		jen.Comment("State " + string(rune('0'+stateIdx%10))),
		jen.If(
			jen.Id("current").Op("&").Add(stateBit).Op("!=").Lit(0).Op("&&").Add(condition),
		).Block(
			jen.Id("next").Op("|=").Id("epsilonClosures").Index(jen.Lit(stateIdx)),
		),
	}, usesR
}

// generateRuneModeCondition builds the condition for a rune-aware program: it
// compares the decoded rune r against full rune ranges (no ASCII clamp). The
// second result reports whether the condition reads r.
func (g *ThompsonGenerator) generateRuneModeCondition(stateIdx int, inst syntax.Inst) (jen.Code, bool) {
	r := func() *jen.Statement { return jen.Id("r") }
	switch inst.Op {
	case syntax.InstRune1:
		if len(inst.Rune) == 0 {
			return nil, false
		}
		return r().Op("==").Lit(inst.Rune[0]), true
	case syntax.InstRuneAny:
		return jen.True(), false
	case syntax.InstRuneAnyNotNL:
		return r().Op("!=").Lit('\n'), true
	case syntax.InstRune:
		runes := inst.Rune
		switch {
		case len(runes) == 0:
			return jen.False(), false
		case len(runes) == 1:
			if syntax.Flags(inst.Arg)&syntax.FoldCase != 0 {
				orbit := foldOrbit(runes[0])
				if len(orbit) > 1 {
					cond := r().Op("==").Lit(orbit[0])
					for _, o := range orbit[1:] {
						cond = cond.Op("||").Add(r()).Op("==").Lit(o)
					}
					return jen.Parens(cond), true
				}
			}
			return r().Op("==").Lit(runes[0]), true
		case len(runes)%2 == 1:
			return jen.False(), false
		case g.usesClassTable(inst):
			return jen.Qual("unicode", "Is").Call(jen.Id(g.classTableName(stateIdx)), r()), true
		default:
			return jen.Op("!").Parens(g.compiler.generateUnicodeInlineRangeCheck(runes)), true
		}
	}
	return nil, false
}

// generateRuneCondition generates the byte-path condition for matching a
// character class (ASCII programs only).
func (g *ThompsonGenerator) generateRuneCondition(inst syntax.Inst) jen.Code {
	runes := inst.Rune
	if len(runes) == 0 {
		return jen.False()
	}

	// Check for case-insensitive flag
	foldCase := syntax.Flags(inst.Arg)&syntax.FoldCase != 0

	// A single rune (what regexp/syntax emits for a case-folded literal): the
	// rune itself or its fold orbit. Every orbit member is ASCII here.
	if len(runes) == 1 {
		if foldCase {
			orbit := foldOrbit(runes[0])
			if len(orbit) > 1 {
				cond := jen.Id("c").Op("==").Lit(byte(orbit[0]))
				for _, o := range orbit[1:] {
					cond = cond.Op("||").Id("c").Op("==").Lit(byte(o))
				}
				return jen.Parens(cond)
			}
		}
		return jen.Id("c").Op("==").Lit(byte(runes[0]))
	}

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
