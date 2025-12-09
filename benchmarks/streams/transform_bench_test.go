package streams

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/KromDaniel/regengo/benchmarks/streams/testdata"
	"github.com/KromDaniel/regengo/stream"
)

// generateInput creates test input of specified size with embedded emails
func generateInput(size int) string {
	var b strings.Builder
	email := "test@example.com "
	for b.Len() < size {
		b.WriteString(email)
	}
	return b.String()[:size]
}

// generateInputWithDensity creates input with specified match density
func generateInputWithDensity(size int, density float64) string {
	var b strings.Builder
	email := "test@example.com"
	padding := "xxxxxxxxxxxxxxxxxxxx" // 20 non-matching chars

	matchEvery := int(float64(len(email)+len(padding)) / density)
	if matchEvery < len(email) {
		matchEvery = len(email)
	}

	for b.Len() < size {
		b.WriteString(email)
		paddingNeeded := matchEvery - len(email)
		for i := 0; i < paddingNeeded && b.Len() < size; i++ {
			b.WriteByte(padding[i%len(padding)])
		}
	}
	return b.String()[:size]
}

// BenchmarkReplaceReader benchmarks ReplaceReader at various input sizes
func BenchmarkReplaceReader(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
		{"1MB", 1024 * 1024},
		{"10MB", 10 * 1024 * 1024},
	}

	for _, s := range sizes {
		input := generateInput(s.size)
		b.Run(s.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(s.size))
			for b.Loop() {
				r := testdata.CompiledEmailPattern.ReplaceReader(
					strings.NewReader(input),
					"[REDACTED]",
				)
				_, _ = io.Copy(io.Discard, r)
			}
		})
	}
}

// BenchmarkSelectReader benchmarks SelectReader filtering
func BenchmarkSelectReader(b *testing.B) {
	input := generateInput(1024 * 1024) // 1MB

	b.Run("KeepAll", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(input)))
		for b.Loop() {
			r := testdata.CompiledEmailPattern.SelectReader(
				strings.NewReader(input),
				func(m *testdata.EmailPatternBytesResult) bool {
					return true
				},
			)
			_, _ = io.Copy(io.Discard, r)
		}
	})

	b.Run("KeepHalf", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(input)))
		i := 0
		for b.Loop() {
			r := testdata.CompiledEmailPattern.SelectReader(
				strings.NewReader(input),
				func(m *testdata.EmailPatternBytesResult) bool {
					i++
					return i%2 == 0
				},
			)
			_, _ = io.Copy(io.Discard, r)
		}
	})

	b.Run("FilterByCapture", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(input)))
		for b.Loop() {
			r := testdata.CompiledEmailPattern.SelectReader(
				strings.NewReader(input),
				func(m *testdata.EmailPatternBytesResult) bool {
					return bytes.Contains(m.Domain, []byte("example"))
				},
			)
			_, _ = io.Copy(io.Discard, r)
		}
	})
}

// BenchmarkRejectReader benchmarks RejectReader filtering
func BenchmarkRejectReader(b *testing.B) {
	input := generateInput(1024 * 1024) // 1MB

	b.Run("RejectAll", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(input)))
		for b.Loop() {
			r := testdata.CompiledEmailPattern.RejectReader(
				strings.NewReader(input),
				func(m *testdata.EmailPatternBytesResult) bool {
					return true
				},
			)
			_, _ = io.Copy(io.Discard, r)
		}
	})

	b.Run("RejectNone", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(input)))
		for b.Loop() {
			r := testdata.CompiledEmailPattern.RejectReader(
				strings.NewReader(input),
				func(m *testdata.EmailPatternBytesResult) bool {
					return false
				},
			)
			_, _ = io.Copy(io.Discard, r)
		}
	})
}

// BenchmarkNewTransformReader benchmarks the low-level transform API
func BenchmarkNewTransformReader(b *testing.B) {
	input := generateInput(1024 * 1024) // 1MB
	cfg := stream.DefaultTransformConfig()

	b.Run("Drop", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(input)))
		for b.Loop() {
			r := testdata.CompiledEmailPattern.NewTransformReader(
				strings.NewReader(input),
				cfg,
				func(m *testdata.EmailPatternBytesResult, emit func([]byte)) {
					// Drop all matches
				},
			)
			_, _ = io.Copy(io.Discard, r)
		}
	})

	b.Run("PassThrough", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(input)))
		for b.Loop() {
			r := testdata.CompiledEmailPattern.NewTransformReader(
				strings.NewReader(input),
				cfg,
				func(m *testdata.EmailPatternBytesResult, emit func([]byte)) {
					emit(m.Match)
				},
			)
			_, _ = io.Copy(io.Discard, r)
		}
	})

	b.Run("MultiEmit", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(input)))
		for b.Loop() {
			r := testdata.CompiledEmailPattern.NewTransformReader(
				strings.NewReader(input),
				cfg,
				func(m *testdata.EmailPatternBytesResult, emit func([]byte)) {
					emit(m.Match)
					emit([]byte("|"))
					emit(m.Match)
				},
			)
			_, _ = io.Copy(io.Discard, r)
		}
	})
}

// BenchmarkBufferSizes benchmarks different buffer sizes
func BenchmarkBufferSizes(b *testing.B) {
	input := generateInput(1024 * 1024) // 1MB

	bufferSizes := []struct {
		name string
		size int
	}{
		{"256B", 256},
		{"1KB", 1024},
		{"4KB", 4 * 1024},
		{"16KB", 16 * 1024},
		{"64KB", 64 * 1024},
		{"256KB", 256 * 1024},
	}

	for _, bs := range bufferSizes {
		b.Run(bs.name, func(b *testing.B) {
			cfg := stream.DefaultTransformConfig()
			cfg.BufferSize = bs.size
			b.ReportAllocs()
			b.SetBytes(int64(len(input)))
			for b.Loop() {
				r := testdata.CompiledEmailPattern.NewTransformReader(
					strings.NewReader(input),
					cfg,
					func(m *testdata.EmailPatternBytesResult, emit func([]byte)) {
						emit([]byte("[X]"))
					},
				)
				_, _ = io.Copy(io.Discard, r)
			}
		})
	}
}

// BenchmarkMatchDensity benchmarks performance with different match densities
func BenchmarkMatchDensity(b *testing.B) {
	densities := []struct {
		name    string
		density float64
	}{
		{"Low_10pct", 0.1},
		{"Medium_50pct", 0.5},
		{"High_90pct", 0.9},
	}

	for _, d := range densities {
		input := generateInputWithDensity(1024*1024, d.density)
		b.Run(d.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(input)))
			for b.Loop() {
				r := testdata.CompiledEmailPattern.ReplaceReader(
					strings.NewReader(input),
					"[X]",
				)
				_, _ = io.Copy(io.Discard, r)
			}
		})
	}
}
