package curated

import (
	"testing"
)

// BenchmarkCompileTemplateComparison compares the performance of:
// 1. Runtime replace (template parsed every call)
// 2. Compiled template (parsed once, reused)
// 3. Pre-compiled (no parsing, inlined)
func BenchmarkCompileTemplateComparison(b *testing.B) {
	input := "Contact alice@example.com and bob@test.org"
	template := "$user@REDACTED.$tld"

	// Compile template once for the compiled benchmark
	compiledTmpl, err := ReplaceEmail{}.CompileReplaceTemplate(template)
	if err != nil {
		b.Fatalf("failed to compile template: %v", err)
	}

	b.Run("runtime", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = ReplaceEmail{}.ReplaceAllString(input, template)
		}
	})

	b.Run("compiled", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = compiledTmpl.ReplaceAllString(input)
		}
	})

	b.Run("precompiled", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = ReplaceEmail{}.ReplaceAllString0(input)
		}
	})
}

// BenchmarkCompileTemplateComparisonBytes compares bytes variants
func BenchmarkCompileTemplateComparisonBytes(b *testing.B) {
	input := []byte("Contact alice@example.com and bob@test.org")
	template := "$user@REDACTED.$tld"

	compiledTmpl, _ := ReplaceEmail{}.CompileReplaceTemplate(template)
	buf := make([]byte, 0, 4096)

	b.Run("runtime", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf = ReplaceEmail{}.ReplaceAllBytesAppend(input, template, buf[:0])
		}
	})

	b.Run("compiled", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf = compiledTmpl.ReplaceAllBytesAppend(input, buf[:0])
		}
	})

	b.Run("precompiled", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf = ReplaceEmail{}.ReplaceAllBytesAppend0(input, buf[:0])
		}
	})
}

// BenchmarkCompileTemplateManyInputs simulates high-throughput scenario
func BenchmarkCompileTemplateManyInputs(b *testing.B) {
	inputs := []string{
		"Contact alice@example.com for help",
		"Email bob@test.org or charlie@domain.net",
		"No emails here",
		"user@domain.tld",
		"Multiple: a@b.com, c@d.org, e@f.net",
	}
	template := "$user@REDACTED.$tld"

	compiledTmpl, _ := ReplaceEmail{}.CompileReplaceTemplate(template)

	b.Run("runtime", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			for _, input := range inputs {
				_ = ReplaceEmail{}.ReplaceAllString(input, template)
			}
		}
	})

	b.Run("compiled", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			for _, input := range inputs {
				_ = compiledTmpl.ReplaceAllString(input)
			}
		}
	})

	b.Run("precompiled", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			for _, input := range inputs {
				_ = ReplaceEmail{}.ReplaceAllString0(input)
			}
		}
	})
}
