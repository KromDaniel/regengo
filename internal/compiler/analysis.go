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
