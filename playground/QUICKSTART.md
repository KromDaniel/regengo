# 🎮 Regengo Playground - Quick Start

Try Regengo patterns instantly, no installation required!

## 🚀 3 Ways to Play

### 1. Browser Playground (Instant) ⭐ Recommended for Quick Tests

**No installation needed!**

```bash
# Option A: Open the file directly
open playground/index.html

# Option B: Serve locally
cd playground
python3 -m http.server 8000
# Visit http://localhost:8000
```

**What you can do:**
- ✅ Test regex patterns instantly
- ✅ See generated Go code template
- ✅ Try pre-loaded examples (Email, Date, URL, Phone)
- ✅ Copy code to use in your project

### 2. Local Playground (Full Power) ⭐ Recommended for Development

**Requires:** Go installed

```bash
# Clone and run
git clone https://github.com/KromDaniel/regengo
cd regengo/playground
go run playground.go

# View generated code
cat playground_output.go

# Customize the pattern
# Edit playground.go, change the pattern variable, and run again!
```

**What you can do:**
- ✅ Generate real optimized Go code
- ✅ Run actual benchmarks
- ✅ See performance comparisons
- ✅ Experiment with custom patterns

### 3. GitHub Codespaces (Cloud IDE)

**No local setup required!**

1. Go to https://github.com/KromDaniel/regengo
2. Click **Code** → **Codespaces** → **Create codespace on main**
3. Wait for environment to load (~30-60 seconds)
4. Run: `cd playground && go run playground.go`

**What you can do:**
- ✅ Full Go environment in browser
- ✅ Pre-configured and ready to use
- ✅ Real benchmarks
- ✅ Free for GitHub users

## 📝 Example Workflow

### Browser Playground

```
1. Open playground/index.html
2. Click "Date" example
3. Click "Generate Code"
4. See the Go code template
5. Click "Run Tests"
6. See test results with sample inputs
7. Click "Copy" to copy code
8. Paste into your Go project
```

### Local Playground

```
1. cd regengo/playground
2. go run playground.go
3. See output: playground_output.go
4. Copy the functions you need
5. Modify pattern in playground.go
6. Run again to regenerate
```

## 🎯 Try These Patterns

### Email Extraction
```regex
(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)
```

**Test with:**
```
john.doe@example.com
jane@test.org
admin+tag@company.co.uk
```

### Date Parsing
```regex
(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})
```

**Test with:**
```
2024-01-15
2024-12-25
2025-06-30
```

### URL Matching
```regex
(?P<protocol>https?)://(?P<host>[\w\.-]+)(?::(?P<port>\d+))?(?P<path>/[\w\./]*)?
```

**Test with:**
```
https://example.com
http://api.github.com:443/repos
https://test.org/path/to/file
```

## 🔥 What Makes It Special

1. **Instant Feedback** - See results immediately in browser
2. **Real Code** - Get actual Go code you can use
3. **Performance** - 3-4x faster than stdlib regexp
4. **Type-Safe** - Named capture groups → struct fields
5. **Zero Dependencies** - Just copy the generated functions

## 💡 Tips

- **Start Simple:** Try the example patterns first
- **Test Multiple Inputs:** Add various test cases to see all behaviors
- **Check Edge Cases:** Test with invalid inputs too
- **Use Named Groups:** `(?P<name>...)` creates readable struct fields
- **Compare Performance:** Use local playground to run benchmarks

## 🐛 Troubleshooting

### Browser Playground: "Serve locally" not working?

Try a different port:
```bash
python3 -m http.server 3000
```

Or use Node.js:
```bash
npx http-server playground -p 8000
```

### Local Playground: "command not found: go"

Install Go from https://go.dev/doc/install

### Codespaces: Environment not loading?

Wait up to 2 minutes. If still stuck, try:
1. Close the codespace
2. Delete it from GitHub
3. Create a new one

## 📚 Next Steps

After trying the playground:

1. **Read the docs:** Check out [docs/CAPTURE_GROUPS.md](../docs/CAPTURE_GROUPS.md)
2. **See examples:** Look at [examples/](../examples/)
3. **Run benchmarks:** Execute `make bench` in the repo
4. **Use in your project:** Add `regengo` to your project's dependencies

## 🌐 Sharing Your Patterns

Want to share a pattern with your team?

**Coming soon:** URL-based sharing
```
https://.../playground/?pattern=...&name=...
```

For now:
1. Copy the pattern
2. Share it via Slack/email/etc.
3. Recipients can paste into playground

## 🚀 Ready to Try?

Pick your adventure:
- **Quick test?** → Open `playground/index.html`
- **Serious dev?** → Run `go run playground/playground.go`
- **Cloud IDE?** → Use GitHub Codespaces

**Have fun experimenting with Regengo!** 🎉
