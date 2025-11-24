package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KromDaniel/regengo/pkg/regengo"
)

// TestCase represents a test case with a pattern and inputs
type TestCase struct {
	Pattern string   `json:"pattern"`
	Inputs  []string `json:"inputs"`
}

// TestE2E runs end-to-end tests for regengo
func TestE2E(t *testing.T) {
	// Read test data
	testDataPath := filepath.Join("testdata.json")
	data, err := os.ReadFile(testDataPath)
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}

	var testCases []TestCase
	if err := json.Unmarshal(data, &testCases); err != nil {
		t.Fatalf("Failed to parse test data: %v", err)
	}

	if len(testCases) == 0 {
		t.Fatal("No test cases found in testdata.json")
	}

	t.Logf("Running %d e2e test cases", len(testCases))

	// Create a temporary directory for all test outputs
	// This directory will be automatically cleaned up after the test
	tempDir := t.TempDir()

	for i, tc := range testCases {
		tc := tc // capture range variable
		testName := fmt.Sprintf("Pattern%02d", i+1)

		t.Run(testName, func(t *testing.T) {
			// Create a subdirectory for this test case
			caseDir := filepath.Join(tempDir, testName)
			if err := os.MkdirAll(caseDir, 0755); err != nil {
				t.Fatalf("Failed to create test directory: %v", err)
			}

			// Step 1: Generate code using regengo
			t.Logf("Generating code for pattern: %s", tc.Pattern)
			outputFile := filepath.Join(caseDir, fmt.Sprintf("%s.go", testName))

			opts := regengo.Options{
				Pattern:          tc.Pattern,
				Name:             testName,
				OutputFile:       outputFile,
				Package:          "generated",
				GenerateTestFile: true,
				TestFileInputs:   tc.Inputs,
			}

			if err := regengo.Compile(opts); err != nil {
				t.Fatalf("Failed to generate code: %v", err)
			}

			// Verify the generated file exists
			if _, err := os.Stat(outputFile); os.IsNotExist(err) {
				t.Fatalf("Generated file does not exist: %s", outputFile)
			}

			// Verify the test file was generated
			testFile := filepath.Join(caseDir, fmt.Sprintf("%s_test.go", testName))
			if _, err := os.Stat(testFile); os.IsNotExist(err) {
				t.Fatalf("Generated test file does not exist: %s", testFile)
			}

			t.Logf("Generated files: %s and %s", outputFile, testFile)

			// Step 2: Initialize go module in the test directory
			t.Logf("Initializing go module...")
			initCmd := exec.Command("go", "mod", "init", "testmodule")
			initCmd.Dir = caseDir
			if output, err := initCmd.CombinedOutput(); err != nil {
				t.Fatalf("Failed to initialize go module:\nOutput: %s\nError: %v", string(output), err)
			}

			// Step 3: Run the generated tests
			t.Logf("Running generated tests...")
			cmd := exec.Command("go", "test", "-v")
			cmd.Dir = caseDir

			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Generated tests failed:\nOutput: %s\nError: %v", string(output), err)
			}

			// Count how many tests ran
			testCount := countTests(string(output))
			if testCount == 0 {
				t.Fatalf("No tests were executed! Output: %s", string(output))
			}

			t.Logf("✓ Ran %d generated tests", testCount)
		})
	}

	t.Logf("All %d e2e test cases passed successfully", len(testCases))
}

// countTests counts the number of tests that were executed by parsing the output
func countTests(output string) int {
	count := 0
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// Look for lines like "=== RUN   TestPattern01MatchString"
		if strings.HasPrefix(strings.TrimSpace(line), "=== RUN") {
			count++
		}
	}
	return count
}
