# Contributing to Headroom Go

First off, thank you for considering contributing! 🎉

## 🚀 Getting Started (Under 10 Minutes)

### Prerequisites

- **Go 1.22+** ([download](https://go.dev/dl/))
- **Git**

### Clone & Build

```bash
git clone https://github.com/superops-team/headroom-go.git
cd headroom-go
go build ./...
```

### Run Tests

```bash
# All tests with race detection
go test -race -count=1 ./...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Benchmarks
go test -bench=. -benchtime=1s ./...
```

### Lint

```bash
go vet ./...
```

## 📁 Project Structure

```
headroom-go/
├── headroom.go              # Public API: Message, Options, Result, Compress()
├── version.go               # Version constants
├── observability.go         # Observer interface
├── content_kind.go          # ContentKind enum
├── internal/
│   ├── compressors/         # SmartCrusher, Code, Text, Diff, Log, etc.
│   ├── engine/              # CompressionEngine, Pipeline, Policy
│   ├── router/              # ContentRouter (auto-detect content type)
│   ├── tokenizer/           # Tokenizer interface + backends
│   ├── ccr/                 # Reversible compression store
│   ├── cachealigner/        # KV cache prefix alignment
│   ├── tagprotector/        # XML tag preservation
│   └── types/               # Shared types
├── proxy/                   # HTTP proxy (OpenAI-compatible)
├── cmd/headroom/            # CLI entry point
└── testdata/                # Test fixtures
```

## 🎯 What to Contribute

| Area | Ideas |
|------|-------|
| **New Compressor** | Add support for a new content type (YAML, TOML, XML, Markdown, Protobuf...) |
| **Tokenizer Backend** | Integrate a new tokenizer (e.g., sentencepiece) |
| **Performance** | Optimize existing compressors, reduce allocations |
| **Bug Fixes** | Check [Issues](https://github.com/superops-team/headroom-go/issues) |
| **Documentation** | Improve godoc comments, add examples, fix typos |
| **Tests** | Increase coverage, add edge cases, fuzz tests |

## 🔌 Adding a Custom Compressor

1. Implement the `Compressor` interface in `internal/compressors/`:

```go
type myCompressor struct{}

func (myCompressor) Kind() ContentKind { return KindText }
func (myCompressor) Compress(content string, cfg CompressionConfig) (string, error) {
    // Your compression logic here
    return compressed, nil
}
```

2. Register it in `DefaultCompressorRegistry()` or via the public API.

3. Add tests in a `_test.go` file.

4. Run `go test -race ./...` to verify.

## 📝 Commit Conventions

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add YAML compressor
fix: handle empty JSON arrays in SmartCrusher
docs: update README with proxy guide
refactor: extract CCR GC to background goroutine
test: add edge cases for TextCompressor
chore: bump version to v0.5.0
```

## 🔍 Pull Request Process

1. **Fork** the repository
2. **Create** a feature branch: `git checkout -b feat/my-feature`
3. **Write** code + tests
4. **Run** `go test -race -count=1 ./...` — must pass
5. **Run** `go vet ./...` — must be clean
6. **Ensure** coverage doesn't drop
7. **Commit** with conventional commit message
8. **Push** and open a Pull Request

## 🧪 Testing Guidelines

- Every new feature must have tests
- Test edge cases: empty input, very large input, invalid input
- Use `testdata/` for fixture files
- E2E tests go in `spec_*_e2e_test.go` at the root
- Unit tests go alongside the code in `internal/` subpackages

## 🏛️ Design Principles

1. **Zero external dependencies** — standard library only
2. **Single binary** — no runtime dependencies
3. **Backward compatible** — public API must not break
4. **Pure functions** — compressors take `string`, return `string`
5. **Test-driven** — write tests first, then implementation

## 📄 License

By contributing, you agree that your contributions will be licensed under the MIT License.
