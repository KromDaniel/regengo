package generated

import (
	"regexp"
	"testing"
)

func TestTDFASemVerFindString(t *testing.T) {
	pattern := "(?P<major>\\d+)\\.(?P<minor>\\d+)\\.(?P<patch>\\d+)(?:-(?P<prerelease>[\\w.-]+))?(?:\\+(?P<build>[\\w.-]+))?"
	stdReg := regexp.MustCompile(pattern)

	t.Run("test input 0", func(t *testing.T) {
		input := "1.0.0"
		stdMatch := stdReg.FindStringSubmatch(input)
		regengoResult, found := TDFASemVer{}.FindString(input)

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
		input := "2.1.3-alpha.1"
		stdMatch := stdReg.FindStringSubmatch(input)
		regengoResult, found := TDFASemVer{}.FindString(input)

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
		input := "3.0.0-beta.2+build.123"
		stdMatch := stdReg.FindStringSubmatch(input)
		regengoResult, found := TDFASemVer{}.FindString(input)

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
		input := "10.20.30-rc.1+20240115"
		stdMatch := stdReg.FindStringSubmatch(input)
		regengoResult, found := TDFASemVer{}.FindString(input)

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

func BenchmarkTDFASemVerFindString(b *testing.B) {
	pattern := "(?P<major>\\d+)\\.(?P<minor>\\d+)\\.(?P<patch>\\d+)(?:-(?P<prerelease>[\\w.-]+))?(?:\\+(?P<build>[\\w.-]+))?"
	stdReg := regexp.MustCompile(pattern)

	b.Run("golang std 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "1.0.0"
		for b.Loop() {
			stdReg.FindStringSubmatch(input)
		}
	})

	b.Run("regengo 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "1.0.0"
		for b.Loop() {
			TDFASemVer{}.FindString(input)
		}
	})

	b.Run("regengo reuse 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "1.0.0"
		var result *TDFASemVerResult
		for b.Loop() {
			result, _ = TDFASemVer{}.FindStringReuse(input, result)
		}
	})

	b.Run("golang std 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "2.1.3-alpha.1"
		for b.Loop() {
			stdReg.FindStringSubmatch(input)
		}
	})

	b.Run("regengo 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "2.1.3-alpha.1"
		for b.Loop() {
			TDFASemVer{}.FindString(input)
		}
	})

	b.Run("regengo reuse 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "2.1.3-alpha.1"
		var result *TDFASemVerResult
		for b.Loop() {
			result, _ = TDFASemVer{}.FindStringReuse(input, result)
		}
	})

	b.Run("golang std 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "3.0.0-beta.2+build.123"
		for b.Loop() {
			stdReg.FindStringSubmatch(input)
		}
	})

	b.Run("regengo 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "3.0.0-beta.2+build.123"
		for b.Loop() {
			TDFASemVer{}.FindString(input)
		}
	})

	b.Run("regengo reuse 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "3.0.0-beta.2+build.123"
		var result *TDFASemVerResult
		for b.Loop() {
			result, _ = TDFASemVer{}.FindStringReuse(input, result)
		}
	})

	b.Run("golang std 3", func(b *testing.B) {
		b.ReportAllocs()
		input := "10.20.30-rc.1+20240115"
		for b.Loop() {
			stdReg.FindStringSubmatch(input)
		}
	})

	b.Run("regengo 3", func(b *testing.B) {
		b.ReportAllocs()
		input := "10.20.30-rc.1+20240115"
		for b.Loop() {
			TDFASemVer{}.FindString(input)
		}
	})

	b.Run("regengo reuse 3", func(b *testing.B) {
		b.ReportAllocs()
		input := "10.20.30-rc.1+20240115"
		var result *TDFASemVerResult
		for b.Loop() {
			result, _ = TDFASemVer{}.FindStringReuse(input, result)
		}
	})

}
