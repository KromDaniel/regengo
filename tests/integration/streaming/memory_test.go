package streaming

import (
	"io"
	"math/rand"
	"runtime"
	"testing"

	stream "github.com/KromDaniel/regengo/stream"
	"github.com/KromDaniel/regengo/tests/integration/streaming/testdata"
)

// PatternedReader generates test data with embedded date patterns.
type PatternedReader struct {
	pattern    []byte
	noise      []byte
	matchEvery int
	pos        int64
	limit      int64
	rng        *rand.Rand
}

func NewPatternedReader(pattern string, matchEvery int, limit int64) *PatternedReader {
	return &PatternedReader{
		pattern:    []byte(pattern),
		noise:      []byte("abcdefghijk \n\t"),
		matchEvery: matchEvery,
		limit:      limit,
		rng:        rand.New(rand.NewSource(42)),
	}
}

func (r *PatternedReader) Read(p []byte) (n int, err error) {
	if r.pos >= r.limit {
		return 0, io.EOF
	}
	for n < len(p) && r.pos < r.limit {
		posInCycle := int(r.pos) % r.matchEvery
		if posInCycle < len(r.pattern) {
			p[n] = r.pattern[posInCycle]
		} else {
			p[n] = r.noise[r.rng.Intn(len(r.noise))]
		}
		n++
		r.pos++
	}
	return n, nil
}

// TestMemoryBounded verifies streaming uses bounded memory regardless of input size.
func TestMemoryBounded(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	// Process 64MB of data
	dataSize := int64(64 << 20)
	gen := NewPatternedReader("2024-01-15", 50, dataSize)

	// Force GC and get baseline
	runtime.GC()
	var startMem runtime.MemStats
	runtime.ReadMemStats(&startMem)
	startAlloc := startMem.TotalAlloc

	// Process stream with default config
	var count int64
	err := testdata.CompiledDatePattern.FindReader(gen, stream.Config{},
		func(m stream.Match[*testdata.DatePatternBytesResult]) bool {
			count++
			return true
		})
	if err != nil {
		t.Fatalf("FindReader error: %v", err)
	}

	// Check memory usage
	runtime.GC()
	var endMem runtime.MemStats
	runtime.ReadMemStats(&endMem)
	totalAlloc := endMem.TotalAlloc - startAlloc

	// Memory should be bounded - allow up to 10MB for 64MB of data
	maxExpected := uint64(10 << 20)
	if totalAlloc > maxExpected {
		t.Errorf("Memory allocation too high: %d bytes (max expected %d)", totalAlloc, maxExpected)
	}

	t.Logf("Processed %d bytes, found %d matches", dataSize, count)
	t.Logf("Total allocations: %d bytes (%.2f%% of data size)",
		totalAlloc, float64(totalAlloc)/float64(dataSize)*100)

	// Verify we found a reasonable number of matches (approximately size/50)
	minExpected := int64(dataSize / 60) // Allow some variance
	maxExpectedMatches := int64(dataSize / 40)
	if count < minExpected || count > maxExpectedMatches {
		t.Errorf("Match count %d outside expected range [%d, %d]",
			count, minExpected, maxExpectedMatches)
	}
}

// TestMemoryDoesNotGrow verifies memory usage is constant across input sizes.
func TestMemoryDoesNotGrow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory growth test in short mode")
	}

	sizes := []int64{
		1 << 18, // 256KB
		1 << 20, // 1MB
		1 << 22, // 4MB
	}

	var allocations []uint64

	for _, size := range sizes {
		gen := NewPatternedReader("2024-01-15", 50, size)

		runtime.GC()
		var startMem runtime.MemStats
		runtime.ReadMemStats(&startMem)
		startAlloc := startMem.TotalAlloc

		// Use default config
		_, err := testdata.CompiledDatePattern.FindReaderCount(gen, stream.Config{})
		if err != nil {
			t.Fatalf("FindReaderCount error for size %d: %v", size, err)
		}

		runtime.GC()
		var endMem runtime.MemStats
		runtime.ReadMemStats(&endMem)
		alloc := endMem.TotalAlloc - startAlloc
		allocations = append(allocations, alloc)

		t.Logf("Size %d: allocated %d bytes", size, alloc)
	}

	// Memory should be roughly constant regardless of input size
	// Allow 4x variance due to GC timing
	baseline := allocations[0]
	maxAllowed := baseline * 4

	for i, alloc := range allocations {
		if alloc > maxAllowed {
			t.Errorf("Size %d: allocated %d bytes, expected < %d (baseline %d)",
				sizes[i], alloc, maxAllowed, baseline)
		}
	}
}
