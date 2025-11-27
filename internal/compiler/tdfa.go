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

// getVarName returns a unique variable name for the current regex.
func (g *TDFAGenerator) getVarName(base string) string {
	return fmt.Sprintf("%s%s", base, g.compiler.config.Name)
}

// GenerateTables generates the static tables as package-level variables.
func (g *TDFAGenerator) GenerateTables() {
	// Generate transition table
	g.compiler.file.Add(g.generateTransitionTable())

	// Generate tag action tables
	g.compiler.file.Add(g.generateTagActionTables()...)

	// Generate accept states
	g.compiler.file.Add(g.generateAcceptStates())

	// Generate accept states EOT
	g.compiler.file.Add(g.generateAcceptStatesEOT())

	// Generate accept actions map
	g.compiler.file.Add(g.generateAcceptActionsMap()...)
}

// GenerateFindFunction generates the TDFA-based FindString function.
func (g *TDFAGenerator) GenerateFindFunction(isBytes bool) ([]jen.Code, error) {
	g.isBytes = isBytes
	g.compiler.logger.Section("Code Generation")
	g.compiler.logger.Log("Generating table-driven TDFA (states: %d, captures: %d)", len(g.states), g.numCaptures)

	code := []jen.Code{
		jen.Id(codegen.InputLenName).Op(":=").Len(jen.Id(codegen.InputName)),
	}

	// Generate local tag state
	code = append(code, g.generateLocalTagState()...)

	// Generate main matching loop
	code = append(code, g.generateMatchingLoop()...)

	return code, nil
}

// generateTransitionTable generates the DFA transition table as a 2D array.
func (g *TDFAGenerator) generateTransitionTable() jen.Code {
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

	return jen.Comment("TDFA transition table [state][char] -> next state (-1 = no transition)").Line().
		Var().Id(g.getVarName("transitions")).Op("=").Index(jen.Lit(numStates)).Index(jen.Lit(128)).Int().Values(rowValues...)
}

// generateLocalTagState generates the local tag variables.
func (g *TDFAGenerator) generateLocalTagState() []jen.Code {
	tagCount := g.getTagCount()
	return []jen.Code{
		jen.Comment("Tag registers for capture positions (pre-allocated)"),
		jen.Var().Id("tags").Index(jen.Lit(tagCount)).Int(),
		jen.Comment("Match tags (pre-allocated outside loop to avoid allocations)"),
		jen.Var().Id("matchTags").Index(jen.Lit(tagCount)).Int(),
	}
}

// generateTagActionTables generates the tag action tables as package-level variables.
func (g *TDFAGenerator) generateTagActionTables() []jen.Code {
	numStates := len(g.states)
	maxActions := g.getMaxActionsPerTransition()

	// If no tag actions, skip the tables
	if maxActions == 0 {
		return nil
	}

	// Generate action count table: tagActionCount[state][char] = number of actions
	countRows := make([]jen.Code, numStates)
	for stateIdx := 0; stateIdx < numStates; stateIdx++ {
		entries := make([]jen.Code, 128)
		for i := 0; i < 128; i++ {
			entries[i] = jen.Lit(0)
		}
		if stateIdx < len(g.tagActions) {
			for c, actions := range g.tagActions[stateIdx] {
				if int(c) < 128 {
					entries[int(c)] = jen.Lit(len(actions))
				}
			}
		}
		countRows[stateIdx] = jen.Index(jen.Lit(128)).Int().Values(entries...)
	}

	code := []jen.Code{
		jen.Comment("Tag action counts [state][char] -> number of actions").Line().
			Var().Id(g.getVarName("tagActionCount")).Op("=").Index(jen.Lit(numStates)).Index(jen.Lit(128)).Int().Values(countRows...),
	}

	// Generate tag action table: tagActionTags[state][char][actionIdx] = tag index
	// Generate tag action offsets: tagActionOffsets[state][char][actionIdx] = offset
	tagRows := make([]jen.Code, numStates)
	offsetRows := make([]jen.Code, numStates)

	for stateIdx := 0; stateIdx < numStates; stateIdx++ {
		charTagEntries := make([]jen.Code, 128)
		charOffsetEntries := make([]jen.Code, 128)

		for i := 0; i < 128; i++ {
			// Initialize with zeros
			tagActionEntries := make([]jen.Code, maxActions)
			offsetActionEntries := make([]jen.Code, maxActions)
			for j := 0; j < maxActions; j++ {
				tagActionEntries[j] = jen.Lit(0)
				offsetActionEntries[j] = jen.Lit(0)
			}

			// Fill in actual actions
			if stateIdx < len(g.tagActions) {
				if actions, ok := g.tagActions[stateIdx][byte(i)]; ok {
					for j, action := range actions {
						if j < maxActions {
							tagActionEntries[j] = jen.Lit(action.Tag)
							offsetActionEntries[j] = jen.Lit(action.Offset)
						}
					}
				}
			}

			charTagEntries[i] = jen.Index(jen.Lit(maxActions)).Int().Values(tagActionEntries...)
			charOffsetEntries[i] = jen.Index(jen.Lit(maxActions)).Int().Values(offsetActionEntries...)
		}

		tagRows[stateIdx] = jen.Index(jen.Lit(128)).Index(jen.Lit(maxActions)).Int().Values(charTagEntries...)
		offsetRows[stateIdx] = jen.Index(jen.Lit(128)).Index(jen.Lit(maxActions)).Int().Values(charOffsetEntries...)
	}

	code = append(code,
		jen.Comment("Tag action tags [state][char][actionIdx] -> tag index").Line().
			Var().Id(g.getVarName("tagActionTags")).Op("=").Index(jen.Lit(numStates)).Index(jen.Lit(128)).Index(jen.Lit(maxActions)).Int().Values(tagRows...),
		jen.Comment("Tag action offsets [state][char][actionIdx] -> offset").Line().
			Var().Id(g.getVarName("tagActionOffsets")).Op("=").Index(jen.Lit(numStates)).Index(jen.Lit(128)).Index(jen.Lit(maxActions)).Int().Values(offsetRows...),
	)

	return code
}

// generateAcceptStates generates the accept states check as a boolean array.
func (g *TDFAGenerator) generateAcceptStates() jen.Code {
	numStates := len(g.states)
	acceptValues := make([]jen.Code, numStates)

	for i := 0; i < numStates; i++ {
		if g.acceptStates[i] {
			acceptValues[i] = jen.True()
		} else {
			acceptValues[i] = jen.False()
		}
	}

	return jen.Comment("Accept states array").Line().
		Var().Id(g.getVarName("acceptStates")).Op("=").Index(jen.Lit(numStates)).Bool().Values(acceptValues...)
}

// generateAcceptStatesEOT generates the accept states EOT check as a boolean array.
func (g *TDFAGenerator) generateAcceptStatesEOT() jen.Code {
	numStates := len(g.states)
	acceptValues := make([]jen.Code, numStates)

	for i := 0; i < numStates; i++ {
		if g.acceptStatesEOT[i] {
			acceptValues[i] = jen.True()
		} else {
			acceptValues[i] = jen.False()
		}
	}

	return jen.Comment("Accept states array (End of Text)").Line().
		Var().Id(g.getVarName("acceptStatesEOT")).Op("=").Index(jen.Lit(numStates)).Bool().Values(acceptValues...)
}

// generateAcceptActionsMap generates the accept actions using arrays instead of maps.
// This eliminates map allocations when checking accept states.
func (g *TDFAGenerator) generateAcceptActionsMap() []jen.Code {
	numStates := len(g.states)
	maxActions := g.getMaxAcceptActions()

	// If no accept actions, return empty
	if maxActions == 0 {
		return nil
	}

	// Generate accept action count array: acceptActionCount[state] = number of actions
	countEntries := make([]jen.Code, numStates)
	for i := 0; i < numStates; i++ {
		if actions, ok := g.acceptActions[i]; ok {
			countEntries[i] = jen.Lit(len(actions))
		} else {
			countEntries[i] = jen.Lit(0)
		}
	}

	// Generate accept action tags array: acceptActionTags[state][actionIdx] = tag
	// Generate accept action offsets array: acceptActionOffsets[state][actionIdx] = offset
	tagRows := make([]jen.Code, numStates)
	offsetRows := make([]jen.Code, numStates)

	for stateIdx := 0; stateIdx < numStates; stateIdx++ {
		tagEntries := make([]jen.Code, maxActions)
		offsetEntries := make([]jen.Code, maxActions)

		for j := 0; j < maxActions; j++ {
			tagEntries[j] = jen.Lit(0)
			offsetEntries[j] = jen.Lit(0)
		}

		if actions, ok := g.acceptActions[stateIdx]; ok {
			for j, action := range actions {
				if j < maxActions {
					tagEntries[j] = jen.Lit(action.Tag)
					offsetEntries[j] = jen.Lit(action.Offset)
				}
			}
		}

		tagRows[stateIdx] = jen.Index(jen.Lit(maxActions)).Int().Values(tagEntries...)
		offsetRows[stateIdx] = jen.Index(jen.Lit(maxActions)).Int().Values(offsetEntries...)
	}

	return []jen.Code{
		jen.Comment("Accept action counts [state] -> number of actions").Line().
			Var().Id(g.getVarName("acceptActionCount")).Op("=").Index(jen.Lit(numStates)).Int().Values(countEntries...),
		jen.Comment("Accept action tags [state][actionIdx] -> tag index").Line().
			Var().Id(g.getVarName("acceptActionTags")).Op("=").Index(jen.Lit(numStates)).Index(jen.Lit(maxActions)).Int().Values(tagRows...),
		jen.Comment("Accept action offsets [state][actionIdx] -> offset").Line().
			Var().Id(g.getVarName("acceptActionOffsets")).Op("=").Index(jen.Lit(numStates)).Index(jen.Lit(maxActions)).Int().Values(offsetRows...),
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

// getMaxActionsPerTransition computes the maximum number of tag actions for any single transition.
func (g *TDFAGenerator) getMaxActionsPerTransition() int {
	maxActions := 0
	for _, stateActions := range g.tagActions {
		for _, actions := range stateActions {
			if len(actions) > maxActions {
				maxActions = len(actions)
			}
		}
	}
	return maxActions
}

// getMaxAcceptActions computes the maximum number of accept actions for any state.
func (g *TDFAGenerator) getMaxAcceptActions() int {
	maxActions := 0
	for _, actions := range g.acceptActions {
		if len(actions) > maxActions {
			maxActions = len(actions)
		}
	}
	return maxActions
}

// generateMatchingLoop generates the main TDFA matching loop.
// Optimized: uses array-based lookups instead of maps for zero allocations.
func (g *TDFAGenerator) generateMatchingLoop() []jen.Code {
	var inputAccess jen.Code
	if g.isBytes {
		inputAccess = jen.Id(codegen.InputName).Index(jen.Id("i"))
	} else {
		inputAccess = jen.Id(codegen.InputName).Index(jen.Id("i"))
	}

	tagCount := g.getTagCount()
	maxTagActions := g.getMaxActionsPerTransition()
	maxAcceptActions := g.getMaxAcceptActions()

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

	// Generate tag action application code (array-based, no allocations)
	var applyTagActions jen.Code
	if maxTagActions > 0 {
		applyTagActions = jen.Comment("Apply tag actions (array-based, zero allocations)").Line().
			For(jen.Id("a").Op(":=").Lit(0), jen.Id("a").Op("<").Id(g.getVarName("tagActionCount")).Index(jen.Id("state")).Index(jen.Id("c")), jen.Id("a").Op("++")).Block(
			jen.Id("tags").Index(jen.Id(g.getVarName("tagActionTags")).Index(jen.Id("state")).Index(jen.Id("c")).Index(jen.Id("a"))).Op("=").
				Id("i").Op("+").Lit(1).Op("-").Id(g.getVarName("tagActionOffsets")).Index(jen.Id("state")).Index(jen.Id("c")).Index(jen.Id("a")),
		)
	} else {
		applyTagActions = jen.Comment("No tag actions")
	}

	// Generate accept action application code (array-based, no allocations)
	var applyAcceptActions jen.Code
	if maxAcceptActions > 0 {
		applyAcceptActions = jen.For(jen.Id("a").Op(":=").Lit(0), jen.Id("a").Op("<").Id(g.getVarName("acceptActionCount")).Index(jen.Id("state")), jen.Id("a").Op("++")).Block(
			jen.Id("tags").Index(jen.Id(g.getVarName("acceptActionTags")).Index(jen.Id("state")).Index(jen.Id("a"))).Op("=").
				Id("i").Op("+").Lit(1).Op("-").Id(g.getVarName("acceptActionOffsets")).Index(jen.Id("state")).Index(jen.Id("a")),
		)
	} else {
		applyAcceptActions = jen.Null()
	}

	// Generate code to copy tags to matchTags (element by element, no allocation)
	copyTagsCode := make([]jen.Code, tagCount)
	for i := 0; i < tagCount; i++ {
		copyTagsCode[i] = jen.Id("matchTags").Index(jen.Lit(i)).Op("=").Id("tags").Index(jen.Lit(i))
	}

	// Build the outer loop body
	outerLoopBody := []jen.Code{
		jen.Id("state").Op(":=").Lit(0),
		jen.Id("matchEnd").Op(":=").Lit(-1),
		jen.Comment("Initialize matchTags to -1 (pre-allocated in outer scope)"),
		jen.For(jen.Id("j").Op(":=").Range().Id("matchTags")).Block(
			jen.Id("matchTags").Index(jen.Id("j")).Op("=").Lit(-1),
		),
		jen.Line(),
		jen.Comment("Iterate over all possible start positions"),
		jen.For(jen.Id("start").Op(":=").Lit(0), jen.Id("start").Op("<=").Id(codegen.InputLenName), jen.Id("start").Op("++")).Block(
			jen.Comment("Reset tags for this attempt"),
			jen.For(jen.Id("j").Op(":=").Range().Id("tags")).Block(
				jen.Id("tags").Index(jen.Id("j")).Op("=").Lit(-1),
			),
			jen.Id("tags").Index(jen.Lit(0)).Op("=").Id("start"), // Group 0 start
			jen.Line(),
			jen.Comment("Choose start state based on position"),
			jen.If(jen.Id("start").Op("==").Lit(0)).Block(setupBegin...).Else().Block(setupAny...),
			jen.Line(),
			jen.Comment("Check if start state is accepting (empty match)"),
			jen.If(jen.Id(g.getVarName("acceptStates")).Index(jen.Id("state"))).Block(
				append([]jen.Code{jen.Id("matchEnd").Op("=").Id("start")}, copyTagsCode...)...,
			),
			jen.Comment("Check if start state is accepting at EOT"),
			jen.If(jen.Id("start").Op("==").Id(codegen.InputLenName).Op("&&").Id(g.getVarName("acceptStatesEOT")).Index(jen.Id("state"))).Block(
				append([]jen.Code{jen.Id("matchEnd").Op("=").Id("start")}, copyTagsCode...)...,
			),
			jen.Line(),
			jen.For(jen.Id("i").Op(":=").Id("start"), jen.Id("i").Op("<").Id(codegen.InputLenName), jen.Id("i").Op("++")).Block(
				jen.Id("c").Op(":=").Add(inputAccess),
				jen.If(jen.Id("c").Op(">=").Lit(128)).Block(
					jen.Break(), // Non-ASCII, exit
				),
				jen.Line(),
				jen.Comment("Look up transition (array-based for speed)"),
				jen.Id("nextState").Op(":=").Id(g.getVarName("transitions")).Index(jen.Id("state")).Index(jen.Id("c")),
				jen.If(jen.Id("nextState").Op("<").Lit(0)).Block(
					jen.Break(),
				),
				jen.Line(),
				applyTagActions,
				jen.Line(),
				jen.Id("state").Op("=").Id("nextState"),
				jen.If(jen.Id(g.getVarName("acceptStates")).Index(jen.Id("state"))).Block(
					append([]jen.Code{
						applyAcceptActions,
						jen.Id("matchEnd").Op("=").Id("i").Op("+").Lit(1),
					}, copyTagsCode...)...,
				),
				jen.If(jen.Id("i").Op("==").Id(codegen.InputLenName).Op("-").Lit(1).Op("&&").Id(g.getVarName("acceptStatesEOT")).Index(jen.Id("state"))).Block(
					append([]jen.Code{
						applyAcceptActions,
						jen.Id("matchEnd").Op("=").Id("i").Op("+").Lit(1),
					}, copyTagsCode...)...,
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
		jen.Return(jen.False()),
	}

	// No need to differentiate bytes vs string for the false return since it's just bool now

	return outerLoopBody
}

// generateResultConstruction generates code to construct the result struct from tags.
// It populates the pointer 'r' directly to avoid allocations in the reuse path.
func (g *TDFAGenerator) generateResultConstruction(tagsVar string) []jen.Code {
	// Helper to get slice expression
	getSlice := func(start, end *jen.Statement) jen.Code {
		return jen.Id(codegen.InputName).Index(start.Clone().Op(":").Add(end))
	}

	code := []jen.Code{}

	// Set Match field (Group 0)
	code = append(code,
		jen.If(jen.Id(tagsVar).Index(jen.Lit(0)).Op(">=").Lit(0)).Block(
			jen.Id("r").Dot("Match").Op("=").Add(getSlice(
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
		//     r.Field = input[tags[2*i]:tags[2*i+1]]
		// }
		code = append(code,
			jen.If(jen.Id(tagsVar).Index(jen.Lit(2*i)).Op(">=").Lit(0)).Block(
				jen.If(jen.Id(tagsVar).Index(jen.Lit(2*i+1)).Op("<").Lit(0)).Block(
					jen.Id(tagsVar).Index(jen.Lit(2*i+1)).Op("=").Id(tagsVar).Index(jen.Lit(1)),
				),
				jen.Id("r").Dot(fieldName).Op("=").Add(getSlice(
					jen.Id(tagsVar).Index(jen.Lit(2*i)),
					jen.Id(tagsVar).Index(jen.Lit(2*i+1)),
				)),
			),
		)
	}

	code = append(code, jen.Return(jen.True()))
	return code
}
