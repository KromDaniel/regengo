//go:build ignore

// Benchmark formatting tool - formats benchmark results as detailed markdown.
//
// Usage:
//
//	go test -bench=. -benchmem ./benchmarks/curated/... 2>&1 | go run scripts/curated/format.go
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

type BenchResult struct {
	FullName    string
	PatternName string
	Method      string // MatchString, FindString, FindStringReuse, FindAllStringAppend, ReplaceAllString, etc.
	IsStdlib    bool
	NsOp        float64
	BOp         int
	Allocs      int
}

type PatternBenchmarks struct {
	Name    string
	Pattern string
	Results []BenchResult
}

// Pattern definitions from cases.go
var patternMap = map[string]string{
	"DateCapture":      `(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})`,
	"EmailCapture":     `(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)`,
	"URLCapture":       `(?P<protocol>https?)://(?P<host>[\w\.-]+)(?::(?P<port>\d+))?(?P<path>/[\w\./]*)?`,
	"Email":            `[\w\.+-]+@[\w\.-]+\.[\w\.-]+`,
	"Greedy":           `(?:(?:a|b)|(?:k)+)*abcd`,
	"Lazy":             `(?:(?:a|b)|(?:k)+)+?abcd`,
	"MultiDate":        `(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})`,
	"MultiEmail":       `(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)`,
	"TDFAPathological": `(?P<outer>(?P<inner>a+)+)b`,
	"TDFANestedWord":   `(?P<words>(?P<word>\w+\s*)+)end`,
	"TDFAComplexURL":   `(?P<scheme>https?)://(?P<auth>(?P<user>[\w.-]+)(?::(?P<pass>[\w.-]+))?@)?(?P<host>[\w.-]+)(?::(?P<port>\d+))?(?P<path>/[\w./-]*)?(?:\?(?P<query>[\w=&.-]+))?`,
	"TDFALogParser":    `(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})(?:\.(?P<ms>\d{3}))?(?P<tz>Z|[+-]\d{2}:\d{2})?\s+\[(?P<level>\w+)\]\s+(?P<message>.+)`,
	"TDFASemVer":       `(?P<major>\d+)\.(?P<minor>\d+)\.(?P<patch>\d+)(?:-(?P<prerelease>[\w.-]+))?(?:\+(?P<build>[\w.-]+))?`,
	"TNFAPathological": `(?P<outer>(?P<inner>a+)+)b`,
	"ReplaceEmail":     `(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)`,
	"ReplaceDate":      `(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})`,
	"ReplaceURL":       `(?P<protocol>https?)://(?P<host>[\w\.-]+)(?::(?P<port>\d+))?(?P<path>/[\w\./]*)?`,
}

// Known pattern names in order of preference for extraction
var knownPatterns = []string{
	"TDFAPathological", "TDFANestedWord", "TDFAComplexURL", "TDFALogParser", "TDFASemVer",
	"TNFAPathological",
	"ReplaceEmail", "ReplaceDate", "ReplaceURL",
	"DateCapture", "EmailCapture", "URLCapture",
	"MultiDate", "MultiEmail",
	"Email", "Greedy", "Lazy",
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	// Regex to parse benchmark lines
	// Format: BenchmarkName-CPU  iterations  ns/op  B/op  allocs/op
	benchRegex := regexp.MustCompile(`^(Benchmark\w+)-\d+\s+\d+\s+([\d.]+)\s+ns/op\s+(\d+)\s+B/op\s+(\d+)\s+allocs/op`)

	var results []BenchResult

	for scanner.Scan() {
		line := scanner.Text()
		matches := benchRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		fullName := matches[1]
		nsOp, _ := strconv.ParseFloat(matches[2], 64)
		bOp, _ := strconv.Atoi(matches[3])
		allocs, _ := strconv.Atoi(matches[4])

		result := parseBenchmarkName(fullName)
		result.NsOp = nsOp
		result.BOp = bOp
		result.Allocs = allocs

		results = append(results, result)
	}

	if len(results) == 0 {
		fmt.Fprintln(os.Stderr, "No benchmark results found in input.")
		fmt.Fprintln(os.Stderr, "Usage: go test -bench=. -benchmem ./benchmarks/curated/... 2>&1 | go run scripts/curated/format.go")
		os.Exit(1)
	}

	// Group by pattern name
	patternGroups := make(map[string]*PatternBenchmarks)
	for _, r := range results {
		if _, ok := patternGroups[r.PatternName]; !ok {
			patternGroups[r.PatternName] = &PatternBenchmarks{
				Name:    r.PatternName,
				Pattern: patternMap[r.PatternName],
				Results: []BenchResult{},
			}
		}
		patternGroups[r.PatternName].Results = append(patternGroups[r.PatternName].Results, r)
	}

	// Sort pattern names
	var patternNames []string
	for name := range patternGroups {
		patternNames = append(patternNames, name)
	}
	sort.Strings(patternNames)

	// Generate detailed markdown output
	generateDetailedMarkdown(patternGroups, patternNames)
}

func parseBenchmarkName(fullName string) BenchResult {
	result := BenchResult{FullName: fullName}

	// Remove "Benchmark" prefix
	name := strings.TrimPrefix(fullName, "Benchmark")

	// Check if it's a stdlib benchmark
	if strings.HasPrefix(name, "Stdlib") {
		result.IsStdlib = true
		name = strings.TrimPrefix(name, "Stdlib")
	}

	// Extract pattern name
	for _, pattern := range knownPatterns {
		if strings.HasPrefix(name, pattern) {
			result.PatternName = pattern
			result.Method = strings.TrimPrefix(name, pattern)
			break
		}
	}

	// If no pattern found, use the whole name
	if result.PatternName == "" {
		result.PatternName = name
		result.Method = "Unknown"
	}

	return result
}

func generateDetailedMarkdown(groups map[string]*PatternBenchmarks, patternNames []string) {
	fmt.Println("# Curated Benchmark Results")
	fmt.Println()
	fmt.Println("Detailed benchmark results comparing regengo vs Go stdlib regexp.")
	fmt.Println()
	fmt.Println("## Summary")
	fmt.Println()

	// Calculate overall stats
	var totalRegengo, totalStdlib int
	var regengoWins, stdlibWins int

	for _, name := range patternNames {
		group := groups[name]
		for _, r := range group.Results {
			if r.IsStdlib {
				totalStdlib++
			} else {
				totalRegengo++
			}
		}
	}

	// Count wins by comparing matching benchmarks
	for _, name := range patternNames {
		group := groups[name]
		stdlibResults := make(map[string]BenchResult)
		regengoResults := make(map[string]BenchResult)

		for _, r := range group.Results {
			if r.IsStdlib {
				stdlibResults[r.Method] = r
			} else {
				regengoResults[r.Method] = r
			}
		}

		// Compare MatchString
		if std, ok := stdlibResults["MatchString"]; ok {
			if reg, ok := regengoResults["MatchString"]; ok {
				if reg.NsOp < std.NsOp {
					regengoWins++
				} else {
					stdlibWins++
				}
			}
		}

		// Compare FindString vs FindStringSubmatch
		if std, ok := stdlibResults["FindStringSubmatch"]; ok {
			if reg, ok := regengoResults["FindString"]; ok {
				if reg.NsOp < std.NsOp {
					regengoWins++
				} else {
					stdlibWins++
				}
			}
		}
	}

	fmt.Printf("- **Patterns tested:** %d\n", len(patternNames))
	fmt.Printf("- **Total benchmarks:** %d\n", totalRegengo+totalStdlib)
	fmt.Printf("- **regengo wins:** %d\n", regengoWins)
	fmt.Printf("- **stdlib wins:** %d\n", stdlibWins)
	fmt.Println()

	fmt.Println("---")
	fmt.Println()
	fmt.Println("## Detailed Results by Pattern")
	fmt.Println()

	for _, name := range patternNames {
		group := groups[name]
		printPatternSection(group)
	}
}

func printPatternSection(group *PatternBenchmarks) {
	fmt.Printf("### %s\n\n", group.Name)

	if group.Pattern != "" {
		fmt.Printf("**Pattern:**\n```regex\n%s\n```\n\n", group.Pattern)
	}

	// Organize results by method type
	methodOrder := []string{
		"MatchString",
		"FindString", "FindStringSubmatch",
		"FindStringReuse",
		"FindAllStringAppend",
		"ReplaceAllString", "ReplaceAllStringN", "ReplaceAllBytesAppendN",
	}

	// Build result map
	stdlibResults := make(map[string]BenchResult)
	regengoResults := make(map[string]BenchResult)

	for _, r := range group.Results {
		if r.IsStdlib {
			stdlibResults[r.Method] = r
		} else {
			regengoResults[r.Method] = r
		}
	}

	// Print comparison table
	fmt.Println("| Benchmark | ns/op | B/op | allocs/op | Speedup |")
	fmt.Println("|-----------|------:|-----:|----------:|--------:|")

	// MatchString comparison
	if std, hasStd := stdlibResults["MatchString"]; hasStd {
		fmt.Printf("| stdlib MatchString | %.1f | %d | %d | - |\n", std.NsOp, std.BOp, std.Allocs)
	}
	if reg, hasReg := regengoResults["MatchString"]; hasReg {
		speedup := "-"
		if std, hasStd := stdlibResults["MatchString"]; hasStd && reg.NsOp > 0 {
			speedup = fmt.Sprintf("**%.1fx**", std.NsOp/reg.NsOp)
		}
		fmt.Printf("| regengo MatchString | %.1f | %d | %d | %s |\n", reg.NsOp, reg.BOp, reg.Allocs, speedup)
	}

	// FindString comparison
	if std, hasStd := stdlibResults["FindStringSubmatch"]; hasStd {
		fmt.Printf("| stdlib FindStringSubmatch | %.1f | %d | %d | - |\n", std.NsOp, std.BOp, std.Allocs)
	}
	if reg, hasReg := regengoResults["FindString"]; hasReg {
		speedup := "-"
		if std, hasStd := stdlibResults["FindStringSubmatch"]; hasStd && reg.NsOp > 0 {
			speedup = fmt.Sprintf("**%.1fx**", std.NsOp/reg.NsOp)
		}
		fmt.Printf("| regengo FindString | %.1f | %d | %d | %s |\n", reg.NsOp, reg.BOp, reg.Allocs, speedup)
	}

	// FindStringReuse (zero-alloc)
	if reg, hasReg := regengoResults["FindStringReuse"]; hasReg {
		speedup := "-"
		if std, hasStd := stdlibResults["FindStringSubmatch"]; hasStd && reg.NsOp > 0 {
			speedup = fmt.Sprintf("**%.1fx**", std.NsOp/reg.NsOp)
		}
		fmt.Printf("| regengo FindStringReuse | %.1f | %d | %d | %s |\n", reg.NsOp, reg.BOp, reg.Allocs, speedup)
	}

	// FindAllStringAppend (zero-alloc multi-match)
	if reg, hasReg := regengoResults["FindAllStringAppend"]; hasReg {
		fmt.Printf("| regengo FindAllStringAppend | %.1f | %d | %d | - |\n", reg.NsOp, reg.BOp, reg.Allocs)
	}

	// Replace benchmarks
	for _, method := range methodOrder {
		if strings.HasPrefix(method, "Replace") {
			if reg, hasReg := regengoResults[method]; hasReg {
				fmt.Printf("| regengo %s | %.1f | %d | %d | - |\n", method, reg.NsOp, reg.BOp, reg.Allocs)
			}
		}
	}

	// Any other methods not covered above
	printed := map[string]bool{
		"MatchString": true, "FindString": true, "FindStringSubmatch": true,
		"FindStringReuse": true, "FindAllStringAppend": true,
		"ReplaceAllString": true, "ReplaceAllStringN": true, "ReplaceAllBytesAppendN": true,
	}

	for method, reg := range regengoResults {
		if !printed[method] {
			fmt.Printf("| regengo %s | %.1f | %d | %d | - |\n", method, reg.NsOp, reg.BOp, reg.Allocs)
		}
	}

	fmt.Println()
}
