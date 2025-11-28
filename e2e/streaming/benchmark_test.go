package streaming

import (
	"bytes"
	"os"
	"regexp"
	"testing"

	"github.com/KromDaniel/regengo/e2e/streaming/testdata"
	stream "github.com/KromDaniel/regengo/stream"
)

// Benchmarks compare streaming vs in-memory performance.
// Run with: go test -bench=. -benchmem ./e2e/streaming/...

var stdlibRe = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)

func generateData(size int64) []byte {
	gen := NewPatternedReader("2024-01-15", 50, size)
	data := make([]byte, size)
	gen.Read(data)
	return data
}

func BenchmarkStreaming_1MB(b *testing.B) {
	if os.Getenv("RUN_BENCHMARKS") == "" {
		b.Skip("Skipping benchmark (set RUN_BENCHMARKS=1 to run)")
	}

	data := generateData(1 << 20)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		testdata.DatePattern{}.FindReaderCount(bytes.NewReader(data), stream.Config{})
	}
}

func BenchmarkInMemory_1MB(b *testing.B) {
	if os.Getenv("RUN_BENCHMARKS") == "" {
		b.Skip("Skipping benchmark (set RUN_BENCHMARKS=1 to run)")
	}

	data := generateData(1 << 20)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		testdata.DatePattern{}.FindAllBytes(data, -1)
	}
}

func BenchmarkStdlib_1MB(b *testing.B) {
	if os.Getenv("RUN_BENCHMARKS") == "" {
		b.Skip("Skipping benchmark (set RUN_BENCHMARKS=1 to run)")
	}

	data := generateData(1 << 20)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		stdlibRe.FindAll(data, -1)
	}
}

func BenchmarkStreamingBufferSize_16KB(b *testing.B) {
	if os.Getenv("RUN_BENCHMARKS") == "" {
		b.Skip("Skipping benchmark (set RUN_BENCHMARKS=1 to run)")
	}

	data := generateData(1 << 20)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		testdata.DatePattern{}.FindReaderCount(bytes.NewReader(data), stream.Config{BufferSize: 16 * 1024})
	}
}

func BenchmarkStreamingBufferSize_64KB(b *testing.B) {
	if os.Getenv("RUN_BENCHMARKS") == "" {
		b.Skip("Skipping benchmark (set RUN_BENCHMARKS=1 to run)")
	}

	data := generateData(1 << 20)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		testdata.DatePattern{}.FindReaderCount(bytes.NewReader(data), stream.Config{BufferSize: 64 * 1024})
	}
}

func BenchmarkStreamingCountOnly(b *testing.B) {
	if os.Getenv("RUN_BENCHMARKS") == "" {
		b.Skip("Skipping benchmark (set RUN_BENCHMARKS=1 to run)")
	}

	data := generateData(1 << 20)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		testdata.DatePattern{}.FindReaderCount(bytes.NewReader(data), stream.Config{})
	}
}

func BenchmarkStreamingFirstOnly(b *testing.B) {
	if os.Getenv("RUN_BENCHMARKS") == "" {
		b.Skip("Skipping benchmark (set RUN_BENCHMARKS=1 to run)")
	}

	data := generateData(1 << 20)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		testdata.DatePattern{}.FindReaderFirst(bytes.NewReader(data), stream.Config{})
	}
}
