package compiler

import (
	"fmt"
	"regexp/syntax"

	"github.com/dave/jennifer/jen"
)

// DFAGenerator generates pure DFA code for Match functions.
// Pure DFA provides O(n) matching per start position without capture group overhead.
// For unanchored patterns, the overall complexity is O(n²) worst case.
type DFAGenerator struct {
	compiler        *Compiler
	prog            *syntax.Prog
	states          []*DFAState
	stateMap        map[string]int // Maps NFA state set key to DFA state index
	transitions     []map[byte]int // transitions[state][char] = next state
	startStateBegin int            // Start state when at position 0 (anchored)
	startStateAny   int            // Start state when at position > 0
	acceptStates    map[int]bool   // Which states are accepting
	acceptStatesEOT map[int]bool   // Which states accept at end of text
	unanchoredSet   []int          // Epsilon closure of start without BeginText (restart threads)
	unanchored      bool           // True if a match can begin at any position (no ^ anchor)
	maxStates       int            // Maximum allowed states before giving up
	isBytes         bool           // Whether generating for bytes or string
}

// DFAState represents a state in the pure DFA.
type DFAState struct {
	ID       int
	NFASet   []int // Set of NFA state IDs (no tag actions)
	IsAccept bool
}

// NewDFAGenerator creates a new pure DFA generator.
func NewDFAGenerator(c *Compiler) *DFAGenerator {
	prog := c.config.Program
	if prog == nil {
		return nil
	}

	// Use configured threshold or default to 500
	maxStates := c.config.DFAThreshold
	if maxStates <= 0 {
		maxStates = 500
	}

	return &DFAGenerator{
		compiler:        c,
		prog:            prog,
		stateMap:        make(map[string]int),
		acceptStates:    make(map[int]bool),
		acceptStatesEOT: make(map[int]bool),
		maxStates:       maxStates,
	}
}

// CanUseDFA checks if DFA can be used for this pattern.
// Returns (canUse, info) where info is a status message for logging.
func (g *DFAGenerator) CanUseDFA() (bool, string) {
	// Check for unsupported instructions
	for _, inst := range g.prog.Inst {
		switch inst.Op {
		case syntax.InstEmptyWidth:
			// Only support BeginText (^) and EndText ($)
			op := syntax.EmptyOp(inst.Arg)
			if op != syntax.EmptyBeginText && op != syntax.EmptyEndText {
				return false, fmt.Sprintf("DFA unsupported empty width op: %v", op)
			}
		case syntax.InstRuneAny, syntax.InstRuneAnyNotNL:
			// Dot (.) can match non-ASCII - reject for pure ASCII DFA
			return false, "DFA does not support . (dot) which can match non-ASCII"
		case syntax.InstRune:
			// Check if any rune range includes non-ASCII
			for i := 0; i < len(inst.Rune); i += 2 {
				if inst.Rune[i+1] >= MaxASCIIRune {
					return false, fmt.Sprintf("DFA does not support non-ASCII character ranges (found range up to %d)", inst.Rune[i+1])
				}
			}
		case syntax.InstRune1:
			// Check if single rune is non-ASCII
			if len(inst.Rune) > 0 && inst.Rune[0] >= MaxASCIIRune {
				return false, fmt.Sprintf("DFA does not support non-ASCII character %d", inst.Rune[0])
			}
		}
	}

	// Build DFA and check if state count is acceptable
	if err := g.buildDFA(); err != nil {
		return false, fmt.Sprintf("DFA construction failed: %v", err)
	}

	if len(g.states) > g.maxStates {
		return false, fmt.Sprintf("DFA has too many states (%d > %d), falling back", len(g.states), g.maxStates)
	}

	return true, fmt.Sprintf("DFA constructed with %d states", len(g.states))
}

// buildDFA constructs the DFA from the NFA using the powerset construction algorithm.
func (g *DFAGenerator) buildDFA() error {
	// Initialize start states
	g.initializeStartStates()

	// Build initial worklist from start states
	worklist := []int{g.startStateBegin}
	if g.startStateAny != g.startStateBegin {
		worklist = append(worklist, g.startStateAny)
	}

	// Process worklist
	return g.processWorklist(worklist)
}

// initializeStartStates creates the initial DFA start states.
func (g *DFAGenerator) initializeStartStates() {
	// Compute start state for anchored match (position 0)
	startNFASetBegin := EpsilonClosure(g.prog, []int{int(g.prog.Start)}, syntax.EmptyBeginText)

	g.states = []*DFAState{{
		ID:     0,
		NFASet: startNFASetBegin,
	}}
	g.stateMap[NFASetKey(startNFASetBegin)] = 0
	g.transitions = []map[byte]int{{}}
	g.startStateBegin = 0

	if IsAcceptingNFASet(g.prog, startNFASetBegin) {
		g.states[0].IsAccept = true
		g.acceptStates[0] = true
	}

	// Compute start state for unanchored match
	startNFASetAny := EpsilonClosure(g.prog, []int{int(g.prog.Start)}, 0)
	g.unanchoredSet = startNFASetAny
	g.unanchored = len(GetPossibleChars(g.prog, startNFASetAny)) > 0
	keyAny := NFASetKey(startNFASetAny)

	if idx, exists := g.stateMap[keyAny]; exists {
		g.startStateAny = idx
	} else {
		idx := len(g.states)
		g.states = append(g.states, &DFAState{
			ID:     idx,
			NFASet: startNFASetAny,
		})
		g.stateMap[keyAny] = idx
		g.transitions = append(g.transitions, make(map[byte]int))
		g.startStateAny = idx

		if IsAcceptingNFASet(g.prog, startNFASetAny) {
			g.states[idx].IsAccept = true
			g.acceptStates[idx] = true
		}
	}
}

// processWorklist runs the DFA construction worklist algorithm.
func (g *DFAGenerator) processWorklist(initialWorklist []int) error {
	worklist := make([]int, len(initialWorklist))
	copy(worklist, initialWorklist)
	processed := make(map[int]bool)

	for len(worklist) > 0 {
		stateIdx := worklist[0]
		worklist = worklist[1:]

		if processed[stateIdx] {
			continue
		}
		processed[stateIdx] = true

		state := g.states[stateIdx]

		// Get possible characters using shared utility
		chars := GetPossibleChars(g.prog, state.NFASet)

		for _, c := range chars {
			// Compute next states using shared utility
			nextNFASet := ComputeNextStates(g.prog, state.NFASet, c, 0)

			// Single-pass unanchored matching: every state carries the restart
			// threads, so a dying thread never requires re-scanning from start+1.
			if g.unanchored {
				nextNFASet = UnionNFASets(nextNFASet, g.unanchoredSet)
			}

			if len(nextNFASet) == 0 {
				continue
			}

			key := NFASetKey(nextNFASet)
			nextStateIdx, exists := g.stateMap[key]

			if !exists {
				nextStateIdx = len(g.states)
				if nextStateIdx >= g.maxStates {
					return fmt.Errorf("DFA state explosion: exceeded %d states", g.maxStates)
				}

				nextState := &DFAState{
					ID:     nextStateIdx,
					NFASet: nextNFASet,
				}

				if IsAcceptingNFASet(g.prog, nextNFASet) {
					nextState.IsAccept = true
					g.acceptStates[nextStateIdx] = true
				}

				g.states = append(g.states, nextState)
				g.stateMap[key] = nextStateIdx
				g.transitions = append(g.transitions, make(map[byte]int))
				worklist = append(worklist, nextStateIdx)
			}

			g.transitions[stateIdx][c] = nextStateIdx
		}
	}

	// Compute EOT accept states
	g.computeAcceptStatesEOT()

	return nil
}

// computeAcceptStatesEOT determines which states accept at end of text.
func (g *DFAGenerator) computeAcceptStatesEOT() {
	for i, state := range g.states {
		closure := EpsilonClosure(g.prog, state.NFASet, syntax.EmptyEndText)
		if IsAcceptingNFASet(g.prog, closure) {
			g.acceptStatesEOT[i] = true
		}
	}
}

// getVarName returns a unique variable name for the current regex.
func (g *DFAGenerator) getVarName(base string) string {
	return fmt.Sprintf("%s%s", base, g.compiler.config.Name)
}

// GenerateTables generates the static tables as package-level variables.
func (g *DFAGenerator) GenerateTables() {
	g.compiler.file.Add(g.generateTransitionTable())
	g.compiler.file.Add(g.generateAcceptStates())
	g.compiler.file.Add(g.generateAcceptStatesEOT())
}

// getOptimalIntType returns the name of the smallest integer type that can hold the state count.
// This optimization significantly reduces binary size:
// - int8 for <= 127 states: ~8KB for 500 states max
// - int16 for <= 32767 states: ~128KB for 500 states max
// - int for larger: ~512KB for 500 states max
func (g *DFAGenerator) getOptimalIntType() string {
	numStates := len(g.states)
	if numStates <= 127 {
		return "int8"
	}
	if numStates <= 32767 {
		return "int16"
	}
	return "int"
}

// generateTransitionTable generates the DFA transition table as a 2D array.
// Uses the smallest integer type that can represent the state count to minimize binary size.
func (g *DFAGenerator) generateTransitionTable() jen.Code {
	numStates := len(g.states)
	intType := g.getOptimalIntType()
	rowValues := make([]jen.Code, numStates)

	for stateIdx := 0; stateIdx < numStates; stateIdx++ {
		// Build row for this state - MaxASCIIRune entries for ASCII
		entries := make([]jen.Code, MaxASCIIRune)
		for i := 0; i < MaxASCIIRune; i++ {
			entries[i] = jen.Lit(-1) // No transition
		}

		// Fill in actual transitions
		for c, nextState := range g.transitions[stateIdx] {
			if int(c) < MaxASCIIRune {
				entries[int(c)] = jen.Lit(nextState)
			}
		}

		rowValues[stateIdx] = jen.Index(jen.Lit(MaxASCIIRune)).Id(intType).Values(entries...)
	}

	return jen.Comment("DFA transition table [state][char] -> next state (-1 = no transition)").Line().
		Var().Id(g.getVarName("dfaTransitions")).Op("=").
		Index(jen.Lit(numStates)).Index(jen.Lit(MaxASCIIRune)).Id(intType).Values(rowValues...)
}

// generateAcceptStates generates the accept states check as a boolean array.
func (g *DFAGenerator) generateAcceptStates() jen.Code {
	numStates := len(g.states)
	acceptValues := make([]jen.Code, numStates)

	for i := 0; i < numStates; i++ {
		if g.acceptStates[i] {
			acceptValues[i] = jen.True()
		} else {
			acceptValues[i] = jen.False()
		}
	}

	return jen.Comment("DFA accept states array").Line().
		Var().Id(g.getVarName("dfaAcceptStates")).Op("=").
		Index(jen.Lit(numStates)).Bool().Values(acceptValues...)
}

// generateAcceptStatesEOT generates the accept states EOT check as a boolean array.
func (g *DFAGenerator) generateAcceptStatesEOT() jen.Code {
	numStates := len(g.states)
	acceptValues := make([]jen.Code, numStates)

	for i := 0; i < numStates; i++ {
		if g.acceptStatesEOT[i] {
			acceptValues[i] = jen.True()
		} else {
			acceptValues[i] = jen.False()
		}
	}

	return jen.Comment("DFA accept states array (End of Text)").Line().
		Var().Id(g.getVarName("dfaAcceptStatesEOT")).Op("=").
		Index(jen.Lit(numStates)).Bool().Values(acceptValues...)
}

// GenerateMatchFunction generates the DFA-based MatchString/MatchBytes function body.
func (g *DFAGenerator) GenerateMatchFunction(isBytes bool) []jen.Code {
	g.isBytes = isBytes

	var inputAccess jen.Code
	if isBytes {
		inputAccess = jen.Id("input").Index(jen.Id("i"))
	} else {
		inputAccess = jen.Id("input").Index(jen.Id("i"))
	}

	// Build the matching loop body
	innerLoop := []jen.Code{
		jen.Id("c").Op(":=").Add(inputAccess),
		jen.If(jen.Id("c").Op(">=").Lit(MaxASCIIRune)).Block(
			jen.Break(), // Non-ASCII, exit inner loop
		),
		jen.Line(),
		jen.Id("nextState").Op(":=").Id(g.getVarName("dfaTransitions")).Index(jen.Id("state")).Index(jen.Id("c")),
		jen.If(jen.Id("nextState").Op("<").Lit(0)).Block(
			jen.Break(), // No transition
		),
		jen.Line(),
		jen.Id("state").Op("=").Int().Call(jen.Id("nextState")), // Cast to int for smaller table types
		jen.If(jen.Id(g.getVarName("dfaAcceptStates")).Index(jen.Id("state"))).Block(
			jen.Return(jen.True()),
		),
		jen.Comment("Check EOT acceptance at last character"),
		jen.If(jen.Id("i").Op("==").Id("inputLen").Op("-").Lit(1).Op("&&").
			Id(g.getVarName("dfaAcceptStatesEOT")).Index(jen.Id("state"))).Block(
			jen.Return(jen.True()),
		),
	}

	// Build the outer loop
	outerLoopBody := []jen.Code{
		jen.Var().Id("state").Int(),
		jen.If(jen.Id("start").Op("==").Lit(0)).Block(
			jen.Id("state").Op("=").Lit(g.startStateBegin),
		).Else().Block(
			jen.Id("state").Op("=").Lit(g.startStateAny),
		),
		jen.Line(),
		jen.Comment("Check if start state accepts (empty match)"),
		jen.If(jen.Id(g.getVarName("dfaAcceptStates")).Index(jen.Id("state"))).Block(
			jen.Return(jen.True()),
		),
		jen.Comment("Check EOT acceptance at empty position"),
		jen.If(jen.Id("start").Op("==").Id("inputLen").Op("&&").
			Id(g.getVarName("dfaAcceptStatesEOT")).Index(jen.Id("state"))).Block(
			jen.Return(jen.True()),
		),
		jen.Line(),
		jen.Comment("Process input characters"),
		jen.For(jen.Id("i").Op(":=").Id("start"), jen.Id("i").Op("<").Id("inputLen"), jen.Id("i").Op("++")).Block(
			innerLoop...,
		),
	}

	return []jen.Code{
		jen.Id("inputLen").Op(":=").Len(jen.Id("input")),
		jen.Line(),
		jen.Comment("Try each start position (O(n²) worst case for unanchored patterns)"),
		jen.For(jen.Id("start").Op(":=").Lit(0), jen.Id("start").Op("<=").Id("inputLen"), jen.Id("start").Op("++")).Block(
			outerLoopBody...,
		),
		jen.Return(jen.False()),
	}
}
