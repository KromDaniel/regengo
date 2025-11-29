package generated

import (
	"regexp"
	"testing"
)

// convertToStdlibTemplate converts regengo template syntax to stdlib syntax.
// regengo: $name, $1, $0 -> stdlib: ${name}, ${1}, ${0}
func convertToStdlibTemplateReplaceDate(template string) string {
	var result []byte
	i := 0
	for i < len(template) {
		if template[i] != '$' {
			result = append(result, template[i])
			i++
			continue
		}
		if i+1 >= len(template) {
			result = append(result, '$')
			i++
			continue
		}
		next := template[i+1]
		switch {
		case next == '$':
			result = append(result, '$', '$')
			i += 2
		case next == '{':
			closeIdx := i + 2
			for closeIdx < len(template) && template[closeIdx] != '}' {
				closeIdx++
			}
			if closeIdx < len(template) {
				result = append(result, template[i:closeIdx+1]...)
				i = closeIdx + 1
			} else {
				result = append(result, '$')
				i++
			}
		case next >= '0' && next <= '9':
			j := i + 1
			for j < len(template) && template[j] >= '0' && template[j] <= '9' {
				j++
			}
			result = append(result, '$', '{')
			result = append(result, template[i+1:j]...)
			result = append(result, '}')
			i = j
		case next == '_' || (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z'):
			j := i + 1
			for j < len(template) && (template[j] == '_' || (template[j] >= 'a' && template[j] <= 'z') || (template[j] >= 'A' && template[j] <= 'Z') || (template[j] >= '0' && template[j] <= '9')) {
				j++
			}
			result = append(result, '$', '{')
			result = append(result, template[i+1:j]...)
			result = append(result, '}')
			i = j
		default:
			result = append(result, '$')
			i++
		}
	}
	return string(result)
}

func TestReplaceDateReplaceAllString(t *testing.T) {
	pattern := "(?P<year>\\d{4})-(?P<month>\\d{2})-(?P<day>\\d{2})"
	stdReg := regexp.MustCompile(pattern)

	t.Run("replacer 0 input 0", func(t *testing.T) {
		input := "Event on 2024-01-15"
		replacer := "$month/$day/$year"
		stdlibTemplate := convertToStdlibTemplateReplaceDate(replacer)
		expected := stdReg.ReplaceAllString(input, stdlibTemplate)
		got := ReplaceDate{}.ReplaceAllString(input, replacer)
		if got != expected {
			t.Errorf("ReplaceAllString mismatch:\n  input=%q\n  replacer=%q\n  got=%q\n  want=%q", input, replacer, got, expected)
		}
	})

	t.Run("replacer 0 input 1", func(t *testing.T) {
		input := "Range: 2024-01-01 to 2024-12-31"
		replacer := "$month/$day/$year"
		stdlibTemplate := convertToStdlibTemplateReplaceDate(replacer)
		expected := stdReg.ReplaceAllString(input, stdlibTemplate)
		got := ReplaceDate{}.ReplaceAllString(input, replacer)
		if got != expected {
			t.Errorf("ReplaceAllString mismatch:\n  input=%q\n  replacer=%q\n  got=%q\n  want=%q", input, replacer, got, expected)
		}
	})

	t.Run("replacer 0 input 2", func(t *testing.T) {
		input := "2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 "
		replacer := "$month/$day/$year"
		stdlibTemplate := convertToStdlibTemplateReplaceDate(replacer)
		expected := stdReg.ReplaceAllString(input, stdlibTemplate)
		got := ReplaceDate{}.ReplaceAllString(input, replacer)
		if got != expected {
			t.Errorf("ReplaceAllString mismatch:\n  input=%q\n  replacer=%q\n  got=%q\n  want=%q", input, replacer, got, expected)
		}
	})

	t.Run("replacer 1 input 0", func(t *testing.T) {
		input := "Event on 2024-01-15"
		replacer := "[DATE]"
		stdlibTemplate := convertToStdlibTemplateReplaceDate(replacer)
		expected := stdReg.ReplaceAllString(input, stdlibTemplate)
		got := ReplaceDate{}.ReplaceAllString(input, replacer)
		if got != expected {
			t.Errorf("ReplaceAllString mismatch:\n  input=%q\n  replacer=%q\n  got=%q\n  want=%q", input, replacer, got, expected)
		}
	})

	t.Run("replacer 1 input 1", func(t *testing.T) {
		input := "Range: 2024-01-01 to 2024-12-31"
		replacer := "[DATE]"
		stdlibTemplate := convertToStdlibTemplateReplaceDate(replacer)
		expected := stdReg.ReplaceAllString(input, stdlibTemplate)
		got := ReplaceDate{}.ReplaceAllString(input, replacer)
		if got != expected {
			t.Errorf("ReplaceAllString mismatch:\n  input=%q\n  replacer=%q\n  got=%q\n  want=%q", input, replacer, got, expected)
		}
	})

	t.Run("replacer 1 input 2", func(t *testing.T) {
		input := "2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 "
		replacer := "[DATE]"
		stdlibTemplate := convertToStdlibTemplateReplaceDate(replacer)
		expected := stdReg.ReplaceAllString(input, stdlibTemplate)
		got := ReplaceDate{}.ReplaceAllString(input, replacer)
		if got != expected {
			t.Errorf("ReplaceAllString mismatch:\n  input=%q\n  replacer=%q\n  got=%q\n  want=%q", input, replacer, got, expected)
		}
	})

	t.Run("replacer 2 input 0", func(t *testing.T) {
		input := "Event on 2024-01-15"
		replacer := "$year"
		stdlibTemplate := convertToStdlibTemplateReplaceDate(replacer)
		expected := stdReg.ReplaceAllString(input, stdlibTemplate)
		got := ReplaceDate{}.ReplaceAllString(input, replacer)
		if got != expected {
			t.Errorf("ReplaceAllString mismatch:\n  input=%q\n  replacer=%q\n  got=%q\n  want=%q", input, replacer, got, expected)
		}
	})

	t.Run("replacer 2 input 1", func(t *testing.T) {
		input := "Range: 2024-01-01 to 2024-12-31"
		replacer := "$year"
		stdlibTemplate := convertToStdlibTemplateReplaceDate(replacer)
		expected := stdReg.ReplaceAllString(input, stdlibTemplate)
		got := ReplaceDate{}.ReplaceAllString(input, replacer)
		if got != expected {
			t.Errorf("ReplaceAllString mismatch:\n  input=%q\n  replacer=%q\n  got=%q\n  want=%q", input, replacer, got, expected)
		}
	})

	t.Run("replacer 2 input 2", func(t *testing.T) {
		input := "2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 "
		replacer := "$year"
		stdlibTemplate := convertToStdlibTemplateReplaceDate(replacer)
		expected := stdReg.ReplaceAllString(input, stdlibTemplate)
		got := ReplaceDate{}.ReplaceAllString(input, replacer)
		if got != expected {
			t.Errorf("ReplaceAllString mismatch:\n  input=%q\n  replacer=%q\n  got=%q\n  want=%q", input, replacer, got, expected)
		}
	})

}

func BenchmarkReplaceDateReplaceAllString(b *testing.B) {
	pattern := "(?P<year>\\d{4})-(?P<month>\\d{2})-(?P<day>\\d{2})"
	stdReg := regexp.MustCompile(pattern)

	b.Run("stdlib r0_i0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Event on 2024-01-15"
		stdlibTemplate := convertToStdlibTemplateReplaceDate("$month/$day/$year")
		for b.Loop() {
			stdReg.ReplaceAllString(input, stdlibTemplate)
		}
	})

	b.Run("regengo runtime r0_i0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Event on 2024-01-15"
		replacer := "$month/$day/$year"
		for b.Loop() {
			ReplaceDate{}.ReplaceAllString(input, replacer)
		}
	})

	b.Run("regengo precompiled0 i0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Event on 2024-01-15"
		for b.Loop() {
			ReplaceDate{}.ReplaceAllString0(input)
		}
	})

	b.Run("stdlib r0_i1", func(b *testing.B) {
		b.ReportAllocs()
		input := "Range: 2024-01-01 to 2024-12-31"
		stdlibTemplate := convertToStdlibTemplateReplaceDate("$month/$day/$year")
		for b.Loop() {
			stdReg.ReplaceAllString(input, stdlibTemplate)
		}
	})

	b.Run("regengo runtime r0_i1", func(b *testing.B) {
		b.ReportAllocs()
		input := "Range: 2024-01-01 to 2024-12-31"
		replacer := "$month/$day/$year"
		for b.Loop() {
			ReplaceDate{}.ReplaceAllString(input, replacer)
		}
	})

	b.Run("regengo precompiled0 i1", func(b *testing.B) {
		b.ReportAllocs()
		input := "Range: 2024-01-01 to 2024-12-31"
		for b.Loop() {
			ReplaceDate{}.ReplaceAllString0(input)
		}
	})

	b.Run("stdlib r0_i2", func(b *testing.B) {
		b.ReportAllocs()
		input := "2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 "
		stdlibTemplate := convertToStdlibTemplateReplaceDate("$month/$day/$year")
		for b.Loop() {
			stdReg.ReplaceAllString(input, stdlibTemplate)
		}
	})

	b.Run("regengo runtime r0_i2", func(b *testing.B) {
		b.ReportAllocs()
		input := "2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 "
		replacer := "$month/$day/$year"
		for b.Loop() {
			ReplaceDate{}.ReplaceAllString(input, replacer)
		}
	})

	b.Run("regengo precompiled0 i2", func(b *testing.B) {
		b.ReportAllocs()
		input := "2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 "
		for b.Loop() {
			ReplaceDate{}.ReplaceAllString0(input)
		}
	})

	b.Run("stdlib r1_i0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Event on 2024-01-15"
		stdlibTemplate := convertToStdlibTemplateReplaceDate("[DATE]")
		for b.Loop() {
			stdReg.ReplaceAllString(input, stdlibTemplate)
		}
	})

	b.Run("regengo runtime r1_i0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Event on 2024-01-15"
		replacer := "[DATE]"
		for b.Loop() {
			ReplaceDate{}.ReplaceAllString(input, replacer)
		}
	})

	b.Run("regengo precompiled1 i0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Event on 2024-01-15"
		for b.Loop() {
			ReplaceDate{}.ReplaceAllString1(input)
		}
	})

	b.Run("stdlib r1_i1", func(b *testing.B) {
		b.ReportAllocs()
		input := "Range: 2024-01-01 to 2024-12-31"
		stdlibTemplate := convertToStdlibTemplateReplaceDate("[DATE]")
		for b.Loop() {
			stdReg.ReplaceAllString(input, stdlibTemplate)
		}
	})

	b.Run("regengo runtime r1_i1", func(b *testing.B) {
		b.ReportAllocs()
		input := "Range: 2024-01-01 to 2024-12-31"
		replacer := "[DATE]"
		for b.Loop() {
			ReplaceDate{}.ReplaceAllString(input, replacer)
		}
	})

	b.Run("regengo precompiled1 i1", func(b *testing.B) {
		b.ReportAllocs()
		input := "Range: 2024-01-01 to 2024-12-31"
		for b.Loop() {
			ReplaceDate{}.ReplaceAllString1(input)
		}
	})

	b.Run("stdlib r1_i2", func(b *testing.B) {
		b.ReportAllocs()
		input := "2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 "
		stdlibTemplate := convertToStdlibTemplateReplaceDate("[DATE]")
		for b.Loop() {
			stdReg.ReplaceAllString(input, stdlibTemplate)
		}
	})

	b.Run("regengo runtime r1_i2", func(b *testing.B) {
		b.ReportAllocs()
		input := "2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 "
		replacer := "[DATE]"
		for b.Loop() {
			ReplaceDate{}.ReplaceAllString(input, replacer)
		}
	})

	b.Run("regengo precompiled1 i2", func(b *testing.B) {
		b.ReportAllocs()
		input := "2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 "
		for b.Loop() {
			ReplaceDate{}.ReplaceAllString1(input)
		}
	})

	b.Run("stdlib r2_i0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Event on 2024-01-15"
		stdlibTemplate := convertToStdlibTemplateReplaceDate("$year")
		for b.Loop() {
			stdReg.ReplaceAllString(input, stdlibTemplate)
		}
	})

	b.Run("regengo runtime r2_i0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Event on 2024-01-15"
		replacer := "$year"
		for b.Loop() {
			ReplaceDate{}.ReplaceAllString(input, replacer)
		}
	})

	b.Run("regengo precompiled2 i0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Event on 2024-01-15"
		for b.Loop() {
			ReplaceDate{}.ReplaceAllString2(input)
		}
	})

	b.Run("stdlib r2_i1", func(b *testing.B) {
		b.ReportAllocs()
		input := "Range: 2024-01-01 to 2024-12-31"
		stdlibTemplate := convertToStdlibTemplateReplaceDate("$year")
		for b.Loop() {
			stdReg.ReplaceAllString(input, stdlibTemplate)
		}
	})

	b.Run("regengo runtime r2_i1", func(b *testing.B) {
		b.ReportAllocs()
		input := "Range: 2024-01-01 to 2024-12-31"
		replacer := "$year"
		for b.Loop() {
			ReplaceDate{}.ReplaceAllString(input, replacer)
		}
	})

	b.Run("regengo precompiled2 i1", func(b *testing.B) {
		b.ReportAllocs()
		input := "Range: 2024-01-01 to 2024-12-31"
		for b.Loop() {
			ReplaceDate{}.ReplaceAllString2(input)
		}
	})

	b.Run("stdlib r2_i2", func(b *testing.B) {
		b.ReportAllocs()
		input := "2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 "
		stdlibTemplate := convertToStdlibTemplateReplaceDate("$year")
		for b.Loop() {
			stdReg.ReplaceAllString(input, stdlibTemplate)
		}
	})

	b.Run("regengo runtime r2_i2", func(b *testing.B) {
		b.ReportAllocs()
		input := "2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 "
		replacer := "$year"
		for b.Loop() {
			ReplaceDate{}.ReplaceAllString(input, replacer)
		}
	})

	b.Run("regengo precompiled2 i2", func(b *testing.B) {
		b.ReportAllocs()
		input := "2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 "
		for b.Loop() {
			ReplaceDate{}.ReplaceAllString2(input)
		}
	})

}

func BenchmarkReplaceDateReplaceAllBytesAppend(b *testing.B) {

	b.Run("precompiled0 i0", func(b *testing.B) {
		b.ReportAllocs()
		input := []byte("Event on 2024-01-15")
		buf := make([]byte, 0, 4096)
		for b.Loop() {
			buf = ReplaceDate{}.ReplaceAllBytesAppend0(input, buf)
			buf = buf[:0]
		}
	})

	b.Run("precompiled0 i1", func(b *testing.B) {
		b.ReportAllocs()
		input := []byte("Range: 2024-01-01 to 2024-12-31")
		buf := make([]byte, 0, 4096)
		for b.Loop() {
			buf = ReplaceDate{}.ReplaceAllBytesAppend0(input, buf)
			buf = buf[:0]
		}
	})

	b.Run("precompiled0 i2", func(b *testing.B) {
		b.ReportAllocs()
		input := []byte("2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 ")
		buf := make([]byte, 0, 4096)
		for b.Loop() {
			buf = ReplaceDate{}.ReplaceAllBytesAppend0(input, buf)
			buf = buf[:0]
		}
	})

	b.Run("precompiled1 i0", func(b *testing.B) {
		b.ReportAllocs()
		input := []byte("Event on 2024-01-15")
		buf := make([]byte, 0, 4096)
		for b.Loop() {
			buf = ReplaceDate{}.ReplaceAllBytesAppend1(input, buf)
			buf = buf[:0]
		}
	})

	b.Run("precompiled1 i1", func(b *testing.B) {
		b.ReportAllocs()
		input := []byte("Range: 2024-01-01 to 2024-12-31")
		buf := make([]byte, 0, 4096)
		for b.Loop() {
			buf = ReplaceDate{}.ReplaceAllBytesAppend1(input, buf)
			buf = buf[:0]
		}
	})

	b.Run("precompiled1 i2", func(b *testing.B) {
		b.ReportAllocs()
		input := []byte("2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 ")
		buf := make([]byte, 0, 4096)
		for b.Loop() {
			buf = ReplaceDate{}.ReplaceAllBytesAppend1(input, buf)
			buf = buf[:0]
		}
	})

	b.Run("precompiled2 i0", func(b *testing.B) {
		b.ReportAllocs()
		input := []byte("Event on 2024-01-15")
		buf := make([]byte, 0, 4096)
		for b.Loop() {
			buf = ReplaceDate{}.ReplaceAllBytesAppend2(input, buf)
			buf = buf[:0]
		}
	})

	b.Run("precompiled2 i1", func(b *testing.B) {
		b.ReportAllocs()
		input := []byte("Range: 2024-01-01 to 2024-12-31")
		buf := make([]byte, 0, 4096)
		for b.Loop() {
			buf = ReplaceDate{}.ReplaceAllBytesAppend2(input, buf)
			buf = buf[:0]
		}
	})

	b.Run("precompiled2 i2", func(b *testing.B) {
		b.ReportAllocs()
		input := []byte("2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 2024-06-15 ")
		buf := make([]byte, 0, 4096)
		for b.Loop() {
			buf = ReplaceDate{}.ReplaceAllBytesAppend2(input, buf)
			buf = buf[:0]
		}
	})

}
