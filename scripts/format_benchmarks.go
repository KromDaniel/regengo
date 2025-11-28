//go:build ignore

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
	Name     string
	Type     string // "golang_std", "regengo", "regengo_reuse"
	InputIdx int
	NsOp     float64
	BOp      int
	Allocs   int
}

type BenchGroup struct {
	Name    string
	Pattern string
	Results map[string][]BenchResult // Type -> results per input
}

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
}

func main() {
	// Parse benchmark output from stdin
	scanner := bufio.NewScanner(os.Stdin)

	// Regex to parse benchmark lines
	benchRegex := regexp.MustCompile(`^Benchmark(\w+)/(\w+)_(\d+)-\d+\s+\d+\s+([\d.]+)\s+ns/op\s+(\d+)\s+B/op\s+(\d+)\s+allocs/op`)

	groups := make(map[string]*BenchGroup)

	for scanner.Scan() {
		line := scanner.Text()
		matches := benchRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		benchName := matches[1]
		benchType := matches[2]
		inputIdx, _ := strconv.Atoi(matches[3])
		nsOp, _ := strconv.ParseFloat(matches[4], 64)
		bOp, _ := strconv.Atoi(matches[5])
		allocs, _ := strconv.Atoi(matches[6])

		result := BenchResult{
			Name:     benchName,
			Type:     benchType,
			InputIdx: inputIdx,
			NsOp:     nsOp,
			BOp:      bOp,
			Allocs:   allocs,
		}

		if _, ok := groups[benchName]; !ok {
			// Extract pattern name from benchmark name
			patternName := benchName
			for suffix := range map[string]bool{"MatchString": true, "FindString": true, "FindAllString": true} {
				patternName = strings.TrimSuffix(patternName, suffix)
			}

			groups[benchName] = &BenchGroup{
				Name:    benchName,
				Pattern: patternMap[patternName],
				Results: make(map[string][]BenchResult),
			}
		}

		groups[benchName].Results[benchType] = append(groups[benchName].Results[benchType], result)
	}

	// Sort group names
	var groupNames []string
	for name := range groups {
		groupNames = append(groupNames, name)
	}
	sort.Strings(groupNames)

	// Generate markdown
	fmt.Println("\n## Detailed Benchmarks")
	fmt.Println()
	fmt.Println("Benchmarks run on Apple M4 Pro. Each benchmark shows performance for Go stdlib vs regengo.")
	fmt.Println()

	for _, groupName := range groupNames {
		group := groups[groupName]

		// Get method type from name
		methodType := "MatchString"
		if strings.HasSuffix(groupName, "FindString") {
			methodType = "FindString"
		} else if strings.HasSuffix(groupName, "FindAllString") {
			methodType = "FindAllString"
		}

		fmt.Printf("### %s\n\n", groupName)
		fmt.Printf("**Pattern:**\n```regex\n%s\n```\n\n", group.Pattern)
		fmt.Printf("**Method:** `%s`\n\n", methodType)

		// Check if we have reuse benchmarks
		hasReuse := len(group.Results["regengo_reuse"]) > 0

		if hasReuse {
			fmt.Println("| Variant | ns/op | B/op | allocs/op | vs stdlib |")
			fmt.Println("|---------|------:|-----:|----------:|----------:|")
		} else {
			fmt.Println("| Variant | ns/op | B/op | allocs/op | vs stdlib |")
			fmt.Println("|---------|------:|-----:|----------:|----------:|")
		}

		// Calculate averages
		stdResults := group.Results["golang_std"]
		regengoResults := group.Results["regengo"]
		reuseResults := group.Results["regengo_reuse"]

		avgStd := avgNsOp(stdResults)
		avgRegengo := avgNsOp(regengoResults)
		avgReuse := avgNsOp(reuseResults)

		avgStdB := avgBOp(stdResults)
		avgRegengoB := avgBOp(regengoResults)
		avgReuseB := avgBOp(reuseResults)

		avgStdAllocs := avgAllocs(stdResults)
		avgRegengoAllocs := avgAllocs(regengoResults)
		avgReuseAllocs := avgAllocs(reuseResults)

		// Print stdlib row
		fmt.Printf("| stdlib | %.1f | %d | %d | - |\n", avgStd, avgStdB, avgStdAllocs)

		// Print regengo row with speedup
		speedup := avgStd / avgRegengo
		fmt.Printf("| regengo | %.1f | %d | %d | **%.1fx faster** |\n", avgRegengo, avgRegengoB, avgRegengoAllocs, speedup)

		// Print reuse row if available
		if hasReuse {
			speedupReuse := avgStd / avgReuse
			fmt.Printf("| regengo (reuse) | %.1f | %d | %d | **%.1fx faster** |\n", avgReuse, avgReuseB, avgReuseAllocs, speedupReuse)
		}

		fmt.Println()
	}
}

func avgNsOp(results []BenchResult) float64 {
	if len(results) == 0 {
		return 0
	}
	var sum float64
	for _, r := range results {
		sum += r.NsOp
	}
	return sum / float64(len(results))
}

func avgBOp(results []BenchResult) int {
	if len(results) == 0 {
		return 0
	}
	var sum int
	for _, r := range results {
		sum += r.BOp
	}
	return sum / len(results)
}

func avgAllocs(results []BenchResult) int {
	if len(results) == 0 {
		return 0
	}
	var sum int
	for _, r := range results {
		sum += r.Allocs
	}
	return sum / len(results)
}
