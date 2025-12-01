// aggregate.go aggregates benchmark results across inputs for summary comparison.
//
// Usage:
//
//	go test ./benchmarks/curated/... -bench="." -benchmem | go run scripts/curated/aggregate.go
//	go test ./benchmarks/curated/... -bench="DateCapture" -benchmem | go run scripts/curated/aggregate.go
package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// BenchmarkResult holds parsed benchmark data
type BenchmarkResult struct {
	Pattern    string
	Category   string
	Input      string
	Variant    string
	Template   string // for Replace category
	NsPerOp    float64
	BytesOp    int64
	AllocsOp   int64
	Iterations int64
}

// AggregatedResult holds aggregated statistics
type AggregatedResult struct {
	Variant   string
	Count     int
	MinNs     float64
	MaxNs     float64
	AvgNs     float64
	MedianNs  float64
	AvgBytes  int64
	AvgAllocs int64
	AllNs     []float64
}

// benchmarkLineRe matches benchmark output lines
// Example: BenchmarkDateCapture/Match/Input[0]/stdlib-12  	16418577	        72.75 ns/op	       0 B/op	       0 allocs/op
var benchmarkLineRe = regexp.MustCompile(
	`^(Benchmark\S+)-\d+\s+(\d+)\s+([\d.]+)\s+ns/op(?:\s+(\d+)\s+B/op)?(?:\s+(\d+)\s+allocs/op)?`,
)

func main() {
	results := parseBenchmarks(os.Stdin)
	if len(results) == 0 {
		fmt.Println("No benchmark results found.")
		return
	}

	// Group by pattern -> category -> (template ->) variant
	grouped := groupResults(results)

	// Print aggregated results
	printAggregated(grouped)
}

func parseBenchmarks(f *os.File) []BenchmarkResult {
	var results []BenchmarkResult
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		match := benchmarkLineRe.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		name := match[1]
		iterations, _ := strconv.ParseInt(match[2], 10, 64)
		nsPerOp, _ := strconv.ParseFloat(match[3], 64)
		var bytesOp, allocsOp int64
		if match[4] != "" {
			bytesOp, _ = strconv.ParseInt(match[4], 10, 64)
		}
		if match[5] != "" {
			allocsOp, _ = strconv.ParseInt(match[5], 10, 64)
		}

		result := parseBenchmarkName(name)
		if result == nil {
			continue
		}

		result.NsPerOp = nsPerOp
		result.BytesOp = bytesOp
		result.AllocsOp = allocsOp
		result.Iterations = iterations

		results = append(results, *result)
	}

	return results
}

// parseBenchmarkName extracts components from benchmark name
// Examples:
//   - BenchmarkDateCapture/Match/Input[0]/stdlib
//   - BenchmarkReplaceDate/Replace/Template[0]/Input[0]/regengo_append
func parseBenchmarkName(name string) *BenchmarkResult {
	// Remove "Benchmark" prefix
	if !strings.HasPrefix(name, "Benchmark") {
		return nil
	}
	name = strings.TrimPrefix(name, "Benchmark")

	parts := strings.Split(name, "/")
	if len(parts) < 3 {
		return nil
	}

	result := &BenchmarkResult{
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

// groupResults groups results by pattern -> category -> (template ->) variant
func groupResults(results []BenchmarkResult) map[string]map[string]map[string][]BenchmarkResult {
	grouped := make(map[string]map[string]map[string][]BenchmarkResult)

	for _, r := range results {
		if grouped[r.Pattern] == nil {
			grouped[r.Pattern] = make(map[string]map[string][]BenchmarkResult)
		}

		categoryKey := r.Category
		if r.Template != "" {
			categoryKey = r.Category + "/" + r.Template
		}

		if grouped[r.Pattern][categoryKey] == nil {
			grouped[r.Pattern][categoryKey] = make(map[string][]BenchmarkResult)
		}

		grouped[r.Pattern][categoryKey][r.Variant] = append(
			grouped[r.Pattern][categoryKey][r.Variant], r,
		)
	}

	return grouped
}

func aggregateVariant(results []BenchmarkResult) AggregatedResult {
	if len(results) == 0 {
		return AggregatedResult{}
	}

	var allNs []float64
	var totalBytes, totalAllocs int64

	for _, r := range results {
		allNs = append(allNs, r.NsPerOp)
		totalBytes += r.BytesOp
		totalAllocs += r.AllocsOp
	}

	sort.Float64s(allNs)

	var sum float64
	for _, ns := range allNs {
		sum += ns
	}

	median := allNs[len(allNs)/2]
	if len(allNs)%2 == 0 && len(allNs) > 1 {
		median = (allNs[len(allNs)/2-1] + allNs[len(allNs)/2]) / 2
	}

	return AggregatedResult{
		Variant:   results[0].Variant,
		Count:     len(results),
		MinNs:     allNs[0],
		MaxNs:     allNs[len(allNs)-1],
		AvgNs:     sum / float64(len(allNs)),
		MedianNs:  median,
		AvgBytes:  totalBytes / int64(len(results)),
		AvgAllocs: totalAllocs / int64(len(results)),
		AllNs:     allNs,
	}
}

func printAggregated(grouped map[string]map[string]map[string][]BenchmarkResult) {
	// Sort patterns for consistent output
	var patterns []string
	for p := range grouped {
		patterns = append(patterns, p)
	}
	sort.Strings(patterns)

	for _, pattern := range patterns {
		fmt.Printf("\n%s\n", strings.Repeat("=", 60))
		fmt.Printf("Pattern: %s\n", pattern)
		fmt.Printf("%s\n", strings.Repeat("=", 60))

		categories := grouped[pattern]

		// Sort categories
		var categoryKeys []string
		for c := range categories {
			categoryKeys = append(categoryKeys, c)
		}
		sort.Strings(categoryKeys)

		for _, categoryKey := range categoryKeys {
			variants := categories[categoryKey]

			fmt.Printf("\n  Category: %s\n", categoryKey)
			fmt.Printf("  %s\n", strings.Repeat("-", 50))

			// Aggregate each variant
			var aggregated []AggregatedResult
			for variant, results := range variants {
				agg := aggregateVariant(results)
				agg.Variant = variant
				aggregated = append(aggregated, agg)
			}

			// Sort variants: stdlib first, then regengo variants
			sort.Slice(aggregated, func(i, j int) bool {
				order := map[string]int{
					"stdlib":          0,
					"regengo":         1,
					"regengo_runtime": 2,
					"regengo_reuse":   3,
					"regengo_append":  4,
				}
				oi, ok1 := order[aggregated[i].Variant]
				oj, ok2 := order[aggregated[j].Variant]
				if !ok1 {
					oi = 100
				}
				if !ok2 {
					oj = 100
				}
				return oi < oj
			})

			// Find stdlib for speedup calculation
			var stdlibAvg float64
			for _, agg := range aggregated {
				if agg.Variant == "stdlib" {
					stdlibAvg = agg.AvgNs
					break
				}
			}

			// Print results
			for _, agg := range aggregated {
				speedup := ""
				if stdlibAvg > 0 && agg.Variant != "stdlib" {
					speedup = fmt.Sprintf("  (%.1fx faster)", stdlibAvg/agg.AvgNs)
				}

				fmt.Printf("    %-18s avg=%8.2f ns  min=%8.2f  max=%8.2f  allocs=%d%s\n",
					agg.Variant+":",
					agg.AvgNs,
					agg.MinNs,
					agg.MaxNs,
					agg.AvgAllocs,
					speedup,
				)
			}
		}
	}
	fmt.Println()
}
