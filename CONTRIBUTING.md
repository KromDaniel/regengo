# Contributing to Regengo

First off, thank you for considering contributing to Regengo! It's people like you that make Regengo such a great tool.

## Code of Conduct

This project and everyone participating in it is governed by respect and professionalism. Please be kind and courteous.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the existing issues as you might find out that you don't need to create one. When you are creating a bug report, please include as many details as possible:

- **Use a clear and descriptive title**
- **Describe the exact steps to reproduce the problem**
- **Provide specific examples** (regex patterns, input strings, etc.)
- **Describe the behavior you observed and what you expected**
- **Include Go version and OS information**

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion, please include:

- **A clear and descriptive title**
- **A detailed description of the proposed feature**
- **Examples of how the feature would be used**
- **Why this enhancement would be useful**

### Pull Requests

1. Fork the repo and create your branch from `main`
2. If you've added code that should be tested, add tests
3. If you've changed APIs, update the documentation
4. Ensure the test suite passes (`make test`)
5. Make sure your code lints (`make lint`)
6. Format your code (`make fmt`)

## Development Setup

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/regengo.git
cd regengo

# Install dependencies
go mod download

# Run tests
make test

# Run benchmarks
make bench

# Format code
make fmt

# Run linter
make lint
```

## Project Structure

```
regengo/
├── cmd/
│   └── regengo/          # CLI tool
├── pkg/
│   └── regengo/          # Public API
├── internal/
│   ├── compiler/         # Core compilation logic
│   └── codegen/          # Code generation helpers
├── examples/             # Usage examples
├── benchmarks/           # Benchmark tests
└── docs/                 # Documentation
```

## Coding Guidelines

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Write clear, self-documenting code
- Add comments for exported functions and complex logic

### Commit Messages

- Use the present tense ("Add feature" not "Added feature")
- Use the imperative mood ("Move cursor to..." not "Moves cursor to...")
- Limit the first line to 72 characters or less
- Reference issues and pull requests liberally after the first line

Example:

```
Add support for capture groups

- Implement capture group parsing
- Add tests for capture functionality
- Update documentation

Fixes #123
```

### Testing

- Write unit tests for new functionality
- Ensure all tests pass before submitting PR
- Aim for good test coverage
- Include both positive and negative test cases

### Documentation

- Update README.md if you change functionality
- Add godoc comments for exported functions
- Include examples in documentation where helpful

## Performance Considerations

- Run benchmarks before and after your changes
- If your change affects performance, include benchmark results in PR
- Avoid premature optimization, but be mindful of performance

## Review Process

1. Maintainer reviews your PR
2. Address any feedback or requested changes
3. Once approved, maintainer will merge your PR
4. Your contribution will be included in the next release!

## Questions?

Feel free to open an issue with your question or reach out to the maintainers.

## Recognition

Contributors will be recognized in the project's README and release notes.

Thank you for contributing to Regengo! 🎉
