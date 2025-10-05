package main

import (
	"fmt"
	"regexp/syntax"
)

func main() {
	patterns := []string{
		"^[a-z]{3}$",
		"^\\d{4}$",
		"^(?:foo|bar){2}baz\\d{2}$",
	}

	for _, pattern := range patterns {
		fmt.Printf("\n=== Pattern: %s ===\n", pattern)

		re, err := syntax.Parse(pattern, syntax.Perl)
		if err != nil {
			fmt.Printf("Parse error: %v\n", err)
			continue
		}

		// Simplify the regex
		re = re.Simplify()

		prog, err := syntax.Compile(re)
		if err != nil {
			fmt.Printf("Compile error: %v\n", err)
			continue
		}

		fmt.Printf("Program has %d instructions:\n", len(prog.Inst))
		for i, inst := range prog.Inst {
			fmt.Printf("  [%d] %s", i, inst.Op)
			if inst.Op == syntax.InstEmptyWidth {
				fmt.Printf(" (Arg=%d, EmptyOp flags:", inst.Arg)
				if inst.Arg&uint32(syntax.EmptyBeginLine) != 0 {
					fmt.Printf(" BeginLine")
				}
				if inst.Arg&uint32(syntax.EmptyEndLine) != 0 {
					fmt.Printf(" EndLine")
				}
				if inst.Arg&uint32(syntax.EmptyBeginText) != 0 {
					fmt.Printf(" BeginText")
				}
				if inst.Arg&uint32(syntax.EmptyEndText) != 0 {
					fmt.Printf(" EndText")
				}
				if inst.Arg&uint32(syntax.EmptyWordBoundary) != 0 {
					fmt.Printf(" WordBoundary")
				}
				if inst.Arg&uint32(syntax.EmptyNoWordBoundary) != 0 {
					fmt.Printf(" NoWordBoundary")
				}
				fmt.Printf(")")
			}
			fmt.Printf(" Out=%d\n", inst.Out)
		}
	}
}
