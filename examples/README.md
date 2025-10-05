# Regengo Examples

This directory contains examples of using Regengo to generate optimized regex matchers.

## Running Examples

```bash
# Generate example matchers
go run examples/main.go

# Or use make
make example
```

This will generate optimized matchers in `examples/generated/`:

- `Email.go` - Email validation
- `URL.go` - URL matching
- `IPv4.go` - IPv4 address validation

## Using Generated Code

After generation, you can import and use the matchers:

```go
package main

import (
    "fmt"
    "github.com/yourproject/examples/generated"
)

func main() {
    email := "test@example.com"
    if generated.EmailMatchString(email) {
        fmt.Println("Valid email!")
    }

    url := "https://example.com"
    if generated.URLMatchString(url) {
        fmt.Println("Valid URL!")
    }
}
```

## Custom Patterns

Create your own patterns by modifying `main.go`:

```go
examples := []struct {
    name    string
    pattern string
}{
    {
        name:    "PhoneNumber",
        pattern: `\d{3}-\d{3}-\d{4}`,
    },
    {
        name:    "Hashtag",
        pattern: `#\w+`,
    },
}
```

Then run:

```bash
go run examples/main.go
```

## Benchmarking

To compare performance with standard `regexp`:

```bash
cd examples/generated
go test -bench=. -benchmem
```

## Common Patterns

### Email

```
Pattern: [\w\.+-]+@[\w\.-]+\.[\w\.-]+
Matches: user@example.com, test.user+tag@domain.co.uk
```

### URL

```
Pattern: https?://[^\s]+
Matches: http://example.com, https://example.com/path
```

### IPv4

```
Pattern: \d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}
Matches: 192.168.1.1, 10.0.0.1
Note: Does not validate ranges (0-255)
```

### Phone Number (US)

```
Pattern: \d{3}-\d{3}-\d{4}
Matches: 123-456-7890
```

### Hex Color

```
Pattern: #[0-9a-fA-F]{6}
Matches: #FF5733, #abc123
```

## Tips

1. **Test First**: Verify your pattern with Go's `regexp` package before generating
2. **Keep It Simple**: Simpler patterns generate more efficient code
3. **Benchmark**: Always benchmark against `regexp` to ensure improvement
4. **Document**: Add comments explaining complex patterns
