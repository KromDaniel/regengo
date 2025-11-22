
package generated

import (
	"regexp"
	"testing"
)


func TestURLCaptureFindString(t *testing.T) {
	pattern := "(?P<protocol>https?)://(?P<host>[\\w\\.-]+)(?::(?P<port>\\d+))?(?P<path>/[\\w\\./]*)?"
	stdReg := regexp.MustCompile(pattern)
	
	
	t.Run("test input 0", func(t *testing.T) {
		input := "http://example.com"
		stdMatch := stdReg.FindStringSubmatch(input)
		regengoResult, found := URLCapture{}.FindString(input)
		
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
		input := "https://api.github.com:443/repos/owner/repo"
		stdMatch := stdReg.FindStringSubmatch(input)
		regengoResult, found := URLCapture{}.FindString(input)
		
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
		input := "http://localhost:8080/api/v1/users"
		stdMatch := stdReg.FindStringSubmatch(input)
		regengoResult, found := URLCapture{}.FindString(input)
		
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

func BenchmarkURLCaptureFindString(b *testing.B) {
	pattern := "(?P<protocol>https?)://(?P<host>[\\w\\.-]+)(?::(?P<port>\\d+))?(?P<path>/[\\w\\./]*)?"
	stdReg := regexp.MustCompile(pattern)
	
	

	b.Run("golang std 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "http://example.com"
		for b.Loop() {
			stdReg.FindStringSubmatch(input)
		}
	})

	b.Run("regengo 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "http://example.com"
		for b.Loop() {
			URLCapture{}.FindString(input)
		}
	})

	b.Run("regengo reuse 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "http://example.com"
		var result *URLCaptureResult
		for b.Loop() {
			result, _ = URLCapture{}.FindStringReuse(input, result)
		}
	})

	

	b.Run("golang std 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "https://api.github.com:443/repos/owner/repo"
		for b.Loop() {
			stdReg.FindStringSubmatch(input)
		}
	})

	b.Run("regengo 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "https://api.github.com:443/repos/owner/repo"
		for b.Loop() {
			URLCapture{}.FindString(input)
		}
	})

	b.Run("regengo reuse 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "https://api.github.com:443/repos/owner/repo"
		var result *URLCaptureResult
		for b.Loop() {
			result, _ = URLCapture{}.FindStringReuse(input, result)
		}
	})

	

	b.Run("golang std 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "http://localhost:8080/api/v1/users"
		for b.Loop() {
			stdReg.FindStringSubmatch(input)
		}
	})

	b.Run("regengo 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "http://localhost:8080/api/v1/users"
		for b.Loop() {
			URLCapture{}.FindString(input)
		}
	})

	b.Run("regengo reuse 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "http://localhost:8080/api/v1/users"
		var result *URLCaptureResult
		for b.Loop() {
			result, _ = URLCapture{}.FindStringReuse(input, result)
		}
	})

	
}
