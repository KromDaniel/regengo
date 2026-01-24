package curated

import (
	"fmt"
	"regexp"
	"testing"
)

// DFAvsThompson_test.go provides direct comparison benchmarks between
// DFA and Thompson NFA engines for the same patterns.
//
// This helps verify the performance benefits of DFA over Thompson for
// patterns that are DFA-favorable.

// Test inputs for DFA vs Thompson comparison
var dfaVsThompsonInputs = []string{
	"abc",
	"abcdefghijklmnopqrstuvwxyz",
	"the quick brown fox",
	"ABCDEF", // no lowercase match
}

// BenchmarkDFAvsThompsonCharClass compares [a-z]+ between DFA and Thompson
func BenchmarkDFAvsThompsonCharClass(b *testing.B) {
	stdRe := regexp.MustCompile("[a-z]+")

	for i, input := range dfaVsThompsonInputs {
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
			b.Run("Thompson", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					_ = ThompsonCharClass{}.MatchString(input)
				}
			})
		})
	}
}

// BenchmarkDFAvsThompsonCatastrophic compares (a+)+b between DFA and Thompson
// This is a pattern that can cause catastrophic backtracking in naive implementations
func BenchmarkDFAvsThompsonCatastrophic(b *testing.B) {
	stdRe := regexp.MustCompile("(a+)+b")

	// Inputs with varying lengths - catastrophic patterns scale poorly
	inputs := []string{
		"ab",
		"aaab",
		"aaaaaaab",
		"aaaaaaaaaaaaaaaaab",
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaab",
	}

	for i, input := range inputs {
		input := input
		b.Run(fmt.Sprintf("Input[%d]/len=%d", i, len(input)), func(b *testing.B) {
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
			b.Run("Thompson", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					_ = ThompsonCatastrophic{}.MatchString(input)
				}
			})
		})
	}
}

// BenchmarkDFAvsThompsonNoMatch compares performance when no match is found
// This tests the worst-case scan performance
func BenchmarkDFAvsThompsonNoMatch(b *testing.B) {
	// Test with inputs that don't match
	inputs := []string{
		"ABCDEF",
		"123456789",
		"!@#$%^&*()",
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
	}

	for i, input := range inputs {
		input := input
		b.Run(fmt.Sprintf("CharClass/Input[%d]", i), func(b *testing.B) {
			b.Run("DFA", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					_ = DFACharClass{}.MatchString(input)
				}
			})
			b.Run("Thompson", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					_ = ThompsonCharClass{}.MatchString(input)
				}
			})
		})
	}
}
