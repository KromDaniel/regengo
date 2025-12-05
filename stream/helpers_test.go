package stream

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestLineFilter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		pred     func([]byte) bool
		expected string
	}{
		{
			name:  "keep lines containing ERROR",
			input: "INFO: starting\nERROR: failed\nINFO: done\nERROR: another\n",
			pred: func(line []byte) bool {
				return bytes.Contains(line, []byte("ERROR"))
			},
			expected: "ERROR: failed\nERROR: another\n",
		},
		{
			name:  "keep all lines",
			input: "line1\nline2\nline3\n",
			pred: func(line []byte) bool {
				return true
			},
			expected: "line1\nline2\nline3\n",
		},
		{
			name:  "keep no lines",
			input: "line1\nline2\nline3\n",
			pred: func(line []byte) bool {
				return false
			},
			expected: "",
		},
		{
			name:  "empty input",
			input: "",
			pred: func(line []byte) bool {
				return true
			},
			expected: "",
		},
		{
			name:  "single line without newline",
			input: "single line",
			pred: func(line []byte) bool {
				return true
			},
			expected: "single line",
		},
		{
			name:  "filter by line length",
			input: "short\nthis is a longer line\nmed\n",
			pred: func(line []byte) bool {
				return len(line) > 10
			},
			expected: "this is a longer line\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := LineFilter(strings.NewReader(tt.input), tt.pred)
			var buf bytes.Buffer
			_, err := io.Copy(&buf, r)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if buf.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, buf.String())
			}
		})
	}
}

func TestLineFilterPartialReads(t *testing.T) {
	input := "line1\nline2\nline3\n"
	r := LineFilter(strings.NewReader(input), func(line []byte) bool {
		return bytes.Contains(line, []byte("2"))
	})

	// Read one byte at a time
	var result bytes.Buffer
	buf := make([]byte, 1)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			result.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	expected := "line2\n"
	if result.String() != expected {
		t.Errorf("expected %q, got %q", expected, result.String())
	}
}

func TestLineFilterLargeInput(t *testing.T) {
	// Create input with many lines
	var inputBuilder strings.Builder
	for i := 0; i < 10000; i++ {
		if i%100 == 0 {
			inputBuilder.WriteString("KEEP: important line\n")
		} else {
			inputBuilder.WriteString("skip: boring line\n")
		}
	}

	r := LineFilter(strings.NewReader(inputBuilder.String()), func(line []byte) bool {
		return bytes.HasPrefix(line, []byte("KEEP"))
	})

	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 100 lines (every 100th line from 0 to 9999)
	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	if len(lines) != 100 {
		t.Errorf("expected 100 lines, got %d", len(lines))
	}
}

func TestLineTransform(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		fn       func([]byte) []byte
		expected string
	}{
		{
			name:  "prefix each line",
			input: "line1\nline2\nline3\n",
			fn: func(line []byte) []byte {
				return append([]byte(">> "), line...)
			},
			expected: ">> line1\n>> line2\n>> line3\n",
		},
		{
			name:  "uppercase",
			input: "hello\nworld\n",
			fn: func(line []byte) []byte {
				return bytes.ToUpper(line)
			},
			expected: "HELLO\nWORLD\n",
		},
		{
			name:  "remove newlines",
			input: "a\nb\nc\n",
			fn: func(line []byte) []byte {
				return bytes.TrimSuffix(line, []byte("\n"))
			},
			expected: "abc",
		},
		{
			name:  "empty input",
			input: "",
			fn: func(line []byte) []byte {
				return append([]byte("prefix: "), line...)
			},
			expected: "",
		},
		{
			name:  "single line without newline",
			input: "single",
			fn: func(line []byte) []byte {
				return append(line, []byte(" transformed")...)
			},
			expected: "single transformed",
		},
		{
			name:  "filter via transform (return empty)",
			input: "keep\nskip\nkeep\n",
			fn: func(line []byte) []byte {
				if bytes.HasPrefix(line, []byte("skip")) {
					return nil
				}
				return line
			},
			expected: "keep\nkeep\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := LineTransform(strings.NewReader(tt.input), tt.fn)
			var buf bytes.Buffer
			_, err := io.Copy(&buf, r)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if buf.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, buf.String())
			}
		})
	}
}

func TestLineTransformPartialReads(t *testing.T) {
	input := "line1\nline2\n"
	r := LineTransform(strings.NewReader(input), func(line []byte) []byte {
		return bytes.ToUpper(line)
	})

	// Read one byte at a time
	var result bytes.Buffer
	buf := make([]byte, 1)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			result.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	expected := "LINE1\nLINE2\n"
	if result.String() != expected {
		t.Errorf("expected %q, got %q", expected, result.String())
	}
}

func TestLineTransformLargeInput(t *testing.T) {
	// Create input with many lines
	var inputBuilder strings.Builder
	for i := 0; i < 10000; i++ {
		inputBuilder.WriteString("test line content\n")
	}

	r := LineTransform(strings.NewReader(inputBuilder.String()), func(line []byte) []byte {
		return append([]byte("TRANSFORMED: "), line...)
	})

	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 10000 transformed lines
	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	if len(lines) != 10000 {
		t.Errorf("expected 10000 lines, got %d", len(lines))
	}

	// Check first line
	if !strings.HasPrefix(lines[0], "TRANSFORMED: ") {
		t.Errorf("first line not transformed: %q", lines[0])
	}
}

func TestLineFilterAndTransformChain(t *testing.T) {
	input := "INFO: starting\nERROR: failed\nINFO: done\nERROR: another\n"

	// Chain: filter ERRORs then uppercase
	var r io.Reader = strings.NewReader(input)
	r = LineFilter(r, func(line []byte) bool {
		return bytes.Contains(line, []byte("ERROR"))
	})
	r = LineTransform(r, func(line []byte) []byte {
		return bytes.ToUpper(line)
	})

	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "ERROR: FAILED\nERROR: ANOTHER\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

// Benchmarks

func BenchmarkLineFilter(b *testing.B) {
	input := strings.Repeat("INFO: some log line here\nERROR: an error occurred\n", 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := LineFilter(strings.NewReader(input), func(line []byte) bool {
			return bytes.Contains(line, []byte("ERROR"))
		})
		io.Copy(io.Discard, r)
	}
}

func BenchmarkLineTransform(b *testing.B) {
	input := strings.Repeat("some log line here\n", 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := LineTransform(strings.NewReader(input), func(line []byte) []byte {
			return append([]byte("PREFIX: "), line...)
		})
		io.Copy(io.Discard, r)
	}
}
