package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"regexp/syntax"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/KromDaniel/regengo"
)

// Stdlib parity test (issue #37).
//
// Every (pattern, engine) variant is compiled into its own package inside ONE
// temporary module, a generated driver imports all of them and compares
// Match/Find/FindAll (String and Bytes forms, positions included) against the
// standard library regexp on deterministic pseudo-random inputs plus fixed
// regression inputs. The driver is built once and run once.

const (
	defaultParitySeed = uint64(37)
	defaultParityN    = 2000
)

type parityEngine string

const (
	engineDefault      parityEngine = "default"
	engineTNFA         parityEngine = "ForceTNFA"
	engineTDFA         parityEngine = "ForceTDFA"
	engineBacktracking parityEngine = "ForceBacktracking"
	engineThompson     parityEngine = "ForceThompson"
)

const (
	markerDFA             = "dfa"
	markerThompson        = "thompson"
	markerBacktracking    = "backtracking"
	markerTDFA            = "tdfa"
	markerTNFA            = "tnfa"
	parityMatchFuncHeader = "func (R) MatchString("
	parityFindFuncHeader  = "func (R) FindStringReuse("
)

type parityVariant struct {
	engine      parityEngine
	matchEngine string // expected Match marker: "dfa", "thompson", "backtracking"
	findEngine  string // expected Find marker: "tdfa", "tnfa", "backtracking"
}

type parityCase struct {
	name     string
	pattern  string
	variants []parityVariant
	extra    []string // fixed regression inputs, always run first, in this order
}

func pv(engine parityEngine, matchEngine, findEngine string) parityVariant {
	return parityVariant{engine: engine, matchEngine: matchEngine, findEngine: findEngine}
}

// thompsonAndBacktracking is the variant pair used by assertion-heavy patterns.
func thompsonAndBacktracking() []parityVariant {
	return []parityVariant{
		pv(engineThompson, markerThompson, markerBacktracking),
		pv(engineBacktracking, markerBacktracking, markerBacktracking),
	}
}

// thompsonOnly is the variant list for patterns that only the Thompson engine
// can be generated for (the backtracking generator panics on them).
func thompsonOnly() []parityVariant {
	return []parityVariant{pv(engineThompson, markerThompson, markerBacktracking)}
}

// allEngines returns one variant per engine. defMatch/defFind are the markers
// the default engine selection produces for the pattern; the forced variants
// keep the default Match engine except Thompson and Backtracking.
func allEngines(defMatch, defFind string) []parityVariant {
	return []parityVariant{
		pv(engineDefault, defMatch, defFind),
		pv(engineThompson, markerThompson, markerBacktracking),
		pv(engineBacktracking, markerBacktracking, markerBacktracking),
		pv(engineTNFA, defMatch, markerTNFA),
		pv(engineTDFA, defMatch, markerTDFA),
	}
}

var parityCases = []parityCase{
	{
		name:    "date",
		pattern: `(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})`,
		variants: []parityVariant{
			pv(engineDefault, markerDFA, markerBacktracking),
			pv(engineTNFA, markerDFA, markerTNFA),
			pv(engineTDFA, markerDFA, markerTDFA),
			pv(engineBacktracking, markerBacktracking, markerBacktracking),
		},
		extra: []string{"12025-10-05", "2025-10-05", "x2025-10-0"},
	},
	{
		name:    "hex",
		pattern: `0x[0-9a-fA-F]+`,
		variants: []parityVariant{
			pv(engineDefault, markerDFA, markerBacktracking),
			pv(engineBacktracking, markerBacktracking, markerBacktracking),
			pv(engineThompson, markerThompson, markerBacktracking),
		},
		extra: []string{"00x1f", "0x1f", "000x"},
	},
	{
		name:    "hexE",
		pattern: `0x[0-9a-f]+é?`,
		variants: []parityVariant{
			pv(engineDefault, markerBacktracking, markerBacktracking),
			pv(engineThompson, markerThompson, markerBacktracking),
		},
		extra: []string{"00x1f", "0x1fé", "00x1é"},
	},
	{
		name:     "abE",
		pattern:  `abé`,
		variants: []parityVariant{pv(engineDefault, markerBacktracking, markerBacktracking)},
		extra:    []string{"aabé", "abé"},
	},
	{
		name:     "abCapE",
		pattern:  `(ab)é`,
		variants: []parityVariant{pv(engineDefault, markerBacktracking, markerBacktracking)},
		extra:    []string{"aabé", "abé"},
	},
	{
		name:    "semver",
		pattern: `(?P<major>\d+)\.(?P<minor>\d+)\.(?P<patch>\d+)`,
		variants: []parityVariant{
			pv(engineDefault, markerDFA, markerBacktracking),
			pv(engineTDFA, markerDFA, markerTDFA),
			pv(engineTNFA, markerDFA, markerTNFA),
		},
		extra: []string{"abcdef 1.2.3", "1.2.3 4.5.6"},
	},
	{
		name:    "anchA",
		pattern: `^(a)`,
		variants: []parityVariant{
			pv(engineDefault, markerDFA, markerBacktracking),
			pv(engineBacktracking, markerBacktracking, markerBacktracking),
			pv(engineTDFA, markerDFA, markerTDFA),
		},
		extra: []string{"aaa", "a"},
	},
	{
		name:    "anchAplus",
		pattern: `^(a+)`,
		variants: []parityVariant{
			pv(engineDefault, markerDFA, markerBacktracking),
			pv(engineTDFA, markerDFA, markerTDFA),
		},
		extra: []string{"aaa", "aaba"},
	},
	{
		name:    "aStar",
		pattern: `(a*)`,
		variants: []parityVariant{
			pv(engineDefault, markerDFA, markerBacktracking),
			pv(engineTDFA, markerDFA, markerTDFA),
		},
		extra: []string{"aba", "éé", "", "baaac", "bab", "bb", "aab"},
	},
	{
		name:    "digitsX",
		pattern: `(\d*)x?`,
		variants: []parityVariant{
			pv(engineDefault, markerDFA, markerBacktracking),
			pv(engineTDFA, markerDFA, markerTDFA),
		},
		extra: []string{"1x2", "x", ""},
	},
	{
		name:    "wordEnd",
		pattern: `(?:\w+\s?)+\bend\b`,
		variants: []parityVariant{
			pv(engineDefault, markerThompson, markerTNFA),
			pv(engineBacktracking, markerBacktracking, markerTNFA),
		},
		extra: []string{"x end", "end", "xend"},
	},
	{
		name:     "altAnch",
		pattern:  `^ab|cd$`,
		variants: thompsonAndBacktracking(),
		extra:    []string{"ab", "xcd", "abc", "cdx"},
	},
	{
		name:     "mlWord",
		pattern:  `(?m)^\w+$`,
		variants: thompsonAndBacktracking(),
		extra:    []string{"ab\ncd", "a b", "\nab"},
	},
	{
		name:     "notB",
		pattern:  `\Bb\B`,
		variants: thompsonAndBacktracking(),
		extra:    []string{"abc", "b", "ab"},
	},
	{
		name:     "aWB",
		pattern:  `a\b`,
		variants: thompsonAndBacktracking(),
		extra:    []string{"a b", "a", "ab"},
	},
	{
		name:     "caretAbc",
		pattern:  `^abc`,
		variants: []parityVariant{pv(engineThompson, markerThompson, markerBacktracking)},
		extra:    []string{"abc"},
	},
	{
		name:     "abcDollar",
		pattern:  `abc$`,
		variants: []parityVariant{pv(engineThompson, markerThompson, markerBacktracking)},
		extra:    []string{"abc"},
	},
	{
		name:    "notComma",
		pattern: `(?:[^,]+,?)+;`,
		variants: []parityVariant{
			pv(engineDefault, markerThompson, markerTDFA),
			pv(engineBacktracking, markerBacktracking, markerTDFA),
		},
		extra: []string{"é;", "a,b;", "€;", "😀;", "a,😀;", "😀,€;"},
	},
	{
		name:     "dotXY",
		pattern:  `x.y`,
		variants: thompsonAndBacktracking(),
		extra:    []string{"xéy", "xay", "x€y", "x😀y", "x\ny"},
	},
	{
		name:     "dotSXY",
		pattern:  `(?s)x.y`,
		variants: []parityVariant{pv(engineBacktracking, markerBacktracking, markerBacktracking)},
		extra:    []string{"xéy", "x\ny", "x€y", "x😀y"},
	},
	{
		name:     "perCap",
		pattern:  `a(b|c)?(d|e)?(f|g)?(h|i)?z`,
		variants: []parityVariant{pv(engineDefault, markerDFA, markerBacktracking)},
		extra:    []string{"aabz", "aaz", "az", "abaz"},
	},
	{
		name:     "negE",
		pattern:  `[^é]z`,
		variants: []parityVariant{pv(engineDefault, markerBacktracking, markerBacktracking)},
		extra:    []string{"éz", "aéz", "az"},
	},
	{
		name:    "eAcute",
		pattern: `(?:é+x?)+`,
		variants: []parityVariant{
			pv(engineDefault, markerThompson, markerTDFA),
			pv(engineBacktracking, markerBacktracking, markerTDFA),
		},
		extra: []string{"é", "éxé"},
	},
	{
		name:    "altPrio",
		pattern: `(a|ab)(c|bcd)?`,
		variants: []parityVariant{
			pv(engineDefault, markerDFA, markerBacktracking),
			pv(engineBacktracking, markerBacktracking, markerBacktracking),
		},
		extra: []string{"ab", "abcd", "abc"},
	},
	{
		name:    "email",
		pattern: `[\w.+-]+@[\w.-]+\.\w+`,
		variants: []parityVariant{
			pv(engineDefault, markerDFA, markerBacktracking),
			pv(engineTDFA, markerDFA, markerTDFA),
		},
		extra: []string{"a@b.c", "x a@b.c y", ".@_@__._."},
	},
	{
		name:     "eStar",
		pattern:  `(é*)`,
		variants: []parityVariant{pv(engineDefault, markerBacktracking, markerBacktracking)},
		extra:    []string{"aéb", "éaé", "éé", ""},
	},
	{
		name:     "wbEmpty",
		pattern:  `(\b)`,
		variants: []parityVariant{pv(engineDefault, markerBacktracking, markerBacktracking)},
		extra:    []string{"ab cd", "a", "", " "},
	},
	{
		name:    "aStarEnd",
		pattern: `(a*)$`,
		variants: []parityVariant{
			pv(engineDefault, markerDFA, markerBacktracking),
			pv(engineTDFA, markerDFA, markerTDFA),
		},
		extra: []string{"baa", "b", "", "aa"},
	},
	{
		name:    "anchAStar",
		pattern: `^(a*)`,
		variants: []parityVariant{
			pv(engineDefault, markerDFA, markerBacktracking),
			pv(engineTDFA, markerDFA, markerTDFA),
		},
		extra: []string{"baa", "aab", ""},
	},
	{
		name:    "hexCap",
		pattern: `0x([0-9a-f]+)`,
		variants: []parityVariant{
			pv(engineDefault, markerDFA, markerBacktracking),
			pv(engineTDFA, markerDFA, markerTDFA),
		},
		extra: []string{"0x1 0x2", "00x1f0x2", "0x"},
	},
	{
		name:    "optCap",
		pattern: `x(a)?y`,
		variants: []parityVariant{
			pv(engineDefault, markerDFA, markerBacktracking),
			pv(engineTDFA, markerDFA, markerTDFA),
		},
		extra: []string{"xay", "xy", "xayxy", "yxxy é a yxa"},
	},
	{
		name:    "altEmpty",
		pattern: `(a*|b)`,
		variants: []parityVariant{
			pv(engineDefault, markerDFA, markerBacktracking),
			pv(engineTNFA, markerDFA, markerTNFA),
		},
		extra: []string{"ab", "aab", "b"},
	},
	{
		name:     "mlA",
		pattern:  `(?m)^a+$`,
		variants: thompsonAndBacktracking(),
		extra:    []string{"a\nb", "b\na", "a\n", "\na", "ab\na"},
	},
	{
		name:     "textAZ",
		pattern:  `\Aa+\z`,
		variants: thompsonAndBacktracking(),
		extra:    []string{"aa", "aab", "ba", "", "a\n"},
	},
	{
		name:     "anchStar",
		pattern:  `^a*`,
		variants: thompsonAndBacktracking(),
		extra:    []string{"bbb", "", "ab"},
	},
	{
		name:     "midRuneB",
		pattern:  `x*\B`,
		variants: thompsonAndBacktracking(),
		extra:    []string{"aéb", "é", "ab", "a€b", "a😀b"},
	},
	{
		name:     "negAb",
		pattern:  `[^a]b`,
		variants: thompsonAndBacktracking(),
		extra:    []string{"😀b", "€b", "éb", "ab", "a€b"},
	},
	{
		name:     "greekZ",
		pattern:  `\p{Greek}+z`,
		variants: thompsonAndBacktracking(),
		extra:    []string{"αβz", "αz", "az", "Ωz", "😀z"},
	},
	{
		name:     "multiByte",
		pattern:  `a€😀b`,
		variants: thompsonAndBacktracking(),
		extra:    []string{"a€😀b", "a€😀", "xa€😀b", "a€€b"},
	},
	{
		name:     "negEth",
		pattern:  `[^é]z`,
		variants: thompsonOnly(),
		extra:    []string{"éz", "aéz", "az", "€z"},
	},
	{
		name:    "mlWords",
		pattern: `(?m)^(?:\w+\s?)+$`,
		variants: []parityVariant{
			pv(engineDefault, markerThompson, markerTNFA),
			pv(engineBacktracking, markerBacktracking, markerTNFA),
		},
		extra: []string{"ab cd\nx", "a\n\nb", " a", "ab \ncd", "é\nab"},
	},
	{
		name:     "mlEmpty",
		pattern:  `(?m)^$`,
		variants: thompsonAndBacktracking(),
		extra:    []string{"a\n\nb", "\n", "", "a", "a\n"},
	},
	{
		name:     "onlyB",
		pattern:  `\B`,
		variants: thompsonAndBacktracking(),
		extra:    []string{"aéb", "é", "ab", "a€b"},
	},
	{
		name:     "foldA",
		pattern:  `(?i)(a)(b)`,
		variants: allEngines(markerDFA, markerBacktracking),
		extra:    []string{"AB", "aB", "ab", "a b"},
	},
	{
		name:     "foldK",
		pattern:  `(?i)(k)`,
		variants: allEngines(markerBacktracking, markerBacktracking),
		extra:    []string{"K", "k", "\u212a", "x", "é"},
	},
	{
		name:     "foldS",
		pattern:  `(?i)a(s)`,
		variants: allEngines(markerBacktracking, markerBacktracking),
		extra:    []string{"aſ", "AS", "Aſ", "as"},
	},
	{
		name:     "foldE",
		pattern:  `(?i)(é+)x`,
		variants: allEngines(markerBacktracking, markerBacktracking),
		extra:    []string{"Éx", "éÉx", "ex", "x"},
	},
	{
		name:     "foldKZ",
		pattern:  `a(?i)(kz)+`,
		variants: allEngines(markerBacktracking, markerBacktracking),
		extra:    []string{"akz", "aKZ", "aKZkz", "ak"},
	},
	{
		name:     "foldMix",
		pattern:  `(?i)(?P<w>[a-c]+)k`,
		variants: allEngines(markerBacktracking, markerBacktracking),
		extra:    []string{"ABk", "abK", "cK", "x"},
	},
	{
		name:     "foldCafe",
		pattern:  `(?i)caf(é)`,
		variants: allEngines(markerBacktracking, markerBacktracking),
		extra:    []string{"CAFÉ", "Café", "café"},
	},
	{
		name:    "foldRisk",
		pattern: `(?i)(k+)+z`,
		// Catastrophic-risk patterns route Find to TDFA whatever the Force*
		// option (Match still follows the forced engine), so every variant
		// expects the tdfa Find marker.
		variants: []parityVariant{
			pv(engineDefault, markerThompson, markerTDFA),
			pv(engineThompson, markerThompson, markerTDFA),
			pv(engineBacktracking, markerBacktracking, markerTDFA),
			pv(engineTNFA, markerThompson, markerTDFA),
			pv(engineTDFA, markerThompson, markerTDFA),
		},
		extra: []string{"Kz", "kKz", "kk"},
	},
}

// Method names, in the order they are reported. Used in driver output and in
// knownDivergence keys.
var parityMethods = []string{
	"MatchString",
	"MatchBytes",
	"FindString",
	"FindBytes",
	"FindAllString(n=-1)",
	"FindAllString(n=2)",
	"FindAllBytes(n=-1)",
	"FindAllBytes(n=2)",
	"FindAllStringAppend(n=-1)",
	"FindAllStringAppend(n=2)",
	"FindAllBytesAppend(n=-1)",
	"FindAllBytesAppend(n=2)",
}

type divKey struct{ variant, method string }

// Tags for knownDivergence entries.
const (
	// tagIssue41: the TDFA Find engine matches bytes, not runes, so it misses
	// non-ASCII input (fold orbit members such as U+212A, U+017F, plus literals
	// like é and classes like [^,]). Delete the entries when #41 lands.
	tagIssue41 = "issue41"
)

// knownDivergence parks divergences from the standard library that are not part
// of issue #37 (see the issue41 entries in buildKnownDivergence).
// TestStdlibParity fails on a divergence that is not listed here, and also on a
// listed entry that no longer diverges (so entries cannot go stale).
var knownDivergence = buildKnownDivergence()

func known(m map[divKey]string, tag, variant string, methods ...string) {
	for _, method := range methods {
		m[divKey{variant: variant, method: method}] = tag
	}
}

func buildKnownDivergence() map[divKey]string {
	m := map[divKey]string{}

	find := []string{"FindString", "FindBytes"}
	findAll := []string{
		"FindAllString(n=-1)", "FindAllString(n=2)",
		"FindAllBytes(n=-1)", "FindAllBytes(n=2)",
		"FindAllStringAppend(n=-1)", "FindAllStringAppend(n=2)",
		"FindAllBytesAppend(n=-1)", "FindAllBytesAppend(n=2)",
	}
	findAndAll := append(append([]string{}, find...), findAll...)

	// Issue #41: the TDFA Find engine is byte-based: non-ASCII literals never
	// match and classes like [^,] only see single bytes (eAcute, notComma Find*).
	for _, v := range []string{
		"eAcute/ForceBacktracking", "eAcute/default",
		"notComma/ForceBacktracking", "notComma/default",
	} {
		known(m, tagIssue41, v, findAndAll...)
	}

	// Issue #41 for case-folded literals whose fold orbit has a non-ASCII member
	// (and so only match on TDFA via their ASCII members). foldA is all-ASCII
	// and fully enforced on TDFA.
	for _, v := range []string{
		"foldK/ForceTDFA", "foldS/ForceTDFA", "foldE/ForceTDFA", "foldKZ/ForceTDFA",
		"foldMix/ForceTDFA", "foldCafe/ForceTDFA",
		"foldRisk/default", "foldRisk/ForceThompson", "foldRisk/ForceBacktracking",
		"foldRisk/ForceTNFA", "foldRisk/ForceTDFA",
	} {
		known(m, tagIssue41, v, findAndAll...)
	}

	return m
}

func parityVariantName(c parityCase, v parityVariant) string {
	return c.name + "/" + string(v.engine)
}

// parityAlphabet derives a small, deterministic input alphabet from the pattern:
// ASCII literal runes, a few runes per (narrowest-first) class range, a space,
// a newline when the pattern is line- or dot-sensitive, and 'é'.
func parityAlphabet(pattern string) ([]rune, error) {
	re, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return nil, err
	}
	prog, err := syntax.Compile(re.Simplify())
	if err != nil {
		return nil, err
	}

	set := map[rune]bool{}
	type runeRange struct{ lo, hi rune }
	var ranges []runeRange
	needNewline := false

	// addFolded adds the full unicode.SimpleFold orbit of r (so U+212A and
	// U+017F show up for k and s).
	addFolded := func(r rune) {
		for f := unicode.SimpleFold(r); f != r; f = unicode.SimpleFold(f) {
			set[f] = true
		}
	}

	for i := range prog.Inst {
		inst := &prog.Inst[i]
		fold := syntax.Flags(inst.Arg)&syntax.FoldCase != 0
		switch inst.Op {
		case syntax.InstRune1:
			if r := inst.Rune[0]; r < utf8.RuneSelf {
				set[r] = true
				if fold {
					addFolded(r)
				}
			}
		case syntax.InstRune:
			if len(inst.Rune) == 1 {
				r := inst.Rune[0]
				ranges = append(ranges, runeRange{r, r})
				if fold {
					set[r] = true
					addFolded(r)
				}
				continue
			}
			for j := 0; j+1 < len(inst.Rune); j += 2 {
				ranges = append(ranges, runeRange{inst.Rune[j], inst.Rune[j+1]})
			}
		case syntax.InstRuneAny, syntax.InstRuneAnyNotNL:
			needNewline = true
		case syntax.InstEmptyWidth:
			op := syntax.EmptyOp(inst.Arg)
			if op&(syntax.EmptyBeginLine|syntax.EmptyEndLine) != 0 {
				needNewline = true
			}
		}
	}

	sort.Slice(ranges, func(a, b int) bool {
		wa, wb := ranges[a].hi-ranges[a].lo, ranges[b].hi-ranges[b].lo
		if wa != wb {
			return wa < wb
		}
		return ranges[a].lo < ranges[b].lo
	})

	const maxClassRunes = 12
	added := 0
	for _, rg := range ranges {
		if added >= maxClassRunes {
			break
		}
		for _, r := range []rune{rg.lo, rg.hi, rg.lo + (rg.hi-rg.lo)/2} {
			if added >= maxClassRunes {
				break
			}
			if !utf8.ValidRune(r) {
				continue
			}
			set[r] = true
			added++
		}
	}

	set[' '] = true
	if needNewline {
		set['\n'] = true
	}
	set['é'] = true

	out := make([]rune, 0, len(set))
	for r := range set {
		out = append(out, r)
	}
	sort.Slice(out, func(a, b int) bool { return out[a] < out[b] })
	return out, nil
}

func TestParityAlphabet(t *testing.T) {
	contains := func(rs []rune, r rune) bool {
		for _, x := range rs {
			if x == r {
				return true
			}
		}
		return false
	}

	t.Run("hex", func(t *testing.T) {
		rs, err := parityAlphabet(`0x[0-9a-f]+`)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []rune{'0', 'x', '9', 'a', 'f', ' ', 'é'} {
			if !contains(rs, want) {
				t.Errorf("alphabet %q missing %q", string(rs), want)
			}
		}
		if contains(rs, '\n') {
			t.Errorf("alphabet %q unexpectedly contains a newline", string(rs))
		}
		if len(rs) > 2+12+3 {
			t.Errorf("alphabet too large: %d runes", len(rs))
		}
	})

	t.Run("fold_orbit", func(t *testing.T) {
		for p, want := range map[string]rune{`(?i)k+`: 0x212A, `(?i)s+`: 0x17F} {
			rs, err := parityAlphabet(p)
			if err != nil {
				t.Fatal(err)
			}
			if !contains(rs, want) {
				t.Errorf("alphabet for %q lacks %U: %q", p, want, string(rs))
			}
		}
	})

	t.Run("newline", func(t *testing.T) {
		for _, p := range []string{`(?m)^\w+$`, `x.y`} {
			rs, err := parityAlphabet(p)
			if err != nil {
				t.Fatal(err)
			}
			if !contains(rs, '\n') {
				t.Errorf("alphabet for %q lacks a newline: %q", p, string(rs))
			}
		}
	})

	t.Run("valid_runes", func(t *testing.T) {
		rs, err := parityAlphabet(`[^,]`)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range rs {
			if !utf8.ValidRune(r) || (r >= 0xD800 && r <= 0xDFFF) {
				t.Errorf("invalid rune %U in alphabet", r)
			}
		}
	})

	t.Run("sorted_unique_deterministic", func(t *testing.T) {
		for _, p := range []string{`0x[0-9a-fA-F]+`, `[\w.+-]+@[\w.-]+\.\w+`, `(?i)k+`} {
			a, err := parityAlphabet(p)
			if err != nil {
				t.Fatal(err)
			}
			b, _ := parityAlphabet(p)
			if string(a) != string(b) {
				t.Errorf("alphabet for %q is not deterministic", p)
			}
			for i := 1; i < len(a); i++ {
				if a[i-1] >= a[i] {
					t.Errorf("alphabet for %q is not sorted and unique: %q", p, string(a))
					break
				}
			}
		}
	})
}

// driverVariant is everything the generated driver needs for one variant.
type driverVariant struct {
	name         string
	matchPattern string
	findPattern  string
	idx          uint64
	alphabet     []rune
	extra        []string
	matchPkg     string
	findPkg      string
}

// engineOptions maps an engine to regengo options on top of base.
func engineOptions(base regengo.Options, engine parityEngine) regengo.Options {
	switch engine {
	case engineTNFA:
		base.ForceTNFA = true
	case engineTDFA:
		base.ForceTDFA = true
	case engineBacktracking:
		base.ForceBacktracking = true
	case engineThompson:
		base.ForceThompson = true
	}
	return base
}

// parityFuncBody returns the text from the line starting with header up to the
// first closing "\n}\n".
func parityFuncBody(src, header string) (string, bool) {
	start := strings.Index(src, header)
	if start < 0 {
		return "", false
	}
	rest := src[start:]
	end := strings.Index(rest, "\n}\n")
	if end < 0 {
		return "", false
	}
	return rest[:end], true
}

func matchMarkerError(engine, body string) string {
	switch engine {
	case markerDFA:
		if !strings.Contains(body, "dfaTransitionsR") {
			return `marker "dfaTransitionsR" not found`
		}
	case markerThompson:
		if !strings.Contains(body, "startClosure") {
			return `marker "startClosure" not found`
		}
	case markerBacktracking:
		if !strings.Contains(body, "StepSelect") {
			return `marker "StepSelect" not found`
		}
		if strings.Contains(body, "dfaTransitionsR") || strings.Contains(body, "startClosure") {
			return "DFA/Thompson marker present in backtracking body"
		}
	}
	return ""
}

func findMarkerError(engine, body string) string {
	switch engine {
	case markerTDFA:
		if !strings.Contains(body, "findStringInternal") {
			return `marker "findStringInternal" not found`
		}
	case markerTNFA:
		if !strings.Contains(body, "visited") || strings.Contains(body, "findStringInternal") {
			return `TNFA markers wrong (want "visited", no "findStringInternal")`
		}
	case markerBacktracking:
		if !strings.Contains(body, "TryFallback") || strings.Contains(body, "findStringInternal") {
			return `backtracking markers wrong (want "TryFallback", no "findStringInternal")`
		}
	}
	return ""
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "\n...(truncated)"
}

// driverLine is one JSON line printed by the generated driver.
type driverLine struct {
	Variant string `json:"variant"`
	Method  string `json:"method"`
	Input   string `json:"input"`
	Want    string `json:"want"`
	Got     string `json:"got"`
	Count   *int   `json:"count"`
	Done    bool   `json:"done"`
	Checked int    `json:"checked"`
}

func envParity(t *testing.T) (seed uint64, n int) {
	t.Helper()
	seed, n = defaultParitySeed, defaultParityN
	if v := os.Getenv("REGENGO_PARITY_SEED"); v != "" {
		parsed, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			t.Fatalf("invalid REGENGO_PARITY_SEED %q: %v", v, err)
		}
		seed = parsed
	}
	if v := os.Getenv("REGENGO_PARITY_N"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil || parsed < 0 {
			t.Fatalf("invalid REGENGO_PARITY_N %q: %v", v, err)
		}
		n = parsed
	}
	return seed, n
}

func methodIndex(method string) int {
	for i, m := range parityMethods {
		if m == method {
			return i
		}
	}
	return len(parityMethods)
}

// TestStdlibParity compares generated Match/Find/FindAll results with the
// standard library for every engine variant (issue #37).
func TestStdlibParity(t *testing.T) {
	if testing.Short() {
		t.Skip("stdlib parity test builds generated code; skipped in -short mode")
	}
	startTime := time.Now()

	seed, n := envParity(t)
	t.Logf("seed=%d n=%d", seed, n)

	// Typo guard for knownDivergence.
	variantNames := map[string]bool{}
	for _, c := range parityCases {
		for _, v := range c.variants {
			variantNames[parityVariantName(c, v)] = true
		}
	}
	for key, tag := range knownDivergence {
		if !variantNames[key.variant] {
			t.Errorf("knownDivergence [%s]: unknown variant %q", tag, key.variant)
		}
		if methodIndex(key.method) == len(parityMethods) {
			t.Errorf("knownDivergence [%s]: unknown method %q", tag, key.method)
		}
	}

	tmp := t.TempDir()
	pkgIdx := 0
	nextPkg := func() string {
		name := fmt.Sprintf("p%02d", pkgIdx)
		pkgIdx++
		return name
	}
	compile := func(pattern, pkg string, engine parityEngine) string {
		out := filepath.Join(tmp, pkg, "r.go")
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		opts := engineOptions(regengo.Options{
			Pattern:    pattern,
			Name:       "R",
			Package:    pkg,
			OutputFile: out,
		}, engine)
		if err := regengo.Compile(opts); err != nil {
			t.Fatalf("compile %q (%s) failed: %v", pattern, engine, err)
		}
		src, err := os.ReadFile(out)
		if err != nil {
			t.Fatalf("read %s: %v", out, err)
		}
		return string(src)
	}

	var variants []driverVariant
	for _, c := range parityCases {
		hasCaptures := regexp.MustCompile(c.pattern).NumSubexp() > 0
		alphabet, err := parityAlphabet(c.pattern)
		if err != nil {
			t.Fatalf("alphabet for %q: %v", c.pattern, err)
		}
		for _, v := range c.variants {
			name := parityVariantName(c, v)
			dv := driverVariant{
				name:         name,
				matchPattern: c.pattern,
				findPattern:  c.pattern,
				idx:          uint64(len(variants)),
				alphabet:     alphabet,
				extra:        c.extra,
				matchPkg:     nextPkg(),
			}
			matchSrc := compile(c.pattern, dv.matchPkg, v.engine)
			findSrc := matchSrc
			dv.findPkg = dv.matchPkg
			if !hasCaptures {
				dv.findPattern = "(" + c.pattern + ")"
				dv.findPkg = nextPkg()
				findSrc = compile(dv.findPattern, dv.findPkg, v.engine)
			}

			if body, ok := parityFuncBody(matchSrc, parityMatchFuncHeader); !ok {
				t.Errorf("%s: MatchString function not found in generated code", name)
			} else if msg := matchMarkerError(v.matchEngine, body); msg != "" {
				t.Errorf("%s: expected %s engine for MatchString: %s", name, v.matchEngine, msg)
			}
			if body, ok := parityFuncBody(findSrc, parityFindFuncHeader); !ok {
				t.Errorf("%s: FindStringReuse function not found in generated code", name)
			} else if msg := findMarkerError(v.findEngine, body); msg != "" {
				t.Errorf("%s: expected %s engine for FindStringReuse: %s", name, v.findEngine, msg)
			}

			variants = append(variants, dv)
		}
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	projectRoot := filepath.Join(wd, "..", "..")
	writeFile(t, filepath.Join(tmp, "go.mod"), "module paritytest\n\ngo 1.27\n\n"+
		"require github.com/KromDaniel/regengo v0.0.0\n\n"+
		"replace github.com/KromDaniel/regengo => "+projectRoot+"\n")

	var driver bytes.Buffer
	writeParityDriver(&driver, variants)
	writeFile(t, filepath.Join(tmp, "main.go"), driver.String())

	run := func(name string, args ...string) (stdout, stderr []byte) {
		cmd := exec.Command(name, args...)
		cmd.Dir = tmp
		var outBuf, errBuf bytes.Buffer
		cmd.Stdout, cmd.Stderr = &outBuf, &errBuf
		if err := cmd.Run(); err != nil {
			t.Fatalf("%s %s failed: %v\nstdout:\n%s\nstderr:\n%s", name, strings.Join(args, " "), err,
				truncate(outBuf.String(), 4096), truncate(errBuf.String(), 4096))
		}
		return outBuf.Bytes(), errBuf.Bytes()
	}

	buildStart := time.Now()
	run("go", "mod", "tidy")
	run("go", "build", "-o", "parity-driver", ".")
	buildTime := time.Since(buildStart)

	stdout, stderr := run("./parity-driver", fmt.Sprintf("-seed=%d", seed), fmt.Sprintf("-n=%d", n))
	if len(stderr) > 0 {
		t.Logf("driver stderr:\n%s", truncate(string(stderr), 4096))
	}

	counts := map[divKey]int{}
	examples := map[divKey][]driverLine{}
	checked, sawDone := 0, false
	for _, line := range strings.Split(string(stdout), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var dl driverLine
		if err := json.Unmarshal([]byte(line), &dl); err != nil {
			t.Fatalf("cannot parse driver output line %q: %v", truncate(line, 300), err)
		}
		key := divKey{variant: dl.Variant, method: dl.Method}
		switch {
		case dl.Done:
			sawDone, checked = true, dl.Checked
		case dl.Count != nil:
			counts[key] = *dl.Count
		default:
			examples[key] = append(examples[key], dl)
		}
	}
	if !sawDone {
		t.Fatalf("driver output has no done line (driver crashed?):\n%s", truncate(string(stdout), 4096))
	}

	keys := make([]divKey, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(a, b int) bool {
		if keys[a].variant != keys[b].variant {
			return keys[a].variant < keys[b].variant
		}
		return methodIndex(keys[a].method) < methodIndex(keys[b].method)
	})

	rerun := fmt.Sprintf("seed=%d; rerun: REGENGO_PARITY_SEED=%d go test ./tests/integration -run StdlibParity -v", seed, seed)
	for _, key := range keys {
		exs := examples[key]
		if tag, ok := knownDivergence[key]; ok {
			t.Logf("known divergence [%s] %s %s: %d mismatches", tag, key.variant, key.method, counts[key])
			if len(exs) > 0 {
				t.Logf("  first example: input=%q want=%s got=%s", exs[0].Input, exs[0].Want, exs[0].Got)
			}
			continue
		}
		for _, ex := range exs {
			t.Errorf("%s %s: input=%q want=%s got=%s (%s)", key.variant, key.method, ex.Input, ex.Want, ex.Got, rerun)
		}
		t.Errorf("%s %s: %d mismatches in total", key.variant, key.method, counts[key])
	}

	// Stale entries: a listed divergence that no longer occurs must be deleted.
	var staleKeys []divKey
	for key := range knownDivergence {
		if counts[key] == 0 {
			staleKeys = append(staleKeys, key)
		}
	}
	sort.Slice(staleKeys, func(a, b int) bool {
		if staleKeys[a].variant != staleKeys[b].variant {
			return staleKeys[a].variant < staleKeys[b].variant
		}
		return methodIndex(staleKeys[a].method) < methodIndex(staleKeys[b].method)
	})
	for _, key := range staleKeys {
		tag := knownDivergence[key]
		if seed == defaultParitySeed && n >= defaultParityN {
			t.Errorf("knownDivergence %v [%s] no longer diverges; delete the entry", key, tag)
		} else {
			t.Logf("knownDivergence %v [%s] did not diverge with this seed/n (not enforced)", key, tag)
		}
	}

	t.Logf("checked %d comparisons across %d variants in %s (go mod tidy + build: %s)",
		checked, len(variants), time.Since(startTime).Round(time.Millisecond), buildTime.Round(time.Millisecond))
}

// writeParityDriver writes the source of the generated driver program.
func writeParityDriver(w io.Writer, vs []driverVariant) {
	var imports strings.Builder
	seen := map[string]bool{}
	for _, v := range vs {
		for _, pkg := range []string{v.matchPkg, v.findPkg} {
			if pkg != "" && !seen[pkg] {
				seen[pkg] = true
				fmt.Fprintf(&imports, "\t%q\n", "paritytest/"+pkg)
			}
		}
	}

	_, _ = io.WriteString(w, driverHeader)
	_, _ = io.WriteString(w, imports.String())
	_, _ = io.WriteString(w, driverTypes)

	fmt.Fprintln(w, "var variants = []variant{")
	for _, v := range vs {
		writeDriverVariant(w, v)
	}
	fmt.Fprintln(w, "}")

	_, _ = io.WriteString(w, driverMain)
}

func writeDriverVariant(w io.Writer, v driverVariant) {
	var alphabet, extra strings.Builder
	for _, r := range v.alphabet {
		fmt.Fprintf(&alphabet, "%d, ", r)
	}
	for _, s := range v.extra {
		fmt.Fprintf(&extra, "%s, ", strconv.Quote(s))
	}
	m, f := v.matchPkg, v.findPkg
	fmt.Fprintf(w, "\t{\n")
	fmt.Fprintf(w, "\t\tname: %s, matchPattern: %s, findPattern: %s, idx: %d,\n",
		strconv.Quote(v.name), strconv.Quote(v.matchPattern), strconv.Quote(v.findPattern), v.idx)
	fmt.Fprintf(w, "\t\talphabet: []rune{%s},\n", alphabet.String())
	fmt.Fprintf(w, "\t\textra: []string{%s},\n", extra.String())
	fmt.Fprintf(w, "\t\tmatch: %s.CompiledR.MatchString,\n", m)
	fmt.Fprintf(w, "\t\tmatchB: %s.CompiledR.MatchBytes,\n", m)
	if f == "" {
		fmt.Fprintf(w, "\t},\n")
		return
	}
	fmt.Fprintf(w, "\t\tfind: func(s string) (any, bool) { r, ok := %s.CompiledR.FindString(s); return r, ok },\n", f)
	fmt.Fprintf(w, "\t\tfindB: func(b []byte) (any, bool) { r, ok := %s.CompiledR.FindBytes(b); return r, ok },\n", f)
	fmt.Fprintf(w, "\t\tfindAll: func(s string, n int) []any {\n")
	fmt.Fprintf(w, "\t\t\trs := %s.CompiledR.FindAllString(s, n)\n", f)
	fmt.Fprintf(w, "\t\t\tout := make([]any, len(rs))\n")
	fmt.Fprintf(w, "\t\t\tfor i, r := range rs {\n\t\t\t\tout[i] = r\n\t\t\t}\n")
	fmt.Fprintf(w, "\t\t\treturn out\n\t\t},\n")
	fmt.Fprintf(w, "\t\tfindAllB: func(b []byte, n int) []any {\n")
	fmt.Fprintf(w, "\t\t\trs := %s.CompiledR.FindAllBytes(b, n)\n", f)
	fmt.Fprintf(w, "\t\t\tout := make([]any, len(rs))\n")
	fmt.Fprintf(w, "\t\t\tfor i, r := range rs {\n\t\t\t\tout[i] = r\n\t\t\t}\n")
	fmt.Fprintf(w, "\t\t\treturn out\n\t\t},\n")
	for _, isBytes := range []bool{false, true} {
		field, kind, elem, call := "findAllApp", "string", "RResult", "FindAllStringAppend"
		if isBytes {
			field, kind, elem, call = "findAllAppB", "[]byte", "RBytesResult", "FindAllBytesAppend"
		}
		fmt.Fprintf(w, "\t\t%s: func() func(%s, int) []any {\n", field, kind)
		fmt.Fprintf(w, "\t\t\tvar buf []*%s.%s\n", f, elem)
		fmt.Fprintf(w, "\t\t\tsentinel := &%s.%s{}\n", f, elem)
		fmt.Fprintf(w, "\t\t\treturn func(s %s, n int) []any {\n", kind)
		fmt.Fprintf(w, "\t\t\t\tbuf = %s.CompiledR.%s(s, n, append(buf[:0], sentinel))\n", f, call)
		fmt.Fprintf(w, "\t\t\t\tif len(buf) == 0 || buf[0] != sentinel {\n\t\t\t\t\tpanic(\"Append dropped the prefix element\")\n\t\t\t\t}\n")
		fmt.Fprintf(w, "\t\t\t\tout := make([]any, len(buf)-1)\n")
		fmt.Fprintf(w, "\t\t\t\tfor i, r := range buf[1:] {\n\t\t\t\t\tout[i] = r\n\t\t\t\t}\n")
		fmt.Fprintf(w, "\t\t\t\treturn out\n\t\t\t}\n\t\t}(),\n")
	}
	fmt.Fprintf(w, "\t},\n")
}

const driverHeader = `// Code generated by the stdlib parity test. DO NOT EDIT.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"

`

const driverTypes = `)

type variant struct {
	name, matchPattern, findPattern string
	idx                             uint64
	alphabet                        []rune
	extra                           []string
	match                           func(string) bool
	matchB                          func([]byte) bool
	find                            func(string) (any, bool)
	findB                           func([]byte) (any, bool)
	findAll                         func(string, int) []any
	findAllB                        func([]byte, int) []any
	findAllApp                      func(string, int) []any
	findAllAppB                     func([]byte, int) []any
}

`

const driverMain = `
var methods = []string{
	"MatchString",
	"MatchBytes",
	"FindString",
	"FindBytes",
	"FindAllString(n=-1)",
	"FindAllString(n=2)",
	"FindAllBytes(n=-1)",
	"FindAllBytes(n=2)",
	"FindAllStringAppend(n=-1)",
	"FindAllStringAppend(n=2)",
	"FindAllBytesAppend(n=-1)",
	"FindAllBytesAppend(n=2)",
}

type mismatchLine struct {
	Variant string ` + "`json:\"variant\"`" + `
	Method  string ` + "`json:\"method\"`" + `
	Input   string ` + "`json:\"input\"`" + `
	Want    string ` + "`json:\"want\"`" + `
	Got     string ` + "`json:\"got\"`" + `
}

type countLine struct {
	Variant string ` + "`json:\"variant\"`" + `
	Method  string ` + "`json:\"method\"`" + `
	Count   int    ` + "`json:\"count\"`" + `
}

type doneLine struct {
	Done    bool ` + "`json:\"done\"`" + `
	Checked int  ` + "`json:\"checked\"`" + `
}

// safe runs f and turns a panic into a result string (counted as a mismatch).
func safe(f func() string) (out string) {
	defer func() {
		if r := recover(); r != nil {
			out = fmt.Sprintf("panic: %v", r)
		}
	}()
	return f()
}

// renderWant renders a stdlib submatch index result: "nil" for no match,
// otherwise {"m", "g1", ...}. With positions (Bytes forms) the match gets
// " @[start,end]" and each non-empty group " @start".
func renderWant(s string, loc []int, withPos bool) string {
	if loc == nil {
		return "nil"
	}
	parts := make([]string, 0, len(loc)/2)
	for i := 0; i+1 < len(loc); i += 2 {
		field := ""
		if loc[i] >= 0 {
			field = s[loc[i]:loc[i+1]]
		}
		part := strconv.Quote(field)
		if withPos {
			if i == 0 {
				part += fmt.Sprintf(" @[%d,%d]", loc[0], loc[1])
			} else if field != "" {
				part += fmt.Sprintf(" @%d", loc[i])
			}
		}
		parts = append(parts, part)
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// renderGot renders a generated result struct pointer the same way. capB is
// cap(input) for the Bytes forms (positions are derived from field capacities).
func renderGot(r any, isBytes bool, capB, numSubexp int) string {
	rv := reflect.ValueOf(r).Elem()
	if rv.NumField() != numSubexp+1 {
		fmt.Fprintf(os.Stderr, "harness error: result has %d fields, want %d\n", rv.NumField(), numSubexp+1)
		os.Exit(2)
	}
	parts := make([]string, 0, rv.NumField())
	for i := 0; i < rv.NumField(); i++ {
		f := rv.Field(i)
		if !isBytes {
			parts = append(parts, strconv.Quote(f.String()))
			continue
		}
		val := string(f.Bytes())
		part := strconv.Quote(val)
		start := capB - f.Cap()
		if i == 0 {
			part += fmt.Sprintf(" @[%d,%d]", start, start+len(val))
		} else if val != "" {
			part += fmt.Sprintf(" @%d", start)
		}
		parts = append(parts, part)
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func main() {
	seed := flag.Uint64("seed", 37, "random seed")
	n := flag.Int("n", 2000, "random inputs per variant")
	flag.Parse()

	counts := map[string]int{}
	checked := 0
	enc := json.NewEncoder(os.Stdout)

	for _, v := range variants {
		reM := regexp.MustCompile(v.matchPattern)
		reF := regexp.MustCompile(v.findPattern)
		ng := reF.NumSubexp()

		rng := rand.New(rand.NewPCG(*seed, v.idx))
		inputs := append([]string{}, v.extra...)
		for i := 0; i < *n; i++ {
			var sb strings.Builder
			for j, l := 0, rng.IntN(13); j < l; j++ {
				sb.WriteRune(v.alphabet[rng.IntN(len(v.alphabet))])
			}
			inputs = append(inputs, sb.String())
		}

		var input string
		report := func(method, want, got string) {
			checked++
			if want == got {
				return
			}
			key := v.name + "\x00" + method
			counts[key]++
			if counts[key] <= 5 {
				_ = enc.Encode(mismatchLine{v.name, method, input, want, got})
			}
		}

		for _, s := range inputs {
			input = s

			report("MatchString", fmt.Sprint(reM.MatchString(s)),
				safe(func() string { return fmt.Sprint(v.match(s)) }))
			report("MatchBytes", fmt.Sprint(reM.MatchString(s)),
				safe(func() string { return fmt.Sprint(v.matchB([]byte(s))) }))

			if v.find == nil || v.findB == nil || v.findAll == nil || v.findAllB == nil || v.findAllApp == nil || v.findAllAppB == nil {
				continue
			}

			report("FindString", renderWant(s, reF.FindStringSubmatchIndex(s), false),
				safe(func() string {
					r, ok := v.find(s)
					if !ok {
						return "nil"
					}
					return renderGot(r, false, 0, ng)
				}))
			report("FindBytes", renderWant(s, reF.FindStringSubmatchIndex(s), true),
				safe(func() string {
					b := []byte(s)
					r, ok := v.findB(b)
					if !ok {
						return "nil"
					}
					return renderGot(r, true, cap(b), ng)
				}))

			for _, lim := range []int{-1, 2} {
				locs := reF.FindAllStringSubmatchIndex(s, lim)
				for _, combo := range [4][2]bool{{false, false}, {true, false}, {false, true}, {true, true}} {
					isBytes, isApp := combo[0], combo[1]
					wants := make([]string, len(locs))
					for i, loc := range locs {
						wants[i] = renderWant(s, loc, isBytes)
					}
					want := "[" + strings.Join(wants, ", ") + "]"
					method := "FindAllString(n=" + strconv.Itoa(lim) + ")"
					if isBytes {
						method = "FindAllBytes(n=" + strconv.Itoa(lim) + ")"
					}
					if isApp {
						method = strings.Replace(method, "(n=", "Append(n=", 1)
					}
					report(method, want, safe(func() string {
						var items []any
						capB := 0
						if isBytes {
							b := []byte(s)
							capB = cap(b)
							if isApp {
								items = v.findAllAppB(b, lim)
							} else {
								items = v.findAllB(b, lim)
							}
						} else if isApp {
							items = v.findAllApp(s, lim)
						} else {
							items = v.findAll(s, lim)
						}
						gots := make([]string, len(items))
						for i, it := range items {
							gots[i] = renderGot(it, isBytes, capB, ng)
						}
						return "[" + strings.Join(gots, ", ") + "]"
					}))
				}
			}
		}
	}

	for _, v := range variants {
		for _, m := range methods {
			if c := counts[v.name+"\x00"+m]; c > 0 {
				_ = enc.Encode(countLine{v.name, m, c})
			}
		}
	}
	_ = enc.Encode(doneLine{true, checked})
}
`
