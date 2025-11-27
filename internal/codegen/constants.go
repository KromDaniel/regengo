// Package codegen provides code generation helpers and constants.
package codegen

import "fmt"

// Variable names used in generated code
const (
	InputName           = "input"
	InputLenName        = "l"
	OffsetName          = "offset"
	StackName           = "stack"
	CapturesName        = "captures"
	NextInstructionName = "nextInstruction"
	StepSelectName      = "StepSelect"
	TryFallbackName     = "TryFallback"
	VisitedName         = "visited"
	NumInstName         = "numInst"
)

// InstructionName returns the label name for an instruction.
func InstructionName(id uint32) string {
	return fmt.Sprintf("Ins%d", id)
}

// LowerFirst converts the first character of a string to lowercase.
func LowerFirst(s string) string {
	if s == "" {
		return s
	}
	return string(s[0]|0x20) + s[1:]
}

// UpperFirst converts the first character of a string to uppercase.
func UpperFirst(s string) string {
	if s == "" {
		return s
	}
	return string(s[0]&^0x20) + s[1:]
}
