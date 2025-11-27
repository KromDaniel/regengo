package generated

import (
	"regexp"
	"testing"
)

func TestTNFAPathologicalFindString(t *testing.T) {
	pattern := "(?P<outer>(?P<inner>a+)+)b"
	stdReg := regexp.MustCompile(pattern)

	t.Run("test input 0", func(t *testing.T) {
		input := "aaaaaaaaaaaaaaaaaaaab"
		stdMatch := stdReg.FindStringSubmatch(input)
		regengoResult, found := TNFAPathological{}.FindString(input)

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
		regengoResult, found := TNFAPathological{}.FindString(input)

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

func BenchmarkTNFAPathologicalFindString(b *testing.B) {
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
			TNFAPathological{}.FindString(input)
		}
	})

	b.Run("regengo reuse 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "aaaaaaaaaaaaaaaaaaaab"
		var result *TNFAPathologicalResult
		for b.Loop() {
			result, _ = TNFAPathological{}.FindStringReuse(input, result)
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
			TNFAPathological{}.FindString(input)
		}
	})

	b.Run("regengo reuse 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaab"
		var result *TNFAPathologicalResult
		for b.Loop() {
			result, _ = TNFAPathological{}.FindStringReuse(input, result)
		}
	})

}
