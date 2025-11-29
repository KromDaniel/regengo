package codegen

import "testing"

func TestInstructionName(t *testing.T) {
	tests := []struct {
		id   uint32
		want string
	}{
		{0, "Ins0"},
		{1, "Ins1"},
		{100, "Ins100"},
	}

	for _, tt := range tests {
		got := InstructionName(tt.id)
		if got != tt.want {
			t.Errorf("InstructionName(%d) = %q, want %q", tt.id, got, tt.want)
		}
	}
}

func TestLowerFirst(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"A", "a"},
		{"ABC", "aBC"},
		{"Hello", "hello"},
		{"hello", "hello"},
		{"X", "x"},
	}

	for _, tt := range tests {
		got := LowerFirst(tt.input)
		if got != tt.want {
			t.Errorf("LowerFirst(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestUpperFirst(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"a", "A"},
		{"abc", "Abc"},
		{"hello", "Hello"},
		{"Hello", "Hello"},
		{"x", "X"},
	}

	for _, tt := range tests {
		got := UpperFirst(tt.input)
		if got != tt.want {
			t.Errorf("UpperFirst(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
