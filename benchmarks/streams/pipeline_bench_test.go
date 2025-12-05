package streams

import (
	"io"
	"strings"
	"testing"

	"github.com/KromDaniel/regengo/benchmarks/streams/testdata"
	"github.com/KromDaniel/regengo/stream"
)

// generateMixedInput creates test input with both emails and IPs
func generateMixedInput(size int) string {
	var b strings.Builder
	items := []string{
		"Contact: user@example.com ",
		"Server: 192.168.1.1 ",
		"Email: admin@test.org ",
		"IP: 10.0.0.1 ",
	}
	i := 0
	for b.Len() < size {
		b.WriteString(items[i%len(items)])
		i++
	}
	return b.String()[:size]
}

// BenchmarkPipeline_TwoStage benchmarks a two-stage transform pipeline
func BenchmarkPipeline_TwoStage(b *testing.B) {
	input := generateMixedInput(1024 * 1024) // 1MB

	b.ReportAllocs()
	b.SetBytes(int64(len(input)))
	for b.Loop() {
		// First stage: redact emails
		r1 := testdata.CompiledEmailPattern.ReplaceReader(
			strings.NewReader(input),
			"[EMAIL]",
		)
		// Second stage: redact IPs
		r2 := testdata.CompiledIPv4Pattern.ReplaceReader(r1, "[IP]")

		_, _ = io.Copy(io.Discard, r2)
	}
}

// BenchmarkPipeline_ThreeStage benchmarks a three-stage transform pipeline
func BenchmarkPipeline_ThreeStage(b *testing.B) {
	input := generateMixedInput(1024 * 1024) // 1MB

	b.ReportAllocs()
	b.SetBytes(int64(len(input)))
	for b.Loop() {
		// Stage 1: redact emails
		r1 := testdata.CompiledEmailPattern.ReplaceReader(
			strings.NewReader(input),
			"[EMAIL]",
		)
		// Stage 2: redact IPs
		r2 := testdata.CompiledIPv4Pattern.ReplaceReader(r1, "[IP]")
		// Stage 3: filter lines (using LineFilter for third stage)
		r3 := stream.LineFilter(r2, func(line []byte) bool {
			return len(line) > 0 // keep non-empty lines
		})

		_, _ = io.Copy(io.Discard, r3)
	}
}

// BenchmarkLineFilter benchmarks LineFilter operations
func BenchmarkLineFilter(b *testing.B) {
	input := generateMixedInput(1024 * 1024) // 1MB

	b.Run("KeepAll", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(input)))
		for b.Loop() {
			r := stream.LineFilter(strings.NewReader(input), func(line []byte) bool {
				return true
			})
			_, _ = io.Copy(io.Discard, r)
		}
	})

	b.Run("KeepMatching", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(input)))
		for b.Loop() {
			r := stream.LineFilter(strings.NewReader(input), func(line []byte) bool {
				return strings.Contains(string(line), "Email")
			})
			_, _ = io.Copy(io.Discard, r)
		}
	})
}

// BenchmarkLineTransform benchmarks LineTransform operations
func BenchmarkLineTransform(b *testing.B) {
	input := generateMixedInput(1024 * 1024) // 1MB

	b.Run("PassThrough", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(input)))
		for b.Loop() {
			r := stream.LineTransform(strings.NewReader(input), func(line []byte) []byte {
				return line
			})
			_, _ = io.Copy(io.Discard, r)
		}
	})

	b.Run("Uppercase", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(input)))
		for b.Loop() {
			r := stream.LineTransform(strings.NewReader(input), func(line []byte) []byte {
				return []byte(strings.ToUpper(string(line)))
			})
			_, _ = io.Copy(io.Discard, r)
		}
	})
}

// BenchmarkMixedPipeline benchmarks mixed transform and line operations
func BenchmarkMixedPipeline(b *testing.B) {
	input := generateMixedInput(1024 * 1024) // 1MB

	b.Run("Transform_LineFilter_Transform", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(input)))
		for b.Loop() {
			// Stage 1: redact emails
			r1 := testdata.CompiledEmailPattern.ReplaceReader(
				strings.NewReader(input),
				"[EMAIL]",
			)
			// Stage 2: filter lines containing "Server"
			r2 := stream.LineFilter(r1, func(line []byte) bool {
				return strings.Contains(string(line), "Server")
			})
			// Stage 3: redact IPs
			r3 := testdata.CompiledIPv4Pattern.ReplaceReader(r2, "[IP]")

			_, _ = io.Copy(io.Discard, r3)
		}
	})

	b.Run("LineTransform_Transform", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(input)))
		for b.Loop() {
			// Stage 1: uppercase all lines
			r1 := stream.LineTransform(strings.NewReader(input), func(line []byte) []byte {
				return []byte(strings.ToUpper(string(line)))
			})
			// Stage 2: redact emails (will match uppercase versions)
			r2 := testdata.CompiledEmailPattern.ReplaceReader(r1, "[EMAIL]")

			_, _ = io.Copy(io.Discard, r2)
		}
	})
}

// BenchmarkPipelineDepth benchmarks pipelines of increasing depth
func BenchmarkPipelineDepth(b *testing.B) {
	input := generateMixedInput(1024 * 1024) // 1MB

	depths := []int{1, 2, 3, 4, 5}

	for _, depth := range depths {
		b.Run(strings.Repeat("Stage_", depth)[:len("Stage_")+depth-1], func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(input)))
			for b.Loop() {
				var r io.Reader = strings.NewReader(input)
				for i := 0; i < depth; i++ {
					if i%2 == 0 {
						r = testdata.CompiledEmailPattern.ReplaceReader(r, "[EMAIL]")
					} else {
						r = testdata.CompiledIPv4Pattern.ReplaceReader(r, "[IP]")
					}
				}
				_, _ = io.Copy(io.Discard, r)
			}
		})
	}
}
