package compiler

import (
	"strings"
	"testing"
)

func forceTDFA(c *Config) { c.ForceTDFA = true }

func TestFindAllBacktrackingIteration(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		anchored bool
	}{
		{"star", `(a*)`, false},
		{"noalt", `(ab)é`, false},
		{"array", `(\d{4})-(\d{2})`, false},
		{"percap", perCapPattern, false},
		{"anchored", `^(a*)`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, src := generateSource(t, tt.pattern, nil)
			if c.useTDFAForCaptures {
				t.Fatal("precondition: pattern must use backtracking for Find")
			}
			if c.isAnchored != tt.anchored {
				t.Fatalf("isAnchored = %v, want %v", c.isAnchored, tt.anchored)
			}
			for _, isBytes := range []bool{false, true} {
				header := "func (Test) FindAllStringAppend("
				if isBytes {
					header = "func (Test) FindAllBytesAppend("
				}
				body := funcBody(t, src, header)
				for _, want := range []string{
					"base := len(s)",
					"pos, prevEnd := 0, -1",
					"searchStart := 0",
					"len(result)-base >= n",
					"if searchStart > l {",
					"if searchStart >= l {",
					"prevEnd = offset",
					"searchStart = pos",
					"if accept {",
					decodeCall(isBytes, "pos"),
					decodeCall(isBytes, "searchStart"),
				} {
					if !strings.Contains(body, want) {
						t.Errorf("%s lacks %q\n%s", header, want, body)
					}
				}
				if strings.Contains(body, "searchStart = captures[1]") {
					t.Errorf("%s still sets searchStart from captures[1]", header)
				}
				a, b, d := strings.Index(body, "accept := true"), strings.Index(body, "if accept {"), strings.Index(body, "var item")
				if !(a >= 0 && a < b && b < d) {
					t.Errorf("%s: want accept := true < if accept { < var item, got %d %d %d", header, a, b, d)
				}
				if tt.anchored && !strings.Contains(body, "if searchStart > 0 {") {
					t.Errorf("%s: anchored body lacks the searchStart > 0 check", header)
				}
			}
		})
	}
}

func TestFindAllMemoClear(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
		memo   bool
	}{
		{"memoized", forceTNFA, true},
		{"plain", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, src := generateSource(t, `(a*|b)`, tt.mutate)
			if c.useMemoization != tt.memo {
				t.Fatalf("useMemoization = %v, want %v", c.useMemoization, tt.memo)
			}
			for _, header := range []string{"func (Test) FindAllStringAppend(", "func (Test) FindAllBytesAppend("} {
				body := funcBody(t, src, header)
				if got := strings.Contains(body, "&^="); got != tt.memo {
					t.Errorf("%s: contains memo clear = %v, want %v\n%s", header, got, tt.memo, body)
				}
			}
		})
	}
}

func TestTDFAFindFrom(t *testing.T) {
	for _, p := range []string{`(a*)`, `x(a)?y`} {
		t.Run(p, func(t *testing.T) {
			c, src := generateSource(t, p, forceTDFA)
			if !c.useTDFAForCaptures {
				t.Fatal("precondition: TDFA must be used for captures")
			}
			for _, want := range []string{
				"func (Test) findStringInternal(input string, from int, r *TestResult) (int, int, bool)",
				"func (Test) findBytesInternal(input []byte, from int, r *TestBytesResult) (int, int, bool)",
			} {
				if !strings.Contains(src, want) {
					t.Errorf("source lacks %q", want)
				}
			}
			strInt := funcBody(t, src, "func (Test) findStringInternal(")
			bytInt := funcBody(t, src, "func (Test) findBytesInternal(")
			for _, body := range []string{strInt, bytInt} {
				for _, want := range []string{
					"for start := from; start <= l; start++",
					"return start, matchEnd, true",
					"return -1, -1, false",
				} {
					if !strings.Contains(body, want) {
						t.Errorf("internal find lacks %q\n%s", want, body)
					}
				}
			}
			if b := funcBody(t, src, "func (Test) FindStringReuse("); !strings.Contains(b, "findStringInternal(input, 0, r)") {
				t.Errorf("FindStringReuse does not call findStringInternal(input, 0, r)\n%s", b)
			}
			if b := funcBody(t, src, "func (Test) FindBytesReuse("); !strings.Contains(b, "findBytesInternal(input, 0, r)") {
				t.Errorf("FindBytesReuse does not call findBytesInternal(input, 0, r)\n%s", b)
			}
			for _, isBytes := range []bool{false, true} {
				header, call, bad := "func (Test) FindAllStringAppend(", "findStringInternal(input, pos, dst)", "FindString(input["
				if isBytes {
					header, call, bad = "func (Test) FindAllBytesAppend(", "findBytesInternal(input, pos, dst)", "FindBytes(input["
				}
				body := funcBody(t, src, header)
				for _, want := range []string{call, "pos, prevEnd := 0, -1", decodeCall(isBytes, "pos"), "len(results)-base < n"} {
					if !strings.Contains(body, want) {
						t.Errorf("%s lacks %q\n%s", header, want, body)
					}
				}
				if strings.Contains(body, bad) {
					t.Errorf("%s still re-slices the input via %q", header, bad)
				}
			}
			if p == `x(a)?y` {
				if !strings.Contains(strInt, `r.Group1 = ""`) {
					t.Errorf("findStringInternal lacks the group reset\n%s", strInt)
				}
				if !strings.Contains(bytInt, "r.Group1 = nil") {
					t.Errorf("findBytesInternal lacks the group reset\n%s", bytInt)
				}
			}
		})
	}
}
