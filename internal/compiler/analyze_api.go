package compiler

import (
	"regexp/syntax"
	"sort"
	"strings"
)

// AnalysisResult contains the results of pattern analysis without code generation.
type AnalysisResult struct {
	// FeatureLabels are derived from pattern structure (sorted alphabetically)
	FeatureLabels []string `json:"feature_labels"`

	// EngineLabels are derived from compilation analysis (sorted alphabetically)
	EngineLabels []string `json:"engine_labels"`

	// Detailed analysis info
	HasCaptures         bool `json:"has_captures"`
	HasCatastrophicRisk bool `json:"has_catastrophic_risk"`
	HasEndAnchor        bool `json:"has_end_anchor"`
	NFAStates           int  `json:"nfa_states"`
}

// AnalyzePattern performs pattern analysis and returns labels without generating code.
// It returns an error if the pattern is invalid.
func AnalyzePattern(pattern string, tdfaThreshold int) (*AnalysisResult, error) {
	// Parse the regex pattern
	regexAST, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return nil, err
	}

	// Simplify the regex syntax tree (same as in Compile)
	regexAST = regexAST.Simplify()

	// Compile to instruction program
	prog, err := syntax.Compile(regexAST)
	if err != nil {
		return nil, err
	}

	// Perform complexity analysis (reuse existing function)
	complexity := analyzeComplexity(prog, regexAST)

	// Determine if pattern has captures
	hasCaptures := prog.NumCap > 2

	// Derive feature labels from pattern
	featureLabels := deriveFeatureLabels(pattern, prog, regexAST)

	// Derive engine labels using the same logic as the compiler
	engineLabels := deriveEngineLabels(prog, regexAST, complexity, hasCaptures, tdfaThreshold)

	result := &AnalysisResult{
		FeatureLabels:       featureLabels,
		EngineLabels:        engineLabels,
		HasCaptures:         hasCaptures,
		HasCatastrophicRisk: complexity.HasCatastrophicRisk,
		HasEndAnchor:        complexity.HasEndAnchor,
		NFAStates:           complexity.EstimatedNFAStates,
	}

	return result, nil
}

// deriveFeatureLabels extracts feature labels from the pattern structure.
// Labels are sorted alphabetically.
func deriveFeatureLabels(pattern string, prog *syntax.Prog, ast *syntax.Regexp) []string {
	var labels []string

	// Anchored: pattern uses ^ or $
	if isAnchored(prog) || hasEndAnchor(prog) {
		labels = append(labels, "Anchored")
	}

	// Alternation: pattern contains | (check AST for OpAlternate)
	if hasAlternation(ast) {
		labels = append(labels, "Alternation")
	}

	// Captures: pattern has named capture groups
	if prog.NumCap > 2 {
		labels = append(labels, "Captures")
	}

	// CharClass: pattern contains [...] or \d, \w, \s, etc.
	if hasCharClass(pattern, ast) {
		labels = append(labels, "CharClass")
	}

	// Multibyte: pattern contains non-ASCII characters
	if hasMultibyte(pattern) {
		labels = append(labels, "Multibyte")
	}

	// NonCapturing: pattern contains (?:...)
	if strings.Contains(pattern, "(?:") {
		labels = append(labels, "NonCapturing")
	}

	// Quantifiers: pattern uses +, *, ?, {n,m}
	if hasQuantifiers(ast) {
		labels = append(labels, "Quantifiers")
	}

	// WordBoundary: pattern uses \b or \B
	if hasWordBoundary(prog) {
		labels = append(labels, "WordBoundary")
	}

	// UnicodeCharClass: pattern contains character classes with Unicode (non-ASCII) runes
	if hasUnicodeCharClass(prog) {
		labels = append(labels, "UnicodeCharClass")
	}

	// Simple: no special features
	if len(labels) == 0 {
		labels = append(labels, "Simple")
	}

	sort.Strings(labels)
	return labels
}

// deriveEngineLabels determines which engine will be used for this pattern.
// Uses the same logic as the compiler to ensure consistency.
// Labels are sorted alphabetically.
func deriveEngineLabels(prog *syntax.Prog, ast *syntax.Regexp, complexity ComplexityAnalysis, hasCaptures bool, tdfaThreshold int) []string {
	var labels []string

	// Determine Thompson NFA for match (same logic as compiler.go:112)
	useThompsonForMatch := complexity.UseThompsonNFA

	// Determine memoization (same logic as compiler.go:115-117)
	useMemoization := false
	if complexity.HasCatastrophicRisk && !useThompsonForMatch {
		useMemoization = true
	}

	// Determine capture engine (same logic as compiler.go:122-138)
	useTDFAForCaptures := false
	useTNFAForCaptures := false

	if hasCaptures && complexity.HasCatastrophicRisk {
		// Try TDFA first - need to check if it's feasible
		if canUseTDFAStandalone(prog, tdfaThreshold) {
			useTDFAForCaptures = true
		} else {
			useTNFAForCaptures = true
		}
	}

	// Add engine labels based on decisions
	if useThompsonForMatch {
		labels = append(labels, "Thompson")
	}

	if hasCaptures {
		if useTDFAForCaptures {
			labels = append(labels, "TDFA")
		} else if useTNFAForCaptures {
			labels = append(labels, "TNFA")
		}
	}

	if useMemoization && !useThompsonForMatch {
		labels = append(labels, "Memoization")
	}

	// Default: Backtracking (when no special engine is needed)
	if !useThompsonForMatch && !useTDFAForCaptures && !useTNFAForCaptures && !useMemoization {
		labels = append(labels, "Backtracking")
	}

	sort.Strings(labels)
	return labels
}

// canUseTDFAStandalone checks if TDFA can be used without creating a full Compiler.
// This is a simplified version of TDFAGenerator.CanUseTDFA().
func canUseTDFAStandalone(prog *syntax.Prog, tdfaThreshold int) bool {
	if prog == nil {
		return false
	}

	if tdfaThreshold <= 0 {
		tdfaThreshold = 500
	}

	// Check for unsupported instructions (word boundaries)
	for _, inst := range prog.Inst {
		if inst.Op == syntax.InstEmptyWidth {
			emptyOp := syntax.EmptyOp(inst.Arg)
			if emptyOp&(syntax.EmptyWordBoundary|syntax.EmptyNoWordBoundary) != 0 {
				return false
			}
		}
	}

	// Build TDFA to check state count (simplified version)
	// For now, use a heuristic based on NFA size
	// Full TDFA construction would require duplicating significant code
	// A reasonable heuristic: if NFA has < threshold/10 states, TDFA is likely feasible
	if len(prog.Inst) > tdfaThreshold/10 {
		return false
	}

	return true
}

// hasAlternation checks if the AST contains alternation (|).
func hasAlternation(re *syntax.Regexp) bool {
	if re == nil {
		return false
	}
	if re.Op == syntax.OpAlternate {
		return true
	}
	for _, sub := range re.Sub {
		if hasAlternation(sub) {
			return true
		}
	}
	return false
}

// hasCharClass checks if the pattern uses character classes.
func hasCharClass(pattern string, ast *syntax.Regexp) bool {
	// Check pattern string for common character class escapes
	if strings.ContainsAny(pattern, "[]") {
		return true
	}
	if strings.Contains(pattern, "\\d") || strings.Contains(pattern, "\\D") ||
		strings.Contains(pattern, "\\w") || strings.Contains(pattern, "\\W") ||
		strings.Contains(pattern, "\\s") || strings.Contains(pattern, "\\S") {
		return true
	}
	// Also check AST for OpCharClass
	return hasCharClassInAST(ast)
}

// hasCharClassInAST checks if the AST contains character class operations.
func hasCharClassInAST(re *syntax.Regexp) bool {
	if re == nil {
		return false
	}
	if re.Op == syntax.OpCharClass || re.Op == syntax.OpAnyCharNotNL || re.Op == syntax.OpAnyChar {
		return true
	}
	for _, sub := range re.Sub {
		if hasCharClassInAST(sub) {
			return true
		}
	}
	return false
}

// hasMultibyte checks if the pattern contains non-ASCII characters.
func hasMultibyte(pattern string) bool {
	for _, r := range pattern {
		if r > 127 {
			return true
		}
	}
	return false
}

// hasQuantifiers checks if the AST contains quantifier operations.
func hasQuantifiers(re *syntax.Regexp) bool {
	if re == nil {
		return false
	}
	switch re.Op {
	case syntax.OpStar, syntax.OpPlus, syntax.OpQuest, syntax.OpRepeat:
		return true
	}
	for _, sub := range re.Sub {
		if hasQuantifiers(sub) {
			return true
		}
	}
	return false
}
