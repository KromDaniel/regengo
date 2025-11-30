//go:build ignore

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	// Get current working directory (should be project root)
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	outputDir := filepath.Join(cwd, "benchmarks", "curated")

	// Clean and recreate output directory
	if err := os.RemoveAll(outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error cleaning output directory: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Track categories for summary
	categories := make(map[string]int)

	for _, c := range CuratedCases {
		fmt.Printf("Generating %s (%s)...\n", c.Name, c.Category)
		categories[c.Category]++

		outputFile := filepath.Join(outputDir, c.Name+".go")

		// Build regengo command arguments
		args := []string{
			"run", "./cmd/regengo",
			"-pattern", c.Pattern,
			"-name", c.Name,
			"-output", outputFile,
			"-package", "curated",
			"-test-inputs", strings.Join(c.Inputs, ","),
		}

		// Add replacers if present
		for _, r := range c.Replacers {
			args = append(args, "-replacer", r)
		}

		// Add force-tnfa flag if needed
		if c.ForceTNFA {
			args = append(args, "-force-tnfa")
		}

		// Run regengo CLI
		cmd := exec.Command("go", args...)
		cmd.Dir = cwd
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating %s: %v\n", c.Name, err)
			os.Exit(1)
		}
	}

	// Print summary
	fmt.Println("\n=== Generation Summary ===")
	total := 0
	for cat, count := range categories {
		fmt.Printf("  %s: %d patterns\n", cat, count)
		total += count
	}
	fmt.Printf("\nTotal: %d curated benchmark patterns generated\n", total)
	fmt.Printf("Output directory: %s\n", outputDir)
}
