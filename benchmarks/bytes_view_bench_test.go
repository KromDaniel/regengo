package benchmarks_test

import (
	"regexp"
	"testing"
)

// Benchmark comparing standard captures (string fields) vs BytesView ([]byte fields)

var testURL = []byte("https://api.example.com:8080/v1/users")

// Standard regexp for comparison
var urlRegexp = regexp.MustCompile(`(?P<protocol>https?)://(?P<host>[\w\.-]+)(?::(?P<port>\d+))?`)

func BenchmarkStandardRegexp(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		matches := urlRegexp.FindSubmatch(testURL)
		if matches == nil {
			b.Fatal("no match")
		}
		// Access all capture groups
		_ = matches[1] // protocol
		_ = matches[2] // host
		_ = matches[3] // port
	}
}

func BenchmarkRegengoStringFields(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		match, ok := URLStringFindBytes(testURL)
		if !ok {
			b.Fatal("no match")
		}
		// Access all capture groups (causes string conversions)
		_ = match.Protocol
		_ = match.Host
		_ = match.Port
	}
}

func BenchmarkRegengoBytesView(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		match, ok := URLBytesFindBytes(testURL)
		if !ok {
			b.Fatal("no match")
		}
		// Access all capture groups (zero-copy []byte slices)
		_ = match.Protocol
		_ = match.Host
		_ = match.Port
	}
}
