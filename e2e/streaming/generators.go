package streaming

import (
	"io"
	"math/rand"
)

// PatternedReader generates data with embedded matches at regular intervals.
// This is useful for testing streaming with predictable match patterns.
type PatternedReader struct {
	pattern    []byte
	noise      []byte
	matchEvery int   // Embed pattern every N bytes
	pos        int64 // Current position
	limit      int64 // Total bytes to generate
	rng        *rand.Rand
}

// NewPatternedReader creates a reader that generates data with embedded patterns.
// The pattern is embedded every matchEvery bytes, with random noise in between.
func NewPatternedReader(pattern string, noiseChars string, matchEvery int, limit int64) *PatternedReader {
	return &PatternedReader{
		pattern:    []byte(pattern),
		noise:      []byte(noiseChars),
		matchEvery: matchEvery,
		limit:      limit,
		rng:        rand.New(rand.NewSource(42)), // Deterministic for reproducibility
	}
}

func (r *PatternedReader) Read(p []byte) (n int, err error) {
	if r.pos >= r.limit {
		return 0, io.EOF
	}

	for n < len(p) && r.pos < r.limit {
		posInCycle := int(r.pos) % r.matchEvery
		if posInCycle < len(r.pattern) {
			// We're within a match
			p[n] = r.pattern[posInCycle]
		} else {
			// We're in noise section
			p[n] = r.noise[r.rng.Intn(len(r.noise))]
		}
		n++
		r.pos++
	}

	return n, nil
}

// Reset resets the reader to the beginning.
func (r *PatternedReader) Reset() {
	r.pos = 0
	r.rng = rand.New(rand.NewSource(42))
}

// ExpectedMatches returns the expected number of matches for the configured limit.
func (r *PatternedReader) ExpectedMatches() int64 {
	return r.limit / int64(r.matchEvery)
}

// ChunkedReader wraps a reader and returns data in fixed-size chunks.
// This simulates slow/fragmented network reads or tests chunk boundary handling.
type ChunkedReader struct {
	reader    io.Reader
	chunkSize int
}

// NewChunkedReader creates a reader that returns at most chunkSize bytes per Read.
func NewChunkedReader(r io.Reader, chunkSize int) *ChunkedReader {
	if chunkSize < 1 {
		chunkSize = 1
	}
	return &ChunkedReader{reader: r, chunkSize: chunkSize}
}

func (r *ChunkedReader) Read(p []byte) (n int, err error) {
	maxRead := r.chunkSize
	if len(p) < maxRead {
		maxRead = len(p)
	}
	return r.reader.Read(p[:maxRead])
}

// NewDateInputGenerator generates data with date patterns embedded.
// Dates are in YYYY-MM-DD format, embedded every 50 bytes.
func NewDateInputGenerator(size int64) *PatternedReader {
	return NewPatternedReader(
		"2024-01-15",
		"abcdefghijk \n\t",
		50,
		size,
	)
}

// NewEmailInputGenerator generates data with email patterns embedded.
// Emails are embedded every 80 bytes.
func NewEmailInputGenerator(size int64) *PatternedReader {
	return NewPatternedReader(
		"user@example.com",
		"abcdefghijk123 \n\t",
		80,
		size,
	)
}

// NewIPv4InputGenerator generates data with IPv4 address patterns embedded.
// IPs are embedded every 40 bytes.
func NewIPv4InputGenerator(size int64) *PatternedReader {
	return NewPatternedReader(
		"192.168.1.100",
		"abcdefghijk \n\t",
		40,
		size,
	)
}

// NewURLInputGenerator generates data with URL patterns embedded.
// URLs are embedded every 100 bytes.
func NewURLInputGenerator(size int64) *PatternedReader {
	return NewPatternedReader(
		"https://example.com/path",
		"abcdefghijk123 \n\t",
		100,
		size,
	)
}

// LimitedReader wraps a reader and limits the total bytes read.
type LimitedReader struct {
	reader io.Reader
	limit  int64
	read   int64
}

// NewLimitedReader creates a reader that returns at most limit bytes.
func NewLimitedReader(r io.Reader, limit int64) *LimitedReader {
	return &LimitedReader{reader: r, limit: limit}
}

func (r *LimitedReader) Read(p []byte) (n int, err error) {
	if r.read >= r.limit {
		return 0, io.EOF
	}
	remaining := r.limit - r.read
	if int64(len(p)) > remaining {
		p = p[:remaining]
	}
	n, err = r.reader.Read(p)
	r.read += int64(n)
	if r.read >= r.limit && err == nil {
		err = io.EOF
	}
	return n, err
}
