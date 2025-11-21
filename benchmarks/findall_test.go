package benchmarks_test

import (
	"regexp"
	"testing"

	"github.com/KromDaniel/regengo/benchmarks/generated"
)

func TestDateCaptureFindAllString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		n     int
		want  int
	}{
		{
			name:  "multiple dates unlimited",
			input: "Dates: 2024-01-15 and 2024-12-25 and 2025-06-30",
			n:     -1,
			want:  3,
		},
		{
			name:  "limited to 2 dates",
			input: "2024-01-01 2024-02-02 2024-03-03",
			n:     2,
			want:  2,
		},
		{
			name:  "no dates",
			input: "no dates here",
			n:     -1,
			want:  0,
		},
		{
			name:  "n=0 returns nil",
			input: "2024-01-15",
			n:     0,
			want:  0,
		},
		{
			name:  "single date",
			input: "The date is 2024-01-15 only",
			n:     -1,
			want:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := generated.DateCapture{}.FindAllString(tt.input, tt.n)

			got := len(matches)
			if got != tt.want {
				t.Errorf("got %d matches, want %d", got, tt.want)
			}

			// For n=0, verify nil return
			if tt.n == 0 && matches != nil {
				t.Errorf("expected nil for n=0, got %v", matches)
			}

			// Verify each match has proper captures
			for i, match := range matches {
				if match.Year == "" || match.Month == "" || match.Day == "" {
					t.Errorf("match %d has empty captures: %+v", i, match)
				}
			}

			t.Logf("Found %d matches for input %q", got, tt.input)
		})
	}
}

func TestDateCaptureFindAllBytes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		n     int
		want  int
	}{
		{
			name:  "multiple dates",
			input: "2024-01-15 and 2024-12-25",
			n:     -1,
			want:  2,
		},
		{
			name:  "limited",
			input: "2024-01-01 2024-02-02 2024-03-03",
			n:     1,
			want:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := generated.DateCapture{}.FindAllBytes([]byte(tt.input), tt.n)

			got := len(matches)
			if got != tt.want {
				t.Errorf("got %d matches, want %d", got, tt.want)
			}
		})
	}
}

// TestFindAllVsStdlib compares our FindAll with stdlib
func TestFindAllVsStdlib(t *testing.T) {
	pattern := `(\d{4})-(\d{2})-(\d{2})`
	stdlibRe := regexp.MustCompile(pattern)

	tests := []struct {
		input string
		n     int
	}{
		{
			input: "2024-01-15 and 2024-12-25 and 2025-06-30",
			n:     -1,
		},
		{
			input: "2024-01-01 2024-02-02 2024-03-03 2024-04-04",
			n:     2,
		},
		{
			input: "no dates here",
			n:     -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			// Get stdlib matches
			stdlibMatches := stdlibRe.FindAllStringSubmatch(tt.input, tt.n)

			// Get our matches
			ourMatches := generated.DateCapture{}.FindAllString(tt.input, tt.n)

			// Compare counts
			stdlibCount := len(stdlibMatches)
			ourCount := len(ourMatches)

			if stdlibCount != ourCount {
				t.Errorf("stdlib found %d matches, we found %d", stdlibCount, ourCount)
			}

			// Compare each match
			for i := 0; i < stdlibCount && i < ourCount; i++ {
				stdlibMatch := stdlibMatches[i]
				ourMatch := ourMatches[i]

				// Compare full match
				if stdlibMatch[0] != ourMatch.Match {
					t.Errorf("match %d: stdlib full=%q, ours=%q", i, stdlibMatch[0], ourMatch.Match)
				}

				// Compare captures
				if len(stdlibMatch) >= 4 {
					if stdlibMatch[1] != ourMatch.Year {
						t.Errorf("match %d: stdlib year=%q, ours=%q", i, stdlibMatch[1], ourMatch.Year)
					}
					if stdlibMatch[2] != ourMatch.Month {
						t.Errorf("match %d: stdlib month=%q, ours=%q", i, stdlibMatch[2], ourMatch.Month)
					}
					if stdlibMatch[3] != ourMatch.Day {
						t.Errorf("match %d: stdlib day=%q, ours=%q", i, stdlibMatch[3], ourMatch.Day)
					}
				}
			}

			t.Logf("✓ Both found %d matches", stdlibCount)
		})
	}
}
