package generated

import (
	"regexp"
	"testing"
)

func TestTDFALogParserFindString(t *testing.T) {
	pattern := "(?P<timestamp>\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})(?:\\.(?P<ms>\\d{3}))?(?P<tz>Z|[+-]\\d{2}:\\d{2})?\\s+\\[(?P<level>\\w+)\\]\\s+(?P<message>.+)"
	stdReg := regexp.MustCompile(pattern)

	t.Run("test input 0", func(t *testing.T) {
		input := "2024-01-15T10:30:45.123Z [INFO] Server started successfully"
		stdMatch := stdReg.FindStringSubmatch(input)
		regengoResult, found := TDFALogParser{}.FindString(input)

		if (len(stdMatch) > 0) != found {
			t.Fatalf("pattern %s stdMatch found=%v, regengo found=%v", input, len(stdMatch) > 0, found)
		}

		if found {
			// Verify the full match
			if stdMatch[0] != regengoResult.Match {
				t.Errorf("Full match mismatch: std=%q regengo=%q", stdMatch[0], regengoResult.Match)
			}
		}
	})

	t.Run("test input 1", func(t *testing.T) {
		input := "2024-01-15T10:30:45+00:00 [ERROR] Connection failed"
		stdMatch := stdReg.FindStringSubmatch(input)
		regengoResult, found := TDFALogParser{}.FindString(input)

		if (len(stdMatch) > 0) != found {
			t.Fatalf("pattern %s stdMatch found=%v, regengo found=%v", input, len(stdMatch) > 0, found)
		}

		if found {
			// Verify the full match
			if stdMatch[0] != regengoResult.Match {
				t.Errorf("Full match mismatch: std=%q regengo=%q", stdMatch[0], regengoResult.Match)
			}
		}
	})

	t.Run("test input 2", func(t *testing.T) {
		input := "2024-01-15T10:30:45 [DEBUG] Processing request id=12345"
		stdMatch := stdReg.FindStringSubmatch(input)
		regengoResult, found := TDFALogParser{}.FindString(input)

		if (len(stdMatch) > 0) != found {
			t.Fatalf("pattern %s stdMatch found=%v, regengo found=%v", input, len(stdMatch) > 0, found)
		}

		if found {
			// Verify the full match
			if stdMatch[0] != regengoResult.Match {
				t.Errorf("Full match mismatch: std=%q regengo=%q", stdMatch[0], regengoResult.Match)
			}
		}
	})

}

func BenchmarkTDFALogParserFindString(b *testing.B) {
	pattern := "(?P<timestamp>\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})(?:\\.(?P<ms>\\d{3}))?(?P<tz>Z|[+-]\\d{2}:\\d{2})?\\s+\\[(?P<level>\\w+)\\]\\s+(?P<message>.+)"
	stdReg := regexp.MustCompile(pattern)

	b.Run("golang std 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "2024-01-15T10:30:45.123Z [INFO] Server started successfully"
		for b.Loop() {
			stdReg.FindStringSubmatch(input)
		}
	})

	b.Run("regengo 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "2024-01-15T10:30:45.123Z [INFO] Server started successfully"
		for b.Loop() {
			TDFALogParser{}.FindString(input)
		}
	})

	b.Run("regengo reuse 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "2024-01-15T10:30:45.123Z [INFO] Server started successfully"
		var result *TDFALogParserResult
		for b.Loop() {
			result, _ = TDFALogParser{}.FindStringReuse(input, result)
		}
	})

	b.Run("golang std 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "2024-01-15T10:30:45+00:00 [ERROR] Connection failed"
		for b.Loop() {
			stdReg.FindStringSubmatch(input)
		}
	})

	b.Run("regengo 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "2024-01-15T10:30:45+00:00 [ERROR] Connection failed"
		for b.Loop() {
			TDFALogParser{}.FindString(input)
		}
	})

	b.Run("regengo reuse 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "2024-01-15T10:30:45+00:00 [ERROR] Connection failed"
		var result *TDFALogParserResult
		for b.Loop() {
			result, _ = TDFALogParser{}.FindStringReuse(input, result)
		}
	})

	b.Run("golang std 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "2024-01-15T10:30:45 [DEBUG] Processing request id=12345"
		for b.Loop() {
			stdReg.FindStringSubmatch(input)
		}
	})

	b.Run("regengo 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "2024-01-15T10:30:45 [DEBUG] Processing request id=12345"
		for b.Loop() {
			TDFALogParser{}.FindString(input)
		}
	})

	b.Run("regengo reuse 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "2024-01-15T10:30:45 [DEBUG] Processing request id=12345"
		var result *TDFALogParserResult
		for b.Loop() {
			result, _ = TDFALogParser{}.FindStringReuse(input, result)
		}
	})

}
