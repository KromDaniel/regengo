package stream

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
)

// simpleProcessor creates a processor that replaces occurrences of 'find' with the onMatch result.
// This is a test helper that simulates what generated code would do.
func simpleProcessor(find []byte) Processor {
	return func(data []byte, isEOF bool, onMatch TransformFunc, emitNonMatch func([]byte)) int {
		processed := 0

		for {
			// Find next match
			idx := bytes.Index(data[processed:], find)
			if idx == -1 {
				// No more matches
				if isEOF {
					// Emit remaining data as non-match
					if processed < len(data) {
						emitNonMatch(data[processed:])
					}
					return len(data)
				}
				// Keep some data for potential boundary matches
				safePoint := len(data) - len(find) + 1
				if safePoint < processed {
					safePoint = processed
				}
				if safePoint > processed {
					emitNonMatch(data[processed:safePoint])
				}
				return safePoint
			}

			// Emit non-match before this match
			matchStart := processed + idx
			if matchStart > processed {
				emitNonMatch(data[processed:matchStart])
			}

			// Call onMatch for the match
			onMatch(find, emitNonMatch)

			processed = matchStart + len(find)
		}
	}
}

// TestTransformerImplementsReader verifies Transformer implements io.Reader (AC1.1)
func TestTransformerImplementsReader(t *testing.T) {
	var _ io.Reader = (*Transformer)(nil)
}

// TestTransformerBasicTransformation tests basic transformation (AC1.2)
func TestTransformerBasicTransformation(t *testing.T) {
	input := strings.NewReader("hello world")
	processor := simpleProcessor([]byte("world"))

	tr := NewTransformer(input, DefaultTransformConfig(), processor, func(match []byte, emit func([]byte)) {
		emit([]byte("WORLD"))
	})

	var buf bytes.Buffer
	_, err := io.Copy(&buf, tr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "hello WORLD"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

// TestTransformerEmptyInput tests empty input returns EOF correctly (AC1.3)
func TestTransformerEmptyInput(t *testing.T) {
	input := strings.NewReader("")
	processor := simpleProcessor([]byte("anything"))

	tr := NewTransformer(input, DefaultTransformConfig(), processor, func(match []byte, emit func([]byte)) {
		emit([]byte("REPLACED"))
	})

	buf := make([]byte, 10)
	n, err := tr.Read(buf)
	if n != 0 {
		t.Errorf("expected 0 bytes, got %d", n)
	}
	if err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}
}

// TestTransformerLargeInput tests streaming without memory blowup (AC1.4)
func TestTransformerLargeInput(t *testing.T) {
	// Create 10MB of input with periodic markers
	const inputSize = 10 * 1024 * 1024
	const marker = "MARKER"
	const markerInterval = 1024 // Place marker every 1KB

	// Build input
	var inputBuilder strings.Builder
	for i := 0; i < inputSize; {
		if i > 0 && i%markerInterval == 0 {
			inputBuilder.WriteString(marker)
			i += len(marker)
		} else {
			inputBuilder.WriteByte('x')
			i++
		}
	}
	input := strings.NewReader(inputBuilder.String())

	processor := simpleProcessor([]byte(marker))
	replacementCount := 0

	tr := NewTransformer(input, DefaultTransformConfig(), processor, func(match []byte, emit func([]byte)) {
		emit([]byte("REPLACED"))
		replacementCount++
	})

	// Read through with small buffer to simulate streaming
	buf := make([]byte, 4096)
	totalRead := 0
	for {
		n, err := tr.Read(buf)
		totalRead += n
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if totalRead == 0 {
		t.Error("expected some output")
	}

	// Should have replaced many markers
	if replacementCount == 0 {
		t.Error("expected some replacements")
	}
}

// TestTransformerPartialReads tests reading with small buffer (AC1.5)
func TestTransformerPartialReads(t *testing.T) {
	input := strings.NewReader("hello world, hello universe")
	processor := simpleProcessor([]byte("hello"))

	tr := NewTransformer(input, DefaultTransformConfig(), processor, func(match []byte, emit func([]byte)) {
		emit([]byte("HI"))
	})

	// Read one byte at a time
	var result bytes.Buffer
	buf := make([]byte, 1)
	for {
		n, err := tr.Read(buf)
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

	expected := "HI world, HI universe"
	if result.String() != expected {
		t.Errorf("expected %q, got %q", expected, result.String())
	}
}

// TestTransformerChunkBoundary tests match spanning chunk boundaries (AC1.6)
func TestTransformerChunkBoundary(t *testing.T) {
	// Use a small buffer to force chunk boundaries
	cfg := DefaultTransformConfig()
	cfg.BufferSize = 16
	cfg.MaxLeftover = 8

	// Input where "MARKER" might span chunks
	input := strings.NewReader("xxxxMARKERxxxx")
	processor := simpleProcessor([]byte("MARKER"))

	tr := NewTransformer(input, cfg, processor, func(match []byte, emit func([]byte)) {
		emit([]byte("FOUND"))
	})

	var buf bytes.Buffer
	_, err := io.Copy(&buf, tr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "xxxxFOUNDxxxx"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

// TestTransformerContextCancellation tests context cancellation (AC1.7)
func TestTransformerContextCancellation(t *testing.T) {
	t.Run("pre-canceled context", func(t *testing.T) {
		// Test with already-canceled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		cfg := DefaultTransformConfig()
		cfg.Context = ctx

		input := strings.NewReader("hello world")
		processor := simpleProcessor([]byte("hello"))
		tr := NewTransformer(input, cfg, processor, func(match []byte, emit func([]byte)) {
			emit([]byte("HI"))
		})

		buf := make([]byte, 1024)
		_, err := tr.Read(buf)
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})

	t.Run("cancellation during processing", func(t *testing.T) {
		// Test cancellation during multi-chunk processing
		ctx, cancel := context.WithCancel(context.Background())
		cfg := DefaultTransformConfig()
		cfg.BufferSize = 64 // Small buffer to force multiple chunks
		cfg.Context = ctx

		// Large input to ensure multiple reads
		input := strings.NewReader(strings.Repeat("hello world ", 10000))
		processor := simpleProcessor([]byte("hello"))

		tr := NewTransformer(input, cfg, processor, func(match []byte, emit func([]byte)) {
			emit([]byte("HI"))
		})

		// Read some data then cancel
		buf := make([]byte, 100)
		_, err := tr.Read(buf)
		if err != nil {
			t.Fatalf("first read failed: %v", err)
		}

		cancel()

		// Next read should return context.Canceled
		_, err = tr.Read(buf)
		if err != context.Canceled {
			t.Errorf("expected context.Canceled after cancel, got %v", err)
		}
	})
}

// TestTransformerDropMatch tests that not calling emit drops the match (AC1.8)
func TestTransformerDropMatch(t *testing.T) {
	input := strings.NewReader("keep DELETE keep DELETE keep")
	processor := simpleProcessor([]byte("DELETE"))

	tr := NewTransformer(input, DefaultTransformConfig(), processor, func(match []byte, emit func([]byte)) {
		// Don't call emit - drop the match
	})

	var buf bytes.Buffer
	_, err := io.Copy(&buf, tr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "keep  keep  keep"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

// TestTransformerPassthrough tests emitting original match (AC1.9)
func TestTransformerPassthrough(t *testing.T) {
	input := strings.NewReader("hello world hello")
	processor := simpleProcessor([]byte("hello"))

	tr := NewTransformer(input, DefaultTransformConfig(), processor, func(match []byte, emit func([]byte)) {
		emit(match) // Passthrough original
	})

	var buf bytes.Buffer
	_, err := io.Copy(&buf, tr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "hello world hello"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

// TestTransformerMultipleEmit tests 1-to-N expansion (AC1.10)
func TestTransformerMultipleEmit(t *testing.T) {
	input := strings.NewReader("a X b")
	processor := simpleProcessor([]byte("X"))

	tr := NewTransformer(input, DefaultTransformConfig(), processor, func(match []byte, emit func([]byte)) {
		emit([]byte("1"))
		emit([]byte("2"))
		emit([]byte("3"))
	})

	var buf bytes.Buffer
	_, err := io.Copy(&buf, tr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "a 123 b"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

// TestTransformerReset tests reusing transformer (buffer reuse)
func TestTransformerReset(t *testing.T) {
	processor := simpleProcessor([]byte("old"))

	tr := NewTransformer(strings.NewReader("old data"), DefaultTransformConfig(), processor, func(match []byte, emit func([]byte)) {
		emit([]byte("new"))
	})

	var buf bytes.Buffer
	io.Copy(&buf, tr)

	if buf.String() != "new data" {
		t.Errorf("first read: expected %q, got %q", "new data", buf.String())
	}

	// Reset and use again
	tr.Reset(strings.NewReader("more old stuff"))
	buf.Reset()
	io.Copy(&buf, tr)

	if buf.String() != "more new stuff" {
		t.Errorf("after reset: expected %q, got %q", "more new stuff", buf.String())
	}
}

// TestTransformerMultipleMatches tests multiple matches in input
func TestTransformerMultipleMatches(t *testing.T) {
	input := strings.NewReader("aaa bbb aaa ccc aaa")
	processor := simpleProcessor([]byte("aaa"))

	tr := NewTransformer(input, DefaultTransformConfig(), processor, func(match []byte, emit func([]byte)) {
		emit([]byte("XXX"))
	})

	var buf bytes.Buffer
	_, err := io.Copy(&buf, tr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "XXX bbb XXX ccc XXX"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

// TestTransformerNoMatches tests input with no matches
func TestTransformerNoMatches(t *testing.T) {
	input := strings.NewReader("hello world")
	processor := simpleProcessor([]byte("xyz"))

	tr := NewTransformer(input, DefaultTransformConfig(), processor, func(match []byte, emit func([]byte)) {
		emit([]byte("REPLACED"))
	})

	var buf bytes.Buffer
	_, err := io.Copy(&buf, tr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "hello world"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

// TestTransformerMatchAtStart tests match at beginning of input
func TestTransformerMatchAtStart(t *testing.T) {
	input := strings.NewReader("hello world")
	processor := simpleProcessor([]byte("hello"))

	tr := NewTransformer(input, DefaultTransformConfig(), processor, func(match []byte, emit func([]byte)) {
		emit([]byte("HI"))
	})

	var buf bytes.Buffer
	_, err := io.Copy(&buf, tr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "HI world"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

// TestTransformerMatchAtEnd tests match at end of input
func TestTransformerMatchAtEnd(t *testing.T) {
	input := strings.NewReader("hello world")
	processor := simpleProcessor([]byte("world"))

	tr := NewTransformer(input, DefaultTransformConfig(), processor, func(match []byte, emit func([]byte)) {
		emit([]byte("EARTH"))
	})

	var buf bytes.Buffer
	_, err := io.Copy(&buf, tr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "hello EARTH"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

// TestTransformerOnlyMatches tests input that is entirely matches
func TestTransformerOnlyMatches(t *testing.T) {
	input := strings.NewReader("aaa")
	processor := simpleProcessor([]byte("aaa"))

	tr := NewTransformer(input, DefaultTransformConfig(), processor, func(match []byte, emit func([]byte)) {
		emit([]byte("BBB"))
	})

	var buf bytes.Buffer
	_, err := io.Copy(&buf, tr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "BBB"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

// TestTransformerExpansion tests replacement larger than match
func TestTransformerExpansion(t *testing.T) {
	input := strings.NewReader("X")
	processor := simpleProcessor([]byte("X"))

	tr := NewTransformer(input, DefaultTransformConfig(), processor, func(match []byte, emit func([]byte)) {
		emit([]byte("EXPANSION_IS_MUCH_LONGER"))
	})

	var buf bytes.Buffer
	_, err := io.Copy(&buf, tr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "EXPANSION_IS_MUCH_LONGER"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

// TestTransformerContraction tests replacement smaller than match
func TestTransformerContraction(t *testing.T) {
	input := strings.NewReader("VERYLONGMATCH")
	processor := simpleProcessor([]byte("VERYLONGMATCH"))

	tr := NewTransformer(input, DefaultTransformConfig(), processor, func(match []byte, emit func([]byte)) {
		emit([]byte("X"))
	})

	var buf bytes.Buffer
	_, err := io.Copy(&buf, tr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "X"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

// BenchmarkTransformer benchmarks the transformer
func BenchmarkTransformer(b *testing.B) {
	input := strings.Repeat("hello world ", 1000)
	processor := simpleProcessor([]byte("hello"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr := NewTransformer(strings.NewReader(input), DefaultTransformConfig(), processor, func(match []byte, emit func([]byte)) {
			emit([]byte("HI"))
		})
		io.Copy(io.Discard, tr)
	}
}

// BenchmarkTransformerSmallBuffer benchmarks with small buffer (more chunks)
func BenchmarkTransformerSmallBuffer(b *testing.B) {
	input := strings.Repeat("hello world ", 1000)
	cfg := DefaultTransformConfig()
	cfg.BufferSize = 256
	processor := simpleProcessor([]byte("hello"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr := NewTransformer(strings.NewReader(input), cfg, processor, func(match []byte, emit func([]byte)) {
			emit([]byte("HI"))
		})
		io.Copy(io.Discard, tr)
	}
}

// TestTransformerPooled tests pooled transformer functionality
func TestTransformerPooled(t *testing.T) {
	input := strings.NewReader("hello world")
	processor := simpleProcessor([]byte("world"))

	tr := NewTransformerPooled(input, DefaultTransformConfig(), processor, func(match []byte, emit func([]byte)) {
		emit([]byte("WORLD"))
	})
	defer tr.Close()

	var buf bytes.Buffer
	_, err := io.Copy(&buf, tr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "hello WORLD"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

// TestTransformerPooledMultipleClose tests Close() can be called multiple times safely
func TestTransformerPooledMultipleClose(t *testing.T) {
	input := strings.NewReader("hello world")
	processor := simpleProcessor([]byte("world"))

	tr := NewTransformerPooled(input, DefaultTransformConfig(), processor, func(match []byte, emit func([]byte)) {
		emit([]byte("WORLD"))
	})

	// Read all data
	io.Copy(io.Discard, tr)

	// Close multiple times - should not panic
	err1 := tr.Close()
	err2 := tr.Close()
	err3 := tr.Close()

	if err1 != nil || err2 != nil || err3 != nil {
		t.Errorf("Close() should return nil on multiple calls")
	}
}

// TestTransformerPooledReuse tests that pool can be used across multiple transformers
func TestTransformerPooledReuse(t *testing.T) {
	processor := simpleProcessor([]byte("hello"))

	// Create and use many transformers to exercise pooling
	for i := 0; i < 100; i++ {
		input := strings.NewReader("hello world hello")
		tr := NewTransformerPooled(input, DefaultTransformConfig(), processor, func(match []byte, emit func([]byte)) {
			emit([]byte("HI"))
		})

		var buf bytes.Buffer
		io.Copy(&buf, tr)
		tr.Close()

		expected := "HI world HI"
		if buf.String() != expected {
			t.Errorf("iteration %d: expected %q, got %q", i, expected, buf.String())
		}
	}
}

// TestTransformerPooledNonDefaultBufferSize tests that non-default buffer sizes allocate new buffers
func TestTransformerPooledNonDefaultBufferSize(t *testing.T) {
	cfg := DefaultTransformConfig()
	cfg.BufferSize = 128 // Non-default size

	input := strings.NewReader("hello world")
	processor := simpleProcessor([]byte("world"))

	tr := NewTransformerPooled(input, cfg, processor, func(match []byte, emit func([]byte)) {
		emit([]byte("WORLD"))
	})
	defer tr.Close()

	var buf bytes.Buffer
	_, err := io.Copy(&buf, tr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "hello WORLD"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

// TestTransformerNonPooledClose tests that Close() on non-pooled transformer is safe
func TestTransformerNonPooledClose(t *testing.T) {
	input := strings.NewReader("hello world")
	processor := simpleProcessor([]byte("world"))

	tr := NewTransformer(input, DefaultTransformConfig(), processor, func(match []byte, emit func([]byte)) {
		emit([]byte("WORLD"))
	})

	io.Copy(io.Discard, tr)

	// Close on non-pooled should be safe no-op
	err := tr.Close()
	if err != nil {
		t.Errorf("Close() on non-pooled should return nil, got %v", err)
	}
}

// BenchmarkTransformerPooled benchmarks pooled transformer
func BenchmarkTransformerPooled(b *testing.B) {
	input := strings.Repeat("hello world ", 1000)
	processor := simpleProcessor([]byte("hello"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr := NewTransformerPooled(strings.NewReader(input), DefaultTransformConfig(), processor, func(match []byte, emit func([]byte)) {
			emit([]byte("HI"))
		})
		io.Copy(io.Discard, tr)
		tr.Close()
	}
}

// BenchmarkTransformerVsPooled compares allocation overhead
func BenchmarkTransformerVsPooled(b *testing.B) {
	input := strings.Repeat("hello world ", 1000)
	processor := simpleProcessor([]byte("hello"))

	b.Run("regular", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tr := NewTransformer(strings.NewReader(input), DefaultTransformConfig(), processor, func(match []byte, emit func([]byte)) {
				emit([]byte("HI"))
			})
			io.Copy(io.Discard, tr)
		}
	})

	b.Run("pooled", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tr := NewTransformerPooled(strings.NewReader(input), DefaultTransformConfig(), processor, func(match []byte, emit func([]byte)) {
				emit([]byte("HI"))
			})
			io.Copy(io.Discard, tr)
			tr.Close()
		}
	})
}
