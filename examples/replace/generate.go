//go:build ignore

package main

import (
	"log"

	"github.com/KromDaniel/regengo"
)

func main() {
	// Generate email pattern with pre-compiled replacers
	err := regengo.Compile(regengo.Options{
		Pattern:    `(?P<user>[\w.+-]+)@(?P<domain>[\w.-]+)\.(?P<tld>\w+)`,
		Name:       "Email",
		OutputFile: "email.go",
		Package:    "main",
		Replacers: []string{
			"$user@REDACTED.$tld", // ReplaceAllString0: mask domain
			"[EMAIL REMOVED]",     // ReplaceAllString1: full redaction
			"$user@***.$tld",      // ReplaceAllString2: partial mask
		},
	})
	if err != nil {
		log.Fatalf("Failed to generate email pattern: %v", err)
	}

	log.Println("Generated email.go with Replace methods")
}
