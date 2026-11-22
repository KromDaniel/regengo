package compiler

import (
	"os"
	"path/filepath"
	"regexp/syntax"
	"strings"
	"testing"
)

// generateSource compiles pattern (with the config adjusted by mutate) and
// returns the compiler together with the generated file text.
func generateSource(t *testing.T, pattern string, mutate func(*Config)) (*Compiler, string) {
	t.Helper()

	regexAST, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	regexAST = regexAST.Simplify()
	prog, err := syntax.Compile(regexAST)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	cfg := Config{
		Pattern:      pattern,
		Name:         "Test",
		Package:      "test",
		Program:      prog,
		RegexAST:     regexAST,
		WithCaptures: prog.NumCap > ImplicitCaptureCount,
	}
	if mutate != nil {
		mutate(&cfg)
	}
	c := New(cfg)
	out := filepath.Join(t.TempDir(), "r.go")
	c.SetOutputFile(out)
	if err := c.Generate(); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	src, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	return c, string(src)
}

func forceBacktracking(c *Config) { c.ForceBacktracking = true }
func forceTNFA(c *Config)         { c.ForceTNFA = true }

const perCapPattern = `a(b|c)?(d|e)?(f|g)?(h|i)?z`

func TestAttemptStartUnanchored(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		mutate  func(*Config)
		match   bool // Match bodies (else Find bodies)
		check   func(t *testing.T, c *Compiler)
		extra   func(isBytes bool) []string
	}{
		{
			name: "match_alt_prefix", pattern: `0x[0-9a-f]+`, mutate: forceBacktracking, match: true,
			check: func(t *testing.T, c *Compiler) {
				if !c.needsBacktracking {
					t.Fatal("needsBacktracking = false")
				}
			},
			extra: func(b bool) []string { return []string{pkgName(b) + ".IndexByte(input[attemptStart:]"} },
		},
		{
			name: "match_alt_noprefix", pattern: `[0-9]+x`, mutate: forceBacktracking, match: true,
			check: func(t *testing.T, c *Compiler) {
				if !c.needsBacktracking {
					t.Fatal("needsBacktracking = false")
				}
			},
			extra: func(b bool) []string { return []string{decodeCall(b, "attemptStart")} },
		},
		{
			name: "match_noalt_prefix", pattern: `abé`, match: true,
			check: func(t *testing.T, c *Compiler) {
				if c.needsBacktracking {
					t.Fatal("needsBacktracking = true")
				}
			},
			extra: func(b bool) []string { return []string{"StepSelect"} },
		},
		{
			name: "match_noalt_noprefix", pattern: `[a-c]bé`, mutate: forceBacktracking, match: true,
			check: func(t *testing.T, c *Compiler) {
				if c.needsBacktracking {
					t.Fatal("needsBacktracking = true")
				}
			},
			extra: func(b bool) []string { return []string{"StepSelect", decodeCall(b, "attemptStart")} },
		},
		{
			name: "find_array", pattern: `a(b|c)?(d|e)?(f|g)?z`,
			check: func(t *testing.T, c *Compiler) {
				if !c.needsBacktracking || c.usePerCaptureCheckpointing || c.useTDFAForCaptures {
					t.Fatal("expected array checkpointing without TDFA")
				}
			},
			extra: func(b bool) []string {
				return []string{"captures[0] = attemptStart", "captureStack = captureStack[:0]"}
			},
		},
		{
			name: "find_percap", pattern: perCapPattern,
			check: func(t *testing.T, c *Compiler) {
				if !c.usePerCaptureCheckpointing {
					t.Fatal("usePerCaptureCheckpointing = false")
				}
			},
			extra: func(b bool) []string { return []string{"captures[0] = attemptStart"} },
		},
		{
			name: "find_noalt", pattern: `(ab)é`,
			check: func(t *testing.T, c *Compiler) {
				if c.needsBacktracking {
					t.Fatal("needsBacktracking = true")
				}
			},
			extra: func(b bool) []string { return []string{"captures[0] = attemptStart"} },
		},
		{
			name: "find_memo", pattern: `(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})`, mutate: forceTNFA,
			check: func(t *testing.T, c *Compiler) {},
			extra: func(b bool) []string { return []string{"visited", "captures[0] = attemptStart"} },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, src := generateSource(t, tt.pattern, tt.mutate)
			if c.isAnchored {
				t.Fatal("pattern unexpectedly anchored")
			}
			tt.check(t, c)

			headers := []struct {
				header  string
				isBytes bool
			}{
				{"func (Test) FindStringReuse(", false},
				{"func (Test) FindBytesReuse(", true},
			}
			if tt.match {
				headers = []struct {
					header  string
					isBytes bool
				}{
					{"func (Test) MatchString(", false},
					{"func (Test) MatchBytes(", true},
				}
			}
			for _, h := range headers {
				body := funcBody(t, src, h.header)
				wants := append([]string{"attemptStart := offset", "offset = attemptStart", "if attemptStart < l {"}, tt.extra(h.isBytes)...)
				if tt.name == "find_memo" && strings.Contains(body, "findStringInternal") {
					t.Fatalf("%s: routed to TDFA, not memoized backtracking", h.header)
				}
				for _, w := range wants {
					if !strings.Contains(body, w) {
						t.Errorf("%s: body does not contain %q\n%s", h.header, w, body)
					}
				}
			}
		})
	}
}

func pkgName(isBytes bool) string {
	if isBytes {
		return "bytes"
	}
	return "strings"
}

func decodeCall(isBytes bool, v string) string {
	if isBytes {
		return "utf8.DecodeRune(input[" + v + ":])"
	}
	return "utf8.DecodeRuneInString(input[" + v + ":])"
}

func TestAttemptStartAnchored(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		mutate  func(*Config)
		headers []string
	}{
		{"match_alt", `^[0-9]+x`, forceBacktracking, []string{"func (Test) MatchString(", "func (Test) MatchBytes("}},
		{"match_noalt", `^abé`, nil, []string{"func (Test) MatchString(", "func (Test) MatchBytes("}},
		{"find_array", `^a(b|c)?(d|e)?(f|g)?z`, nil, []string{"func (Test) FindStringReuse(", "func (Test) FindBytesReuse("}},
		{"find_noalt", `^(ab)é`, nil, []string{"func (Test) FindStringReuse(", "func (Test) FindBytesReuse("}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, src := generateSource(t, tt.pattern, tt.mutate)
			if !c.isAnchored {
				t.Fatal("pattern not anchored")
			}
			for _, h := range tt.headers {
				if body := funcBody(t, src, h); strings.Contains(body, "attemptStart") {
					t.Errorf("%s: anchored body contains attemptStart", h)
				}
			}
		})
	}
}

func TestFindAllAdvancesByRune(t *testing.T) {
	for _, p := range []string{`(ab)é`, `(\d{4})-(\d{2})`, perCapPattern} {
		t.Run(p, func(t *testing.T) {
			_, src := generateSource(t, p, nil)
			s := funcBody(t, src, "func (Test) FindAllStringAppend(")
			if !strings.Contains(s, decodeCall(false, "searchStart")) {
				t.Errorf("FindAllStringAppend does not advance searchStart by rune\n%s", s)
			}
			b := funcBody(t, src, "func (Test) FindAllBytesAppend(")
			if !strings.Contains(b, decodeCall(true, "searchStart")) {
				t.Errorf("FindAllBytesAppend does not advance searchStart by rune\n%s", b)
			}
		})
	}
}

func TestRuneAnyDecodesRune(t *testing.T) {
	tests := []struct {
		pattern string
		notNL   bool
		match   bool
	}{
		{`x.y`, true, true},
		{`(?s)x.y`, false, true},
		{`(x.y)`, true, false},
		{`((?s)x.y)`, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			_, src := generateSource(t, tt.pattern, forceBacktracking)
			sh, bh := "func (Test) FindStringReuse(", "func (Test) FindBytesReuse("
			if tt.match {
				sh, bh = "func (Test) MatchString(", "func (Test) MatchBytes("
			}
			sb, bb := funcBody(t, src, sh), funcBody(t, src, bh)
			if !strings.Contains(sb, decodeCall(false, "offset")) {
				t.Errorf("String body lacks rune decoding\n%s", sb)
			}
			if !strings.Contains(bb, decodeCall(true, "offset")) {
				t.Errorf("Bytes body lacks rune decoding\n%s", bb)
			}
			if tt.notNL && !strings.Contains(sb, "input[offset] == uint8(0xa)") {
				t.Errorf("String body lacks newline check\n%s", sb)
			}
		})
	}
}
