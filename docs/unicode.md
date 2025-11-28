# Unicode & Multibyte Support

Regengo fully supports Unicode character classes and multibyte UTF-8 patterns.

## Supported Patterns

| Pattern | Description | Example Match |
|---------|-------------|---------------|
| `\p{L}` | Any Unicode letter | `café`, `日本語`, `שלום` |
| `\p{Greek}` | Greek script | `αβγδ` |
| `[α-ω]` | Unicode range | `αβγ` |
| `[\p{L}\p{N}]` | Letters and numbers | `abc123日本` |
| `\p{Sc}` | Currency symbols | `$`, `€`, `¥` |
| `\p{Emoji}` | Emoji characters | `😀`, `🎉` |

## Performance Characteristics

Regengo uses **compile-time detection** to choose the optimal path:

| Pattern Type | Code Path | Performance |
|--------------|-----------|-------------|
| ASCII-only (`[a-z]`, `\d`, `\w`) | 256-bit bitmap, O(1) lookup | **Fastest** |
| Unicode-only (`[α-ω]`, `\p{Greek}`) | UTF-8 decode + range check | ~5-10ns overhead per char |
| Mixed (`[a-zα-ω]`, `\p{L}`) | ASCII fast-path + Unicode fallback | Best of both |

## Basic Example

```bash
regengo -pattern '\p{L}+' -name UnicodeWord -output unicode.go
```

```go
// Matches any sequence of Unicode letters
CompiledUnicodeWord.MatchString("hello")    // true
CompiledUnicodeWord.MatchString("日本語")    // true
CompiledUnicodeWord.MatchString("café")     // true
CompiledUnicodeWord.MatchString("123")      // false
```

## Unicode Categories

### General Categories

| Category | Description | Example |
|----------|-------------|---------|
| `\p{L}` | Letters | `a`, `日`, `α` |
| `\p{Lu}` | Uppercase letters | `A`, `Α` |
| `\p{Ll}` | Lowercase letters | `a`, `α` |
| `\p{N}` | Numbers | `1`, `①`, `٣` |
| `\p{Nd}` | Decimal digits | `0-9` |
| `\p{P}` | Punctuation | `.`, `,`, `!` |
| `\p{S}` | Symbols | `$`, `+`, `©` |
| `\p{Z}` | Separators | spaces, line separators |

### Scripts

| Script | Pattern | Example |
|--------|---------|---------|
| Greek | `\p{Greek}` | `αβγδ` |
| Cyrillic | `\p{Cyrillic}` | `абвг` |
| Han | `\p{Han}` | `漢字` |
| Hiragana | `\p{Hiragana}` | `ひらがな` |
| Arabic | `\p{Arabic}` | `العربية` |
| Hebrew | `\p{Hebrew}` | `עברית` |

## Unicode Ranges

You can use Unicode characters directly in character classes:

```go
// Greek letters alpha through omega
pattern := `[α-ω]+`

// Mixed ASCII and Unicode
pattern := `[a-zA-Zα-ωА-я]+`
```

## Word Boundaries

The `\b` word boundary works correctly with Unicode:

```go
pattern := `\b\p{L}+\b`

// Matches: "café" in "I love café culture"
// Matches: "日本語" in "日本語 is Japanese"
```

## Case Sensitivity

Unicode case folding is supported:

```go
pattern := `(?i)café`

// Matches: "CAFÉ", "Café", "café"
```

## Normalization

Regengo matches bytes, not normalized Unicode. For patterns that should match different Unicode normalizations, normalize input first:

```go
import "golang.org/x/text/unicode/norm"

input := norm.NFC.String(rawInput)
result := CompiledPattern.FindString(input)
```

## Complete Example

```go
package main

import "fmt"

func main() {
    // Match international names
    input := "Users: María, 田中, Αλέξανδρος, محمد"

    matches := CompiledName.FindAllString(input, -1)
    for _, m := range matches {
        fmt.Printf("Name: %s\n", m.Match)
    }
}
```

Pattern:
```bash
regengo -pattern '(?P<name>\p{L}+)' -name Name -output name.go
```
