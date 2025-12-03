package streaming

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/KromDaniel/regengo/stream"
	"github.com/KromDaniel/regengo/tests/integration/streaming/testdata"
)

// ============================================================================
// Basic Transform Operations
// ============================================================================

func TestReplaceReader_Basic(t *testing.T) {
	input := "Contact user@example.com for help"
	r := testdata.CompiledEmailPattern.ReplaceReader(strings.NewReader(input), "[REDACTED]")

	result, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Contact [REDACTED] for help"
	if string(result) != expected {
		t.Errorf("expected %q, got %q", expected, string(result))
	}
}

func TestReplaceReader_Template(t *testing.T) {
	input := "Contact john@example.com for help"
	r := testdata.CompiledEmailPattern.ReplaceReader(strings.NewReader(input), "[$user AT $domain]")

	result, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Contact [john AT example.com] for help"
	if string(result) != expected {
		t.Errorf("expected %q, got %q", expected, string(result))
	}
}

func TestReplaceReader_NoMatches(t *testing.T) {
	input := "No emails here, just text"
	r := testdata.CompiledEmailPattern.ReplaceReader(strings.NewReader(input), "[EMAIL]")

	result, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != input {
		t.Errorf("expected %q, got %q", input, string(result))
	}
}

func TestReplaceReader_AllMatches(t *testing.T) {
	input := "a@b.com"
	r := testdata.CompiledEmailPattern.ReplaceReader(strings.NewReader(input), "[X]")

	result, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "[X]"
	if string(result) != expected {
		t.Errorf("expected %q, got %q", expected, string(result))
	}
}

func TestSelectReader_KeepAll(t *testing.T) {
	input := "Contact john@a.com and jane@b.org"
	r := testdata.CompiledEmailPattern.SelectReader(strings.NewReader(input), func(m *testdata.EmailPatternBytesResult) bool {
		return true
	})

	result, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain both emails
	if !bytes.Contains(result, []byte("john@a.com")) || !bytes.Contains(result, []byte("jane@b.org")) {
		t.Errorf("expected both emails, got %q", string(result))
	}
	// Should not contain non-match text
	if bytes.Contains(result, []byte("Contact")) {
		t.Errorf("should not contain non-match text, got %q", string(result))
	}
}

func TestSelectReader_FilterByCapture(t *testing.T) {
	input := "john@company.com and jane@external.org and bob@company.com"
	r := testdata.CompiledEmailPattern.SelectReader(strings.NewReader(input), func(m *testdata.EmailPatternBytesResult) bool {
		return bytes.Equal(m.Domain, []byte("company.com"))
	})

	result, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Contains(result, []byte("john@company.com")) || !bytes.Contains(result, []byte("bob@company.com")) {
		t.Errorf("missing company emails: %q", string(result))
	}
	if bytes.Contains(result, []byte("jane@external.org")) {
		t.Errorf("should not contain external email: %q", string(result))
	}
}

func TestRejectReader_RemoveAll(t *testing.T) {
	input := "Contact user@example.com for help"
	r := testdata.CompiledEmailPattern.RejectReader(strings.NewReader(input), func(m *testdata.EmailPatternBytesResult) bool {
		return true
	})

	result, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Contact  for help"
	if string(result) != expected {
		t.Errorf("expected %q, got %q", expected, string(result))
	}
}

func TestRejectReader_FilterByCapture(t *testing.T) {
	input := "good@keep.com and bad@spam.com and ok@keep.com"
	r := testdata.CompiledEmailPattern.RejectReader(strings.NewReader(input), func(m *testdata.EmailPatternBytesResult) bool {
		return bytes.Equal(m.Domain, []byte("spam.com"))
	})

	result, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Contains(result, []byte("good@keep.com")) || !bytes.Contains(result, []byte("ok@keep.com")) {
		t.Errorf("missing kept emails: %q", string(result))
	}
	if bytes.Contains(result, []byte("bad@spam.com")) {
		t.Errorf("should have removed spam email: %q", string(result))
	}
}

// ============================================================================
// Edge Cases
// ============================================================================

func TestTransform_EmptyInput(t *testing.T) {
	r := testdata.CompiledEmailPattern.ReplaceReader(strings.NewReader(""), "[EMAIL]")

	result, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty output, got %q", string(result))
	}
}

func TestTransform_SingleByte(t *testing.T) {
	r := testdata.CompiledEmailPattern.ReplaceReader(strings.NewReader("x"), "[EMAIL]")

	result, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != "x" {
		t.Errorf("expected %q, got %q", "x", string(result))
	}
}

func TestTransform_MatchAtBoundary(t *testing.T) {
	// CRITICAL: Use small buffer to force chunk boundary
	// Buffer size 16, match "a@b.co" (6 chars) positioned to straddle boundary
	cfg := stream.DefaultTransformConfig()
	cfg.BufferSize = 16
	cfg.MaxLeftover = 8

	// Use spaces (not matched by email pattern) to avoid greedy consumption
	// Position email to straddle indices 14-19 (crosses 16-byte boundary)
	input := "              " + "a@b.co" + "      " // 14 spaces + email + 6 spaces
	r := testdata.CompiledEmailPattern.NewTransformReader(
		strings.NewReader(input),
		cfg,
		func(m *testdata.EmailPatternBytesResult, emit func([]byte)) {
			emit([]byte("[FOUND]"))
		})

	result, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "              [FOUND]      "
	if string(result) != expected {
		t.Errorf("expected %q, got %q", expected, string(result))
	}
}

func TestTransform_PartialReads(t *testing.T) {
	input := "test@example.com"
	r := testdata.CompiledEmailPattern.ReplaceReader(strings.NewReader(input), "[X]")

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

	expected := "[X]"
	if result.String() != expected {
		t.Errorf("expected %q, got %q", expected, result.String())
	}
}

func TestTransform_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	cfg := stream.DefaultTransformConfig()
	cfg.Context = ctx

	input := strings.NewReader("test@example.com")
	r := testdata.CompiledEmailPattern.NewTransformReader(input, cfg,
		func(m *testdata.EmailPatternBytesResult, emit func([]byte)) {
			emit([]byte("[X]"))
		})

	buf := make([]byte, 1024)
	_, err := r.Read(buf)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// ============================================================================
// Pipeline Tests
// ============================================================================

func TestPipeline_TwoTransforms(t *testing.T) {
	input := "Email: user@test.com, Date: 2024-01-15"

	var r io.Reader = strings.NewReader(input)
	r = testdata.CompiledEmailPattern.ReplaceReader(r, "[EMAIL]")
	r = testdata.CompiledDatePattern.ReplaceReader(r, "[DATE]")

	result, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Email: [EMAIL], Date: [DATE]"
	if string(result) != expected {
		t.Errorf("expected %q, got %q", expected, string(result))
	}
}

func TestPipeline_ThreeTransforms(t *testing.T) {
	input := "user@test.com visited 192.168.1.1 on 2024-01-15"

	var r io.Reader = strings.NewReader(input)
	r = testdata.CompiledEmailPattern.ReplaceReader(r, "[EMAIL]")
	r = testdata.CompiledIPv4Pattern.ReplaceReader(r, "[IP]")
	r = testdata.CompiledDatePattern.ReplaceReader(r, "[DATE]")

	result, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "[EMAIL] visited [IP] on [DATE]"
	if string(result) != expected {
		t.Errorf("expected %q, got %q", expected, string(result))
	}
}

func TestPipeline_TransformThenLineFilter(t *testing.T) {
	input := "INFO: user@test.com logged in\nDEBUG: checking\nERROR: admin@test.com failed\n"

	var r io.Reader = strings.NewReader(input)
	r = testdata.CompiledEmailPattern.ReplaceReader(r, "[EMAIL]")
	r = stream.LineFilter(r, func(line []byte) bool {
		return bytes.Contains(line, []byte("ERROR"))
	})

	result, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "ERROR: [EMAIL] failed\n"
	if string(result) != expected {
		t.Errorf("expected %q, got %q", expected, string(result))
	}
}

func TestPipeline_LineFilterThenTransform(t *testing.T) {
	input := "INFO: user@test.com logged in\nDEBUG: checking\nERROR: admin@test.com failed\n"

	var r io.Reader = strings.NewReader(input)
	r = stream.LineFilter(r, func(line []byte) bool {
		return !bytes.HasPrefix(line, []byte("DEBUG"))
	})
	r = testdata.CompiledEmailPattern.ReplaceReader(r, "[EMAIL]")

	result, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "INFO: [EMAIL] logged in\nERROR: [EMAIL] failed\n"
	if string(result) != expected {
		t.Errorf("expected %q, got %q", expected, string(result))
	}
}

func TestPipeline_ComplexChain(t *testing.T) {
	input := "DEBUG: user@test.com\nINFO: 192.168.1.1 on 2024-01-15\nERROR: admin@x.com at 10.0.0.1\n"

	var r io.Reader = strings.NewReader(input)
	// 1. Filter out DEBUG
	r = stream.LineFilter(r, func(line []byte) bool {
		return !bytes.HasPrefix(line, []byte("DEBUG"))
	})
	// 2. Mask emails
	r = testdata.CompiledEmailPattern.ReplaceReader(r, "[EMAIL]")
	// 3. Mask IPs
	r = testdata.CompiledIPv4Pattern.ReplaceReader(r, "[IP]")
	// 4. Mask dates
	r = testdata.CompiledDatePattern.ReplaceReader(r, "[DATE]")
	// 5. Add prefix
	r = stream.LineTransform(r, func(line []byte) []byte {
		return append([]byte(">>> "), line...)
	})

	result, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := ">>> INFO: [IP] on [DATE]\n>>> ERROR: [EMAIL] at [IP]\n"
	if string(result) != expected {
		t.Errorf("expected %q, got %q", expected, string(result))
	}
}

// errorReader returns an error after reading some data
type errorReader struct {
	data    string
	pos     int
	errAt   int
	testErr error
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	if r.pos >= r.errAt {
		return 0, r.testErr
	}
	remaining := r.data[r.pos:]
	if len(remaining) > len(p) {
		remaining = remaining[:len(p)]
	}
	if r.pos+len(remaining) > r.errAt {
		remaining = remaining[:r.errAt-r.pos]
	}
	n = copy(p, remaining)
	r.pos += n
	if r.pos >= r.errAt {
		return n, r.testErr
	}
	return n, nil
}

func TestPipeline_ErrorPropagation(t *testing.T) {
	testErr := errors.New("source reader error")
	source := &errorReader{
		data:    "user@test.com some more data here",
		errAt:   20,
		testErr: testErr,
	}

	var r io.Reader = source
	r = testdata.CompiledEmailPattern.ReplaceReader(r, "[EMAIL]")
	r = stream.LineTransform(r, func(line []byte) []byte {
		return append([]byte("PREFIX: "), line...)
	})

	_, err := io.ReadAll(r)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !errors.Is(err, testErr) {
		t.Errorf("expected %v, got %v", testErr, err)
	}
}

// ============================================================================
// Correctness Validation
// ============================================================================

func TestTransform_MatchesInMemoryReplace(t *testing.T) {
	inputs := []string{
		"Contact user@example.com for help",
		"user@a.com and user@b.com",
		"No matches here",
		"single@match.com",
		"a@b.c d@e.f g@h.i",
	}

	for _, input := range inputs {
		// In-memory replacement
		expected := testdata.CompiledEmailPattern.ReplaceAllString(input, "[X]")

		// Streaming replacement
		r := testdata.CompiledEmailPattern.ReplaceReader(strings.NewReader(input), "[X]")
		result, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("input %q: unexpected error: %v", input, err)
		}

		if string(result) != expected {
			t.Errorf("input %q: streaming %q != in-memory %q", input, string(result), expected)
		}
	}
}

func TestTransform_PreservesNonMatches(t *testing.T) {
	input := "abc user@test.com xyz 123 another@email.org end"
	r := testdata.CompiledEmailPattern.ReplaceReader(strings.NewReader(input), "[]")

	result, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify non-match segments preserved
	resultStr := string(result)
	if !strings.Contains(resultStr, "abc ") {
		t.Error("missing 'abc '")
	}
	if !strings.Contains(resultStr, " xyz 123 ") {
		t.Error("missing ' xyz 123 '")
	}
	if !strings.Contains(resultStr, " end") {
		t.Error("missing ' end'")
	}
}

func TestTransform_OrderPreserved(t *testing.T) {
	input := "a@1.com b@2.com c@3.com"
	r := testdata.CompiledEmailPattern.SelectReader(strings.NewReader(input), func(m *testdata.EmailPatternBytesResult) bool {
		return true
	})

	result, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify order
	resultStr := string(result)
	idx1 := strings.Index(resultStr, "a@1.com")
	idx2 := strings.Index(resultStr, "b@2.com")
	idx3 := strings.Index(resultStr, "c@3.com")

	if idx1 == -1 || idx2 == -1 || idx3 == -1 {
		t.Fatalf("missing emails in output: %q", resultStr)
	}
	if !(idx1 < idx2 && idx2 < idx3) {
		t.Errorf("order not preserved: a@%d, b@%d, c@%d in %q", idx1, idx2, idx3, resultStr)
	}
}
