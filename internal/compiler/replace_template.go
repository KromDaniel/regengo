package compiler

import (
	"fmt"
	"strings"
	"unicode"
)

// TemplateSegmentType indicates the type of segment in a replacement template.
type TemplateSegmentType int

const (
	// SegmentLiteral represents literal text (no capture reference).
	SegmentLiteral TemplateSegmentType = iota
	// SegmentFullMatch represents a reference to the full match ($0).
	SegmentFullMatch
	// SegmentCaptureIndex represents a reference to a capture group by index ($1, $2, etc.).
	SegmentCaptureIndex
	// SegmentCaptureName represents a reference to a capture group by name ($name, ${name}).
	SegmentCaptureName
)

// TemplateSegment represents a parsed segment of a replacement template.
type TemplateSegment struct {
	Type         TemplateSegmentType
	Literal      string // For SegmentLiteral: the literal text
	CaptureIndex int    // For SegmentCaptureIndex: 1-based index; for SegmentFullMatch: 0
	CaptureName  string // For SegmentCaptureName: the capture group name
}

// ParsedTemplate represents a fully parsed replacement template.
type ParsedTemplate struct {
	Original string
	Segments []TemplateSegment
}

// ParseReplaceTemplate parses a replacement template string into segments.
// Template syntax:
//   - $0 or ${0}: full match
//   - $1, $2, ..., $99 or ${1}, ${2}: capture group by index
//   - $name or ${name}: capture group by name
//   - $$: literal dollar sign
//   - Everything else: literal text
//
// This function does NOT validate capture references against actual pattern groups.
// Use ValidateTemplate for that.
func ParseReplaceTemplate(template string) (*ParsedTemplate, error) {
	result := &ParsedTemplate{
		Original: template,
		Segments: make([]TemplateSegment, 0),
	}

	if len(template) == 0 {
		return result, nil
	}

	i := 0
	literalStart := 0

	for i < len(template) {
		if template[i] != '$' {
			i++
			continue
		}

		// Found a $, flush any accumulated literal
		if i > literalStart {
			result.Segments = append(result.Segments, TemplateSegment{
				Type:    SegmentLiteral,
				Literal: template[literalStart:i],
			})
		}

		// Check what follows the $
		if i+1 >= len(template) {
			// $ at end of string - treat as literal
			result.Segments = append(result.Segments, TemplateSegment{
				Type:    SegmentLiteral,
				Literal: "$",
			})
			i++
			literalStart = i
			continue
		}

		next := template[i+1]

		switch {
		case next == '$':
			// Escaped dollar: $$
			result.Segments = append(result.Segments, TemplateSegment{
				Type:    SegmentLiteral,
				Literal: "$",
			})
			i += 2
			literalStart = i

		case next == '{':
			// Explicit boundary: ${...}
			seg, consumed, err := parseBracedRef(template[i:])
			if err != nil {
				return nil, fmt.Errorf("at position %d: %w", i, err)
			}
			result.Segments = append(result.Segments, seg)
			i += consumed
			literalStart = i

		case next == '0':
			// Full match: $0
			result.Segments = append(result.Segments, TemplateSegment{
				Type: SegmentFullMatch,
			})
			i += 2
			literalStart = i

		case next >= '1' && next <= '9':
			// Indexed capture: $1, $12, etc.
			seg, consumed := parseIndexedRef(template[i:])
			result.Segments = append(result.Segments, seg)
			i += consumed
			literalStart = i

		case isNameStart(rune(next)):
			// Named capture: $name
			seg, consumed := parseNamedRef(template[i:])
			result.Segments = append(result.Segments, seg)
			i += consumed
			literalStart = i

		default:
			// Just a lone $ followed by something that's not a valid reference
			// Treat the $ as literal
			result.Segments = append(result.Segments, TemplateSegment{
				Type:    SegmentLiteral,
				Literal: "$",
			})
			i++
			literalStart = i
		}
	}

	// Flush any remaining literal
	if i > literalStart {
		result.Segments = append(result.Segments, TemplateSegment{
			Type:    SegmentLiteral,
			Literal: template[literalStart:i],
		})
	}

	return result, nil
}

// parseBracedRef parses ${...} reference starting at s[0]='$', s[1]='{'
// Returns the segment, number of characters consumed, and error if any.
func parseBracedRef(s string) (TemplateSegment, int, error) {
	// s starts with "${", find the closing "}"
	closeIdx := strings.Index(s, "}")
	if closeIdx == -1 {
		return TemplateSegment{}, 0, fmt.Errorf("unclosed ${")
	}

	// Content is between ${ and }
	content := s[2:closeIdx]

	if len(content) == 0 {
		return TemplateSegment{}, 0, fmt.Errorf("empty ${}")
	}

	// Check if it's a number (indexed reference)
	if content[0] >= '0' && content[0] <= '9' {
		// Must be all digits
		for j := 0; j < len(content); j++ {
			if content[j] < '0' || content[j] > '9' {
				return TemplateSegment{}, 0, fmt.Errorf("invalid capture reference ${%s}: mixed digits and non-digits", content)
			}
		}

		index := 0
		for j := 0; j < len(content); j++ {
			index = index*10 + int(content[j]-'0')
		}

		if index == 0 {
			return TemplateSegment{Type: SegmentFullMatch}, closeIdx + 1, nil
		}
		return TemplateSegment{
			Type:         SegmentCaptureIndex,
			CaptureIndex: index,
		}, closeIdx + 1, nil
	}

	// Must be a valid identifier (name)
	if !isValidIdentifier(content) {
		return TemplateSegment{}, 0, fmt.Errorf("invalid capture name ${%s}", content)
	}

	return TemplateSegment{
		Type:        SegmentCaptureName,
		CaptureName: content,
	}, closeIdx + 1, nil
}

// parseIndexedRef parses $N or $NN where N is 1-9 and NN is 10-99.
// s starts with '$' followed by a digit 1-9.
// Returns the segment and number of characters consumed.
func parseIndexedRef(s string) (TemplateSegment, int) {
	// s[0] = '$', s[1] = '1'-'9'
	index := int(s[1] - '0')
	consumed := 2

	// Check for second digit (support $10 through $99)
	if len(s) > 2 && s[2] >= '0' && s[2] <= '9' {
		index = index*10 + int(s[2]-'0')
		consumed = 3
	}

	return TemplateSegment{
		Type:         SegmentCaptureIndex,
		CaptureIndex: index,
	}, consumed
}

// parseNamedRef parses $name where name is a valid identifier.
// s starts with '$' followed by a valid identifier start character.
// Returns the segment and number of characters consumed.
func parseNamedRef(s string) (TemplateSegment, int) {
	// s[0] = '$', s[1] = start of identifier
	end := 2
	for end < len(s) && isNameContinue(rune(s[end])) {
		end++
	}

	name := s[1:end]
	return TemplateSegment{
		Type:        SegmentCaptureName,
		CaptureName: name,
	}, end
}

// isNameStart returns true if r can start an identifier (letter or underscore).
func isNameStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

// isNameContinue returns true if r can continue an identifier (letter, digit, or underscore).
func isNameContinue(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// isValidIdentifier returns true if s is a valid identifier.
func isValidIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}

	for i, r := range s {
		if i == 0 {
			if !isNameStart(r) {
				return false
			}
		} else {
			if !isNameContinue(r) {
				return false
			}
		}
	}
	return true
}

// ValidateTemplate checks that all capture references in the template
// exist in the given pattern's capture groups.
//
// Parameters:
//   - parsed: the parsed template to validate
//   - captureNames: slice of capture group names (index 0 = group 1, empty string = unnamed)
//   - numCaptures: total number of capture groups (not including full match)
//
// Returns an error if any reference is invalid, with a descriptive message.
func ValidateTemplate(parsed *ParsedTemplate, captureNames []string, numCaptures int) error {
	for i, seg := range parsed.Segments {
		switch seg.Type {
		case SegmentLiteral, SegmentFullMatch:
			// Always valid
			continue

		case SegmentCaptureIndex:
			if seg.CaptureIndex < 1 || seg.CaptureIndex > numCaptures {
				return fmt.Errorf("segment %d: capture group %d does not exist (pattern has %d capture groups)",
					i, seg.CaptureIndex, numCaptures)
			}

		case SegmentCaptureName:
			found := false
			for _, name := range captureNames {
				if name == seg.CaptureName {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("segment %d: capture group %q does not exist in pattern",
					i, seg.CaptureName)
			}
		}
	}

	return nil
}

// ResolveTemplate converts named references to indices for code generation.
// Returns a new ParsedTemplate with all SegmentCaptureName converted to SegmentCaptureIndex.
//
// Parameters:
//   - parsed: the parsed template to resolve
//   - captureNames: slice of capture group names (index 0 = group 1, empty string = unnamed)
//
// Returns an error if any named reference doesn't exist in captureNames.
func ResolveTemplate(parsed *ParsedTemplate, captureNames []string) (*ParsedTemplate, error) {
	resolved := &ParsedTemplate{
		Original: parsed.Original,
		Segments: make([]TemplateSegment, len(parsed.Segments)),
	}

	for i, seg := range parsed.Segments {
		if seg.Type == SegmentCaptureName {
			// Find the index of this name
			index := -1
			for j, name := range captureNames {
				if name == seg.CaptureName {
					index = j + 1 // Convert to 1-based index
					break
				}
			}
			if index == -1 {
				return nil, fmt.Errorf("segment %d: capture group %q does not exist in pattern",
					i, seg.CaptureName)
			}
			resolved.Segments[i] = TemplateSegment{
				Type:         SegmentCaptureIndex,
				CaptureIndex: index,
			}
		} else {
			// Copy as-is
			resolved.Segments[i] = seg
		}
	}

	return resolved, nil
}
