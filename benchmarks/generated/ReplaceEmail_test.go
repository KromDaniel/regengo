package generated

import (
	"regexp"
	"testing"
)

// convertToStdlibTemplate converts regengo template syntax to stdlib syntax.
// regengo: $name, $1, $0 -> stdlib: ${name}, ${1}, ${0}
func convertToStdlibTemplateReplaceEmail(template string) string {
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

func TestReplaceEmailReplaceAllString(t *testing.T) {
	pattern := "(?P<user>[\\w\\.+-]+)@(?P<domain>[\\w\\.-]+)\\.(?P<tld>[\\w\\.-]+)"
	stdReg := regexp.MustCompile(pattern)

	t.Run("replacer 0 input 0", func(t *testing.T) {
		input := "Contact support@example.com for help"
		replacer := "$user@REDACTED.$tld"
		stdlibTemplate := convertToStdlibTemplateReplaceEmail(replacer)
		expected := stdReg.ReplaceAllString(input, stdlibTemplate)
		got := ReplaceEmail{}.ReplaceAllString(input, replacer)
		if got != expected {
			t.Errorf("ReplaceAllString mismatch:\n  input=%q\n  replacer=%q\n  got=%q\n  want=%q", input, replacer, got, expected)
		}
	})

	t.Run("replacer 0 input 1", func(t *testing.T) {
		input := "Multiple: a@b.com, c@d.org, e@f.net in one line"
		replacer := "$user@REDACTED.$tld"
		stdlibTemplate := convertToStdlibTemplateReplaceEmail(replacer)
		expected := stdReg.ReplaceAllString(input, stdlibTemplate)
		got := ReplaceEmail{}.ReplaceAllString(input, replacer)
		if got != expected {
			t.Errorf("ReplaceAllString mismatch:\n  input=%q\n  replacer=%q\n  got=%q\n  want=%q", input, replacer, got, expected)
		}
	})

	t.Run("replacer 0 input 2", func(t *testing.T) {
		input := "user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com "
		replacer := "$user@REDACTED.$tld"
		stdlibTemplate := convertToStdlibTemplateReplaceEmail(replacer)
		expected := stdReg.ReplaceAllString(input, stdlibTemplate)
		got := ReplaceEmail{}.ReplaceAllString(input, replacer)
		if got != expected {
			t.Errorf("ReplaceAllString mismatch:\n  input=%q\n  replacer=%q\n  got=%q\n  want=%q", input, replacer, got, expected)
		}
	})

	t.Run("replacer 1 input 0", func(t *testing.T) {
		input := "Contact support@example.com for help"
		replacer := "[EMAIL]"
		stdlibTemplate := convertToStdlibTemplateReplaceEmail(replacer)
		expected := stdReg.ReplaceAllString(input, stdlibTemplate)
		got := ReplaceEmail{}.ReplaceAllString(input, replacer)
		if got != expected {
			t.Errorf("ReplaceAllString mismatch:\n  input=%q\n  replacer=%q\n  got=%q\n  want=%q", input, replacer, got, expected)
		}
	})

	t.Run("replacer 1 input 1", func(t *testing.T) {
		input := "Multiple: a@b.com, c@d.org, e@f.net in one line"
		replacer := "[EMAIL]"
		stdlibTemplate := convertToStdlibTemplateReplaceEmail(replacer)
		expected := stdReg.ReplaceAllString(input, stdlibTemplate)
		got := ReplaceEmail{}.ReplaceAllString(input, replacer)
		if got != expected {
			t.Errorf("ReplaceAllString mismatch:\n  input=%q\n  replacer=%q\n  got=%q\n  want=%q", input, replacer, got, expected)
		}
	})

	t.Run("replacer 1 input 2", func(t *testing.T) {
		input := "user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com "
		replacer := "[EMAIL]"
		stdlibTemplate := convertToStdlibTemplateReplaceEmail(replacer)
		expected := stdReg.ReplaceAllString(input, stdlibTemplate)
		got := ReplaceEmail{}.ReplaceAllString(input, replacer)
		if got != expected {
			t.Errorf("ReplaceAllString mismatch:\n  input=%q\n  replacer=%q\n  got=%q\n  want=%q", input, replacer, got, expected)
		}
	})

	t.Run("replacer 2 input 0", func(t *testing.T) {
		input := "Contact support@example.com for help"
		replacer := "$0"
		stdlibTemplate := convertToStdlibTemplateReplaceEmail(replacer)
		expected := stdReg.ReplaceAllString(input, stdlibTemplate)
		got := ReplaceEmail{}.ReplaceAllString(input, replacer)
		if got != expected {
			t.Errorf("ReplaceAllString mismatch:\n  input=%q\n  replacer=%q\n  got=%q\n  want=%q", input, replacer, got, expected)
		}
	})

	t.Run("replacer 2 input 1", func(t *testing.T) {
		input := "Multiple: a@b.com, c@d.org, e@f.net in one line"
		replacer := "$0"
		stdlibTemplate := convertToStdlibTemplateReplaceEmail(replacer)
		expected := stdReg.ReplaceAllString(input, stdlibTemplate)
		got := ReplaceEmail{}.ReplaceAllString(input, replacer)
		if got != expected {
			t.Errorf("ReplaceAllString mismatch:\n  input=%q\n  replacer=%q\n  got=%q\n  want=%q", input, replacer, got, expected)
		}
	})

	t.Run("replacer 2 input 2", func(t *testing.T) {
		input := "user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com "
		replacer := "$0"
		stdlibTemplate := convertToStdlibTemplateReplaceEmail(replacer)
		expected := stdReg.ReplaceAllString(input, stdlibTemplate)
		got := ReplaceEmail{}.ReplaceAllString(input, replacer)
		if got != expected {
			t.Errorf("ReplaceAllString mismatch:\n  input=%q\n  replacer=%q\n  got=%q\n  want=%q", input, replacer, got, expected)
		}
	})

}

func BenchmarkReplaceEmailReplaceAllString(b *testing.B) {
	pattern := "(?P<user>[\\w\\.+-]+)@(?P<domain>[\\w\\.-]+)\\.(?P<tld>[\\w\\.-]+)"
	stdReg := regexp.MustCompile(pattern)

	b.Run("stdlib r0_i0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Contact support@example.com for help"
		stdlibTemplate := convertToStdlibTemplateReplaceEmail("$user@REDACTED.$tld")
		for b.Loop() {
			stdReg.ReplaceAllString(input, stdlibTemplate)
		}
	})

	b.Run("regengo runtime r0_i0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Contact support@example.com for help"
		replacer := "$user@REDACTED.$tld"
		for b.Loop() {
			ReplaceEmail{}.ReplaceAllString(input, replacer)
		}
	})

	b.Run("regengo precompiled0 i0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Contact support@example.com for help"
		for b.Loop() {
			ReplaceEmail{}.ReplaceAllString0(input)
		}
	})

	b.Run("stdlib r0_i1", func(b *testing.B) {
		b.ReportAllocs()
		input := "Multiple: a@b.com, c@d.org, e@f.net in one line"
		stdlibTemplate := convertToStdlibTemplateReplaceEmail("$user@REDACTED.$tld")
		for b.Loop() {
			stdReg.ReplaceAllString(input, stdlibTemplate)
		}
	})

	b.Run("regengo runtime r0_i1", func(b *testing.B) {
		b.ReportAllocs()
		input := "Multiple: a@b.com, c@d.org, e@f.net in one line"
		replacer := "$user@REDACTED.$tld"
		for b.Loop() {
			ReplaceEmail{}.ReplaceAllString(input, replacer)
		}
	})

	b.Run("regengo precompiled0 i1", func(b *testing.B) {
		b.ReportAllocs()
		input := "Multiple: a@b.com, c@d.org, e@f.net in one line"
		for b.Loop() {
			ReplaceEmail{}.ReplaceAllString0(input)
		}
	})

	b.Run("stdlib r0_i2", func(b *testing.B) {
		b.ReportAllocs()
		input := "user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com "
		stdlibTemplate := convertToStdlibTemplateReplaceEmail("$user@REDACTED.$tld")
		for b.Loop() {
			stdReg.ReplaceAllString(input, stdlibTemplate)
		}
	})

	b.Run("regengo runtime r0_i2", func(b *testing.B) {
		b.ReportAllocs()
		input := "user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com "
		replacer := "$user@REDACTED.$tld"
		for b.Loop() {
			ReplaceEmail{}.ReplaceAllString(input, replacer)
		}
	})

	b.Run("regengo precompiled0 i2", func(b *testing.B) {
		b.ReportAllocs()
		input := "user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com "
		for b.Loop() {
			ReplaceEmail{}.ReplaceAllString0(input)
		}
	})

	b.Run("stdlib r1_i0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Contact support@example.com for help"
		stdlibTemplate := convertToStdlibTemplateReplaceEmail("[EMAIL]")
		for b.Loop() {
			stdReg.ReplaceAllString(input, stdlibTemplate)
		}
	})

	b.Run("regengo runtime r1_i0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Contact support@example.com for help"
		replacer := "[EMAIL]"
		for b.Loop() {
			ReplaceEmail{}.ReplaceAllString(input, replacer)
		}
	})

	b.Run("regengo precompiled1 i0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Contact support@example.com for help"
		for b.Loop() {
			ReplaceEmail{}.ReplaceAllString1(input)
		}
	})

	b.Run("stdlib r1_i1", func(b *testing.B) {
		b.ReportAllocs()
		input := "Multiple: a@b.com, c@d.org, e@f.net in one line"
		stdlibTemplate := convertToStdlibTemplateReplaceEmail("[EMAIL]")
		for b.Loop() {
			stdReg.ReplaceAllString(input, stdlibTemplate)
		}
	})

	b.Run("regengo runtime r1_i1", func(b *testing.B) {
		b.ReportAllocs()
		input := "Multiple: a@b.com, c@d.org, e@f.net in one line"
		replacer := "[EMAIL]"
		for b.Loop() {
			ReplaceEmail{}.ReplaceAllString(input, replacer)
		}
	})

	b.Run("regengo precompiled1 i1", func(b *testing.B) {
		b.ReportAllocs()
		input := "Multiple: a@b.com, c@d.org, e@f.net in one line"
		for b.Loop() {
			ReplaceEmail{}.ReplaceAllString1(input)
		}
	})

	b.Run("stdlib r1_i2", func(b *testing.B) {
		b.ReportAllocs()
		input := "user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com "
		stdlibTemplate := convertToStdlibTemplateReplaceEmail("[EMAIL]")
		for b.Loop() {
			stdReg.ReplaceAllString(input, stdlibTemplate)
		}
	})

	b.Run("regengo runtime r1_i2", func(b *testing.B) {
		b.ReportAllocs()
		input := "user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com "
		replacer := "[EMAIL]"
		for b.Loop() {
			ReplaceEmail{}.ReplaceAllString(input, replacer)
		}
	})

	b.Run("regengo precompiled1 i2", func(b *testing.B) {
		b.ReportAllocs()
		input := "user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com "
		for b.Loop() {
			ReplaceEmail{}.ReplaceAllString1(input)
		}
	})

	b.Run("stdlib r2_i0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Contact support@example.com for help"
		stdlibTemplate := convertToStdlibTemplateReplaceEmail("$0")
		for b.Loop() {
			stdReg.ReplaceAllString(input, stdlibTemplate)
		}
	})

	b.Run("regengo runtime r2_i0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Contact support@example.com for help"
		replacer := "$0"
		for b.Loop() {
			ReplaceEmail{}.ReplaceAllString(input, replacer)
		}
	})

	b.Run("regengo precompiled2 i0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Contact support@example.com for help"
		for b.Loop() {
			ReplaceEmail{}.ReplaceAllString2(input)
		}
	})

	b.Run("stdlib r2_i1", func(b *testing.B) {
		b.ReportAllocs()
		input := "Multiple: a@b.com, c@d.org, e@f.net in one line"
		stdlibTemplate := convertToStdlibTemplateReplaceEmail("$0")
		for b.Loop() {
			stdReg.ReplaceAllString(input, stdlibTemplate)
		}
	})

	b.Run("regengo runtime r2_i1", func(b *testing.B) {
		b.ReportAllocs()
		input := "Multiple: a@b.com, c@d.org, e@f.net in one line"
		replacer := "$0"
		for b.Loop() {
			ReplaceEmail{}.ReplaceAllString(input, replacer)
		}
	})

	b.Run("regengo precompiled2 i1", func(b *testing.B) {
		b.ReportAllocs()
		input := "Multiple: a@b.com, c@d.org, e@f.net in one line"
		for b.Loop() {
			ReplaceEmail{}.ReplaceAllString2(input)
		}
	})

	b.Run("stdlib r2_i2", func(b *testing.B) {
		b.ReportAllocs()
		input := "user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com "
		stdlibTemplate := convertToStdlibTemplateReplaceEmail("$0")
		for b.Loop() {
			stdReg.ReplaceAllString(input, stdlibTemplate)
		}
	})

	b.Run("regengo runtime r2_i2", func(b *testing.B) {
		b.ReportAllocs()
		input := "user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com "
		replacer := "$0"
		for b.Loop() {
			ReplaceEmail{}.ReplaceAllString(input, replacer)
		}
	})

	b.Run("regengo precompiled2 i2", func(b *testing.B) {
		b.ReportAllocs()
		input := "user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com "
		for b.Loop() {
			ReplaceEmail{}.ReplaceAllString2(input)
		}
	})

}

func BenchmarkReplaceEmailReplaceAllBytesAppend(b *testing.B) {

	b.Run("precompiled0 i0", func(b *testing.B) {
		b.ReportAllocs()
		input := []byte("Contact support@example.com for help")
		buf := make([]byte, 0, 4096)
		for b.Loop() {
			buf = ReplaceEmail{}.ReplaceAllBytesAppend0(input, buf)
			buf = buf[:0]
		}
	})

	b.Run("precompiled0 i1", func(b *testing.B) {
		b.ReportAllocs()
		input := []byte("Multiple: a@b.com, c@d.org, e@f.net in one line")
		buf := make([]byte, 0, 4096)
		for b.Loop() {
			buf = ReplaceEmail{}.ReplaceAllBytesAppend0(input, buf)
			buf = buf[:0]
		}
	})

	b.Run("precompiled0 i2", func(b *testing.B) {
		b.ReportAllocs()
		input := []byte("user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com ")
		buf := make([]byte, 0, 4096)
		for b.Loop() {
			buf = ReplaceEmail{}.ReplaceAllBytesAppend0(input, buf)
			buf = buf[:0]
		}
	})

	b.Run("precompiled1 i0", func(b *testing.B) {
		b.ReportAllocs()
		input := []byte("Contact support@example.com for help")
		buf := make([]byte, 0, 4096)
		for b.Loop() {
			buf = ReplaceEmail{}.ReplaceAllBytesAppend1(input, buf)
			buf = buf[:0]
		}
	})

	b.Run("precompiled1 i1", func(b *testing.B) {
		b.ReportAllocs()
		input := []byte("Multiple: a@b.com, c@d.org, e@f.net in one line")
		buf := make([]byte, 0, 4096)
		for b.Loop() {
			buf = ReplaceEmail{}.ReplaceAllBytesAppend1(input, buf)
			buf = buf[:0]
		}
	})

	b.Run("precompiled1 i2", func(b *testing.B) {
		b.ReportAllocs()
		input := []byte("user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com ")
		buf := make([]byte, 0, 4096)
		for b.Loop() {
			buf = ReplaceEmail{}.ReplaceAllBytesAppend1(input, buf)
			buf = buf[:0]
		}
	})

	b.Run("precompiled2 i0", func(b *testing.B) {
		b.ReportAllocs()
		input := []byte("Contact support@example.com for help")
		buf := make([]byte, 0, 4096)
		for b.Loop() {
			buf = ReplaceEmail{}.ReplaceAllBytesAppend2(input, buf)
			buf = buf[:0]
		}
	})

	b.Run("precompiled2 i1", func(b *testing.B) {
		b.ReportAllocs()
		input := []byte("Multiple: a@b.com, c@d.org, e@f.net in one line")
		buf := make([]byte, 0, 4096)
		for b.Loop() {
			buf = ReplaceEmail{}.ReplaceAllBytesAppend2(input, buf)
			buf = buf[:0]
		}
	})

	b.Run("precompiled2 i2", func(b *testing.B) {
		b.ReportAllocs()
		input := []byte("user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com user@example.com ")
		buf := make([]byte, 0, 4096)
		for b.Loop() {
			buf = ReplaceEmail{}.ReplaceAllBytesAppend2(input, buf)
			buf = buf[:0]
		}
	})

}
