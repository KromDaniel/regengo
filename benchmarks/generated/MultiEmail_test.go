
package generated

import (
	"reflect"
	"regexp"
	"testing"
)


func TestMultiEmailFindAllString(t *testing.T) {
	pattern := "(?P<user>[\\w\\.+-]+)@(?P<domain>[\\w\\.-]+)\\.(?P<tld>[\\w\\.-]+)"
	stdReg := regexp.MustCompile(pattern)
	
	
	t.Run("test input 0", func(t *testing.T) {
		input := "Contact us at support@example.com or sales@company.org for help"
		stdMatches := stdReg.FindAllStringSubmatch(input, -1)
		regengoResults := MultiEmail{}.FindAllString(input, -1)

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
		input := "Multiple: a@b.com, c@d.org, e@f.net"
		stdMatches := stdReg.FindAllStringSubmatch(input, -1)
		regengoResults := MultiEmail{}.FindAllString(input, -1)

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
		input := "No emails in this text"
		stdMatches := stdReg.FindAllStringSubmatch(input, -1)
		regengoResults := MultiEmail{}.FindAllString(input, -1)

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

func BenchmarkMultiEmailFindAllString(b *testing.B) {
	pattern := "(?P<user>[\\w\\.+-]+)@(?P<domain>[\\w\\.-]+)\\.(?P<tld>[\\w\\.-]+)"
	stdReg := regexp.MustCompile(pattern)
	
	

	b.Run("golang std 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Contact us at support@example.com or sales@company.org for help"
		for b.Loop() {
			stdReg.FindAllStringSubmatch(input, -1)
		}
	})

	b.Run("regengo 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Contact us at support@example.com or sales@company.org for help"
		for b.Loop() {
			MultiEmail{}.FindAllString(input, -1)
		}
	})

	b.Run("regengo reuse 0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Contact us at support@example.com or sales@company.org for help"
		results := make([]*MultiEmailResult, 0, 10)
		for b.Loop() {
			results = MultiEmail{}.FindAllStringAppend(input, -1, results[:0])
		}
	})

	

	b.Run("golang std 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "Multiple: a@b.com, c@d.org, e@f.net"
		for b.Loop() {
			stdReg.FindAllStringSubmatch(input, -1)
		}
	})

	b.Run("regengo 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "Multiple: a@b.com, c@d.org, e@f.net"
		for b.Loop() {
			MultiEmail{}.FindAllString(input, -1)
		}
	})

	b.Run("regengo reuse 1", func(b *testing.B) {
		b.ReportAllocs()
		input := "Multiple: a@b.com, c@d.org, e@f.net"
		results := make([]*MultiEmailResult, 0, 10)
		for b.Loop() {
			results = MultiEmail{}.FindAllStringAppend(input, -1, results[:0])
		}
	})

	

	b.Run("golang std 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "No emails in this text"
		for b.Loop() {
			stdReg.FindAllStringSubmatch(input, -1)
		}
	})

	b.Run("regengo 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "No emails in this text"
		for b.Loop() {
			MultiEmail{}.FindAllString(input, -1)
		}
	})

	b.Run("regengo reuse 2", func(b *testing.B) {
		b.ReportAllocs()
		input := "No emails in this text"
		results := make([]*MultiEmailResult, 0, 10)
		for b.Loop() {
			results = MultiEmail{}.FindAllStringAppend(input, -1, results[:0])
		}
	})

	
}
