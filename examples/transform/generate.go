//go:build ignore

// Run this to generate the email pattern:
//
//	go run generate.go
package main

import (
	"log"

	"github.com/KromDaniel/regengo"
)

func main() {
	err := regengo.Compile(regengo.Options{
		Pattern:          `(?P<user>[a-zA-Z0-9._%+-]+)@(?P<domain>[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`,
		Name:             "Email",
		OutputFile:       "email.go",
		Package:          "main",
		GenerateTestFile: false,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Generated email.go")
}
