package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/KromDaniel/regengo/pkg/regengo"
)

type patternCategory string

const (
	categorySimple      patternCategory = "simple"
	categoryComplex     patternCategory = "complex"
	categoryVeryComplex patternCategory = "very_complex"
	categoryTDFA        patternCategory = "tdfa" // Tagged DFA patterns with catastrophic backtracking risk
)

type patternSpec struct {
	Category patternCategory
	Name     string
	Pattern  string
	Inputs   []string
}

type categoryStats struct {
	Patterns  int
	TestCases int
}

type commandResult struct {
	Command  string
	Output   string
	Duration time.Duration
	Err      error
}

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
	SlowerPatterns   []slowPattern // Patterns where regengo is slower
}

type slowPattern struct {
	Name        string
	Pattern     string
	RegengoNs   float64
	StdlibNs    float64
	SlowerByPct float64
}

var (
	command   = flag.String("command", "", "Command to run: generate, benchmark, delete")
	outputDir = flag.String("output-dir", "", "Output directory for generated tests (default: benchmarks/mass_generated)")
	helpFlag  = flag.Bool("help", false, "Show help message")
	version   = flag.Bool("version", false, "Print version information")
)

const (
	appVersion = "1.0.0"
	appName    = "mass_generator"
)

func main() {
	flag.Parse()

	if *helpFlag {
		printHelp()
		return
	}

	if *version {
		fmt.Printf("%s version %s\n", appName, appVersion)
		return
	}

	if *command == "" {
		fmt.Fprintf(os.Stderr, "Error: -command flag is required\n\n")
		printHelp()
		os.Exit(1)
	}

	workingDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to resolve working directory: %v\n", err)
		os.Exit(1)
	}

	// Set default output directory if not provided
	var targetDir string
	if *outputDir == "" {
		targetDir = filepath.Join(workingDir, "benchmarks", "mass_generated")
	} else {
		targetDir = *outputDir
	}

	switch *command {
	case "generate":
		if err := generateTests(targetDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating tests: %v\n", err)
			os.Exit(1)
		}
	case "benchmark":
		if err := runBenchmarks(targetDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error running benchmarks: %v\n", err)
			os.Exit(1)
		}
	case "delete":
		if err := deleteGeneratedTests(targetDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting tests: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command '%s'. Use 'generate', 'benchmark', or 'delete'\n", *command)
		os.Exit(1)
	}
}

func buildPatternSpecs() []patternSpec {
	specs := make([]patternSpec, 0, 90+45+20+50)
	specs = append(specs, generateSimpleSpecs(90)...)
	specs = append(specs, generateComplexSpecs(45)...)
	specs = append(specs, generateVeryComplexSpecs(20)...)
	specs = append(specs, generateTDFASpecs(50)...)
	return specs
}

func generateSimpleSpecs(count int) []patternSpec {
	specs := make([]patternSpec, 0, count)
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("SimpleCase%03d", i+1)
		length := 3 + (i % 5)

		var pattern string
		var rawInputs []string

		switch i % 3 {
		case 0:
			pattern = fmt.Sprintf("^[a-z]{%d}$", length)
			rawInputs = []string{
				strings.Repeat("a", length),
				strings.Repeat("bc", length/2+1)[:length],
				strings.Repeat("a", length-1) + "1",
				strings.Repeat("A", length),
				strings.Repeat("a", length+1),
			}
		case 1:
			pattern = fmt.Sprintf("^\\d{%d}$", length)
			rawInputs = []string{
				strings.Repeat(strconv.Itoa((i%9)+1), length),
				strings.Repeat("0", length),
				strings.Repeat("1", length-1) + "a",
				strings.Repeat("2", length+1),
				strings.Repeat("3", length-2) + " ",
			}
		default:
			pattern = fmt.Sprintf("^[a-f0-9]{%d}$", length)
			rawInputs = []string{
				strings.Repeat("a", length),
				strings.Repeat("0f", length/2+1)[:length],
				strings.Repeat("g", length),
				strings.Repeat("f", length+1),
				strings.Repeat("A", length),
			}
		}

		specs = append(specs, patternSpec{
			Category: categorySimple,
			Name:     name,
			Pattern:  pattern,
			Inputs:   normalizeInputs(rawInputs, 5),
		})
	}
	return specs
}

func generateComplexSpecs(count int) []patternSpec {
	specs := make([]patternSpec, 0, count)
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("ComplexCase%03d", i+1)

		var pattern string
		var rawInputs []string

		switch i % 3 {
		case 0:
			repeat := 2 + (i % 3)
			digits := 2 + (i % 4)
			pattern = fmt.Sprintf("^(?:foo|bar){%d}baz\\d{%d}$", repeat, digits)
			rawInputs = []string{
				strings.Repeat("foo", repeat) + "baz" + strings.Repeat("7", digits),
				strings.Repeat("bar", repeat-1) + "foo" + "baz" + strings.Repeat("3", digits),
				strings.Repeat("foo", repeat) + "baz" + strings.Repeat("9", digits-1),
				strings.Repeat("foo", repeat) + "qux" + strings.Repeat("7", digits),
				strings.Repeat("foo", repeat) + "baz" + strings.Repeat("7", digits) + "extra",
			}
		case 1:
			extDigits := 2 + (i % 3)
			pattern = fmt.Sprintf("^(?P<area>\\d{3})-(?P<prefix>\\d{3})-(?P<line>\\d{4})(?: x(?P<ext>\\d{%d}))?$", extDigits)
			rawInputs = []string{
				fmt.Sprintf("%03d-%03d-%04d", 200+(i%300), 100+(i%500), 1000+(i%700)),
				fmt.Sprintf("555-123-9876 x%s", strings.Repeat("8", extDigits)),
				fmt.Sprintf("%03d-%03d-%03d", 100+(i%200), 200+(i%200), 300+(i%200)),
				"12-3456-7890",
				fmt.Sprintf("%03d-%03d-%04d ext%s", 400+(i%200), 500+(i%200), 6000+(i%200), strings.Repeat("5", extDigits)),
			}
		case 2:
			words := 2 + (i % 4)
			pattern = fmt.Sprintf("^(?:[A-Z][a-z]+\\s){%d}[A-Z][a-z]+$", words-1)
			names := capitalizedNamePool()
			start := (i * 3) % (len(names) - words)
			ordered := make([]string, words)
			copy(ordered, names[start:start+words])
			reversed := reverseStrings(ordered)
			match1 := strings.Join(ordered, " ")
			match2 := strings.Join(reversed, " ")
			rawInputs = []string{
				match1,
				match2,
				strings.ToLower(match1),
				match1 + " ",
				match1 + "!",
			}
		}

		specs = append(specs, patternSpec{
			Category: categoryComplex,
			Name:     name,
			Pattern:  pattern,
			Inputs:   normalizeInputs(rawInputs, 5),
		})
	}
	return specs
}

func generateVeryComplexSpecs(count int) []patternSpec {
	templates := []func(int, string) patternSpec{
		buildURLSpec,
		buildTimestampSpec,
		buildKeyValueSpec,
		buildAPIPathSpec,
	}

	specs := make([]patternSpec, 0, count)
	for i := 0; i < count; i++ {
		template := templates[i%len(templates)]
		name := fmt.Sprintf("VeryComplexCase%03d", i+1)
		specs = append(specs, template(i, name))
	}

	return specs
}

func buildURLSpec(idx int, name string) patternSpec {
	hostSegments := 1 + (idx % 3)
	pathSegments := 2 + (idx % 4)
	queryMin := 4 + (idx % 4)
	queryMax := queryMin + 4

	pattern := fmt.Sprintf(
		"^(?P<protocol>https?)://(?P<host>(?:[a-z0-9-]+\\.){%d}[a-z]{2,})(?P<path>(?:/[a-z0-9._-]{2,}){%d})(?:\\?(?P<query>[a-z0-9=&_-]{%d,%d}))?$",
		hostSegments,
		pathSegments,
		queryMin,
		queryMax,
	)

	hostParts := make([]string, 0, hostSegments+1)
	for h := 0; h < hostSegments; h++ {
		hostParts = append(hostParts, fmt.Sprintf("sub%d", (idx+h)%20))
	}
	hostParts = append(hostParts, "com")
	host := strings.Join(hostParts, ".")

	var pathBuilder strings.Builder
	segmentNames := []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta", "theta", "iota", "kappa"}
	for p := 0; p < pathSegments; p++ {
		pathBuilder.WriteString("/")
		pathBuilder.WriteString(segmentNames[(idx+p)%len(segmentNames)])
	}
	path := pathBuilder.String()

	query := fmt.Sprintf("id=%d&ref=%d", idx%97, (idx+13)%97)
	for len(query) < queryMin {
		query += "a"
	}
	if len(query) > queryMax {
		query = query[:queryMax]
	}

	match1 := fmt.Sprintf("https://%s%s?%s", host, path, query)
	match2 := fmt.Sprintf("http://%s%s", host, path)
	missingProtocol := fmt.Sprintf("%s%s", host, path)
	badHost := fmt.Sprintf("https://%s%s", strings.ReplaceAll(host, ".", "_"), path)
	shortQuery := fmt.Sprintf("https://%s%s?x", host, path)
	extraSlash := fmt.Sprintf("https://%s%s//", host, path)

	inputs := []string{match1, match2, missingProtocol, badHost, shortQuery, extraSlash}

	return patternSpec{
		Category: categoryVeryComplex,
		Name:     name,
		Pattern:  pattern,
		Inputs:   normalizeInputs(inputs, 6),
	}
}

func buildTimestampSpec(idx int, name string) patternSpec {
	fractionDigits := 1 + (idx % 3)
	pattern := fmt.Sprintf(
		"^(?P<year>\\d{4})-(?P<month>0[1-9]|1[0-2])-(?P<day>0[1-9]|[12]\\d|3[01])T(?P<hour>[01]\\d|2[0-3]):(?P<minute>[0-5]\\d):(?P<second>[0-5]\\d)(?:\\.(?P<fraction>\\d{%d}))?(?:Z|(?P<zone>[+-][01]\\d:[0-5]\\d))$",
		fractionDigits,
	)

	match1 := "2024-11-23T16:45:30Z"
	match2 := fmt.Sprintf("2023-01-02T03:04:05.%s+02:00", strings.Repeat("7", fractionDigits))
	badMonth := "2024-13-01T00:00:00Z"
	badHour := "2024-11-23T24:00:00Z"
	missingT := "2024-11-23 16:45:30Z"
	badFraction := "2024-11-23T16:45:30.123456Z"

	inputs := []string{match1, match2, badMonth, badHour, missingT, badFraction}

	return patternSpec{
		Category: categoryVeryComplex,
		Name:     name,
		Pattern:  pattern,
		Inputs:   normalizeInputs(inputs, 6),
	}
}

func buildKeyValueSpec(idx int, name string) patternSpec {
	pairCount := 2 + (idx % 3)
	pattern := fmt.Sprintf("^(?:[a-z]{3}=[0-9]{2}&){%d}[a-z]{3}=[0-9]{2}$", pairCount-1)

	keys := []string{"aaa", "bbb", "ccc", "ddd", "eee", "fff", "ggg", "hhh", "iii", "jjj", "kkk", "lll"}
	pairs := make([]string, 0, pairCount)
	for j := 0; j < pairCount; j++ {
		key := keys[(idx+j)%len(keys)]
		value := fmt.Sprintf("%02d", (idx*3+j)%90)
		pairs = append(pairs, key+"="+value)
	}

	match1 := strings.Join(pairs, "&")
	match2 := strings.Join(reverseStrings(pairs), "&")
	trailingAmp := match1 + "&"
	replaced := strings.Replace(match1, "=", ":", 1)
	upper := strings.ToUpper(match1)
	doubleAmp := strings.Replace(match1, "&", "&&", 1)

	inputs := []string{match1, match2, trailingAmp, replaced, upper, doubleAmp}

	return patternSpec{
		Category: categoryVeryComplex,
		Name:     name,
		Pattern:  pattern,
		Inputs:   normalizeInputs(inputs, 6),
	}
}

func buildAPIPathSpec(idx int, name string) patternSpec {
	segmentCount := 3 + (idx % 3)
	optionalPairs := idx % 3
	pattern := fmt.Sprintf(
		"^/api/v1(?:/[a-z]{3,8}){%d}/(?P<id>[1-9]\\d{3,5})(?:\\?(?P<params>[a-z]+=[0-9]{2}(?:&[a-z]+=[0-9]{2}){0,%d}))?$",
		segmentCount,
		optionalPairs,
	)

	segments := []string{"users", "orders", "reports", "metrics", "devices", "profiles", "settings", "logs", "audits", "events", "alerts", "widgets"}
	var builder strings.Builder
	builder.WriteString("/api/v1")
	for s := 0; s < segmentCount; s++ {
		builder.WriteString("/")
		builder.WriteString(segments[(idx+s)%len(segments)])
	}
	path := builder.String()
	identifier := fmt.Sprintf("%d", 1000+idx*7)

	params := []string{
		fmt.Sprintf("page=%02d", idx%50),
		fmt.Sprintf("size=%02d", (idx+5)%50),
		fmt.Sprintf("tag=%02d", (idx+11)%50),
	}
	pairCount := 1 + optionalPairs
	queryParts := make([]string, 0, pairCount)
	for q := 0; q < pairCount; q++ {
		queryParts = append(queryParts, params[(idx+q)%len(params)])
	}
	query := strings.Join(queryParts, "&")

	match1 := fmt.Sprintf("%s/%s?%s", path, identifier, query)
	match2 := fmt.Sprintf("%s/%s", path, identifier)
	upperAPI := strings.Replace(match1, "/api", "/API", 1)
	badID := strings.Replace(match1, identifier, "ID"+identifier, 1)
	missingID := fmt.Sprintf("%s?%s", path, query)
	tooManyParams := match1 + "&extra=99"

	inputs := []string{match1, match2, upperAPI, badID, missingID, tooManyParams}

	return patternSpec{
		Category: categoryVeryComplex,
		Name:     name,
		Pattern:  pattern,
		Inputs:   normalizeInputs(inputs, 6),
	}
}

// generateTDFASpecs generates patterns that trigger Tagged DFA due to catastrophic backtracking risk.
// These patterns have nested quantifiers + captures which would cause exponential backtracking without TDFA.
func generateTDFASpecs(count int) []patternSpec {
	specs := make([]patternSpec, 0, count)

	templates := []func(int, string) patternSpec{
		buildPathologicalSpec,
		buildNestedWordSpec,
		buildComplexURLSpec,
		buildLogParserSpec,
		buildSemVerSpec,
		buildEmailCaptureSpec,
		buildIPv4CaptureSpec,
		buildUUIDCaptureSpec,
		buildCreditCardSpec,
		buildHTMLTagSpec,
	}

	for i := 0; i < count; i++ {
		template := templates[i%len(templates)]
		name := fmt.Sprintf("TDFACase%03d", i+1)
		specs = append(specs, template(i, name))
	}

	return specs
}

// buildPathologicalSpec creates patterns with nested quantifiers that cause catastrophic backtracking
func buildPathologicalSpec(idx int, name string) patternSpec {
	// Vary the nested quantifier pattern
	repeatChar := string(rune('a' + (idx % 5)))
	innerRepeat := 1 + (idx % 3)
	outerRepeat := 1 + ((idx / 3) % 3)

	var pattern string
	switch idx % 5 {
	case 0:
		// Classic (a+)+b pattern
		pattern = fmt.Sprintf(`(?P<outer>(?P<inner>%s+)+)b`, repeatChar)
	case 1:
		// Nested alternation with quantifier
		pattern = fmt.Sprintf(`(?P<match>(?:%s|%s%s)+)end`, repeatChar, repeatChar, repeatChar)
	case 2:
		// Multiple nested groups
		pattern = fmt.Sprintf(`(?P<g1>(?P<g2>%s{%d})+)(?P<g3>b+)`, repeatChar, innerRepeat)
	case 3:
		// Alternation with nested quantifier
		pattern = fmt.Sprintf(`(?P<result>(?:x+|y+|%s+){%d})z`, repeatChar, outerRepeat)
	default:
		// Complex nesting
		pattern = fmt.Sprintf(`(?P<full>(?P<part>(?:%s+)+){%d})end`, repeatChar, innerRepeat)
	}

	repeatCount := 10 + (idx % 20)
	inputs := []string{
		strings.Repeat(repeatChar, repeatCount) + "b",
		strings.Repeat(repeatChar, repeatCount+10) + "end",
		strings.Repeat(repeatChar, repeatCount) + strings.Repeat("b", 5),
		strings.Repeat(repeatChar, repeatCount) + "z",
		strings.Repeat(repeatChar, repeatCount/2), // non-match
	}

	return patternSpec{
		Category: categoryTDFA,
		Name:     name,
		Pattern:  pattern,
		Inputs:   normalizeInputs(inputs, 5),
	}
}

// buildNestedWordSpec creates patterns with nested word quantifiers
func buildNestedWordSpec(idx int, name string) patternSpec {
	wordCount := 2 + (idx % 4)
	spacePattern := `\s+`
	if idx%2 == 0 {
		spacePattern = `\s*`
	}

	pattern := fmt.Sprintf(`(?P<words>(?P<word>\w+%s){%d})(?P<end>\w+)`, spacePattern, wordCount)

	words := []string{"hello", "world", "foo", "bar", "test", "data", "alpha", "beta"}
	var builder strings.Builder
	for i := 0; i <= wordCount; i++ {
		if i > 0 {
			builder.WriteString(" ")
		}
		builder.WriteString(words[(idx+i)%len(words)])
	}
	match1 := builder.String()

	inputs := []string{
		match1,
		strings.Repeat("word ", wordCount) + "end",
		"a b c d e f g h i j k end",
		strings.Repeat("x ", wordCount*2) + "y",
		"no match here",
	}

	return patternSpec{
		Category: categoryTDFA,
		Name:     name,
		Pattern:  pattern,
		Inputs:   normalizeInputs(inputs, 5),
	}
}

// buildComplexURLSpec creates complex URL patterns with many optional groups
func buildComplexURLSpec(idx int, name string) patternSpec {
	// Vary the complexity
	var pattern string
	switch idx % 4 {
	case 0:
		pattern = `(?P<scheme>https?)://(?P<auth>(?P<user>[\w.-]+)(?::(?P<pass>[\w.-]+))?@)?(?P<host>[\w.-]+)(?::(?P<port>\d+))?(?P<path>/[\w./-]*)?`
	case 1:
		pattern = `(?P<protocol>https?)://(?P<host>[\w.-]+)(?::(?P<port>\d{1,5}))?(?P<path>(?:/[\w.-]+)*)(?:\?(?P<query>[\w=&.-]+))?(?:#(?P<fragment>[\w.-]+))?`
	case 2:
		pattern = `(?P<url>(?P<scheme>https?)://(?P<domain>(?:[\w-]+\.)+[\w-]+)(?P<port>:\d+)?(?P<path>/[^\s?#]*)?(?P<query>\?[^\s#]*)?)`
	default:
		pattern = `(?P<full>(?P<proto>https?|ftp)://(?P<host>[\w.-]+)(?P<port>:\d+)?(?P<path>/[\w./-]*)?)(?P<extra>.*)?`
	}

	inputs := []string{
		"https://example.com",
		"http://user:pass@api.example.com:8080/path/to/resource?key=value",
		fmt.Sprintf("https://sub%d.domain.com/api/v%d/users", idx%10, idx%5+1),
		"http://localhost:3000/test",
		"not a url",
	}

	return patternSpec{
		Category: categoryTDFA,
		Name:     name,
		Pattern:  pattern,
		Inputs:   normalizeInputs(inputs, 5),
	}
}

// buildLogParserSpec creates log line parsing patterns
func buildLogParserSpec(idx int, name string) patternSpec {
	var pattern string
	switch idx % 4 {
	case 0:
		pattern = `(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})(?:\.(?P<ms>\d{3}))?(?P<tz>Z|[+-]\d{2}:\d{2})?\s+\[(?P<level>\w+)\]\s+(?P<message>.+)`
	case 1:
		pattern = `(?P<date>\d{4}/\d{2}/\d{2})\s+(?P<time>\d{2}:\d{2}:\d{2})\s+(?P<level>DEBUG|INFO|WARN|ERROR)\s+(?P<component>[\w.]+):\s+(?P<msg>.*)`
	case 2:
		pattern = `\[(?P<level>\w+)\]\s+(?P<timestamp>\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2})\s+-\s+(?P<logger>[\w.]+)\s+-\s+(?P<message>.*)`
	default:
		pattern = `(?P<ip>\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})\s+-\s+-\s+\[(?P<date>[^\]]+)\]\s+"(?P<method>\w+)\s+(?P<path>[^\s]+)\s+(?P<proto>[^"]+)"\s+(?P<status>\d+)\s+(?P<size>\d+)`
	}

	inputs := []string{
		"2024-01-15T10:30:45.123Z [INFO] Server started successfully",
		"2024/01/15 10:30:45 ERROR app.server: Connection failed",
		"[WARN] 2024-01-15 10:30:45 - com.app.Service - Timeout occurred",
		`192.168.1.1 - - [15/Jan/2024:10:30:45 +0000] "GET /api/users HTTP/1.1" 200 1234`,
		"invalid log line",
	}

	return patternSpec{
		Category: categoryTDFA,
		Name:     name,
		Pattern:  pattern,
		Inputs:   normalizeInputs(inputs, 5),
	}
}

// buildSemVerSpec creates semantic version patterns
func buildSemVerSpec(idx int, name string) patternSpec {
	var pattern string
	switch idx % 3 {
	case 0:
		pattern = `(?P<major>\d+)\.(?P<minor>\d+)\.(?P<patch>\d+)(?:-(?P<prerelease>[\w.-]+))?(?:\+(?P<build>[\w.-]+))?`
	case 1:
		pattern = `v?(?P<version>(?P<major>\d+)\.(?P<minor>\d+)(?:\.(?P<patch>\d+))?)(?P<suffix>-[\w.]+)?`
	default:
		pattern = `(?P<full>(?P<major>\d+)\.(?P<minor>\d+)\.(?P<patch>\d+))(?P<pre>-(?:alpha|beta|rc)\.?\d*)?(?P<meta>\+[\w.]+)?`
	}

	inputs := []string{
		"1.0.0",
		"2.1.3-alpha.1",
		"3.0.0-beta.2+build.123",
		fmt.Sprintf("%d.%d.%d-rc.%d+20240115", idx%10, idx%20, idx%100, idx%5),
		"v10.20.30",
	}

	return patternSpec{
		Category: categoryTDFA,
		Name:     name,
		Pattern:  pattern,
		Inputs:   normalizeInputs(inputs, 5),
	}
}

// buildEmailCaptureSpec creates email patterns with captures
func buildEmailCaptureSpec(idx int, name string) patternSpec {
	var pattern string
	switch idx % 3 {
	case 0:
		pattern = `(?P<email>(?P<local>[\w.+-]+)@(?P<domain>[\w.-]+)\.(?P<tld>[a-z]{2,}))`
	case 1:
		pattern = `(?P<user>[\w.%+-]+)@(?P<subdomain>(?:[\w-]+\.)*)?(?P<domain>[\w-]+)\.(?P<tld>[a-z]{2,6})`
	default:
		pattern = `(?P<full>(?P<name>[\w.+-]+)@(?P<host>[\w.-]+))(?P<extra>\s.*)?`
	}

	inputs := []string{
		"user@example.com",
		"john.doe+tag@subdomain.example.co.uk",
		fmt.Sprintf("test%d@domain%d.org", idx%100, idx%50),
		"complex.email+filter@sub.domain.example.com",
		"invalid email",
	}

	return patternSpec{
		Category: categoryTDFA,
		Name:     name,
		Pattern:  pattern,
		Inputs:   normalizeInputs(inputs, 5),
	}
}

// buildIPv4CaptureSpec creates IPv4 patterns with captures
func buildIPv4CaptureSpec(idx int, name string) patternSpec {
	var pattern string
	switch idx % 2 {
	case 0:
		pattern = `(?P<ip>(?P<a>\d{1,3})\.(?P<b>\d{1,3})\.(?P<c>\d{1,3})\.(?P<d>\d{1,3}))(?::(?P<port>\d{1,5}))?`
	default:
		pattern = `(?P<full>(?P<ip>(?:\d{1,3}\.){3}\d{1,3})(?:/(?P<cidr>\d{1,2}))?)`
	}

	inputs := []string{
		"192.168.1.1",
		"10.0.0.1:8080",
		fmt.Sprintf("%d.%d.%d.%d", idx%256, (idx+1)%256, (idx+2)%256, (idx+3)%256),
		"192.168.0.0/24",
		"not an ip",
	}

	return patternSpec{
		Category: categoryTDFA,
		Name:     name,
		Pattern:  pattern,
		Inputs:   normalizeInputs(inputs, 5),
	}
}

// buildUUIDCaptureSpec creates UUID patterns with captures
func buildUUIDCaptureSpec(idx int, name string) patternSpec {
	pattern := `(?P<uuid>(?P<time_low>[0-9a-f]{8})-(?P<time_mid>[0-9a-f]{4})-(?P<version>[0-9a-f]{4})-(?P<variant>[0-9a-f]{4})-(?P<node>[0-9a-f]{12}))`

	inputs := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", idx, idx%0xffff, 0x4000|(idx%0xfff), 0x8000|(idx%0x3fff), idx*12345),
		"123e4567-e89b-12d3-a456-426614174000",
		"not-a-valid-uuid-string",
		"AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE",
	}

	return patternSpec{
		Category: categoryTDFA,
		Name:     name,
		Pattern:  pattern,
		Inputs:   normalizeInputs(inputs, 5),
	}
}

// buildCreditCardSpec creates credit card patterns with captures
func buildCreditCardSpec(idx int, name string) patternSpec {
	var pattern string
	switch idx % 2 {
	case 0:
		pattern = `(?P<card>(?P<issuer>4\d{3}|5[1-5]\d{2}|3[47]\d{2}|6(?:011|5\d{2}))[\s-]?(?P<digits>(?:\d{4}[\s-]?){3}|\d{6}[\s-]?\d{5}))`
	default:
		pattern = `(?P<number>(?P<prefix>\d{4})[\s-]?(?P<middle>\d{4}[\s-]?\d{4})[\s-]?(?P<suffix>\d{4}))`
	}

	inputs := []string{
		"4111-1111-1111-1111",
		"5500 0000 0000 0004",
		"378282246310005",
		fmt.Sprintf("4%03d-%04d-%04d-%04d", idx%1000, idx%10000, idx%10000, idx%10000),
		"invalid card number",
	}

	return patternSpec{
		Category: categoryTDFA,
		Name:     name,
		Pattern:  pattern,
		Inputs:   normalizeInputs(inputs, 5),
	}
}

// buildHTMLTagSpec creates HTML tag patterns with captures
func buildHTMLTagSpec(idx int, name string) patternSpec {
	var pattern string
	switch idx % 3 {
	case 0:
		pattern = `<(?P<tag>[\w-]+)(?:\s+(?P<attrs>(?:[\w-]+="[^"]*"\s*)*))?(?P<close>/?)>`
	case 1:
		pattern = `<(?P<element>(?P<name>[\w-]+)(?:\s+(?P<attr>[\w-]+)="(?P<value>[^"]*)")*\s*/?)>`
	default:
		pattern = `(?P<full><(?P<tag>\w+)(?P<attributes>(?:\s+\w+="[^"]*")*)>(?P<content>.*?)</\2>)`
	}

	tags := []string{"div", "span", "p", "a", "input", "button", "form", "table"}
	tag := tags[idx%len(tags)]
	inputs := []string{
		fmt.Sprintf(`<%s class="container" id="main">`, tag),
		fmt.Sprintf(`<%s/>`, tag),
		fmt.Sprintf(`<%s href="https://example.com" target="_blank">`, tag),
		fmt.Sprintf(`<%s>content</%s>`, tag, tag),
		"not an html tag",
	}

	return patternSpec{
		Category: categoryTDFA,
		Name:     name,
		Pattern:  pattern,
		Inputs:   normalizeInputs(inputs, 5),
	}
}

func normalizeInputs(raw []string, min int) []string {
	inputs := dedupeInputs(raw)
	for len(inputs) < min {
		inputs = append(inputs, fmt.Sprintf("invalid_input_%d", len(inputs)))
	}
	return inputs
}

func dedupeInputs(inputs []string) []string {
	seen := make(map[string]struct{}, len(inputs))
	result := make([]string, 0, len(inputs))
	for _, input := range inputs {
		if _, ok := seen[input]; ok {
			continue
		}
		seen[input] = struct{}{}
		result = append(result, input)
	}
	return result
}

func reverseStrings(values []string) []string {
	copySlice := make([]string, len(values))
	copy(copySlice, values)
	for i, j := 0, len(copySlice)-1; i < j; i, j = i+1, j-1 {
		copySlice[i], copySlice[j] = copySlice[j], copySlice[i]
	}
	return copySlice
}

func capitalizedNamePool() []string {
	return []string{
		"Alpha", "Bravo", "Charlie", "Delta", "Echo", "Foxtrot", "Golf", "Hotel", "India", "Juliet",
		"Kilo", "Lima", "Mike", "November", "Oscar", "Papa", "Quebec", "Romeo", "Sierra", "Tango",
		"Uniform", "Victor", "Whiskey", "Xray", "Yankee", "Zulu",
	}
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
		// Parse: BenchmarkName-12  1  12345 ns/op  1234 B/op  12 allocs/op

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

func analyzeBenchmarks(output string, specs []patternSpec) map[patternCategory]*benchmarkComparison {
	benchResults := parseBenchmarkResults(output)
	comparisons := make(map[patternCategory]*benchmarkComparison)

	// Initialize comparisons for each category
	for _, cat := range []patternCategory{categorySimple, categoryComplex, categoryVeryComplex, categoryTDFA} {
		comparisons[cat] = &benchmarkComparison{Category: cat}
	}

	// Build a map of pattern names to categories
	nameToCategory := make(map[string]patternCategory)
	for _, spec := range specs {
		nameToCategory[spec.Name] = spec.Category
	}

	// Analyze each benchmark pair
	for name, result := range benchResults {
		// Skip stdlib benchmarks for now
		if strings.Contains(name, "Stdlib") {
			continue
		}

		// Extract pattern name (e.g., "BenchmarkSimpleCase001MatchString-12" -> "SimpleCase001")
		nameWithoutBench := strings.TrimPrefix(name, "Benchmark")
		var patternName string
		for key := range nameToCategory {
			if strings.HasPrefix(nameWithoutBench, key) {
				patternName = key
				break
			}
		}

		if patternName == "" {
			continue
		}

		category := nameToCategory[patternName]
		comp := comparisons[category]

		// Find corresponding stdlib benchmark
		stdlibName := strings.Replace(name, "Benchmark", "BenchmarkStdlib", 1)
		if strings.Contains(name, "MatchString") {
			stdlibName = strings.Replace(stdlibName, "MatchString", "MatchString", 1)
		} else if strings.Contains(name, "FindString") {
			stdlibName = strings.Replace(stdlibName, "FindString", "FindStringSubmatch", 1)
		}

		stdlibResult, hasStdlib := benchResults[stdlibName]
		if !hasStdlib {
			continue
		}

		// Compare performance
		if result.NsPerOp < stdlibResult.NsPerOp {
			comp.RegengoFaster++
		} else {
			comp.StdlibFaster++
			// Track patterns where regengo is slower
			slowerByPct := ((result.NsPerOp - stdlibResult.NsPerOp) / stdlibResult.NsPerOp) * 100
			comp.SlowerPatterns = append(comp.SlowerPatterns, slowPattern{
				Name:        patternName,
				Pattern:     "", // Will be filled in later
				RegengoNs:   result.NsPerOp,
				StdlibNs:    stdlibResult.NsPerOp,
				SlowerByPct: slowerByPct,
			})
		}

		// Accumulate for averages
		comp.RegengoAvgNs += result.NsPerOp
		comp.StdlibAvgNs += stdlibResult.NsPerOp
		comp.RegengoAvgBytes += float64(result.BytesPerOp)
		comp.StdlibAvgBytes += float64(stdlibResult.BytesPerOp)
		comp.RegengoAvgAllocs += float64(result.AllocsPerOp)
		comp.StdlibAvgAllocs += float64(stdlibResult.AllocsPerOp)
	}

	// Fill in pattern strings and sort by slowdown percentage
	specsMap := make(map[string]string)
	for _, spec := range specs {
		specsMap[spec.Name] = spec.Pattern
	}

	for _, comp := range comparisons {
		for i := range comp.SlowerPatterns {
			comp.SlowerPatterns[i].Pattern = specsMap[comp.SlowerPatterns[i].Name]
		}
		// Sort slower patterns by percentage (worst first)
		for i := 0; i < len(comp.SlowerPatterns)-1; i++ {
			for j := i + 1; j < len(comp.SlowerPatterns); j++ {
				if comp.SlowerPatterns[i].SlowerByPct < comp.SlowerPatterns[j].SlowerByPct {
					comp.SlowerPatterns[i], comp.SlowerPatterns[j] = comp.SlowerPatterns[j], comp.SlowerPatterns[i]
				}
			}
		}
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

	return comparisons
}

func runGoCommand(dir string, args ...string) commandResult {
	start := time.Now()
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Env = os.Environ()

	var buffer bytes.Buffer
	cmd.Stdout = &buffer
	cmd.Stderr = &buffer

	err := cmd.Run()

	return commandResult{
		Command:  strings.Join(args, " "),
		Output:   buffer.String(),
		Duration: time.Since(start),
		Err:      err,
	}
}

func printSummary(stats map[patternCategory]*categoryStats, totalPatterns, totalTestCases int, outputDir string, testResult, benchResult commandResult, start time.Time, specs []patternSpec) {
	fmt.Println()
	fmt.Println("======== Mass Generation Summary ========")
	fmt.Printf("Artifacts directory: %s\n", outputDir)

	orderedCategories := []patternCategory{categorySimple, categoryComplex, categoryVeryComplex}
	for _, cat := range orderedCategories {
		bucket := stats[cat]
		if bucket == nil {
			bucket = &categoryStats{}
		}
		fmt.Printf("Category %-12s -> patterns: %3d, test cases: %4d\n", cat, bucket.Patterns, bucket.TestCases)
	}

	fmt.Printf("TOTAL patterns: %d\n", totalPatterns)
	fmt.Printf("TOTAL test cases: %d\n", totalTestCases)
	fmt.Println()

	printCommandSummary(testResult)
	fmt.Println()

	printCommandSummary(benchResult)

	// Analyze and print benchmark comparison
	if benchResult.Err == nil && len(benchResult.Output) > 0 {
		fmt.Println()
		printBenchmarkAnalysis(benchResult.Output, specs)
	}

	fmt.Println()
	fmt.Printf("Completed in %s\n", time.Since(start).Round(time.Millisecond))
}

func printBenchmarkAnalysis(output string, specs []patternSpec) {
	comparisons := analyzeBenchmarks(output, specs)

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

		bytesDiff := ((comp.StdlibAvgBytes - comp.RegengoAvgBytes) / comp.StdlibAvgBytes) * 100
		fmt.Printf("  Avg memory:     %.0f B/op (regengo) vs %.0f B/op (stdlib) ",
			comp.RegengoAvgBytes, comp.StdlibAvgBytes)
		if bytesDiff > 0 {
			fmt.Printf("[%.1f%% less]\n", bytesDiff)
		} else {
			fmt.Printf("[%.1f%% more]\n", -bytesDiff)
		}

		allocsDiff := ((comp.StdlibAvgAllocs - comp.RegengoAvgAllocs) / comp.StdlibAvgAllocs) * 100
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

	// Print overall summary
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

	// Print detailed analysis of slower patterns
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

		// Show top 10 worst performers
		limit := 10
		if len(comp.SlowerPatterns) < limit {
			limit = len(comp.SlowerPatterns)
		}

		for i := 0; i < limit; i++ {
			sp := comp.SlowerPatterns[i]
			fmt.Printf("  %2d. %s\n", i+1, sp.Name)
			fmt.Printf("      Pattern: %s\n", sp.Pattern)
			fmt.Printf("      Regengo: %.0f ns/op | Stdlib: %.0f ns/op | Slower by: %.1f%%\n",
				sp.RegengoNs, sp.StdlibNs, sp.SlowerByPct)
		}

		if len(comp.SlowerPatterns) > limit {
			fmt.Printf("  ... and %d more patterns\n", len(comp.SlowerPatterns)-limit)
		}
	}

	if !hasSlowerPatterns {
		fmt.Println("\n  🎉 No patterns where stdlib is faster!")
	}
}

func printCommandSummary(result commandResult) {
	status := "PASS"
	if result.Err != nil {
		status = "FAIL"
	}
	fmt.Printf("Command: %s\n", result.Command)
	fmt.Printf("Status: %s (duration %s)\n", status, result.Duration.Round(time.Millisecond))
	if result.Err != nil {
		fmt.Printf("Error output:\n%s\n", result.Output)
	}
}

func validateSpecs(specs []patternSpec) error {
	if len(specs) == 0 {
		return fmt.Errorf("no pattern specs generated")
	}

	categorySet := make(map[patternCategory]struct{})
	total := 0
	for _, spec := range specs {
		if spec.Pattern == "" {
			return fmt.Errorf("spec %s is missing a pattern", spec.Name)
		}
		if len(spec.Inputs) < 2 {
			return fmt.Errorf("spec %s has insufficient inputs", spec.Name)
		}
		categorySet[spec.Category] = struct{}{}
		total += len(spec.Inputs)
	}

	if len(categorySet) < 3 {
		return fmt.Errorf("expected patterns across simple, complex, and very complex categories, found %d", len(categorySet))
	}

	if total < 500 {
		return fmt.Errorf("only %d test cases generated; expected at least 500", total)
	}

	return nil
}

func printHelp() {
	fmt.Printf("Usage: %s [OPTIONS]\n\n", appName)
	fmt.Println("Mass test generator for regengo benchmark testing")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  generate   Generate test files for benchmarking")
	fmt.Println("  benchmark  Run benchmarks on generated tests")
	fmt.Println("  delete     Delete all generated test files")
	fmt.Println()
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Printf("  %s -command=generate                    # Generate tests in benchmarks/mass_generated\n", appName)
	fmt.Printf("  %s -command=benchmark                   # Run benchmarks (preserves tests)\n", appName)
	fmt.Printf("  %s -command=delete                      # Delete generated tests\n", appName)
	fmt.Printf("  %s -command=generate -output-dir=/tmp/tests  # Generate in custom directory\n", appName)
	fmt.Printf("\n  # Complete workflow:\n")
	fmt.Printf("  %s -command=generate && %s -command=benchmark && %s -command=delete\n", appName, appName, appName)
}

func generateTests(outputDir string) error {
	fmt.Printf("Generating tests in directory: %s\n", outputDir)
	start := time.Now()

	specs := buildPatternSpecs()
	if err := validateSpecs(specs); err != nil {
		return fmt.Errorf("spec validation failed: %v", err)
	}

	// Clean existing directory if it exists
	if err := os.RemoveAll(outputDir); err != nil {
		return fmt.Errorf("failed to clean existing directory: %v", err)
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	packageName := "generated"
	stats := map[patternCategory]*categoryStats{}
	totalTestCases := 0

	for _, spec := range specs {
		caseDir := filepath.Join(outputDir, spec.Name)
		if err := os.MkdirAll(caseDir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %v", spec.Name, err)
		}

		opts := regengo.Options{
			Pattern:          spec.Pattern,
			Name:             spec.Name,
			OutputFile:       filepath.Join(caseDir, fmt.Sprintf("%s.go", spec.Name)),
			Package:          packageName,
			GenerateTestFile: true,
			TestFileInputs:   spec.Inputs,
		}

		if err := regengo.Compile(opts); err != nil {
			return fmt.Errorf("failed to generate artifacts for %s: %v", spec.Name, err)
		}

		bucket := stats[spec.Category]
		if bucket == nil {
			bucket = &categoryStats{}
			stats[spec.Category] = bucket
		}
		bucket.Patterns++
		bucket.TestCases += len(spec.Inputs)
		totalTestCases += len(spec.Inputs)
	}

	duration := time.Since(start)
	fmt.Printf("✅ Successfully generated %d patterns (%d test cases) in %v\n", len(specs), totalTestCases, duration.Round(time.Millisecond))

	// Print category breakdown
	fmt.Println("\nGenerated patterns by category:")
	for category, stat := range stats {
		fmt.Printf("  %s: %d patterns, %d test cases\n", category, stat.Patterns, stat.TestCases)
	}

	return nil
}

func runBenchmarks(outputDir string) error {
	// Check if tests exist
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		return fmt.Errorf("no generated tests found in %s. Run 'generate' command first", outputDir)
	}

	fmt.Printf("Running benchmarks on tests in: %s\n", outputDir)
	start := time.Now()

	specs := buildPatternSpecs()
	if err := validateSpecs(specs); err != nil {
		return fmt.Errorf("spec validation failed: %v", err)
	}

	stats := map[patternCategory]*categoryStats{}
	totalTestCases := 0
	for _, spec := range specs {
		bucket := stats[spec.Category]
		if bucket == nil {
			bucket = &categoryStats{}
			stats[spec.Category] = bucket
		}
		bucket.Patterns++
		bucket.TestCases += len(spec.Inputs)
		totalTestCases += len(spec.Inputs)
	}

	testResult := runGoCommand(outputDir, "go", "test", "./...")
	benchResult := runGoCommand(outputDir, "go", "test", "-run", "^$", "-bench", ".", "-benchmem", "./...")

	printSummary(stats, len(specs), totalTestCases, outputDir, testResult, benchResult, start, specs)

	if testResult.Err != nil || benchResult.Err != nil {
		return fmt.Errorf("benchmark execution failed")
	}

	fmt.Printf("\n📁 Benchmark completed. Tests preserved in: %s\n", outputDir)
	fmt.Printf("💡 Use '%s -command=delete' to clean up when done\n", appName)

	return nil
}

func deleteGeneratedTests(outputDir string) error {
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		fmt.Printf("No generated tests found in %s\n", outputDir)
		return nil
	}

	fmt.Printf("Deleting generated tests from: %s\n", outputDir)

	if err := os.RemoveAll(outputDir); err != nil {
		return fmt.Errorf("failed to delete directory %s: %v", outputDir, err)
	}

	fmt.Printf("✅ Successfully deleted generated tests from %s\n", outputDir)
	return nil
}
