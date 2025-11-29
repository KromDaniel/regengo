// Package testdata contains pre-generated regex patterns for streaming tests.
//
// Run `go generate ./...` from the e2e/streaming directory to regenerate patterns.
package testdata

//go:generate go run ../../../cmd/regengo/main.go -pattern (\d{4}-\d{2}-\d{2}) -name DatePattern -output date_pattern.go -package testdata -no-test
//go:generate go run ../../../cmd/regengo/main.go -pattern ([\w.+-]+@[\w.-]+\.\w+) -name EmailPattern -output email_pattern.go -package testdata -no-test
//go:generate go run ../../../cmd/regengo/main.go -pattern (\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}) -name IPv4Pattern -output ipv4_pattern.go -package testdata -no-test
//go:generate go run ../../../cmd/regengo/main.go -pattern (\d+) -name DigitsPattern -output digits_pattern.go -package testdata -no-test
