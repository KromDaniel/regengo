//go:build ignore

// Benchmark analysis tool - analyzes and summarizes benchmark results.
//
// Usage:
//
//	go test -bench=. -benchmem ./benchmarks/curated/... 2>&1 | go run scripts/curated/analyze.go
package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type patternCategory string

const (
	categoryMatch   patternCategory = "match"
	categoryCapture patternCategory = "capture"
	categoryFindAll patternCategory = "findall"
	categoryTDFA    patternCategory = "tdfa"
	categoryTNFA    patternCategory = "tnfa"
	categoryReplace patternCategory = "replace"
)

type benchmarkResult struct {
	Pattern     string
	Category    string // Match, FindFirst, FindAll, Replace
	Input       string
	Variant     string // stdlib, regengo, regengo_reuse, regengo_append, regengo_runtime
	Template    string // for Replace category
	NsPerOp     float64
	BytesPerOp  int64
	AllocsPerOp int64
}

type benchmarkComparison struct {
	Category         patternCategory
	RegengoFaster    int
	StdlibFaster     int
	RegengoAvgNs     float64
	StdlibAvgNs      float64
	RegengoAvgBytes  float64
	StdlibAvgBytes   float64
	RegengoAvgAllocs float64
	StdlibAvgAllocs  float64
	SlowerPatterns   []slowPattern
}

type slowPattern struct {
	Name        string
	Pattern     string
	RegengoNs   float64
	StdlibNs    float64
	SlowerByPct float64
}

// benchmarkLineRe matches new nested benchmark output lines
// Example: BenchmarkDateCapture/Match/Input[0]/stdlib-12  	16418577	        72.75 ns/op	       0 B/op	       0 allocs/op
var benchmarkLineRe = regexp.MustCompile(
	`^(Benchmark\S+)-\d+\s+\d+\s+([\d.]+)\s+ns/op(?:\s+(\d+)\s+B/op)?(?:\s+(\d+)\s+allocs/op)?`,
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	output := strings.Join(lines, "\n")
	benchResults := parseBenchmarkResults(output)

	if len(benchResults) == 0 {
		fmt.Println("No benchmark results found in input.")
		os.Exit(0)
	}

	// Analyze and compare results
	comparisons := analyzeBenchmarks(benchResults)
	printBenchmarkAnalysis(comparisons)
}

func parseBenchmarkResults(output string) []benchmarkResult {
	var results []benchmarkResult
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if !strings.HasPrefix(line, "Benchmark") {
			continue
		}

		match := benchmarkLineRe.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		name := match[1]
		var nsPerOp float64
		var bytesPerOp, allocsPerOp int64

		fmt.Sscanf(match[2], "%f", &nsPerOp)
		if match[3] != "" {
			fmt.Sscanf(match[3], "%d", &bytesPerOp)
		}
		if match[4] != "" {
			fmt.Sscanf(match[4], "%d", &allocsPerOp)
		}

		result := parseBenchmarkName(name)
		if result == nil {
			continue
		}

		result.NsPerOp = nsPerOp
		result.BytesPerOp = bytesPerOp
		result.AllocsPerOp = allocsPerOp

		results = append(results, *result)
	}

	return results
}

// parseBenchmarkName extracts components from nested benchmark name
// Examples:
//   - BenchmarkDateCapture/Match/Input[0]/stdlib
//   - BenchmarkDateCapture/FindFirst/Input[0]/regengo_reuse
//   - BenchmarkReplaceDate/Replace/Template[0]/Input[0]/regengo_append
func parseBenchmarkName(name string) *benchmarkResult {
	if !strings.HasPrefix(name, "Benchmark") {
		return nil
	}
	name = strings.TrimPrefix(name, "Benchmark")

	parts := strings.Split(name, "/")
	if len(parts) < 4 {
		return nil
	}

	result := &benchmarkResult{
		Pattern: parts[0],
	}

	// Handle different structures
	// Match/FindFirst/FindAll: Pattern/Category/Input[i]/variant
	// Replace: Pattern/Category/Template[j]/Input[i]/variant
	if parts[1] == "Replace" && len(parts) >= 5 {
		result.Category = parts[1]
		result.Template = parts[2]
		result.Input = parts[3]
		result.Variant = parts[4]
	} else if len(parts) >= 4 {
		result.Category = parts[1]
		result.Input = parts[2]
		result.Variant = parts[3]
	} else {
		return nil
	}

	return result
}

func detectPatternCategory(patternName string) patternCategory {
	lowerName := strings.ToLower(patternName)
	if strings.HasPrefix(lowerName, "tdfa") {
		return categoryTDFA
	}
	if strings.HasPrefix(lowerName, "tnfa") {
		return categoryTNFA
	}
	if strings.HasPrefix(lowerName, "replace") {
		return categoryReplace
	}
	if strings.HasPrefix(lowerName, "multi") {
		return categoryFindAll
	}
	if strings.Contains(lowerName, "capture") {
		return categoryCapture
	}
	return categoryMatch
}

func analyzeBenchmarks(benchResults []benchmarkResult) map[patternCategory]*benchmarkComparison {
	comparisons := make(map[patternCategory]*benchmarkComparison)

	for _, cat := range []patternCategory{categoryMatch, categoryCapture, categoryFindAll, categoryTDFA, categoryTNFA, categoryReplace} {
		comparisons[cat] = &benchmarkComparison{Category: cat}
	}

	// Group results by pattern+category+input+template to find stdlib vs regengo pairs
	type resultKey struct {
		pattern  string
		category string
		input    string
		template string
	}

	grouped := make(map[resultKey]map[string]benchmarkResult) // key -> variant -> result

	for _, r := range benchResults {
		key := resultKey{
			pattern:  r.Pattern,
			category: r.Category,
			input:    r.Input,
			template: r.Template,
		}
		if grouped[key] == nil {
			grouped[key] = make(map[string]benchmarkResult)
		}
		grouped[key][r.Variant] = r
	}

	// Compare stdlib vs regengo (default variant, not reuse/append)
	for key, variants := range grouped {
		stdlibResult, hasStdlib := variants["stdlib"]
		regengoResult, hasRegengo := variants["regengo"]

		if !hasStdlib || !hasRegengo {
			continue
		}

		patCat := detectPatternCategory(key.pattern)
		comp := comparisons[patCat]

		if regengoResult.NsPerOp < stdlibResult.NsPerOp {
			comp.RegengoFaster++
		} else {
			comp.StdlibFaster++
			slowerByPct := ((regengoResult.NsPerOp - stdlibResult.NsPerOp) / stdlibResult.NsPerOp) * 100
			benchName := fmt.Sprintf("%s/%s/%s", key.pattern, key.category, key.input)
			if key.template != "" {
				benchName = fmt.Sprintf("%s/%s/%s/%s", key.pattern, key.category, key.template, key.input)
			}
			comp.SlowerPatterns = append(comp.SlowerPatterns, slowPattern{
				Name:        benchName,
				RegengoNs:   regengoResult.NsPerOp,
				StdlibNs:    stdlibResult.NsPerOp,
				SlowerByPct: slowerByPct,
			})
		}

		comp.RegengoAvgNs += regengoResult.NsPerOp
		comp.StdlibAvgNs += stdlibResult.NsPerOp
		comp.RegengoAvgBytes += float64(regengoResult.BytesPerOp)
		comp.StdlibAvgBytes += float64(stdlibResult.BytesPerOp)
		comp.RegengoAvgAllocs += float64(regengoResult.AllocsPerOp)
		comp.StdlibAvgAllocs += float64(stdlibResult.AllocsPerOp)
	}

	// Calculate averages
	for _, comp := range comparisons {
		total := float64(comp.RegengoFaster + comp.StdlibFaster)
		if total > 0 {
			comp.RegengoAvgNs /= total
			comp.StdlibAvgNs /= total
			comp.RegengoAvgBytes /= total
			comp.StdlibAvgBytes /= total
			comp.RegengoAvgAllocs /= total
			comp.StdlibAvgAllocs /= total
		}
	}

	// Sort slower patterns by percentage
	for _, comp := range comparisons {
		for i := 0; i < len(comp.SlowerPatterns)-1; i++ {
			for j := i + 1; j < len(comp.SlowerPatterns); j++ {
				if comp.SlowerPatterns[i].SlowerByPct < comp.SlowerPatterns[j].SlowerByPct {
					comp.SlowerPatterns[i], comp.SlowerPatterns[j] = comp.SlowerPatterns[j], comp.SlowerPatterns[i]
				}
			}
		}
	}

	return comparisons
}

func printBenchmarkAnalysis(comparisons map[patternCategory]*benchmarkComparison) {
	fmt.Println("======== Benchmark Comparison Summary ========")
	fmt.Println()

	orderedCategories := []patternCategory{categoryMatch, categoryCapture, categoryFindAll, categoryTDFA, categoryTNFA, categoryReplace}

	var totalRegengoFaster, totalStdlibFaster int
	var overallRegengoNs, overallStdlibNs float64
	var overallRegengoBytes, overallStdlibBytes float64
	var overallRegengoAllocs, overallStdlibAllocs float64

	for _, cat := range orderedCategories {
		comp := comparisons[cat]
		if comp.RegengoFaster+comp.StdlibFaster == 0 {
			continue
		}

		fmt.Printf("Category: %s\n", cat)
		fmt.Printf("  Regengo faster: %3d  |  Stdlib faster: %3d\n", comp.RegengoFaster, comp.StdlibFaster)

		speedupPct := 0.0
		if comp.StdlibAvgNs > 0 {
			speedupPct = ((comp.StdlibAvgNs - comp.RegengoAvgNs) / comp.StdlibAvgNs) * 100
		}

		fmt.Printf("  Avg time:       %.0f ns/op (regengo) vs %.0f ns/op (stdlib) ",
			comp.RegengoAvgNs, comp.StdlibAvgNs)
		if speedupPct > 0 {
			fmt.Printf("[%.1f%% faster]\n", speedupPct)
		} else {
			fmt.Printf("[%.1f%% slower]\n", -speedupPct)
		}

		bytesDiff := 0.0
		if comp.StdlibAvgBytes > 0 {
			bytesDiff = ((comp.StdlibAvgBytes - comp.RegengoAvgBytes) / comp.StdlibAvgBytes) * 100
		}
		fmt.Printf("  Avg memory:     %.0f B/op (regengo) vs %.0f B/op (stdlib) ",
			comp.RegengoAvgBytes, comp.StdlibAvgBytes)
		if bytesDiff > 0 {
			fmt.Printf("[%.1f%% less]\n", bytesDiff)
		} else if bytesDiff < 0 {
			fmt.Printf("[%.1f%% more]\n", -bytesDiff)
		} else {
			fmt.Printf("[same]\n")
		}

		allocsDiff := 0.0
		if comp.StdlibAvgAllocs > 0 {
			allocsDiff = ((comp.StdlibAvgAllocs - comp.RegengoAvgAllocs) / comp.StdlibAvgAllocs) * 100
		}
		fmt.Printf("  Avg allocs:     %.1f allocs/op (regengo) vs %.1f allocs/op (stdlib) ",
			comp.RegengoAvgAllocs, comp.StdlibAvgAllocs)
		if allocsDiff > 0 {
			fmt.Printf("[%.1f%% less]\n", allocsDiff)
		} else if allocsDiff < 0 {
			fmt.Printf("[%.1f%% more]\n", -allocsDiff)
		} else {
			fmt.Printf("[same]\n")
		}
		fmt.Println()

		totalRegengoFaster += comp.RegengoFaster
		totalStdlibFaster += comp.StdlibFaster

		total := float64(comp.RegengoFaster + comp.StdlibFaster)
		overallRegengoNs += comp.RegengoAvgNs * total
		overallStdlibNs += comp.StdlibAvgNs * total
		overallRegengoBytes += comp.RegengoAvgBytes * total
		overallStdlibBytes += comp.StdlibAvgBytes * total
		overallRegengoAllocs += comp.RegengoAvgAllocs * total
		overallStdlibAllocs += comp.StdlibAvgAllocs * total
	}

	if totalRegengoFaster+totalStdlibFaster > 0 {
		totalComparisons := float64(totalRegengoFaster + totalStdlibFaster)
		overallRegengoNs /= totalComparisons
		overallStdlibNs /= totalComparisons
		overallRegengoBytes /= totalComparisons
		overallStdlibBytes /= totalComparisons
		overallRegengoAllocs /= totalComparisons
		overallStdlibAllocs /= totalComparisons

		fmt.Println("========== OVERALL SUMMARY ==========")
		fmt.Printf("Regengo faster: %3d  |  Stdlib faster: %3d\n", totalRegengoFaster, totalStdlibFaster)

		overallSpeedupPct := ((overallStdlibNs - overallRegengoNs) / overallStdlibNs) * 100
		fmt.Printf("Avg time:       %.0f ns/op (regengo) vs %.0f ns/op (stdlib) ",
			overallRegengoNs, overallStdlibNs)
		if overallSpeedupPct > 0 {
			fmt.Printf("[%.1f%% faster]\n", overallSpeedupPct)
		} else {
			fmt.Printf("[%.1f%% slower]\n", -overallSpeedupPct)
		}

		overallBytesDiff := 0.0
		if overallStdlibBytes > 0 {
			overallBytesDiff = ((overallStdlibBytes - overallRegengoBytes) / overallStdlibBytes) * 100
		}
		fmt.Printf("Avg memory:     %.0f B/op (regengo) vs %.0f B/op (stdlib) ",
			overallRegengoBytes, overallStdlibBytes)
		if overallBytesDiff > 0 {
			fmt.Printf("[%.1f%% less]\n", overallBytesDiff)
		} else if overallBytesDiff < 0 {
			fmt.Printf("[%.1f%% more]\n", -overallBytesDiff)
		} else {
			fmt.Printf("[same]\n")
		}

		overallAllocsDiff := 0.0
		if overallStdlibAllocs > 0 {
			overallAllocsDiff = ((overallStdlibAllocs - overallRegengoAllocs) / overallStdlibAllocs) * 100
		}
		fmt.Printf("Avg allocs:     %.1f allocs/op (regengo) vs %.1f allocs/op (stdlib) ",
			overallRegengoAllocs, overallStdlibAllocs)
		if overallAllocsDiff > 0 {
			fmt.Printf("[%.1f%% less]\n", overallAllocsDiff)
		} else if overallAllocsDiff < 0 {
			fmt.Printf("[%.1f%% more]\n", -overallAllocsDiff)
		} else {
			fmt.Printf("[same]\n")
		}
	}

	// Print patterns where stdlib wins
	printSlowerPatternsAnalysis(comparisons, orderedCategories)
}

func printSlowerPatternsAnalysis(comparisons map[patternCategory]*benchmarkComparison, orderedCategories []patternCategory) {
	fmt.Println()
	fmt.Println("========== PATTERNS WHERE STDLIB WINS ==========")

	hasSlowerPatterns := false
	for _, cat := range orderedCategories {
		comp := comparisons[cat]
		if len(comp.SlowerPatterns) == 0 {
			continue
		}

		hasSlowerPatterns = true
		fmt.Printf("\nCategory: %s (%d patterns slower)\n", cat, len(comp.SlowerPatterns))

		limit := 10
		if len(comp.SlowerPatterns) < limit {
			limit = len(comp.SlowerPatterns)
		}

		for i := 0; i < limit; i++ {
			sp := comp.SlowerPatterns[i]
			fmt.Printf("  %2d. %s\n", i+1, sp.Name)
			fmt.Printf("      Regengo: %.0f ns/op | Stdlib: %.0f ns/op | Slower by: %.1f%%\n",
				sp.RegengoNs, sp.StdlibNs, sp.SlowerByPct)
		}

		if len(comp.SlowerPatterns) > limit {
			fmt.Printf("  ... and %d more patterns\n", len(comp.SlowerPatterns)-limit)
		}
	}

	if !hasSlowerPatterns {
		fmt.Println("\n  No patterns where stdlib is faster!")
	}
}
