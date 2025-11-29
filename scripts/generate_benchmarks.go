//go:build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/KromDaniel/regengo/pkg/regengo"
)

var testCases = []TestCase{
	{
		Name:    "Email",
		Pattern: `[\w\.+-]+@[\w\.-]+\.[\w\.-]+`,
		Input: []string{
			strings.Repeat("a", 1e+2) + "| me@myself.com",
		},
	},
	{
		Name:    "Greedy",
		Pattern: `(?:(?:a|b)|(?:k)+)*abcd`,
		Input: []string{
			strings.Repeat("a", 1e+2) + "aaaaaaabcd",
		},
	},
	{
		Name:    "Lazy",
		Pattern: `(?:(?:a|b)|(?:k)+)+?abcd`,
		Input: []string{
			strings.Repeat("a", 1e+2) + "aaaaaaabcd",
		},
	},
}

var captureCases = []CaptureCase{
	{
		Name:    "EmailCapture",
		Pattern: `(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)`,
		Input: []string{
			"user@example.com",
			"john.doe+tag@subdomain.example.co.uk",
			"test@test.org",
		},
	},
	{
		Name:    "URLCapture",
		Pattern: `(?P<protocol>https?)://(?P<host>[\w\.-]+)(?::(?P<port>\d+))?(?P<path>/[\w\./]*)?`,
		Input: []string{
			"http://example.com",
			"https://api.github.com:443/repos/owner/repo",
			"http://localhost:8080/api/v1/users",
		},
	},
	{
		Name:    "DateCapture",
		Pattern: `(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})`,
		Input: []string{
			"2025-10-05",
			"1999-12-31",
			"2000-01-01",
		},
	},
}

var findAllCases = []CaptureCase{
	{
		Name:    "MultiDate",
		Pattern: `(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})`,
		Input: []string{
			"Events: 2024-01-15, 2024-06-20, and 2024-12-25 are holidays",
			"No dates here",
			"Single date 2025-10-05 in text",
		},
	},
	{
		Name:    "MultiEmail",
		Pattern: `(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)`,
		Input: []string{
			"Contact us at support@example.com or sales@company.org for help",
			"Multiple: a@b.com, c@d.org, e@f.net",
			"No emails in this text",
		},
	},
}

// tdfaCases contains patterns that trigger Tagged DFA due to catastrophic backtracking risk.
// These patterns have nested quantifiers + captures which would cause exponential backtracking
// without TDFA's O(n) guarantee.
var tdfaCases = []CaptureCase{
	{
		// Classic pathological pattern: (a+)+ with captures
		// Without TDFA this would be O(2^n), with TDFA it's O(n)
		Name:    "TDFAPathological",
		Pattern: `(?P<outer>(?P<inner>a+)+)b`,
		Input: []string{
			"aaaaaaaaaaaaaaaaaaaab",           // 20 a's + b (matches)
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaab", // 30 a's + b (matches)
			strings.Repeat("a", 50) + "b",     // 50 a's + b (matches)
			strings.Repeat("a", 30),           // no match - would hang without TDFA
		},
	},
	{
		// Nested quantifiers with word boundaries - common in real patterns
		Name:    "TDFANestedWord",
		Pattern: `(?P<words>(?P<word>\w+\s*)+)end`,
		Input: []string{
			"hello world end",
			"a b c d e f g h i j end",
			strings.Repeat("word ", 20) + "end",
		},
	},
	{
		// Complex URL with optional components - uses character class compression
		Name:    "TDFAComplexURL",
		Pattern: `(?P<scheme>https?)://(?P<auth>(?P<user>[\w.-]+)(?::(?P<pass>[\w.-]+))?@)?(?P<host>[\w.-]+)(?::(?P<port>\d+))?(?P<path>/[\w./-]*)?(?:\?(?P<query>[\w=&.-]+))?`,
		Input: []string{
			"https://example.com",
			"http://user:pass@example.com:8080/path/to/resource?key=value&foo=bar",
			"https://api.github.com/repos/owner/repo",
		},
	},
	{
		// Log line parser with multiple optional groups
		Name:    "TDFALogParser",
		Pattern: `(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})(?:\.(?P<ms>\d{3}))?(?P<tz>Z|[+-]\d{2}:\d{2})?\s+\[(?P<level>\w+)\]\s+(?P<message>.+)`,
		Input: []string{
			"2024-01-15T10:30:45.123Z [INFO] Server started successfully",
			"2024-01-15T10:30:45+00:00 [ERROR] Connection failed",
			"2024-01-15T10:30:45 [DEBUG] Processing request id=12345",
		},
	},
	{
		// Semantic version with optional pre-release and build metadata
		Name:    "TDFASemVer",
		Pattern: `(?P<major>\d+)\.(?P<minor>\d+)\.(?P<patch>\d+)(?:-(?P<prerelease>[\w.-]+))?(?:\+(?P<build>[\w.-]+))?`,
		Input: []string{
			"1.0.0",
			"2.1.3-alpha.1",
			"3.0.0-beta.2+build.123",
			"10.20.30-rc.1+20240115",
		},
	},
}

var tnfaCases = []CaptureCase{
	{
		Name:    "TNFAPathological",
		Pattern: `(?P<outer>(?P<inner>a+)+)b`,
		Input: []string{
			"aaaaaaaaaaaaaaaaaaaab",           // 20 a's + b (matches)
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaab", // 30 a's + b (matches)
		},
	},
}

var replaceCases = []ReplaceCase{
	{
		Name:    "ReplaceEmail",
		Pattern: `(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)`,
		Input: []string{
			"Contact support@example.com for help",
			"Multiple: a@b.com, c@d.org, e@f.net in one line",
			strings.Repeat("user@example.com ", 50),
		},
		Replacers: []string{
			"$user@REDACTED.$tld",
			"[EMAIL]",
			"$0",
		},
	},
	{
		Name:    "ReplaceDate",
		Pattern: `(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})`,
		Input: []string{
			"Event on 2024-01-15",
			"Range: 2024-01-01 to 2024-12-31",
			strings.Repeat("2024-06-15 ", 100),
		},
		Replacers: []string{
			"$month/$day/$year",
			"[DATE]",
			"$year",
		},
	},
	{
		Name:    "ReplaceURL",
		Pattern: `(?P<protocol>https?)://(?P<host>[\w\.-]+)(?::(?P<port>\d+))?(?P<path>/[\w\./]*)?`,
		Input: []string{
			"Visit https://example.com/page for info",
			"API at http://localhost:8080/api/v1/users",
			strings.Repeat("https://test.com/path ", 30),
		},
		Replacers: []string{
			"$protocol://$host[REDACTED]",
			"[URL]",
			"$host",
		},
	},
}

var testTemplate = `
package generated

import (
	"regexp"
	"testing"
)


func Test{{ .Name }}MatchString(t *testing.T) {
	pattern := {{ quote .Pattern }}
	stdReg := regexp.MustCompile(pattern)
	{{ $out := . }}
	{{ range $index, $input := .Input }}
	t.Run("test input {{ $index }}", func(t *testing.T) {
		input := {{ quote $input }}
        isStdMatch := stdReg.MatchString(input)
        isRegengoMatch := {{ $out.Name }}{}.MatchString(input)
        if isStdMatch != isRegengoMatch {
			t.Fatalf("pattern %s stdMatch - %v, regengoMatch - %v", input, isStdMatch, isRegengoMatch)
        }
	})

	{{ end }}
}

func Benchmark{{ .Name }}MatchString(b *testing.B) {
	pattern := {{ quote .Pattern }}
	stdReg := regexp.MustCompile(pattern)
	{{ $out := . }}
	{{ range $index, $input := .Input }}

	b.Run("golang std {{ $index }}", func(b *testing.B) {
		b.ReportAllocs()
		input := {{ quote $input }}
		for b.Loop() {
			stdReg.MatchString(input)
		}
	})

	b.Run("regengo {{ $index }}", func(b *testing.B) {
		b.ReportAllocs()
		input := {{ quote $input }}
		for b.Loop() {
			{{ $out.Name }}{}.MatchString(input)
		}
	})

	{{ end }}
}
`

var captureTestTemplate = `
package generated

import (
	"regexp"
	"testing"
)


func Test{{ .Name }}FindString(t *testing.T) {
	pattern := {{ quote .Pattern }}
	stdReg := regexp.MustCompile(pattern)
	{{ $out := . }}
	{{ range $index, $input := .Input }}
	t.Run("test input {{ $index }}", func(t *testing.T) {
		input := {{ quote $input }}
		stdMatch := stdReg.FindStringSubmatch(input)
		regengoResult, found := {{ $out.Name }}{}.FindString(input)
		
		if (len(stdMatch) > 0) != found {
			t.Fatalf("pattern %s stdMatch found=%v, regengo found=%v", input, len(stdMatch) > 0, found)
		}
		
		if found {
			// Verify the full match
			if stdMatch[0] != regengoResult.Match {
				t.Errorf("Full match mismatch: std=%q regengo=%q", stdMatch[0], regengoResult.Match)
			}
		}
	})

	{{ end }}
}

func Benchmark{{ .Name }}FindString(b *testing.B) {
	pattern := {{ quote .Pattern }}
	stdReg := regexp.MustCompile(pattern)
	{{ $out := . }}
	{{ range $index, $input := .Input }}

	b.Run("golang std {{ $index }}", func(b *testing.B) {
		b.ReportAllocs()
		input := {{ quote $input }}
		for b.Loop() {
			stdReg.FindStringSubmatch(input)
		}
	})

	b.Run("regengo {{ $index }}", func(b *testing.B) {
		b.ReportAllocs()
		input := {{ quote $input }}
		for b.Loop() {
			{{ $out.Name }}{}.FindString(input)
		}
	})

	b.Run("regengo reuse {{ $index }}", func(b *testing.B) {
		b.ReportAllocs()
		input := {{ quote $input }}
		var result *{{ $out.Name }}Result
		for b.Loop() {
			result, _ = {{ $out.Name }}{}.FindStringReuse(input, result)
		}
	})

	{{ end }}
}
`

var findAllTestTemplate = `
package generated

import (
	"reflect"
	"regexp"
	"testing"
)


func Test{{ .Name }}FindAllString(t *testing.T) {
	pattern := {{ quote .Pattern }}
	stdReg := regexp.MustCompile(pattern)
	{{ $out := . }}
	{{ range $index, $input := .Input }}
	t.Run("test input {{ $index }}", func(t *testing.T) {
		input := {{ quote $input }}
		stdMatches := stdReg.FindAllStringSubmatch(input, -1)
		regengoResults := {{ $out.Name }}{}.FindAllString(input, -1)

		if len(stdMatches) != len(regengoResults) {
			t.Fatalf("match count mismatch: std=%d, regengo=%d", len(stdMatches), len(regengoResults))
		}

		for i, stdMatch := range stdMatches {
			result := regengoResults[i]

			// Compare full match
			if stdMatch[0] != result.Match {
				t.Errorf("match %d: full match mismatch: std=%q regengo=%q", i, stdMatch[0], result.Match)
			}

			// Compare each capture group using reflection
			v := reflect.ValueOf(result).Elem()
			typ := v.Type()

			// Field 0 is Match, so capture groups start at field 1
			for j := 1; j < v.NumField(); j++ {
				fieldName := typ.Field(j).Name
				fieldValue := v.Field(j).String()

				// stdMatch index: 0=full match, 1=first group, etc.
				if j < len(stdMatch) {
					if stdMatch[j] != fieldValue {
						t.Errorf("match %d, field %s: std=%q regengo=%q", i, fieldName, stdMatch[j], fieldValue)
					}
				}
			}
		}
	})

	{{ end }}
}

func Benchmark{{ .Name }}FindAllString(b *testing.B) {
	pattern := {{ quote .Pattern }}
	stdReg := regexp.MustCompile(pattern)
	{{ $out := . }}
	{{ range $index, $input := .Input }}

	b.Run("golang std {{ $index }}", func(b *testing.B) {
		b.ReportAllocs()
		input := {{ quote $input }}
		for b.Loop() {
			stdReg.FindAllStringSubmatch(input, -1)
		}
	})

	b.Run("regengo {{ $index }}", func(b *testing.B) {
		b.ReportAllocs()
		input := {{ quote $input }}
		for b.Loop() {
			{{ $out.Name }}{}.FindAllString(input, -1)
		}
	})

	b.Run("regengo reuse {{ $index }}", func(b *testing.B) {
		b.ReportAllocs()
		input := {{ quote $input }}
		results := make([]*{{ $out.Name }}Result, 0, 10)
		for b.Loop() {
			results = {{ $out.Name }}{}.FindAllStringAppend(input, -1, results[:0])
		}
	})

	{{ end }}
}
`

var replaceTestTemplate = `
package generated

import (
	"regexp"
	"testing"
)

// convertToStdlibTemplate converts regengo template syntax to stdlib syntax.
// regengo: $name, $1, $0 -> stdlib: ${name}, ${1}, ${0}
func convertToStdlibTemplate{{ .Name }}(template string) string {
	var result []byte
	i := 0
	for i < len(template) {
		if template[i] != '$' {
			result = append(result, template[i])
			i++
			continue
		}
		if i+1 >= len(template) {
			result = append(result, '$')
			i++
			continue
		}
		next := template[i+1]
		switch {
		case next == '$':
			result = append(result, '$', '$')
			i += 2
		case next == '{':
			closeIdx := i + 2
			for closeIdx < len(template) && template[closeIdx] != '}' {
				closeIdx++
			}
			if closeIdx < len(template) {
				result = append(result, template[i:closeIdx+1]...)
				i = closeIdx + 1
			} else {
				result = append(result, '$')
				i++
			}
		case next >= '0' && next <= '9':
			j := i + 1
			for j < len(template) && template[j] >= '0' && template[j] <= '9' {
				j++
			}
			result = append(result, '$', '{')
			result = append(result, template[i+1:j]...)
			result = append(result, '}')
			i = j
		case next == '_' || (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z'):
			j := i + 1
			for j < len(template) && (template[j] == '_' || (template[j] >= 'a' && template[j] <= 'z') || (template[j] >= 'A' && template[j] <= 'Z') || (template[j] >= '0' && template[j] <= '9')) {
				j++
			}
			result = append(result, '$', '{')
			result = append(result, template[i+1:j]...)
			result = append(result, '}')
			i = j
		default:
			result = append(result, '$')
			i++
		}
	}
	return string(result)
}

func Test{{ .Name }}ReplaceAllString(t *testing.T) {
	pattern := {{ quote .Pattern }}
	stdReg := regexp.MustCompile(pattern)
	{{ $out := . }}
	{{ range $rIdx, $replacer := .Replacers }}
	{{ range $iIdx, $input := $out.Input }}
	t.Run("replacer {{ $rIdx }} input {{ $iIdx }}", func(t *testing.T) {
		input := {{ quote $input }}
		replacer := {{ quote $replacer }}
		stdlibTemplate := convertToStdlibTemplate{{ $out.Name }}(replacer)
		expected := stdReg.ReplaceAllString(input, stdlibTemplate)
		got := {{ $out.Name }}{}.ReplaceAllString(input, replacer)
		if got != expected {
			t.Errorf("ReplaceAllString mismatch:\n  input=%q\n  replacer=%q\n  got=%q\n  want=%q", input, replacer, got, expected)
		}
	})
	{{ end }}
	{{ end }}
}

func Benchmark{{ .Name }}ReplaceAllString(b *testing.B) {
	pattern := {{ quote .Pattern }}
	stdReg := regexp.MustCompile(pattern)
	{{ $out := . }}
	{{ range $rIdx, $replacer := .Replacers }}
	{{ range $iIdx, $input := $out.Input }}

	b.Run("stdlib r{{ $rIdx }}_i{{ $iIdx }}", func(b *testing.B) {
		b.ReportAllocs()
		input := {{ quote $input }}
		stdlibTemplate := convertToStdlibTemplate{{ $out.Name }}({{ quote $replacer }})
		for b.Loop() {
			stdReg.ReplaceAllString(input, stdlibTemplate)
		}
	})

	b.Run("regengo runtime r{{ $rIdx }}_i{{ $iIdx }}", func(b *testing.B) {
		b.ReportAllocs()
		input := {{ quote $input }}
		replacer := {{ quote $replacer }}
		for b.Loop() {
			{{ $out.Name }}{}.ReplaceAllString(input, replacer)
		}
	})

	b.Run("regengo precompiled{{ $rIdx }} i{{ $iIdx }}", func(b *testing.B) {
		b.ReportAllocs()
		input := {{ quote $input }}
		for b.Loop() {
			{{ $out.Name }}{}.ReplaceAllString{{ $rIdx }}(input)
		}
	})

	{{ end }}
	{{ end }}
}

func Benchmark{{ .Name }}ReplaceAllBytesAppend(b *testing.B) {
	{{ $out := . }}
	{{ range $rIdx, $replacer := .Replacers }}
	{{ range $iIdx, $input := $out.Input }}

	b.Run("precompiled{{ $rIdx }} i{{ $iIdx }}", func(b *testing.B) {
		b.ReportAllocs()
		input := []byte({{ quote $input }})
		buf := make([]byte, 0, 4096)
		for b.Loop() {
			buf = {{ $out.Name }}{}.ReplaceAllBytesAppend{{ $rIdx }}(input, buf)
			buf = buf[:0]
		}
	})

	{{ end }}
	{{ end }}
}
`

var cwd string

func init() {
	var err error
	cwd, err = os.Getwd()
	if err != nil {
		panic(fmt.Errorf("unable to get cwd: %w", err))
	}
}

type TestCase struct {
	Name    string   `json:"name"`
	Pattern string   `json:"pattern"`
	Input   []string `json:"input"`
}

type CaptureCase struct {
	Name    string   `json:"name"`
	Pattern string   `json:"pattern"`
	Input   []string `json:"input"`
}

type ReplaceCase struct {
	Name      string   `json:"name"`
	Pattern   string   `json:"pattern"`
	Input     []string `json:"input"`
	Replacers []string `json:"replacers"`
}

func main() {
	testTemplate, err := template.New("auto_gen_test").Funcs(map[string]interface{}{
		"quote": func(v string) string { return strconv.Quote(v) },
	}).Parse(testTemplate)

	if err != nil {
		panic(err)
	}

	captureTemplate, err := template.New("auto_gen_capture_test").Funcs(map[string]interface{}{
		"quote": func(v string) string { return strconv.Quote(v) },
	}).Parse(captureTestTemplate)

	if err != nil {
		panic(err)
	}

	findAllTemplate, err := template.New("auto_gen_findall_test").Funcs(map[string]interface{}{
		"quote": func(v string) string { return strconv.Quote(v) },
	}).Parse(findAllTestTemplate)

	if err != nil {
		panic(err)
	}

	// Generate regular matchers
	for _, testCase := range testCases {
		if err := regengo.Compile(regengo.Options{
			Pattern:    testCase.Pattern,
			Name:       testCase.Name,
			OutputFile: filepath.Join(cwd, "benchmarks", "generated", fmt.Sprintf("%s.go", testCase.Name)),
			Package:    "generated",
		}); err != nil {
			panic(err)
		}

		testFile, err := os.Create(filepath.Join(cwd, "benchmarks", "generated", fmt.Sprintf("%s_test.go", testCase.Name)))
		if err != nil {
			panic(err)
		}
		if err := testTemplate.Execute(testFile, testCase); err != nil {
			panic(err)
		}
		if err := testFile.Close(); err != nil {
			panic(err)
		}
	}

	// Generate capture group matchers
	for _, captureCase := range captureCases {
		if err := regengo.Compile(regengo.Options{
			Pattern:    captureCase.Pattern,
			Name:       captureCase.Name,
			OutputFile: filepath.Join(cwd, "benchmarks", "generated", fmt.Sprintf("%s.go", captureCase.Name)),
			Package:    "generated",
			// WithCaptures removed - now auto-detected from pattern
		}); err != nil {
			panic(err)
		}

		testFile, err := os.Create(filepath.Join(cwd, "benchmarks", "generated", fmt.Sprintf("%s_test.go", captureCase.Name)))
		if err != nil {
			panic(err)
		}
		if err := captureTemplate.Execute(testFile, captureCase); err != nil {
			panic(err)
		}
		if err := testFile.Close(); err != nil {
			panic(err)
		}
	}

	// Generate FindAll matchers
	for _, findAllCase := range findAllCases {
		if err := regengo.Compile(regengo.Options{
			Pattern:    findAllCase.Pattern,
			Name:       findAllCase.Name,
			OutputFile: filepath.Join(cwd, "benchmarks", "generated", fmt.Sprintf("%s.go", findAllCase.Name)),
			Package:    "generated",
		}); err != nil {
			panic(err)
		}

		testFile, err := os.Create(filepath.Join(cwd, "benchmarks", "generated", fmt.Sprintf("%s_test.go", findAllCase.Name)))
		if err != nil {
			panic(err)
		}
		if err := findAllTemplate.Execute(testFile, findAllCase); err != nil {
			panic(err)
		}
		if err := testFile.Close(); err != nil {
			panic(err)
		}
	}

	// Generate TDFA benchmark cases (patterns with catastrophic backtracking risk)
	for _, tdfaCase := range tdfaCases {
		if err := regengo.Compile(regengo.Options{
			Pattern:    tdfaCase.Pattern,
			Name:       tdfaCase.Name,
			OutputFile: filepath.Join(cwd, "benchmarks", "generated", fmt.Sprintf("%s.go", tdfaCase.Name)),
			Package:    "generated",
		}); err != nil {
			panic(err)
		}

		testFile, err := os.Create(filepath.Join(cwd, "benchmarks", "generated", fmt.Sprintf("%s_test.go", tdfaCase.Name)))
		if err != nil {
			panic(err)
		}
		if err := captureTemplate.Execute(testFile, tdfaCase); err != nil {
			panic(err)
		}
		if err := testFile.Close(); err != nil {
			panic(err)
		}
	}

	// Generate TNFA benchmark cases (forced TNFA/memoization)
	for _, tnfaCase := range tnfaCases {
		if err := regengo.Compile(regengo.Options{
			Pattern:    tnfaCase.Pattern,
			Name:       tnfaCase.Name,
			OutputFile: filepath.Join(cwd, "benchmarks", "generated", fmt.Sprintf("%s.go", tnfaCase.Name)),
			Package:    "generated",
			ForceTNFA:  true,
		}); err != nil {
			panic(err)
		}

		testFile, err := os.Create(filepath.Join(cwd, "benchmarks", "generated", fmt.Sprintf("%s_test.go", tnfaCase.Name)))
		if err != nil {
			panic(err)
		}
		if err := captureTemplate.Execute(testFile, tnfaCase); err != nil {
			panic(err)
		}
		if err := testFile.Close(); err != nil {
			panic(err)
		}
	}

	// Generate Replace benchmark cases
	replaceTemplate, err := template.New("auto_gen_replace_test").Funcs(map[string]interface{}{
		"quote": func(v string) string { return strconv.Quote(v) },
	}).Parse(replaceTestTemplate)

	if err != nil {
		panic(err)
	}

	for _, replaceCase := range replaceCases {
		if err := regengo.Compile(regengo.Options{
			Pattern:    replaceCase.Pattern,
			Name:       replaceCase.Name,
			OutputFile: filepath.Join(cwd, "benchmarks", "generated", fmt.Sprintf("%s.go", replaceCase.Name)),
			Package:    "generated",
			Replacers:  replaceCase.Replacers,
		}); err != nil {
			panic(err)
		}

		testFile, err := os.Create(filepath.Join(cwd, "benchmarks", "generated", fmt.Sprintf("%s_test.go", replaceCase.Name)))
		if err != nil {
			panic(err)
		}
		if err := replaceTemplate.Execute(testFile, replaceCase); err != nil {
			panic(err)
		}
		if err := testFile.Close(); err != nil {
			panic(err)
		}
	}

	fmt.Printf("✓ Generated %d regular matchers\n", len(testCases))
	fmt.Printf("✓ Generated %d capture group matchers\n", len(captureCases))
	fmt.Printf("✓ Generated %d FindAll matchers\n", len(findAllCases))
	fmt.Printf("✓ Generated %d TDFA benchmark matchers\n", len(tdfaCases))
	fmt.Printf("✓ Generated %d TNFA benchmark matchers\n", len(tnfaCases))
	fmt.Printf("✓ Generated %d Replace benchmark matchers\n", len(replaceCases))
}
