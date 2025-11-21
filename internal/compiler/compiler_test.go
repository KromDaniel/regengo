package compiler

import (
	"os"
	"path/filepath"
	"regexp/syntax"
	"testing"
)

func TestCompilerGenerate(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
	}{
		{"simple", "test"},
		{"digit", `\d+`},
		{"word", `\w+`},
		{"alternation", "a|b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse pattern
			regexAST, err := syntax.Parse(tt.pattern, syntax.Perl)
			if err != nil {
				t.Fatalf("failed to parse pattern: %v", err)
			}

			regexAST = regexAST.Simplify()
			prog, err := syntax.Compile(regexAST)
			if err != nil {
				t.Fatalf("failed to compile pattern: %v", err)
			}

			// Create compiler
			tmpDir := t.TempDir()
			outputFile := filepath.Join(tmpDir, "test.go")

			c := New(Config{
				Pattern:    tt.pattern,
				Name:       "Test",
				OutputFile: outputFile,
				Package:    "test",
				Program:    prog,
			})

			// Generate code
			if err := c.Generate(); err != nil {
				t.Errorf("generation failed: %v", err)
			}

			// Verify file was created
			if _, err := os.Stat(outputFile); os.IsNotExist(err) {
				t.Error("output file was not created")
			}
		})
	}
}

func TestInstructionGeneration(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		wantErr bool
	}{
		{"InstRune1", "a", false},
		{"InstRune", "[a-z]", false},
		{"InstRuneAnyNotNL", ".", false},
		{"InstAlt", "a|b", false},
		{"InstMatch", "$", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regexAST, err := syntax.Parse(tt.pattern, syntax.Perl)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			regexAST = regexAST.Simplify()
			prog, err := syntax.Compile(regexAST)
			if err != nil {
				t.Fatalf("compile error: %v", err)
			}

			c := New(Config{
				Pattern: tt.pattern,
				Name:    "Test",
				Program: prog,
			})

			// Test each instruction
			for i, inst := range prog.Inst {
				_, err := c.generateInstruction(uint32(i), &inst)
				if tt.wantErr && err == nil {
					t.Error("expected error but got none")
				}
				if !tt.wantErr && err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
