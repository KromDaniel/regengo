package regengo

import (
	"github.com/KromDaniel/regengo/internal/compiler"
)

// AnalysisResult contains the results of pattern analysis without code generation.
// This is useful for determining which labels apply to a pattern for testing.
type AnalysisResult = compiler.AnalysisResult

// Analyze performs pattern analysis and returns labels without generating code.
// This function validates that the pattern is valid and returns an error if not.
//
// The analysis returns:
//   - FeatureLabels: Derived from pattern structure (e.g., "Captures", "Multibyte")
//   - EngineLabels: Derived from compilation analysis (e.g., "TDFA", "Thompson")
//
// Both label arrays are sorted alphabetically for deterministic comparison.
//
// Example:
//
//	result, err := regengo.Analyze("(?P<name>\\w+)")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(result.FeatureLabels)  // ["Captures", "CharClass", "Quantifiers"]
//	fmt.Println(result.EngineLabels)   // ["Backtracking"]
func Analyze(pattern string) (*AnalysisResult, error) {
	return AnalyzeWithThreshold(pattern, 500)
}

// AnalyzeWithThreshold performs pattern analysis with a custom TDFA state threshold.
// The tdfaThreshold controls when TDFA falls back to other engines (default: 500).
func AnalyzeWithThreshold(pattern string, tdfaThreshold int) (*AnalysisResult, error) {
	return compiler.AnalyzePattern(pattern, tdfaThreshold)
}
