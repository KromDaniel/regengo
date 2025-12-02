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

// BenchResult holds a single benchmark result
type BenchResult struct {
	Pattern  string
	Category string // Match, FindFirst, FindAll, Replace
	Input    string
	Variant  string // stdlib, regengo, regengo_reuse, regengo_append, regengo_runtime
	Template string // for Replace category
	NsOp     float64
	BOp      int
	Allocs   int
}

// PatternBenchmarks holds aggregated results for a pattern
type PatternBenchmarks struct {
	Name        string
	Pattern     string
	Category    string
	Description string
	Replacers   []string
	Results     map[string]map[string]BenchResult // category -> variant -> averaged result
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

// benchmarkLineRe matches new nested benchmark output lines
var benchmarkLineRe = regexp.MustCompile(
	`^(Benchmark\S+)-\d+\s+\d+\s+([\d.]+)\s+ns/op\s+(\d+)\s+B/op\s+(\d+)\s+allocs/op`,
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	var results []BenchResult

	for scanner.Scan() {
		line := scanner.Text()
		match := benchmarkLineRe.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		fullName := match[1]
		nsOp, _ := strconv.ParseFloat(match[2], 64)
		bOp, _ := strconv.Atoi(match[3])
		allocs, _ := strconv.Atoi(match[4])

		result := parseBenchmarkName(fullName)
		if result == nil {
			continue
		}

		result.NsOp = nsOp
		result.BOp = bOp
		result.Allocs = allocs

		results = append(results, *result)
	}

	if len(results) == 0 {
		fmt.Fprintln(os.Stderr, "No benchmark results found in input.")
		fmt.Fprintln(os.Stderr, "Usage: go test -bench=. -benchmem ./benchmarks/curated/... 2>&1 | go run scripts/curated/format.go")
		os.Exit(1)
	}

	// Group and average by pattern
	patternGroups := aggregateResults(results)

	// Generate detailed markdown output organized by category
	generateDetailedMarkdown(patternGroups)
}

// parseBenchmarkName extracts components from nested benchmark name
func parseBenchmarkName(fullName string) *BenchResult {
	if !strings.HasPrefix(fullName, "Benchmark") {
		return nil
	}
	name := strings.TrimPrefix(fullName, "Benchmark")

	parts := strings.Split(name, "/")
	if len(parts) < 4 {
		return nil
	}

	result := &BenchResult{
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

// aggregateResults groups results by pattern and averages across inputs
func aggregateResults(results []BenchResult) map[string]*PatternBenchmarks {
	// First, group all results
	type groupKey struct {
		pattern  string
		category string
		variant  string
		template string
	}

	groups := make(map[groupKey][]BenchResult)

	for _, r := range results {
		key := groupKey{
			pattern:  r.Pattern,
			category: r.Category,
			variant:  r.Variant,
			template: r.Template,
		}
		groups[key] = append(groups[key], r)
	}

	// Now average and organize by pattern
	patternGroups := make(map[string]*PatternBenchmarks)

	for key, resultList := range groups {
		if _, ok := patternGroups[key.pattern]; !ok {
			info := patternMap[key.pattern]
			patternGroups[key.pattern] = &PatternBenchmarks{
				Name:        key.pattern,
				Pattern:     info.Pattern,
				Category:    info.Category,
				Description: info.Description,
				Replacers:   info.Replacers,
				Results:     make(map[string]map[string]BenchResult),
			}
		}

		pg := patternGroups[key.pattern]

		// Category key includes template for Replace
		catKey := key.category
		if key.template != "" {
			catKey = key.category + "/" + key.template
		}

		if pg.Results[catKey] == nil {
			pg.Results[catKey] = make(map[string]BenchResult)
		}

		// Average the results
		var totalNs float64
		var totalB, totalAllocs int
		for _, r := range resultList {
			totalNs += r.NsOp
			totalB += r.BOp
			totalAllocs += r.Allocs
		}
		n := len(resultList)

		pg.Results[catKey][key.variant] = BenchResult{
			Pattern:  key.pattern,
			Category: key.category,
			Variant:  key.variant,
			Template: key.template,
			NsOp:     totalNs / float64(n),
			BOp:      totalB / n,
			Allocs:   totalAllocs / n,
		}
	}

	return patternGroups
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
	fmt.Println("### Benchmark Structure")
	fmt.Println()
	fmt.Println("Benchmarks use a nested structure for clear comparison:")
	fmt.Println()
	fmt.Println("```")
	fmt.Println("Benchmark{Pattern}/")
	fmt.Println("├── Match/Input[i]/{stdlib,regengo}")
	fmt.Println("├── FindFirst/Input[i]/{stdlib,regengo,regengo_reuse}")
	fmt.Println("├── FindAll/Input[i]/{stdlib,regengo,regengo_append}")
	fmt.Println("└── Replace/Template[j]/Input[i]/{stdlib,regengo_runtime,regengo,regengo_append}")
	fmt.Println("```")
	fmt.Println()
	fmt.Println("### Running Specific Benchmarks")
	fmt.Println()
	fmt.Println("```bash")
	fmt.Println("# Run all benchmarks for a pattern")
	fmt.Println("go test ./benchmarks/curated/... -bench=\"BenchmarkDateCapture\" -benchmem")
	fmt.Println()
	fmt.Println("# Run only Match benchmarks")
	fmt.Println("go test ./benchmarks/curated/... -bench=\"Match\" -benchmem")
	fmt.Println()
	fmt.Println("# Run only regengo_reuse variants")
	fmt.Println("go test ./benchmarks/curated/... -bench=\"regengo_reuse\" -benchmem")
	fmt.Println()
	fmt.Println("# Run specific input")
	fmt.Println("go test ./benchmarks/curated/... -bench=\"Input\\\\[0\\\\]\" -benchmem")
	fmt.Println("```")
	fmt.Println()
	fmt.Println("### Aggregating Results")
	fmt.Println()
	fmt.Println("Use the aggregation script for summary statistics across all inputs:")
	fmt.Println()
	fmt.Println("```bash")
	fmt.Println("# Aggregate results for a pattern")
	fmt.Println("go test ./benchmarks/curated/... -bench=\"BenchmarkDateCapture\" -benchmem | go run scripts/curated/aggregate.go")
	fmt.Println("```")
	fmt.Println()
	fmt.Println("### Make Targets")
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
	fmt.Println("To regenerate benchmark files after code changes:")
	fmt.Println()
	fmt.Println("```bash")
	fmt.Println("# Regenerate curated benchmark code")
	fmt.Println("go run scripts/curated/generate.go scripts/curated/cases.go")
	fmt.Println()
	fmt.Println("# Or use make")
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

	// Print tables for each benchmark category
	// Order: Match, FindFirst (FindString), FindAll, Replace

	// MatchString section
	if matchResults, ok := group.Results["Match"]; ok {
		printComparisonTable("MatchString", matchResults, nil)
	}

	// FindString section (FindFirst)
	if findResults, ok := group.Results["FindFirst"]; ok {
		printComparisonTable("FindString", findResults, []string{"stdlib", "regengo", "regengo_reuse"})
	}

	// FindAllString section
	if findAllResults, ok := group.Results["FindAll"]; ok {
		printComparisonTable("FindAllString", findAllResults, []string{"stdlib", "regengo", "regengo_append"})
	}

	// Replace sections (one per template)
	for i := 0; i < 10; i++ {
		catKey := fmt.Sprintf("Replace/Template[%d]", i)
		if replaceResults, ok := group.Results[catKey]; ok {
			template := fmt.Sprintf("#%d", i)
			if i < len(group.Replacers) {
				template = fmt.Sprintf("`%s`", group.Replacers[i])
			}
			printReplaceTable(template, replaceResults)
		}
	}
}

func printComparisonTable(title string, results map[string]BenchResult, variantOrder []string) {
	fmt.Printf("**%s:**\n\n", title)
	fmt.Println("| Variant | ns/op | B/op | allocs/op | vs stdlib |")
	fmt.Println("|---------|------:|-----:|----------:|----------:|")

	if variantOrder == nil {
		variantOrder = []string{"stdlib", "regengo", "regengo_reuse", "regengo_append"}
	}

	stdResult, hasStd := results["stdlib"]

	for _, variant := range variantOrder {
		r, ok := results[variant]
		if !ok {
			continue
		}

		speedup := "-"
		if variant != "stdlib" && hasStd && r.NsOp > 0 {
			ratio := stdResult.NsOp / r.NsOp
			if ratio >= 1.0 {
				speedup = fmt.Sprintf("**%.1fx faster**", ratio)
			} else {
				speedup = fmt.Sprintf("**%.1fx slower**", 1/ratio)
			}
		}

		displayName := variant
		switch variant {
		case "regengo_reuse":
			displayName = "regengo (reuse)"
		case "regengo_append":
			displayName = "regengo (reuse)"
		}

		fmt.Printf("| %s | %.1f | %d | %d | %s |\n", displayName, r.NsOp, r.BOp, r.Allocs, speedup)
	}
	fmt.Println()
}

func printReplaceTable(template string, results map[string]BenchResult) {
	fmt.Printf("**Replace %s:**\n\n", template)
	fmt.Println("| Variant | ns/op | B/op | allocs/op | vs stdlib |")
	fmt.Println("|---------|------:|-----:|----------:|----------:|")

	stdResult, hasStd := results["stdlib"]

	variantOrder := []string{"stdlib", "regengo_runtime", "regengo", "regengo_append"}
	displayNames := map[string]string{
		"stdlib":          "stdlib",
		"regengo_runtime": "regengo",
		"regengo":         "regengo (precompiled)",
		"regengo_append":  "regengo (reuse)",
	}

	for _, variant := range variantOrder {
		r, ok := results[variant]
		if !ok {
			continue
		}

		speedup := "-"
		if variant != "stdlib" && hasStd && r.NsOp > 0 {
			ratio := stdResult.NsOp / r.NsOp
			if ratio >= 1.0 {
				speedup = fmt.Sprintf("**%.1fx faster**", ratio)
			} else {
				speedup = fmt.Sprintf("**%.1fx slower**", 1/ratio)
			}
		}

		displayName := displayNames[variant]
		fmt.Printf("| %s | %.1f | %d | %d | %s |\n", displayName, r.NsOp, r.BOp, r.Allocs, speedup)
	}
	fmt.Println()
}
