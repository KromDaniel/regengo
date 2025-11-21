package generated

import (
	"regexp"
	"testing"
)

func TestEmailMatchString(t *testing.T) {
	pattern := "[\\w\\.+-]+@[\\w\\.-]+\\.[\\w\\.-]+"
	stdReg := regexp.MustCompile(pattern)

	t.Run("test input 0", func(t *testing.T) {
		input := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa| me@myself.com"
		isStdMatch := stdReg.MatchString(input)
		isRegengoMatch := Email{}.MatchString(input)
		if isStdMatch != isRegengoMatch {
			t.Fatalf("pattern %s stdMatch - %v, regengoMatch - %v", input, isStdMatch, isRegengoMatch)
		}
	})

}

func BenchmarkEmailMatchString(b *testing.B) {
	pattern := "[\\w\\.+-]+@[\\w\\.-]+\\.[\\w\\.-]+"
	stdReg := regexp.MustCompile(pattern)

	b.Run("golang std 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa| me@myself.com"
		for i := 0; i < b.N; i++ {
			stdReg.MatchString(input)
		}
	})

	b.Run("regengo 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa| me@myself.com"
		for i := 0; i < b.N; i++ {
			Email{}.MatchString(input)
		}
	})

}
