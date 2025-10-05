package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/KromDaniel/regengo/pkg/regengo"
)

func main() {
	examples := []struct {
		name    string
		pattern string
	}{
		{
			name:    "Email",
			pattern: `[\w\.+-]+@[\w\.-]+\.[\w\.-]+`,
		},
		{
			name:    "URL",
			pattern: `https?://[^\s]+`,
		},
		{
			name:    "IPv4",
			pattern: `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`,
		},
	}

	outputDir := "examples/generated"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		panic(err)
	}

	for _, ex := range examples {
		fmt.Printf("Generating %s matcher...\n", ex.name)

		err := regengo.Compile(regengo.Options{
			Pattern:    ex.pattern,
			Name:       ex.name,
			OutputFile: filepath.Join(outputDir, fmt.Sprintf("%s.go", ex.name)),
			Package:    "generated",
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating %s: %v\n", ex.name, err)
			os.Exit(1)
		}
	}

	fmt.Println("All examples generated successfully!")
}
