package generated

import (
	"regexp"
	"testing"
)

func TestTDFANestedWordFindString(t *testing.T) {
	pattern := "(?P<words>(?P<word>\\w+\\s*)+)end"
	stdReg := regexp.MustCompile(pattern)

	t.Run("test input 0", func(t *testing.T) {
		input := "hello world end"
		stdMatch := stdReg.FindStringSubmatch(input)
		regengoResult, found := TDFANestedWord{}.FindString(input)

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
		input := "a b c d e f g h i j end"
		stdMatch := stdReg.FindStringSubmatch(input)
		regengoResult, found := TDFANestedWord{}.FindString(input)

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
		input := "word word word word word word word word word word word word word word word word word word word word end"
		stdMatch := stdReg.FindStringSubmatch(input)
		regengoResult, found := TDFANestedWord{}.FindString(input)

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

func BenchmarkTDFANestedWordFindString(b *testing.B) {
	pattern := "(?P<words>(?P<word>\\w+\\s*)+)end"
	stdReg := regexp.MustCompile(pattern)

	b.Run("golang std 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "hello world end"
		for b.Loop() {
			stdReg.FindStringSubmatch(input)
		}
	})

	b.Run("regengo 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "hello world end"
		for b.Loop() {
			TDFANestedWord{}.FindString(input)
		}
	})

	b.Run("regengo reuse 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "hello world end"
		var result *TDFANestedWordResult
		for b.Loop() {
			result, _ = TDFANestedWord{}.FindStringReuse(input, result)
		}
	})

	b.Run("golang std 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "a b c d e f g h i j end"
		for b.Loop() {
			stdReg.FindStringSubmatch(input)
		}
	})

	b.Run("regengo 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "a b c d e f g h i j end"
		for b.Loop() {
			TDFANestedWord{}.FindString(input)
		}
	})

	b.Run("regengo reuse 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "a b c d e f g h i j end"
		var result *TDFANestedWordResult
		for b.Loop() {
			result, _ = TDFANestedWord{}.FindStringReuse(input, result)
		}
	})

	b.Run("golang std 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "word word word word word word word word word word word word word word word word word word word word end"
		for b.Loop() {
			stdReg.FindStringSubmatch(input)
		}
	})

	b.Run("regengo 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "word word word word word word word word word word word word word word word word word word word word end"
		for b.Loop() {
			TDFANestedWord{}.FindString(input)
		}
	})

	b.Run("regengo reuse 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "word word word word word word word word word word word word word word word word word word word word end"
		var result *TDFANestedWordResult
		for b.Loop() {
			result, _ = TDFANestedWord{}.FindStringReuse(input, result)
		}
	})

}
