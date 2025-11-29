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

func TestReplaceAppendZeroAlloc(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "email.go")

	// Generate a pattern with pre-compiled replacers
	opts := regengo.Options{
		Pattern:    `(?P<user>\w+)@(?P<domain>\w+)\.(?P<tld>\w+)`,
		Name:       "Email",
		OutputFile: outputFile,
		Package:    "main",
		Replacers:  []string{"$user@REDACTED.$tld"},
	}

	if err := regengo.Compile(opts); err != nil {
		t.Fatalf("compilation failed: %v", err)
	}

	// Get project root
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	projectRoot := filepath.Join(wd, "..", "..")

	// Create a test program that verifies zero-allocation behavior
	testProgram := `package main

import (
	"fmt"
)

func main() {
	input := []byte("test@example.com and admin@site.org")

	// Create a buffer with a marker byte to verify reuse
	buf := make([]byte, 1, 1024)
	buf[0] = 0xFF // Marker byte
	origCap := cap(buf)

	// Perform replacement - pass buf[:0] to reset length
	result := CompiledEmail.ReplaceAllBytesAppend0(input, buf[:0])

	// Verify same backing array by checking:
	// 1. The result capacity should match original (uses same backing array)
	// 2. The original buffer should show the written data (backing array modified)
	sameBackingArray := cap(result) == origCap && buf[0] == result[0]
	if sameBackingArray {
		fmt.Println("SameArray: true")
	} else {
		fmt.Printf("SameArray: false (cap result=%d, orig=%d)\n", cap(result), origCap)
	}

	// Verify correct output
	expected := "test@REDACTED.com and admin@REDACTED.org"
	if string(result) == expected {
		fmt.Println("Output: correct")
	} else {
		fmt.Printf("Output: wrong, got %q\n", string(result))
	}

	// Test with nil buffer (should still work, just allocate)
	resultNil := CompiledEmail.ReplaceAllBytesAppend0(input, nil)
	if string(resultNil) == expected {
		fmt.Println("NilBuf: correct")
	} else {
		fmt.Printf("NilBuf: wrong, got %q\n", string(resultNil))
	}

	// Test buffer reuse in loop (simulating zero-alloc pattern)
	buf2 := make([]byte, 0, 1024)
	for i := 0; i < 3; i++ {
		buf2 = CompiledEmail.ReplaceAllBytesAppend0(input, buf2[:0])
	}
	if string(buf2) == expected {
		fmt.Println("Loop: correct")
	} else {
		fmt.Printf("Loop: wrong, got %q\n", string(buf2))
	}

	// Test runtime version too - verify it uses the same backing array
	buf3 := make([]byte, 1, 1024)
	buf3[0] = 0xFE
	runtimeResult := CompiledEmail.ReplaceAllBytesAppend(input, "$user@REDACTED.$tld", buf3[:0])
	sameBackingArrayRuntime := cap(runtimeResult) == cap(buf3) && buf3[0] == runtimeResult[0]
	if sameBackingArrayRuntime {
		fmt.Println("RuntimeSameArray: true")
	} else {
		fmt.Println("RuntimeSameArray: false")
	}
}
`

	testFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(testFile, []byte(testProgram), 0644); err != nil {
		t.Fatalf("failed to write test program: %v", err)
	}

	// Create go.mod
	goMod := `module test

go 1.21

require github.com/KromDaniel/regengo v0.0.0

replace github.com/KromDaniel/regengo => ` + projectRoot + `
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Run go mod tidy
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = tmpDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %v\n%s", err, output)
	}

	// Run the test program
	cmd = exec.Command("go", "run", ".")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("test program failed: %v\n%s", err, output)
	}

	outputStr := string(output)

	// Verify results
	zeroAllocExpected := []struct {
		prefix string
		want   string
	}{
		{"SameArray:", "SameArray: true"},
		{"Output:", "Output: correct"},
		{"NilBuf:", "NilBuf: correct"},
		{"Loop:", "Loop: correct"},
		{"RuntimeSameArray:", "RuntimeSameArray: true"},
	}

	for _, exp := range zeroAllocExpected {
		if !containsLine(outputStr, exp.want) {
			t.Errorf("expected output to contain %q, got:\n%s", exp.want, outputStr)
		}
	}
}

func TestSpecialCaseOptimizations(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "pattern.go")

	// Generate a pattern with special case replacers:
	// - Literal-only template (FILTERED)
	// - Full-match-only template ([$0])
	// - Regular template with captures ($user@REDACTED.$tld)
	opts := regengo.Options{
		Pattern:    `(?P<user>\w+)@(?P<domain>\w+)\.(?P<tld>\w+)`,
		Name:       "Pattern",
		OutputFile: outputFile,
		Package:    "main",
		Replacers:  []string{"FILTERED", "[$0]", "$user@REDACTED.$tld"},
	}

	if err := regengo.Compile(opts); err != nil {
		t.Fatalf("compilation failed: %v", err)
	}

	// Read generated file and verify optimization comments
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}
	contentStr := string(content)

	// Check for optimization comments in generated code
	if !containsString(contentStr, "Optimized: literal-only template") {
		t.Error("expected literal-only optimization comment in generated code")
	}
	if !containsString(contentStr, "Optimized: uses only full match") {
		t.Error("expected full-match-only optimization comment in generated code")
	}

	// Get project root
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	projectRoot := filepath.Join(wd, "..", "..")

	// Create a test program that verifies optimized replacers work correctly
	testProgram := `package main

import (
	"fmt"
)

func main() {
	input := "Contact alice@example.com and bob@test.org"

	// Test literal-only replacer (optimized)
	result0 := CompiledPattern.ReplaceAllString0(input)
	if result0 == "Contact FILTERED and FILTERED" {
		fmt.Println("LiteralOnly: correct")
	} else {
		fmt.Printf("LiteralOnly: wrong, got %q\n", result0)
	}

	// Test full-match-only replacer (optimized)
	result1 := CompiledPattern.ReplaceAllString1(input)
	if result1 == "Contact [alice@example.com] and [bob@test.org]" {
		fmt.Println("FullMatchOnly: correct")
	} else {
		fmt.Printf("FullMatchOnly: wrong, got %q\n", result1)
	}

	// Test regular replacer with captures
	result2 := CompiledPattern.ReplaceAllString2(input)
	if result2 == "Contact alice@REDACTED.com and bob@REDACTED.org" {
		fmt.Println("WithCaptures: correct")
	} else {
		fmt.Printf("WithCaptures: wrong, got %q\n", result2)
	}

	// Test bytes variants too
	inputBytes := []byte("test@example.com")

	resultBytes0 := CompiledPattern.ReplaceAllBytes0(inputBytes)
	if string(resultBytes0) == "FILTERED" {
		fmt.Println("BytesLiteralOnly: correct")
	} else {
		fmt.Printf("BytesLiteralOnly: wrong, got %q\n", string(resultBytes0))
	}

	resultBytes1 := CompiledPattern.ReplaceAllBytes1(inputBytes)
	if string(resultBytes1) == "[test@example.com]" {
		fmt.Println("BytesFullMatchOnly: correct")
	} else {
		fmt.Printf("BytesFullMatchOnly: wrong, got %q\n", string(resultBytes1))
	}
}
`

	testFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(testFile, []byte(testProgram), 0644); err != nil {
		t.Fatalf("failed to write test program: %v", err)
	}

	// Create go.mod
	goMod := `module test

go 1.21

require github.com/KromDaniel/regengo v0.0.0

replace github.com/KromDaniel/regengo => ` + projectRoot + `
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Run go mod tidy
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = tmpDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %v\n%s", err, output)
	}

	// Run the test program
	cmd = exec.Command("go", "run", ".")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("test program failed: %v\n%s", err, output)
	}

	outputStr := string(output)

	// Verify results
	expectedResults := []struct {
		prefix string
		want   string
	}{
		{"LiteralOnly:", "LiteralOnly: correct"},
		{"FullMatchOnly:", "FullMatchOnly: correct"},
		{"WithCaptures:", "WithCaptures: correct"},
		{"BytesLiteralOnly:", "BytesLiteralOnly: correct"},
		{"BytesFullMatchOnly:", "BytesFullMatchOnly: correct"},
	}

	for _, exp := range expectedResults {
		if !containsLine(outputStr, exp.want) {
			t.Errorf("expected output to contain %q, got:\n%s", exp.want, outputStr)
		}
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
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

func TestReplaceEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "email.go")

	// Generate a pattern with named captures
	opts := regengo.Options{
		Pattern:    `(?P<user>\w+)@(?P<domain>\w+)\.(?P<tld>\w+)`,
		Name:       "Email",
		OutputFile: outputFile,
		Package:    "main",
		Replacers:  []string{"$user@REDACTED.$tld"},
	}

	if err := regengo.Compile(opts); err != nil {
		t.Fatalf("compilation failed: %v", err)
	}

	// Get project root
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	projectRoot := filepath.Join(wd, "..", "..")

	// Create a test program that tests edge cases
	testProgram := `package main

import (
	"fmt"
	"strings"
)

func main() {
	// Edge case: empty input
	result := CompiledEmail.ReplaceAllString("", "$0")
	if result == "" {
		fmt.Println("EmptyInput: correct")
	} else {
		fmt.Printf("EmptyInput: wrong, got %q\n", result)
	}

	// Edge case: no match
	result = CompiledEmail.ReplaceAllString("no emails here", "[$0]")
	if result == "no emails here" {
		fmt.Println("NoMatch: correct")
	} else {
		fmt.Printf("NoMatch: wrong, got %q\n", result)
	}

	// Edge case: match at start
	result = CompiledEmail.ReplaceAllString("a@b.c rest of text", "X")
	if result == "X rest of text" {
		fmt.Println("MatchAtStart: correct")
	} else {
		fmt.Printf("MatchAtStart: wrong, got %q\n", result)
	}

	// Edge case: match at end
	result = CompiledEmail.ReplaceAllString("rest of text a@b.c", "X")
	if result == "rest of text X" {
		fmt.Println("MatchAtEnd: correct")
	} else {
		fmt.Printf("MatchAtEnd: wrong, got %q\n", result)
	}

	// Edge case: adjacent matches
	result = CompiledEmail.ReplaceAllString("a@b.c d@e.f", "X")
	if result == "X X" {
		fmt.Println("AdjacentMatches: correct")
	} else {
		fmt.Printf("AdjacentMatches: wrong, got %q\n", result)
	}

	// Edge case: only match (entire input is match)
	result = CompiledEmail.ReplaceAllString("a@b.c", "X")
	if result == "X" {
		fmt.Println("OnlyMatch: correct")
	} else {
		fmt.Printf("OnlyMatch: wrong, got %q\n", result)
	}

	// Edge case: large input with many matches
	largeInput := strings.Repeat("user@example.com ", 1000)
	largeExpected := strings.Repeat("X ", 1000)
	result = CompiledEmail.ReplaceAllString(largeInput, "X")
	if result == largeExpected {
		fmt.Println("LargeInput: correct")
	} else {
		fmt.Println("LargeInput: wrong")
	}

	// Edge case: ReplaceFirst with no match
	result = CompiledEmail.ReplaceFirstString("no emails", "[FIRST]")
	if result == "no emails" {
		fmt.Println("FirstNoMatch: correct")
	} else {
		fmt.Printf("FirstNoMatch: wrong, got %q\n", result)
	}

	// Edge case: ReplaceFirst with single match
	result = CompiledEmail.ReplaceFirstString("one@email.com", "[FIRST]")
	if result == "[FIRST]" {
		fmt.Println("FirstSingleMatch: correct")
	} else {
		fmt.Printf("FirstSingleMatch: wrong, got %q\n", result)
	}

	// Edge case: ReplaceFirst with multiple matches
	result = CompiledEmail.ReplaceFirstString("a@b.c and d@e.f", "[FIRST]")
	if result == "[FIRST] and d@e.f" {
		fmt.Println("FirstMultipleMatches: correct")
	} else {
		fmt.Printf("FirstMultipleMatches: wrong, got %q\n", result)
	}

	// Edge case: pre-compiled with empty input
	result = CompiledEmail.ReplaceAllString0("")
	if result == "" {
		fmt.Println("PrecompiledEmpty: correct")
	} else {
		fmt.Printf("PrecompiledEmpty: wrong, got %q\n", result)
	}

	// Edge case: bytes with empty input
	resultBytes := CompiledEmail.ReplaceAllBytes([]byte{}, "$0")
	if len(resultBytes) == 0 {
		fmt.Println("BytesEmpty: correct")
	} else {
		fmt.Printf("BytesEmpty: wrong, got %q\n", string(resultBytes))
	}
}
`

	testFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(testFile, []byte(testProgram), 0644); err != nil {
		t.Fatalf("failed to write test program: %v", err)
	}

	// Create go.mod
	goMod := `module test

go 1.21

require github.com/KromDaniel/regengo v0.0.0

replace github.com/KromDaniel/regengo => ` + projectRoot + `
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Run go mod tidy
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = tmpDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %v\n%s", err, output)
	}

	// Run the test program
	cmd = exec.Command("go", "run", ".")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("test program failed: %v\n%s", err, output)
	}

	outputStr := string(output)

	// Verify results
	expectedResults := []struct {
		prefix string
		want   string
	}{
		{"EmptyInput:", "EmptyInput: correct"},
		{"NoMatch:", "NoMatch: correct"},
		{"MatchAtStart:", "MatchAtStart: correct"},
		{"MatchAtEnd:", "MatchAtEnd: correct"},
		{"AdjacentMatches:", "AdjacentMatches: correct"},
		{"OnlyMatch:", "OnlyMatch: correct"},
		{"LargeInput:", "LargeInput: correct"},
		{"FirstNoMatch:", "FirstNoMatch: correct"},
		{"FirstSingleMatch:", "FirstSingleMatch: correct"},
		{"FirstMultipleMatches:", "FirstMultipleMatches: correct"},
		{"PrecompiledEmpty:", "PrecompiledEmpty: correct"},
		{"BytesEmpty:", "BytesEmpty: correct"},
	}

	for _, exp := range expectedResults {
		if !containsLine(outputStr, exp.want) {
			t.Errorf("expected output to contain %q, got:\n%s", exp.want, outputStr)
		}
	}
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

func TestPrecompiledReplace(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "email.go")

	// Generate a pattern with pre-compiled replacers
	opts := regengo.Options{
		Pattern:    `(?P<user>\w+)@(?P<domain>\w+)\.(?P<tld>\w+)`,
		Name:       "Email",
		OutputFile: outputFile,
		Package:    "main",
		Replacers:  []string{"$user@REDACTED.$tld", "[$0]", "FILTERED"},
	}

	if err := regengo.Compile(opts); err != nil {
		t.Fatalf("compilation failed: %v", err)
	}

	// Get project root
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	projectRoot := filepath.Join(wd, "..", "..")

	// Create a test program that uses the pre-compiled Replace methods
	testProgram := `package main

import (
	"fmt"
)

func main() {
	input := "Contact alice@example.com and bob@test.org"

	// Test pre-compiled replacer 0: $user@REDACTED.$tld
	result0 := CompiledEmail.ReplaceAllString0(input)
	fmt.Println("Replacer0:", result0)

	// Test pre-compiled replacer 1: [$0]
	result1 := CompiledEmail.ReplaceAllString1(input)
	fmt.Println("Replacer1:", result1)

	// Test pre-compiled replacer 2: FILTERED
	result2 := CompiledEmail.ReplaceAllString2(input)
	fmt.Println("Replacer2:", result2)

	// Test ReplaceFirst variants
	resultFirst0 := CompiledEmail.ReplaceFirstString0(input)
	fmt.Println("First0:", resultFirst0)

	// Test bytes variant
	inputBytes := []byte("test@example.com")
	resultBytes := CompiledEmail.ReplaceAllBytes0(inputBytes)
	fmt.Println("Bytes0:", string(resultBytes))

	// Test BytesAppend variant
	buf := make([]byte, 0, 100)
	resultAppend := CompiledEmail.ReplaceAllBytesAppend0(inputBytes, buf)
	fmt.Println("Append0:", string(resultAppend))

	// Verify pre-compiled matches runtime for same template
	runtime := CompiledEmail.ReplaceAllString(input, "$user@REDACTED.$tld")
	precompiled := CompiledEmail.ReplaceAllString0(input)
	if runtime == precompiled {
		fmt.Println("Match: runtime equals precompiled")
	} else {
		fmt.Printf("Mismatch: runtime=%q precompiled=%q\n", runtime, precompiled)
	}
}
`

	testFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(testFile, []byte(testProgram), 0644); err != nil {
		t.Fatalf("failed to write test program: %v", err)
	}

	// Create go.mod
	goMod := `module test

go 1.21

require github.com/KromDaniel/regengo v0.0.0

replace github.com/KromDaniel/regengo => ` + projectRoot + `
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Run go mod tidy
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = tmpDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %v\n%s", err, output)
	}

	// Run the test program
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
		{"Replacer0:", "Replacer0: Contact alice@REDACTED.com and bob@REDACTED.org"},
		{"Replacer1:", "Replacer1: Contact [alice@example.com] and [bob@test.org]"},
		{"Replacer2:", "Replacer2: Contact FILTERED and FILTERED"},
		{"First0:", "First0: Contact alice@REDACTED.com and bob@test.org"},
		{"Bytes0:", "Bytes0: test@REDACTED.com"},
		{"Append0:", "Append0: test@REDACTED.com"},
		{"Match:", "Match: runtime equals precompiled"},
	}

	for _, exp := range expected {
		if !containsLine(outputStr, exp.want) {
			t.Errorf("expected output to contain %q, got:\n%s", exp.want, outputStr)
		}
	}
}
