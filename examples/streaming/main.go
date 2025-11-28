// Example: Streaming regex matching with regengo
//
// This example demonstrates how to use the streaming API to process
// large files or network streams with constant memory usage.
//
// To run this example:
//  1. First generate the date pattern:
//     go run ../../cmd/regengo/main.go -pattern '\d{4}-\d{2}-\d{2}' \
//     -name DatePattern -output date_pattern.go -package main
//  2. Then run: go run .
package main

import (
	"fmt"
	"io"
	"math/rand"
	"strings"

	stream "github.com/KromDaniel/regengo/stream"
)

// The following imports would be used with a generated pattern:
// import "your/module/generated"

func main() {
	fmt.Println("=== Streaming Regex Example ===")

	// Example 1: Basic streaming match with callback
	fmt.Println("1. Basic streaming match:")
	basicStreamingExample()

	// Example 2: Counting matches in a stream
	fmt.Println("\n2. Counting matches:")
	countingExample()

	// Example 3: Finding first match
	fmt.Println("\n3. Finding first match:")
	firstMatchExample()

	// Example 4: Early termination
	fmt.Println("\n4. Early termination (first 3 matches):")
	earlyTerminationExample()

	// Example 5: Processing large data
	fmt.Println("\n5. Processing large generated data:")
	largeDataExample()
}

// PatternedReader generates test data with embedded date patterns
type PatternedReader struct {
	pattern    []byte
	noise      []byte
	matchEvery int
	pos        int64
	limit      int64
	rng        *rand.Rand
}

func NewPatternedReader(pattern string, matchEvery int, limit int64) *PatternedReader {
	return &PatternedReader{
		pattern:    []byte(pattern),
		noise:      []byte("abcdefghijk \n\t"),
		matchEvery: matchEvery,
		limit:      limit,
		rng:        rand.New(rand.NewSource(42)),
	}
}

func (r *PatternedReader) Read(p []byte) (n int, err error) {
	if r.pos >= r.limit {
		return 0, io.EOF
	}
	for n < len(p) && r.pos < r.limit {
		posInCycle := int(r.pos) % r.matchEvery
		if posInCycle < len(r.pattern) {
			p[n] = r.pattern[posInCycle]
		} else {
			p[n] = r.noise[r.rng.Intn(len(r.noise))]
		}
		n++
		r.pos++
	}
	return n, nil
}

// For demonstration, we'll simulate what the generated API looks like.
// In real usage, you would use the generated DatePattern struct.

func basicStreamingExample() {
	input := "Log entries: 2024-01-15 event, 2024-02-20 another, 2024-12-31 final"

	// With a generated pattern, you would do:
	// err := CompiledDatePattern.FindReader(strings.NewReader(input), stream.Config{},
	//     func(m stream.Match[*DatePatternBytesResult]) bool {
	//         fmt.Printf("  Found: %s at offset %d\n", m.Result.Match, m.StreamOffset)
	//         return true // continue
	//     })

	fmt.Printf("  Input: %q\n", input)
	fmt.Println("  (Use generated pattern to find matches)")
	fmt.Println("  Expected matches: 2024-01-15, 2024-02-20, 2024-12-31")
}

func countingExample() {
	input := "2024-01-01 2024-02-02 2024-03-03 2024-04-04 2024-05-05"

	// With a generated pattern:
	// count, err := CompiledDatePattern.FindReaderCount(
	//     strings.NewReader(input),
	//     stream.Config{},
	// )
	// fmt.Printf("  Found %d matches\n", count)

	fmt.Printf("  Input: %q\n", input)
	fmt.Println("  (Use FindReaderCount to count matches)")
	fmt.Println("  Expected count: 5")
}

func firstMatchExample() {
	input := "Some text before 2024-07-15 and more text"

	// With a generated pattern:
	// result, found, err := CompiledDatePattern.FindReaderFirst(
	//     strings.NewReader(input),
	//     stream.Config{},
	// )
	// if found {
	//     fmt.Printf("  First match: %s\n", result.Match)
	// }

	fmt.Printf("  Input: %q\n", input)
	fmt.Println("  (Use FindReaderFirst to get first match)")
	fmt.Println("  Expected: 2024-07-15")
}

func earlyTerminationExample() {
	input := strings.Repeat("2024-01-01 ", 100) // Many dates
	_ = input                                   // Used by demonstration

	// With a generated pattern:
	// var count int
	// CompiledDatePattern.FindReader(strings.NewReader(input), stream.Config{},
	//     func(m stream.Match[*DatePatternBytesResult]) bool {
	//         fmt.Printf("  Match %d: %s\n", count+1, m.Result.Match)
	//         count++
	//         return count < 3 // Stop after 3 matches
	//     })

	fmt.Printf("  Input: 100 dates (showing first 3)\n")
	fmt.Println("  (Return false from callback to stop early)")
	fmt.Println("  Expected: First 3 matches then stop")
}

func largeDataExample() {
	// Generate 1MB of data with dates every 50 bytes
	gen := NewPatternedReader("2024-01-15", 50, 1<<20)
	_ = gen // Used by demonstration
	expectedMatches := int64(1<<20) / 50

	fmt.Printf("  Generated: 1MB of data (~%d expected matches)\n", expectedMatches)

	// With a generated pattern:
	// count, err := CompiledDatePattern.FindReaderCount(gen, stream.Config{
	//     BufferSize: 64 * 1024, // 64KB buffer
	// })
	// fmt.Printf("  Found %d matches\n", count)

	// Show stream.Config options
	cfg := stream.DefaultConfig()
	fmt.Printf("  Default buffer size: %d bytes\n", cfg.BufferSize)
	fmt.Println("  Memory usage: Constant (~BufferSize) regardless of input")
	fmt.Println("  (Use FindReaderCount on the PatternedReader)")
}
