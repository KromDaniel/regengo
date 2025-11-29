// Package replace provides runtime template parsing for regex replacement operations.
package replace

import (
	"fmt"
	"strings"
	"unicode"
)

// SegmentType indicates the type of segment in a replacement template.
type SegmentType int

const (
	// SegmentLiteral represents literal text (no capture reference).
	SegmentLiteral SegmentType = iota
	// SegmentFullMatch represents a reference to the full match ($0).
	SegmentFullMatch
	// SegmentCaptureIndex represents a reference to a capture group by index ($1, $2, etc.).
	SegmentCaptureIndex
	// SegmentCaptureName represents a reference to a capture group by name ($name, ${name}).
	SegmentCaptureName
)

// Segment represents a parsed segment of a replacement template.
type Segment struct {
	Type         SegmentType
	Literal      string // For SegmentLiteral: the literal text
	CaptureIndex int    // For SegmentCaptureIndex: 1-based index; for SegmentFullMatch: 0
	CaptureName  string // For SegmentCaptureName: the capture group name
}

// Template represents a fully parsed replacement template.
type Template struct {
	Original string
	Segments []Segment
}

// Parse parses a replacement template string into segments.
// Template syntax:
//   - $0 or ${0}: full match
//   - $1, $2, ..., $99 or ${1}, ${2}: capture group by index
//   - $name or ${name}: capture group by name
//   - $$: literal dollar sign
//   - Everything else: literal text
func Parse(template string) (*Template, error) {
	result := &Template{
		Original: template,
		Segments: make([]Segment, 0),
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
			result.Segments = append(result.Segments, Segment{
				Type:    SegmentLiteral,
				Literal: template[literalStart:i],
			})
		}

		// Check what follows the $
		if i+1 >= len(template) {
			// $ at end of string - treat as literal
			result.Segments = append(result.Segments, Segment{
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
			result.Segments = append(result.Segments, Segment{
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
			result.Segments = append(result.Segments, Segment{
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
			result.Segments = append(result.Segments, Segment{
				Type:    SegmentLiteral,
				Literal: "$",
			})
			i++
			literalStart = i
		}
	}

	// Flush any remaining literal
	if i > literalStart {
		result.Segments = append(result.Segments, Segment{
			Type:    SegmentLiteral,
			Literal: template[literalStart:i],
		})
	}

	return result, nil
}

// parseBracedRef parses ${...} reference starting at s[0]='$', s[1]='{'
func parseBracedRef(s string) (Segment, int, error) {
	closeIdx := strings.Index(s, "}")
	if closeIdx == -1 {
		return Segment{}, 0, fmt.Errorf("unclosed ${")
	}

	content := s[2:closeIdx]

	if len(content) == 0 {
		return Segment{}, 0, fmt.Errorf("empty ${}")
	}

	// Check if it's a number (indexed reference)
	if content[0] >= '0' && content[0] <= '9' {
		for j := 0; j < len(content); j++ {
			if content[j] < '0' || content[j] > '9' {
				return Segment{}, 0, fmt.Errorf("invalid capture reference ${%s}: mixed digits and non-digits", content)
			}
		}

		index := 0
		for j := 0; j < len(content); j++ {
			index = index*10 + int(content[j]-'0')
		}

		if index == 0 {
			return Segment{Type: SegmentFullMatch}, closeIdx + 1, nil
		}
		return Segment{
			Type:         SegmentCaptureIndex,
			CaptureIndex: index,
		}, closeIdx + 1, nil
	}

	// Must be a valid identifier (name)
	if !isValidIdentifier(content) {
		return Segment{}, 0, fmt.Errorf("invalid capture name ${%s}", content)
	}

	return Segment{
		Type:        SegmentCaptureName,
		CaptureName: content,
	}, closeIdx + 1, nil
}

// parseIndexedRef parses $N or $NN where N is 1-9 and NN is 10-99.
func parseIndexedRef(s string) (Segment, int) {
	index := int(s[1] - '0')
	consumed := 2

	// Check for second digit (support $10 through $99)
	if len(s) > 2 && s[2] >= '0' && s[2] <= '9' {
		index = index*10 + int(s[2]-'0')
		consumed = 3
	}

	return Segment{
		Type:         SegmentCaptureIndex,
		CaptureIndex: index,
	}, consumed
}

// parseNamedRef parses $name where name is a valid identifier.
func parseNamedRef(s string) (Segment, int) {
	end := 2
	for end < len(s) && isNameContinue(rune(s[end])) {
		end++
	}

	name := s[1:end]
	return Segment{
		Type:        SegmentCaptureName,
		CaptureName: name,
	}, end
}

func isNameStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isNameContinue(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

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
