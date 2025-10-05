package bytes_view_test

import (
	"regexp"
	"testing"
)

var testURL = []byte("https://api.example.com:8080/v1/users")
var urlRegexp = regexp.MustCompile(`(?P<protocol>https?)://(?P<host>[\w\.-]+)(?::(?P<port>\d+))?`)

func BenchmarkStandardRegexp(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		matches := urlRegexp.FindSubmatch(testURL)
		if matches == nil {
			b.Fatal("no match")
		}
		_ = matches[1]
		_ = matches[2]
		_ = matches[3]
	}
}

func BenchmarkRegengoStringFields(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		match, ok := URLStringFindBytes(testURL)
		if !ok {
			b.Fatal("no match")
		}
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
		_ = match.Protocol
		_ = match.Host
		_ = match.Port
	}
}
