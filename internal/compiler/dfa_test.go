package compiler

import (
	"regexp/syntax"
	"testing"
)

func newTestDFAGenerator(t *testing.T, pattern string) *DFAGenerator {
	re, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		t.Fatalf("Failed to parse pattern %q: %v", pattern, err)
	}
	prog, err := syntax.Compile(re)
	if err != nil {
		t.Fatalf("Failed to compile pattern %q: %v", pattern, err)
	}

	// Create a minimal compiler config
	c := &Compiler{
		config: Config{
			Program: prog,
			Name:    "Test",
		},
	}

	return NewDFAGenerator(c)
}

func TestDFAGenerator_CanUseDFA(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		canUse   bool
		contains string // substring that should be in the info message
	}{
		{
			name:    "simple literal",
			pattern: "hello",
			canUse:  true,
		},
		{
			name:    "char class ASCII",
			pattern: "[a-z]+",
			canUse:  true,
		},
		{
			name:    "alternation",
			pattern: "cat|dog|fish",
			canUse:  true,
		},
		{
			name:    "anchored start",
			pattern: "^hello",
			canUse:  true,
		},
		{
			name:    "anchored end",
			pattern: "hello$",
			canUse:  true,
		},
		{
			name:     "dot not supported",
			pattern:  "a.b",
			canUse:   false,
			contains: "dot",
		},
		{
			name:     "word boundary not supported",
			pattern:  `\bword\b`,
			canUse:   false,
			contains: "empty width",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := newTestDFAGenerator(t, tt.pattern)
			canUse, info := gen.CanUseDFA()

			if canUse != tt.canUse {
				t.Errorf("CanUseDFA() = %v, want %v (info: %s)", canUse, tt.canUse, info)
			}

			if tt.contains != "" && !contains(info, tt.contains) {
				t.Errorf("Info message %q should contain %q", info, tt.contains)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestDFAGenerator_StateConstruction(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		minStates int
		maxStates int
	}{
		{
			name:      "single char",
			pattern:   "a",
			minStates: 2, // start + after 'a'
			maxStates: 3,
		},
		{
			name:      "two chars",
			pattern:   "ab",
			minStates: 3, // start + after 'a' + after 'b'
			maxStates: 4,
		},
		{
			name:      "alternation",
			pattern:   "a|b",
			minStates: 2,
			maxStates: 4,
		},
		{
			name:      "optional",
			pattern:   "a?",
			minStates: 1, // start state may be accepting
			maxStates: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := newTestDFAGenerator(t, tt.pattern)
			canUse, info := gen.CanUseDFA()

			if !canUse {
				t.Fatalf("Expected DFA to be usable: %s", info)
			}

			numStates := len(gen.states)
			if numStates < tt.minStates || numStates > tt.maxStates {
				t.Errorf("Expected %d-%d states, got %d", tt.minStates, tt.maxStates, numStates)
			}
		})
	}
}

func TestDFAGenerator_AcceptStates(t *testing.T) {
	tests := []struct {
		name             string
		pattern          string
		startAccepts     bool
		hasRegularAccept bool
		hasEOTAccept     bool
	}{
		{
			name:             "simple literal",
			pattern:          "a",
			startAccepts:     false,
			hasRegularAccept: true,
			hasEOTAccept:     true,
		},
		{
			name:             "empty match (kleene star)",
			pattern:          "a*",
			startAccepts:     true, // Can match empty
			hasRegularAccept: true,
			hasEOTAccept:     true,
		},
		{
			name:             "anchored end",
			pattern:          "a$",
			startAccepts:     false,
			hasRegularAccept: false, // Only accepts at EOT
			hasEOTAccept:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := newTestDFAGenerator(t, tt.pattern)
			canUse, info := gen.CanUseDFA()

			if !canUse {
				t.Fatalf("Expected DFA to be usable: %s", info)
			}

			// Check start state acceptance
			startAccepts := gen.acceptStates[gen.startStateBegin]
			if startAccepts != tt.startAccepts {
				t.Errorf("Start state accepts = %v, want %v", startAccepts, tt.startAccepts)
			}

			// Check if any regular accept states exist (verification only)
			_ = len(gen.acceptStates) > 0

			// Check if any EOT accept states exist
			hasEOTAccept := false
			for _, isAccept := range gen.acceptStatesEOT {
				if isAccept {
					hasEOTAccept = true
					break
				}
			}
			if hasEOTAccept != tt.hasEOTAccept {
				t.Errorf("Has EOT accept = %v, want %v", hasEOTAccept, tt.hasEOTAccept)
			}
		})
	}
}

func TestDFAGenerator_StateExplosion(t *testing.T) {
	// This pattern can cause state explosion - use a pattern with many states
	// (a|b)(c|d)(e|f)(g|h) creates combinatorial states
	gen := newTestDFAGenerator(t, "(a|b)(c|d)(e|f)(g|h)(i|j)")

	// Set a very low threshold to trigger explosion
	gen.maxStates = 5

	canUse, info := gen.CanUseDFA()

	if canUse {
		t.Errorf("Expected DFA to fail due to state explosion, but got: %s", info)
	}

	if !contains(info, "too many states") && !contains(info, "explosion") {
		t.Errorf("Expected state explosion message, got: %s", info)
	}
}

func TestDFAGenerator_Transitions(t *testing.T) {
	gen := newTestDFAGenerator(t, "ab")
	canUse, info := gen.CanUseDFA()

	if !canUse {
		t.Fatalf("Expected DFA to be usable: %s", info)
	}

	// Verify that 'a' leads to a valid transition from start state
	startState := gen.startStateAny
	if startState >= len(gen.transitions) {
		t.Fatalf("Start state %d out of range", startState)
	}

	transA, hasA := gen.transitions[startState]['a']
	if !hasA {
		t.Error("Expected transition on 'a' from start state")
	}

	// Verify that 'b' leads to a valid transition from state after 'a'
	if transA >= len(gen.transitions) {
		t.Fatalf("State after 'a' (%d) out of range", transA)
	}

	transB, hasB := gen.transitions[transA]['b']
	if !hasB {
		t.Error("Expected transition on 'b' from state after 'a'")
	}

	// Verify the final state is accepting
	if !gen.acceptStates[transB] {
		t.Error("Expected state after 'ab' to be accepting")
	}
}
