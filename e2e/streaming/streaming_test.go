package streaming

import (
	"regexp"
	"strings"
	"testing"

	"github.com/KromDaniel/regengo/e2e/streaming/testdata"
	stream "github.com/KromDaniel/regengo/stream"
)

// TestStreamingVsStdlib compares streaming results with stdlib regexp results
// using pre-generated patterns.
func TestStreamingVsStdlib(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		input   string
	}{
		{
			name:    "Date",
			pattern: `(\d{4}-\d{2}-\d{2})`,
			input:   "Log: 2024-01-15 event, 2024-02-20 another, 2024-12-31 final",
		},
		{
			name:    "Email",
			pattern: `([\w.+-]+@[\w.-]+\.\w+)`,
			input:   "Contact user@example.com and test@domain.org for help",
		},
		{
			name:    "IPv4",
			pattern: `(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})`,
			input:   "Server 192.168.1.100 gateway 10.0.0.1 dns 8.8.8.8",
		},
		{
			name:    "Digits",
			pattern: `(\d+)`,
			input:   "abc123def456ghi789",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stdlibRe := regexp.MustCompile(tc.pattern)
			expected := stdlibRe.FindAllString(tc.input, -1)

			var results []string
			var err error

			switch tc.name {
			case "Date":
				err = testdata.DatePattern{}.FindReader(
					strings.NewReader(tc.input),
					stream.Config{},
					func(m stream.Match[*testdata.DatePatternBytesResult]) bool {
						results = append(results, string(m.Result.Match))
						return true
					},
				)
			case "Email":
				err = testdata.EmailPattern{}.FindReader(
					strings.NewReader(tc.input),
					stream.Config{},
					func(m stream.Match[*testdata.EmailPatternBytesResult]) bool {
						results = append(results, string(m.Result.Match))
						return true
					},
				)
			case "IPv4":
				err = testdata.IPv4Pattern{}.FindReader(
					strings.NewReader(tc.input),
					stream.Config{},
					func(m stream.Match[*testdata.IPv4PatternBytesResult]) bool {
						results = append(results, string(m.Result.Match))
						return true
					},
				)
			case "Digits":
				err = testdata.DigitsPattern{}.FindReader(
					strings.NewReader(tc.input),
					stream.Config{},
					func(m stream.Match[*testdata.DigitsPatternBytesResult]) bool {
						results = append(results, string(m.Result.Match))
						return true
					},
				)
			}

			if err != nil {
				t.Fatalf("FindReader error: %v", err)
			}

			if len(results) != len(expected) {
				t.Errorf("Count mismatch: got %d, want %d", len(results), len(expected))
				return
			}

			for i := range results {
				if results[i] != expected[i] {
					t.Errorf("Match[%d] mismatch: got %q, want %q", i, results[i], expected[i])
				}
			}
		})
	}
}

// TestStreamingCount verifies FindReaderCount matches stdlib count.
func TestStreamingCount(t *testing.T) {
	input := "2024-01-01 2024-02-02 2024-03-03 2024-04-04 2024-05-05"
	stdlibRe := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
	expected := int64(len(stdlibRe.FindAllString(input, -1)))

	count, err := testdata.DatePattern{}.FindReaderCount(strings.NewReader(input), stream.Config{})
	if err != nil {
		t.Fatalf("FindReaderCount error: %v", err)
	}

	if count != expected {
		t.Errorf("Count = %d, want %d", count, expected)
	}
}

// TestStreamingFirst verifies FindReaderFirst returns correct first match.
func TestStreamingFirst(t *testing.T) {
	input := "Some text before 2024-07-15 and more text 2024-08-20"
	expected := "2024-07-15"

	result, _, err := testdata.DatePattern{}.FindReaderFirst(strings.NewReader(input), stream.Config{})
	if err != nil {
		t.Fatalf("FindReaderFirst error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected to find a match")
	}

	if string(result.Match) != expected {
		t.Errorf("First match = %q, want %q", result.Match, expected)
	}
}

// TestStreamingEarlyTermination verifies callback can stop iteration.
func TestStreamingEarlyTermination(t *testing.T) {
	input := strings.Repeat("2024-01-01 ", 100)

	var count int
	err := testdata.DatePattern{}.FindReader(
		strings.NewReader(input),
		stream.Config{},
		func(m stream.Match[*testdata.DatePatternBytesResult]) bool {
			count++
			return count < 5
		},
	)
	if err != nil {
		t.Fatalf("FindReader error: %v", err)
	}

	if count != 5 {
		t.Errorf("Expected 5 matches before termination, got %d", count)
	}
}

// TestStreamingEmptyInput verifies handling of empty input.
func TestStreamingEmptyInput(t *testing.T) {
	count, err := testdata.DatePattern{}.FindReaderCount(strings.NewReader(""), stream.Config{})
	if err != nil {
		t.Fatalf("FindReaderCount error: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 matches on empty input, got %d", count)
	}
}

// TestStreamingNoMatches verifies handling of input with no matches.
func TestStreamingNoMatches(t *testing.T) {
	input := strings.Repeat("x", 10000)
	count, err := testdata.DatePattern{}.FindReaderCount(strings.NewReader(input), stream.Config{})
	if err != nil {
		t.Fatalf("FindReaderCount error: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 matches, got %d", count)
	}
}

// TestStreamingOffsets verifies that stream offsets are correct.
func TestStreamingOffsets(t *testing.T) {
	input := "prefix 2024-01-15 middle 2024-02-20 suffix"
	stdlibRe := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
	stdlibIndices := stdlibRe.FindAllStringIndex(input, -1)

	var streamOffsets [][2]int64
	err := testdata.DatePattern{}.FindReader(
		strings.NewReader(input),
		stream.Config{},
		func(m stream.Match[*testdata.DatePatternBytesResult]) bool {
			streamOffsets = append(streamOffsets, [2]int64{
				m.StreamOffset,
				m.StreamOffset + int64(len(m.Result.Match)),
			})
			return true
		},
	)
	if err != nil {
		t.Fatalf("FindReader error: %v", err)
	}

	if len(streamOffsets) != len(stdlibIndices) {
		t.Errorf("Count mismatch: stream=%d stdlib=%d", len(streamOffsets), len(stdlibIndices))
		return
	}

	for i := range streamOffsets {
		if streamOffsets[i][0] != int64(stdlibIndices[i][0]) ||
			streamOffsets[i][1] != int64(stdlibIndices[i][1]) {
			t.Errorf("Offset[%d] mismatch: stream=%v stdlib=%v",
				i, streamOffsets[i], stdlibIndices[i])
		}
	}
}
