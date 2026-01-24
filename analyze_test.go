package regengo

import (
	"testing"
)

func TestAnalyze(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		wantErr bool
	}{
		{
			name:    "simple pattern",
			pattern: "hello",
			wantErr: false,
		},
		{
			name:    "pattern with captures",
			pattern: "(?P<user>\\w+)@(?P<domain>\\w+)",
			wantErr: false,
		},
		{
			name:    "invalid pattern",
			pattern: "[invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Analyze(tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("Analyze() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("Analyze() returned nil result for valid pattern")
			}
		})
	}
}

func TestAnalyzeWithThreshold(t *testing.T) {
	// Test the deprecated function still works
	result, err := AnalyzeWithThreshold("hello", 500)
	if err != nil {
		t.Errorf("AnalyzeWithThreshold() error = %v", err)
		return
	}
	if result == nil {
		t.Error("AnalyzeWithThreshold() returned nil result")
	}
}

func TestAnalyzeWithThresholds(t *testing.T) {
	tests := []struct {
		name          string
		pattern       string
		tdfaThreshold int
		dfaThreshold  int
		wantEngine    string // expected engine label
		wantErr       bool
	}{
		{
			name:          "simple pattern uses backtracking",
			pattern:       "hello",
			tdfaThreshold: 500,
			dfaThreshold:  500,
			wantEngine:    "Backtracking",
			wantErr:       false,
		},
		{
			name:          "catastrophic pattern with DFA",
			pattern:       "(a+)+b",
			tdfaThreshold: 500,
			dfaThreshold:  500,
			wantEngine:    "DFA",
			wantErr:       false,
		},
		{
			name:          "pattern with captures and catastrophic risk uses TDFA",
			pattern:       "(?P<x>(a+)+)b",
			tdfaThreshold: 500,
			dfaThreshold:  500,
			wantEngine:    "TDFA",
			wantErr:       false,
		},
		{
			name:          "invalid pattern",
			pattern:       "[invalid",
			tdfaThreshold: 500,
			dfaThreshold:  500,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := AnalyzeWithThresholds(tt.pattern, tt.tdfaThreshold, tt.dfaThreshold)
			if (err != nil) != tt.wantErr {
				t.Errorf("AnalyzeWithThresholds() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if result == nil {
				t.Error("AnalyzeWithThresholds() returned nil result")
				return
			}

			// Check that expected engine is in labels
			found := false
			for _, label := range result.EngineLabels {
				if label == tt.wantEngine {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected engine label %q not found in %v", tt.wantEngine, result.EngineLabels)
			}
		})
	}
}
