//go:build ignore

// This example demonstrates the Transform API for streaming transformations.
// Run: go run generate.go && go run main.go email.go
package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/KromDaniel/regengo/stream"
)

func main() {
	input := `Server logs from 2024-12-01:
DEBUG: Starting server
INFO: User john@example.com logged in
ERROR: Failed connection from admin@internal.corp
INFO: User jane@company.org performed action
DEBUG: Heartbeat check
INFO: User john@example.com logged out
`

	fmt.Println("=== Original Input ===")
	fmt.Println(input)

	// Example 1: Simple replacement
	fmt.Println("=== 1. ReplaceReader: Mask all emails ===")
	r1 := CompiledEmail.ReplaceReader(strings.NewReader(input), "[REDACTED]")
	result1, _ := io.ReadAll(r1)
	fmt.Println(string(result1))

	// Example 2: Template replacement with captures
	fmt.Println("=== 2. ReplaceReader: Format emails ===")
	r2 := CompiledEmail.ReplaceReader(strings.NewReader(input), "<$user AT $domain>")
	result2, _ := io.ReadAll(r2)
	fmt.Println(string(result2))

	// Example 3: Select only matches
	fmt.Println("=== 3. SelectReader: Extract all emails ===")
	r3 := CompiledEmail.SelectReader(strings.NewReader(input), func(m *EmailBytesResult) bool {
		return true
	})
	result3, _ := io.ReadAll(r3)
	fmt.Println(string(result3))
	fmt.Println()

	// Example 4: Filter by domain
	fmt.Println("=== 4. SelectReader: Only .com emails ===")
	r4 := CompiledEmail.SelectReader(strings.NewReader(input), func(m *EmailBytesResult) bool {
		return bytes.HasSuffix(m.Domain, []byte(".com"))
	})
	result4, _ := io.ReadAll(r4)
	fmt.Println(string(result4))
	fmt.Println()

	// Example 5: Remove matches
	fmt.Println("=== 5. RejectReader: Remove internal emails ===")
	r5 := CompiledEmail.RejectReader(strings.NewReader(input), func(m *EmailBytesResult) bool {
		return bytes.HasSuffix(m.Domain, []byte(".corp"))
	})
	result5, _ := io.ReadAll(r5)
	fmt.Println(string(result5))

	// Example 6: Custom transformation with NewTransformReader
	fmt.Println("=== 6. NewTransformReader: Multi-emit expansion ===")
	r6 := CompiledEmail.NewTransformReader(
		strings.NewReader("Contact: alice@test.com"),
		stream.DefaultTransformConfig(),
		func(m *EmailBytesResult, emit func([]byte)) {
			emit([]byte("[User: "))
			emit(m.User)
			emit([]byte(", Domain: "))
			emit(m.Domain)
			emit([]byte("]"))
		})
	result6, _ := io.ReadAll(r6)
	fmt.Println(string(result6))
	fmt.Println()

	// Example 7: Conditional transformation
	fmt.Println("=== 7. NewTransformReader: Conditional (keep public, mask internal) ===")
	r7 := CompiledEmail.NewTransformReader(
		strings.NewReader(input),
		stream.DefaultTransformConfig(),
		func(m *EmailBytesResult, emit func([]byte)) {
			if bytes.HasSuffix(m.Domain, []byte(".corp")) {
				emit([]byte("[INTERNAL]"))
			} else {
				emit(m.Match) // pass through unchanged
			}
		})
	result7, _ := io.ReadAll(r7)
	fmt.Println(string(result7))

	// Example 8: Chaining transformations
	fmt.Println("=== 8. Chaining: LineFilter + ReplaceReader ===")
	var r8 io.Reader = strings.NewReader(input)
	// First: filter to only ERROR/INFO lines
	r8 = stream.LineFilter(r8, func(line []byte) bool {
		return bytes.Contains(line, []byte("ERROR")) || bytes.Contains(line, []byte("INFO"))
	})
	// Then: mask emails
	r8 = CompiledEmail.ReplaceReader(r8, "[EMAIL]")
	result8, _ := io.ReadAll(r8)
	fmt.Println(string(result8))
}
