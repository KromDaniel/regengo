package curated

import (
	"fmt"
	"regexp"
	"testing"
)

// DFASimple_test.go consolidates DFA-favorable pattern benchmarks.
// These patterns are specifically chosen to benefit from DFA:
// - ASCII-only characters
// - No dot (.) or word boundaries (\b)
// - Character classes, alternation, quantifiers

// Benchmark inputs of varying sizes
var dfaSimpleInputs = []string{
	"hello world", // short
	"the quick brown fox jumps over the lazy dog",              // medium
	"abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz0123", // long with match
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ",                               // no match (uppercase)
}

// BenchmarkDFASimpleLiteral benchmarks literal string matching with DFA
func BenchmarkDFASimpleLiteral(b *testing.B) {
	// Pattern: "hello" - simple literal, DFA-favorable
	stdRe := regexp.MustCompile("hello")

	for i, input := range dfaSimpleInputs {
		input := input
		b.Run(fmt.Sprintf("Input[%d]", i), func(b *testing.B) {
			b.Run("stdlib", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					_ = stdRe.MatchString(input)
				}
			})
			b.Run("DFA/DFACharClass", func(b *testing.B) {
				// DFACharClass uses [a-z]+ which matches "hello"
				b.ReportAllocs()
				for b.Loop() {
					_ = DFACharClass{}.MatchString(input)
				}
			})
		})
	}
}

// BenchmarkDFASimpleCharClass benchmarks character class matching with DFA
func BenchmarkDFASimpleCharClass(b *testing.B) {
	// Pattern: [a-z]+ - character class, DFA-favorable
	stdRe := regexp.MustCompile("[a-z]+")

	for i, input := range dfaSimpleInputs {
		input := input
		b.Run(fmt.Sprintf("Input[%d]", i), func(b *testing.B) {
			b.Run("stdlib", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					_ = stdRe.MatchString(input)
				}
			})
			b.Run("DFA", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					_ = DFACharClass{}.MatchString(input)
				}
			})
		})
	}
}

// BenchmarkDFASimpleAlternation benchmarks alternation matching with DFA
func BenchmarkDFASimpleAlternation(b *testing.B) {
	// Pattern: cat|dog|fish|bird|lion - alternation, DFA-favorable
	stdRe := regexp.MustCompile("cat|dog|fish|bird|lion")

	inputs := []string{
		"cat",
		"the dog ran",
		"no match here",
		"fish and bird and lion",
	}

	for i, input := range inputs {
		input := input
		b.Run(fmt.Sprintf("Input[%d]", i), func(b *testing.B) {
			b.Run("stdlib", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					_ = stdRe.MatchString(input)
				}
			})
			b.Run("DFA", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					_ = DFAAlternation{}.MatchString(input)
				}
			})
		})
	}
}

// BenchmarkDFASimpleCatastrophic benchmarks patterns that cause catastrophic backtracking
func BenchmarkDFASimpleCatastrophic(b *testing.B) {
	// Pattern: (a+)+b - catastrophic backtracking pattern, DFA handles safely
	stdRe := regexp.MustCompile("(a+)+b")

	// Inputs with varying lengths of 'a' followed by 'b'
	inputs := []string{
		"ab",
		"aaab",
		"aaaaaaab",
		"aaaaaaaaaaaaaaaaab",
	}

	for i, input := range inputs {
		input := input
		b.Run(fmt.Sprintf("Input[%d]", i), func(b *testing.B) {
			b.Run("stdlib", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					_ = stdRe.MatchString(input)
				}
			})
			b.Run("DFA", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					_ = DFACatastrophic{}.MatchString(input)
				}
			})
		})
	}
}
