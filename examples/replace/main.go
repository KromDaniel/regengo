//go:build ignore

// This example demonstrates the Replace API.
// Run: go run generate.go && go run main.go email.go
package main

import "fmt"

func main() {
	input := "Contact support@example.com or sales@company.org for help"

	fmt.Println("Input:", input)
	fmt.Println()

	// Pre-compiled replacers (fastest - no template parsing at runtime)
	fmt.Println("=== Pre-compiled Replacers ===")

	// Replacer 0: "$user@REDACTED.$tld"
	result0 := CompiledEmail.ReplaceAllString0(input)
	fmt.Println("Mask domain:", result0)

	// Replacer 1: "[EMAIL REMOVED]"
	result1 := CompiledEmail.ReplaceAllString1(input)
	fmt.Println("Full redact:", result1)

	// Replacer 2: "$user@***.$tld"
	result2 := CompiledEmail.ReplaceAllString2(input)
	fmt.Println("Partial mask:", result2)

	fmt.Println()

	// Runtime replacer (flexible - any template at runtime)
	fmt.Println("=== Runtime Replacer ===")

	// Custom template at runtime
	result := CompiledEmail.ReplaceAllString(input, "[$user AT $domain]")
	fmt.Println("Custom format:", result)

	// Full match reference
	result = CompiledEmail.ReplaceAllString(input, "<$0>")
	fmt.Println("Wrap match:", result)

	// Replace first only
	result = CompiledEmail.ReplaceFirstString(input, "[FIRST EMAIL]")
	fmt.Println("First only:", result)

	fmt.Println()

	// Zero-allocation example
	fmt.Println("=== Zero-Allocation ===")
	inputs := [][]byte{
		[]byte("user1@a.com"),
		[]byte("user2@b.org"),
		[]byte("user3@c.net"),
	}
	buf := make([]byte, 0, 256)
	for _, in := range inputs {
		buf = CompiledEmail.ReplaceAllBytesAppend0(in, buf)
		fmt.Printf("  %s -> %s\n", in, buf)
		buf = buf[:0]
	}
}
