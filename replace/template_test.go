package replace

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		template string
		wantSegs []Segment
		wantErr  bool
	}{
		{
			name:     "empty",
			template: "",
			wantSegs: []Segment{},
		},
		{
			name:     "literal only",
			template: "hello world",
			wantSegs: []Segment{
				{Type: SegmentLiteral, Literal: "hello world"},
			},
		},
		{
			name:     "full match",
			template: "$0",
			wantSegs: []Segment{
				{Type: SegmentFullMatch},
			},
		},
		{
			name:     "indexed capture single digit",
			template: "$1",
			wantSegs: []Segment{
				{Type: SegmentCaptureIndex, CaptureIndex: 1},
			},
		},
		{
			name:     "indexed capture double digit",
			template: "$12",
			wantSegs: []Segment{
				{Type: SegmentCaptureIndex, CaptureIndex: 12},
			},
		},
		{
			name:     "named capture",
			template: "$name",
			wantSegs: []Segment{
				{Type: SegmentCaptureName, CaptureName: "name"},
			},
		},
		{
			name:     "escaped dollar",
			template: "$$",
			wantSegs: []Segment{
				{Type: SegmentLiteral, Literal: "$"},
			},
		},
		{
			name:     "braced index",
			template: "${1}",
			wantSegs: []Segment{
				{Type: SegmentCaptureIndex, CaptureIndex: 1},
			},
		},
		{
			name:     "braced name",
			template: "${name}",
			wantSegs: []Segment{
				{Type: SegmentCaptureName, CaptureName: "name"},
			},
		},
		{
			name:     "braced full match",
			template: "${0}",
			wantSegs: []Segment{
				{Type: SegmentFullMatch},
			},
		},
		{
			name:     "complex template",
			template: "$user@REDACTED.$tld",
			wantSegs: []Segment{
				{Type: SegmentCaptureName, CaptureName: "user"},
				{Type: SegmentLiteral, Literal: "@REDACTED."},
				{Type: SegmentCaptureName, CaptureName: "tld"},
			},
		},
		{
			name:     "mixed indices and names",
			template: "$1 by $author ($2)",
			wantSegs: []Segment{
				{Type: SegmentCaptureIndex, CaptureIndex: 1},
				{Type: SegmentLiteral, Literal: " by "},
				{Type: SegmentCaptureName, CaptureName: "author"},
				{Type: SegmentLiteral, Literal: " ("},
				{Type: SegmentCaptureIndex, CaptureIndex: 2},
				{Type: SegmentLiteral, Literal: ")"},
			},
		},
		{
			name:     "dollar at end",
			template: "cost: $",
			wantSegs: []Segment{
				{Type: SegmentLiteral, Literal: "cost: "},
				{Type: SegmentLiteral, Literal: "$"},
			},
		},
		{
			name:     "dollar followed by non-ref",
			template: "$ not a ref",
			wantSegs: []Segment{
				{Type: SegmentLiteral, Literal: "$"},
				{Type: SegmentLiteral, Literal: " not a ref"},
			},
		},
		{
			name:     "unclosed brace",
			template: "${unclosed",
			wantErr:  true,
		},
		{
			name:     "empty braces",
			template: "${}",
			wantErr:  true,
		},
		{
			name:     "invalid braced content",
			template: "${1abc}",
			wantErr:  true,
		},
		{
			name:     "invalid braced name",
			template: "${123abc}",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.template)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if len(got.Segments) != len(tt.wantSegs) {
				t.Errorf("Parse() got %d segments, want %d", len(got.Segments), len(tt.wantSegs))
				return
			}
			for i, seg := range got.Segments {
				if seg.Type != tt.wantSegs[i].Type {
					t.Errorf("segment[%d].Type = %v, want %v", i, seg.Type, tt.wantSegs[i].Type)
				}
				if seg.Literal != tt.wantSegs[i].Literal {
					t.Errorf("segment[%d].Literal = %q, want %q", i, seg.Literal, tt.wantSegs[i].Literal)
				}
				if seg.CaptureIndex != tt.wantSegs[i].CaptureIndex {
					t.Errorf("segment[%d].CaptureIndex = %d, want %d", i, seg.CaptureIndex, tt.wantSegs[i].CaptureIndex)
				}
				if seg.CaptureName != tt.wantSegs[i].CaptureName {
					t.Errorf("segment[%d].CaptureName = %q, want %q", i, seg.CaptureName, tt.wantSegs[i].CaptureName)
				}
			}
		})
	}
}

func TestValidateAndResolve(t *testing.T) {
	tests := []struct {
		name         string
		template     string
		captureNames map[string]int
		numCaptures  int
		wantErr      bool
		errContains  string
	}{
		{
			name:         "literal only",
			template:     "hello",
			captureNames: map[string]int{},
			numCaptures:  0,
		},
		{
			name:         "valid index",
			template:     "$1",
			captureNames: map[string]int{},
			numCaptures:  2,
		},
		{
			name:         "valid name",
			template:     "$user",
			captureNames: map[string]int{"user": 1},
			numCaptures:  1,
		},
		{
			name:         "valid full match",
			template:     "$0",
			captureNames: map[string]int{},
			numCaptures:  0,
		},
		{
			name:         "invalid index out of range",
			template:     "$3",
			captureNames: map[string]int{},
			numCaptures:  2,
			wantErr:      true,
			errContains:  "out of range",
		},
		{
			name:         "invalid name not found",
			template:     "$invalid",
			captureNames: map[string]int{"user": 1},
			numCaptures:  1,
			wantErr:      true,
			errContains:  "not found",
		},
		{
			name:         "complex valid",
			template:     "$user@REDACTED.$tld",
			captureNames: map[string]int{"user": 1, "domain": 2, "tld": 3},
			numCaptures:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := Parse(tt.template)
			if err != nil {
				t.Fatalf("Parse() failed: %v", err)
			}

			resolved, err := tmpl.ValidateAndResolve(tt.captureNames, tt.numCaptures)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAndResolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if tt.errContains != "" && err != nil {
					if !contains(err.Error(), tt.errContains) {
						t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
					}
				}
				return
			}

			// Verify all SegmentCaptureName are resolved to SegmentCaptureIndex
			for i, seg := range resolved {
				if seg.Type == SegmentCaptureName {
					t.Errorf("segment[%d] should be resolved, got SegmentCaptureName", i)
				}
			}
		})
	}
}

func TestValidateAndResolve_ResolvesNames(t *testing.T) {
	tmpl, err := Parse("$user@$domain")
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	captureNames := map[string]int{"user": 1, "domain": 2}
	resolved, err := tmpl.ValidateAndResolve(captureNames, 2)
	if err != nil {
		t.Fatalf("ValidateAndResolve() failed: %v", err)
	}

	// First segment should be resolved to index 1
	if resolved[0].Type != SegmentCaptureIndex {
		t.Errorf("segment[0].Type = %v, want SegmentCaptureIndex", resolved[0].Type)
	}
	if resolved[0].CaptureIndex != 1 {
		t.Errorf("segment[0].CaptureIndex = %d, want 1", resolved[0].CaptureIndex)
	}

	// Second segment is literal "@"
	if resolved[1].Type != SegmentLiteral {
		t.Errorf("segment[1].Type = %v, want SegmentLiteral", resolved[1].Type)
	}

	// Third segment should be resolved to index 2
	if resolved[2].Type != SegmentCaptureIndex {
		t.Errorf("segment[2].Type = %v, want SegmentCaptureIndex", resolved[2].Type)
	}
	if resolved[2].CaptureIndex != 2 {
		t.Errorf("segment[2].CaptureIndex = %d, want 2", resolved[2].CaptureIndex)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
