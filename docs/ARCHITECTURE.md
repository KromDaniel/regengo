# Regengo Architecture

## Overview

Regengo transforms regular expressions into optimized Go code at compile time. This document describes the internal architecture and design decisions.

## Project Structure

```
regengo/
├── pkg/regengo/          # Public API
├── internal/
│   ├── compiler/         # Core compilation logic
│   └── codegen/          # Code generation helpers
├── cmd/regengo/          # CLI tool
├── examples/             # Usage examples
├── benchmarks/           # Performance benchmarks
└── docs/                 # Documentation
```

## Components

### 1. Public API (`pkg/regengo`)

The public API provides a simple interface for compiling regex patterns:

```go
type Options struct {
    Pattern    string
    Name       string
    OutputFile string
    Package    string
}

func Compile(opts Options) error
```

### 2. Compiler (`internal/compiler`)

The compiler is the core of regengo. It:

1. **Parses** regex patterns using Go's `regexp/syntax`
2. **Simplifies** the syntax tree
3. **Compiles** to instruction format
4. **Generates** optimized Go code

#### Key Components:

- `Compiler`: Main compilation orchestrator
- `generateMatchFunction()`: Creates the main matching logic
- `generateInstructions()`: Converts each instruction to Go code
- `generateInstruction()`: Handles specific instruction types

### 3. Code Generator (`internal/codegen`)

Provides helper functions and constants for code generation:

- Variable naming conventions
- Label generation
- Common code patterns

### 4. CLI (`cmd/regengo`)

Command-line interface for batch code generation.

## Compilation Process

### Phase 1: Parsing

```
Regex Pattern → regexp/syntax.Parse() → AST
```

Go's standard `regexp/syntax` package parses the pattern into an abstract syntax tree.

### Phase 2: Simplification

```
AST → Simplify() → Simplified AST
```

The AST is simplified to reduce redundancy and optimize structure.

### Phase 3: Compilation

```
Simplified AST → syntax.Compile() → Instruction Program
```

The AST is compiled into a linear sequence of instructions representing a finite state machine.

### Phase 4: Code Generation

```
Instruction Program → generateMatchFunction() → Go Code
```

Each instruction is translated into Go code:

- **InstRune1**: Direct byte comparison
- **InstRune**: Range checking
- **InstAlt**: Backtracking with stack
- **InstMatch**: Return true
- **InstFail**: Return false

## Generated Code Structure

The generated code follows this pattern:

```go
func NameMatchString(input string) bool {
    // 1. Initialize variables
    l := len(input)
    offset := 0
    stack := make([][2]int, 0)
    nextInstruction := 0
    goto StepSelect

    // 2. Backtracking handler
TryFallback:
    // Pop from stack or try next position

    // 3. Instruction dispatcher
StepSelect:
    switch nextInstruction {
        case 0: goto Ins0
        case 1: goto Ins1
        // ...
    }

    // 4. Individual instructions
Ins0:
    // Instruction-specific logic

Ins1:
    // ...
}
```

## Optimization Techniques

### 1. Inline State Transitions

Instead of function calls, use `goto` for zero-overhead transitions.

### 2. Explicit Stack Management

Backtracking uses a pre-allocated stack instead of recursion.

### 3. Direct Comparisons

Character matching uses direct byte/rune comparisons instead of regex engine interpretation.

### 4. Bounds Checking

Early bounds checks prevent unnecessary computation.

## Performance Characteristics

| Aspect     | Regengo        | Standard regexp |
| ---------- | -------------- | --------------- |
| Match Time | O(n) - O(n²)\* | O(n) - O(n²)\*  |
| Startup    | None           | Parse + Compile |
| Memory     | Stack only     | Engine state    |
| Code Size  | Larger         | Smaller         |

\*Depends on pattern complexity

## Limitations

1. **Code Size**: Generated code can be large for complex patterns
2. **Feature Support**: Not all regex features are implemented
3. **Compilation Time**: Code generation adds to build time

## Future Improvements

1. **Capture Groups**: Support for extracting matched substrings
2. **Find Operations**: Beyond just Match
3. **Optimization Passes**: Pattern-specific optimizations
4. **Parallel Matching**: For very large inputs

## Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines on contributing to the compiler.
