package compiler

import (
	"regexp/syntax"
	"unicode/utf8"
)

// MatchLengthAnalysis holds the computed match length bounds for a pattern.
type MatchLengthAnalysis struct {
	// MinMatchLen is the minimum number of bytes any match can have.
	// Always >= 0.
	MinMatchLen int

	// MaxMatchLen is the maximum number of bytes any match can have.
	// -1 means unbounded (e.g., patterns with * or + quantifiers).
	MaxMatchLen int
}

// AnalyzeMatchLength computes the minimum and maximum match lengths for a pattern.
// This is used by the streaming API to determine how much leftover buffer to keep
// between chunks to ensure matches spanning chunk boundaries are not missed.
func AnalyzeMatchLength(re *syntax.Regexp) MatchLengthAnalysis {
	if re == nil {
		return MatchLengthAnalysis{MinMatchLen: 0, MaxMatchLen: 0}
	}
	return MatchLengthAnalysis{
		MinMatchLen: minMatchLen(re),
		MaxMatchLen: maxMatchLen(re),
	}
}

// minMatchLen computes the minimum number of bytes required for a match.
// This is a lower bound - the actual match will be at least this many bytes.
func minMatchLen(re *syntax.Regexp) int {
	if re == nil {
		return 0
	}

	switch re.Op {
	case syntax.OpNoMatch:
		// Can never match
		return 0

	case syntax.OpEmptyMatch:
		// Empty string matches
		return 0

	case syntax.OpLiteral:
		// Sum of UTF-8 encoded lengths of all runes
		total := 0
		for _, r := range re.Rune {
			total += utf8.RuneLen(r)
		}
		return total

	case syntax.OpCharClass:
		// At least one rune from the class
		if len(re.Rune) == 0 {
			return 0
		}
		// Find minimum rune length in the class
		// Runes are stored as pairs [lo, hi]
		minLen := 4 // Max UTF-8 length
		for i := 0; i < len(re.Rune); i += 2 {
			lo := re.Rune[i]
			// The smallest rune in this range determines min bytes
			runeLen := utf8.RuneLen(lo)
			if runeLen < minLen {
				minLen = runeLen
			}
		}
		return minLen

	case syntax.OpAnyCharNotNL, syntax.OpAnyChar:
		// At least 1 byte (could be multi-byte for Unicode)
		return 1

	case syntax.OpBeginLine, syntax.OpEndLine,
		syntax.OpBeginText, syntax.OpEndText,
		syntax.OpWordBoundary, syntax.OpNoWordBoundary:
		// Zero-width assertions
		return 0

	case syntax.OpCapture:
		// Capture group has same length as its content
		if len(re.Sub) > 0 {
			return minMatchLen(re.Sub[0])
		}
		return 0

	case syntax.OpStar:
		// Zero or more - minimum is 0
		return 0

	case syntax.OpPlus:
		// One or more - minimum is one occurrence
		if len(re.Sub) > 0 {
			return minMatchLen(re.Sub[0])
		}
		return 0

	case syntax.OpQuest:
		// Zero or one - minimum is 0
		return 0

	case syntax.OpRepeat:
		// {n,m} - minimum is n occurrences
		if len(re.Sub) > 0 {
			return re.Min * minMatchLen(re.Sub[0])
		}
		return 0

	case syntax.OpConcat:
		// Sum of all parts
		total := 0
		for _, sub := range re.Sub {
			total += minMatchLen(sub)
		}
		return total

	case syntax.OpAlternate:
		// Minimum of all alternatives
		if len(re.Sub) == 0 {
			return 0
		}
		min := minMatchLen(re.Sub[0])
		for _, sub := range re.Sub[1:] {
			subMin := minMatchLen(sub)
			if subMin < min {
				min = subMin
			}
		}
		return min

	default:
		return 0
	}
}

// maxMatchLen computes the maximum number of bytes a match can have.
// Returns -1 if the match length is unbounded (e.g., patterns with * or +).
func maxMatchLen(re *syntax.Regexp) int {
	if re == nil {
		return 0
	}

	switch re.Op {
	case syntax.OpNoMatch:
		return 0

	case syntax.OpEmptyMatch:
		return 0

	case syntax.OpLiteral:
		// Sum of UTF-8 encoded lengths of all runes
		total := 0
		for _, r := range re.Rune {
			total += utf8.RuneLen(r)
		}
		return total

	case syntax.OpCharClass:
		// Maximum byte length of any rune in the class
		if len(re.Rune) == 0 {
			return 0
		}
		maxLen := 1
		for i := 0; i < len(re.Rune); i += 2 {
			hi := re.Rune[i+1]
			// The largest rune in this range determines max bytes
			runeLen := utf8.RuneLen(hi)
			if runeLen > maxLen {
				maxLen = runeLen
			}
		}
		return maxLen

	case syntax.OpAnyCharNotNL, syntax.OpAnyChar:
		// Could be up to 4 bytes for Unicode
		return 4

	case syntax.OpBeginLine, syntax.OpEndLine,
		syntax.OpBeginText, syntax.OpEndText,
		syntax.OpWordBoundary, syntax.OpNoWordBoundary:
		// Zero-width assertions
		return 0

	case syntax.OpCapture:
		if len(re.Sub) > 0 {
			return maxMatchLen(re.Sub[0])
		}
		return 0

	case syntax.OpStar, syntax.OpPlus:
		// Unbounded
		return -1

	case syntax.OpQuest:
		// Zero or one - maximum is one occurrence
		if len(re.Sub) > 0 {
			return maxMatchLen(re.Sub[0])
		}
		return 0

	case syntax.OpRepeat:
		// {n,m} - if m is -1 (unbounded), return -1
		if re.Max == -1 {
			return -1
		}
		if len(re.Sub) > 0 {
			subMax := maxMatchLen(re.Sub[0])
			if subMax == -1 {
				return -1
			}
			return re.Max * subMax
		}
		return 0

	case syntax.OpConcat:
		// Sum of all parts - if any is unbounded, result is unbounded
		total := 0
		for _, sub := range re.Sub {
			subMax := maxMatchLen(sub)
			if subMax == -1 {
				return -1
			}
			total += subMax
		}
		return total

	case syntax.OpAlternate:
		// Maximum of all alternatives - if any is unbounded, result is unbounded
		if len(re.Sub) == 0 {
			return 0
		}
		max := 0
		for _, sub := range re.Sub {
			subMax := maxMatchLen(sub)
			if subMax == -1 {
				return -1
			}
			if subMax > max {
				max = subMax
			}
		}
		return max

	default:
		return 0
	}
}

// DefaultMaxLeftover returns a sensible default for MaxLeftover based on pattern analysis.
// For bounded patterns, returns 10 * maxMatchLen.
// For unbounded patterns, returns 1MB as a safety limit.
func DefaultMaxLeftover(analysis MatchLengthAnalysis) int {
	const maxDefault = 1 << 20 // 1MB

	if analysis.MaxMatchLen == -1 {
		// Unbounded pattern - use safety limit
		return maxDefault
	}

	suggested := analysis.MaxMatchLen * 10
	if suggested > maxDefault {
		return maxDefault
	}
	if suggested < 1024 {
		return 1024 // Minimum 1KB
	}
	return suggested
}

// MinBufferSize returns the minimum buffer size required for streaming.
// This must be at least 2 * maxMatchLen to ensure we can always find matches
// that span chunk boundaries. For unbounded patterns, returns a sensible default.
func MinBufferSize(analysis MatchLengthAnalysis) int {
	const defaultMin = 64 * 1024 // 64KB default minimum

	if analysis.MaxMatchLen == -1 {
		// Unbounded - use default
		return defaultMin
	}

	min := 2 * analysis.MaxMatchLen
	if min < defaultMin {
		return defaultMin
	}
	return min
}
