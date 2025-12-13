package compiler

// Capture group constants
const (
	// ImplicitCaptureCount is the number of implicit capture groups in a regex.
	// Group 0 captures the full match, group 1 is unused/internal.
	// User-defined capture groups start at index 2.
	// Usage: prog.NumCap > ImplicitCaptureCount means user-defined groups exist.
	ImplicitCaptureCount = 2
)

// Stack entry type constants for backtracking.
// The stack entry is a [3]int where:
//   - [0] = offset or old capture value
//   - [1] = next instruction or capture index
//   - [2] = entry type (one of the constants below)
const (
	// StackEntryAlt indicates a standard alternative branch backtrack point.
	// No capture state needs to be restored.
	StackEntryAlt = 0

	// StackEntryCheckpoint indicates a backtrack point with a capture checkpoint.
	// When backtracking to this point, restore captures from the captureStack.
	StackEntryCheckpoint = 1

	// StackEntryPerCaptureRestore indicates a per-capture restore point.
	// Used in per-capture checkpointing mode where individual captures are saved.
	StackEntryPerCaptureRestore = 2
)

// ASCII boundary constants
const (
	// MaxASCIIRune is the exclusive upper bound for ASCII characters.
	// Runes with value < MaxASCIIRune are ASCII.
	MaxASCIIRune = 128
)

// Stack entry size constants for backtracking.
const (
	// StackEntrySizeDefault is the size of a stack entry without captures.
	// Entry format: [offset, nextInstruction]
	StackEntrySizeDefault = 2

	// StackEntrySizeWithCaptures is the size of a stack entry with captures.
	// Entry format: [offset, nextInstruction, entryType]
	StackEntrySizeWithCaptures = 3
)
