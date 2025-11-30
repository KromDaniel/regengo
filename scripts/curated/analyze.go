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
	"strings"
)

type patternCategory string

const (
	categorySimple      patternCategory = "simple"
	categoryComplex     patternCategory = "complex"
	categoryVeryComplex patternCategory = "very_complex"
	categoryTDFA        patternCategory = "tdfa"
)

type benchmarkResult struct {
	Name        string
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

	// Detect categories from benchmark names
	comparisons := analyzeBenchmarks(benchResults)
	printBenchmarkAnalysis(comparisons)
}

func parseBenchmarkResults(output string) map[string]*benchmarkResult {
	results := make(map[string]*benchmarkResult)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if !strings.HasPrefix(line, "Benchmark") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		name := fields[0]

		var nsPerOp float64
		var bytesPerOp, allocsPerOp int64

		for i := 2; i < len(fields); i++ {
			if i+1 < len(fields) && fields[i+1] == "ns/op" {
				fmt.Sscanf(fields[i], "%f", &nsPerOp)
			}
			if i+1 < len(fields) && fields[i+1] == "B/op" {
				fmt.Sscanf(fields[i], "%d", &bytesPerOp)
			}
			if i+1 < len(fields) && fields[i+1] == "allocs/op" {
				fmt.Sscanf(fields[i], "%d", &allocsPerOp)
			}
		}

		results[name] = &benchmarkResult{
			Name:        name,
			NsPerOp:     nsPerOp,
			BytesPerOp:  bytesPerOp,
			AllocsPerOp: allocsPerOp,
		}
	}

	return results
}

func detectCategory(benchName string) patternCategory {
	lowerName := strings.ToLower(benchName)
	if strings.Contains(lowerName, "tdfa") {
		return categoryTDFA
	}
	if strings.Contains(lowerName, "verycomplex") {
		return categoryVeryComplex
	}
	if strings.Contains(lowerName, "complex") {
		return categoryComplex
	}
	return categorySimple
}

func analyzeBenchmarks(benchResults map[string]*benchmarkResult) map[patternCategory]*benchmarkComparison {
	comparisons := make(map[patternCategory]*benchmarkComparison)

	for _, cat := range []patternCategory{categorySimple, categoryComplex, categoryVeryComplex, categoryTDFA} {
		comparisons[cat] = &benchmarkComparison{Category: cat}
	}

	// Find pairs of regengo vs stdlib benchmarks
	for name, result := range benchResults {
		if strings.Contains(name, "golang_std") {
			continue
		}

		// Find stdlib counterpart
		var stdlibName string
		if strings.Contains(name, "/regengo") {
			stdlibName = strings.Replace(name, "/regengo", "/golang_std", 1)
		} else {
			continue
		}

		stdlibResult, hasStdlib := benchResults[stdlibName]
		if !hasStdlib {
			continue
		}

		category := detectCategory(name)
		comp := comparisons[category]

		if result.NsPerOp < stdlibResult.NsPerOp {
			comp.RegengoFaster++
		} else {
			comp.StdlibFaster++
			slowerByPct := ((result.NsPerOp - stdlibResult.NsPerOp) / stdlibResult.NsPerOp) * 100
			comp.SlowerPatterns = append(comp.SlowerPatterns, slowPattern{
				Name:        name,
				RegengoNs:   result.NsPerOp,
				StdlibNs:    stdlibResult.NsPerOp,
				SlowerByPct: slowerByPct,
			})
		}

		comp.RegengoAvgNs += result.NsPerOp
		comp.StdlibAvgNs += stdlibResult.NsPerOp
		comp.RegengoAvgBytes += float64(result.BytesPerOp)
		comp.StdlibAvgBytes += float64(stdlibResult.BytesPerOp)
		comp.RegengoAvgAllocs += float64(result.AllocsPerOp)
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

	orderedCategories := []patternCategory{categorySimple, categoryComplex, categoryVeryComplex, categoryTDFA}

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
		} else {
			fmt.Printf("[%.1f%% more]\n", -bytesDiff)
		}

		allocsDiff := 0.0
		if comp.StdlibAvgAllocs > 0 {
			allocsDiff = ((comp.StdlibAvgAllocs - comp.RegengoAvgAllocs) / comp.StdlibAvgAllocs) * 100
		}
		fmt.Printf("  Avg allocs:     %.1f allocs/op (regengo) vs %.1f allocs/op (stdlib) ",
			comp.RegengoAvgAllocs, comp.StdlibAvgAllocs)
		if allocsDiff > 0 {
			fmt.Printf("[%.1f%% less]\n", allocsDiff)
		} else {
			fmt.Printf("[%.1f%% more]\n", -allocsDiff)
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

		overallBytesDiff := ((overallStdlibBytes - overallRegengoBytes) / overallStdlibBytes) * 100
		fmt.Printf("Avg memory:     %.0f B/op (regengo) vs %.0f B/op (stdlib) ",
			overallRegengoBytes, overallStdlibBytes)
		if overallBytesDiff > 0 {
			fmt.Printf("[%.1f%% less]\n", overallBytesDiff)
		} else {
			fmt.Printf("[%.1f%% more]\n", -overallBytesDiff)
		}

		overallAllocsDiff := ((overallStdlibAllocs - overallRegengoAllocs) / overallStdlibAllocs) * 100
		fmt.Printf("Avg allocs:     %.1f allocs/op (regengo) vs %.1f allocs/op (stdlib) ",
			overallRegengoAllocs, overallStdlibAllocs)
		if overallAllocsDiff > 0 {
			fmt.Printf("[%.1f%% less]\n", overallAllocsDiff)
		} else {
			fmt.Printf("[%.1f%% more]\n", -overallAllocsDiff)
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
