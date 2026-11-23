package compiler

import (
	"regexp/syntax"
	"sort"
	"unicode"
)

// foldOrbit returns r and every rune in its unicode.SimpleFold orbit.
func foldOrbit(r rune) []rune {
	orbit := []rune{r}
	for f := unicode.SimpleFold(r); f != r; f = unicode.SimpleFold(f) {
		orbit = append(orbit, f)
	}
	return orbit
}

// foldOrbitPairs returns the SimpleFold orbit of r as sorted [lo, hi] pairs,
// merging adjacent code points into a single range.
func foldOrbitPairs(r rune) []rune {
	orbit := foldOrbit(r)
	sort.Slice(orbit, func(i, j int) bool { return orbit[i] < orbit[j] })
	pairs := make([]rune, 0, len(orbit)*2)
	for _, o := range orbit {
		if n := len(pairs); n > 0 && pairs[n-1]+1 == o {
			pairs[n-1] = o
			continue
		}
		pairs = append(pairs, o, o)
	}
	return pairs
}

// NormalizeFoldCase rewrites every InstRune that holds a single rune with the
// syntax.FoldCase flag (what regexp/syntax emits for a case-folded literal)
// into an ordinary character class: the rune's SimpleFold orbit as sorted
// [lo, hi] pairs, with the FoldCase bit cleared. Generators then only deal
// with the pair form.
//
// It is copy-on-write: if nothing needs rewriting, prog itself is returned;
// otherwise a shallow copy with a fresh Inst slice is returned and the input
// is never mutated. Instruction indices and links are unchanged. It is
// idempotent. A nil prog yields nil.
func NormalizeFoldCase(prog *syntax.Prog) *syntax.Prog {
	if prog == nil {
		return nil
	}
	needs := func(inst *syntax.Inst) bool {
		return inst.Op == syntax.InstRune && len(inst.Rune) == 1 &&
			syntax.Flags(inst.Arg)&syntax.FoldCase != 0
	}
	first := -1
	for i := range prog.Inst {
		if needs(&prog.Inst[i]) {
			first = i
			break
		}
	}
	if first < 0 {
		return prog
	}
	out := *prog
	out.Inst = make([]syntax.Inst, len(prog.Inst))
	copy(out.Inst, prog.Inst)
	for i := first; i < len(out.Inst); i++ {
		inst := &out.Inst[i]
		if !needs(inst) {
			continue
		}
		inst.Rune = foldOrbitPairs(inst.Rune[0])
		inst.Arg &^= uint32(syntax.FoldCase)
	}
	return &out
}
