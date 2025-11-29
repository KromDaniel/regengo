package compiler

import "testing"

func TestBoolToInt(t *testing.T) {
	tests := []struct {
		input bool
		want  int
	}{
		{true, 1},
		{false, 0},
	}

	for _, tt := range tests {
		got := boolToInt(tt.input)
		if got != tt.want {
			t.Errorf("boolToInt(%v) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
