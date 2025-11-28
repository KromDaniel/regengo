package streaming

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/KromDaniel/regengo/pkg/regengo"
)

// TestFuzzStreamingBoundary sets up a fuzz test that compares streaming with in-memory results.
// To run: go test -fuzz=FuzzStreamingBoundary ./e2e/streaming/...
func TestFuzzStreamingBoundary(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping fuzz test setup in short mode")
	}

	// Only run if -fuzz flag would be passed
	if os.Getenv("RUN_FUZZ") == "" {
		t.Skip("Skipping fuzz test (set RUN_FUZZ=1 to run)")
	}

	tempDir := t.TempDir()
	caseDir := filepath.Join(tempDir, "fuzz_test")
	if err := os.MkdirAll(caseDir, 0755); err != nil {
		t.Fatalf("Failed to create case dir: %v", err)
	}

	// Use a simple pattern for fuzz testing
	// Pattern MUST have capture groups because streaming methods
	// are only generated for patterns with captures (WithCaptures=true).
	pattern := `(\d+)`
	patternName := "FuzzPattern"
	outputFile := filepath.Join(caseDir, patternName+".go")
	testFile := filepath.Join(caseDir, patternName+"_test.go")

	// Generate code
	opts := regengo.Options{
		Pattern:          pattern,
		Name:             patternName,
		OutputFile:       outputFile,
		Package:          "generated",
		GenerateTestFile: false,
	}
	if err := regengo.Compile(opts); err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Generate fuzz test file
	testContent := generateFuzzTestFile(patternName, pattern)
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Setup go module
	setupFuzzGoModule(t, caseDir)

	// Run a quick test to verify the fuzz test compiles
	cmd := exec.Command("go", "test", "-v", "-run=TestFuzzSanity", "-count=1")
	cmd.Dir = caseDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Fuzz test setup failed:\nOutput: %s\nError: %v", string(output), err)
	}

	t.Logf("Fuzz test setup verified:\n%s", string(output))
	t.Logf("To run fuzz tests: cd %s && go test -fuzz=FuzzStreamingBoundary -fuzztime=10s", caseDir)
}

func generateFuzzTestFile(patternName, pattern string) string {
	return fmt.Sprintf(`package generated

import (
	"bytes"
	"io"
	"reflect"
	"regexp"
	"testing"

	stream "github.com/KromDaniel/regengo/stream"
)

var stdlibRe = regexp.MustCompile(%q)

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

// TestFuzzSanity is a quick sanity test to verify fuzz infrastructure works
func TestFuzzSanity(t *testing.T) {
	testCases := []struct {
		data      []byte
		chunkSize int
	}{
		{[]byte("abc123def456"), 4},
		{[]byte("xxxx"), 2},
		{[]byte("12345"), 1},
		{[]byte(""), 1},
		{[]byte("no matches here"), 10},
	}

	for _, tc := range testCases {
		verifyStreamingMatchesInMemory(t, tc.data, tc.chunkSize)
	}
}

// FuzzStreamingBoundary fuzzes the streaming implementation
// Run with: go test -fuzz=FuzzStreamingBoundary -fuzztime=30s
func FuzzStreamingBoundary(f *testing.F) {
	// Seed corpus
	f.Add([]byte("abc123def456"), 4)
	f.Add([]byte("xxxx"), 2)
	f.Add([]byte("12345"), 1)
	f.Add([]byte(""), 1)
	f.Add([]byte("no matches here"), 10)
	f.Add([]byte("1"), 1)
	f.Add([]byte("12"), 1)
	f.Add([]byte("123456789012345"), 3)
	f.Add([]byte("a1b2c3d4e5f6g7h8i9j0"), 5)

	f.Fuzz(func(t *testing.T, data []byte, chunkSize int) {
		// Normalize chunkSize
		if chunkSize < 1 {
			chunkSize = 1
		}
		if chunkSize > len(data)+1 {
			chunkSize = len(data) + 1
		}
		if chunkSize > 1000 {
			chunkSize = 1000
		}

		verifyStreamingMatchesInMemory(t, data, chunkSize)
	})
}

func verifyStreamingMatchesInMemory(t testing.TB, data []byte, chunkSize int) {
	// Get in-memory results
	memMatches := %s{}.FindAllBytes(data, -1)
	var memStrings []string
	for _, m := range memMatches {
		memStrings = append(memStrings, string(m.Match))
	}

	// Get streaming results with chunked reader
	reader := NewChunkedReader(bytes.NewReader(data), chunkSize)
	var streamMatches []string
	err := %s{}.FindReader(reader, stream.Config{
		BufferSize: maxInt(chunkSize*2, 64),
	}, func(m stream.Match[*%sBytesResult]) bool {
		streamMatches = append(streamMatches, string(m.Result.Match))
		return true
	})
	if err != nil {
		t.Errorf("FindReader error: %%v", err)
		return
	}

	// Compare results
	if !reflect.DeepEqual(streamMatches, memStrings) {
		t.Errorf("Mismatch for data=%%q chunkSize=%%d:\nstream=%%v\nmem=%%v",
			truncate(data, 100), chunkSize, streamMatches, memStrings)
	}
}

// FuzzStreamingOffset verifies that stream offsets are correct
func FuzzStreamingOffset(f *testing.F) {
	f.Add([]byte("abc123def456"), 4)
	f.Add([]byte("12345"), 1)

	f.Fuzz(func(t *testing.T, data []byte, chunkSize int) {
		if chunkSize < 1 {
			chunkSize = 1
		}
		if chunkSize > len(data)+1 {
			chunkSize = len(data) + 1
		}

		// Get stdlib indices
		stdlibIndices := stdlibRe.FindAllIndex(data, -1)

		// Get streaming offsets
		reader := NewChunkedReader(bytes.NewReader(data), chunkSize)
		var streamOffsets [][2]int
		err := %s{}.FindReader(reader, stream.Config{
			BufferSize: maxInt(chunkSize*2, 64),
		}, func(m stream.Match[*%sBytesResult]) bool {
			streamOffsets = append(streamOffsets, [2]int{
				int(m.StreamOffset),
				int(m.StreamOffset) + len(m.Result.Match),
			})
			return true
		})
		if err != nil {
			t.Errorf("FindReader error: %%v", err)
			return
		}

		// Compare
		if len(streamOffsets) != len(stdlibIndices) {
			t.Errorf("Count mismatch for data=%%q chunkSize=%%d: stream=%%d stdlib=%%d",
				truncate(data, 50), chunkSize, len(streamOffsets), len(stdlibIndices))
			return
		}

		for i := range streamOffsets {
			if streamOffsets[i][0] != stdlibIndices[i][0] || streamOffsets[i][1] != stdlibIndices[i][1] {
				t.Errorf("Offset mismatch at %%d for data=%%q chunkSize=%%d: stream=%%v stdlib=%%v",
					i, truncate(data, 50), chunkSize, streamOffsets[i], stdlibIndices[i])
			}
		}
	})
}

func truncate(data []byte, n int) []byte {
	if len(data) <= n {
		return data
	}
	return append(data[:n], []byte("...")...)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
`, pattern, patternName, patternName, patternName, patternName, patternName)
}

func setupFuzzGoModule(t *testing.T, dir string) {
	t.Helper()

	regengoPath, err := getFuzzRegengoPath()
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

func getFuzzRegengoPath() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", io.EOF
	}
	dir := filepath.Dir(file)
	return filepath.Abs(filepath.Join(dir, "..", ".."))
}
