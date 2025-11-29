//go:build ignore

package main

import "strings"

// CuratedCase defines a benchmark case for documentation and performance tracking
type CuratedCase struct {
	Name        string   // Readable name (e.g., "EmailCapture")
	Pattern     string   // Regex pattern
	Inputs      []string // Benchmark inputs (can use Go expressions)
	Replacers   []string // Optional: replacement templates
	Description string   // For documentation
	Category    string   // match, capture, findall, tdfa, tnfa, replace
	ForceTNFA   bool     // Force TNFA engine (for testing memoization)
}

// CuratedCases contains all curated benchmark patterns organized by category
var CuratedCases = []CuratedCase{
	// =============================================================================
	// Match (no captures) - Simple pattern matching without capture groups
	// =============================================================================
	{
		Name:    "Email",
		Pattern: `[\w\.+-]+@[\w\.-]+\.[\w\.-]+`,
		Inputs: []string{
			strings.Repeat("a", 100) + "| me@myself.com",
		},
		Category:    "match",
		Description: "Simple email matching without capture groups",
	},
	{
		Name:    "Greedy",
		Pattern: `(?:(?:a|b)|(?:k)+)*abcd`,
		Inputs: []string{
			strings.Repeat("a", 100) + "aaaaaaabcd",
		},
		Category:    "match",
		Description: "Greedy quantifier with alternation",
	},
	{
		Name:    "Lazy",
		Pattern: `(?:(?:a|b)|(?:k)+)+?abcd`,
		Inputs: []string{
			strings.Repeat("a", 100) + "aaaaaaabcd",
		},
		Category:    "match",
		Description: "Lazy quantifier with alternation",
	},

	// =============================================================================
	// Capture Groups - Patterns with named capture groups
	// =============================================================================
	{
		Name:    "EmailCapture",
		Pattern: `(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)`,
		Inputs: []string{
			"user@example.com",
			"john.doe+tag@subdomain.example.co.uk",
			"test@test.org",
		},
		Category:    "capture",
		Description: "Email pattern with named capture groups",
	},
	{
		Name:    "URLCapture",
		Pattern: `(?P<protocol>https?)://(?P<host>[\w\.-]+)(?::(?P<port>\d+))?(?P<path>/[\w\./]*)?`,
		Inputs: []string{
			"http://example.com",
			"https://api.github.com:443/repos/owner/repo",
			"http://localhost:8080/api/v1/users",
		},
		Category:    "capture",
		Description: "URL pattern with protocol, host, port, and path capture groups",
	},
	{
		Name:    "DateCapture",
		Pattern: `(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})`,
		Inputs: []string{
			"2025-10-05",
			"1999-12-31",
			"2000-01-01",
		},
		Category:    "capture",
		Description: "ISO date pattern with year, month, day capture groups",
	},

	// =============================================================================
	// FindAll - Patterns for finding multiple matches in text
	// =============================================================================
	{
		Name:    "MultiDate",
		Pattern: `(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})`,
		Inputs: []string{
			"Events: 2024-01-15, 2024-06-20, and 2024-12-25 are holidays",
			"No dates here",
			"Single date 2025-10-05 in text",
		},
		Category:    "findall",
		Description: "Find multiple dates in text",
	},
	{
		Name:    "MultiEmail",
		Pattern: `(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)`,
		Inputs: []string{
			"Contact us at support@example.com or sales@company.org for help",
			"Multiple: a@b.com, c@d.org, e@f.net",
			"No emails in this text",
		},
		Category:    "findall",
		Description: "Find multiple email addresses in text",
	},

	// =============================================================================
	// TDFA - Tagged DFA patterns (catastrophic backtracking prevention)
	// These patterns have nested quantifiers + captures which would cause exponential
	// backtracking without TDFA's O(n) guarantee.
	// =============================================================================
	{
		Name:    "TDFAPathological",
		Pattern: `(?P<outer>(?P<inner>a+)+)b`,
		Inputs: []string{
			"aaaaaaaaaaaaaaaaaaaab",           // 20 a's + b (matches)
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaab", // 30 a's + b (matches)
			strings.Repeat("a", 50) + "b",     // 50 a's + b (matches)
			strings.Repeat("a", 30),           // no match - would hang without TDFA
		},
		Category:    "tdfa",
		Description: "Classic (a+)+b pattern - O(2^n) without TDFA, O(n) with TDFA",
	},
	{
		Name:    "TDFANestedWord",
		Pattern: `(?P<words>(?P<word>\w+\s*)+)end`,
		Inputs: []string{
			"hello world end",
			"a b c d e f g h i j end",
			strings.Repeat("word ", 20) + "end",
		},
		Category:    "tdfa",
		Description: "Nested quantifiers with word boundaries - common in real patterns",
	},
	{
		Name:    "TDFAComplexURL",
		Pattern: `(?P<scheme>https?)://(?P<auth>(?P<user>[\w.-]+)(?::(?P<pass>[\w.-]+))?@)?(?P<host>[\w.-]+)(?::(?P<port>\d+))?(?P<path>/[\w./-]*)?(?:\?(?P<query>[\w=&.-]+))?`,
		Inputs: []string{
			"https://example.com",
			"http://user:pass@example.com:8080/path/to/resource?key=value&foo=bar",
			"https://api.github.com/repos/owner/repo",
		},
		Category:    "tdfa",
		Description: "Complex URL with optional components - uses character class compression",
	},
	{
		Name:    "TDFALogParser",
		Pattern: `(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})(?:\.(?P<ms>\d{3}))?(?P<tz>Z|[+-]\d{2}:\d{2})?\s+\[(?P<level>\w+)\]\s+(?P<message>.+)`,
		Inputs: []string{
			"2024-01-15T10:30:45.123Z [INFO] Server started successfully",
			"2024-01-15T10:30:45+00:00 [ERROR] Connection failed",
			"2024-01-15T10:30:45 [DEBUG] Processing request id=12345",
		},
		Category:    "tdfa",
		Description: "Log line parser with multiple optional groups",
	},
	{
		Name:    "TDFASemVer",
		Pattern: `(?P<major>\d+)\.(?P<minor>\d+)\.(?P<patch>\d+)(?:-(?P<prerelease>[\w.-]+))?(?:\+(?P<build>[\w.-]+))?`,
		Inputs: []string{
			"1.0.0",
			"2.1.3-alpha.1",
			"3.0.0-beta.2+build.123",
			"10.20.30-rc.1+20240115",
		},
		Category:    "tdfa",
		Description: "Semantic version with optional pre-release and build metadata",
	},

	// =============================================================================
	// TNFA - Thompson NFA patterns (forced memoization for testing)
	// =============================================================================
	{
		Name:    "TNFAPathological",
		Pattern: `(?P<outer>(?P<inner>a+)+)b`,
		Inputs: []string{
			"aaaaaaaaaaaaaaaaaaaab",           // 20 a's + b (matches)
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaab", // 30 a's + b (matches)
		},
		ForceTNFA:   true,
		Category:    "tnfa",
		Description: "Pathological pattern forced to use TNFA with memoization",
	},

	// =============================================================================
	// Replace - String replacement patterns with precompiled replacers
	// =============================================================================
	{
		Name:    "ReplaceEmail",
		Pattern: `(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)`,
		Inputs: []string{
			"Contact support@example.com for help",
			"Multiple: a@b.com, c@d.org, e@f.net in one line",
			strings.Repeat("user@example.com ", 50),
		},
		Replacers: []string{
			"$user@REDACTED.$tld",
			"[EMAIL]",
			"$0",
		},
		Category:    "replace",
		Description: "Email replacement with capture group references",
	},
	{
		Name:    "ReplaceDate",
		Pattern: `(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})`,
		Inputs: []string{
			"Event on 2024-01-15",
			"Range: 2024-01-01 to 2024-12-31",
			strings.Repeat("2024-06-15 ", 100),
		},
		Replacers: []string{
			"$month/$day/$year",
			"[DATE]",
			"$year",
		},
		Category:    "replace",
		Description: "Date format conversion using capture groups",
	},
	{
		Name:    "ReplaceURL",
		Pattern: `(?P<protocol>https?)://(?P<host>[\w\.-]+)(?::(?P<port>\d+))?(?P<path>/[\w\./]*)?`,
		Inputs: []string{
			"Visit https://example.com/page for info",
			"API at http://localhost:8080/api/v1/users",
			"URLs: https://a.com https://b.com https://c.com https://d.com https://e.com",
		},
		Replacers: []string{
			"$protocol://$host[REDACTED]",
			"[URL]",
			"$host",
		},
		Category:    "replace",
		Description: "URL redaction with selective capture group output",
	},
}
