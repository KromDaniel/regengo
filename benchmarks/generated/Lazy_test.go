
package generated

import (
	"regexp"
	"testing"
)


func TestLazyMatchString(t *testing.T) {
	pattern := "(?:(?:a|b)|(?:k)+)+?abcd"
	stdReg := regexp.MustCompile(pattern)
	
	
	t.Run("test input 0", func(t *testing.T) {
		input := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabcd"
        isStdMatch := stdReg.MatchString(input)
        isRegengoMatch := LazyMatchString(input)
        if isStdMatch != isRegengoMatch {
			t.Fatalf("pattern %s stdMatch - %v, regengoMatch - %v", input, isStdMatch, isRegengoMatch)
        }
	})

	
}

func BenchmarkLazyMatchString(b *testing.B) {
	pattern := "(?:(?:a|b)|(?:k)+)+?abcd"
	stdReg := regexp.MustCompile(pattern)
	
	

	b.Run("golang std 0", func(b *testing.B) {
        b.ReportAllocs()
		input := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabcd"
        for i:=0 ; i < b.N; i++ {
          stdReg.MatchString(input)
        }
	})
	
	b.Run("regengo 0", func(b *testing.B) {
        b.ReportAllocs()
		input := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabcd"
        for i:=0 ; i < b.N; i++ {
          LazyMatchString(input)
        }
	})

	
}
