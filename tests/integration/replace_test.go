package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/KromDaniel/regengo/pkg/regengo"
)

func TestRuntimeReplace(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "email.go")

	// Generate a pattern with named captures
	opts := regengo.Options{
		Pattern:    `(?P<user>\w+)@(?P<domain>\w+)\.(?P<tld>\w+)`,
		Name:       "Email",
		OutputFile: outputFile,
		Package:    "main",
	}

	if err := regengo.Compile(opts); err != nil {
		t.Fatalf("compilation failed: %v", err)
	}

	// Create a test program that uses the generated Replace methods
	testProgram := `package main

import (
	"fmt"
)

func main() {
	// Test ReplaceAllString
	input := "Contact alice@example.com and bob@test.org"

	// Test literal replacement
	result := CompiledEmail.ReplaceAllString(input, "REDACTED")
	fmt.Println("Literal:", result)

	// Test full match
	result = CompiledEmail.ReplaceAllString(input, "[$0]")
	fmt.Println("FullMatch:", result)

	// Test indexed captures
	result = CompiledEmail.ReplaceAllString(input, "$1@REDACTED.$3")
	fmt.Println("Indexed:", result)

	// Test named captures
	result = CompiledEmail.ReplaceAllString(input, "$user@hidden.$tld")
	fmt.Println("Named:", result)

	// Test escaped dollar
	result = CompiledEmail.ReplaceAllString(input, "$$user=$user")
	fmt.Println("Escaped:", result)

	// Test ReplaceFirstString
	result = CompiledEmail.ReplaceFirstString(input, "[FIRST]")
	fmt.Println("First:", result)

	// Test no match
	result = CompiledEmail.ReplaceAllString("no emails here", "[$0]")
	fmt.Println("NoMatch:", result)

	// Test ReplaceAllBytes
	inputBytes := []byte("test@example.com")
	resultBytes := CompiledEmail.ReplaceAllBytes(inputBytes, "[$user]")
	fmt.Println("Bytes:", string(resultBytes))

	// Test ReplaceAllBytesAppend
	buf := make([]byte, 0, 100)
	resultBytes = CompiledEmail.ReplaceAllBytesAppend(inputBytes, "[$user]", buf)
	fmt.Println("BytesAppend:", string(resultBytes))
}
`

	testFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(testFile, []byte(testProgram), 0644); err != nil {
		t.Fatalf("failed to write test program: %v", err)
	}

	// Get the project root directory
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	// Working dir is tests/integration, so go up two levels
	projectRoot := filepath.Join(wd, "..", "..")

	// Create go.mod
	goMod := `module test

go 1.21

require github.com/KromDaniel/regengo v0.0.0

replace github.com/KromDaniel/regengo => ` + projectRoot + `
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Run go mod tidy and then the test
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = tmpDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %v\n%s", err, output)
	}

	cmd = exec.Command("go", "run", ".")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("test program failed: %v\n%s", err, output)
	}

	outputStr := string(output)

	// Verify results
	expected := []struct {
		prefix string
		want   string
	}{
		{"Literal:", "Literal: Contact REDACTED and REDACTED"},
		{"FullMatch:", "FullMatch: Contact [alice@example.com] and [bob@test.org]"},
		{"Indexed:", "Indexed: Contact alice@REDACTED.com and bob@REDACTED.org"},
		{"Named:", "Named: Contact alice@hidden.com and bob@hidden.org"},
		{"Escaped:", "Escaped: Contact $user=alice and $user=bob"},
		{"First:", "First: Contact [FIRST] and bob@test.org"},
		{"NoMatch:", "NoMatch: no emails here"},
		{"Bytes:", "Bytes: [test]"},
		{"BytesAppend:", "BytesAppend: [test]"},
	}

	for _, exp := range expected {
		if !containsLine(outputStr, exp.want) {
			t.Errorf("expected output to contain %q, got:\n%s", exp.want, outputStr)
		}
	}
}

func containsLine(output, want string) bool {
	for _, line := range splitLines(output) {
		if line == want {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func TestRuntimeReplaceEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "digit.go")

	// Generate a simple pattern without captures (only $0 available)
	opts := regengo.Options{
		Pattern:    `\d+`,
		Name:       "Digit",
		OutputFile: outputFile,
		Package:    "main",
	}

	if err := regengo.Compile(opts); err != nil {
		t.Fatalf("compilation failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("output file was not created")
	}
}
