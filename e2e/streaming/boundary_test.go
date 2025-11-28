package streaming

import (
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/KromDaniel/regengo/e2e/streaming/testdata"
	stream "github.com/KromDaniel/regengo/stream"
)

// ChunkedReader wraps a reader and returns data in fixed-size chunks.
// This helps test boundary conditions where matches may span read boundaries.
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

// TestStreamingChunkBoundaries tests that matches spanning chunk boundaries are handled correctly.
func TestStreamingChunkBoundaries(t *testing.T) {
	input := "prefix 2024-01-15 middle 2024-02-20 suffix 2024-12-31 end"
	stdlibRe := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
	expected := stdlibRe.FindAllString(input, -1)

	// Test with various chunk sizes
	chunkSizes := []int{100, 500, 1000, 2000, 4000}
	for _, chunkSize := range chunkSizes {
		t.Run(formatChunkSize(chunkSize), func(t *testing.T) {
			reader := NewChunkedReader(strings.NewReader(input), chunkSize)

			var results []string
			err := testdata.DatePattern{}.FindReader(
				reader,
				stream.Config{},
				func(m stream.Match[*testdata.DatePatternBytesResult]) bool {
					results = append(results, string(m.Result.Match))
					return true
				},
			)
			if err != nil {
				t.Fatalf("FindReader error: %v", err)
			}

			if len(results) != len(expected) {
				t.Errorf("Count mismatch at chunkSize=%d: got %d, want %d",
					chunkSize, len(results), len(expected))
				return
			}
			for i := range results {
				if results[i] != expected[i] {
					t.Errorf("Match[%d] mismatch at chunkSize=%d: got %q, want %q",
						i, chunkSize, results[i], expected[i])
				}
			}
		})
	}
}

// TestStreamingLargeInput tests streaming with larger inputs (without chunking).
func TestStreamingLargeInput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large input test in short mode")
	}

	// Generate input with many matches
	var sb strings.Builder
	for i := 0; i < 1000; i++ {
		sb.WriteString("noise 2024-01-15 more noise ")
	}
	input := sb.String()

	stdlibRe := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
	expected := int64(len(stdlibRe.FindAllString(input, -1)))

	// Test without artificial chunking - uses default buffer
	count, err := testdata.DatePattern{}.FindReaderCount(strings.NewReader(input), stream.Config{})
	if err != nil {
		t.Fatalf("FindReaderCount error: %v", err)
	}

	if count != expected {
		t.Errorf("Count: got %d, want %d", count, expected)
	}
}

// TestStreamingDigitsBoundary tests digit pattern with various chunk sizes.
func TestStreamingDigitsBoundary(t *testing.T) {
	input := "abc123def456789ghi0jkl12345"
	stdlibRe := regexp.MustCompile(`(\d+)`)
	expected := stdlibRe.FindAllString(input, -1)

	chunkSizes := []int{100, 200, 500}
	for _, chunkSize := range chunkSizes {
		t.Run(formatChunkSize(chunkSize), func(t *testing.T) {
			reader := NewChunkedReader(strings.NewReader(input), chunkSize)

			var results []string
			err := testdata.DigitsPattern{}.FindReader(
				reader,
				stream.Config{},
				func(m stream.Match[*testdata.DigitsPatternBytesResult]) bool {
					results = append(results, string(m.Result.Match))
					return true
				},
			)
			if err != nil {
				t.Fatalf("FindReader error: %v", err)
			}

			if len(results) != len(expected) {
				t.Errorf("Count mismatch: got %d, want %d", len(results), len(expected))
				return
			}
			for i := range results {
				if results[i] != expected[i] {
					t.Errorf("Match[%d]: got %q, want %q", i, results[i], expected[i])
				}
			}
		})
	}
}

func formatChunkSize(n int) string {
	if n >= 1024 {
		return string(rune('0'+n/1024)) + "KB"
	}
	return string(rune('0'+n/100)) + "00B"
}
