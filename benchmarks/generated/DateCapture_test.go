package generated

import (
	"regexp"
	"testing"
)

func TestDateCaptureFindString(t *testing.T) {
	pattern := "(?P<year>\\d{4})-(?P<month>\\d{2})-(?P<day>\\d{2})"
	stdReg := regexp.MustCompile(pattern)

	t.Run("test input 0", func(t *testing.T) {
		input := "2025-10-05"
		stdMatch := stdReg.FindStringSubmatch(input)
		regengoResult, found := DateCapture{}.FindString(input)

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
		input := "1999-12-31"
		stdMatch := stdReg.FindStringSubmatch(input)
		regengoResult, found := DateCapture{}.FindString(input)

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
		input := "2000-01-01"
		stdMatch := stdReg.FindStringSubmatch(input)
		regengoResult, found := DateCapture{}.FindString(input)

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

func BenchmarkDateCaptureFindString(b *testing.B) {
	pattern := "(?P<year>\\d{4})-(?P<month>\\d{2})-(?P<day>\\d{2})"
	stdReg := regexp.MustCompile(pattern)

	b.Run("golang std 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "2025-10-05"
		for i := 0; i < b.N; i++ {
			stdReg.FindStringSubmatch(input)
		}
	})

	b.Run("regengo 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "2025-10-05"
		for i := 0; i < b.N; i++ {
			DateCapture{}.FindString(input)
		}
	})

	b.Run("golang std 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "1999-12-31"
		for i := 0; i < b.N; i++ {
			stdReg.FindStringSubmatch(input)
		}
	})

	b.Run("regengo 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "1999-12-31"
		for i := 0; i < b.N; i++ {
			DateCapture{}.FindString(input)
		}
	})

	b.Run("golang std 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "2000-01-01"
		for i := 0; i < b.N; i++ {
			stdReg.FindStringSubmatch(input)
		}
	})

	b.Run("regengo 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "2000-01-01"
		for i := 0; i < b.N; i++ {
			DateCapture{}.FindString(input)
		}
	})

}
