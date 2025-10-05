package benchmarks_test

import (
	"regexp"
	"testing"

	"github.com/KromDaniel/regengo/examples/generated"
)

var emailInputs = []string{
	"test@example.com",
	"user.name+tag@domain.co.uk",
	"a@b.c",
	"invalid@",
	"not-an-email",
	"",
}

var urlInputs = []string{
	"http://example.com",
	"https://example.com/path/to/resource?query=value",
	"http://a",
	"not-a-url",
	"",
}

var ipv4Inputs = []string{
	"192.168.1.1",
	"10.0.0.1",
	"255.255.255.255",
	"999.999.999.999",
	"not-an-ip",
	"",
}

// Email benchmarks - Standard regexp
func BenchmarkEmailStdRegexp(b *testing.B) {
	re := regexp.MustCompile(`[\w\.+-]+@[\w\.-]+\.[\w\.-]+`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range emailInputs {
			re.MatchString(input)
		}
	}
}

// Email benchmarks - Regengo without pool
func BenchmarkEmailRegengo(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range emailInputs {
			generated.EmailMatchString(input)
		}
	}
}

// Email benchmarks - Regengo with pool
func BenchmarkEmailRegengoPooled(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range emailInputs {
			EmailPooledMatchString(input)
		}
	}
}

// URL benchmarks
func BenchmarkURLStdRegexp(b *testing.B) {
	re := regexp.MustCompile(`https?://[^\s]+`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range urlInputs {
			re.MatchString(input)
		}
	}
}

func BenchmarkURLRegengo(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range urlInputs {
			generated.URLMatchString(input)
		}
	}
}

func BenchmarkURLRegengoPooled(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range urlInputs {
			URLPooledMatchString(input)
		}
	}
}

// IPv4 benchmarks
func BenchmarkIPv4StdRegexp(b *testing.B) {
	re := regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range ipv4Inputs {
			re.MatchString(input)
		}
	}
}

func BenchmarkIPv4Regengo(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range ipv4Inputs {
			generated.IPv4MatchString(input)
		}
	}
}

func BenchmarkIPv4RegengoPooled(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range ipv4Inputs {
			IPv4PooledMatchString(input)
		}
	}
}
