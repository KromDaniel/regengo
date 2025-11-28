package stream

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.BufferSize != 64*1024 {
		t.Errorf("DefaultConfig().BufferSize = %d, want %d", cfg.BufferSize, 64*1024)
	}
	if cfg.MaxLeftover != 0 {
		t.Errorf("DefaultConfig().MaxLeftover = %d, want 0", cfg.MaxLeftover)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name      string
		cfg       Config
		minBuffer int
		wantErr   bool
	}{
		{
			name:      "zero buffer size is valid",
			cfg:       Config{BufferSize: 0},
			minBuffer: 100,
			wantErr:   false,
		},
		{
			name:      "buffer size equal to minimum",
			cfg:       Config{BufferSize: 100},
			minBuffer: 100,
			wantErr:   false,
		},
		{
			name:      "buffer size larger than minimum",
			cfg:       Config{BufferSize: 1000},
			minBuffer: 100,
			wantErr:   false,
		},
		{
			name:      "buffer size smaller than minimum",
			cfg:       Config{BufferSize: 50},
			minBuffer: 100,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate(tt.minBuffer)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigApplyDefaults(t *testing.T) {
	tests := []struct {
		name            string
		cfg             Config
		minBuffer       int
		defaultLeftover int
		wantBufferSize  int
		wantMaxLeftover int
	}{
		{
			name:            "all zeros get defaults",
			cfg:             Config{},
			minBuffer:       100,
			defaultLeftover: 1024,
			wantBufferSize:  64 * 1024,
			wantMaxLeftover: 1024,
		},
		{
			name:            "buffer below minimum gets minimum",
			cfg:             Config{BufferSize: 50},
			minBuffer:       100,
			defaultLeftover: 1024,
			wantBufferSize:  100,
			wantMaxLeftover: 1024,
		},
		{
			name:            "explicit values preserved",
			cfg:             Config{BufferSize: 1000, MaxLeftover: 500},
			minBuffer:       100,
			defaultLeftover: 1024,
			wantBufferSize:  1000,
			wantMaxLeftover: 500,
		},
		{
			name:            "unlimited leftover preserved",
			cfg:             Config{MaxLeftover: -1},
			minBuffer:       100,
			defaultLeftover: 1024,
			wantBufferSize:  64 * 1024,
			wantMaxLeftover: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.ApplyDefaults(tt.minBuffer, tt.defaultLeftover)

			if got.BufferSize != tt.wantBufferSize {
				t.Errorf("ApplyDefaults().BufferSize = %d, want %d", got.BufferSize, tt.wantBufferSize)
			}
			if got.MaxLeftover != tt.wantMaxLeftover {
				t.Errorf("ApplyDefaults().MaxLeftover = %d, want %d", got.MaxLeftover, tt.wantMaxLeftover)
			}
		})
	}
}

func TestErrBufferTooSmall(t *testing.T) {
	err := ErrBufferTooSmall{Requested: 50, Minimum: 100}

	if err.Error() == "" {
		t.Error("ErrBufferTooSmall.Error() returned empty string")
	}
}

func TestMatchGenericType(t *testing.T) {
	// Test that Match works with different type parameters
	type TestResult struct {
		Value string
	}

	match := Match[*TestResult]{
		Result:       &TestResult{Value: "test"},
		StreamOffset: 100,
		ChunkIndex:   2,
	}

	if match.Result.Value != "test" {
		t.Errorf("Match.Result.Value = %q, want %q", match.Result.Value, "test")
	}
	if match.StreamOffset != 100 {
		t.Errorf("Match.StreamOffset = %d, want %d", match.StreamOffset, 100)
	}
	if match.ChunkIndex != 2 {
		t.Errorf("Match.ChunkIndex = %d, want %d", match.ChunkIndex, 2)
	}
}
