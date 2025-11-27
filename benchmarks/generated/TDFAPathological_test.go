package generated

import (
	"regexp"
	"testing"
)

func TestTDFAPathologicalFindString(t *testing.T) {
	pattern := "(?P<outer>(?P<inner>a+)+)b"
	stdReg := regexp.MustCompile(pattern)

	t.Run("test input 0", func(t *testing.T) {
		input := "aaaaaaaaaaaaaaaaaaaab"
		stdMatch := stdReg.FindStringSubmatch(input)
		regengoResult, found := TDFAPathological{}.FindString(input)

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
		input := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaab"
		stdMatch := stdReg.FindStringSubmatch(input)
		regengoResult, found := TDFAPathological{}.FindString(input)

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
		input := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaab"
		stdMatch := stdReg.FindStringSubmatch(input)
		regengoResult, found := TDFAPathological{}.FindString(input)

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

	t.Run("test input 3", func(t *testing.T) {
		input := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		stdMatch := stdReg.FindStringSubmatch(input)
		regengoResult, found := TDFAPathological{}.FindString(input)

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

func BenchmarkTDFAPathologicalFindString(b *testing.B) {
	pattern := "(?P<outer>(?P<inner>a+)+)b"
	stdReg := regexp.MustCompile(pattern)

	b.Run("golang std 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "aaaaaaaaaaaaaaaaaaaab"
		for b.Loop() {
			stdReg.FindStringSubmatch(input)
		}
	})

	b.Run("regengo 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "aaaaaaaaaaaaaaaaaaaab"
		for b.Loop() {
			TDFAPathological{}.FindString(input)
		}
	})

	b.Run("regengo reuse 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "aaaaaaaaaaaaaaaaaaaab"
		var result *TDFAPathologicalResult
		for b.Loop() {
			result, _ = TDFAPathological{}.FindStringReuse(input, result)
		}
	})

	b.Run("golang std 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaab"
		for b.Loop() {
			stdReg.FindStringSubmatch(input)
		}
	})

	b.Run("regengo 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaab"
		for b.Loop() {
			TDFAPathological{}.FindString(input)
		}
	})

	b.Run("regengo reuse 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaab"
		var result *TDFAPathologicalResult
		for b.Loop() {
			result, _ = TDFAPathological{}.FindStringReuse(input, result)
		}
	})

	b.Run("golang std 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaab"
		for b.Loop() {
			stdReg.FindStringSubmatch(input)
		}
	})

	b.Run("regengo 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaab"
		for b.Loop() {
			TDFAPathological{}.FindString(input)
		}
	})

	b.Run("regengo reuse 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaab"
		var result *TDFAPathologicalResult
		for b.Loop() {
			result, _ = TDFAPathological{}.FindStringReuse(input, result)
		}
	})

	b.Run("golang std 3", func(b *testing.B) {
		b.ReportAllocs()
		input := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		for b.Loop() {
			stdReg.FindStringSubmatch(input)
		}
	})

	b.Run("regengo 3", func(b *testing.B) {
		b.ReportAllocs()
		input := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		for b.Loop() {
			TDFAPathological{}.FindString(input)
		}
	})

	b.Run("regengo reuse 3", func(b *testing.B) {
		b.ReportAllocs()
		input := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		var result *TDFAPathologicalResult
		for b.Loop() {
			result, _ = TDFAPathological{}.FindStringReuse(input, result)
		}
	})

}
