package streaming

import (
	"bytes"
	"io"
	"testing"

	"github.com/KromDaniel/regengo/stream"
	"github.com/KromDaniel/regengo/tests/integration/streaming/testdata"
)

// FuzzReplaceReader tests ReplaceReader with arbitrary input
func FuzzReplaceReader(f *testing.F) {
	// Seed corpus with representative inputs
	f.Add([]byte("test@example.com"))
	f.Add([]byte("no matches here"))
	f.Add([]byte("a@b.co"))
	f.Add([]byte("user@domain.org more text another@test.com"))
	f.Add([]byte(""))
	f.Add([]byte("@"))
	f.Add([]byte("user@"))
	f.Add([]byte("@domain.com"))
	f.Add(make([]byte, 1000)) // zeros

	f.Fuzz(func(t *testing.T, data []byte) {
		r := testdata.CompiledEmailPattern.ReplaceReader(
			bytes.NewReader(data),
			"[REDACTED]",
		)
		result, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("ReplaceReader returned error: %v", err)
		}
		// Basic sanity check: result should not be absurdly large
		if len(result) > len(data)*10+100 {
			t.Errorf("output unexpectedly large: input %d bytes, output %d bytes", len(data), len(result))
		}
	})
}

// FuzzSelectReader tests SelectReader with arbitrary input
func FuzzSelectReader(f *testing.F) {
	f.Add([]byte("test@example.com"))
	f.Add([]byte("no matches here"))
	f.Add([]byte("a@b.co x@y.zz"))
	f.Add([]byte(""))

	f.Fuzz(func(t *testing.T, data []byte) {
		r := testdata.CompiledEmailPattern.SelectReader(
			bytes.NewReader(data),
			func(m *testdata.EmailPatternBytesResult) bool {
				return len(m.User) > 2
			},
		)
		result, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("SelectReader returned error: %v", err)
		}
		// Output should be <= input (we're only filtering, not expanding)
		if len(result) > len(data)+100 {
			t.Errorf("output larger than input: input %d, output %d", len(data), len(result))
		}
	})
}

// FuzzRejectReader tests RejectReader with arbitrary input
func FuzzRejectReader(f *testing.F) {
	f.Add([]byte("test@example.com"))
	f.Add([]byte("no matches here"))
	f.Add([]byte(""))

	f.Fuzz(func(t *testing.T, data []byte) {
		r := testdata.CompiledEmailPattern.RejectReader(
			bytes.NewReader(data),
			func(m *testdata.EmailPatternBytesResult) bool {
				return true // reject all
			},
		)
		result, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("RejectReader returned error: %v", err)
		}
		// Output should be <= input (we're removing matches)
		if len(result) > len(data) {
			t.Errorf("output larger than input: input %d, output %d", len(data), len(result))
		}
	})
}

// FuzzTransformReader tests NewTransformReader with arbitrary input
func FuzzTransformReader(f *testing.F) {
	f.Add([]byte("test@example.com"))
	f.Add([]byte(""))
	f.Add([]byte("x@y.zz a@b.cc"))

	cfg := stream.DefaultTransformConfig()

	f.Fuzz(func(t *testing.T, data []byte) {
		r := testdata.CompiledEmailPattern.NewTransformReader(
			bytes.NewReader(data),
			cfg,
			func(m *testdata.EmailPatternBytesResult, emit func([]byte)) {
				// Emit reversed match
				reversed := make([]byte, len(m.Match))
				for i, b := range m.Match {
					reversed[len(m.Match)-1-i] = b
				}
				emit(reversed)
			},
		)
		_, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("NewTransformReader returned error: %v", err)
		}
	})
}

// FuzzPipeline tests multi-stage pipeline with arbitrary input
func FuzzPipeline(f *testing.F) {
	f.Add([]byte("user@example.com 192.168.1.1"))
	f.Add([]byte("no matches"))
	f.Add([]byte(""))
	f.Add([]byte("10.0.0.1 admin@test.org 172.16.0.1"))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Stage 1: redact emails
		r1 := testdata.CompiledEmailPattern.ReplaceReader(
			bytes.NewReader(data),
			"[EMAIL]",
		)
		// Stage 2: redact IPs
		r2 := testdata.CompiledIPv4Pattern.ReplaceReader(r1, "[IP]")

		result, err := io.ReadAll(r2)
		if err != nil {
			t.Fatalf("Pipeline returned error: %v", err)
		}
		// Basic sanity
		if len(result) > len(data)*10+100 {
			t.Errorf("output unexpectedly large")
		}
	})
}

// FuzzSmallBuffer tests with small buffer sizes to stress boundary handling
func FuzzSmallBuffer(f *testing.F) {
	f.Add([]byte("test@example.com"))
	f.Add([]byte("a@b.co"))
	f.Add([]byte("xxxxxxxxxxxx@yyyyyyyy.zzz"))

	f.Fuzz(func(t *testing.T, data []byte) {
		cfg := stream.DefaultTransformConfig()
		cfg.BufferSize = 32 // Small buffer to stress boundary handling
		cfg.MaxLeftover = 16

		r := testdata.CompiledEmailPattern.NewTransformReader(
			bytes.NewReader(data),
			cfg,
			func(m *testdata.EmailPatternBytesResult, emit func([]byte)) {
				emit([]byte("[X]"))
			},
		)
		_, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("small buffer transform returned error: %v", err)
		}
	})
}
