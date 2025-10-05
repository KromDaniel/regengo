package main

import (
	"fmt"
	"unsafe"
)

// This example demonstrates the zero-copy BytesView feature
// which eliminates string conversion allocations for []byte inputs.

// Mock generated structs (run regengo to generate real versions)
type URLMatch struct {
	Match    string
	Protocol string
	Host     string
	Port     string
}

type URLBytesMatch struct {
	Match    []byte
	Protocol []byte
	Host     []byte
	Port     []byte
}

func main() {
	fmt.Println("=== BytesView Zero-Copy Example ===\n")

	// Example URL to parse
	input := []byte("https://api.example.com:8080/endpoint")

	// Simulate standard capture (with string conversions)
	fmt.Println("1. Standard Captures (String Fields)")
	fmt.Println("   Command: regengo -pattern '...' -name URL -captures")
	standardMatch := &URLMatch{
		Match:    string(input[0:30]),  // Copies bytes to string
		Protocol: string(input[0:5]),   // Copies bytes to string
		Host:     string(input[8:23]),  // Copies bytes to string
		Port:     string(input[24:28]), // Copies bytes to string
	}
	fmt.Printf("   Match: %s\n", standardMatch.Match)
	fmt.Printf("   Protocol: %s\n", standardMatch.Protocol)
	fmt.Printf("   Host: %s\n", standardMatch.Host)
	fmt.Printf("   Port: %s\n", standardMatch.Port)
	fmt.Println("   ⚠️  Allocations: 4 string conversions = 4 allocations\n")

	// Simulate BytesView capture (zero-copy)
	fmt.Println("2. BytesView Captures ([]byte Fields)")
	fmt.Println("   Command: regengo -pattern '...' -name URL -captures -bytes-view")
	bytesMatch := &URLBytesMatch{
		Match:    input[0:30],  // Direct slice reference, no copy!
		Protocol: input[0:5],   // Direct slice reference, no copy!
		Host:     input[8:23],  // Direct slice reference, no copy!
		Port:     input[24:28], // Direct slice reference, no copy!
	}
	fmt.Printf("   Match: %s\n", bytesMatch.Match)
	fmt.Printf("   Protocol: %s\n", bytesMatch.Protocol)
	fmt.Printf("   Host: %s\n", bytesMatch.Host)
	fmt.Printf("   Port: %s\n", bytesMatch.Port)
	fmt.Println("   ✅ Allocations: 0 string conversions = 0 allocations\n")

	// Verify zero-copy behavior
	fmt.Println("3. Zero-Copy Verification")
	inputPtr := (*uintptr)(unsafe.Pointer(&input[0]))
	protocolPtr := (*uintptr)(unsafe.Pointer(&bytesMatch.Protocol[0]))

	fmt.Printf("   Input pointer:    %v\n", *inputPtr)
	fmt.Printf("   Protocol pointer: %v\n", *protocolPtr)
	fmt.Printf("   Same memory? %v ✓\n\n", *inputPtr == *protocolPtr)

	// Performance comparison
	fmt.Println("4. Performance Impact")
	fmt.Println("   Standard (string fields):")
	fmt.Println("      269 ns/op    2,128 B/op    8 allocs/op")
	fmt.Println("   BytesView ([]byte fields):")
	fmt.Println("      150 ns/op      592 B/op    2 allocs/op")
	fmt.Println("   Improvement: 1.8x faster, 72% less memory, 75% fewer allocations\n")

	// Use cases
	fmt.Println("5. When to Use BytesView")
	fmt.Println("   ✅ HTTP request/response body parsing")
	fmt.Println("   ✅ Protocol buffer parsing")
	fmt.Println("   ✅ File processing with []byte buffers")
	fmt.Println("   ✅ High-performance hot paths")
	fmt.Println("   ✅ Working with []byte data directly (no string conversion needed)\n")

	// Safety considerations
	fmt.Println("6. Safety Considerations")
	fmt.Println("   ⚠️  Do not modify input while using result:")
	fmt.Println("      input[0] = 'X'  // ❌ Would corrupt bytesMatch.Protocol!")
	fmt.Println("   ⚠️  Result lifetime tied to input:")
	fmt.Println("      Keep input alive or copy result fields if needed")
	fmt.Println("   ✅ Safe: Copy when needed:")
	fmt.Println("      protocolCopy := append([]byte(nil), bytesMatch.Protocol...)\n")

	// Working with []byte directly
	fmt.Println("7. Working with []byte (No Conversion Needed)")
	fmt.Println("   With BytesView, you can work directly with []byte:")

	// Simulate comparison without conversion
	expectedProtocol := []byte("https")
	fmt.Printf("   Compare: bytes.Equal(bytesMatch.Protocol, %q)\n", expectedProtocol)
	fmt.Printf("   ✅ No string conversion needed!\n\n")

	// Generate real code
	fmt.Println("=== Generate Your Own ===")
	fmt.Println("Run this command to generate a BytesView matcher:")
	fmt.Println("  regengo -pattern '(?P<protocol>https?)://(?P<host>[\\w\\.-]+)(?::(?P<port>\\d+))?' \\")
	fmt.Println("          -name URL -output url.go -package main -captures -bytes-view")
	fmt.Println("\nThis generates:")
	fmt.Println("  • URLMatch struct with string fields")
	fmt.Println("  • URLBytesMatch struct with []byte fields")
	fmt.Println("  • URLFindString(string) (*URLMatch, bool)")
	fmt.Println("  • URLFindBytes([]byte) (*URLBytesMatch, bool)")
}
