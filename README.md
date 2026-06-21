# ⚡ Headroom Go

<p align="center">
  <b>Intelligent Context Compression for the AI Agent Era</b><br>
  <sub>Single binary · Zero dependencies · Up to <b>70% token savings</b></sub>
</p>

<p align="center">
  <a href="https://goreportcard.com/report/github.com/superops-team/headroom-go"><img src="https://goreportcard.com/badge/github.com/superops-team/headroom-go" alt="Go Report Card"></a>
  <a href="https://github.com/superops-team/headroom-go"><img src="https://img.shields.io/badge/coverage-92.8%25-brightgreen" alt="Coverage"></a>
  <a href="https://github.com/superops-team/headroom-go"><img src="https://img.shields.io/badge/tests-138%20passing-brightgreen" alt="Tests"></a>
  <a href="https://github.com/superops-team/headroom-go/releases"><img src="https://img.shields.io/github/v/release/superops-team/headroom-go" alt="Release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
  <a href="https://pkg.go.dev/github.com/superops-team/headroom-go"><img src="https://pkg.go.dev/badge/github.com/superops-team/headroom-go.svg" alt="Go Reference"></a>
</p>

---

## 💸 The Problem

Every token you send to an LLM costs money. Agent workflows amplify this — tool outputs, logs, RAG snippets, search results, and conversation history pile up fast. A single agent run can easily burn **50,000+ tokens** in context alone.

**Headroom Go** compresses everything your agent reads *before* it hits the LLM — slashing token costs by up to **70%** while preserving semantic accuracy. It's a production-grade Go port of [headroom](https://github.com/chopratejas/headroom), purpose-built for the AI agent era.

> **The math is simple:** If you spend $1,000/month on LLM API calls, Headroom Go can save you **$700/month**. For teams running hundreds of agent sessions daily, that's real money.

---

## 🎯 Why Headroom Go?

|  | Headroom Go | Raw Python Headroom | No Compression |
|---|---|---|---|
| **Deployment** | Single binary, drop-in | Python + pip + venv | — |
| **Dependencies** | Zero (pure Go stdlib) | 10+ pip packages | — |
| **Speed** | ~650 ops/s | ~50 ops/s | — |
| **Content Types** | 10 auto-detected | 5 | 0 |
| **Proxy Mode** | ✅ OpenAI-compatible | ❌ | — |
| **Reversible (CCR)** | ✅ Built-in | ❌ | — |
| **KV Cache Friendly** | ✅ CacheAligner | ❌ | — |
| **Token Savings** | Up to 70% | Up to 50% | 0% |

---

## 🚀 Quick Start

### One-liner Install

```bash
curl -sSL https://raw.githubusercontent.com/superops-team/headroom-go/main/install.sh | bash
```

Or with Go:

```bash
go install github.com/superops-team/headroom-go/cmd/headroom@latest
```

### Compress in 5 Seconds

```bash
# Pipe anything — logs, JSON, code, HTML — through it
cat huge_log.txt | headroom compress --stats
# → Original: 12,500 tokens | Compressed: 3,750 tokens | Savings: 70.0%

# Aggressive mode for maximum savings
echo '{"items":[1,2,3,4,5,6,7,8],"metadata":{...}}' | headroom compress -a 0.8

# Transparent OpenAI proxy — all messages auto-compressed
headroom proxy --upstream https://api.openai.com/v1 --port 8080
```

### Use as a Go Library

```go
import headroom "github.com/superops-team/headroom-go"

result, _ := headroom.Compress(messages, headroom.Options{
    Aggressiveness: 0.5,
    Reversible:     true,   // retrieve originals later
    AlignPrefix:    true,   // boost KV cache hits
})

fmt.Printf("Saved %.0f%% tokens\n", result.Savings*100)
// → Saved 68% tokens
```

---

## 🧠 How It Works

Headroom Go sits between your application and the LLM, acting as an intelligent compression layer:

```
   Your App                Headroom Go                LLM API
  ──────────             ──────────────             ──────────
  │ Tool outputs │──→  │ Auto-detect   │──→  │  Compressed   │
  │ Logs         │     │ content type  │     │  messages     │──→  OpenAI
  │ RAG snippets │     │ Apply best    │     │  (70% fewer   │     Anthropic
  │ Code diffs   │     │ compressor    │     │   tokens!)    │     etc.
  │ Search hits  │     │ Preserve tags │     │               │
  ───────────────     ────────────────     ───────────────
```

### 10 Content Types, Each with Specialized Compression

| Content Type | Detection | Compression Strategy |
|-------------|-----------|---------------------|
| **JSON** | `{...}`, `[...]` | Remove nulls/empties, fold arrays, truncate floats |
| **Code** | Keywords (`func`, `class`, `def`) | Strip comments, fold long functions, preserve error handling |
| **Text** | Default fallback | Deduplicate lines, remove stopwords, fold paragraphs |
| **Diff** | `@@` headers | Collapse unchanged hunks, preserve +/- context |
| **Log** | Timestamp + level | Preserve FATAL/ERROR, fold repeated INFO/DEBUG |
| **Search** | `filename:line:` format | Collapse repeated matches, preserve file grouping |
| **Tabular** | TSV/CSV | Column-aware truncation, header preservation |
| **Spreadsheet** | Multi-column data | Cell-level compression with schema awareness |
| **HTML** | Tag structure | Strip comments, collapse inline styles |

---

## 🔥 Killer Features

### 🏷️ Tag Protector
Never worry about compression mangling your structured outputs. `<thinking>`, `<tool_call>`, and custom XML tags are automatically preserved.

### 🔙 Reversible Compression (CCR)
Compress aggressively, recover losslessly. Every compressed output gets a retrieval ID — call it back anytime.

```go
opts := headroom.Options{Reversible: true}
result, _ := headroom.Compress(messages, opts)
// Output: [headroom:retrieve id=v3_a1b2c3d4e5f6]

original, found := ccr.Retrieve("v3_a1b2c3d4e5f6") // Full original, anytime
```

### ⚡ KV Cache Friendly
The `CacheAligner` prefixes output so identical configs produce identical prefixes — boosting provider-side cache hit rates and saving even more.

### 🔌 Pluggable Architecture
Need a custom compressor? Implement the `Compressor` interface and register it — no core code changes needed.

```go
registry.Register(headroom.NewCompressorFunc(headroom.KindText,
    func(content string, opts headroom.Options) (string, error) {
        return yourCustomCompression(content), nil
    },
))
```

### 🌐 OpenAI-Compatible Proxy
Drop-in replacement. Point your client to `http://localhost:8080/v1/chat/completions` and every message is transparently compressed.

```bash
headroom proxy --upstream https://api.openai.com/v1 --port 8080
```

---

## 📊 Real-World Performance

Benchmarks on Intel Xeon (32 cores), Go 1.22:

| Benchmark | Throughput | What It Means |
|-----------|-----------|---------------|
| Content Detection (1MB) | **390 MB/s** | 10 content types detected in ~2.5ms |
| Tokenizer (1MB) | **95 MB/s** | Token counting at wire speed |
| Diff Compressor (5k lines) | **6.1M lines/s** | Entire PR diffs compressed instantly |
| Log Compressor (50k lines) | **2.3M lines/s** | Production log files in milliseconds |
| End-to-End (mixed) | **650 ops/s** | 650 messages compressed per second |

---

## 🏗️ Architecture

```
                         ┌──────────────────────────┐
                         │     Compress(messages)    │
                         └────────────┬─────────────┘
                                      │
                   ┌──────────────────┼──────────────────┐
                   ▼                                     ▼
          ┌───────────────┐                    ┌─────────────────┐
          │  Legacy Path  │                    │  Pipeline Path   │
          │ (simple/fast) │                    │ (policy-driven)  │
          └───────┬───────┘                    └────────┬────────┘
                  │                                     │
                  ▼                                     ▼
         ┌─────────────────────────────────────────────────────┐
         │              ContentRouter.Detect()                  │
         │  JSON │ Code │ Text │ Diff │ Log │ Search │ ...     │
         └──────────────────────┬──────────────────────────────┘
                                │
                   ┌────────────┼────────────┐
                   ▼            ▼            ▼
            ┌──────────┐ ┌──────────┐ ┌──────────┐
            │SmartCrush│ │  Code    │ │  Text    │  ... 7 more
            │  (JSON)  │ │Compressor│ │Compressor│
            └────┬─────┘ └────┬─────┘ └────┬─────┘
                 │            │            │
                 └────────────┼────────────┘
                              ▼
                   ┌─────────────────────┐
                   │   CacheAligner      │ ← KV cache prefix
                   │   Tag Protector     │ ← preserve XML tags
                   │   CCR Store         │ ← reversible ID
                   └─────────────────────┘
```

---

## 🎯 Use Cases

| Scenario | Without Headroom | With Headroom |
|----------|-----------------|---------------|
| **AI Coding Agent** (50 tool calls/session) | 80K tokens/session | **24K tokens/session** |
| **RAG Pipeline** (100 documents/query) | 45K tokens/query | **13K tokens/query** |
| **Log Analysis Agent** (10MB log file) | 200K tokens | **60K tokens** |
| **Multi-turn Chat** (20 exchanges) | 35K tokens | **10K tokens** |
| **CI/CD Error Summarizer** (build logs) | 150K tokens | **45K tokens** |

*Estimates based on standard aggressiveness (0.5). Aggressive mode (0.8) can push savings beyond 75%.*

---

## 🔧 Development

```bash
# Clone & test
git clone https://github.com/superops-team/headroom-go.git
cd headroom-go

# Run all tests with race detection
go test -race -count=1 ./...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Benchmarks
go test -bench=. -benchtime=1s ./...

# Build
go build -o headroom ./cmd/headroom
```

---

## 🤝 Contributing

Contributions are welcome! Whether it's a new content type compressor, a tokenizer backend, or a bug fix — we'd love your help.

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feat/amazing-feature`)
5. Open a Pull Request

Please ensure tests pass (`go test -race ./...`) and coverage doesn't drop.

---

## 📄 License

MIT — see [LICENSE](LICENSE).

---

<p align="center">
  <sub>Built with ❤️ by <a href="https://github.com/superops-team">superops-team</a> · Powered by pure Go · No snakes were harmed 🐍</sub>
</p>
