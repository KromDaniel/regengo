package streaming

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/KromDaniel/regengo/pkg/regengo"
)

// TestStreamingBenchmarks generates and runs benchmarks comparing streaming vs in-memory matching.
func TestStreamingBenchmarks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping benchmarks in short mode")
	}

	// Only run if -bench flag is passed
	if os.Getenv("RUN_BENCHMARKS") == "" {
		t.Skip("Skipping benchmarks (set RUN_BENCHMARKS=1 to run)")
	}

	tempDir := t.TempDir()
	caseDir := filepath.Join(tempDir, "benchmarks")
	if err := os.MkdirAll(caseDir, 0755); err != nil {
		t.Fatalf("Failed to create case dir: %v", err)
	}

	// Pattern MUST have capture groups because streaming methods
	// are only generated for patterns with captures (WithCaptures=true).
	pattern := `(\d{4}-\d{2}-\d{2})`
	patternName := "BenchmarkPattern"
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

	// Generate benchmark test file
	testContent := generateBenchmarkTestFile(patternName)
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Setup go module
	setupBenchmarkGoModule(t, caseDir)

	// Run benchmarks
	cmd := exec.Command("go", "test", "-bench=.", "-benchmem", "-benchtime=1s")
	cmd.Dir = caseDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Benchmarks failed:\nOutput: %s\nError: %v", string(output), err)
	}

	t.Logf("Benchmark results:\n%s", string(output))
}

func generateBenchmarkTestFile(patternName string) string {
	return fmt.Sprintf(`package generated

import (
	"bytes"
	"io"
	"math/rand"
	"regexp"
	"testing"

	stream "github.com/KromDaniel/regengo/stream"
)

var stdlibRe = regexp.MustCompile("\\d{4}-\\d{2}-\\d{2}")

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

func (r *PatternedReader) Reset() {
	r.pos = 0
	r.rng = rand.New(rand.NewSource(42))
}

func generateData(size int64) []byte {
	gen := NewPatternedReader("2024-01-15", "abcdefghijk \n\t", 50, size)
	data := make([]byte, size)
	gen.Read(data)
	return data
}

// Benchmark streaming vs in-memory at various sizes

func BenchmarkStreaming_64KB(b *testing.B) {
	data := generateData(64 * 1024)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		%s{}.FindReaderCount(bytes.NewReader(data), stream.Config{})
	}
}

func BenchmarkInMemory_64KB(b *testing.B) {
	data := generateData(64 * 1024)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		%s{}.FindAllBytes(data, -1)
	}
}

func BenchmarkStdlib_64KB(b *testing.B) {
	data := generateData(64 * 1024)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		stdlibRe.FindAll(data, -1)
	}
}

func BenchmarkStreaming_1MB(b *testing.B) {
	data := generateData(1 << 20)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		%s{}.FindReaderCount(bytes.NewReader(data), stream.Config{})
	}
}

func BenchmarkInMemory_1MB(b *testing.B) {
	data := generateData(1 << 20)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		%s{}.FindAllBytes(data, -1)
	}
}

func BenchmarkStdlib_1MB(b *testing.B) {
	data := generateData(1 << 20)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		stdlibRe.FindAll(data, -1)
	}
}

func BenchmarkStreaming_16MB(b *testing.B) {
	data := generateData(16 << 20)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		%s{}.FindReaderCount(bytes.NewReader(data), stream.Config{})
	}
}

func BenchmarkInMemory_16MB(b *testing.B) {
	data := generateData(16 << 20)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		%s{}.FindAllBytes(data, -1)
	}
}

func BenchmarkStdlib_16MB(b *testing.B) {
	data := generateData(16 << 20)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		stdlibRe.FindAll(data, -1)
	}
}

// Benchmark different buffer sizes

func BenchmarkStreamingBufferSize_4KB(b *testing.B) {
	data := generateData(1 << 20)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		%s{}.FindReaderCount(bytes.NewReader(data), stream.Config{BufferSize: 4 * 1024})
	}
}

func BenchmarkStreamingBufferSize_16KB(b *testing.B) {
	data := generateData(1 << 20)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		%s{}.FindReaderCount(bytes.NewReader(data), stream.Config{BufferSize: 16 * 1024})
	}
}

func BenchmarkStreamingBufferSize_64KB(b *testing.B) {
	data := generateData(1 << 20)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		%s{}.FindReaderCount(bytes.NewReader(data), stream.Config{BufferSize: 64 * 1024})
	}
}

func BenchmarkStreamingBufferSize_256KB(b *testing.B) {
	data := generateData(1 << 20)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		%s{}.FindReaderCount(bytes.NewReader(data), stream.Config{BufferSize: 256 * 1024})
	}
}

func BenchmarkStreamingBufferSize_1MB(b *testing.B) {
	data := generateData(1 << 20)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		%s{}.FindReaderCount(bytes.NewReader(data), stream.Config{BufferSize: 1024 * 1024})
	}
}

// Benchmark callback overhead

func BenchmarkStreamingWithCallback(b *testing.B) {
	data := generateData(1 << 20)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var count int64
		%s{}.FindReader(bytes.NewReader(data), stream.Config{}, func(m stream.Match[*%sBytesResult]) bool {
			count++
			return true
		})
	}
}

func BenchmarkStreamingCountOnly(b *testing.B) {
	data := generateData(1 << 20)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		%s{}.FindReaderCount(bytes.NewReader(data), stream.Config{})
	}
}

func BenchmarkStreamingFirstOnly(b *testing.B) {
	data := generateData(1 << 20)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		%s{}.FindReaderFirst(bytes.NewReader(data), stream.Config{})
	}
}
`, patternName, patternName, patternName, patternName, patternName, patternName, patternName, patternName, patternName, patternName, patternName, patternName, patternName, patternName, patternName)
}

func setupBenchmarkGoModule(t *testing.T, dir string) {
	t.Helper()

	regengoPath, err := getBenchmarkRegengoPath()
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

func getBenchmarkRegengoPath() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", io.EOF
	}
	dir := filepath.Dir(file)
	return filepath.Abs(filepath.Join(dir, "..", ".."))
}

// Unused helper to satisfy imports
var _ = bytes.NewReader
