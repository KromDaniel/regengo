package main

import (
"fmt"
"regexp"
)

func main() {
	re := regexp.MustCompile(`([a-z]+)@([a-z]+\.[a-z]+)`)
	
	inputs := []string{"user@example.com", "test@test.org", "invalid"}
	for _, input := range inputs {
		fmt.Printf("%q: MatchString=%v, FindString=%q\n", input, re.MatchString(input), re.FindString(input))
	}
}
