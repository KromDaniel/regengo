package streaming

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/KromDaniel/regengo/pkg/regengo"
)

// StreamingTestCase defines a test pattern for streaming tests.
type StreamingTestCase struct {
	Name    string
	Pattern string
	// InputGenerator creates a reader that generates test data with embedded matches.
	InputGenerator func(size int64) *PatternedReader
	// InputSizes to test (in bytes).
	InputSizes []int64
	// ChunkSizes to test for boundary crossing.
	ChunkSizes []int
}

// Note: All patterns MUST have capture groups because streaming methods
// are only generated for patterns with captures (WithCaptures=true).
var streamingTestCases = []StreamingTestCase{
	{
		Name:    "Date",
		Pattern: `(\d{4}-\d{2}-\d{2})`,
		InputGenerator: func(size int64) *PatternedReader {
			return NewDateInputGenerator(size)
		},
		InputSizes: []int64{1 << 16, 1 << 20}, // 64KB, 1MB
		ChunkSizes: []int{64, 256, 1024, 4096},
	},
	{
		Name:    "Email",
		Pattern: `([\w.+-]+@[\w.-]+\.\w+)`,
		InputGenerator: func(size int64) *PatternedReader {
			return NewEmailInputGenerator(size)
		},
		InputSizes: []int64{1 << 16, 1 << 20},
		ChunkSizes: []int{128, 512, 2048},
	},
	{
		Name:    "IPv4",
		Pattern: `(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})`,
		InputGenerator: func(size int64) *PatternedReader {
			return NewIPv4InputGenerator(size)
		},
		InputSizes: []int64{1 << 16, 1 << 20},
		ChunkSizes: []int{64, 256, 1024},
	},
	{
		Name:    "URL",
		Pattern: `(https?://[^\s]+)`,
		InputGenerator: func(size int64) *PatternedReader {
			return NewURLInputGenerator(size)
		},
		InputSizes: []int64{1 << 16, 1 << 20},
		ChunkSizes: []int{128, 512, 2048},
	},
}

// TestStreamingVsStdlib compares streaming results with stdlib regexp results.
// It generates code for each pattern, runs it, and compares match counts.
func TestStreamingVsStdlib(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping streaming tests in short mode")
	}

	tempDir := t.TempDir()

	for _, tc := range streamingTestCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			// Setup: generate the pattern code
			caseDir := filepath.Join(tempDir, tc.Name)
			if err := os.MkdirAll(caseDir, 0755); err != nil {
				t.Fatalf("Failed to create case dir: %v", err)
			}

			outputFile := filepath.Join(caseDir, tc.Name+".go")
			testFile := filepath.Join(caseDir, tc.Name+"_test.go")

			// Generate code
			opts := regengo.Options{
				Pattern:          tc.Pattern,
				Name:             tc.Name,
				OutputFile:       outputFile,
				Package:          "generated",
				GenerateTestFile: false, // We'll generate our own test
			}
			if err := regengo.Compile(opts); err != nil {
				t.Fatalf("Failed to generate code: %v", err)
			}

			// Generate custom streaming test file
			testContent := generateStreamingTestFile(tc)
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Setup go module
			setupGoModule(t, caseDir)

			// Run tests
			cmd := exec.Command("go", "test", "-v", "-count=1")
			cmd.Dir = caseDir
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Streaming tests failed:\nOutput: %s\nError: %v", string(output), err)
			}

			t.Logf("Tests passed for pattern %s", tc.Name)
		})
	}
}

// generateStreamingTestFile generates a test file that compares streaming with stdlib.
func generateStreamingTestFile(tc StreamingTestCase) string {
	return `package generated

import (
	"io"
	"math/rand"
	"regexp"
	"strings"
	"testing"

	stream "github.com/KromDaniel/regengo/stream"
)

var stdlibRe = regexp.MustCompile(` + "`" + tc.Pattern + "`" + `)

// PatternedReader generates data with embedded matches
type PatternedReader struct {
	pattern    []byte
	noise      []byte
	matchEvery int
	pos        int64
	limit      int64
	rng        *rand.Rand
}

func NewPatternedReader(pattern string, noiseChars string, matchEvery int, limit int64) *PatternedReader {
	return &PatternedReader{
		pattern:    []byte(pattern),
		noise:      []byte(noiseChars),
		matchEvery: matchEvery,
		limit:      limit,
		rng:        rand.New(rand.NewSource(42)),
	}
}

func (r *PatternedReader) Read(p []byte) (n int, err error) {
	if r.pos >= r.limit {
		return 0, io.EOF
	}
	for n < len(p) && r.pos < r.limit {
		posInCycle := int(r.pos) % r.matchEvery
		if posInCycle < len(r.pattern) {
			p[n] = r.pattern[posInCycle]
		} else {
			p[n] = r.noise[r.rng.Intn(len(r.noise))]
		}
		n++
		r.pos++
	}
	return n, nil
}

func (r *PatternedReader) Reset() {
	r.pos = 0
	r.rng = rand.New(rand.NewSource(42))
}

func (r *PatternedReader) ExpectedMatches() int64 {
	// Matches at positions 0, matchEvery, 2*matchEvery, ... up to limit
	// Formula: floor((limit - 1) / matchEvery) + 1
	if r.limit == 0 {
		return 0
	}
	return (r.limit-1)/int64(r.matchEvery) + 1
}

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

func TestStreamingFindReaderCount(t *testing.T) {
	// Test count on known input
	input := ` + getSmallTestInput(tc) + `
	expected := int64(len(stdlibRe.FindAllString(input, -1)))

	count, err := ` + tc.Name + `{}.FindReaderCount(strings.NewReader(input), stream.Config{})
	if err != nil {
		t.Fatalf("FindReaderCount error: %v", err)
	}

	if count != expected {
		t.Errorf("Match count = %d, expected %d", count, expected)
	}
}

func TestStreamingVsStdlibSmall(t *testing.T) {
	// Small input for detailed comparison
	input := ` + getSmallTestInput(tc) + `

	// Get stdlib matches
	stdlibMatches := stdlibRe.FindAllString(input, -1)

	// Get streaming matches
	var streamMatches []string
	err := ` + tc.Name + `{}.FindReader(strings.NewReader(input), stream.Config{}, func(m stream.Match[*` + tc.Name + `BytesResult]) bool {
		streamMatches = append(streamMatches, string(m.Result.Match))
		return true
	})
	if err != nil {
		t.Fatalf("FindReader error: %v", err)
	}

	if len(streamMatches) != len(stdlibMatches) {
		t.Errorf("Count mismatch: stream=%d stdlib=%d", len(streamMatches), len(stdlibMatches))
		return
	}

	for i := range streamMatches {
		if streamMatches[i] != stdlibMatches[i] {
			t.Errorf("Match[%d] mismatch: stream=%q stdlib=%q", i, streamMatches[i], stdlibMatches[i])
		}
	}
}

func TestStreamingChunkBoundaries(t *testing.T) {
	input := ` + getSmallTestInput(tc) + `
	expected := stdlibRe.FindAllString(input, -1)

	// Test with various chunk sizes
	// Note: ChunkedReader simulates slow reads, but the streaming API uses its own
	// internal buffer (default 64KB). Use realistic chunk sizes.
	chunkSizes := []int{100, 500, 1000, 2000, 4000}
	for _, chunkSize := range chunkSizes {
		t.Run(bytesToString(chunkSize), func(t *testing.T) {
			reader := NewChunkedReader(strings.NewReader(input), chunkSize)

			var results []string
			err := ` + tc.Name + `{}.FindReader(reader, stream.Config{}, func(m stream.Match[*` + tc.Name + `BytesResult]) bool {
				results = append(results, string(m.Result.Match))
				return true
			})
			if err != nil {
				t.Fatalf("error: %v", err)
			}

			if len(results) != len(expected) {
				t.Errorf("count mismatch at chunkSize=%d: got %d, want %d", chunkSize, len(results), len(expected))
				return
			}
			for i := range results {
				if results[i] != expected[i] {
					t.Errorf("match[%d] mismatch at chunkSize=%d: got %q, want %q", i, chunkSize, results[i], expected[i])
				}
			}
		})
	}
}

func TestStreamingEarlyTermination(t *testing.T) {
	gen := ` + getGeneratorConstructor(tc) + `

	// Stop after 5 matches
	var count int
	err := ` + tc.Name + `{}.FindReader(gen, stream.Config{}, func(m stream.Match[*` + tc.Name + `BytesResult]) bool {
		count++
		return count < 5
	})
	if err != nil {
		t.Fatalf("FindReader error: %v", err)
	}

	if count != 5 {
		t.Errorf("Expected 5 matches before termination, got %d", count)
	}
}

func TestStreamingEmptyInput(t *testing.T) {
	count, err := ` + tc.Name + `{}.FindReaderCount(strings.NewReader(""), stream.Config{})
	if err != nil {
		t.Fatalf("FindReaderCount error: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 matches on empty input, got %d", count)
	}
}

func TestStreamingNoMatches(t *testing.T) {
	input := strings.Repeat("x", 10000)
	count, err := ` + tc.Name + `{}.FindReaderCount(strings.NewReader(input), stream.Config{})
	if err != nil {
		t.Fatalf("FindReaderCount error: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 matches, got %d", count)
	}
}

func TestStreamingOffsets(t *testing.T) {
	input := ` + getSmallTestInput(tc) + `

	// Get stdlib indices
	stdlibIndices := stdlibRe.FindAllStringIndex(input, -1)

	// Get streaming offsets
	var streamOffsets [][2]int64
	err := ` + tc.Name + `{}.FindReader(strings.NewReader(input), stream.Config{}, func(m stream.Match[*` + tc.Name + `BytesResult]) bool {
		streamOffsets = append(streamOffsets, [2]int64{
			m.StreamOffset,
			m.StreamOffset + int64(len(m.Result.Match)),
		})
		return true
	})
	if err != nil {
		t.Fatalf("FindReader error: %v", err)
	}

	if len(streamOffsets) != len(stdlibIndices) {
		t.Errorf("Count mismatch: stream=%d stdlib=%d", len(streamOffsets), len(stdlibIndices))
		return
	}

	for i := range streamOffsets {
		if streamOffsets[i][0] != int64(stdlibIndices[i][0]) || streamOffsets[i][1] != int64(stdlibIndices[i][1]) {
			t.Errorf("Offset[%d] mismatch: stream=%v stdlib=%v", i, streamOffsets[i], stdlibIndices[i])
		}
	}
}

func bytesToString(n int) string {
	return "chunk" + string(rune('0'+n%10))
}
`
}

// getGeneratorConstructor returns the constructor call for the pattern's generator.
func getGeneratorConstructor(tc StreamingTestCase) string {
	switch tc.Name {
	case "Date":
		return `NewPatternedReader("2024-01-15", "abcdefghijk \n\t", 50, 1<<16)`
	case "Email":
		return `NewPatternedReader("user@example.com", "abcdefghijk123 \n\t", 80, 1<<16)`
	case "IPv4":
		return `NewPatternedReader("192.168.1.100", "abcdefghijk \n\t", 40, 1<<16)`
	case "URL":
		return `NewPatternedReader("https://example.com/path", "abcdefghijk123 \n\t", 100, 1<<16)`
	default:
		return `NewPatternedReader("test", "abc", 20, 1<<16)`
	}
}

// getSmallTestInput returns a small test input string for detailed comparison.
func getSmallTestInput(tc StreamingTestCase) string {
	switch tc.Name {
	case "Date":
		return `"prefix 2024-01-15 middle 2024-02-20 suffix 2024-12-31 end"`
	case "Email":
		return `"hello user@example.com and test@domain.org bye admin@site.net end"`
	case "IPv4":
		return `"server 192.168.1.100 gateway 10.0.0.1 dns 8.8.8.8 end"`
	case "URL":
		return `"visit https://example.com and http://test.org/path done"`
	default:
		return `"test input"`
	}
}

// setupGoModule initializes go module in test directory.
func setupGoModule(t *testing.T, dir string) {
	t.Helper()

	// Get regengo module path
	regengoPath, err := getRegengoModulePath()
	if err != nil {
		t.Fatalf("Failed to get regengo module path: %v", err)
	}

	// Initialize module
	initCmd := exec.Command("go", "mod", "init", "testmodule")
	initCmd.Dir = dir
	if output, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod init failed:\nOutput: %s\nError: %v", string(output), err)
	}

	// Add replace directive
	editCmd := exec.Command("go", "mod", "edit", "-replace",
		"github.com/KromDaniel/regengo="+regengoPath)
	editCmd.Dir = dir
	if output, err := editCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod edit failed:\nOutput: %s\nError: %v", string(output), err)
	}

	// Run go mod tidy
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = dir
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed:\nOutput: %s\nError: %v", string(output), err)
	}
}

// getRegengoModulePath returns the absolute path to the regengo module root.
func getRegengoModulePath() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", io.EOF
	}
	dir := filepath.Dir(file)
	// Go up from e2e/streaming to repo root
	return filepath.Abs(filepath.Join(dir, "..", ".."))
}
