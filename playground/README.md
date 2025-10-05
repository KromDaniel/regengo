# Regengo Playground

An interactive web-based playground for testing and benchmarking Regengo patterns.

## Features

- ✅ **Live Code Generation**: See generated Go code instantly
- ✅ **Interactive Testing**: Test patterns against sample inputs
- ✅ **Performance Comparison**: Compare with stdlib regexp
- ✅ **Share Links**: Share your patterns via URL
- ✅ **Examples**: Pre-loaded common patterns

## Architecture Options

### Option 1: WebAssembly (Full Go in Browser)

**Pros:**
- Full Go compilation in browser
- No server needed for compilation
- Instant feedback
- Works offline

**Cons:**
- Larger initial download (~2-3MB WASM)
- Can't run actual benchmarks (no Go runtime)

**Implementation:**
```
playground/
  wasm/
    main.go          # WASM entry point
    compile.js       # JS wrapper
  web/
    index.html       # UI
    editor.js        # Code editor (Monaco/CodeMirror)
    runner.js        # Test runner
```

### Option 2: Server-Based API

**Pros:**
- Real benchmarks
- Smaller client bundle
- Full Go toolchain available

**Cons:**
- Requires server infrastructure
- API rate limiting needed
- Security concerns (code execution)

**Implementation:**
```
playground/
  server/
    main.go          # API server
    sandbox.go       # Secure execution
  web/
    index.html       # UI
```

### Option 3: GitHub Codespaces Template (Quick Start)

**Pros:**
- No infrastructure needed
- Full Go environment
- Real benchmarks
- Free for users

**Cons:**
- Requires GitHub account
- Less instant than browser-based

**Implementation:**
```
.devcontainer/
  devcontainer.json  # Codespaces config
templates/
  playground-template.go
```

### Option 4: Go Playground + Regengo CLI

**Pros:**
- Uses existing Go Playground
- No new infrastructure
- Official Go environment

**Cons:**
- Can't install external packages
- Manual copy-paste workflow

## Recommended: Hybrid Approach

1. **WASM for code generation** - Instant preview of generated code
2. **Examples with "Run in Codespaces"** - For real benchmarks
3. **Optional API server** - For users who want instant benchmarks

## Quick Implementation: Static HTML Playground

For a quick start, we can create a static HTML page that:
1. Uses `regengo` CLI (pre-installed)
2. Generates code on the client side
3. Shows diff with stdlib
4. Provides copy-paste templates

This requires no infrastructure and can be hosted on GitHub Pages.

## See Also

- [Implementation Guide](./IMPLEMENTATION.md) - Detailed setup
- [Security Considerations](./SECURITY.md) - Sandboxing approaches
- [Deployment Options](./DEPLOYMENT.md) - Hosting recommendations
