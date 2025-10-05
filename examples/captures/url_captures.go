package main

import (
	"fmt"
	"os"

	"github.com/KromDaniel/regengo/test"
)

func main() {
	fmt.Println("Testing mixed named and indexed capture groups:\n")

	testCases := []struct {
		input    string
		expected bool
		protocol string
		sub      string
		domain   string
		tld      string
	}{
		{"http://www.example.com", true, "http", "www", "example", "com"},
		{"https://api.github.io", true, "https", "api", "github", "io"},
		{"http://mail.google.org", true, "http", "mail", "google", "org"},
		{"ftp://test.foo.bar", false, "", "", "", ""},
		{"http://example.com", false, "", "", "", ""}, // Missing subdomain
	}

	passed := 0
	total := len(testCases)

	for _, tc := range testCases {
		result, found := test.URLCaptureFindString(tc.input)

		fmt.Printf("%-30s ", fmt.Sprintf(`"%s"`, tc.input))

		if found != tc.expected {
			fmt.Printf("found=%v (expected %v) ✗\n", found, tc.expected)
			continue
		}

		if !found {
			fmt.Printf("found=false ✓\n")
			passed++
			continue
		}

		// Check all fields
		if result.Protocol != tc.protocol {
			fmt.Printf("protocol=%q (expected %q) ✗\n", result.Protocol, tc.protocol)
			continue
		}
		if result.Match2 != tc.sub {
			fmt.Printf("Match2=%q (expected %q) ✗\n", result.Match2, tc.sub)
			continue
		}
		if result.Domain != tc.domain {
			fmt.Printf("domain=%q (expected %q) ✗\n", result.Domain, tc.domain)
			continue
		}
		if result.Match4 != tc.tld {
			fmt.Printf("Match4=%q (expected %q) ✗\n", result.Match4, tc.tld)
			continue
		}

		fmt.Printf("found=true protocol=%q Match2=%q domain=%q Match4=%q ✓\n",
			result.Protocol, result.Match2, result.Domain, result.Match4)
		passed++
	}

	fmt.Printf("\nResult: %d/%d tests passed\n", passed, total)
	if passed == total {
		fmt.Println("✓ All tests passed!")
	} else {
		fmt.Println("✗ Some tests failed")
		os.Exit(1)
	}
}
