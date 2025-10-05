package playground
package main

import (
	"fmt"
	"os"

	"github.com/KromDaniel/regengo/pkg/regengo"
)

func main() {
	// Example: Generate a date matcher
	// Modify the pattern, name, and test it!
	
	pattern := `(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})`
	name := "DateCapture"
	
	fmt.Printf("🚀 Generating matcher for pattern: %s\n\n", pattern)
	
	err := regengo.Compile(regengo.Options{
		Pattern:    pattern,
		Name:       name,
		OutputFile: "./playground_output.go",
		Package:    "main",
	})
	
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println("✓ Generated matcher successfully!")
	fmt.Println("\nGenerated functions:")
	fmt.Printf("  - %sMatchString(input string) bool\n", name)
	fmt.Printf("  - %sMatchBytes(input []byte) bool\n", name)
	fmt.Printf("  - %sFindString(input string) (*%sMatch, bool)\n", name, name)
	fmt.Printf("  - %sFindBytes(input []byte) (*%sMatch, bool)\n", name, name)
	fmt.Printf("  - %sFindAllString(input string, n int) []*%sMatch\n", name, name)
	fmt.Printf("  - %sFindAllBytes(input []byte, n int) []*%sMatch\n", name, name)
	
	fmt.Println("\n📝 Output saved to: playground_output.go")
	fmt.Println("\n🎯 Next steps:")
	fmt.Println("  1. Review the generated code in playground_output.go")
	fmt.Println("  2. Copy the functions you need to your project")
	fmt.Println("  3. Run benchmarks: go test -bench=. -benchmem")
	fmt.Println("\n💡 Try modifying the pattern variable above and run again!")
	fmt.Println("\nExamples:")
	fmt.Println("  Email:  `(?P<user>[\\w\\.+-]+)@(?P<domain>[\\w\\.-]+)\\.(?P<tld>[\\w\\.-]+)`")
	fmt.Println("  URL:    `(?P<protocol>https?)://(?P<host>[\\w\\.-]+)(?::(?P<port>\\d+))?`")
	fmt.Println("  Phone:  `(?P<area>\\d{3})-(?P<prefix>\\d{3})-(?P<line>\\d{4})`")
}
