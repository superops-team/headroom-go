# ⚡ Headroom Go

<p align="center">
  <b>Intelligent Context Compression for the AI Agent Era</b><br>
  <sub>Single binary · Zero dependencies · Up to <b>70% token savings</b> · MCP Native · Docker <10MB</sub>
</p>

<p align="center">
  <a href="https://goreportcard.com/report/github.com/superops-team/headroom-go"><img src="https://goreportcard.com/badge/github.com/superops-team/headroom-go" alt="Go Report Card"></a>
  <a href="https://github.com/superops-team/headroom-go/actions/workflows/ci.yml"><img src="https://github.com/superops-team/headroom-go/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/superops-team/headroom-go"><img src="https://img.shields.io/badge/coverage-85%25-brightgreen" alt="Coverage"></a>
  <a href="https://github.com/superops-team/headroom-go/releases/latest"><img src="https://img.shields.io/badge/version-v0.8.0-blue" alt="Release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
  <a href="https://pkg.go.dev/github.com/superops-team/headroom-go"><img src="https://pkg.go.dev/badge/github.com/superops-team/headroom-go.svg" alt="Go Reference"></a>
</p>

---

## 📑 Table of Contents

- [💸 The Problem](#-the-problem)
- [🚀 Quick Start](#-quick-start)
- [🔥 Killer Features](#-killer-features)
- [📖 CLI Reference](#-cli-reference)
- [📚 Go Library API](#-go-library-api)
- [🌐 HTTP Proxy Guide](#-http-proxy-guide)
- [🔌 Integrations](#-integrations)
- [🚢 Deployment](#-deployment)
- [🧠 How It Works](#-how-it-works)
- [📦 Content Types](#-content-types)
- [📊 Real-World Performance](#-real-world-performance)
- [🔧 Development](#-development)
- [🤝 Contributing](#-contributing)

---

## 💸 The Problem

Every token you send to an LLM costs money. Agent workflows amplify this — tool outputs, logs, RAG snippets, search results, and conversation history pile up fast. A single agent run can easily burn **50,000+ tokens** in context alone.

**Headroom Go** compresses everything your agent reads *before* it hits the LLM — slashing token costs by up to **70%** while preserving semantic accuracy. It's a production-grade Go port of [headroom](https://github.com/chopratejas/headroom), purpose-built for the AI agent era.

> **The math is simple:** If you spend $1,000/month on LLM API calls, Headroom Go can save you **$700/month**.

---

## 🚀 Quick Start

### Install

```bash
# One-liner (Linux / macOS)
curl -sSL https://raw.githubusercontent.com/superops-team/headroom-go/main/install.sh | bash

# Go install
go install github.com/superops-team/headroom-go/cmd/headroom@latest

# Docker
docker pull ghcr.io/superops-team/headroom-go:latest

# Homebrew (macOS)
brew tap superops-team/headroom && brew install headroom
```

### Compress in 5 Seconds

```bash
# Pipe anything through it
cat huge_log.txt | headroom compress --stats
# → original_tokens=12500 compressed_tokens=3750 savings_pct=70.0%

# Aggressive JSON compression
echo '{"items":[1,2,3,4,5],"debug":null}' | headroom compress --aggressiveness 0.8

# Transparent OpenAI proxy
headroom proxy --port 8080
```

### Use as a Go Library

```go
import headroom "github.com/superops-team/headroom-go"

messages := []headroom.Message{{Role: "user", Content: "ERROR failed\nINFO retry\nINFO retry\n"}}
result, _ := headroom.Compress(messages, headroom.DefaultOptions())
fmt.Printf("Saved %.0f%% tokens\n", result.Savings*100)
```

---

## 🔥 Killer Features

### 🤖 MCP Server — Claude Code / Codex / Cursor 一键接入

```bash
headroom mcp serve
```

提供 4 个 MCP 工具：`headroom_compress`、`headroom_retrieve`、`headroom_stats`、`headroom_read`。Claude Code 配置：

```json
{ "mcpServers": { "headroom": { "command": "headroom", "args": ["mcp", "serve"] } } }
```

### 🎯 Wrap — 自动配置 IDE 代理

```bash
headroom wrap claude              # 打印配置指令
headroom wrap claude --apply      # 自动修改 ~/.claude/settings.json
headroom wrap codex --apply       # 自动修改 ~/.codex/config.yaml
headroom wrap generic             # 通用配置指南
```

启动本地代理 + 自动配置 IDE，Ctrl+C 恢复原配置。

### 🌐 OpenAI-Compatible Proxy（支持流式）

```bash
headroom proxy --port 8787
```

- `POST /v1/chat/completions` — 压缩后转发
- `GET /healthz` — 健康检查
- ✅ **SSE 流式响应**（`stream:true`）
- 自动转发 `Authorization` / `X-Request-ID`
- 50MB 请求体限制，60s 超时，优雅关闭

### 🔙 Reversible Compression (CCR)

压缩后可检索原始内容：

```go
store := headroom.NewCCR(headroom.CCRConfig{TTL: 24 * time.Hour})
id := store.Store("original", "compressed", headroom.KindText)
original, _ := store.Retrieve(id)
```

### ⚡ KV Cache Friendly

`CacheAligner` 为输出添加稳定前缀，提升 LLM 提供方 KV Cache 命中率。

### 🏷️ Tag Protector

Pipeline 模式下保护 `<thinking>`、`<tool_call>` 等 XML 标签不被压缩破坏。

### 🔌 Pluggable Architecture

```go
registry := headroom.NewCompressorRegistry()
registry.Register(headroom.NewCompressorFunc(headroom.KindText,
    func(content string, opts headroom.Options) (string, error) {
        return strings.ReplaceAll(content, "secret", "[redacted]"), nil
    },
))
```

---

## 📖 CLI Reference

```bash
headroom <command> [flags]
```

| Command | Description |
|---------|-------------|
| `compress` | 压缩 stdin 或文件 |
| `proxy` | 启动 HTTP 代理（OpenAI 兼容，支持流式） |
| `mcp serve` | 启动 MCP Server（stdio 模式，4 工具） |
| `wrap <agent>` | 启动代理 + 配置 IDE（claude/codex/copilot/generic） |
| `version` | 打印版本 |

### `compress`

| Flag | Type | Default | Description |
|------|------|--------|------|
| `--aggressiveness` | float | `0.5` | 压缩强度 0.0-1.0 |
| `--no-reversible` | bool | `false` | 关闭可逆压缩 |
| `--no-align` | bool | `false` | 关闭前缀对齐 |
| `--tokenizer-backend` | string | `""` | `fallback` / `tiktoken` / `huggingface` |
| `--token-budget` | int | `0` | Pipeline 目标 token 数 |
| `--enable-pipeline` | bool | `false` | 启用 Pipeline 模式 |
| `--query` | string | `""` | diff/search 评分查询词 |
| `--input` | string | `""` | 输入文件（默认 stdin） |
| `--output` | string | `""` | 输出文件（默认 stdout） |
| `--stats` | bool | `false` | 打印 token 统计 |

### `proxy`

| Flag | Type | Default | Description |
|------|------|--------|------|
| `--port` | int | `8787` | 监听端口 |
| `--upstream` | string | `https://api.openai.com/v1` | 上游 API URL |
| `--aggressiveness` | float | `0.5` | 压缩强度 |
| `--no-reversible` | bool | `false` | 关闭可逆压缩 |
| `--enable-pipeline` | bool | `false` | Pipeline 模式 |
| `--token-budget` | int | `0` | 目标 token 数 |

环境变量：`HEADROOM_API_KEY` — 当请求无 `Authorization` 头时的上游 Bearer token。

### `wrap`

```bash
headroom wrap <claude|codex|copilot|generic> [--apply] [--port=18787]
```

### `mcp`

```bash
headroom mcp serve    # 启动 MCP Server
```

---

## 📚 Go Library API

Module: `github.com/superops-team/headroom-go`

### Core Functions

```go
func DefaultOptions() Options
func Compress(messages []Message, opts Options) (*Result, error)
func CompressString(content string, opts Options) (string, error)
func NewCompressionEngine(opts Options) (*CompressionEngine, []Warning)
func NewDefaultPipeline() *Pipeline
```

### Core Types

```go
type Message struct { Role, Content string; Name string }
type Options struct {
    Aggressiveness  float64         // 0.0-1.0
    Reversible      bool            // 可逆压缩
    AlignPrefix     bool            // KV Cache 对齐
    TokenLimit      int             // 跳过阈值
    TokenizerConfig TokenizerConfig
    TokenBudget     int             // Pipeline 目标
    Query           string          // 相关性评分
    EnablePipeline  bool            // Pipeline 模式
    Observer        Observer        // 步骤回调
}
type Result struct {
    Messages         []Message
    CompressedTokens, OriginalTokens int
    Savings          float64
    Warnings         []Warning
    Steps            []CompressionStep
}
```

---

## 🌐 HTTP Proxy Guide

### Endpoints

- `POST /v1/chat/completions` — 压缩后转发（支持 `stream:true` SSE 流式）
- `GET /healthz` — `{"status":"ok","version":"v0.8.0","uptime":"..."}`

### Quick Start

```bash
HEADROOM_API_KEY="$OPENAI_API_KEY" headroom proxy --port 8787 &

curl http://localhost:8787/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hello"}]}'
```

### Streaming

```bash
curl http://localhost:8787/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"model":"gpt-4o-mini","stream":true,"messages":[{"role":"user","content":"hello"}]}'
```

流式响应注入压缩统计：`data: {"headroom_stats":{"original_tokens":...,"compressed_tokens":...,"savings":...}}`

---

## 🔌 Integrations

### Go Ecosystem

| Integration | Package | Description |
|-------------|---------|-------------|
| **OpenAI Go SDK** | `integrations/go-openai` | HTTP RoundTripper 透明压缩 |
| **langchaingo** | `integrations/langchaingo` | Document Compressor |
| **Ollama** | `integrations/ollama` | HTTP 中间件 |

```go
// OpenAI Go SDK
import headroom "github.com/superops-team/headroom-go/integrations/go-openai"

client := headroom.WrapClient(http.DefaultClient, headroom.Config{Aggressiveness: 0.5})
```

```go
// langchaingo
import headroom "github.com/superops-team/headroom-go/integrations/langchaingo"

compressor := headroom.NewDocumentCompressor(headroom.Config{Aggressiveness: 0.5})
docs, _ := compressor.CompressDocuments(ctx, documents)
```

### MCP (Model Context Protocol)

```bash
headroom mcp serve    # 4 tools: compress, retrieve, stats, read
```

兼容 Claude Code、Codex、Cursor 等所有 MCP 客户端。

### IDE Wrap

```bash
headroom wrap claude --apply     # Claude Code
headroom wrap codex --apply      # OpenAI Codex
headroom wrap copilot            # GitHub Copilot CLI
headroom wrap generic            # 通用配置
```

---

## 🚢 Deployment

### Docker

```bash
docker pull ghcr.io/superops-team/headroom-go:latest
docker run -p 18787:18787 ghcr.io/superops-team/headroom-go:latest
```

镜像 <10MB（`FROM scratch`），支持 `linux/amd64` + `linux/arm64`。

### Kubernetes

```bash
# Helm
helm repo add headroom https://superops-team.github.io/headroom-go
helm install headroom headroom/headroom

# Sidecar
kubectl apply -f https://raw.githubusercontent.com/superops-team/headroom-go/main/integrations/k8s/sidecar.yaml
```

### systemd

```ini
[Service]
Environment=HEADROOM_API_KEY=sk-xxx
ExecStart=/usr/local/bin/headroom proxy --port 8787
Restart=always
```

### CI/CD

GitHub Actions 自动化：`ci.yml`（build + test + lint + security）、`release.yml`（多平台二进制 + Docker 推送）、`docs.yml`、`benchmark.yml`。

---

## 🧠 How It Works

```
   Your App                Headroom Go                LLM API
  ──────────             ──────────────             ──────────
  │ Tool outputs │──→  │ Auto-detect   │──→  │  Compressed   │
  │ Logs         │     │ content type  │     │  messages     │──→  OpenAI
  │ RAG snippets │     │ Apply best    │     │  (70% fewer   │     Anthropic
  │ Code diffs   │     │ compressor    │     │   tokens!)    │     etc.
  ───────────────     ────────────────     ───────────────
```

Two paths: **Legacy** (fast, default) and **Pipeline** (policy-driven, budget-aware).

---

## 📦 Content Types

| Type | Detection | Strategy |
|------|-----------|----------|
| **JSON** | `{`/`[` + valid | Remove nulls, fold arrays, truncate floats |
| **Code** | 3+ lines with keywords | Strip comments, fold long functions |
| **Text** | Default | Deduplicate lines, remove stopwords |
| **Diff** | `@@` headers | Collapse unchanged hunks |
| **Log** | ERROR/WARN/INFO patterns | Preserve FATAL/ERROR, fold repeated |
| **Search** | `filename:line:` format | Collapse repeats, preserve grouping |
| **Tabular** | TSV/CSV/table | Header-preserving text fallback |
| **Spreadsheet** | Reserved | Cell-level (planned) |
| **HTML** | `<!doctype>`/`<html>` | Strip comments/script/style |
| **Unknown** | Fallback | Text compression |

---

## 📊 Real-World Performance

| Benchmark | Throughput |
|-----------|-----------|
| Content Detection (1MB) | **390 MB/s** |
| Tokenizer (1MB) | **95 MB/s** |
| Diff Compressor (5k lines) | **6.1M lines/s** |
| Log Compressor (50k lines) | **2.3M lines/s** |
| End-to-End (mixed) | **650 ops/s** |

```bash
go test -bench=. -benchtime=1s ./...
```

---

## 🔧 Development

```bash
git clone https://github.com/superops-team/headroom-go.git
cd headroom-go

go test -race -count=1 ./...        # All tests
go vet ./...                         # Static analysis
go test -cover ./...                 # Coverage (85.2%)
go build -o headroom ./cmd/headroom  # Build
```

---

## 🤝 Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md).

1. Fork → 2. Branch → 3. Commit → 4. Push → 5. PR

```bash
go test -race ./... && go vet ./...
```

---

## 📄 License

MIT — see [LICENSE](LICENSE).

---

<p align="center">
  <sub>Built with ❤️ by <a href="https://github.com/superops-team">superops-team</a> · Pure Go · Zero deps · MCP Native · Docker <10MB</sub>
</p>
