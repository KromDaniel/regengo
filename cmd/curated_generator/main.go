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

	fmt.Printf("✓ Generated %d regular matchers\n", len(testCases))
	fmt.Printf("✓ Generated %d capture group matchers\n", len(captureCases))
	fmt.Printf("✓ Generated %d FindAll matchers\n", len(findAllCases))
}
