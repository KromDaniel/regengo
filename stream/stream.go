// Package stream provides types and utilities for streaming regex matching.
//
// The streaming API enables matching regular expressions on arbitrarily large
// inputs (files, network streams) with constant memory usage. Matches are
// delivered via callbacks to avoid buffering results.
//
// Example usage with a generated pattern:
//
//	file, _ := os.Open("large.log")
//	defer file.Close()
//
//	err := CompiledLogPattern.FindReader(file, stream.Config{
//	    BufferSize: 2 * 1024 * 1024, // 2MB chunks
//	}, func(m stream.Match[*LogPatternBytesResult]) bool {
//	    fmt.Printf("Match at offset %d: %s\n", m.StreamOffset, m.Result.Match)
//	    return true // continue
//	})
package stream

// Config configures streaming regex matching behavior.
type Config struct {
	// BufferSize is the chunk size for reading from the io.Reader.
	// Default: 64KB (65536).
	// Larger values reduce syscall overhead but use more memory.
	// Minimum: 2 * MaxMatchLength (enforced at runtime).
	BufferSize int

	// MaxLeftover limits bytes kept between chunks when no match is found.
	// This prevents unbounded memory growth on streams with very long
	// non-matching sections.
	//
	// Default: computed from pattern analysis:
	//   - Bounded patterns: 10 * MaxMatchLength
	//   - Unbounded patterns: 1MB
	//
	// Set to -1 for unlimited (use with caution on infinite streams!).
	// Set to 0 to use the default.
	MaxLeftover int
}

// DefaultConfig returns a Config with sensible defaults.
// BufferSize defaults to 64KB.
// MaxLeftover is set to 0, meaning the pattern-specific default will be used.
func DefaultConfig() Config {
	return Config{
		BufferSize:  64 * 1024, // 64KB
		MaxLeftover: 0,         // Use pattern default
	}
}

// Match contains match information with stream positioning.
//
// WARNING: The Result's byte slices (Match, capture groups) point into an
// internal buffer that may be reused after the callback returns.
// You MUST copy any data you need to retain after the callback!
//
// Example - safely copying match data:
//
//	var savedMatch []byte
//	err := pattern.FindReader(r, cfg, func(m stream.Match[*Result]) bool {
//	    // WRONG: savedMatch = m.Result.Match (will be overwritten)
//	    // RIGHT:
//	    savedMatch = append([]byte{}, m.Result.Match...)
//	    return true
//	})
type Match[T any] struct {
	// Result is the pattern-specific result struct.
	// For example, *DatePatternBytesResult for a DatePattern.
	// WARNING: Byte slices in Result are only valid during the callback!
	Result T

	// StreamOffset is the absolute byte position of the match start
	// within the entire stream (0-indexed).
	StreamOffset int64

	// ChunkIndex indicates which chunk this match was found in (0-indexed).
	// Useful for debugging or progress reporting.
	ChunkIndex int
}

// Error types for streaming operations.

// ErrBufferTooSmall is returned when Config.BufferSize is less than
// the minimum required for the pattern (2 * MaxMatchLength).
type ErrBufferTooSmall struct {
	Requested int
	Minimum   int
}

func (e ErrBufferTooSmall) Error() string {
	return "stream: buffer size too small"
}

// Validate validates the Config and returns an error if invalid.
// minBuffer is the minimum buffer size required (2 * MaxMatchLength).
func (c Config) Validate(minBuffer int) error {
	if c.BufferSize > 0 && c.BufferSize < minBuffer {
		return ErrBufferTooSmall{Requested: c.BufferSize, Minimum: minBuffer}
	}
	return nil
}

// ApplyDefaults returns a Config with defaults applied for any zero values.
// minBuffer is the minimum buffer size for the pattern.
// defaultLeftover is the pattern-specific default MaxLeftover.
func (c Config) ApplyDefaults(minBuffer, defaultLeftover int) Config {
	result := c

	// Apply buffer size default
	if result.BufferSize == 0 {
		result.BufferSize = 64 * 1024 // 64KB
	}

	// Enforce minimum buffer size
	if result.BufferSize < minBuffer {
		result.BufferSize = minBuffer
	}

	// Apply MaxLeftover default
	if result.MaxLeftover == 0 {
		result.MaxLeftover = defaultLeftover
	}

	return result
}
