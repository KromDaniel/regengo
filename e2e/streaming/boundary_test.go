package streaming

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/KromDaniel/regengo/pkg/regengo"
)

// BoundaryTestCase tests specific boundary crossing scenarios.
type BoundaryTestCase struct {
	Name       string
	Pattern    string
	Input      string
	ChunkSizes []int
	Expected   []string // Expected match strings
}

// Note: All patterns MUST have capture groups because streaming methods
// are only generated for patterns with captures (WithCaptures=true).
//
// Note: Chunk sizes are for simulating slow reads, but the streaming API
// uses its own internal buffer (default 64KB). The ChunkedReader tests how
// the API handles being fed data slowly, not internal buffer boundaries.
var boundaryTestCases = []BoundaryTestCase{
	{
		Name:       "DatePattern",
		Pattern:    `(\d{4}-\d{2}-\d{2})`,
		Input:      "date 2024-01-15 and 2024-12-31 end",
		ChunkSizes: []int{100, 500, 1000}, // Reasonable sizes
		Expected:   []string{"2024-01-15", "2024-12-31"},
	},
	{
		Name:       "EmailPattern",
		Pattern:    `(\w+@\w+\.\w+)`,
		Input:      "contact user@example.com or admin@test.org today",
		ChunkSizes: []int{100, 500, 1000},
		Expected:   []string{"user@example.com", "admin@test.org"},
	},
	{
		Name:       "MultipleDigits",
		Pattern:    `(\d+)`,
		Input:      "values: 123 456 789 done",
		ChunkSizes: []int{100, 500, 1000},
		Expected:   []string{"123", "456", "789"},
	},
	{
		Name:       "WordBoundary",
		Pattern:    `(hello)`,
		Input:      "hello world hello again",
		ChunkSizes: []int{100, 500, 1000},
		Expected:   []string{"hello", "hello"},
	},
	{
		Name:       "LongerMatch",
		Pattern:    `(a+)`,
		Input:      "x" + strings.Repeat("a", 50) + "x",
		ChunkSizes: []int{100, 500, 1000},
		Expected:   []string{strings.Repeat("a", 50)},
	},
	{
		Name:       "UnicodeMatch",
		Pattern:    `(日本)`,
		Input:      "prefix日本suffix",
		ChunkSizes: []int{100, 500, 1000},
		Expected:   []string{"日本"},
	},
}

// TestBoundaryConditions runs boundary condition tests by generating and running code.
func TestBoundaryConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping boundary tests in short mode")
	}

	tempDir := t.TempDir()

	for i, tc := range boundaryTestCases {
		tc := tc
		idx := i
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			// Setup: generate the pattern code
			caseDir := filepath.Join(tempDir, fmt.Sprintf("boundary%d", idx))
			if err := os.MkdirAll(caseDir, 0755); err != nil {
				t.Fatalf("Failed to create case dir: %v", err)
			}

			patternName := fmt.Sprintf("Boundary%d", idx)
			outputFile := filepath.Join(caseDir, patternName+".go")
			testFile := filepath.Join(caseDir, patternName+"_test.go")

			// Generate code
			opts := regengo.Options{
				Pattern:          tc.Pattern,
				Name:             patternName,
				OutputFile:       outputFile,
				Package:          "generated",
				GenerateTestFile: false,
			}
			if err := regengo.Compile(opts); err != nil {
				t.Fatalf("Failed to generate code: %v", err)
			}

			// Generate custom test file
			testContent := generateBoundaryTestFile(tc, patternName)
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Setup go module
			setupBoundaryGoModule(t, caseDir)

			// Run tests
			cmd := exec.Command("go", "test", "-v", "-count=1")
			cmd.Dir = caseDir
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Boundary tests failed:\nOutput: %s\nError: %v", string(output), err)
			}

			t.Logf("Boundary test passed: %s", tc.Name)
		})
	}
}

// generateBoundaryTestFile creates a test file for boundary conditions.
func generateBoundaryTestFile(tc BoundaryTestCase, patternName string) string {
	expectedSlice := "[]string{"
	for i, exp := range tc.Expected {
		if i > 0 {
			expectedSlice += ", "
		}
		expectedSlice += fmt.Sprintf("%q", exp)
	}
	expectedSlice += "}"

	chunkSizeSlice := "[]int{"
	for i, cs := range tc.ChunkSizes {
		if i > 0 {
			chunkSizeSlice += ", "
		}
		chunkSizeSlice += fmt.Sprintf("%d", cs)
	}
	chunkSizeSlice += "}"

	return fmt.Sprintf(`package generated

import (
	"io"
	"reflect"
	"strings"
	"testing"

	stream "github.com/KromDaniel/regengo/stream"
)

var input = %q
var expected = %s
var chunkSizes = %s

// ChunkedReader wraps a reader and returns data in fixed-size chunks
type ChunkedReader struct {
	reader    io.Reader
	chunkSize int
}

func NewChunkedReader(r io.Reader, chunkSize int) *ChunkedReader {
	if chunkSize < 1 {
		chunkSize = 1
	}
	return &ChunkedReader{reader: r, chunkSize: chunkSize}
}

func (r *ChunkedReader) Read(p []byte) (n int, err error) {
	maxRead := r.chunkSize
	if len(p) < maxRead {
		maxRead = len(p)
	}
	return r.reader.Read(p[:maxRead])
}

func TestBoundaryAllChunkSizes(t *testing.T) {
	for _, chunkSize := range chunkSizes {
		t.Run(intToStr(chunkSize), func(t *testing.T) {
			reader := NewChunkedReader(strings.NewReader(input), chunkSize)

			var results []string
			// Use default config - the streaming API handles minimum buffer size
			err := %s{}.FindReader(reader, stream.Config{}, func(m stream.Match[*%sBytesResult]) bool {
				results = append(results, string(m.Result.Match))
				return true
			})
			if err != nil {
				t.Fatalf("FindReader error: %%v", err)
			}

			if !reflect.DeepEqual(results, expected) {
				t.Errorf("chunkSize=%%d: got %%v, want %%v", chunkSize, results, expected)
			}
		})
	}
}

func TestBoundaryMatchCount(t *testing.T) {
	// Without chunking
	count, err := %s{}.FindReaderCount(strings.NewReader(input), stream.Config{})
	if err != nil {
		t.Fatalf("FindReaderCount error: %%v", err)
	}
	if count != int64(len(expected)) {
		t.Errorf("count = %%d, want %%d", count, len(expected))
	}
}

func TestBoundaryFirstMatch(t *testing.T) {
	if len(expected) == 0 {
		t.Skip("No expected matches")
	}

	result, _, err := %s{}.FindReaderFirst(strings.NewReader(input), stream.Config{})
	if err != nil {
		t.Fatalf("FindReaderFirst error: %%v", err)
	}
	if result == nil {
		t.Fatal("Expected a match but got nil")
	}
	if string(result.Match) != expected[0] {
		t.Errorf("first match = %%q, want %%q", string(result.Match), expected[0])
	}
}

func intToStr(n int) string {
	return "chunk_" + string([]byte{byte('0' + n%%10)})
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
`, tc.Input, expectedSlice, chunkSizeSlice, patternName, patternName, patternName, patternName)
}

// setupBoundaryGoModule initializes go module in test directory.
func setupBoundaryGoModule(t *testing.T, dir string) {
	t.Helper()

	regengoPath, err := getBoundaryRegengoPath()
	if err != nil {
		t.Fatalf("Failed to get regengo module path: %v", err)
	}

	initCmd := exec.Command("go", "mod", "init", "testmodule")
	initCmd.Dir = dir
	if output, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod init failed:\nOutput: %s\nError: %v", string(output), err)
	}

	editCmd := exec.Command("go", "mod", "edit", "-replace",
		"github.com/KromDaniel/regengo="+regengoPath)
	editCmd.Dir = dir
	if output, err := editCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod edit failed:\nOutput: %s\nError: %v", string(output), err)
	}

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = dir
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed:\nOutput: %s\nError: %v", string(output), err)
	}
}

func getBoundaryRegengoPath() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", io.EOF
	}
	dir := filepath.Dir(file)
	return filepath.Abs(filepath.Join(dir, "..", ".."))
}
