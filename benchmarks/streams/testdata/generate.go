// Package testdata contains pre-generated regex patterns for streaming benchmarks.
//
// Run `go generate ./...` from this directory to regenerate patterns.
package testdata

//go:generate go run github.com/KromDaniel/regengo/cmd/regengo -pattern (?P<user>[\w.+-]+)@(?P<domain>[\w.-]+\.\w+) -name EmailPattern -output email_pattern.go -package testdata -no-test
//go:generate go run github.com/KromDaniel/regengo/cmd/regengo -pattern (\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}) -name IPv4Pattern -output ip_pattern.go -package testdata -no-test
