package compiler

import "regexp/syntax"

// extractCaptureNames extracts capture group names from the regex AST.
func extractCaptureNames(re *syntax.Regexp) []string {
	var names []string
	names = append(names, "") // Group 0 is always the full match (unnamed)

	var walk func(*syntax.Regexp)
	walk = func(r *syntax.Regexp) {
		if r.Op == syntax.OpCapture {
			names = append(names, r.Name)
		}
		for _, sub := range r.Sub {
			walk(sub)
		}
	}

	walk(re)
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
