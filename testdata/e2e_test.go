package testdata

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/KromDaniel/regengo/pkg/regengo"
)

// TestCase represents a test case with a pattern, inputs, and labels
type TestCase struct {
	Pattern       string   `json:"pattern"`
	Inputs        []string `json:"inputs"`
	FeatureLabels []string `json:"feature_labels,omitempty"`
	EngineLabels  []string `json:"engine_labels,omitempty"`
}

// TestE2E runs end-to-end tests for regengo
// Use -run flag to filter by labels, e.g.:
//
//	go test ./testdata/... -run "Multibyte"
//	go test ./testdata/... -run "TDFA"
//	go test ./testdata/... -run "Captures.*WordBoundary"
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
	tempDir := t.TempDir()

	for i, tc := range testCases {
		tc := tc // capture range variable
		idx := i + 1

		// Build test name from labels
		testName := buildTestName(tc, idx)

		t.Run(testName, func(t *testing.T) {
			t.Parallel() // Run subtests concurrently (limited by GOMAXPROCS)

			// Step 0: Verify engine labels (regression detection)
			// Every pattern must have engine labels defined
			result, err := regengo.Analyze(tc.Pattern)
			if err != nil {
				t.Fatalf("Failed to analyze pattern: %v", err)
			}

			// Engine labels must be present in testdata.json
			if len(tc.EngineLabels) == 0 {
				t.Fatalf("Missing engine_labels in testdata.json:\n"+
					"  Pattern: %s\n"+
					"  Actual engine labels: %v\n"+
					"Run './scripts/manage_e2e_test.py -p %q' to add labels",
					tc.Pattern, result.EngineLabels, tc.Pattern)
			}

			// Compare expected vs actual engine labels
			actualEngineLabels := result.EngineLabels
			sort.Strings(actualEngineLabels)
			expectedEngineLabels := make([]string, len(tc.EngineLabels))
			copy(expectedEngineLabels, tc.EngineLabels)
			sort.Strings(expectedEngineLabels)

			if !reflect.DeepEqual(actualEngineLabels, expectedEngineLabels) {
				t.Fatalf("Engine label mismatch (possible regression):\n"+
					"  Pattern: %s\n"+
					"  Expected: %v\n"+
					"  Actual: %v\n"+
					"Run './scripts/manage_e2e_test.py -p %q' to update if intentional",
					tc.Pattern, expectedEngineLabels, actualEngineLabels, tc.Pattern)
			}

			// Create unique name for generated code
			uniqueName := fmt.Sprintf("Pattern%03d", idx)

			// Create a subdirectory for this test case
			caseDir := filepath.Join(tempDir, uniqueName)
			if err := os.MkdirAll(caseDir, 0755); err != nil {
				t.Fatalf("Failed to create test directory: %v", err)
			}

			// Step 1: Generate code using regengo
			t.Logf("Generating code for pattern: %s", tc.Pattern)
			outputFile := filepath.Join(caseDir, fmt.Sprintf("%s.go", uniqueName))

			opts := regengo.Options{
				Pattern:          tc.Pattern,
				Name:             uniqueName,
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
			testFile := filepath.Join(caseDir, fmt.Sprintf("%s_test.go", uniqueName))
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

			// Add replace directive for the stream package (used by streaming tests)
			// Get the absolute path to the regengo module
			regengoPath, err := getRegengoModulePath()
			if err != nil {
				t.Fatalf("Failed to get regengo module path: %v", err)
			}
			editCmd := exec.Command("go", "mod", "edit", "-replace",
				fmt.Sprintf("github.com/KromDaniel/regengo=%s", regengoPath))
			editCmd.Dir = caseDir
			if output, err := editCmd.CombinedOutput(); err != nil {
				t.Fatalf("Failed to add replace directive:\nOutput: %s\nError: %v", string(output), err)
			}

			// Run go mod tidy to resolve dependencies
			tidyCmd := exec.Command("go", "mod", "tidy")
			tidyCmd.Dir = caseDir
			if output, err := tidyCmd.CombinedOutput(); err != nil {
				t.Fatalf("Failed to tidy go module:\nOutput: %s\nError: %v", string(output), err)
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

			t.Logf("Ran %d generated tests", testCount)
		})
	}

	t.Logf("All %d e2e test cases passed successfully", len(testCases))
}

// buildTestName creates a test name from labels and index
// Format: Labels_Joined/Pattern001 (e.g., "Captures_Multibyte_TDFA/Pattern001")
func buildTestName(tc TestCase, idx int) string {
	// Combine feature and engine labels
	var allLabels []string
	allLabels = append(allLabels, tc.FeatureLabels...)
	allLabels = append(allLabels, tc.EngineLabels...)

	// Sort for consistent ordering
	sort.Strings(allLabels)

	// Build label string
	labelStr := "NoLabels"
	if len(allLabels) > 0 {
		labelStr = strings.Join(allLabels, "_")
	}

	return fmt.Sprintf("%s/Pattern%03d", labelStr, idx)
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

// getRegengoModulePath returns the absolute path to the regengo module root.
// This is needed for the replace directive so that generated tests can import
// the stream package used by streaming tests.
func getRegengoModulePath() (string, error) {
	// The tests are in the testdata directory, so go up one level
	// Use runtime.Caller to get the current file's directory
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to get current file path")
	}
	dir := filepath.Dir(file)
	return filepath.Abs(filepath.Join(dir, ".."))
}
