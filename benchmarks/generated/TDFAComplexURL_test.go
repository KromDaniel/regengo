package generated

import (
	"regexp"
	"testing"
)

func TestTDFAComplexURLFindString(t *testing.T) {
	pattern := "(?P<scheme>https?)://(?P<auth>(?P<user>[\\w.-]+)(?::(?P<pass>[\\w.-]+))?@)?(?P<host>[\\w.-]+)(?::(?P<port>\\d+))?(?P<path>/[\\w./-]*)?(?:\\?(?P<query>[\\w=&.-]+))?"
	stdReg := regexp.MustCompile(pattern)

	t.Run("test input 0", func(t *testing.T) {
		input := "https://example.com"
		stdMatch := stdReg.FindStringSubmatch(input)
		regengoResult, found := TDFAComplexURL{}.FindString(input)

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
		input := "http://user:pass@example.com:8080/path/to/resource?key=value&foo=bar"
		stdMatch := stdReg.FindStringSubmatch(input)
		regengoResult, found := TDFAComplexURL{}.FindString(input)

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
		input := "https://api.github.com/repos/owner/repo"
		stdMatch := stdReg.FindStringSubmatch(input)
		regengoResult, found := TDFAComplexURL{}.FindString(input)

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

func BenchmarkTDFAComplexURLFindString(b *testing.B) {
	pattern := "(?P<scheme>https?)://(?P<auth>(?P<user>[\\w.-]+)(?::(?P<pass>[\\w.-]+))?@)?(?P<host>[\\w.-]+)(?::(?P<port>\\d+))?(?P<path>/[\\w./-]*)?(?:\\?(?P<query>[\\w=&.-]+))?"
	stdReg := regexp.MustCompile(pattern)

	b.Run("golang std 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "https://example.com"
		for b.Loop() {
			stdReg.FindStringSubmatch(input)
		}
	})

	b.Run("regengo 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "https://example.com"
		for b.Loop() {
			TDFAComplexURL{}.FindString(input)
		}
	})

	b.Run("regengo reuse 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "https://example.com"
		var result *TDFAComplexURLResult
		for b.Loop() {
			result, _ = TDFAComplexURL{}.FindStringReuse(input, result)
		}
	})

	b.Run("golang std 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "http://user:pass@example.com:8080/path/to/resource?key=value&foo=bar"
		for b.Loop() {
			stdReg.FindStringSubmatch(input)
		}
	})

	b.Run("regengo 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "http://user:pass@example.com:8080/path/to/resource?key=value&foo=bar"
		for b.Loop() {
			TDFAComplexURL{}.FindString(input)
		}
	})

	b.Run("regengo reuse 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "http://user:pass@example.com:8080/path/to/resource?key=value&foo=bar"
		var result *TDFAComplexURLResult
		for b.Loop() {
			result, _ = TDFAComplexURL{}.FindStringReuse(input, result)
		}
	})

	b.Run("golang std 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "https://api.github.com/repos/owner/repo"
		for b.Loop() {
			stdReg.FindStringSubmatch(input)
		}
	})

	b.Run("regengo 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "https://api.github.com/repos/owner/repo"
		for b.Loop() {
			TDFAComplexURL{}.FindString(input)
		}
	})

	b.Run("regengo reuse 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "https://api.github.com/repos/owner/repo"
		var result *TDFAComplexURLResult
		for b.Loop() {
			result, _ = TDFAComplexURL{}.FindStringReuse(input, result)
		}
	})

}
