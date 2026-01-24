package compiler

import (
	"regexp/syntax"
	"testing"
)

func compilePattern(t *testing.T, pattern string) *syntax.Prog {
	re, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		t.Fatalf("Failed to parse pattern %q: %v", pattern, err)
	}
	prog, err := syntax.Compile(re)
	if err != nil {
		t.Fatalf("Failed to compile pattern %q: %v", pattern, err)
	}
	return prog
}

func TestGetPossibleChars(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		expected []byte
	}{
		{
			name:     "single char",
			pattern:  "a",
			expected: []byte{'a'},
		},
		{
			name:     "char class",
			pattern:  "[abc]",
			expected: []byte{'a', 'b', 'c'},
		},
		{
			name:     "char range",
			pattern:  "[a-d]",
			expected: []byte{'a', 'b', 'c', 'd'},
		},
		{
			name:     "digit class",
			pattern:  "[0-9]",
			expected: []byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'},
		},
		{
			name:     "alternation",
			pattern:  "a|b|c",
			expected: []byte{'a', 'b', 'c'},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := compilePattern(t, tt.pattern)

			// Get initial states via epsilon closure
			startStates := EpsilonClosure(prog, []int{int(prog.Start)}, 0)
			chars := GetPossibleChars(prog, startStates)

			if len(chars) != len(tt.expected) {
				t.Errorf("Expected %d chars, got %d: %v", len(tt.expected), len(chars), chars)
				return
			}

			for i, c := range chars {
				if c != tt.expected[i] {
					t.Errorf("Char %d: expected %q, got %q", i, tt.expected[i], c)
				}
			}
		})
	}
}

func TestIsAcceptingNFASet(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		input    string
		expected bool
	}{
		{
			name:     "empty pattern accepts immediately",
			pattern:  "",
			input:    "",
			expected: true,
		},
		{
			name:     "non-empty pattern does not accept at start",
			pattern:  "a",
			input:    "",
			expected: false,
		},
		{
			name:     "optional pattern accepts at start",
			pattern:  "a?",
			input:    "",
			expected: true,
		},
		{
			name:     "kleene star accepts at start",
			pattern:  "a*",
			input:    "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := compilePattern(t, tt.pattern)

			// Get initial states via epsilon closure
			startStates := EpsilonClosure(prog, []int{int(prog.Start)}, syntax.EmptyBeginText)
			result := IsAcceptingNFASet(prog, startStates)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestNFASetKey(t *testing.T) {
	tests := []struct {
		name     string
		stateIDs []int
		expected string
	}{
		{
			name:     "empty set",
			stateIDs: []int{},
			expected: "",
		},
		{
			name:     "single state",
			stateIDs: []int{5},
			expected: "5",
		},
		{
			name:     "multiple states sorted",
			stateIDs: []int{1, 3, 5},
			expected: "1,3,5",
		},
		{
			name:     "multiple states unsorted",
			stateIDs: []int{5, 1, 3},
			expected: "1,3,5", // Should be sorted in output
		},
		{
			name:     "duplicate handling",
			stateIDs: []int{1, 1, 3},
			expected: "1,1,3", // Duplicates preserved (caller's responsibility)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NFASetKey(tt.stateIDs)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestEpsilonClosure(t *testing.T) {
	tests := []struct {
		name          string
		pattern       string
		matchFlags    syntax.EmptyOp
		minStates     int // Minimum expected states in closure
		shouldContain []syntax.InstOp
	}{
		{
			name:          "simple literal",
			pattern:       "abc",
			matchFlags:    0,
			minStates:     1,
			shouldContain: []syntax.InstOp{syntax.InstRune1},
		},
		{
			name:          "alternation",
			pattern:       "cat|dog",
			matchFlags:    0,
			minStates:     2, // Both alternatives should be reachable
			shouldContain: []syntax.InstOp{syntax.InstRune1},
		},
		{
			name:          "optional",
			pattern:       "a?",
			matchFlags:    0,
			minStates:     2, // Both 'a' state and match state
			shouldContain: []syntax.InstOp{syntax.InstRune1, syntax.InstMatch},
		},
		{
			name:          "anchored at start",
			pattern:       "^a",
			matchFlags:    syntax.EmptyBeginText,
			minStates:     1,
			shouldContain: []syntax.InstOp{syntax.InstRune1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := compilePattern(t, tt.pattern)
			closure := EpsilonClosure(prog, []int{int(prog.Start)}, tt.matchFlags)

			if len(closure) < tt.minStates {
				t.Errorf("Expected at least %d states, got %d", tt.minStates, len(closure))
			}

			// Check that expected instruction types are present
			for _, expectedOp := range tt.shouldContain {
				found := false
				for _, stateID := range closure {
					if prog.Inst[stateID].Op == expectedOp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find instruction %v in closure", expectedOp)
				}
			}
		})
	}
}

func TestMatchesChar(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		char     byte
		expected bool
	}{
		{
			name:     "exact match",
			pattern:  "a",
			char:     'a',
			expected: true,
		},
		{
			name:     "exact non-match",
			pattern:  "a",
			char:     'b',
			expected: false,
		},
		{
			name:     "char class match",
			pattern:  "[abc]",
			char:     'b',
			expected: true,
		},
		{
			name:     "char class non-match",
			pattern:  "[abc]",
			char:     'd',
			expected: false,
		},
		{
			name:     "dot matches any",
			pattern:  ".",
			char:     'x',
			expected: true,
		},
		{
			name:     "dot does not match newline",
			pattern:  ".",
			char:     '\n',
			expected: false, // Default dot doesn't match newline
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := compilePattern(t, tt.pattern)
			closure := EpsilonClosure(prog, []int{int(prog.Start)}, 0)

			// Find the rune instruction
			var inst *syntax.Inst
			for _, stateID := range closure {
				op := prog.Inst[stateID].Op
				if op == syntax.InstRune1 || op == syntax.InstRune || op == syntax.InstRuneAny || op == syntax.InstRuneAnyNotNL {
					inst = &prog.Inst[stateID]
					break
				}
			}

			if inst == nil {
				t.Fatal("Could not find rune instruction")
			}

			result := MatchesChar(inst, tt.char)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestComputeNextStates(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		char        byte
		shouldMatch bool
	}{
		{
			name:        "simple advance",
			pattern:     "ab",
			char:        'a',
			shouldMatch: true,
		},
		{
			name:        "no match",
			pattern:     "ab",
			char:        'x',
			shouldMatch: false,
		},
		{
			name:        "alternation first",
			pattern:     "a|b",
			char:        'a',
			shouldMatch: true,
		},
		{
			name:        "alternation second",
			pattern:     "a|b",
			char:        'b',
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := compilePattern(t, tt.pattern)
			startStates := EpsilonClosure(prog, []int{int(prog.Start)}, 0)
			nextStates := ComputeNextStates(prog, startStates, tt.char, 0)

			hasNext := len(nextStates) > 0
			if hasNext != tt.shouldMatch {
				t.Errorf("Expected shouldMatch=%v, got %v (nextStates=%v)", tt.shouldMatch, hasNext, nextStates)
			}
		})
	}
}
