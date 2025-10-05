# Regengo Playground - Implementation Guide

## Overview

The Regengo playground provides multiple ways for users to experiment with patterns, generate code, and run benchmarks.

## Available Options

### 1. Static HTML Playground (Instant Preview)

**Location:** `playground/index.html`

**Features:**
- ✅ Browser-based, no installation
- ✅ Live pattern testing with JavaScript regex
- ✅ Generates Go code template
- ✅ Pre-loaded examples
- ✅ Copy-paste workflow

**Usage:**
```bash
# Open in browser
open playground/index.html

# Or serve with Python
cd playground
python3 -m http.server 8000
# Visit http://localhost:8000
```

**Deployment:**
Can be hosted on:
- GitHub Pages
- Netlify
- Vercel
- Any static hosting

### 2. GitHub Codespaces (Full Environment)

**Location:** `.devcontainer/devcontainer.json`

**Features:**
- ✅ Full Go environment in browser
- ✅ Real benchmarks
- ✅ Pre-configured with Regengo
- ✅ Free for GitHub users

**Usage:**
1. Go to https://github.com/KromDaniel/regengo
2. Click "Code" → "Codespaces" → "Create codespace"
3. Wait for environment to load
4. Run: `cd playground && go run playground.go`

**One-Click Link:**
```
[![Open in GitHub Codespaces](https://github.com/codespaces/badge.svg)](https://codespaces.new/KromDaniel/regengo?quickstart=1)
```

### 3. Local Playground Template

**Location:** `playground/playground.go`

**Features:**
- ✅ Quick local experimentation
- ✅ Modify and run instantly
- ✅ See generated code
- ✅ Full Go toolchain

**Usage:**
```bash
# Clone repo
git clone https://github.com/KromDaniel/regengo
cd regengo/playground

# Run playground
go run playground.go

# View generated code
cat playground_output.go

# Test it
go test -bench=. -benchmem
```

## Future: WebAssembly Playground

For a fully interactive browser experience, we could implement:

```
playground/
  wasm/
    main.go           # WASM entry point using syscall/js
    wasm_exec.html    # Host page
  
  build.sh            # GOOS=js GOARCH=wasm go build
```

**Pros:**
- Full Go compilation in browser
- No server needed
- Instant feedback

**Cons:**
- Larger initial download (~2-3MB)
- Can't run real benchmarks
- Requires WASM support

### WASM Implementation Sketch

```go
// playground/wasm/main.go
package main

import (
    "syscall/js"
    "github.com/KromDaniel/regengo/pkg/regengo"
)

func generateCode(this js.Value, args []js.Value) interface{} {
    pattern := args[0].String()
    name := args[1].String()
    
    // Generate code to string instead of file
    code, err := regengo.CompileToString(regengo.Options{
        Pattern: pattern,
        Name:    name,
        Package: "generated",
    })
    
    if err != nil {
        return map[string]interface{}{
            "error": err.Error(),
        }
    }
    
    return map[string]interface{}{
        "code": code,
    }
}

func main() {
    js.Global().Set("regengoCompile", js.FuncOf(generateCode))
    select {} // Keep alive
}
```

## Comparison Matrix

| Feature | Static HTML | Codespaces | Local | WASM (Future) |
|---------|------------|------------|-------|---------------|
| Setup Time | Instant | 30-60s | 5min | Instant |
| Code Generation | Template | Real | Real | Real |
| Pattern Testing | JS Regex | Real | Real | Real |
| Benchmarks | No | Yes | Yes | No |
| Internet Required | No* | Yes | No | No* |
| Installation | None | None | Go + Git | None |

*After first load

## Recommended User Flows

### Quick Experimentation
1. Open `playground/index.html`
2. Try different patterns
3. See instant JS regex results
4. Copy generated code template

### Serious Benchmarking
1. Use GitHub Codespaces or local setup
2. Generate real code
3. Run actual benchmarks
4. Compare with stdlib

### Sharing Patterns
1. Use static HTML playground
2. Share URL with pattern in query params
3. Others can instantly try it

## URL Parameters (Future Enhancement)

Add support for sharing via URL:

```
https://kromdaniel.github.io/regengo/playground/?pattern=...&name=...&input=...
```

```javascript
// In index.html
window.onload = () => {
    const params = new URLSearchParams(window.location.search);
    if (params.has('pattern')) {
        document.getElementById('pattern').value = decodeURIComponent(params.get('pattern'));
    }
    if (params.has('name')) {
        document.getElementById('name').value = params.get('name');
    }
    if (params.has('input')) {
        document.getElementById('testInput').value = decodeURIComponent(params.get('input'));
    }
    generateCode();
};

function shareLink() {
    const pattern = encodeURIComponent(document.getElementById('pattern').value);
    const name = document.getElementById('name').value;
    const input = encodeURIComponent(document.getElementById('testInput').value);
    const url = `${window.location.origin}${window.location.pathname}?pattern=${pattern}&name=${name}&input=${input}`;
    
    navigator.clipboard.writeText(url);
    alert('Share link copied to clipboard!');
}
```

## Security Considerations

### For Server-Based Playgrounds

If implementing an API server for code execution:

1. **Sandboxing:**
   - Use Docker containers
   - Resource limits (CPU, memory, time)
   - No network access

2. **Rate Limiting:**
   - Per IP: 10 requests/minute
   - Per session: 100 requests/day
   - Pattern complexity limits

3. **Code Validation:**
   - Reject imports
   - Pattern length limits
   - Timeout after 5 seconds

```go
// Example sandbox
func executeSandboxed(pattern string) (string, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
        "--memory=100m",
        "--cpus=0.5",
        "--network=none",
        "regengo-sandbox",
        pattern,
    )
    
    return cmd.Output()
}
```

## Deployment

### GitHub Pages (Recommended for Static)

```bash
# Deploy to gh-pages branch
git subtree push --prefix playground origin gh-pages
```

### Docker (For API Server)

```dockerfile
FROM golang:1.21-alpine
WORKDIR /app
COPY . .
RUN go build -o playground-server ./playground/server
CMD ["./playground-server"]
```

## Analytics

Track usage to improve the playground:

```javascript
// Simple analytics (privacy-friendly)
function trackEvent(event) {
    // Use Plausible, Simple Analytics, or similar
    if (window.plausible) {
        plausible(event);
    }
}

// Track pattern generation
function generateCode() {
    // ... existing code ...
    trackEvent('Generate Code');
}
```

## Next Steps

1. **Phase 1:** Deploy static HTML to GitHub Pages ✅
2. **Phase 2:** Add URL parameter sharing
3. **Phase 3:** Implement WASM version
4. **Phase 4:** Optional API server for real benchmarks

## See Also

- [playground/README.md](./README.md) - Overview
- [playground/index.html](./index.html) - Static playground
- [playground/playground.go](./playground.go) - Local template
