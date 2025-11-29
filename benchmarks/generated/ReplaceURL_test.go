package generated

import (
	"regexp"
	"testing"
)

// convertToStdlibTemplate converts regengo template syntax to stdlib syntax.
// regengo: $name, $1, $0 -> stdlib: ${name}, ${1}, ${0}
func convertToStdlibTemplateReplaceURL(template string) string {
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

func TestReplaceURLReplaceAllString(t *testing.T) {
	pattern := "(?P<protocol>https?)://(?P<host>[\\w\\.-]+)(?::(?P<port>\\d+))?(?P<path>/[\\w\\./]*)?"
	stdReg := regexp.MustCompile(pattern)

	t.Run("replacer 0 input 0", func(t *testing.T) {
		input := "Visit https://example.com/page for info"
		replacer := "$protocol://$host[REDACTED]"
		stdlibTemplate := convertToStdlibTemplateReplaceURL(replacer)
		expected := stdReg.ReplaceAllString(input, stdlibTemplate)
		got := ReplaceURL{}.ReplaceAllString(input, replacer)
		if got != expected {
			t.Errorf("ReplaceAllString mismatch:\n  input=%q\n  replacer=%q\n  got=%q\n  want=%q", input, replacer, got, expected)
		}
	})

	t.Run("replacer 0 input 1", func(t *testing.T) {
		input := "API at http://localhost:8080/api/v1/users"
		replacer := "$protocol://$host[REDACTED]"
		stdlibTemplate := convertToStdlibTemplateReplaceURL(replacer)
		expected := stdReg.ReplaceAllString(input, stdlibTemplate)
		got := ReplaceURL{}.ReplaceAllString(input, replacer)
		if got != expected {
			t.Errorf("ReplaceAllString mismatch:\n  input=%q\n  replacer=%q\n  got=%q\n  want=%q", input, replacer, got, expected)
		}
	})

	t.Run("replacer 0 input 2", func(t *testing.T) {
		input := "https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path "
		replacer := "$protocol://$host[REDACTED]"
		stdlibTemplate := convertToStdlibTemplateReplaceURL(replacer)
		expected := stdReg.ReplaceAllString(input, stdlibTemplate)
		got := ReplaceURL{}.ReplaceAllString(input, replacer)
		if got != expected {
			t.Errorf("ReplaceAllString mismatch:\n  input=%q\n  replacer=%q\n  got=%q\n  want=%q", input, replacer, got, expected)
		}
	})

	t.Run("replacer 1 input 0", func(t *testing.T) {
		input := "Visit https://example.com/page for info"
		replacer := "[URL]"
		stdlibTemplate := convertToStdlibTemplateReplaceURL(replacer)
		expected := stdReg.ReplaceAllString(input, stdlibTemplate)
		got := ReplaceURL{}.ReplaceAllString(input, replacer)
		if got != expected {
			t.Errorf("ReplaceAllString mismatch:\n  input=%q\n  replacer=%q\n  got=%q\n  want=%q", input, replacer, got, expected)
		}
	})

	t.Run("replacer 1 input 1", func(t *testing.T) {
		input := "API at http://localhost:8080/api/v1/users"
		replacer := "[URL]"
		stdlibTemplate := convertToStdlibTemplateReplaceURL(replacer)
		expected := stdReg.ReplaceAllString(input, stdlibTemplate)
		got := ReplaceURL{}.ReplaceAllString(input, replacer)
		if got != expected {
			t.Errorf("ReplaceAllString mismatch:\n  input=%q\n  replacer=%q\n  got=%q\n  want=%q", input, replacer, got, expected)
		}
	})

	t.Run("replacer 1 input 2", func(t *testing.T) {
		input := "https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path "
		replacer := "[URL]"
		stdlibTemplate := convertToStdlibTemplateReplaceURL(replacer)
		expected := stdReg.ReplaceAllString(input, stdlibTemplate)
		got := ReplaceURL{}.ReplaceAllString(input, replacer)
		if got != expected {
			t.Errorf("ReplaceAllString mismatch:\n  input=%q\n  replacer=%q\n  got=%q\n  want=%q", input, replacer, got, expected)
		}
	})

	t.Run("replacer 2 input 0", func(t *testing.T) {
		input := "Visit https://example.com/page for info"
		replacer := "$host"
		stdlibTemplate := convertToStdlibTemplateReplaceURL(replacer)
		expected := stdReg.ReplaceAllString(input, stdlibTemplate)
		got := ReplaceURL{}.ReplaceAllString(input, replacer)
		if got != expected {
			t.Errorf("ReplaceAllString mismatch:\n  input=%q\n  replacer=%q\n  got=%q\n  want=%q", input, replacer, got, expected)
		}
	})

	t.Run("replacer 2 input 1", func(t *testing.T) {
		input := "API at http://localhost:8080/api/v1/users"
		replacer := "$host"
		stdlibTemplate := convertToStdlibTemplateReplaceURL(replacer)
		expected := stdReg.ReplaceAllString(input, stdlibTemplate)
		got := ReplaceURL{}.ReplaceAllString(input, replacer)
		if got != expected {
			t.Errorf("ReplaceAllString mismatch:\n  input=%q\n  replacer=%q\n  got=%q\n  want=%q", input, replacer, got, expected)
		}
	})

	t.Run("replacer 2 input 2", func(t *testing.T) {
		input := "https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path "
		replacer := "$host"
		stdlibTemplate := convertToStdlibTemplateReplaceURL(replacer)
		expected := stdReg.ReplaceAllString(input, stdlibTemplate)
		got := ReplaceURL{}.ReplaceAllString(input, replacer)
		if got != expected {
			t.Errorf("ReplaceAllString mismatch:\n  input=%q\n  replacer=%q\n  got=%q\n  want=%q", input, replacer, got, expected)
		}
	})

}

func BenchmarkReplaceURLReplaceAllString(b *testing.B) {
	pattern := "(?P<protocol>https?)://(?P<host>[\\w\\.-]+)(?::(?P<port>\\d+))?(?P<path>/[\\w\\./]*)?"
	stdReg := regexp.MustCompile(pattern)

	b.Run("stdlib r0_i0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Visit https://example.com/page for info"
		stdlibTemplate := convertToStdlibTemplateReplaceURL("$protocol://$host[REDACTED]")
		for b.Loop() {
			stdReg.ReplaceAllString(input, stdlibTemplate)
		}
	})

	b.Run("regengo runtime r0_i0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Visit https://example.com/page for info"
		replacer := "$protocol://$host[REDACTED]"
		for b.Loop() {
			ReplaceURL{}.ReplaceAllString(input, replacer)
		}
	})

	b.Run("regengo precompiled0 i0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Visit https://example.com/page for info"
		for b.Loop() {
			ReplaceURL{}.ReplaceAllString0(input)
		}
	})

	b.Run("stdlib r0_i1", func(b *testing.B) {
		b.ReportAllocs()
		input := "API at http://localhost:8080/api/v1/users"
		stdlibTemplate := convertToStdlibTemplateReplaceURL("$protocol://$host[REDACTED]")
		for b.Loop() {
			stdReg.ReplaceAllString(input, stdlibTemplate)
		}
	})

	b.Run("regengo runtime r0_i1", func(b *testing.B) {
		b.ReportAllocs()
		input := "API at http://localhost:8080/api/v1/users"
		replacer := "$protocol://$host[REDACTED]"
		for b.Loop() {
			ReplaceURL{}.ReplaceAllString(input, replacer)
		}
	})

	b.Run("regengo precompiled0 i1", func(b *testing.B) {
		b.ReportAllocs()
		input := "API at http://localhost:8080/api/v1/users"
		for b.Loop() {
			ReplaceURL{}.ReplaceAllString0(input)
		}
	})

	b.Run("stdlib r0_i2", func(b *testing.B) {
		b.ReportAllocs()
		input := "https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path "
		stdlibTemplate := convertToStdlibTemplateReplaceURL("$protocol://$host[REDACTED]")
		for b.Loop() {
			stdReg.ReplaceAllString(input, stdlibTemplate)
		}
	})

	b.Run("regengo runtime r0_i2", func(b *testing.B) {
		b.ReportAllocs()
		input := "https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path "
		replacer := "$protocol://$host[REDACTED]"
		for b.Loop() {
			ReplaceURL{}.ReplaceAllString(input, replacer)
		}
	})

	b.Run("regengo precompiled0 i2", func(b *testing.B) {
		b.ReportAllocs()
		input := "https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path "
		for b.Loop() {
			ReplaceURL{}.ReplaceAllString0(input)
		}
	})

	b.Run("stdlib r1_i0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Visit https://example.com/page for info"
		stdlibTemplate := convertToStdlibTemplateReplaceURL("[URL]")
		for b.Loop() {
			stdReg.ReplaceAllString(input, stdlibTemplate)
		}
	})

	b.Run("regengo runtime r1_i0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Visit https://example.com/page for info"
		replacer := "[URL]"
		for b.Loop() {
			ReplaceURL{}.ReplaceAllString(input, replacer)
		}
	})

	b.Run("regengo precompiled1 i0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Visit https://example.com/page for info"
		for b.Loop() {
			ReplaceURL{}.ReplaceAllString1(input)
		}
	})

	b.Run("stdlib r1_i1", func(b *testing.B) {
		b.ReportAllocs()
		input := "API at http://localhost:8080/api/v1/users"
		stdlibTemplate := convertToStdlibTemplateReplaceURL("[URL]")
		for b.Loop() {
			stdReg.ReplaceAllString(input, stdlibTemplate)
		}
	})

	b.Run("regengo runtime r1_i1", func(b *testing.B) {
		b.ReportAllocs()
		input := "API at http://localhost:8080/api/v1/users"
		replacer := "[URL]"
		for b.Loop() {
			ReplaceURL{}.ReplaceAllString(input, replacer)
		}
	})

	b.Run("regengo precompiled1 i1", func(b *testing.B) {
		b.ReportAllocs()
		input := "API at http://localhost:8080/api/v1/users"
		for b.Loop() {
			ReplaceURL{}.ReplaceAllString1(input)
		}
	})

	b.Run("stdlib r1_i2", func(b *testing.B) {
		b.ReportAllocs()
		input := "https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path "
		stdlibTemplate := convertToStdlibTemplateReplaceURL("[URL]")
		for b.Loop() {
			stdReg.ReplaceAllString(input, stdlibTemplate)
		}
	})

	b.Run("regengo runtime r1_i2", func(b *testing.B) {
		b.ReportAllocs()
		input := "https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path "
		replacer := "[URL]"
		for b.Loop() {
			ReplaceURL{}.ReplaceAllString(input, replacer)
		}
	})

	b.Run("regengo precompiled1 i2", func(b *testing.B) {
		b.ReportAllocs()
		input := "https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path "
		for b.Loop() {
			ReplaceURL{}.ReplaceAllString1(input)
		}
	})

	b.Run("stdlib r2_i0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Visit https://example.com/page for info"
		stdlibTemplate := convertToStdlibTemplateReplaceURL("$host")
		for b.Loop() {
			stdReg.ReplaceAllString(input, stdlibTemplate)
		}
	})

	b.Run("regengo runtime r2_i0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Visit https://example.com/page for info"
		replacer := "$host"
		for b.Loop() {
			ReplaceURL{}.ReplaceAllString(input, replacer)
		}
	})

	b.Run("regengo precompiled2 i0", func(b *testing.B) {
		b.ReportAllocs()
		input := "Visit https://example.com/page for info"
		for b.Loop() {
			ReplaceURL{}.ReplaceAllString2(input)
		}
	})

	b.Run("stdlib r2_i1", func(b *testing.B) {
		b.ReportAllocs()
		input := "API at http://localhost:8080/api/v1/users"
		stdlibTemplate := convertToStdlibTemplateReplaceURL("$host")
		for b.Loop() {
			stdReg.ReplaceAllString(input, stdlibTemplate)
		}
	})

	b.Run("regengo runtime r2_i1", func(b *testing.B) {
		b.ReportAllocs()
		input := "API at http://localhost:8080/api/v1/users"
		replacer := "$host"
		for b.Loop() {
			ReplaceURL{}.ReplaceAllString(input, replacer)
		}
	})

	b.Run("regengo precompiled2 i1", func(b *testing.B) {
		b.ReportAllocs()
		input := "API at http://localhost:8080/api/v1/users"
		for b.Loop() {
			ReplaceURL{}.ReplaceAllString2(input)
		}
	})

	b.Run("stdlib r2_i2", func(b *testing.B) {
		b.ReportAllocs()
		input := "https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path "
		stdlibTemplate := convertToStdlibTemplateReplaceURL("$host")
		for b.Loop() {
			stdReg.ReplaceAllString(input, stdlibTemplate)
		}
	})

	b.Run("regengo runtime r2_i2", func(b *testing.B) {
		b.ReportAllocs()
		input := "https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path "
		replacer := "$host"
		for b.Loop() {
			ReplaceURL{}.ReplaceAllString(input, replacer)
		}
	})

	b.Run("regengo precompiled2 i2", func(b *testing.B) {
		b.ReportAllocs()
		input := "https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path "
		for b.Loop() {
			ReplaceURL{}.ReplaceAllString2(input)
		}
	})

}

func BenchmarkReplaceURLReplaceAllBytesAppend(b *testing.B) {

	b.Run("precompiled0 i0", func(b *testing.B) {
		b.ReportAllocs()
		input := []byte("Visit https://example.com/page for info")
		buf := make([]byte, 0, 4096)
		for b.Loop() {
			buf = ReplaceURL{}.ReplaceAllBytesAppend0(input, buf)
			buf = buf[:0]
		}
	})

	b.Run("precompiled0 i1", func(b *testing.B) {
		b.ReportAllocs()
		input := []byte("API at http://localhost:8080/api/v1/users")
		buf := make([]byte, 0, 4096)
		for b.Loop() {
			buf = ReplaceURL{}.ReplaceAllBytesAppend0(input, buf)
			buf = buf[:0]
		}
	})

	b.Run("precompiled0 i2", func(b *testing.B) {
		b.ReportAllocs()
		input := []byte("https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path ")
		buf := make([]byte, 0, 4096)
		for b.Loop() {
			buf = ReplaceURL{}.ReplaceAllBytesAppend0(input, buf)
			buf = buf[:0]
		}
	})

	b.Run("precompiled1 i0", func(b *testing.B) {
		b.ReportAllocs()
		input := []byte("Visit https://example.com/page for info")
		buf := make([]byte, 0, 4096)
		for b.Loop() {
			buf = ReplaceURL{}.ReplaceAllBytesAppend1(input, buf)
			buf = buf[:0]
		}
	})

	b.Run("precompiled1 i1", func(b *testing.B) {
		b.ReportAllocs()
		input := []byte("API at http://localhost:8080/api/v1/users")
		buf := make([]byte, 0, 4096)
		for b.Loop() {
			buf = ReplaceURL{}.ReplaceAllBytesAppend1(input, buf)
			buf = buf[:0]
		}
	})

	b.Run("precompiled1 i2", func(b *testing.B) {
		b.ReportAllocs()
		input := []byte("https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path ")
		buf := make([]byte, 0, 4096)
		for b.Loop() {
			buf = ReplaceURL{}.ReplaceAllBytesAppend1(input, buf)
			buf = buf[:0]
		}
	})

	b.Run("precompiled2 i0", func(b *testing.B) {
		b.ReportAllocs()
		input := []byte("Visit https://example.com/page for info")
		buf := make([]byte, 0, 4096)
		for b.Loop() {
			buf = ReplaceURL{}.ReplaceAllBytesAppend2(input, buf)
			buf = buf[:0]
		}
	})

	b.Run("precompiled2 i1", func(b *testing.B) {
		b.ReportAllocs()
		input := []byte("API at http://localhost:8080/api/v1/users")
		buf := make([]byte, 0, 4096)
		for b.Loop() {
			buf = ReplaceURL{}.ReplaceAllBytesAppend2(input, buf)
			buf = buf[:0]
		}
	})

	b.Run("precompiled2 i2", func(b *testing.B) {
		b.ReportAllocs()
		input := []byte("https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path https://test.com/path ")
		buf := make([]byte, 0, 4096)
		for b.Loop() {
			buf = ReplaceURL{}.ReplaceAllBytesAppend2(input, buf)
			buf = buf[:0]
		}
	})

}
