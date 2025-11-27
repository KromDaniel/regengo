package compiler

import "regexp/syntax"

// ComplexityAnalysis holds the results of pattern complexity analysis.
// This determines which engine strategy to use for code generation.
type ComplexityAnalysis struct {
	// HasNestedLoops is true if the pattern has nested alternation loops
	// (current memoization trigger from detectComplexity)
	HasNestedLoops bool

	// HasCatastrophicRisk is true if the pattern has nested quantifiers
	// like (a+)+b that can cause exponential backtracking
	HasCatastrophicRisk bool

	// UseThompsonNFA is true if Thompson NFA is recommended for this pattern
	UseThompsonNFA bool

	// UseTDFA is true if Tagged DFA is recommended for captures
	// (catastrophic risk + captures + feasible state count)
	UseTDFA bool

	// EstimatedNFAStates is the number of instructions in the compiled program
	EstimatedNFAStates int

	// HasCaptures is true if the pattern has capture groups
	HasCaptures bool

	// HasEndAnchor is true if the pattern has $ anchor
	HasEndAnchor bool
}

// extractCaptureNames extracts capture group names from the regex AST.
// When {n} expands captures (e.g., (?P<x>a){2} becomes two OpCapture nodes),
// we deduplicate by Cap number to match Go's regexp behavior.
func extractCaptureNames(re *syntax.Regexp) []string {
	// Use a map to track captures by Cap number, then build ordered slice
	capMap := make(map[int]string)
	maxCap := 0

	var walk func(*syntax.Regexp)
	walk = func(r *syntax.Regexp) {
		if r.Op == syntax.OpCapture {
			// Only record the first occurrence of each Cap number
			if _, exists := capMap[r.Cap]; !exists {
				capMap[r.Cap] = r.Name
				if r.Cap > maxCap {
					maxCap = r.Cap
				}
			}
		}
		for _, sub := range r.Sub {
			walk(sub)
		}
	}

	walk(re)

	// Build names slice in order: group 0 (unnamed), then 1, 2, etc.
	names := make([]string, maxCap+1)
	names[0] = "" // Group 0 is always the full match (unnamed)
	for cap, name := range capMap {
		names[cap] = name
	}
	return names
}

// hasRepeatingCaptures checks if the regex has any capture groups in repeating context.
// Repeating contexts include *, +, ?, and {n,m} quantifiers.
// Note: Standard regex behavior (including Go's stdlib) captures only the LAST match
// from repeating groups. For example, (\w)+ matching "abc" will capture "c", not ["a","b","c"].
func hasRepeatingCaptures(re *syntax.Regexp) bool {
	return walkCheckRepeating(re, false)
}

// needsBacktracking checks if the compiled program requires backtracking.
// Returns true if the program contains InstAlt instructions (alternations).
func needsBacktracking(prog *syntax.Prog) bool {
	if prog == nil {
		return false
	}

	for i := range prog.Inst {
		if prog.Inst[i].Op == syntax.InstAlt {
			return true
		}
	}

	return false
}

// walkCheckRepeating recursively walks the AST to detect captures in repeating context.
func walkCheckRepeating(re *syntax.Regexp, inRepeat bool) bool {
	// If this is a capture and we're in a repeating context
	if re.Op == syntax.OpCapture && inRepeat {
		return true
	}

	// Check if this node introduces repetition
	isRepeating := false
	switch re.Op {
	case syntax.OpStar, syntax.OpPlus, syntax.OpQuest, syntax.OpRepeat:
		isRepeating = true
	}

	// Recursively check children
	for _, sub := range re.Sub {
		if walkCheckRepeating(sub, inRepeat || isRepeating) {
			return true
		}
	}

	return false
}

// isAnchored checks if the regex is anchored to the start of text.
func isAnchored(prog *syntax.Prog) bool {
	if prog == nil || len(prog.Inst) == 0 {
		return false
	}
	// Check if the first instruction is an empty width assertion for begin text
	startInst := prog.Inst[prog.Start]
	return startInst.Op == syntax.InstEmptyWidth && syntax.EmptyOp(startInst.Arg)&syntax.EmptyBeginText != 0
}

// hasWordBoundary checks if the compiled program uses word boundary assertions (\b or \B).
func hasWordBoundary(prog *syntax.Prog) bool {
	if prog == nil {
		return false
	}

	for i := range prog.Inst {
		if prog.Inst[i].Op == syntax.InstEmptyWidth {
			emptyOp := syntax.EmptyOp(prog.Inst[i].Arg)
			if emptyOp&(syntax.EmptyWordBoundary|syntax.EmptyNoWordBoundary) != 0 {
				return true
			}
		}
	}

	return false
}

// detectComplexity analyzes the program for patterns that could cause catastrophic backtracking.
// It returns true if the program contains nested loops or ambiguous alternations that warrant
// using memoization to guarantee O(N) complexity.
func detectComplexity(prog *syntax.Prog) bool {
	if prog == nil {
		return false
	}

	// Step 1: Identify all Alt instructions
	var alts []int
	for i, inst := range prog.Inst {
		if inst.Op == syntax.InstAlt {
			alts = append(alts, i)
		}
	}

	if len(alts) < 2 {
		return false
	}

	// Step 2: Find "Simple Loops" (cycles containing exactly one Alt)
	simpleLoops := make(map[int]bool)
	for _, altIdx := range alts {
		if isSimpleLoop(prog, altIdx) {
			simpleLoops[altIdx] = true
		}
	}

	// Step 3: Check for nested loops
	// A simple loop is "nested" if it is mutually reachable with another Alt (which implies another loop)
	for loopHead := range simpleLoops {
		for _, otherAlt := range alts {
			if loopHead == otherAlt {
				continue
			}

			// Check if loopHead and otherAlt are mutually reachable
			if reaches(prog, loopHead, otherAlt) && reaches(prog, otherAlt, loopHead) {
				return true
			}
		}
	}

	return false
}

// isSimpleLoop checks if the given Alt instruction forms a cycle that involves no other Alt instructions.
func isSimpleLoop(prog *syntax.Prog, startIdx int) bool {
	// BFS to find cycle back to startIdx
	// Start from successors
	inst := prog.Inst[startIdx]
	queue := []int{int(inst.Out), int(inst.Arg)}
	visited := make(map[int]bool)
	visited[startIdx] = true

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if curr == startIdx {
			return true // Found cycle back to start
		}

		if visited[curr] {
			continue
		}
		visited[curr] = true

		currInst := prog.Inst[curr]
		if currInst.Op == syntax.InstAlt {
			// Found another Alt in the path - not a simple loop
			return false
		}

		// Follow transitions
		if currInst.Op != syntax.InstMatch && currInst.Op != syntax.InstFail {
			queue = append(queue, int(currInst.Out))
		}
	}

	return false
}

// reaches checks if startIdx can reach targetIdx in the instruction graph.
func reaches(prog *syntax.Prog, start, target int) bool {
	queue := []int{start}
	visited := make(map[int]bool)
	visited[start] = true

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if curr == target {
			return true
		}

		inst := prog.Inst[curr]
		var next []int
		if inst.Op == syntax.InstAlt {
			next = []int{int(inst.Out), int(inst.Arg)}
		} else if inst.Op != syntax.InstMatch && inst.Op != syntax.InstFail {
			next = []int{int(inst.Out)}
		}

		for _, n := range next {
			if !visited[n] {
				visited[n] = true
				queue = append(queue, n)
			}
		}
	}
	return false
}

// analyzeComplexity performs comprehensive pattern analysis to determine
// the optimal engine strategy for code generation.
func analyzeComplexity(prog *syntax.Prog, ast *syntax.Regexp) ComplexityAnalysis {
	analysis := ComplexityAnalysis{}

	if prog == nil {
		return analysis
	}

	// Count NFA states
	analysis.EstimatedNFAStates = len(prog.Inst)

	// Check for captures
	analysis.HasCaptures = prog.NumCap > 2

	// Check for end anchor ($) in program - Thompson NFA doesn't handle this yet
	analysis.HasEndAnchor = hasEndAnchor(prog)

	// Detect nested loops at program level (catches issues even after AST simplification)
	analysis.HasNestedLoops = detectComplexity(prog)

	// Detect nested quantifiers at AST level (may miss some after simplification)
	if ast != nil {
		analysis.HasCatastrophicRisk = detectNestedQuantifiers(ast)
	}

	// Decision logic: recommend Thompson NFA if pattern has catastrophic risk
	// OR if it has nested loops that could cause exponential backtracking
	// BUT NOT if it has end anchor (Thompson NFA doesn't handle $ yet)
	if (analysis.HasCatastrophicRisk || analysis.HasNestedLoops) && !analysis.HasEndAnchor {
		analysis.UseThompsonNFA = true
	}

	return analysis
}

// hasEndAnchor checks if the program has an end-of-text anchor ($).
func hasEndAnchor(prog *syntax.Prog) bool {
	if prog == nil {
		return false
	}

	for _, inst := range prog.Inst {
		if inst.Op == syntax.InstEmptyWidth {
			// Check if this is an end-of-text anchor
			if syntax.EmptyOp(inst.Arg)&syntax.EmptyEndText != 0 {
				return true
			}
		}
	}
	return false
}

// detectNestedQuantifiers checks if the regex AST has nested quantifiers
// that could cause catastrophic backtracking (e.g., (a+)+b, (a*)*b).
func detectNestedQuantifiers(re *syntax.Regexp) bool {
	return walkDetectNestedQuantifiers(re, 0)
}

// walkDetectNestedQuantifiers recursively walks the AST to detect nested quantifiers.
// quantifierDepth tracks how many quantifiers we're currently nested inside.
func walkDetectNestedQuantifiers(re *syntax.Regexp, quantifierDepth int) bool {
	if re == nil {
		return false
	}

	isQuantifier := false
	switch re.Op {
	case syntax.OpStar, syntax.OpPlus, syntax.OpQuest, syntax.OpRepeat:
		isQuantifier = true
		if quantifierDepth > 0 {
			// Found a quantifier nested inside another quantifier
			return true
		}
	}

	newDepth := quantifierDepth
	if isQuantifier {
		newDepth++
	}

	// Recursively check children
	for _, sub := range re.Sub {
		if walkDetectNestedQuantifiers(sub, newDepth) {
			return true
		}
	}

	return false
}

// computeAltsNeedingCheckpoint returns a set of Alt instruction indices that
// need capture checkpointing because their Out branch can reach an InstCapture.
// This optimization reduces checkpoint overhead for Alts where captures won't change.
func computeAltsNeedingCheckpoint(prog *syntax.Prog) map[int]bool {
	if prog == nil {
		return nil
	}

	result := make(map[int]bool)

	// For each Alt instruction, check if the Out branch can reach an InstCapture
	for i, inst := range prog.Inst {
		if inst.Op == syntax.InstAlt {
			if canReachCapture(prog, int(inst.Out)) {
				result[i] = true
			}
		}
	}

	return result
}

// PerCaptureCheckpointThreshold is the number of Alts needing checkpoints
// above which we switch to the stdlib-style per-capture checkpointing approach.
// Below this threshold, array copying is simpler and efficient enough.
const PerCaptureCheckpointThreshold = 3

// shouldUsePerCaptureCheckpointing determines if the pattern would benefit from
// the stdlib-style per-capture checkpointing approach.
// Returns true if the number of Alt instructions needing capture checkpoints
// exceeds the threshold, indicating array-copying would be too expensive.
func shouldUsePerCaptureCheckpointing(altsNeedingCheckpoint map[int]bool) bool {
	return len(altsNeedingCheckpoint) > PerCaptureCheckpointThreshold
}

// canReachCapture checks if any path from startIdx can reach an InstCapture.
// Uses BFS to explore all reachable instructions.
func canReachCapture(prog *syntax.Prog, startIdx int) bool {
	visited := make(map[int]bool)
	queue := []int{startIdx}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if curr < 0 || curr >= len(prog.Inst) || visited[curr] {
			continue
		}
		visited[curr] = true

		inst := prog.Inst[curr]

		// Found a capture instruction - this Alt needs checkpointing
		if inst.Op == syntax.InstCapture {
			return true
		}

		// Don't follow past Match or Fail
		if inst.Op == syntax.InstMatch || inst.Op == syntax.InstFail {
			continue
		}

		// Follow successors
		if inst.Op == syntax.InstAlt {
			queue = append(queue, int(inst.Out), int(inst.Arg))
		} else {
			queue = append(queue, int(inst.Out))
		}
	}

	return false
}

// computeEpsilonClosures precomputes epsilon closures for all states in the program.
// Returns a slice where closures[i] is the bitset of states reachable via epsilon
// transitions from state i.
func computeEpsilonClosures(prog *syntax.Prog) []uint64 {
	if prog == nil || len(prog.Inst) == 0 {
		return nil
	}

	closures := make([]uint64, len(prog.Inst))
	for i := range prog.Inst {
		closures[i] = computeEpsilonClosureForState(prog, i)
	}
	return closures
}

// computeEpsilonClosureForState computes the epsilon closure for a single state.
// The epsilon closure is the set of all states reachable by following only
// epsilon transitions (Nop, Capture, Alt branches).
func computeEpsilonClosureForState(prog *syntax.Prog, start int) uint64 {
	if start >= 64 {
		// For now, only support up to 64 states with bitset
		// Larger programs will need different representation
		return 0
	}

	var result uint64
	visited := make(map[int]bool)
	queue := []int{start}

	for len(queue) > 0 {
		state := queue[0]
		queue = queue[1:]

		if visited[state] {
			continue
		}
		visited[state] = true

		if state < 64 {
			result |= (1 << state)
		}

		if state >= len(prog.Inst) {
			continue
		}

		inst := prog.Inst[state]
		// Follow epsilon transitions
		switch inst.Op {
		case syntax.InstNop, syntax.InstCapture:
			queue = append(queue, int(inst.Out))
		case syntax.InstAlt:
			queue = append(queue, int(inst.Out), int(inst.Arg))
		}
	}

	return result
}
