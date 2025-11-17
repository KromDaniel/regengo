package main

import (
	"fmt"

	"github.com/KromDaniel/regengo/benchmarks/generated"
)

func main() {
	fmt.Println("=== FindAll Demo ===")
	fmt.Println()

	// Example 1: Find all dates
	fmt.Println("Example 1: Find all dates")
	text1 := "Important dates: 2024-01-15, 2024-12-25, and 2025-06-30"
	matches1 := generated.DateCaptureFindAllString(text1, -1)
	fmt.Printf("Input: %s\n", text1)
	fmt.Printf("Found %d dates:\n", len(matches1))
	for i, m := range matches1 {
		fmt.Printf("  %d. %s (Year=%s, Month=%s, Day=%s)\n", i+1, m.Match, m.Year, m.Month, m.Day)
	}
	fmt.Println()

	// Example 2: Limited matches
	fmt.Println("Example 2: Find up to 2 dates")
	text2 := "Dates: 2024-01-01, 2024-02-02, 2024-03-03, 2024-04-04"
	matches2 := generated.DateCaptureFindAllString(text2, 2)
	fmt.Printf("Input: %s\n", text2)
	fmt.Printf("Found %d dates (limited to 2):\n", len(matches2))
	for i, m := range matches2 {
		fmt.Printf("  %d. %s\n", i+1, m.Match)
	}
	fmt.Println()

	// Example 3: No matches
	fmt.Println("Example 3: No dates found")
	text3 := "No dates in this text"
	matches3 := generated.DateCaptureFindAllString(text3, -1)
	fmt.Printf("Input: %s\n", text3)
	fmt.Printf("Found %d dates\n", len(matches3))
	fmt.Println()

	// Example 4: n=0 returns nil
	fmt.Println("Example 4: n=0 returns nil immediately")
	text4 := "2024-01-15"
	matches4 := generated.DateCaptureFindAllString(text4, 0)
	fmt.Printf("Input: %s\n", text4)
	fmt.Printf("matches == nil: %v\n", matches4 == nil)
	fmt.Println()

	// Example 5: Find all emails
	fmt.Println("Example 5: Find all emails")
	text5 := "Contact: john@example.com, jane@test.org, and admin@company.net"
	matches5 := generated.EmailCaptureFindAllString(text5, -1)
	fmt.Printf("Input: %s\n", text5)
	fmt.Printf("Found %d emails:\n", len(matches5))
	for i, m := range matches5 {
		fmt.Printf("  %d. %s (User=%s, Domain=%s)\n", i+1, m.Match, m.User, m.Domain)
	}
	fmt.Println()

	// Example 6: Using bytes
	fmt.Println("Example 6: FindAllBytes with []byte input")
	text6 := []byte("Dates in bytes: 2024-01-15 and 2024-12-25")
	matches6 := generated.DateCaptureFindAllBytes(text6, -1)
	fmt.Printf("Input: %s\n", string(text6))
	fmt.Printf("Found %d dates:\n", len(matches6))
	for i, m := range matches6 {
		fmt.Printf("  %d. %s\n", i+1, m.Match)
	}
}
