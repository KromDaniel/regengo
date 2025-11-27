package compiler

import (
	"fmt"
	"regexp/syntax"
	"sort"

	"github.com/KromDaniel/regengo/internal/codegen"
	"github.com/dave/jennifer/jen"
)

// TDFAGenerator generates Tagged DFA code for capture functions.
// Tagged DFA extends traditional DFA with "tags" that record capture positions.
// This provides O(n) matching with captures, unlike backtracking which can be O(2^n).
type TDFAGenerator struct {
	compiler         *Compiler
	prog             *syntax.Prog
	numCaptures      int
	states           []*TDFAState
	stateMap         map[string]int // Maps NFA state set to DFA state index
	transitions      []map[byte]int // transitions[state][char] = next state
	tagActions       []map[byte][]TagAction
	startStateBegin  int
	startStateAny    int
	acceptStates     map[int]bool
	acceptStatesEOT  map[int]bool
	acceptActions    map[int][]TagAction // Tag actions to apply when accepting at each state
	isBytes          bool
	maxStates        int         // Maximum allowed states before giving up
	initialTagsBegin []TagAction // Tags to set at start of match (at index 0)
	initialTagsAny   []TagAction // Tags to set at start of match (at index > 0)
	useDirectCode    bool        // Use direct-coded generation for small DFAs
}

// NFAStateWithActions represents an NFA state with pending tag actions.
type NFAStateWithActions struct {
	ID      int
	Actions []TagAction
}

// TDFAState represents a state in the Tagged DFA.
// Each TDFA state corresponds to a set of NFA states with associated tag values.
type TDFAState struct {
	ID       int
	NFASet   []NFAStateWithActions // Set of NFA states with pending actions
	Tags     map[int]int           // tag index -> relative position (-1 = unset)
	IsAccept bool
}

// TagAction represents an action to set a capture tag.
type TagAction struct {
	Tag    int // Which tag (0 = group 0 start, 1 = group 0 end, 2 = group 1 start, etc.)
	Offset int // Offset from current position (0 = current, -1 = previous)
}

// NewTDFAGenerator creates a new Tagged DFA generator.
func NewTDFAGenerator(c *Compiler) *TDFAGenerator {
	prog := c.config.Program
	if prog == nil {
		return nil
	}

	// Use configured threshold or default to 500
	maxStates := c.config.TDFAThreshold
	if maxStates <= 0 {
		maxStates = 500
	}

	gen := &TDFAGenerator{
		compiler:        c,
		prog:            prog,
		numCaptures:     prog.NumCap,
		stateMap:        make(map[string]int),
		acceptStates:    make(map[int]bool),
		acceptStatesEOT: make(map[int]bool),
		acceptActions:   make(map[int][]TagAction),
		maxStates:       maxStates,
		useDirectCode:   true, // Use direct-coded for small DFAs (< 64 states)
	}

	return gen
}

// CanUseTDFA returns true if the pattern can use Tagged DFA.
func (g *TDFAGenerator) CanUseTDFA() bool {
	// Check for unsupported instructions (e.g. word boundaries)
	for _, inst := range g.prog.Inst {
		if inst.Op == syntax.InstEmptyWidth {
			// We only support BeginText (^) and EndText ($) for now
			if syntax.EmptyOp(inst.Arg) != syntax.EmptyBeginText && syntax.EmptyOp(inst.Arg) != syntax.EmptyEndText {
				g.compiler.logger.Log("TDFA unsupported empty width op: %v", inst.Arg)
				return false
			}
		}
	}

	// Build TDFA and check if state count is acceptable
	if err := g.buildTDFA(); err != nil {
		g.compiler.logger.Log("TDFA construction failed: %v", err)
		return false
	}

	if len(g.states) > g.maxStates {
		g.compiler.logger.Log("TDFA has too many states (%d > %d), falling back", len(g.states), g.maxStates)
		return false
	}

	g.compiler.logger.Log("TDFA constructed with %d states", len(g.states))
	return true
}

// buildTDFA constructs the Tagged DFA from the NFA.
func (g *TDFAGenerator) buildTDFA() error {
	// 1. Create startStateBegin (at index 0)
	startNFA := []NFAStateWithActions{{ID: int(g.prog.Start), Actions: nil}}
	startNFASetBegin := g.epsilonClosureWithCaptures(startNFA, true, syntax.EmptyBeginText)

	if len(startNFASetBegin) > 0 {
		g.initialTagsBegin = startNFASetBegin[0].Actions
	} else {
		g.initialTagsBegin = nil
	}

	// Create start state begin
	startStateBegin := &TDFAState{
		ID:     0,
		NFASet: startNFASetBegin,
		Tags:   make(map[int]int),
	}

	g.states = []*TDFAState{startStateBegin}
	g.stateMap[g.nfaSetKey(startNFASetBegin)] = 0
	g.transitions = []map[byte]int{{}}
	g.tagActions = []map[byte][]TagAction{{}}
	g.startStateBegin = 0

	// Check if start state begin is accepting
	for _, nfaState := range startNFASetBegin {
		if g.prog.Inst[nfaState.ID].Op == syntax.InstMatch {
			startStateBegin.IsAccept = true
			g.acceptStates[0] = true
			break
		}
	}

	// 2. Create startStateAny
	startNFASetAny := g.epsilonClosureWithCaptures(startNFA, true, 0)

	if len(startNFASetAny) > 0 {
		g.initialTagsAny = startNFASetAny[0].Actions
	} else {
		g.initialTagsAny = nil
	}

	// Check if it's the same as startStateBegin
	keyAny := g.nfaSetKey(startNFASetAny)
	if idx, exists := g.stateMap[keyAny]; exists {
		g.startStateAny = idx
	} else {
		// Create new state
		idx = len(g.states)
		startStateAny := &TDFAState{
			ID:     idx,
			NFASet: startNFASetAny,
			Tags:   make(map[int]int),
		}
		g.states = append(g.states, startStateAny)
		g.stateMap[keyAny] = idx
		g.transitions = append(g.transitions, make(map[byte]int))
		g.tagActions = append(g.tagActions, make(map[byte][]TagAction))
		g.startStateAny = idx

		// Check acceptance
		for _, nfaState := range startNFASetAny {
			if g.prog.Inst[nfaState.ID].Op == syntax.InstMatch {
				startStateAny.IsAccept = true
				g.acceptStates[idx] = true
				break
			}
		}
	}

	// Process states using worklist algorithm
	worklist := []int{0}
	if g.startStateAny != 0 {
		worklist = append(worklist, g.startStateAny)
	}
	processed := make(map[int]bool)

	for len(worklist) > 0 {
		stateIdx := worklist[0]
		worklist = worklist[1:]

		if processed[stateIdx] {
			continue
		}
		processed[stateIdx] = true

		state := g.states[stateIdx]

		// Find all possible input characters from this state
		chars := g.getPossibleChars(state.NFASet)

		for _, c := range chars {
			// Compute next NFA state set and tag actions
			nextNFASet, actions := g.computeTransition(state.NFASet, c)

			if len(nextNFASet) == 0 {
				continue
			}

			// Check if this NFA set already exists as a DFA state
			key := g.nfaSetKey(nextNFASet)
			nextStateIdx, exists := g.stateMap[key]

			if !exists {
				// Create new DFA state
				nextStateIdx = len(g.states)
				if nextStateIdx >= g.maxStates {
					return fmt.Errorf("TDFA state explosion: exceeded %d states", g.maxStates)
				}

				nextState := &TDFAState{
					ID:     nextStateIdx,
					NFASet: nextNFASet,
					Tags:   make(map[int]int),
				}

				// Check if accepting
				for _, nfaState := range nextNFASet {
					if g.prog.Inst[nfaState.ID].Op == syntax.InstMatch {
						nextState.IsAccept = true
						g.acceptStates[nextStateIdx] = true
						break
					}
				}

				g.states = append(g.states, nextState)
				g.stateMap[key] = nextStateIdx
				g.transitions = append(g.transitions, make(map[byte]int))
				g.tagActions = append(g.tagActions, make(map[byte][]TagAction))
				worklist = append(worklist, nextStateIdx)
			}

			// Record transition and tag actions
			g.transitions[stateIdx][c] = nextStateIdx
			if len(actions) > 0 {
				g.tagActions[stateIdx][c] = actions
			}
		}
	}

	// Compute acceptStatesEOT and acceptActions
	for i, state := range g.states {
		// Check EOT acceptance
		// We use a temporary closure with EmptyEndText
		closure := g.epsilonClosureWithCaptures(state.NFASet, true, syntax.EmptyEndText)
		for _, s := range closure {
			if g.prog.Inst[s.ID].Op == syntax.InstMatch {
				g.acceptStatesEOT[i] = true
				// Collect pending tag actions for this accept state
				// These are actions that need to be applied when accepting
				if len(s.Actions) > 0 {
					g.acceptActions[i] = g.compactActions(s.Actions)
				}
				break
			}
		}
	}

	// Also compute acceptActions for regular accept states (not just EOT)
	for i, state := range g.states {
		if !state.IsAccept {
			continue
		}
		// Skip if already computed via EOT
		if _, exists := g.acceptActions[i]; exists {
			continue
		}
		// Find the NFA state that reaches Match
		for _, s := range state.NFASet {
			if g.prog.Inst[s.ID].Op == syntax.InstMatch {
				if len(s.Actions) > 0 {
					g.acceptActions[i] = g.compactActions(s.Actions)
				}
				break
			}
		}
	}

	return nil
}

// getPossibleChars returns all possible input characters from the given NFA states.
func (g *TDFAGenerator) getPossibleChars(nfaSet []NFAStateWithActions) []byte {
	charSet := make(map[byte]bool)

	for _, state := range nfaSet {
		inst := g.prog.Inst[state.ID]
		switch inst.Op {
		case syntax.InstRune1:
			if len(inst.Rune) > 0 && inst.Rune[0] < 128 {
				charSet[byte(inst.Rune[0])] = true
			}
		case syntax.InstRune:
			// Character class - add all characters in ranges
			for i := 0; i < len(inst.Rune); i += 2 {
				lo, hi := inst.Rune[i], inst.Rune[i+1]
				if lo < 128 {
					end := hi
					if end >= 128 {
						end = 127
					}
					for c := lo; c <= end; c++ {
						charSet[byte(c)] = true
					}
				}
			}
		case syntax.InstRuneAny:
			// Match any - add all printable ASCII for now
			for c := byte(0); c < 128; c++ {
				charSet[c] = true
			}
		case syntax.InstRuneAnyNotNL:
			for c := byte(0); c < 128; c++ {
				if c != '\n' {
					charSet[c] = true
				}
			}
		}
	}

	chars := make([]byte, 0, len(charSet))
	for c := range charSet {
		chars = append(chars, c)
	}
	sort.Slice(chars, func(i, j int) bool { return chars[i] < chars[j] })
	return chars
}

// computeTransition computes the next NFA state set after consuming character c.
func (g *TDFAGenerator) computeTransition(nfaSet []NFAStateWithActions, c byte) ([]NFAStateWithActions, []TagAction) {
	nextStates := make([]NFAStateWithActions, 0)

	for _, state := range nfaSet {
		inst := g.prog.Inst[state.ID]
		matches := false

		switch inst.Op {
		case syntax.InstRune1:
			if len(inst.Rune) > 0 {
				r := inst.Rune[0]
				if r < 128 && byte(r) == c {
					matches = true
				}
			}
		case syntax.InstRune:
			for i := 0; i < len(inst.Rune); i += 2 {
				lo, hi := inst.Rune[i], inst.Rune[i+1]
				if rune(c) >= lo && rune(c) <= hi {
					matches = true
					break
				}
			}
		case syntax.InstRuneAny:
			matches = true
		case syntax.InstRuneAnyNotNL:
			matches = c != '\n'
		}

		if matches {
			// Propagate actions with incremented offset
			propagatedActions := make([]TagAction, len(state.Actions))
			for k, a := range state.Actions {
				propagatedActions[k] = TagAction{Tag: a.Tag, Offset: a.Offset + 1}
			}
			nextStates = append(nextStates, NFAStateWithActions{ID: int(inst.Out), Actions: propagatedActions})
		}
	}

	if len(nextStates) == 0 {
		return nil, nil
	}

	// Compute epsilon closure of next states
	// Pass true to collect all tags (START and END)
	result := g.epsilonClosureWithCaptures(nextStates, true, 0)

	// Find common actions
	if len(result) == 0 {
		return nil, nil
	}

	// Start with actions of the first state (which is the highest priority one due to epsilonClosure order)
	common := result[0].Actions

	for i := 1; i < len(result); i++ {
		common = g.longestCommonPrefix(common, result[i].Actions)
		if len(common) == 0 {
			break
		}
	}

	if len(common) > 0 {
		// Remove common actions from all states
		for i := range result {
			result[i].Actions = result[i].Actions[len(common):]
		}
	}

	return result, common
}

func (g *TDFAGenerator) longestCommonPrefix(a, b []TagAction) []TagAction {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}
	for i := 0; i < minLen; i++ {
		if a[i] != b[i] {
			return a[:i]
		}
	}
	return a[:minLen]
}

// compactActions removes redundant tag actions, keeping only the last action for each tag.
func (g *TDFAGenerator) compactActions(actions []TagAction) []TagAction {
	if len(actions) == 0 {
		return nil
	}

	// Keep only the last occurrence of each tag
	lastSeen := make(map[int]TagAction)
	for _, a := range actions {
		lastSeen[a.Tag] = a
	}

	result := make([]TagAction, 0, len(lastSeen))
	for _, a := range lastSeen {
		result = append(result, a)
	}

	// Sort for canonical order
	sort.Slice(result, func(i, j int) bool { return result[i].Tag < result[j].Tag })
	return result
}

// epsilonClosureWithCaptures computes epsilon closure and collects capture tag actions.
// It returns the set of reachable states with their pending actions.
func (g *TDFAGenerator) epsilonClosureWithCaptures(states []NFAStateWithActions, collectStartTags bool, matchFlags syntax.EmptyOp) []NFAStateWithActions {
	visited := make(map[int]bool)
	result := make([]NFAStateWithActions, 0, len(states)*2)

	// Initialize stack with states in reverse order so that we process the first state first (LIFO)
	// This preserves the priority of the input states (Greedy > Lazy)
	stack := make([]NFAStateWithActions, 0, len(states))
	for i := len(states) - 1; i >= 0; i-- {
		stack = append(stack, states[i])
	}

	for len(stack) > 0 {
		state := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		// Compact actions to ensure canonical form and prevent infinite growth
		state.Actions = g.compactActions(state.Actions)

		// Use only the state ID for visited check to implement Leftmost-Greedy disambiguation.
		// If we reach the same NFA state via multiple paths, we keep only the first one (which is the highest priority one).
		if visited[state.ID] || state.ID >= len(g.prog.Inst) {
			continue
		}
		visited[state.ID] = true
		result = append(result, state)

		inst := g.prog.Inst[state.ID]
		switch inst.Op {
		case syntax.InstNop:
			stack = append(stack, NFAStateWithActions{ID: int(inst.Out), Actions: state.Actions})
		case syntax.InstCapture:
			// Record capture tag action
			// Even tag indices are START tags, odd are END tags
			tagIdx := int(inst.Arg)
			isStartTag := tagIdx%2 == 0

			newActions := make([]TagAction, len(state.Actions))
			copy(newActions, state.Actions)

			if !isStartTag || collectStartTags {
				newActions = append(newActions, TagAction{Tag: tagIdx, Offset: 0})
			}
			// Compact will happen when popped from stack
			stack = append(stack, NFAStateWithActions{ID: int(inst.Out), Actions: newActions})
		case syntax.InstAlt:
			// Priority: try Out first (greedy), then Arg
			// Push Arg first so Out is popped first
			stack = append(stack,
				NFAStateWithActions{ID: int(inst.Arg), Actions: state.Actions},
				NFAStateWithActions{ID: int(inst.Out), Actions: state.Actions})
		case syntax.InstEmptyWidth:
			// Check if the empty width condition is satisfied
			if syntax.EmptyOp(inst.Arg)&matchFlags == syntax.EmptyOp(inst.Arg) {
				stack = append(stack, NFAStateWithActions{ID: int(inst.Out), Actions: state.Actions})
			}
		}
	}

	// Do not sort by ID here, as it would destroy the priority order.
	// The result is implicitly ordered by priority (Greedy first).
	return result
}

// nfaSetKey creates a unique string key for an NFA state set.
func (g *TDFAGenerator) nfaSetKey(states []NFAStateWithActions) string {
	// Sort a copy to ensure canonical key
	sorted := make([]NFAStateWithActions, len(states))
	copy(sorted, states)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ID < sorted[j].ID
	})

	key := ""
	for i, s := range sorted {
		if i > 0 {
			key += ","
		}
		key += fmt.Sprintf("%d", s.ID)
		if len(s.Actions) > 0 {
			key += "["
			for j, a := range s.Actions {
				if j > 0 {
					key += ";"
				}
				key += fmt.Sprintf("%d:%d", a.Tag, a.Offset)
			}
			key += "]"
		}
	}
	return key
}

// GenerateFindFunction generates the TDFA-based FindString function.
func (g *TDFAGenerator) GenerateFindFunction(isBytes bool) ([]jen.Code, error) {
	g.isBytes = isBytes
	g.compiler.logger.Section("Code Generation")
	g.compiler.logger.Log("Generating table-driven TDFA (states: %d, captures: %d)", len(g.states), g.numCaptures)

	code := []jen.Code{
		jen.Id(codegen.InputLenName).Op(":=").Len(jen.Id(codegen.InputName)),
	}

	// Generate transition table
	code = append(code, g.generateTransitionTable()...)

	// Generate tag action table
	code = append(code, g.generateTagActionTable()...)

	// Generate accept states set
	code = append(code, g.generateAcceptStates()...)

	// Generate accept states EOT set
	code = append(code, g.generateAcceptStatesEOT()...)

	// Generate accept actions map (for finalizing captures at accept states)
	code = append(code, g.generateAcceptActionsMap()...)

	// Generate main matching loop
	code = append(code, g.generateMatchingLoop()...)

	return code, nil
}

// generateTransitionTable generates the DFA transition table as a 2D array.
func (g *TDFAGenerator) generateTransitionTable() []jen.Code {
	numStates := len(g.states)

	// Build transition table as [numStates][128]int where -1 means no transition
	rowValues := make([]jen.Code, numStates)

	for stateIdx := 0; stateIdx < numStates; stateIdx++ {
		// Build row for this state - 128 entries for ASCII
		entries := make([]jen.Code, 128)
		for i := 0; i < 128; i++ {
			entries[i] = jen.Lit(-1)
		}

		// Fill in actual transitions
		if stateIdx < len(g.transitions) {
			for c, nextState := range g.transitions[stateIdx] {
				if int(c) < 128 {
					entries[int(c)] = jen.Lit(nextState)
				}
			}
		}

		rowValues[stateIdx] = jen.Index(jen.Lit(128)).Int().Values(entries...)
	}

	return []jen.Code{
		jen.Comment("TDFA transition table [state][char] -> next state (-1 = no transition)"),
		jen.Id("transitions").Op(":=").Index(jen.Lit(numStates)).Index(jen.Lit(128)).Int().Values(rowValues...),
	}
}

// generateTagActionTable generates the tag action table.
func (g *TDFAGenerator) generateTagActionTable() []jen.Code {
	tagCount := g.getTagCount()

	// Define the action struct type
	actionType := jen.Struct(jen.Id("Tag").Int(), jen.Id("Offset").Int())

	// Build tag action table as map[int]map[byte][]Action
	actionTableEntries := make([]jen.Code, 0)

	for stateIdx, actions := range g.tagActions {
		if len(actions) == 0 {
			continue
		}

		charEntries := make([]jen.Code, 0)
		for c, tagActions := range actions {
			if len(tagActions) == 0 {
				continue
			}

			actionStructs := make([]jen.Code, len(tagActions))
			for i, action := range tagActions {
				actionStructs[i] = jen.Values(jen.Dict{
					jen.Id("Tag"):    jen.Lit(action.Tag),
					jen.Id("Offset"): jen.Lit(action.Offset),
				})
			}

			charEntries = append(charEntries,
				jen.Lit(int(c)).Op(":").Index().Add(actionType).Values(actionStructs...),
			)
		}

		if len(charEntries) > 0 {
			actionTableEntries = append(actionTableEntries,
				jen.Lit(stateIdx).Op(":").Map(jen.Byte()).Index().Add(actionType).Values(charEntries...),
			)
		}
	}

	code := []jen.Code{
		jen.Comment("Tag registers for capture positions"),
		jen.Id("tags").Op(":=").Index(jen.Lit(tagCount)).Int().Values(),
		jen.For(jen.Id("i").Op(":=").Range().Id("tags")).Block(
			jen.Id("tags").Index(jen.Id("i")).Op("=").Lit(-1),
		),
	}

	// Add tag action table if there are any actions
	if len(actionTableEntries) > 0 {
		code = append(code,
			jen.Comment("Tag actions per transition"),
			jen.Id("tagActions").Op(":=").Map(jen.Int()).Map(jen.Byte()).Index().Add(actionType).Values(actionTableEntries...),
		)
	} else {
		// Empty map if no tag actions
		code = append(code,
			jen.Comment("Tag actions per transition (none for this pattern)"),
			jen.Id("tagActions").Op(":=").Map(jen.Int()).Map(jen.Byte()).Index().Add(actionType).Values(),
		)
	}

	return code
}

// generateAcceptStates generates the accept states check as a boolean array.
func (g *TDFAGenerator) generateAcceptStates() []jen.Code {
	numStates := len(g.states)
	acceptValues := make([]jen.Code, numStates)

	for i := 0; i < numStates; i++ {
		if g.acceptStates[i] {
			acceptValues[i] = jen.True()
		} else {
			acceptValues[i] = jen.False()
		}
	}

	return []jen.Code{
		jen.Comment("Accept states array"),
		jen.Id("acceptStates").Op(":=").Index(jen.Lit(numStates)).Bool().Values(acceptValues...),
	}
}

// generateAcceptStatesEOT generates the accept states EOT check as a boolean array.
func (g *TDFAGenerator) generateAcceptStatesEOT() []jen.Code {
	numStates := len(g.states)
	acceptValues := make([]jen.Code, numStates)

	for i := 0; i < numStates; i++ {
		if g.acceptStatesEOT[i] {
			acceptValues[i] = jen.True()
		} else {
			acceptValues[i] = jen.False()
		}
	}

	return []jen.Code{
		jen.Comment("Accept states array (End of Text)"),
		jen.Id("acceptStatesEOT").Op(":=").Index(jen.Lit(numStates)).Bool().Values(acceptValues...),
	}
}

// generateAcceptActionsMap generates the accept actions map for finalizing captures at accept states.
func (g *TDFAGenerator) generateAcceptActionsMap() []jen.Code {
	actionType := jen.Struct(
		jen.Id("Tag").Int(),
		jen.Id("Offset").Int(),
	)

	entries := make([]jen.Code, 0)
	for stateIdx, actions := range g.acceptActions {
		if len(actions) == 0 {
			continue
		}
		actionStructs := make([]jen.Code, len(actions))
		for i, action := range actions {
			actionStructs[i] = jen.Values(jen.Dict{
				jen.Id("Tag"):    jen.Lit(action.Tag),
				jen.Id("Offset"): jen.Lit(action.Offset),
			})
		}
		entries = append(entries,
			jen.Lit(stateIdx).Op(":").Index().Add(actionType).Values(actionStructs...),
		)
	}

	return []jen.Code{
		jen.Comment("Accept actions - tag actions to apply when accepting at each state"),
		jen.Id("acceptActionsMap").Op(":=").Map(jen.Int()).Index().Add(actionType).Values(entries...),
	}
}

// getTagCount returns the number of tags needed for capture groups.
func (g *TDFAGenerator) getTagCount() int {
	numGroups := len(g.compiler.captureNames)
	if numGroups == 0 {
		numGroups = 1
	}
	return numGroups * 2
}

// generateMatchingLoop generates the main TDFA matching loop.
func (g *TDFAGenerator) generateMatchingLoop() []jen.Code {
	var inputAccess jen.Code
	if g.isBytes {
		inputAccess = jen.Id(codegen.InputName).Index(jen.Id("i"))
	} else {
		inputAccess = jen.Id(codegen.InputName).Index(jen.Id("i"))
	}

	tagCount := g.getTagCount()

	// Generate initial tag setup code
	setupBegin := []jen.Code{
		jen.Id("state").Op("=").Lit(g.startStateBegin),
	}
	for _, action := range g.initialTagsBegin {
		setupBegin = append(setupBegin, jen.Id("tags").Index(jen.Lit(action.Tag)).Op("=").Id("start"))
	}

	setupAny := []jen.Code{
		jen.Id("state").Op("=").Lit(g.startStateAny),
	}
	for _, action := range g.initialTagsAny {
		setupAny = append(setupAny, jen.Id("tags").Index(jen.Lit(action.Tag)).Op("=").Id("start"))
	}

	// Build the outer loop body
	outerLoopBody := []jen.Code{
		jen.Id("state").Op(":=").Lit(0),
		jen.Id("matchEnd").Op(":=").Lit(-1),
		jen.Id("matchTags").Op(":=").Index(jen.Lit(tagCount)).Int().Values(),
		jen.For(jen.Id("i").Op(":=").Range().Id("matchTags")).Block(
			jen.Id("matchTags").Index(jen.Id("i")).Op("=").Lit(-1),
		),
		jen.Line(),
		jen.Comment("Iterate over all possible start positions"),
		jen.For(jen.Id("start").Op(":=").Lit(0), jen.Id("start").Op("<=").Id(codegen.InputLenName), jen.Id("start").Op("++")).Block(
			jen.Comment("Reset tags for this attempt"),
			jen.For(jen.Id("i").Op(":=").Range().Id("tags")).Block(
				jen.Id("tags").Index(jen.Id("i")).Op("=").Lit(-1),
			),
			jen.Id("tags").Index(jen.Lit(0)).Op("=").Id("start"), // Group 0 start
			jen.Line(),
			jen.Comment("Choose start state based on position"),
			jen.If(jen.Id("start").Op("==").Lit(0)).Block(setupBegin...).Else().Block(setupAny...),
			jen.Line(),
			jen.Comment("Check if start state is accepting (empty match)"),
			jen.If(jen.Id("acceptStates").Index(jen.Id("state"))).Block(
				jen.Id("matchEnd").Op("=").Id("start"),
				jen.Id("matchTags").Op("=").Id("tags"),
			),
			jen.Comment("Check if start state is accepting at EOT"),
			jen.If(jen.Id("start").Op("==").Id(codegen.InputLenName).Op("&&").Id("acceptStatesEOT").Index(jen.Id("state"))).Block(
				jen.Id("matchEnd").Op("=").Id("start"),
				jen.Id("matchTags").Op("=").Id("tags"),
			),
			jen.Line(),
			jen.For(jen.Id("i").Op(":=").Id("start"), jen.Id("i").Op("<").Id(codegen.InputLenName), jen.Id("i").Op("++")).Block(
				jen.Id("c").Op(":=").Add(inputAccess),
				jen.If(jen.Id("c").Op(">=").Lit(128)).Block(
					jen.Break(), // Non-ASCII, exit
				),
				jen.Line(),
				jen.Comment("Look up transition (array-based for speed)"),
				jen.Id("nextState").Op(":=").Id("transitions").Index(jen.Id("state")).Index(jen.Id("c")),
				jen.If(jen.Id("nextState").Op("<").Lit(0)).Block(
					jen.Break(),
				),
				jen.Line(),
				jen.Comment("Apply tag actions for this transition"),
				jen.If(jen.List(jen.Id("stateActions"), jen.Id("ok")).Op(":=").Id("tagActions").Index(jen.Id("state")), jen.Id("ok")).Block(
					jen.If(jen.List(jen.Id("charActions"), jen.Id("ok")).Op(":=").Id("stateActions").Index(jen.Id("c")), jen.Id("ok")).Block(
						jen.For(jen.List(jen.Id("_"), jen.Id("action")).Op(":=").Range().Id("charActions")).Block(
							jen.Id("tags").Index(jen.Id("action").Dot("Tag")).Op("=").Id("i").Op("+").Lit(1).Op("-").Id("action").Dot("Offset"),
						),
					),
				),
				jen.Line(),
				jen.Id("state").Op("=").Id("nextState"),
				jen.If(jen.Id("acceptStates").Index(jen.Id("state"))).Block(
					// Apply accept actions to finalize captures
					jen.If(jen.List(jen.Id("actions"), jen.Id("ok")).Op(":=").Id("acceptActionsMap").Index(jen.Id("state")), jen.Id("ok")).Block(
						jen.For(jen.List(jen.Id("_"), jen.Id("action")).Op(":=").Range().Id("actions")).Block(
							jen.Id("tags").Index(jen.Id("action").Dot("Tag")).Op("=").Id("i").Op("+").Lit(1).Op("-").Id("action").Dot("Offset"),
						),
					),
					jen.Id("matchEnd").Op("=").Id("i").Op("+").Lit(1),
					jen.Id("matchTags").Op("=").Id("tags"),
				),
				jen.If(jen.Id("i").Op("==").Id(codegen.InputLenName).Op("-").Lit(1).Op("&&").Id("acceptStatesEOT").Index(jen.Id("state"))).Block(
					// Apply accept actions to finalize captures at EOT
					jen.If(jen.List(jen.Id("actions"), jen.Id("ok")).Op(":=").Id("acceptActionsMap").Index(jen.Id("state")), jen.Id("ok")).Block(
						jen.For(jen.List(jen.Id("_"), jen.Id("action")).Op(":=").Range().Id("actions")).Block(
							jen.Id("tags").Index(jen.Id("action").Dot("Tag")).Op("=").Id("i").Op("+").Lit(1).Op("-").Id("action").Dot("Offset"),
						),
					),
					jen.Id("matchEnd").Op("=").Id("i").Op("+").Lit(1),
					jen.Id("matchTags").Op("=").Id("tags"),
				),
			),
			jen.Line(),
			jen.Comment("If we found a match, return it"),
			jen.If(jen.Id("matchEnd").Op(">=").Lit(0)).Block(
				append(
					[]jen.Code{jen.Id("matchTags").Index(jen.Lit(1)).Op("=").Id("matchEnd")}, // Group 0 end
					g.generateResultConstruction("matchTags")...,
				)...,
			),
		),
		jen.Return(jen.Id(g.compiler.config.Name+"Result").Values(), jen.False()),
	}

	if g.isBytes {
		outerLoopBody[len(outerLoopBody)-1] = jen.Return(jen.Id(g.compiler.config.Name+"BytesResult").Values(), jen.False())
	}

	return outerLoopBody
}

// generateResultConstruction generates code to construct the result struct from tags.
func (g *TDFAGenerator) generateResultConstruction(tagsVar string) []jen.Code {
	structName := g.compiler.config.Name + "Result"
	if g.isBytes {
		structName = g.compiler.config.Name + "BytesResult"
	}

	// Create result variable
	code := []jen.Code{
		jen.Var().Id("result").Id(structName),
	}

	// Helper to get slice expression
	getSlice := func(start, end *jen.Statement) jen.Code {
		// We need to clone the statements to avoid modifying them if they are reused (though here they are fresh)
		// But jen.Statement is mutable.
		// Actually, start.Op(":") modifies start.
		// We should construct the index expression carefully.
		// input[start:end]
		return jen.Id(codegen.InputName).Index(start.Clone().Op(":").Add(end))
	}

	// Set Match field (Group 0)
	code = append(code,
		jen.If(jen.Id(tagsVar).Index(jen.Lit(0)).Op(">=").Lit(0)).Block(
			jen.Id("result").Dot("Match").Op("=").Add(getSlice(
				jen.Id(tagsVar).Index(jen.Lit(0)),
				jen.Id(tagsVar).Index(jen.Lit(1)),
			)),
		),
	)

	// Set other fields
	usedNames := make(map[string]bool)
	usedNames["Match"] = true

	for i := 1; i < len(g.compiler.captureNames); i++ {
		fieldName := g.compiler.captureNames[i]
		if fieldName == "" {
			fieldName = fmt.Sprintf("Group%d", i)
		} else {
			fieldName = codegen.UpperFirst(fieldName)
		}
		if usedNames[fieldName] {
			fieldName = fmt.Sprintf("%s%d", fieldName, i)
		}
		usedNames[fieldName] = true

		// Generate assignment
		// if tags[2*i] >= 0 {
		//     if tags[2*i+1] < 0 { tags[2*i+1] = tags[1] } // Close open group at match end
		//     result.Field = input[tags[2*i]:tags[2*i+1]]
		// }
		code = append(code,
			jen.If(jen.Id(tagsVar).Index(jen.Lit(2*i)).Op(">=").Lit(0)).Block(
				jen.If(jen.Id(tagsVar).Index(jen.Lit(2*i+1)).Op("<").Lit(0)).Block(
					jen.Id(tagsVar).Index(jen.Lit(2*i+1)).Op("=").Id(tagsVar).Index(jen.Lit(1)),
				),
				jen.Id("result").Dot(fieldName).Op("=").Add(getSlice(
					jen.Id(tagsVar).Index(jen.Lit(2*i)),
					jen.Id(tagsVar).Index(jen.Lit(2*i+1)),
				)),
			),
		)
	}

	code = append(code, jen.Return(jen.Id("result"), jen.True()))
	return code
}
