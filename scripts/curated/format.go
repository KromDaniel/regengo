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
	Name        string
	Pattern     string
	Category    string
	Description string
	Replacers   []string
	Results     []BenchResult
}

// PatternInfo holds pattern metadata
type PatternInfo struct {
	Pattern     string
	Category    string
	Description string
	Replacers   []string
}

// Pattern definitions from cases.go
var patternMap = map[string]PatternInfo{
	"Email":            {`[\w\.+-]+@[\w\.-]+\.[\w\.-]+`, "match", "Simple email matching without capture groups", nil},
	"Greedy":           {`(?:(?:a|b)|(?:k)+)*abcd`, "match", "Greedy quantifier with alternation", nil},
	"Lazy":             {`(?:(?:a|b)|(?:k)+)+?abcd`, "match", "Lazy quantifier with alternation", nil},
	"DateCapture":      {`(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})`, "capture", "ISO date pattern with year, month, day capture groups", nil},
	"EmailCapture":     {`(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)`, "capture", "Email pattern with named capture groups", nil},
	"URLCapture":       {`(?P<protocol>https?)://(?P<host>[\w\.-]+)(?::(?P<port>\d+))?(?P<path>/[\w\./]*)?`, "capture", "URL pattern with protocol, host, port, and path capture groups", nil},
	"MultiDate":        {`(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})`, "findall", "Find multiple dates in text", nil},
	"MultiEmail":       {`(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)`, "findall", "Find multiple email addresses in text", nil},
	"TDFAPathological": {`(?P<outer>(?P<inner>a+)+)b`, "tdfa", "Classic (a+)+b pattern - O(2^n) without TDFA, O(n) with TDFA", nil},
	"TDFANestedWord":   {`(?P<words>(?P<word>\w+\s*)+)end`, "tdfa", "Nested quantifiers with word boundaries", nil},
	"TDFAComplexURL":   {`(?P<scheme>https?)://(?P<auth>(?P<user>[\w.-]+)(?::(?P<pass>[\w.-]+))?@)?(?P<host>[\w.-]+)(?::(?P<port>\d+))?(?P<path>/[\w./-]*)?(?:\?(?P<query>[\w=&.-]+))?`, "tdfa", "Complex URL with optional components", nil},
	"TDFALogParser":    {`(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})(?:\.(?P<ms>\d{3}))?(?P<tz>Z|[+-]\d{2}:\d{2})?\s+\[(?P<level>\w+)\]\s+(?P<message>.+)`, "tdfa", "Log line parser with multiple optional groups", nil},
	"TDFASemVer":       {`(?P<major>\d+)\.(?P<minor>\d+)\.(?P<patch>\d+)(?:-(?P<prerelease>[\w.-]+))?(?:\+(?P<build>[\w.-]+))?`, "tdfa", "Semantic version with optional pre-release and build metadata", nil},
	"TNFAPathological": {`(?P<outer>(?P<inner>a+)+)b`, "tnfa", "Pathological pattern forced to use TNFA with memoization", nil},
	"ReplaceEmail":     {`(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)`, "replace", "Email replacement with capture group references", []string{"$user@REDACTED.$tld", "[EMAIL]", "$0"}},
	"ReplaceDate":      {`(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})`, "replace", "Date format conversion using capture groups", []string{"$month/$day/$year", "[DATE]", "$year"}},
	"ReplaceURL":       {`(?P<protocol>https?)://(?P<host>[\w\.-]+)(?::(?P<port>\d+))?(?P<path>/[\w\./]*)?`, "replace", "URL redaction with selective capture group output", []string{"$protocol://$host[REDACTED]", "[URL]", "$host"}},
}

// Category info
var categoryInfo = map[string]struct {
	Title       string
	Description string
}{
	"match":   {"Match Patterns (No Captures)", "Simple pattern matching without capture groups - uses DFA for O(n) performance."},
	"capture": {"Capture Patterns", "Patterns with named capture groups - uses TDFA or Thompson NFA."},
	"findall": {"FindAll Patterns", "Finding multiple matches in text."},
	"tdfa":    {"TDFA Patterns (Catastrophic Backtracking Prevention)", "These patterns have nested quantifiers + captures which would cause exponential backtracking without TDFA's O(n) guarantee."},
	"tnfa":    {"TNFA Patterns (Thompson NFA with Memoization)", "Patterns forced to use Thompson NFA with memoization for testing."},
	"replace": {"Replace Patterns", "String replacement with precompiled replacer templates."},
}

// Category order for output
var categoryOrder = []string{"match", "capture", "findall", "tdfa", "tnfa", "replace"}

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
			info := patternMap[r.PatternName]
			patternGroups[r.PatternName] = &PatternBenchmarks{
				Name:        r.PatternName,
				Pattern:     info.Pattern,
				Category:    info.Category,
				Description: info.Description,
				Replacers:   info.Replacers,
				Results:     []BenchResult{},
			}
		}
		patternGroups[r.PatternName].Results = append(patternGroups[r.PatternName].Results, r)
	}

	// Generate detailed markdown output organized by category
	generateDetailedMarkdown(patternGroups)
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

func generateDetailedMarkdown(groups map[string]*PatternBenchmarks) {
	fmt.Println("# Detailed Benchmarks")
	fmt.Println()
	fmt.Println("Benchmarks run on Apple M4 Pro. Each benchmark shows performance for Go stdlib vs regengo.")
	fmt.Println()

	// Print summary table
	fmt.Println("## Summary")
	fmt.Println()
	fmt.Println("| Pattern Type | Typical Speedup | Memory Reduction |")
	fmt.Println("|--------------|-----------------|------------------|")
	fmt.Println("| Simple match | 2-3x faster | 0 allocs |")
	fmt.Println("| Capture groups | 2-5x faster | 50% fewer allocs |")
	fmt.Println("| FindAll | 5-9x faster | 50-100% fewer allocs |")
	fmt.Println("| With reuse API | 10-15x faster | 0 allocs |")
	fmt.Println()

	// Group patterns by category
	categoryPatterns := make(map[string][]*PatternBenchmarks)
	for _, group := range groups {
		cat := group.Category
		if cat == "" {
			cat = "other"
		}
		categoryPatterns[cat] = append(categoryPatterns[cat], group)
	}

	// Sort patterns within each category
	for cat := range categoryPatterns {
		sort.Slice(categoryPatterns[cat], func(i, j int) bool {
			return categoryPatterns[cat][i].Name < categoryPatterns[cat][j].Name
		})
	}

	// Output by category
	for _, cat := range categoryOrder {
		patterns := categoryPatterns[cat]
		if len(patterns) == 0 {
			continue
		}

		info := categoryInfo[cat]
		fmt.Printf("## %s\n\n", info.Title)
		fmt.Printf("%s\n\n", info.Description)

		for _, group := range patterns {
			printPatternSection(group)
		}
	}

	// Running instructions
	fmt.Println("---")
	fmt.Println()
	fmt.Println("## Running Benchmarks")
	fmt.Println()
	fmt.Println("To run benchmarks yourself:")
	fmt.Println()
	fmt.Println("```bash")
	fmt.Println("# Run benchmarks (generates and runs curated benchmarks)")
	fmt.Println("make bench")
	fmt.Println()
	fmt.Println("# Analyze benchmark results with comparison summary")
	fmt.Println("make bench-analyze")
	fmt.Println()
	fmt.Println("# Generate markdown output")
	fmt.Println("make bench-format")
	fmt.Println()
	fmt.Println("# Generate performance chart")
	fmt.Println("make bench-chart")
	fmt.Println("```")
	fmt.Println()
	fmt.Println("## Regenerating Results")
	fmt.Println()
	fmt.Println("To regenerate these benchmark tables:")
	fmt.Println()
	fmt.Println("```bash")
	fmt.Println("make bench-format")
	fmt.Println("```")
}

func printPatternSection(group *PatternBenchmarks) {
	// Pattern heading
	fmt.Printf("### %s\n\n", group.Name)

	if group.Description != "" {
		fmt.Printf("%s\n\n", group.Description)
	}

	if group.Pattern != "" {
		fmt.Printf("**Pattern:**\n```regex\n%s\n```\n\n", group.Pattern)
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

	// Determine which comparison tables to show based on available results
	hasMatch := len(stdlibResults["MatchString"].FullName) > 0 || len(regengoResults["MatchString"].FullName) > 0
	hasFind := len(stdlibResults["FindStringSubmatch"].FullName) > 0 || len(regengoResults["FindString"].FullName) > 0
	hasReplace := false
	for method := range regengoResults {
		if strings.HasPrefix(method, "Replace") {
			hasReplace = true
			break
		}
	}

	// MatchString section
	if hasMatch {
		printMethodTable("MatchString", stdlibResults, regengoResults)
	}

	// FindString section
	if hasFind {
		printFindTable(stdlibResults, regengoResults)
	}

	// Replace section
	if hasReplace {
		printReplaceTable(stdlibResults, regengoResults, group.Replacers)
	}
}

func printMethodTable(method string, stdlibResults, regengoResults map[string]BenchResult) {
	fmt.Printf("**%s:**\n\n", method)
	fmt.Println("| Variant | ns/op | B/op | allocs/op | vs stdlib |")
	fmt.Println("|---------|------:|-----:|----------:|----------:|")

	if std, ok := stdlibResults[method]; ok {
		fmt.Printf("| stdlib | %.1f | %d | %d | - |\n", std.NsOp, std.BOp, std.Allocs)
	}
	if reg, ok := regengoResults[method]; ok {
		speedup := "-"
		if std, ok := stdlibResults[method]; ok && reg.NsOp > 0 {
			ratio := std.NsOp / reg.NsOp
			if ratio >= 1.0 {
				speedup = fmt.Sprintf("**%.1fx faster**", ratio)
			} else {
				speedup = fmt.Sprintf("**%.1fx slower**", 1/ratio)
			}
		}
		fmt.Printf("| regengo | %.1f | %d | %d | %s |\n", reg.NsOp, reg.BOp, reg.Allocs, speedup)
	}
	fmt.Println()
}

func printFindTable(stdlibResults, regengoResults map[string]BenchResult) {
	fmt.Println("**FindString:**")
	fmt.Println()
	fmt.Println("| Variant | ns/op | B/op | allocs/op | vs stdlib |")
	fmt.Println("|---------|------:|-----:|----------:|----------:|")

	std, hasStd := stdlibResults["FindStringSubmatch"]
	if hasStd {
		fmt.Printf("| stdlib | %.1f | %d | %d | - |\n", std.NsOp, std.BOp, std.Allocs)
	}

	if reg, ok := regengoResults["FindString"]; ok {
		speedup := "-"
		if hasStd && reg.NsOp > 0 {
			ratio := std.NsOp / reg.NsOp
			if ratio >= 1.0 {
				speedup = fmt.Sprintf("**%.1fx faster**", ratio)
			} else {
				speedup = fmt.Sprintf("**%.1fx slower**", 1/ratio)
			}
		}
		fmt.Printf("| regengo | %.1f | %d | %d | %s |\n", reg.NsOp, reg.BOp, reg.Allocs, speedup)
	}

	if reg, ok := regengoResults["FindStringReuse"]; ok {
		speedup := "-"
		if hasStd && reg.NsOp > 0 {
			ratio := std.NsOp / reg.NsOp
			if ratio >= 1.0 {
				speedup = fmt.Sprintf("**%.1fx faster**", ratio)
			} else {
				speedup = fmt.Sprintf("**%.1fx slower**", 1/ratio)
			}
		}
		fmt.Printf("| regengo (reuse) | %.1f | %d | %d | %s |\n", reg.NsOp, reg.BOp, reg.Allocs, speedup)
	}

	fmt.Println()

	// FindAllString section
	regFindAll, hasFindAll := regengoResults["FindAllString"]
	regFindAllAppend, hasFindAllAppend := regengoResults["FindAllStringAppend"]
	stdAll, hasStdAll := stdlibResults["FindAllStringSubmatch"]

	if hasFindAll || hasFindAllAppend {
		fmt.Println("**FindAllString:**")
		fmt.Println()
		fmt.Println("| Variant | ns/op | B/op | allocs/op | vs stdlib |")
		fmt.Println("|---------|------:|-----:|----------:|----------:|")

		if hasStdAll {
			fmt.Printf("| stdlib | %.1f | %d | %d | - |\n", stdAll.NsOp, stdAll.BOp, stdAll.Allocs)
		}

		if hasFindAll {
			speedup := "-"
			if hasStdAll && regFindAll.NsOp > 0 {
				ratio := stdAll.NsOp / regFindAll.NsOp
				if ratio >= 1.0 {
					speedup = fmt.Sprintf("**%.1fx faster**", ratio)
				} else {
					speedup = fmt.Sprintf("**%.1fx slower**", 1/ratio)
				}
			}
			fmt.Printf("| regengo | %.1f | %d | %d | %s |\n", regFindAll.NsOp, regFindAll.BOp, regFindAll.Allocs, speedup)
		}

		if hasFindAllAppend {
			speedup := "-"
			if hasStdAll && regFindAllAppend.NsOp > 0 {
				ratio := stdAll.NsOp / regFindAllAppend.NsOp
				if ratio >= 1.0 {
					speedup = fmt.Sprintf("**%.1fx faster**", ratio)
				} else {
					speedup = fmt.Sprintf("**%.1fx slower**", 1/ratio)
				}
			}
			fmt.Printf("| regengo (reuse) | %.1f | %d | %d | %s |\n", regFindAllAppend.NsOp, regFindAllAppend.BOp, regFindAllAppend.Allocs, speedup)
		}
		fmt.Println()
	}
}

func printReplaceTable(stdlibResults, regengoResults map[string]BenchResult, replacers []string) {
	// Print one table per template
	for i := 0; i < 10; i++ {
		stdMethod := fmt.Sprintf("ReplaceAllString%d", i)
		std, hasStd := stdlibResults[stdMethod]
		if !hasStd {
			continue // No more templates
		}

		template := fmt.Sprintf("#%d", i)
		if i < len(replacers) {
			template = fmt.Sprintf("`%s`", replacers[i])
		}

		fmt.Printf("**Replace %s:**\n\n", template)
		fmt.Println("| Variant | ns/op | B/op | allocs/op | vs stdlib |")
		fmt.Println("|---------|------:|-----:|----------:|----------:|")

		// Stdlib
		fmt.Printf("| stdlib | %.1f | %d | %d | - |\n", std.NsOp, std.BOp, std.Allocs)

		// Runtime replace
		runtimeMethod := fmt.Sprintf("ReplaceAllStringRuntime%d", i)
		if reg, ok := regengoResults[runtimeMethod]; ok {
			speedup := calcSpeedup(std.NsOp, reg.NsOp)
			fmt.Printf("| regengo | %.1f | %d | %d | %s |\n", reg.NsOp, reg.BOp, reg.Allocs, speedup)
		}

		// Precompiled
		precompiledMethod := fmt.Sprintf("ReplaceAllString%d", i)
		if reg, ok := regengoResults[precompiledMethod]; ok {
			speedup := calcSpeedup(std.NsOp, reg.NsOp)
			fmt.Printf("| regengo (precompiled) | %.1f | %d | %d | %s |\n", reg.NsOp, reg.BOp, reg.Allocs, speedup)
		}

		// Reuse (bytes append)
		reuseMethod := fmt.Sprintf("ReplaceAllBytesAppend%d", i)
		if reg, ok := regengoResults[reuseMethod]; ok {
			speedup := calcSpeedup(std.NsOp, reg.NsOp)
			fmt.Printf("| regengo (reuse) | %.1f | %d | %d | %s |\n", reg.NsOp, reg.BOp, reg.Allocs, speedup)
		}

		fmt.Println()
	}
}

func calcSpeedup(stdNsOp, regNsOp float64) string {
	if regNsOp <= 0 {
		return "-"
	}
	ratio := stdNsOp / regNsOp
	if ratio >= 1.0 {
		return fmt.Sprintf("**%.1fx faster**", ratio)
	}
	return fmt.Sprintf("**%.1fx slower**", 1/ratio)
}
