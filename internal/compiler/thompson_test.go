package compiler

import (
	"regexp"
	"regexp/syntax"
	"strings"
	"testing"
)

func forceThompson(c *Config) { c.ForceThompson = true }

const (
	thompsonStringHeader = "func (Test) MatchString("
	thompsonBytesHeader  = "func (Test) MatchBytes("
)

func compileProg(t *testing.T, pattern string) *syntax.Prog {
	t.Helper()
	re, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		t.Fatalf("parse %q: %v", pattern, err)
	}
	prog, err := syntax.Compile(re.Simplify())
	if err != nil {
		t.Fatalf("compile %q: %v", pattern, err)
	}
	return prog
}

func TestIsASCIIProgram(t *testing.T) {
	tests := []struct {
		pattern string
		want    bool
	}{
		{`abc`, true},
		{`[a-z]+`, true},
		{`(?i)a`, true},
		{`(?i)ab`, true},
		{`(?i)[a-c]`, true},
		{`^abc$`, true},
		{`\bfoo\b`, true},
		{`(?m)^$`, true},
		{`x*\B`, true},
		{`\B`, true},
		{`.`, false},
		{`(?s).`, false},
		{`é`, false},
		{`[^a]`, false},
		{`\pL`, false},
		{`(?i)k`, false}, // K, k, U+212A KELVIN SIGN
		{`(?i)s`, false}, // s, S, U+017F LONG S
		{`(?i)é`, false},
		{`[a-c]é?`, false},
		{`x.y`, false},
	}
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			if got := isASCIIProgram(compileProg(t, tt.pattern)); got != tt.want {
				t.Errorf("isASCIIProgram(%q) = %v, want %v", tt.pattern, got, tt.want)
			}
		})
	}
}

func thompsonBodies(t *testing.T, pattern string) (string, string, string) {
	t.Helper()
	_, src := generateSource(t, pattern, forceThompson)
	return funcBody(t, src, thompsonStringHeader), funcBody(t, src, thompsonBytesHeader), src
}

func TestThompsonAssertionExpansion(t *testing.T) {
	has := func(pattern, sub string) (bool, bool) {
		s, b, _ := thompsonBodies(t, pattern)
		return strings.Contains(s, sub), strings.Contains(b, sub)
	}
	check := func(pattern, sub string, want bool) {
		t.Helper()
		s, b := has(pattern, sub)
		if s != want || b != want {
			t.Errorf("%q: contains %q: String=%v Bytes=%v, want %v", pattern, sub, s, b, want)
		}
	}

	for _, p := range []string{`^abc`, `abc$`, `a\b`, `(?m)^\w+$`, `\Aa\z`, `\B`, `^$`, `x*\B`} {
		check(p, "ewMask", true)
	}
	for _, p := range []string{`abc`, `[a-z]+`, `x.y`, `(a+)+b`} {
		check(p, "ewMask", false)
	}

	check(`a\b`, "isWordChar(", true)
	check(`^abc`, "isWordChar(", false)
	check(`(?m)^a$`, "isWordChar(", false)

	check(`^$`, "prev := current", true)
	check(`^abc`, "prev := current", false)
	check(`a\b`, "prev := current", false)

	// Anchored: immediate accept check comes before the offset loop.
	for _, body := range func() []string { s, b, _ := thompsonBodies(t, `^abc`); return []string{s, b} }() {
		i, j := strings.Index(body, "if current&acceptMask != 0"), strings.Index(body, "for offset")
		if i < 0 || j < 0 || i > j {
			t.Errorf("^abc: immediate accept check must precede the offset loop (check=%d loop=%d)", i, j)
		}
	}

	// Continuation-byte start guard (valid UTF-8 only): ASCII program with assertions.
	check(`x*\B`, "0xc0", true)
	check(`abc`, "0xc0", false)
}

func TestThompsonRunePath(t *testing.T) {
	for _, p := range []string{`x.y`, `[^,]+;`, `é`, `(?i)k`, `\pL+z`} {
		s, b, _ := thompsonBodies(t, p)
		if !strings.Contains(s, decodeCall(false, "offset")) || !strings.Contains(s, decodeCall(false, "searchStart")) {
			t.Errorf("%q: String body lacks rune decode calls", p)
		}
		if !strings.Contains(b, decodeCall(true, "offset")) || !strings.Contains(b, decodeCall(true, "searchStart")) {
			t.Errorf("%q: Bytes body lacks rune decode calls", p)
		}
		if strings.Contains(s, "offset++") || strings.Contains(b, "offset++") {
			t.Errorf("%q: rune path must not step offset++", p)
		}
	}
	for _, p := range []string{`abc`, `[a-z]+`, `^abc`, `a\b`, `(?i)ab`} {
		s, b, _ := thompsonBodies(t, p)
		for _, body := range []string{s, b} {
			if !strings.Contains(body, "offset++") || strings.Contains(body, "utf8.") {
				t.Errorf("%q: byte path must use offset++ and no utf8", p)
			}
		}
	}

	s, _, _ := thompsonBodies(t, `(?i)k`)
	if !strings.Contains(s, "8490") {
		t.Errorf("(?i)k: expected U+212A (8490) in the fold orbit, got:\n%s", s)
	}

	_, _, src := thompsonBodies(t, `\pL+z`)
	if !strings.Contains(src, "unicode.Is(thompsonClassTest") {
		t.Errorf(`\pL+z: expected unicode.Is(thompsonClassTest...) in generated file`)
	}
	decls := regexp.MustCompile(`var thompsonClassTest\d+ =`).FindAllString(src, -1)
	distinct := map[string]bool{}
	for _, d := range decls {
		distinct[d] = true
	}
	if len(decls) == 0 || len(decls) != len(distinct) {
		t.Errorf(`\pL+z: range table declarations: %d total, %d distinct (want equal, > 0)`, len(decls), len(distinct))
	}
}

func TestThompsonNoConsumingStates(t *testing.T) {
	for _, p := range []string{`\B`, `^$`, `(?m)^$`} {
		s, b, _ := thompsonBodies(t, p)
		for _, body := range []string{s, b} {
			if strings.Contains(body, "var next") || strings.Contains(body, "c := input") {
				t.Errorf("%q: body must not declare next or c:\n%s", p, body)
			}
		}
	}
}

func TestThompsonRuneAnyOnly(t *testing.T) {
	s, _, _ := thompsonBodies(t, `(?s).+`)
	if strings.Contains(s, "r := rune(c)") {
		t.Errorf("(?s).+: r must not be declared when unused:\n%s", s)
	}
}
