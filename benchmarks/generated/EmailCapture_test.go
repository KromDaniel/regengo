package generated

import (
	"regexp"
	"testing"
)

func TestEmailCaptureFindString(t *testing.T) {
	pattern := "(?P<user>[\\w\\.+-]+)@(?P<domain>[\\w\\.-]+)\\.(?P<tld>[\\w\\.-]+)"
	stdReg := regexp.MustCompile(pattern)

	t.Run("test input 0", func(t *testing.T) {
		input := "user@example.com"
		stdMatch := stdReg.FindStringSubmatch(input)
		regengoResult, found := EmailCapture{}.FindString(input)

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
		input := "john.doe+tag@subdomain.example.co.uk"
		stdMatch := stdReg.FindStringSubmatch(input)
		regengoResult, found := EmailCapture{}.FindString(input)

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
		input := "test@test.org"
		stdMatch := stdReg.FindStringSubmatch(input)
		regengoResult, found := EmailCapture{}.FindString(input)

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

func BenchmarkEmailCaptureFindString(b *testing.B) {
	pattern := "(?P<user>[\\w\\.+-]+)@(?P<domain>[\\w\\.-]+)\\.(?P<tld>[\\w\\.-]+)"
	stdReg := regexp.MustCompile(pattern)

	b.Run("golang std 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "user@example.com"
		for b.Loop() {
			stdReg.FindStringSubmatch(input)
		}
	})

	b.Run("regengo 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "user@example.com"
		for b.Loop() {
			EmailCapture{}.FindString(input)
		}
	})

	b.Run("regengo reuse 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "user@example.com"
		var result *EmailCaptureResult
		for b.Loop() {
			result, _ = EmailCapture{}.FindStringReuse(input, result)
		}
	})

	b.Run("golang std 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "john.doe+tag@subdomain.example.co.uk"
		for b.Loop() {
			stdReg.FindStringSubmatch(input)
		}
	})

	b.Run("regengo 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "john.doe+tag@subdomain.example.co.uk"
		for b.Loop() {
			EmailCapture{}.FindString(input)
		}
	})

	b.Run("regengo reuse 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "john.doe+tag@subdomain.example.co.uk"
		var result *EmailCaptureResult
		for b.Loop() {
			result, _ = EmailCapture{}.FindStringReuse(input, result)
		}
	})

	b.Run("golang std 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "test@test.org"
		for b.Loop() {
			stdReg.FindStringSubmatch(input)
		}
	})

	b.Run("regengo 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "test@test.org"
		for b.Loop() {
			EmailCapture{}.FindString(input)
		}
	})

	b.Run("regengo reuse 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "test@test.org"
		var result *EmailCaptureResult
		for b.Loop() {
			result, _ = EmailCapture{}.FindStringReuse(input, result)
		}
	})

}
