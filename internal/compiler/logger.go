package compiler

import (
	"fmt"
	"io"
	"os"
)

// Logger provides verbose output for analysis decisions during compilation.
type Logger struct {
	enabled bool
	out     io.Writer
}

// NewLogger creates a new logger instance.
func NewLogger(enabled bool) *Logger {
	return &Logger{
		enabled: enabled,
		out:     os.Stderr,
	}
}

// SetOutput sets the output writer for the logger.
func (l *Logger) SetOutput(w io.Writer) {
	l.out = w
}

// Log prints a formatted message if verbose mode is enabled.
func (l *Logger) Log(format string, args ...interface{}) {
	if l.enabled {
		fmt.Fprintf(l.out, "[regengo] "+format+"\n", args...)
	}
}

// Section prints a section header if verbose mode is enabled.
func (l *Logger) Section(name string) {
	if l.enabled {
		fmt.Fprintf(l.out, "\n[regengo] === %s ===\n", name)
	}
}

// Enabled returns whether the logger is enabled.
func (l *Logger) Enabled() bool {
	return l.enabled
}
