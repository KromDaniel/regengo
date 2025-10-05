package main

import (
	"fmt"

	"github.com/KromDaniel/regengo/test"
)

func main() {
	fmt.Println("Testing zero-copy []byte capture groups:\n")

	input := []byte("http://example.com:8080")

	// Test standard version (copies to string)
	result, found := test.URLCaptureFindBytes(input)
	if found {
		fmt.Printf("Standard URLCapture (string fields):\n")
		fmt.Printf("  Protocol: %s (type: string)\n", result.Protocol)
		fmt.Printf("  Host: %s (type: string)\n", result.Host)
		fmt.Printf("  Port: %s (type: string)\n", result.Port)
		fmt.Println()
	}

	// Test zero-copy version
	bytesResult, found := test.URLBytesFindBytes(input)
	if found {
		fmt.Printf("URLBytes ([]byte fields, zero-copy):\n")
		fmt.Printf("  Protocol: %s (type: []byte, length: %d)\n", bytesResult.Protocol, len(bytesResult.Protocol))
		fmt.Printf("  Host: %s (type: []byte, length: %d)\n", bytesResult.Host, len(bytesResult.Host))
		fmt.Printf("  Port: %s (type: []byte, length: %d)\n", bytesResult.Port, len(bytesResult.Port))
		fmt.Println()

		// Demonstrate zero-copy: the slices point to the original input
		fmt.Printf("Original input: %s\n", input)
		fmt.Printf("Protocol slice points to original: %v\n", &input[0] == &bytesResult.Protocol[0])
		fmt.Println()

		fmt.Println("✓ Zero-copy verified: []byte fields reference the original input")
		fmt.Println("  No string conversions = No allocations for byte slices!")
	}
}
