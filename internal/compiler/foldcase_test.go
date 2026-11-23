package compiler

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp/syntax"
	"testing"
)

func compileFoldProg(t *testing.T, pattern string) (*syntax.Regexp, *syntax.Prog) {
	t.Helper()
	re, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		t.Fatalf("parse %q: %v", pattern, err)
	}
	re = re.Simplify()
	prog, err := syntax.Compile(re)
	if err != nil {
		t.Fatalf("compile %q: %v", pattern, err)
	}
	return re, prog
}

func TestNormalizeFoldCase(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    []rune // expected Rune of the first consuming inst; nil = untouched
		op      syntax.InstOp
	}{
		{"k", `(?i)k`, []rune{'K', 'K', 'k', 'k', 0x212A, 0x212A}, syntax.InstRune},
		{"s", `(?i)s`, []rune{'S', 'S', 's', 's', 0x17F, 0x17F}, syntax.InstRune},
		{"a", `(?i)a`, []rune{'A', 'A', 'a', 'a'}, syntax.InstRune},
		{"adjacent orbit merged", `(?i)\x{1C4}`, []rune{0x1C4, 0x1C6}, syntax.InstRune},
		{"digit untouched", `(?i)1`, nil, syntax.InstRune1},
		{"class untouched", `(?i)[a-c]`, nil, syntax.InstRune},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, prog := compileFoldProg(t, tt.pattern)
			orig := make([]syntax.Inst, len(prog.Inst))
			for i, in := range prog.Inst {
				in.Rune = append([]rune(nil), in.Rune...)
				orig[i] = in
			}

			got := NormalizeFoldCase(prog)

			if !reflect.DeepEqual(prog.Inst, orig) {
				t.Fatal("input program was mutated")
			}
			var inst *syntax.Inst
			for i := range got.Inst {
				switch got.Inst[i].Op {
				case syntax.InstRune, syntax.InstRune1:
					inst = &got.Inst[i]
				}
				if inst != nil {
					break
				}
			}
			if inst == nil || inst.Op != tt.op {
				t.Fatalf("op = %v, want %v", inst, tt.op)
			}
			if tt.want == nil {
				if got != prog {
					t.Error("no-op must return the same pointer")
				}
				return
			}
			if got == prog {
				t.Fatal("expected a copy")
			}
			if !reflect.DeepEqual(inst.Rune, tt.want) {
				t.Errorf("Rune = %U, want %U", inst.Rune, tt.want)
			}
			if syntax.Flags(inst.Arg)&syntax.FoldCase != 0 {
				t.Error("FoldCase not cleared")
			}
			for i := 0; i+1 < len(inst.Rune); i += 2 {
				if inst.Rune[i] > inst.Rune[i+1] || (i >= 2 && inst.Rune[i] <= inst.Rune[i-1]) {
					t.Errorf("pairs not sorted/disjoint: %U", inst.Rune)
				}
			}
			// Idempotent: a second pass returns the same pointer.
			if again := NormalizeFoldCase(got); again != got {
				t.Error("not idempotent")
			}
		})
	}
}

func TestNormalizeFoldCaseNilAndArgBits(t *testing.T) {
	if NormalizeFoldCase(nil) != nil {
		t.Error("nil prog must yield nil")
	}
	// Only the FoldCase bit may be cleared; other Arg bits survive.
	prog := &syntax.Prog{Inst: []syntax.Inst{
		{Op: syntax.InstRune, Rune: []rune{'K'}, Arg: uint32(syntax.FoldCase) | 0x100},
		{Op: syntax.InstMatch},
	}}
	got := NormalizeFoldCase(prog)
	if got.Inst[0].Arg != 0x100 {
		t.Errorf("Arg = %#x, want 0x100", got.Inst[0].Arg)
	}
	if prog.Inst[0].Arg != uint32(syntax.FoldCase)|0x100 {
		t.Error("input mutated")
	}
}

func TestFoldCaseGenerateNoPanic(t *testing.T) {
	patterns := []string{`(?i)a`, `a(?i)kz`, `(?i)(k)z`, `(?i)(s+)x`, `(?i)(a+)+z`, `(?i)café`}
	engines := map[string]func(*Config){
		"default":      func(*Config) {},
		"backtracking": func(c *Config) { c.ForceBacktracking = true },
		"thompson":     func(c *Config) { c.ForceThompson = true },
		"tnfa":         func(c *Config) { c.ForceTNFA = true },
		"tdfa":         func(c *Config) { c.ForceTDFA = true },
		"dfa":          func(c *Config) { c.ForceDFA = true },
	}
	for _, p := range patterns {
		for name, apply := range engines {
			t.Run(p+"/"+name, func(t *testing.T) {
				re, prog := compileFoldProg(t, p)
				out := filepath.Join(t.TempDir(), "gen.go")
				cfg := Config{
					Pattern: p, Name: "Fold", OutputFile: out, Package: "gen",
					Program: prog, RegexAST: re, WithCaptures: true,
				}
				apply(&cfg)
				if err := New(cfg).Generate(); err != nil {
					t.Fatalf("Generate: %v", err)
				}
				src, err := os.ReadFile(out)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := parser.ParseFile(token.NewFileSet(), out, src, 0); err != nil {
					t.Fatalf("output does not parse: %v", err)
				}
			})
		}
	}
}

func hasLabel(labels []string, l string) bool {
	for _, x := range labels {
		if x == l {
			return true
		}
	}
	return false
}

func TestAnalyzePatternFoldCase(t *testing.T) {
	res, err := AnalyzePattern(`(?i)(k)z`, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !hasLabel(res.FeatureLabels, "UnicodeCharClass") && !hasLabel(res.EngineLabels, "UnicodeCharClass") {
		t.Errorf("missing UnicodeCharClass: %v %v", res.FeatureLabels, res.EngineLabels)
	}

	res, err = AnalyzePattern(`(?i)(a)z`, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !hasLabel(res.EngineLabels, "DFA") {
		t.Errorf("want DFA label, got %v", res.EngineLabels)
	}
}

func TestCanUseDFAFoldCase(t *testing.T) {
	check := func(pattern string, want bool) {
		_, prog := compileFoldProg(t, pattern)
		gen := NewDFAGenerator(New(Config{Package: "x", Program: prog, Name: "X"}))
		got, info := gen.CanUseDFA()
		if got != want {
			t.Errorf("%s: CanUseDFA = %v (%s), want %v", pattern, got, info, want)
		}
	}
	check(`(?i)abc`, true)
	check(`(?i)k`, false)
}
