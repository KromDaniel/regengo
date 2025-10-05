package main

import (
"fmt"
)

func main() {
	// Test generated email matcher
	testInputs := []string{
		"user@example.com",
		"test@test.org", 
		"invalid",
		"another@domain.co",
	}
	
	fmt.Println("Testing Email2 matcher (with named captures):")
	for _, input := range testInputs {
		match := Email2MatchString(input)
		fmt.Printf("  %q: match=%v", input, match)
		
		if match {
			result, ok := Email2FindString(input)
			if ok {
				fmt.Printf(" -> User=%q, Domain=%q", result.User, result.Domain)
			}
		}
		fmt.Println()
	}
}
