package compiler

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseReplaceTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     []TemplateSegment
		wantErr  bool
	}{
		// Empty and literal-only cases
		{
			name:     "empty template",
			template: "",
			want:     []TemplateSegment{},
			wantErr:  false,
		},
		{
			name:     "literal only",
			template: "hello world",
			want: []TemplateSegment{
				{Type: SegmentLiteral, Literal: "hello world"},
			},
			wantErr: false,
		},
		{
			name:     "literal with special chars",
			template: "hello@world.com",
			want: []TemplateSegment{
				{Type: SegmentLiteral, Literal: "hello@world.com"},
			},
			wantErr: false,
		},

		// Full match reference ($0)
		{
			name:     "full match $0",
			template: "$0",
			want: []TemplateSegment{
				{Type: SegmentFullMatch},
			},
			wantErr: false,
		},
		{
			name:     "full match ${0}",
			template: "${0}",
			want: []TemplateSegment{
				{Type: SegmentFullMatch},
			},
			wantErr: false,
		},
		{
			name:     "full match with brackets",
			template: "[$0]",
			want: []TemplateSegment{
				{Type: SegmentLiteral, Literal: "["},
				{Type: SegmentFullMatch},
				{Type: SegmentLiteral, Literal: "]"},
			},
			wantErr: false,
		},

		// Indexed capture references ($1, $2, ..., $99)
		{
			name:     "single digit index $1",
			template: "$1",
			want: []TemplateSegment{
				{Type: SegmentCaptureIndex, CaptureIndex: 1},
			},
			wantErr: false,
		},
		{
			name:     "single digit index $9",
			template: "$9",
			want: []TemplateSegment{
				{Type: SegmentCaptureIndex, CaptureIndex: 9},
			},
			wantErr: false,
		},
		{
			name:     "double digit index $12",
			template: "$12",
			want: []TemplateSegment{
				{Type: SegmentCaptureIndex, CaptureIndex: 12},
			},
			wantErr: false,
		},
		{
			name:     "double digit index $99",
			template: "$99",
			want: []TemplateSegment{
				{Type: SegmentCaptureIndex, CaptureIndex: 99},
			},
			wantErr: false,
		},
		{
			name:     "explicit index ${1}",
			template: "${1}",
			want: []TemplateSegment{
				{Type: SegmentCaptureIndex, CaptureIndex: 1},
			},
			wantErr: false,
		},
		{
			name:     "explicit index ${12}",
			template: "${12}",
			want: []TemplateSegment{
				{Type: SegmentCaptureIndex, CaptureIndex: 12},
			},
			wantErr: false,
		},
		{
			name:     "explicit index ${99}",
			template: "${99}",
			want: []TemplateSegment{
				{Type: SegmentCaptureIndex, CaptureIndex: 99},
			},
			wantErr: false,
		},
		{
			name:     "index followed by digit literal $1x",
			template: "$1x",
			want: []TemplateSegment{
				{Type: SegmentCaptureIndex, CaptureIndex: 1},
				{Type: SegmentLiteral, Literal: "x"},
			},
			wantErr: false,
		},
		{
			name:     "triple digit becomes double + literal $123",
			template: "$123",
			want: []TemplateSegment{
				{Type: SegmentCaptureIndex, CaptureIndex: 12},
				{Type: SegmentLiteral, Literal: "3"},
			},
			wantErr: false,
		},

		// Named capture references ($name, ${name})
		{
			name:     "named simple $user",
			template: "$user",
			want: []TemplateSegment{
				{Type: SegmentCaptureName, CaptureName: "user"},
			},
			wantErr: false,
		},
		{
			name:     "named explicit ${user}",
			template: "${user}",
			want: []TemplateSegment{
				{Type: SegmentCaptureName, CaptureName: "user"},
			},
			wantErr: false,
		},
		{
			name:     "named with underscore $user_name",
			template: "$user_name",
			want: []TemplateSegment{
				{Type: SegmentCaptureName, CaptureName: "user_name"},
			},
			wantErr: false,
		},
		{
			name:     "named starting with underscore $_private",
			template: "$_private",
			want: []TemplateSegment{
				{Type: SegmentCaptureName, CaptureName: "_private"},
			},
			wantErr: false,
		},
		{
			name:     "named with digits $user2",
			template: "$user2",
			want: []TemplateSegment{
				{Type: SegmentCaptureName, CaptureName: "user2"},
			},
			wantErr: false,
		},
		{
			name:     "explicit named with digits ${user2}",
			template: "${user2}",
			want: []TemplateSegment{
				{Type: SegmentCaptureName, CaptureName: "user2"},
			},
			wantErr: false,
		},

		// Escaped dollar ($$)
		{
			name:     "escaped dollar $$",
			template: "$$",
			want: []TemplateSegment{
				{Type: SegmentLiteral, Literal: "$"},
			},
			wantErr: false,
		},
		{
			name:     "escaped dollar in text cost: $$100",
			template: "cost: $$100",
			want: []TemplateSegment{
				{Type: SegmentLiteral, Literal: "cost: "},
				{Type: SegmentLiteral, Literal: "$"},
				{Type: SegmentLiteral, Literal: "100"},
			},
			wantErr: false,
		},
		{
			name:     "multiple escaped dollars $$$$",
			template: "$$$$",
			want: []TemplateSegment{
				{Type: SegmentLiteral, Literal: "$"},
				{Type: SegmentLiteral, Literal: "$"},
			},
			wantErr: false,
		},

		// Mixed/combined templates
		{
			name:     "mixed $1-$name-$$-end",
			template: "$1-$name-$$-end",
			want: []TemplateSegment{
				{Type: SegmentCaptureIndex, CaptureIndex: 1},
				{Type: SegmentLiteral, Literal: "-"},
				{Type: SegmentCaptureName, CaptureName: "name"},
				{Type: SegmentLiteral, Literal: "-"},
				{Type: SegmentLiteral, Literal: "$"},
				{Type: SegmentLiteral, Literal: "-end"},
			},
			wantErr: false,
		},
		{
			name:     "email mask $user@REDACTED.$tld",
			template: "$user@REDACTED.$tld",
			want: []TemplateSegment{
				{Type: SegmentCaptureName, CaptureName: "user"},
				{Type: SegmentLiteral, Literal: "@REDACTED."},
				{Type: SegmentCaptureName, CaptureName: "tld"},
			},
			wantErr: false,
		},
		{
			name:     "complex with all types",
			template: "prefix $0 and $1 with $name and $$100 suffix",
			want: []TemplateSegment{
				{Type: SegmentLiteral, Literal: "prefix "},
				{Type: SegmentFullMatch},
				{Type: SegmentLiteral, Literal: " and "},
				{Type: SegmentCaptureIndex, CaptureIndex: 1},
				{Type: SegmentLiteral, Literal: " with "},
				{Type: SegmentCaptureName, CaptureName: "name"},
				{Type: SegmentLiteral, Literal: " and "},
				{Type: SegmentLiteral, Literal: "$"},
				{Type: SegmentLiteral, Literal: "100 suffix"},
			},
			wantErr: false,
		},
		{
			name:     "explicit boundary prevents name continuation ${user}name",
			template: "${user}name",
			want: []TemplateSegment{
				{Type: SegmentCaptureName, CaptureName: "user"},
				{Type: SegmentLiteral, Literal: "name"},
			},
			wantErr: false,
		},
		{
			name:     "explicit boundary for index ${1}23",
			template: "${1}23",
			want: []TemplateSegment{
				{Type: SegmentCaptureIndex, CaptureIndex: 1},
				{Type: SegmentLiteral, Literal: "23"},
			},
			wantErr: false,
		},

		// Edge cases with dollar at various positions
		{
			name:     "dollar at end",
			template: "test$",
			want: []TemplateSegment{
				{Type: SegmentLiteral, Literal: "test"},
				{Type: SegmentLiteral, Literal: "$"},
			},
			wantErr: false,
		},
		{
			name:     "just dollar",
			template: "$",
			want: []TemplateSegment{
				{Type: SegmentLiteral, Literal: "$"},
			},
			wantErr: false,
		},
		{
			name:     "dollar followed by space",
			template: "$ hello",
			want: []TemplateSegment{
				{Type: SegmentLiteral, Literal: "$"},
				{Type: SegmentLiteral, Literal: " hello"},
			},
			wantErr: false,
		},
		{
			name:     "dollar followed by special char",
			template: "$@test",
			want: []TemplateSegment{
				{Type: SegmentLiteral, Literal: "$"},
				{Type: SegmentLiteral, Literal: "@test"},
			},
			wantErr: false,
		},

		// Error cases
		{
			name:     "unclosed brace ${name",
			template: "${name",
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "unclosed brace at end ${",
			template: "test${",
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "empty braces ${}",
			template: "${}",
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "invalid mixed in braces ${123abc}",
			template: "${123abc}",
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "invalid start in braces ${1abc}",
			template: "${1abc}",
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "invalid char in braces ${a-b}",
			template: "${a-b}",
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "space in braces ${a b}",
			template: "${a b}",
			want:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseReplaceTemplate(tt.template)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseReplaceTemplate(%q) error = %v, wantErr %v", tt.template, err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return // Error expected, no need to check segments
			}

			if got.Original != tt.template {
				t.Errorf("ParseReplaceTemplate(%q).Original = %q, want %q", tt.template, got.Original, tt.template)
			}

			if !reflect.DeepEqual(got.Segments, tt.want) {
				t.Errorf("ParseReplaceTemplate(%q).Segments = %+v, want %+v", tt.template, got.Segments, tt.want)
			}
		})
	}
}

func TestParseReplaceTemplate_AllDigitIndices(t *testing.T) {
	// Test $1 through $9 individually
	for i := 1; i <= 9; i++ {
		template := "$" + string(rune('0'+i))
		got, err := ParseReplaceTemplate(template)
		if err != nil {
			t.Errorf("ParseReplaceTemplate(%q) unexpected error: %v", template, err)
			continue
		}
		if len(got.Segments) != 1 {
			t.Errorf("ParseReplaceTemplate(%q) got %d segments, want 1", template, len(got.Segments))
			continue
		}
		if got.Segments[0].Type != SegmentCaptureIndex {
			t.Errorf("ParseReplaceTemplate(%q) got type %v, want SegmentCaptureIndex", template, got.Segments[0].Type)
			continue
		}
		if got.Segments[0].CaptureIndex != i {
			t.Errorf("ParseReplaceTemplate(%q) got index %d, want %d", template, got.Segments[0].CaptureIndex, i)
		}
	}

	// Test $10 through $99 (sampling)
	samples := []int{10, 15, 20, 50, 75, 99}
	for _, i := range samples {
		template := "$" + string(rune('0'+i/10)) + string(rune('0'+i%10))
		got, err := ParseReplaceTemplate(template)
		if err != nil {
			t.Errorf("ParseReplaceTemplate(%q) unexpected error: %v", template, err)
			continue
		}
		if len(got.Segments) != 1 {
			t.Errorf("ParseReplaceTemplate(%q) got %d segments, want 1", template, len(got.Segments))
			continue
		}
		if got.Segments[0].CaptureIndex != i {
			t.Errorf("ParseReplaceTemplate(%q) got index %d, want %d", template, got.Segments[0].CaptureIndex, i)
		}
	}
}

func TestParseReplaceTemplate_NamedVariants(t *testing.T) {
	// Test various valid identifier patterns
	validNames := []string{
		"a",
		"Z",
		"_",
		"_a",
		"_1",
		"abc",
		"ABC",
		"abc123",
		"a_b_c",
		"CamelCase",
		"snake_case",
		"SCREAMING_SNAKE",
		"mixedCase123",
	}

	for _, name := range validNames {
		// Test $name form
		template := "$" + name
		got, err := ParseReplaceTemplate(template)
		if err != nil {
			t.Errorf("ParseReplaceTemplate(%q) unexpected error: %v", template, err)
			continue
		}
		if len(got.Segments) != 1 {
			t.Errorf("ParseReplaceTemplate(%q) got %d segments, want 1", template, len(got.Segments))
			continue
		}
		if got.Segments[0].Type != SegmentCaptureName {
			t.Errorf("ParseReplaceTemplate(%q) got type %v, want SegmentCaptureName", template, got.Segments[0].Type)
			continue
		}
		if got.Segments[0].CaptureName != name {
			t.Errorf("ParseReplaceTemplate(%q) got name %q, want %q", template, got.Segments[0].CaptureName, name)
		}

		// Test ${name} form
		template = "${" + name + "}"
		got, err = ParseReplaceTemplate(template)
		if err != nil {
			t.Errorf("ParseReplaceTemplate(%q) unexpected error: %v", template, err)
			continue
		}
		if len(got.Segments) != 1 {
			t.Errorf("ParseReplaceTemplate(%q) got %d segments, want 1", template, len(got.Segments))
			continue
		}
		if got.Segments[0].CaptureName != name {
			t.Errorf("ParseReplaceTemplate(%q) got name %q, want %q", template, got.Segments[0].CaptureName, name)
		}
	}
}

func TestParseReplaceTemplate_RealWorldExamples(t *testing.T) {
	tests := []struct {
		name     string
		template string
		wantLen  int // Just check segment count for these
	}{
		{"IP masking", "$1.$2.$3.***", 6},
		{"Email redaction", "$user@REDACTED", 2},
		{"JSON replacement", `{"value": "$1"}`, 3},
		{"SQL sanitization", "SELECT * FROM $table WHERE id = $id", 4},
		{"URL rewrite", "https://$domain/$path?$query", 6},
		{"Log formatting", "[$timestamp] $level: $message", 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseReplaceTemplate(tt.template)
			if err != nil {
				t.Errorf("ParseReplaceTemplate(%q) unexpected error: %v", tt.template, err)
				return
			}
			if len(got.Segments) != tt.wantLen {
				t.Errorf("ParseReplaceTemplate(%q) got %d segments, want %d. Segments: %+v",
					tt.template, len(got.Segments), tt.wantLen, got.Segments)
			}
		})
	}
}

func TestIsValidIdentifier(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		// Valid
		{"a", true},
		{"A", true},
		{"_", true},
		{"abc", true},
		{"ABC", true},
		{"_abc", true},
		{"abc123", true},
		{"a1b2c3", true},
		{"_123", true},

		// Invalid
		{"", false},
		{"1", false},
		{"123", false},
		{"1abc", false},
		{"a-b", false},
		{"a b", false},
		{"a.b", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := isValidIdentifier(tt.input); got != tt.want {
				t.Errorf("isValidIdentifier(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateTemplate(t *testing.T) {
	tests := []struct {
		name         string
		template     string
		captureNames []string
		numCaptures  int
		wantErr      bool
		errContains  string
	}{
		// Valid cases
		{
			name:         "valid full match",
			template:     "$0",
			captureNames: []string{"user", "domain"},
			numCaptures:  2,
			wantErr:      false,
		},
		{
			name:         "valid index 1",
			template:     "$1",
			captureNames: []string{"user", "domain"},
			numCaptures:  2,
			wantErr:      false,
		},
		{
			name:         "valid index 2",
			template:     "$2",
			captureNames: []string{"user", "domain"},
			numCaptures:  2,
			wantErr:      false,
		},
		{
			name:         "valid named user",
			template:     "$user",
			captureNames: []string{"user", "domain"},
			numCaptures:  2,
			wantErr:      false,
		},
		{
			name:         "valid named domain",
			template:     "$domain",
			captureNames: []string{"user", "domain"},
			numCaptures:  2,
			wantErr:      false,
		},
		{
			name:         "valid mixed",
			template:     "$1-$domain",
			captureNames: []string{"user", "domain"},
			numCaptures:  2,
			wantErr:      false,
		},
		{
			name:         "literal only",
			template:     "REDACTED",
			captureNames: []string{"user", "domain"},
			numCaptures:  2,
			wantErr:      false,
		},
		{
			name:         "complex valid",
			template:     "$user@REDACTED.$domain",
			captureNames: []string{"user", "domain"},
			numCaptures:  2,
			wantErr:      false,
		},

		// Invalid index cases
		{
			name:         "index too high",
			template:     "$3",
			captureNames: []string{"user", "domain"},
			numCaptures:  2,
			wantErr:      true,
			errContains:  "capture group 3",
		},
		{
			name:         "index way too high",
			template:     "$99",
			captureNames: []string{"user", "domain"},
			numCaptures:  2,
			wantErr:      true,
			errContains:  "capture group 99",
		},

		// Full match is always valid
		{
			name:         "full match always valid with captures",
			template:     "$0",
			captureNames: []string{"user", "domain"},
			numCaptures:  2,
			wantErr:      false,
		},
		{
			name:         "full match valid with no captures",
			template:     "$0",
			captureNames: []string{},
			numCaptures:  0,
			wantErr:      false,
		},

		// Invalid name cases
		{
			name:         "unknown name",
			template:     "$invalid",
			captureNames: []string{"user", "domain"},
			numCaptures:  2,
			wantErr:      true,
			errContains:  "invalid",
		},
		{
			name:         "typo in name",
			template:     "$usr",
			captureNames: []string{"user", "domain"},
			numCaptures:  2,
			wantErr:      true,
			errContains:  "usr",
		},
		{
			name:         "case sensitive",
			template:     "$User",
			captureNames: []string{"user", "domain"},
			numCaptures:  2,
			wantErr:      true,
			errContains:  "User",
		},

		// No captures pattern
		{
			name:         "no captures valid $0",
			template:     "$0",
			captureNames: []string{},
			numCaptures:  0,
			wantErr:      false,
		},
		{
			name:         "no captures invalid $1",
			template:     "$1",
			captureNames: []string{},
			numCaptures:  0,
			wantErr:      true,
			errContains:  "capture group 1",
		},
		{
			name:         "no captures literal ok",
			template:     "REPLACEMENT",
			captureNames: []string{},
			numCaptures:  0,
			wantErr:      false,
		},

		// Unnamed captures only
		{
			name:         "unnamed captures index ok",
			template:     "$1",
			captureNames: []string{"", ""},
			numCaptures:  2,
			wantErr:      false,
		},
		{
			name:         "unnamed captures name fails",
			template:     "$name",
			captureNames: []string{"", ""},
			numCaptures:  2,
			wantErr:      true,
			errContains:  "name",
		},

		// Multiple errors - returns first
		{
			name:         "first error returned",
			template:     "$invalid1-$invalid2",
			captureNames: []string{"user"},
			numCaptures:  1,
			wantErr:      true,
			errContains:  "invalid1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseReplaceTemplate(tt.template)
			if err != nil {
				t.Fatalf("ParseReplaceTemplate(%q) failed: %v", tt.template, err)
			}

			err = ValidateTemplate(parsed, tt.captureNames, tt.numCaptures)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateTemplate() error = %v, want error containing %q", err, tt.errContains)
				}
			}
		})
	}
}

func TestResolveTemplate(t *testing.T) {
	tests := []struct {
		name         string
		template     string
		captureNames []string
		wantIndices  []int // Expected CaptureIndex for each non-literal segment, in order
		wantErr      bool
	}{
		{
			name:         "single named to index",
			template:     "$user",
			captureNames: []string{"user", "domain"},
			wantIndices:  []int{1},
			wantErr:      false,
		},
		{
			name:         "second named to index",
			template:     "$domain",
			captureNames: []string{"user", "domain"},
			wantIndices:  []int{2},
			wantErr:      false,
		},
		{
			name:         "multiple named to indices",
			template:     "$user-$domain",
			captureNames: []string{"user", "domain"},
			wantIndices:  []int{1, 2}, // $user=1, literal, $domain=2
			wantErr:      false,
		},
		{
			name:         "mixed named and indexed",
			template:     "$1-$domain",
			captureNames: []string{"user", "domain"},
			wantIndices:  []int{1, 2}, // $1=1, literal, $domain=2
			wantErr:      false,
		},
		{
			name:         "full match preserved",
			template:     "$0-$user",
			captureNames: []string{"user", "domain"},
			wantIndices:  []int{0, 1}, // $0=0 (FullMatch), literal, $user=1
			wantErr:      false,
		},
		{
			name:         "literal only - no indices",
			template:     "REDACTED",
			captureNames: []string{"user", "domain"},
			wantIndices:  nil, // No capture refs, so no indices
			wantErr:      false,
		},
		{
			name:         "invalid name errors",
			template:     "$invalid",
			captureNames: []string{"user", "domain"},
			wantIndices:  nil,
			wantErr:      true,
		},
		{
			name:         "complex template",
			template:     "$user@REDACTED.$domain",
			captureNames: []string{"user", "domain"},
			wantIndices:  []int{1, 2}, // $user=1, literal, $domain=2
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseReplaceTemplate(tt.template)
			if err != nil {
				t.Fatalf("ParseReplaceTemplate(%q) failed: %v", tt.template, err)
			}

			resolved, err := ResolveTemplate(parsed, tt.captureNames)

			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Verify Original is preserved
			if resolved.Original != tt.template {
				t.Errorf("ResolveTemplate() Original = %q, want %q", resolved.Original, tt.template)
			}

			// Collect actual indices from non-literal segments
			var gotIndices []int
			for _, seg := range resolved.Segments {
				switch seg.Type {
				case SegmentFullMatch:
					gotIndices = append(gotIndices, 0)
				case SegmentCaptureIndex:
					gotIndices = append(gotIndices, seg.CaptureIndex)
				case SegmentCaptureName:
					t.Errorf("ResolveTemplate() still has SegmentCaptureName: %+v", seg)
				}
			}

			if !reflect.DeepEqual(gotIndices, tt.wantIndices) {
				t.Errorf("ResolveTemplate() indices = %v, want %v", gotIndices, tt.wantIndices)
			}
		})
	}
}

func TestResolveTemplate_PreservesSegmentTypes(t *testing.T) {
	// Verify that literals and full match are preserved correctly
	template := "prefix-$0-$user-$1-suffix"
	captureNames := []string{"user", "domain"}

	parsed, err := ParseReplaceTemplate(template)
	if err != nil {
		t.Fatalf("ParseReplaceTemplate failed: %v", err)
	}

	resolved, err := ResolveTemplate(parsed, captureNames)
	if err != nil {
		t.Fatalf("ResolveTemplate failed: %v", err)
	}

	// Expected: Literal, FullMatch, Literal, CaptureIndex(1), Literal, CaptureIndex(1), Literal
	expected := []struct {
		segType TemplateSegmentType
		literal string
		index   int
	}{
		{SegmentLiteral, "prefix-", 0},
		{SegmentFullMatch, "", 0},
		{SegmentLiteral, "-", 0},
		{SegmentCaptureIndex, "", 1}, // $user resolved to index 1
		{SegmentLiteral, "-", 0},
		{SegmentCaptureIndex, "", 1}, // $1 stays as index 1
		{SegmentLiteral, "-suffix", 0},
	}

	if len(resolved.Segments) != len(expected) {
		t.Fatalf("ResolveTemplate() got %d segments, want %d", len(resolved.Segments), len(expected))
	}

	for i, exp := range expected {
		got := resolved.Segments[i]
		if got.Type != exp.segType {
			t.Errorf("segment[%d].Type = %v, want %v", i, got.Type, exp.segType)
		}
		if exp.segType == SegmentLiteral && got.Literal != exp.literal {
			t.Errorf("segment[%d].Literal = %q, want %q", i, got.Literal, exp.literal)
		}
		if exp.segType == SegmentCaptureIndex && got.CaptureIndex != exp.index {
			t.Errorf("segment[%d].CaptureIndex = %d, want %d", i, got.CaptureIndex, exp.index)
		}
	}
}

func TestValidateAndResolve_Integration(t *testing.T) {
	// Test that Validate + Resolve work together correctly
	template := "$user@$domain.$tld"
	captureNames := []string{"user", "domain", "tld"}
	numCaptures := 3

	parsed, err := ParseReplaceTemplate(template)
	if err != nil {
		t.Fatalf("ParseReplaceTemplate failed: %v", err)
	}

	// First validate
	if err := ValidateTemplate(parsed, captureNames, numCaptures); err != nil {
		t.Fatalf("ValidateTemplate failed: %v", err)
	}

	// Then resolve
	resolved, err := ResolveTemplate(parsed, captureNames)
	if err != nil {
		t.Fatalf("ResolveTemplate failed: %v", err)
	}

	// Verify all named refs are now indexed
	for i, seg := range resolved.Segments {
		if seg.Type == SegmentCaptureName {
			t.Errorf("segment[%d] still has SegmentCaptureName after resolve", i)
		}
	}

	// Verify correct indices
	expectedIndices := map[int]int{
		0: 1, // $user -> 1
		2: 2, // $domain -> 2
		4: 3, // $tld -> 3
	}
	for segIdx, wantIndex := range expectedIndices {
		if resolved.Segments[segIdx].CaptureIndex != wantIndex {
			t.Errorf("segment[%d].CaptureIndex = %d, want %d",
				segIdx, resolved.Segments[segIdx].CaptureIndex, wantIndex)
		}
	}
}
