package main

import (
	"fmt"
	"github.com/KromDaniel/regengo/test"
)

func main() {
	testCases := []struct {
		input    string
		expected bool
		user     string
		domain   string
		tld      string
	}{
		{"john@example.com", true, "john", "example", "com"},
		{"alice@test.org", true, "alice", "test", "org"},
		{"bob@company.co", true, "bob", "company", "co"},
		{"invalid", false, "", "", ""},
		{"@example.com", false, "", "", ""},
		{"test@", false, "", "", ""},
	}

	fmt.Println("Testing EmailCapture with named groups:")
	fmt.Println(string(make([]byte, 70)))

	passCount := 0
	for _, tc := range testCases {
		result, found := test.EmailCaptureFindString(tc.input)
		
		status := "✗"
		if found == tc.expected {
			if !found {
				// Both expect no match
				status = "✓"
				passCount++
			} else if result.User == tc.user && result.Domain == tc.domain && result.Tld == tc.tld {
				// Match found with correct captures
				status = "✓"
				passCount++
			}
		}
		
		fmt.Printf("%s %-25s found=%v", status, fmt.Sprintf("%q", tc.input), found)
		if found && result != nil {
			fmt.Printf(" user=%q domain=%q tld=%q", result.User, result.Domain, result.Tld)
		}
		fmt.Println()
	}

	fmt.Println(string(make([]byte, 70)))
	fmt.Printf("Result: %d/%d tests passed\n", passCount, len(testCases))
	
	if passCount == len(testCases) {
		fmt.Println("✓ All tests passed!")
	} else {
		fmt.Printf("✗ %d tests failed!\n", len(testCases)-passCount)
	}
}
