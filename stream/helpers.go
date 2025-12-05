package stream

import (
	"bytes"
	"io"
)

// LineFilter returns an io.Reader that only outputs lines matching the predicate.
// Lines are delimited by '\n'. The newline character is included in the line
// passed to the predicate and in the output.
//
// Example - keep only lines containing "ERROR":
//
//	r := stream.LineFilter(input, func(line []byte) bool {
//	    return bytes.Contains(line, []byte("ERROR"))
//	})
//	io.Copy(os.Stdout, r)
func LineFilter(r io.Reader, pred func(line []byte) bool) io.Reader {
	return &lineFilterReader{
		source: r,
		pred:   pred,
		buf:    make([]byte, 0, 4096),
	}
}

// LineTransform returns an io.Reader that transforms each line using the given function.
// Lines are delimited by '\n'. The newline character is included in the line
// passed to the function. The function should return the transformed line
// (including newline if desired).
//
// Example - prefix each line with timestamp:
//
//	r := stream.LineTransform(input, func(line []byte) []byte {
//	    return append([]byte(time.Now().Format(time.RFC3339)+" "), line...)
//	})
//	io.Copy(os.Stdout, r)
func LineTransform(r io.Reader, fn func(line []byte) []byte) io.Reader {
	return &lineTransformReader{
		source: r,
		fn:     fn,
		buf:    make([]byte, 0, 4096),
	}
}

// lineFilterReader implements io.Reader for LineFilter.
type lineFilterReader struct {
	source io.Reader
	pred   func(line []byte) bool

	// Input buffer
	buf       []byte
	bufStart  int
	bufEnd    int
	sourceEOF bool

	// Output buffer (lines that passed the filter)
	output      []byte
	outputStart int

	err error
}

func (r *lineFilterReader) Read(p []byte) (n int, err error) {
	// Return buffered output first
	if r.outputStart < len(r.output) {
		n = copy(p, r.output[r.outputStart:])
		r.outputStart += n
		if r.outputStart == len(r.output) {
			r.output = r.output[:0]
			r.outputStart = 0
		}
		return n, nil
	}

	// Check for previous error
	if r.err != nil {
		return 0, r.err
	}

	// Process more input until we have output or hit EOF
	for len(r.output) == 0 {
		if err := r.processMore(); err != nil {
			if len(r.output) > 0 {
				// Return output first, save error for next call
				n = copy(p, r.output[r.outputStart:])
				r.outputStart += n
				if r.outputStart == len(r.output) {
					r.output = r.output[:0]
					r.outputStart = 0
				}
				r.err = err
				return n, nil
			}
			return 0, err
		}
	}

	// Return buffered output
	n = copy(p, r.output[r.outputStart:])
	r.outputStart += n
	if r.outputStart == len(r.output) {
		r.output = r.output[:0]
		r.outputStart = 0
	}
	return n, nil
}

func (r *lineFilterReader) processMore() error {
	// Compact buffer if needed
	if r.bufStart > 0 {
		remaining := r.bufEnd - r.bufStart
		if remaining > 0 {
			copy(r.buf[:remaining], r.buf[r.bufStart:r.bufEnd])
		}
		r.buf = r.buf[:remaining]
		r.bufStart = 0
		r.bufEnd = remaining
	}

	// Read more data if not at EOF
	if !r.sourceEOF {
		// Grow buffer if needed
		if cap(r.buf)-r.bufEnd < 4096 {
			newBuf := make([]byte, r.bufEnd, r.bufEnd+4096)
			copy(newBuf, r.buf[:r.bufEnd])
			r.buf = newBuf
		}

		n, err := r.source.Read(r.buf[r.bufEnd:cap(r.buf)])
		r.buf = r.buf[:r.bufEnd+n]
		r.bufEnd += n

		if err != nil {
			if err == io.EOF {
				r.sourceEOF = true
			} else {
				return err
			}
		}
	}

	// No data to process
	if r.bufEnd == 0 {
		return io.EOF
	}

	// Process complete lines
	data := r.buf[r.bufStart:r.bufEnd]
	for {
		idx := bytes.IndexByte(data, '\n')
		if idx < 0 {
			// No complete line
			if r.sourceEOF && len(data) > 0 {
				// Last line without newline
				if r.pred(data) {
					r.output = append(r.output, data...)
				}
				r.bufStart = r.bufEnd
				return io.EOF
			}
			break
		}

		// Found a complete line (including newline)
		line := data[:idx+1]
		if r.pred(line) {
			r.output = append(r.output, line...)
		}

		data = data[idx+1:]
		r.bufStart += idx + 1
	}

	if r.sourceEOF && r.bufStart >= r.bufEnd {
		return io.EOF
	}

	return nil
}

// lineTransformReader implements io.Reader for LineTransform.
type lineTransformReader struct {
	source io.Reader
	fn     func(line []byte) []byte

	// Input buffer
	buf       []byte
	bufStart  int
	bufEnd    int
	sourceEOF bool

	// Output buffer (transformed lines)
	output      []byte
	outputStart int

	err error
}

func (r *lineTransformReader) Read(p []byte) (n int, err error) {
	// Return buffered output first
	if r.outputStart < len(r.output) {
		n = copy(p, r.output[r.outputStart:])
		r.outputStart += n
		if r.outputStart == len(r.output) {
			r.output = r.output[:0]
			r.outputStart = 0
		}
		return n, nil
	}

	// Check for previous error
	if r.err != nil {
		return 0, r.err
	}

	// Process more input until we have output or hit EOF
	for len(r.output) == 0 {
		if err := r.processMore(); err != nil {
			if len(r.output) > 0 {
				n = copy(p, r.output[r.outputStart:])
				r.outputStart += n
				if r.outputStart == len(r.output) {
					r.output = r.output[:0]
					r.outputStart = 0
				}
				r.err = err
				return n, nil
			}
			return 0, err
		}
	}

	// Return buffered output
	n = copy(p, r.output[r.outputStart:])
	r.outputStart += n
	if r.outputStart == len(r.output) {
		r.output = r.output[:0]
		r.outputStart = 0
	}
	return n, nil
}

func (r *lineTransformReader) processMore() error {
	// Compact buffer if needed
	if r.bufStart > 0 {
		remaining := r.bufEnd - r.bufStart
		if remaining > 0 {
			copy(r.buf[:remaining], r.buf[r.bufStart:r.bufEnd])
		}
		r.buf = r.buf[:remaining]
		r.bufStart = 0
		r.bufEnd = remaining
	}

	// Read more data if not at EOF
	if !r.sourceEOF {
		// Grow buffer if needed
		if cap(r.buf)-r.bufEnd < 4096 {
			newBuf := make([]byte, r.bufEnd, r.bufEnd+4096)
			copy(newBuf, r.buf[:r.bufEnd])
			r.buf = newBuf
		}

		n, err := r.source.Read(r.buf[r.bufEnd:cap(r.buf)])
		r.buf = r.buf[:r.bufEnd+n]
		r.bufEnd += n

		if err != nil {
			if err == io.EOF {
				r.sourceEOF = true
			} else {
				return err
			}
		}
	}

	// No data to process
	if r.bufEnd == 0 {
		return io.EOF
	}

	// Process complete lines
	data := r.buf[r.bufStart:r.bufEnd]
	for {
		idx := bytes.IndexByte(data, '\n')
		if idx < 0 {
			// No complete line
			if r.sourceEOF && len(data) > 0 {
				// Last line without newline
				transformed := r.fn(data)
				r.output = append(r.output, transformed...)
				r.bufStart = r.bufEnd
				return io.EOF
			}
			break
		}

		// Found a complete line (including newline)
		line := data[:idx+1]
		transformed := r.fn(line)
		r.output = append(r.output, transformed...)

		data = data[idx+1:]
		r.bufStart += idx + 1
	}

	if r.sourceEOF && r.bufStart >= r.bufEnd {
		return io.EOF
	}

	return nil
}
