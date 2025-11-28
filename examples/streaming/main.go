// Example: Streaming regex matching with regengo
//
// This example demonstrates how to use the streaming API to process
// large files or network streams with constant memory usage.
//
// Run with: go generate && go run .
package main

import (
	"fmt"
	"io"
	"math/rand"
	"strings"

	stream "github.com/KromDaniel/regengo/stream"
)

//go:generate go run ../../cmd/regengo/main.go -pattern (\d{4}-\d{2}-\d{2}) -name DatePattern -output date_pattern.go -package main -no-test

func main() {
	fmt.Println("=== Streaming Regex Example ===")

	// Example 1: Basic streaming match with callback
	fmt.Println("\n1. Basic streaming match:")
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

func basicStreamingExample() {
	input := "Log entries: 2024-01-15 event, 2024-02-20 another, 2024-12-31 final"
	fmt.Printf("  Input: %q\n", input)

	err := DatePattern{}.FindReader(strings.NewReader(input), stream.Config{},
		func(m stream.Match[*DatePatternBytesResult]) bool {
			fmt.Printf("  Found: %s at offset %d\n", m.Result.Match, m.StreamOffset)
			return true // continue
		})
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
	}
}

func countingExample() {
	input := "2024-01-01 2024-02-02 2024-03-03 2024-04-04 2024-05-05"
	fmt.Printf("  Input: %q\n", input)

	count, err := DatePattern{}.FindReaderCount(strings.NewReader(input), stream.Config{})
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
		return
	}
	fmt.Printf("  Found %d matches\n", count)
}

func firstMatchExample() {
	input := "Some text before 2024-07-15 and more text"
	fmt.Printf("  Input: %q\n", input)

	result, offset, err := DatePattern{}.FindReaderFirst(strings.NewReader(input), stream.Config{})
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
		return
	}
	if result != nil {
		fmt.Printf("  First match: %s at offset %d\n", result.Match, offset)
	} else {
		fmt.Println("  No match found")
	}
}

func earlyTerminationExample() {
	input := strings.Repeat("2024-01-01 ", 100) // Many dates
	fmt.Println("  Input: 100 dates (showing first 3)")

	var count int
	DatePattern{}.FindReader(strings.NewReader(input), stream.Config{},
		func(m stream.Match[*DatePatternBytesResult]) bool {
			count++
			fmt.Printf("  Match %d: %s\n", count, m.Result.Match)
			return count < 3 // Stop after 3 matches
		})
	fmt.Printf("  Stopped after %d matches\n", count)
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

func largeDataExample() {
	// Generate 1MB of data with dates every 50 bytes
	gen := NewPatternedReader("2024-01-15", 50, 1<<20)

	fmt.Printf("  Processing: 1MB of generated data\n")
	fmt.Printf("  Buffer size: %d bytes (constant memory)\n", stream.DefaultConfig().BufferSize)

	count, err := DatePattern{}.FindReaderCount(gen, stream.Config{})
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
		return
	}
	fmt.Printf("  Found %d matches in 1MB stream\n", count)
}
