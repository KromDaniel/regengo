package stream

import (
	"context"
	"io"
	"sync"
)

// TransformFunc is the type signature for the match transformation callback.
// It receives:
//   - match: the matched bytes
//   - emit: a function to emit output bytes (can be called multiple times for 1-to-N expansion)
//
// If emit is not called, the match is dropped (filter behavior).
// Non-matching segments are automatically passed through to output.
type TransformFunc func(match []byte, emit func([]byte))

// Processor is the pattern-specific function that processes input data.
// It should:
//  1. Find all matches in the data
//  2. For each match, call onMatch with the match bytes
//  3. Emit non-matching segments via emitNonMatch
//  4. Return the number of bytes that were fully processed (safe to discard)
//
// The processor must handle the case where a match might span the current data boundary.
// In such cases, it should return a smaller "processed" count to keep potential partial matches.
type Processor func(data []byte, isEOF bool, onMatch TransformFunc, emitNonMatch func([]byte)) (processed int)

// TransformConfig extends Config with transform-specific options.
type TransformConfig struct {
	Config

	// MaxOutputBuffer limits the internal output buffer size.
	// If exceeded, Read will block until the buffer is drained.
	// Default: 0 (unlimited, grows as needed).
	MaxOutputBuffer int

	// Context for cancellation support.
	// Default: nil (no cancellation).
	Context context.Context
}

// DefaultTransformConfig returns a TransformConfig with sensible defaults.
func DefaultTransformConfig() TransformConfig {
	return TransformConfig{
		Config:          DefaultConfig(),
		MaxOutputBuffer: 0,
		Context:         nil,
	}
}

// Buffer pools for reuse across transformers
var (
	// inputBufPool holds input buffers (default 64KB)
	inputBufPool = sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 64*1024)
			return &buf
		},
	}

	// outputBufPool holds output buffer slices
	outputBufPool = sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 0, 64*1024)
			return &buf
		},
	}
)

// Transformer wraps a source io.Reader and applies transformations to matches.
// It implements io.Reader, allowing standard Go composition via io.Copy, etc.
//
// The transformation is lazy - processing happens only when Read is called.
type Transformer struct {
	source    io.Reader
	cfg       TransformConfig
	processor Processor
	onMatch   TransformFunc

	// Input buffering
	inputBuf   []byte
	inputStart int // start of unprocessed data
	inputEnd   int // end of valid data

	// Output buffering (ring buffer semantics)
	outputBuf   []byte
	outputStart int // start of unread output
	outputEnd   int // end of valid output

	// State
	sourceEOF bool
	err       error

	// Pool management
	usePool       bool
	inputBufPtr   *[]byte // Pointer for returning to pool
	outputBufPtr  *[]byte // Pointer for returning to pool
	poolsReturned bool
}

// NewTransformer creates a new Transformer that reads from source and applies
// the given processor and match transformation function.
//
// The processor is pattern-specific and handles finding matches.
// The onMatch function is called for each match to produce output.
func NewTransformer(
	source io.Reader,
	cfg TransformConfig,
	processor Processor,
	onMatch TransformFunc,
) *Transformer {
	return newTransformer(source, cfg, processor, onMatch, false)
}

// NewTransformerPooled creates a new Transformer using pooled buffers.
// This reduces allocations for high-throughput scenarios.
//
// IMPORTANT: You MUST call Close() when done to return buffers to the pool.
// Failure to call Close() will cause memory leaks.
func NewTransformerPooled(
	source io.Reader,
	cfg TransformConfig,
	processor Processor,
	onMatch TransformFunc,
) *Transformer {
	return newTransformer(source, cfg, processor, onMatch, true)
}

func newTransformer(
	source io.Reader,
	cfg TransformConfig,
	processor Processor,
	onMatch TransformFunc,
	usePool bool,
) *Transformer {
	// Apply defaults
	if cfg.BufferSize == 0 {
		cfg.BufferSize = 64 * 1024 // 64KB default
	}
	if cfg.MaxLeftover == 0 {
		cfg.MaxLeftover = cfg.BufferSize / 2
	}

	t := &Transformer{
		source:    source,
		cfg:       cfg,
		processor: processor,
		onMatch:   onMatch,
		usePool:   usePool,
	}

	if usePool && cfg.BufferSize == 64*1024 {
		// Use pooled buffers for default size
		t.inputBufPtr, _ = inputBufPool.Get().(*[]byte) //nolint:errcheck // type assertion always succeeds for our pool
		t.inputBuf = *t.inputBufPtr
		t.outputBufPtr, _ = outputBufPool.Get().(*[]byte) //nolint:errcheck // type assertion always succeeds for our pool
		t.outputBuf = (*t.outputBufPtr)[:0]
	} else {
		// Allocate new buffers
		t.inputBuf = make([]byte, cfg.BufferSize)
		t.outputBuf = make([]byte, 0, cfg.BufferSize)
	}

	return t
}

// Close returns pooled buffers and cleans up resources.
// For transformers created with NewTransformerPooled, this MUST be called
// to return buffers to the pool and prevent memory leaks.
// For regular transformers, Close is a no-op but safe to call.
func (t *Transformer) Close() error {
	if t.poolsReturned {
		return nil
	}
	t.poolsReturned = true

	if t.usePool && t.inputBufPtr != nil {
		// Clear and return input buffer
		for i := range *t.inputBufPtr {
			(*t.inputBufPtr)[i] = 0
		}
		inputBufPool.Put(t.inputBufPtr)
		t.inputBufPtr = nil
		t.inputBuf = nil
	}

	if t.usePool && t.outputBufPtr != nil {
		// Clear and return output buffer
		*t.outputBufPtr = (*t.outputBufPtr)[:0]
		outputBufPool.Put(t.outputBufPtr)
		t.outputBufPtr = nil
		t.outputBuf = nil
	}

	return nil
}

// Read implements io.Reader.
// It processes input from the source, applies transformations, and returns the result.
func (t *Transformer) Read(p []byte) (n int, err error) {
	// Check for context cancellation
	if t.cfg.Context != nil {
		select {
		case <-t.cfg.Context.Done():
			return 0, t.cfg.Context.Err()
		default:
		}
	}

	// If we have buffered output, return it first
	if t.outputStart < t.outputEnd {
		n = copy(p, t.outputBuf[t.outputStart:t.outputEnd])
		t.outputStart += n
		// Reset buffer if fully consumed
		if t.outputStart == t.outputEnd {
			t.outputStart = 0
			t.outputEnd = 0
			t.outputBuf = t.outputBuf[:0]
		}
		return n, nil
	}

	// If we've already hit an error, return it
	if t.err != nil {
		return 0, t.err
	}

	// Process more input until we have output or hit EOF/error
	for t.outputStart == t.outputEnd {
		err := t.processMore()
		if err != nil {
			if t.outputStart < t.outputEnd {
				// We have some output to return first
				n = copy(p, t.outputBuf[t.outputStart:t.outputEnd])
				t.outputStart += n
				if t.outputStart == t.outputEnd {
					t.outputStart = 0
					t.outputEnd = 0
					t.outputBuf = t.outputBuf[:0]
				}
				// Save error for next call
				t.err = err
				return n, nil
			}
			return 0, err
		}

		// Check context again after processing
		if t.cfg.Context != nil {
			select {
			case <-t.cfg.Context.Done():
				t.err = t.cfg.Context.Err()
				return 0, t.err
			default:
			}
		}
	}

	// Return buffered output
	n = copy(p, t.outputBuf[t.outputStart:t.outputEnd])
	t.outputStart += n
	if t.outputStart == t.outputEnd {
		t.outputStart = 0
		t.outputEnd = 0
		t.outputBuf = t.outputBuf[:0]
	}
	return n, nil
}

// processMore reads from the source and processes data.
// Returns io.EOF when all data has been processed.
func (t *Transformer) processMore() error {
	// If source is exhausted and we have no more input, we're done
	if t.sourceEOF && t.inputStart >= t.inputEnd {
		return io.EOF
	}

	// Compact input buffer if needed
	if t.inputStart > 0 {
		remaining := t.inputEnd - t.inputStart
		if remaining > 0 {
			copy(t.inputBuf[:remaining], t.inputBuf[t.inputStart:t.inputEnd])
		}
		t.inputStart = 0
		t.inputEnd = remaining
	}

	// Read more data if not at EOF
	if !t.sourceEOF {
		n, err := t.source.Read(t.inputBuf[t.inputEnd:])
		t.inputEnd += n

		if err != nil {
			if err == io.EOF {
				t.sourceEOF = true
			} else {
				return err
			}
		}
	}

	// No data to process
	if t.inputEnd == 0 {
		return io.EOF
	}

	// Get the data to process
	data := t.inputBuf[t.inputStart:t.inputEnd]

	// Process the data
	processed := t.processor(data, t.sourceEOF, t.onMatch, t.emitOutput)

	// Update input position
	t.inputStart += processed

	// Handle leftover management for non-EOF case
	if !t.sourceEOF {
		leftover := t.inputEnd - t.inputStart
		if leftover > t.cfg.MaxLeftover && t.cfg.MaxLeftover >= 0 {
			// Too much leftover - emit the excess as non-match and advance
			excess := leftover - t.cfg.MaxLeftover
			t.emitOutput(t.inputBuf[t.inputStart : t.inputStart+excess])
			t.inputStart += excess
		}
	}

	return nil
}

// emitOutput appends data to the output buffer.
func (t *Transformer) emitOutput(data []byte) {
	if len(data) == 0 {
		return
	}
	t.outputBuf = append(t.outputBuf, data...)
	t.outputEnd = len(t.outputBuf)
}

// Reset resets the transformer to read from a new source.
// This allows reuse of the transformer's buffers.
func (t *Transformer) Reset(source io.Reader) {
	t.source = source
	t.inputStart = 0
	t.inputEnd = 0
	t.outputStart = 0
	t.outputEnd = 0
	t.outputBuf = t.outputBuf[:0]
	t.sourceEOF = false
	t.err = nil
}
