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

// TestMemoryConstant verifies that streaming uses constant memory regardless of input size.
// This is a critical property of the streaming API.
func TestMemoryConstant(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory tests in short mode")
	}

	tempDir := t.TempDir()
	caseDir := filepath.Join(tempDir, "memory_test")
	if err := os.MkdirAll(caseDir, 0755); err != nil {
		t.Fatalf("Failed to create case dir: %v", err)
	}

	// Pattern MUST have capture groups because streaming methods
	// are only generated for patterns with captures (WithCaptures=true).
	pattern := `(\d{4}-\d{2}-\d{2})`
	patternName := "MemoryTest"
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

	// Generate memory test file
	testContent := generateMemoryTestFile(patternName)
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Setup go module
	setupMemoryGoModule(t, caseDir)

	// Run tests
	cmd := exec.Command("go", "test", "-v", "-count=1")
	cmd.Dir = caseDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Memory tests failed:\nOutput: %s\nError: %v", string(output), err)
	}

	t.Logf("Memory test output:\n%s", string(output))
}

func generateMemoryTestFile(patternName string) string {
	return fmt.Sprintf(`package generated

import (
	"io"
	"math/rand"
	"runtime"
	"testing"

	stream "github.com/KromDaniel/regengo/stream"
)

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
		posInCycle := int(r.pos) %% r.matchEvery
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

func (r *PatternedReader) ExpectedMatches() int64 {
	return r.limit / int64(r.matchEvery)
}

func TestMemoryBounded(t *testing.T) {
	// Process 64MB of data
	dataSize := int64(64 << 20)
	gen := NewPatternedReader("2024-01-15", "abcdefghijk \n\t", 50, dataSize)

	// Force GC and get baseline
	runtime.GC()
	var startMem runtime.MemStats
	runtime.ReadMemStats(&startMem)
	startAlloc := startMem.TotalAlloc

	// Process stream
	var count int64
	err := %s{}.FindReader(gen, stream.Config{
		BufferSize: 1 << 16, // 64KB buffer
	}, func(m stream.Match[*%sBytesResult]) bool {
		count++
		return true
	})
	if err != nil {
		t.Fatalf("FindReader error: %%v", err)
	}

	// Check memory usage
	runtime.GC()
	var endMem runtime.MemStats
	runtime.ReadMemStats(&endMem)
	totalAlloc := endMem.TotalAlloc - startAlloc

	// Memory should be bounded - we expect roughly buffer size + overhead
	// Allow up to 10MB total allocations for 64MB of data processing
	maxExpected := uint64(10 << 20)
	if totalAlloc > maxExpected {
		t.Errorf("Memory allocation too high: %%d bytes (max expected %%d)", totalAlloc, maxExpected)
	}

	t.Logf("Processed %%d bytes, found %%d matches", dataSize, count)
	t.Logf("Total allocations: %%d bytes (%%0.2f%% of data size)", totalAlloc, float64(totalAlloc)/float64(dataSize)*100)

	// Verify match count
	expected := gen.ExpectedMatches()
	if count != expected {
		t.Errorf("Match count = %%d, expected %%d", count, expected)
	}
}

func TestMemoryDoesNotGrowWithInputSize(t *testing.T) {
	sizes := []int64{
		1 << 16,  // 64KB
		1 << 18,  // 256KB
		1 << 20,  // 1MB
		1 << 22,  // 4MB
	}

	var allocations []uint64

	for _, size := range sizes {
		gen := NewPatternedReader("2024-01-15", "abcdefghijk \n\t", 50, size)

		runtime.GC()
		var startMem runtime.MemStats
		runtime.ReadMemStats(&startMem)
		startAlloc := startMem.TotalAlloc

		_, err := %s{}.FindReaderCount(gen, stream.Config{
			BufferSize: 1 << 14, // 16KB buffer
		})
		if err != nil {
			t.Fatalf("FindReaderCount error for size %%d: %%v", size, err)
		}

		runtime.GC()
		var endMem runtime.MemStats
		runtime.ReadMemStats(&endMem)
		alloc := endMem.TotalAlloc - startAlloc
		allocations = append(allocations, alloc)

		t.Logf("Size %%d: allocated %%d bytes", size, alloc)
	}

	// Memory should be roughly constant regardless of input size
	// Allow 2x variance due to GC timing
	baseline := allocations[0]
	maxAllowed := baseline * 4

	for i, alloc := range allocations {
		if alloc > maxAllowed {
			t.Errorf("Size %%d: allocated %%d bytes, expected < %%d (baseline %%d)",
				sizes[i], alloc, maxAllowed, baseline)
		}
	}
}

func TestNoLeaksOnEarlyTermination(t *testing.T) {
	// Large data size
	dataSize := int64(10 << 20) // 10MB
	gen := NewPatternedReader("2024-01-15", "abcdefghijk \n\t", 50, dataSize)

	runtime.GC()
	var startMem runtime.MemStats
	runtime.ReadMemStats(&startMem)
	startAlloc := startMem.TotalAlloc

	// Process but stop early
	var count int64
	%s{}.FindReader(gen, stream.Config{}, func(m stream.Match[*%sBytesResult]) bool {
		count++
		return count < 100 // Stop after 100 matches
	})

	runtime.GC()
	var endMem runtime.MemStats
	runtime.ReadMemStats(&endMem)
	totalAlloc := endMem.TotalAlloc - startAlloc

	// Should have minimal memory usage since we stopped early
	maxExpected := uint64(1 << 20) // 1MB
	if totalAlloc > maxExpected {
		t.Errorf("Memory allocation on early termination too high: %%d bytes", totalAlloc)
	}

	if count != 100 {
		t.Errorf("Expected 100 matches, got %%d", count)
	}

	t.Logf("Early termination: allocated %%d bytes for %%d matches", totalAlloc, count)
}
`, patternName, patternName, patternName, patternName, patternName)
}

func setupMemoryGoModule(t *testing.T, dir string) {
	t.Helper()

	regengoPath, err := getMemoryRegengoPath()
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

func getMemoryRegengoPath() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", io.EOF
	}
	dir := filepath.Dir(file)
	return filepath.Abs(filepath.Join(dir, "..", ".."))
}
