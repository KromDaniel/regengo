package compiler

import (
	"regexp/syntax"
	"sort"
	"strconv"
	"strings"
)

// Shared automata utilities for DFA construction.
// These functions operate on simple NFA state IDs ([]int) without tag tracking,
// enabling reuse between TDFA and pure DFA implementations.

// GetPossibleChars returns all possible ASCII input characters from the given NFA states.
// This is the core character enumeration used in DFA construction.
// Returns a sorted slice of bytes for deterministic iteration.
//
// IMPORTANT: This function only returns ASCII characters (0-127). Unicode characters
// outside the ASCII range are ignored. This is intentional as the DFA engine uses
// byte-indexed transition tables optimized for ASCII. Non-ASCII input causes the
// DFA to fall back to other engines.
func GetPossibleChars(prog *syntax.Prog, stateIDs []int) []byte {
	charSet := make(map[byte]bool)

	for _, stateID := range stateIDs {
		if stateID >= len(prog.Inst) {
			continue
		}
		inst := prog.Inst[stateID]
		switch inst.Op {
		case syntax.InstRune1:
			if len(inst.Rune) > 0 && inst.Rune[0] < MaxASCIIRune {
				charSet[byte(inst.Rune[0])] = true
			}
		case syntax.InstRune:
			// Character class - add all characters in ranges
			for i := 0; i < len(inst.Rune); i += 2 {
				lo, hi := inst.Rune[i], inst.Rune[i+1]
				if lo < MaxASCIIRune {
					end := hi
					if end >= MaxASCIIRune {
						end = MaxASCIIRune - 1
					}
					for c := lo; c <= end; c++ {
						charSet[byte(c)] = true
					}
				}
			}
		case syntax.InstRuneAny:
			// Match any - add all ASCII characters
			for c := byte(0); c < MaxASCIIRune; c++ {
				charSet[c] = true
			}
		case syntax.InstRuneAnyNotNL:
			// Match any except newline
			for c := byte(0); c < MaxASCIIRune; c++ {
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

// IsAcceptingNFASet checks if any state in the set is an accepting state (InstMatch).
func IsAcceptingNFASet(prog *syntax.Prog, stateIDs []int) bool {
	for _, stateID := range stateIDs {
		if stateID >= len(prog.Inst) {
			continue
		}
		if prog.Inst[stateID].Op == syntax.InstMatch {
			return true
		}
	}
	return false
}

// NFASetKey creates a unique string key for an NFA state set.
// Used to detect duplicate DFA states during construction.
// The key is canonical (sorted) to ensure consistent hashing.
// Uses strings.Builder for efficient O(n) string construction.
func NFASetKey(stateIDs []int) string {
	if len(stateIDs) == 0 {
		return ""
	}

	// Sort a copy to ensure canonical key
	sorted := make([]int, len(stateIDs))
	copy(sorted, stateIDs)
	sort.Ints(sorted)

	var sb strings.Builder
	for i, id := range sorted {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(strconv.Itoa(id))
	}
	return sb.String()
}

// EpsilonClosure computes the epsilon closure of NFA states without tag tracking.
// Returns all states reachable via epsilon transitions (Nop, Alt, Capture, EmptyWidth).
// matchFlags specifies which empty-width conditions are satisfied (e.g., EmptyBeginText).
// The result preserves priority order for leftmost-greedy disambiguation.
func EpsilonClosure(prog *syntax.Prog, stateIDs []int, matchFlags syntax.EmptyOp) []int {
	visited := make(map[int]bool)
	result := make([]int, 0, len(stateIDs)*2)

	// Initialize stack with states in reverse order for LIFO processing
	// This preserves priority (first state in input has highest priority)
	stack := make([]int, 0, len(stateIDs))
	for i := len(stateIDs) - 1; i >= 0; i-- {
		stack = append(stack, stateIDs[i])
	}

	for len(stack) > 0 {
		stateID := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if visited[stateID] || stateID >= len(prog.Inst) {
			continue
		}
		visited[stateID] = true
		result = append(result, stateID)

		inst := prog.Inst[stateID]
		switch inst.Op {
		case syntax.InstNop:
			stack = append(stack, int(inst.Out))
		case syntax.InstCapture:
			// For pure DFA, we ignore captures but still follow the transition
			stack = append(stack, int(inst.Out))
		case syntax.InstAlt:
			// Priority: try Out first (greedy), then Arg
			// Push Arg first so Out is popped first (LIFO)
			stack = append(stack, int(inst.Arg), int(inst.Out))
		case syntax.InstEmptyWidth:
			// Check if the empty width condition is satisfied
			if syntax.EmptyOp(inst.Arg)&matchFlags == syntax.EmptyOp(inst.Arg) {
				stack = append(stack, int(inst.Out))
			}
		}
	}

	return result
}

// MatchesChar checks if an NFA instruction matches a given character.
// Returns true if the instruction consumes the character.
func MatchesChar(inst *syntax.Inst, c byte) bool {
	switch inst.Op {
	case syntax.InstRune1:
		if len(inst.Rune) > 0 {
			r := inst.Rune[0]
			return r < MaxASCIIRune && byte(r) == c
		}
	case syntax.InstRune:
		for i := 0; i < len(inst.Rune); i += 2 {
			lo, hi := inst.Rune[i], inst.Rune[i+1]
			if rune(c) >= lo && rune(c) <= hi {
				return true
			}
		}
	case syntax.InstRuneAny:
		return true
	case syntax.InstRuneAnyNotNL:
		return c != '\n'
	}
	return false
}

// ComputeNextStates computes the next NFA states after consuming character c.
// Returns the state IDs reachable by consuming c and then following epsilon transitions.
func ComputeNextStates(prog *syntax.Prog, stateIDs []int, c byte, matchFlags syntax.EmptyOp) []int {
	nextStates := make([]int, 0)

	for _, stateID := range stateIDs {
		if stateID >= len(prog.Inst) {
			continue
		}
		inst := prog.Inst[stateID]
		if MatchesChar(&inst, c) {
			nextStates = append(nextStates, int(inst.Out))
		}
	}

	if len(nextStates) == 0 {
		return nil
	}

	// Compute epsilon closure of next states
	return EpsilonClosure(prog, nextStates, matchFlags)
}

// UnionNFASets returns the union of two NFA state sets.
// Order is preserved: all of a first, then elements of b not already present.
// Used by the pure DFA to embed unanchored restart threads in every state.
func UnionNFASets(a, b []int) []int {
	seen := make(map[int]bool, len(a)+len(b))
	result := make([]int, 0, len(a)+len(b))
	for _, s := range a {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	for _, s := range b {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
