package generated

import (
	"reflect"
	"regexp"
	"testing"
)

func TestMultiDateFindAllString(t *testing.T) {
	pattern := "(?P<year>\\d{4})-(?P<month>\\d{2})-(?P<day>\\d{2})"
	stdReg := regexp.MustCompile(pattern)

	t.Run("test input 0", func(t *testing.T) {
		input := "Events: 2024-01-15, 2024-06-20, and 2024-12-25 are holidays"
		stdMatches := stdReg.FindAllStringSubmatch(input, -1)
		regengoResults := MultiDate{}.FindAllString(input, -1)

		if len(stdMatches) != len(regengoResults) {
			t.Fatalf("match count mismatch: std=%d, regengo=%d", len(stdMatches), len(regengoResults))
		}

		for i, stdMatch := range stdMatches {
			result := regengoResults[i]

			// Compare full match
			if stdMatch[0] != result.Match {
				t.Errorf("match %d: full match mismatch: std=%q regengo=%q", i, stdMatch[0], result.Match)
			}

			// Compare each capture group using reflection
			v := reflect.ValueOf(result).Elem()
			typ := v.Type()

			// Field 0 is Match, so capture groups start at field 1
			for j := 1; j < v.NumField(); j++ {
				fieldName := typ.Field(j).Name
				fieldValue := v.Field(j).String()

				// stdMatch index: 0=full match, 1=first group, etc.
				if j < len(stdMatch) {
					if stdMatch[j] != fieldValue {
						t.Errorf("match %d, field %s: std=%q regengo=%q", i, fieldName, stdMatch[j], fieldValue)
					}
				}
			}
		}
	})

	t.Run("test input 1", func(t *testing.T) {
		input := "No dates here"
		stdMatches := stdReg.FindAllStringSubmatch(input, -1)
		regengoResults := MultiDate{}.FindAllString(input, -1)

		if len(stdMatches) != len(regengoResults) {
			t.Fatalf("match count mismatch: std=%d, regengo=%d", len(stdMatches), len(regengoResults))
		}

		for i, stdMatch := range stdMatches {
			result := regengoResults[i]

			// Compare full match
			if stdMatch[0] != result.Match {
				t.Errorf("match %d: full match mismatch: std=%q regengo=%q", i, stdMatch[0], result.Match)
			}

			// Compare each capture group using reflection
			v := reflect.ValueOf(result).Elem()
			typ := v.Type()

			// Field 0 is Match, so capture groups start at field 1
			for j := 1; j < v.NumField(); j++ {
				fieldName := typ.Field(j).Name
				fieldValue := v.Field(j).String()

				// stdMatch index: 0=full match, 1=first group, etc.
				if j < len(stdMatch) {
					if stdMatch[j] != fieldValue {
						t.Errorf("match %d, field %s: std=%q regengo=%q", i, fieldName, stdMatch[j], fieldValue)
					}
				}
			}
		}
	})

	t.Run("test input 2", func(t *testing.T) {
		input := "Single date 2025-10-05 in text"
		stdMatches := stdReg.FindAllStringSubmatch(input, -1)
		regengoResults := MultiDate{}.FindAllString(input, -1)

		if len(stdMatches) != len(regengoResults) {
			t.Fatalf("match count mismatch: std=%d, regengo=%d", len(stdMatches), len(regengoResults))
		}

		for i, stdMatch := range stdMatches {
			result := regengoResults[i]

			// Compare full match
			if stdMatch[0] != result.Match {
				t.Errorf("match %d: full match mismatch: std=%q regengo=%q", i, stdMatch[0], result.Match)
			}

			// Compare each capture group using reflection
			v := reflect.ValueOf(result).Elem()
			typ := v.Type()

			// Field 0 is Match, so capture groups start at field 1
			for j := 1; j < v.NumField(); j++ {
				fieldName := typ.Field(j).Name
				fieldValue := v.Field(j).String()

				// stdMatch index: 0=full match, 1=first group, etc.
				if j < len(stdMatch) {
					if stdMatch[j] != fieldValue {
						t.Errorf("match %d, field %s: std=%q regengo=%q", i, fieldName, stdMatch[j], fieldValue)
					}
				}
			}
		}
	})

}

func BenchmarkMultiDateFindAllString(b *testing.B) {
	pattern := "(?P<year>\\d{4})-(?P<month>\\d{2})-(?P<day>\\d{2})"
	stdReg := regexp.MustCompile(pattern)

	b.Run("golang std 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Events: 2024-01-15, 2024-06-20, and 2024-12-25 are holidays"
		for b.Loop() {
			stdReg.FindAllStringSubmatch(input, -1)
		}
	})

	b.Run("regengo 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Events: 2024-01-15, 2024-06-20, and 2024-12-25 are holidays"
		for b.Loop() {
			MultiDate{}.FindAllString(input, -1)
		}
	})

	b.Run("regengo reuse 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Events: 2024-01-15, 2024-06-20, and 2024-12-25 are holidays"
		results := make([]*MultiDateResult, 0, 10)
		for b.Loop() {
			results = MultiDate{}.FindAllStringAppend(input, -1, results[:0])
		}
	})

	b.Run("golang std 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "No dates here"
		for b.Loop() {
			stdReg.FindAllStringSubmatch(input, -1)
		}
	})

	b.Run("regengo 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "No dates here"
		for b.Loop() {
			MultiDate{}.FindAllString(input, -1)
		}
	})

	b.Run("regengo reuse 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "No dates here"
		results := make([]*MultiDateResult, 0, 10)
		for b.Loop() {
			results = MultiDate{}.FindAllStringAppend(input, -1, results[:0])
		}
	})

	b.Run("golang std 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "Single date 2025-10-05 in text"
		for b.Loop() {
			stdReg.FindAllStringSubmatch(input, -1)
		}
	})

	b.Run("regengo 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "Single date 2025-10-05 in text"
		for b.Loop() {
			MultiDate{}.FindAllString(input, -1)
		}
	})

	b.Run("regengo reuse 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "Single date 2025-10-05 in text"
		results := make([]*MultiDateResult, 0, 10)
		for b.Loop() {
			results = MultiDate{}.FindAllStringAppend(input, -1, results[:0])
		}
	})

}
