# ⚡ Headroom Go

<p align="center">
  <b>Intelligent Context Compression for the AI Agent Era</b><br>
  <sub>Single binary · Zero dependencies · Up to <b>70% token savings</b></sub>
</p>

<p align="center">
  <a href="https://goreportcard.com/report/github.com/superops-team/headroom-go"><img src="https://goreportcard.com/badge/github.com/superops-team/headroom-go" alt="Go Report Card"></a>
  <a href="https://github.com/superops-team/headroom-go"><img src="https://img.shields.io/badge/coverage-92.8%25-brightgreen" alt="Coverage"></a>
  <a href="https://github.com/superops-team/headroom-go"><img src="https://img.shields.io/badge/tests-140%20passing-brightgreen" alt="Tests"></a>
  <a href="https://github.com/superops-team/headroom-go/releases/tag/v0.5.0"><img src="https://img.shields.io/badge/version-v0.5.0-blue" alt="Release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
  <a href="https://pkg.go.dev/github.com/superops-team/headroom-go"><img src="https://pkg.go.dev/badge/github.com/superops-team/headroom-go.svg" alt="Go Reference"></a>
</p>

---

## 📑 Table of Contents

- [💸 The Problem](#-the-problem)
- [🎯 Why Headroom Go?](#-why-headroom-go)
- [🚀 Quick Start](#-quick-start)
- [📖 CLI Reference](#-cli-reference)
- [📚 Go Library API](#-go-library-api)
- [🧠 How It Works](#-how-it-works)
- [📦 Content Types](#-content-types)
- [🎚️ Compression Modes](#️-compression-modes)
- [🔥 Killer Features](#-killer-features)
- [🌐 HTTP Proxy Guide](#-http-proxy-guide)
- [🔤 Tokenizer Guide](#-tokenizer-guide)
- [🔄 Pipeline Mode](#-pipeline-mode)
- [🔙 Reversible Compression 详解](#-reversible-compression-详解)
- [🔌 Custom Compressor](#-custom-compressor)
- [📊 Observability](#-observability)
- [📊 Real-World Performance](#-real-world-performance)
- [🏗️ Architecture](#️-architecture)
- [🎯 Use Cases](#-use-cases)
- [🚢 Deployment](#-deployment)
- [🔍 Troubleshooting](#-troubleshooting)
- [🔧 Development](#-development)
- [🤝 Contributing](#-contributing)
- [📄 License](#-license)

---

## 💸 The Problem

Every token you send to an LLM costs money. Agent workflows amplify this — tool outputs, logs, RAG snippets, search results, and conversation history pile up fast. A single agent run can easily burn **50,000+ tokens** in context alone.

**Headroom Go** compresses everything your agent reads *before* it hits the LLM — slashing token costs by up to **70%** while preserving semantic accuracy. It's a production-grade Go port of [headroom](https://github.com/chopratejas/headroom), purpose-built for the AI agent era.

> **The math is simple:** If you spend $1,000/month on LLM API calls, Headroom Go can save you **$700/month**. For teams running hundreds of agent sessions daily, that's real money.

```bash
# The problem in one command: noisy context goes in, smaller context comes out.
cat <<'EOF' | headroom compress --no-reversible --stats
ERROR payment failed for order=1001 retry=1
INFO retrying payment for order=1001
INFO retrying payment for order=1001
INFO retrying payment for order=1001
EOF
```

---

## 🎯 Why Headroom Go?

|  | Headroom Go | Raw Python Headroom | No Compression |
|---|---|---|---|
| **Deployment** | Single binary, drop-in | Python + pip + venv | — |
| **Dependencies** | Zero (pure Go stdlib) | 10+ pip packages | — |
| **Speed** | ~650 ops/s | ~50 ops/s | — |
| **Content Types** | 10 auto-detected / represented | 5 | 0 |
| **Proxy Mode** | ✅ OpenAI-compatible | ❌ | — |
| **Reversible (CCR)** | ✅ Built-in | ❌ | — |
| **KV Cache Friendly** | ✅ CacheAligner | ❌ | — |
| **Token Savings** | Up to 70% | Up to 50% | 0% |

```bash
# Compare raw vs compressed size locally.
printf '%s\n' "$(seq 1 20 | sed 's/.*/INFO worker heartbeat ok/')" | \
  headroom compress --aggressiveness 0.5 --no-reversible
```

---

## 🚀 Quick Start

### One-liner Install

```bash
curl -sSL https://raw.githubusercontent.com/superops-team/headroom-go/main/install.sh | bash
```

`install.sh` supports:

- **Version lock**: pass a release tag as the first argument.
- **Platform detection**: maps `linux` / `darwin` and `amd64` / `arm64` to the correct release binary.
- **No package manager requirement**: downloads with `curl` or `wget`, installs to `/usr/local/bin/headroom`.

```bash
# Install exactly v0.5.0 instead of latest.
curl -sSL https://raw.githubusercontent.com/superops-team/headroom-go/main/install.sh | bash -s -- v0.5.0

# Verify.
headroom version
```

Or with Go:

```bash
go install github.com/superops-team/headroom-go/cmd/headroom@v0.5.0
```

### Compress in 5 Seconds

```bash
# Pipe anything — logs, JSON, code, HTML — through it.
cat huge_log.txt | headroom compress --stats
# → original_tokens=12500 compressed_tokens=3750 savings_pct=70.0%

# Aggressive mode for maximum savings.
echo '{"items":[1,2,3,4,5,6,7,8],"debug":null,"ok":true}' | \
  headroom compress --aggressiveness 0.8 --no-reversible

# Transparent OpenAI proxy — all messages auto-compressed.
HEADROOM_API_KEY="$OPENAI_API_KEY" headroom proxy --upstream https://api.openai.com/v1 --port 8080
```

### Use as a Go Library

```go
package main

import (
	"fmt"

	headroom "github.com/superops-team/headroom-go"
)

func main() {
	messages := []headroom.Message{{Role: "user", Content: "ERROR failed\nINFO retry\nINFO retry\n"}}
	result, err := headroom.Compress(messages, headroom.Options{
		Aggressiveness: 0.5,
		Reversible:     true,  // retrieve originals later
		AlignPrefix:    true,  // boost KV cache hits
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Saved %.0f%% tokens\n", result.Savings*100)
}
```

---

## 📖 CLI Reference

`headroom` has three subcommands: `compress`, `proxy`, and `version`.

```bash
headroom <command> [flags]
```

### `compress`

Compress stdin or an input file and write to stdout or an output file.

| Flag | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--aggressiveness` | float | `0.5` | 压缩强度 `0.0-1.0` |
| `--no-reversible` | bool | `false` | 关闭可逆压缩 |
| `--no-align` | bool | `false` | 关闭前缀对齐 |
| `--tokenizer-backend` | string | `""` | `fallback` / `tiktoken` / `huggingface` |
| `--token-budget` | int | `0` | 目标 token budget |
| `--enable-pipeline` | bool | `false` | 启用 Pipeline 模式 |
| `--query` | string | `""` | diff/search scoring 查询词 |
| `--input` | string | `""` | 输入文件（默认 stdin） |
| `--output` | string | `""` | 输出文件（默认 stdout） |
| `--stats` | bool | `false` | 打印 token 统计 |

```bash
# stdin → stdout, stats → stderr.
cat ./testdata/log/sample.log | headroom compress --stats --no-reversible

# file → file with pipeline and a target budget.
headroom compress \
  --input ./testdata/diff/sample.diff \
  --output /tmp/sample.compressed.diff \
  --enable-pipeline \
  --token-budget 500 \
  --query error
```

### `proxy`

Start an OpenAI-compatible HTTP proxy.

| Flag | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--port` | int | `8787` | 监听端口 |
| `--upstream` | string | `https://api.openai.com/v1` | 上游 Base URL |
| `--aggressiveness` | float | `0.5` | 压缩强度 |
| `--no-reversible` | bool | `false` | 关闭可逆压缩 |
| `--enable-pipeline` | bool | `false` | 启用 Pipeline 模式 |
| `--token-budget` | int | `0` | 目标 token budget |

Environment variable:

| Name | 说明 |
|------|------|
| `HEADROOM_API_KEY` | 当客户端请求没有 `Authorization` header 时，用作上游 Bearer token |

```bash
# Default OpenAI upstream on :8787.
HEADROOM_API_KEY="$OPENAI_API_KEY" headroom proxy

# Pipeline proxy on :8080.
HEADROOM_API_KEY="$OPENAI_API_KEY" headroom proxy \
  --port 8080 \
  --upstream https://api.openai.com/v1 \
  --enable-pipeline \
  --token-budget 2000
```

### `version`

Print the version number.

```bash
headroom version
# headroom-go v0.5.0
```

---

## 📚 Go Library API

Module path: `github.com/superops-team/headroom-go`.

### `Compress(messages, opts)`

Compresses a slice of chat messages and returns `*Result`.

```go
package main

import (
	"fmt"
	headroom "github.com/superops-team/headroom-go"
)

func main() {
	res, err := headroom.Compress([]headroom.Message{
		{Role: "system", Content: "You are concise."},
		{Role: "user", Content: "INFO ok\nINFO ok\nERROR disk full\n"},
	}, headroom.DefaultOptions())
	if err != nil { panic(err) }
	fmt.Println(res.Messages[1].Content)
}
```

### `CompressString(content, opts)`

Compresses a single text block as one `user` message.

```go
package main

import (
	"fmt"
	headroom "github.com/superops-team/headroom-go"
)

func main() {
	out, err := headroom.CompressString("ERROR one\nINFO repeat\nINFO repeat\n", headroom.Options{
		Aggressiveness: 0.5,
		Reversible: false,
	})
	if err != nil { panic(err) }
	fmt.Println(out)
}
```

### `Message{Role, Content, Name}`

OpenAI-compatible chat message.

| Field | Type | 说明 |
|------|------|------|
| `Role` | string | `system` / `user` / `assistant` / `tool` 等 |
| `Content` | string | 待压缩内容 |
| `Name` | string | 可选 name，JSON 中 `omitempty` |

```go
msg := headroom.Message{Role: "user", Content: "long context", Name: "retriever"}
fmt.Println(msg.Role, msg.Name)
```

### `Options{...}`

| Field | Type | 默认值 | 说明 |
|------|------|--------|------|
| `Aggressiveness` | float64 | `0.5` | 压缩强度，建议 `0.0-1.0` |
| `Reversible` | bool | `true` | 启用 CCR 可逆压缩 |
| `AlignPrefix` | bool | `false` (`compress` CLI 默认会打开，除非 `--no-align`) | 是否添加稳定前缀 |
| `TokenLimit` | int | `0` | 消息 token 低于该值时跳过压缩 |
| `TokenizerConfig` | `TokenizerConfig` | zero value | tokenizer 后端配置 |
| `TokenBudget` | int | `0` | Pipeline 目标 token 数；大于 0 会走 Pipeline |
| `Query` | string | `""` | Pipeline diff/search 查询词；非空会走 Pipeline |
| `EnablePipeline` | bool | `false` | 强制启用 Pipeline |
| `Observer` | `Observer` | `nil` | 压缩步骤回调；engine 内部会使用 no-op observer |

```go
opts := headroom.Options{
	Aggressiveness: 0.7,
	Reversible: true,
	AlignPrefix: true,
	TokenLimit: 100,
	TokenizerConfig: headroom.TokenizerConfig{
		Backend: headroom.TokenizerFallback,
		AllowFallback: true,
	},
	TokenBudget: 800,
	Query: "panic",
	EnablePipeline: true,
}
fmt.Printf("%+v\n", opts)
```

### `Result{...}`

| Field | Type | 说明 |
|------|------|------|
| `Messages` | `[]Message` | 压缩后的消息数组 |
| `CompressedTokens` | int | 压缩后 token 数 |
| `OriginalTokens` | int | 原始 token 数 |
| `Savings` | float64 | 节省比例，例如 `0.68` 表示 68% |
| `Warnings` | `[]Warning` | 非致命告警 |
| `Steps` | `[]CompressionStep` | 压缩步骤明细 |

```go
res, _ := headroom.Compress([]headroom.Message{{Role: "user", Content: "INFO x\nINFO x\n"}}, headroom.Options{Reversible:false})
fmt.Printf("original=%d compressed=%d savings=%.1f%%\n", res.OriginalTokens, res.CompressedTokens, res.Savings*100)
for _, step := range res.Steps { fmt.Println(step.Name, step.Kind, step.Skipped) }
```

### `DefaultOptions()`

Returns the library defaults: `Aggressiveness=0.5`, `Reversible=true`, `AlignPrefix=false`, `TokenLimit=0`.

```go
opts := headroom.DefaultOptions()
opts.Reversible = false
out, _ := headroom.CompressString("hello hello hello", opts)
fmt.Println(out)
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

Two execution paths are available:

| Path | Trigger | Best for | Capabilities |
|------|---------|----------|--------------|
| **Legacy** | default (`EnablePipeline=false`, `TokenBudget=0`, `Query=""`) | Simple, fast, direct compression | Router → compressor registry → optional CacheAligner/CCR |
| **Pipeline** | `EnablePipeline=true`, or `TokenBudget>0`, or `Query!=""` | Budget-aware and query-aware agent workflows | Policy-driven transforms, `TokenBudget`, `Query`, `Observer`, tag protection |

```go
legacy, _ := headroom.CompressString("INFO ok\nINFO ok\n", headroom.Options{Reversible:false})
pipeline, _ := headroom.CompressString("INFO ok\nINFO ok\n", headroom.Options{EnablePipeline:true, Reversible:false})
fmt.Println(legacy, pipeline)
```

---

## 📦 Content Types

### Summary

| 类型 | ContentKind | 检测规则 | 压缩策略 |
|------|-------------|---------|---------|
| JSON | `KindJSON` | `{` 或 `[` 开头 + valid JSON | 去 null/空值、折叠数组、激进模式截断浮点到 2 位 |
| Code | `KindCode` | >=3 行含代码关键字，或 fenced code / comment+brace | 去注释/空行、折叠长函数(>20行)、保留错误处理 |
| Text | `KindText` | 默认 | 去重行、移除 stopwords、折叠段落(>30行) |
| Diff | `KindDiff` | `diff --git`、`@@` 头部、或 `---`/`+++` | Pipeline 折叠未变更块、保留 +/- 上下文 |
| Log | `KindLog` | 多行包含 ERROR/WARN/FAIL/FATAL 或 `[INFO]`/`[DEBUG]` | 保留 FATAL/ERROR、折叠重复 INFO/DEBUG |
| Search | `KindSearch` | 至少 2 行 `filename:line:` 或 `filename-line-` | 折叠重复匹配、保留文件分组 |
| Tabular | `KindTabular` | TSV/CSV/Markdown table | 当前回落到文本压缩；保留表头倾向 |
| Spreadsheet | `KindSpreadsheet` | API 预留类型；当前 router 不自动检测 | 当前回落到文本压缩；用于未来单元格级压缩 |
| HTML | `KindHTML` | `<!doctype html`、`<html`、或 `<head` + `<body` | Pipeline 去注释、移除 script/style |
| Unknown | `KindUnknown` | API 预留类型 | registry 未命中时回落到 Text |

> Note: `KindSpreadsheet` and `KindUnknown` are public content-kind values in v0.5.0, but the default router does not auto-detect Spreadsheet and the default registry falls back to text compression for unknown/unregistered kinds.

### JSON — `KindJSON`

Detection: trimmed content starts with `{` or `[` and `json.Valid` returns true.

```bash
cat <<'EOF' | headroom compress --aggressiveness 0.8 --no-reversible
{"items":[{"id":1,"ok":true,"debug":null},{"id":2,"ok":true,"debug":null},{"id":3,"ok":true,"debug":null},{"id":4,"ok":true,"debug":null},{"id":5,"ok":true,"debug":null},{"id":6,"ok":true,"debug":null}],"score":3.14159}
EOF
# Example output includes an object-array summary and score "3.14" in aggressive mode.
```

### Code — `KindCode`

Detection: code keywords such as `func`, `def`, `class`, `return`, `import`, `struct`, `async` across multiple lines.

```bash
cat <<'EOF' | headroom compress --no-reversible
package main
// comment removed
func main() {
    err := run() // keep code, drop comment
    if err != nil { return }
}
EOF
# Output removes comments/blank lines while keeping err/return anchors.
```

### Text — `KindText`

Detection: default fallback.

```bash
cat <<'EOF' | headroom compress --aggressiveness 0.5 --no-reversible
this is the line with a repeated message
this is the line with a repeated message
this is the line with a repeated message
EOF
# Example output: line repeated message [x3]
```

### Diff — `KindDiff`

Detection: `diff --git`, `@@`, or unified diff headers.

```bash
cat ./testdata/diff/sample.diff | headroom compress --enable-pipeline --query panic --no-reversible
# Output keeps file/hunk headers and query-matching lines, omitting lower-signal diff lines.
```

### Log — `KindLog`

Detection: repeated log levels or high-priority lines.

```bash
cat <<'EOF' | headroom compress --enable-pipeline --no-reversible
[INFO] worker started
[INFO] worker heartbeat
[ERROR] database unavailable
[DEBUG] retry scheduled
EOF
# Output keeps ERROR and may omit low-priority log lines for long logs.
```

### Search — `KindSearch`

Detection: two or more `filename:line:match` style lines.

```bash
cat <<'EOF' | headroom compress --enable-pipeline --query TODO --no-reversible
main.go:10:func main() {}
main.go:11:// TODO handle error
router.go:22:return KindText
router.go:23:// TODO add kind
EOF
# Output groups by file and keeps query matches.
```

### Tabular — `KindTabular`

Detection: matching comma/tab counts across rows, or Markdown table shape.

```bash
cat <<'EOF' | headroom compress --no-reversible
name,status,count
api,ok,10
worker,error,2
EOF
# Current v0.5.0 default registry falls back to text compression for KindTabular.
```

### Spreadsheet — `KindSpreadsheet`

Detection: public kind reserved for multi-column/cell-aware data; not auto-detected by the default router in v0.5.0.

```go
registry := headroom.NewCompressorRegistry()
registry.Register(headroom.NewCompressorFunc(headroom.KindSpreadsheet, func(content string, opts headroom.Options) (string, error) {
	return "spreadsheet summary: " + content, nil
}))
out, _ := registry.Compress(headroom.KindSpreadsheet, "A1,B1\nA2,B2", headroom.Options{})
fmt.Println(out)
```

### HTML — `KindHTML`

Detection: document-like HTML structure.

```bash
cat <<'EOF' | headroom compress --enable-pipeline --no-reversible
<!doctype html><html><head><style>.x{color:red}</style></head><body><!-- comment --><p>Hello</p><script>alert(1)</script></body></html>
EOF
# Output removes comments and script/style blocks in Pipeline mode.
```

### Unknown — `KindUnknown`

Detection: API-reserved; useful for custom registries.

```go
registry := headroom.NewCompressorRegistry()
registry.Register(headroom.NewCompressorFunc(headroom.KindText, func(content string, opts headroom.Options) (string, error) {
	return "fallback:" + content, nil
}))
out, _ := registry.Compress(headroom.KindUnknown, "???", headroom.Options{})
fmt.Println(out) // fallback:???
```

---

## 🎚️ Compression Modes

### Aggressiveness

| Range | Name | Behavior |
|------|------|----------|
| `0.0-0.3` | Conservative / 保守 | Minimal loss; JSON arrays often pass through |
| `0.3-0.7` | Standard / 标准 | Balanced defaults for agent context |
| `0.7-1.0` | Aggressive / 激进 | More lossy transforms; JSON floats become 2-decimal strings |

```go
for _, a := range []float64{0.2, 0.5, 0.8} {
	out, _ := headroom.CompressString(`{"score":3.14159,"debug":null}`, headroom.Options{Aggressiveness:a, Reversible:false})
	fmt.Println(a, out)
}
```

### TokenLimit

Skip compression for messages below a threshold.

```go
res, _ := headroom.Compress([]headroom.Message{{Role:"user", Content:"short text"}}, headroom.Options{
	TokenLimit: 100,
	Reversible: false,
})
fmt.Println(res.Steps[0].Skipped, res.Steps[0].Reason) // true below token limit
```

### TokenBudget

`TokenBudget` activates Pipeline mode and tells the policy the target token count.

```go
longContext := `INFO worker heartbeat ok
INFO worker heartbeat ok
INFO worker heartbeat ok
ERROR payment failed`
out, _ := headroom.CompressString(longContext, headroom.Options{
	TokenBudget: 500,
	Reversible: false,
})
fmt.Println(out)
```

---

## 🔥 Killer Features

### 🏷️ Tag Protector

Never worry about compression mangling your structured outputs. Custom XML-like tags such as `<thinking>`, `<tool_call>`, and `<my_tag>` are protected in Pipeline mode.

```go
protector := headroom.NewTagProtector()
protected := protector.Protect(`before <tool_call>{"name":"search"}</tool_call> after`)
restored, warnings := protector.Restore(protected)
fmt.Println(restored, warnings)
```

Custom tags are detected automatically when they are not standard HTML tags:

```go
out, _ := headroom.CompressString(`<audit_trace><id>123</id></audit_trace>
INFO repeated
INFO repeated`, headroom.Options{EnablePipeline:true, Reversible:false})
fmt.Println(out) // custom audit_trace block is preserved
```

### 🔙 Reversible Compression (CCR)

Compress aggressively, recover losslessly. Every compressed output can append a retrieval marker like `[headroom:retrieve id=v2_...]` when `Reversible=true`.

```go
store := headroom.NewCCR(headroom.CCRConfig{
	TTL: 24 * time.Hour,
	MaxEntries: 10000,
})
id := store.Store("original content", "compressed", headroom.KindText)
original, found := store.Retrieve(id)
entries, bytes := store.Stats()
fmt.Println(id, original, found, entries, bytes)
```

### ⚡ KV Cache Friendly

The `CacheAligner` prefixes output so identical configs produce identical prefixes — boosting provider-side cache hit rates and saving even more.

```go
aligner := headroom.NewCacheAligner(headroom.CacheAlignerConfig{Enabled:true, Version:"v0.5"})
fmt.Println(aligner.Align("compressed context"))
// [headroom/v0.5]
// compressed context
```

### 🔌 Pluggable Architecture

Need a custom compressor? Implement the `Compressor` interface and register it — no core code changes needed.

```go
type redactor struct{}
func (redactor) Kind() headroom.ContentKind { return headroom.KindText }
func (redactor) Compress(content string, opts headroom.Options) (string, error) {
	return strings.ReplaceAll(content, "secret", "[redacted]"), nil
}

registry := headroom.NewCompressorRegistry()
registry.Register(redactor{})
out, _ := registry.Compress(headroom.KindText, "secret token", headroom.Options{})
fmt.Println(out)
```

`CompressorFunc` is the lightweight shortcut:

```go
registry.Register(headroom.NewCompressorFunc(headroom.KindText,
	func(content string, opts headroom.Options) (string, error) {
		return strings.ToUpper(content), nil
	},
))
```

### 🌐 OpenAI-Compatible Proxy

Drop-in replacement. Point your client to `http://localhost:8787/v1/chat/completions` and every message is transparently compressed.

```bash
HEADROOM_API_KEY="$OPENAI_API_KEY" headroom proxy --upstream https://api.openai.com/v1 --port 8787
```

---

## 🌐 HTTP Proxy Guide

### Endpoints

- `POST /v1/chat/completions` — OpenAI-compatible chat completions proxy.
- `GET /healthz` — health check.

```bash
curl -s http://localhost:8787/healthz
# {"status":"ok","version":"v0.5.0","uptime":"..."}
```

### Authentication

The proxy forwards the incoming `Authorization` header. If the request has no `Authorization`, it falls back to `HEADROOM_API_KEY`.

```bash
HEADROOM_API_KEY="$OPENAI_API_KEY" headroom proxy --port 8787 &

curl -s http://localhost:8787/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hello"}]}'
```

### Streaming limitation

`stream:true` returns HTTP 400 in v0.5.0.

```bash
curl -i http://localhost:8787/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"model":"gpt-4o-mini","stream":true,"messages":[{"role":"user","content":"hello"}]}'
# HTTP/1.1 400 Bad Request
# {"error":"streaming not supported in v0.5.0"}
```

### Request ID

`X-Request-ID` is forwarded if present; otherwise an 8-byte random hex ID is generated and returned.

```bash
curl -i http://localhost:8787/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -H 'X-Request-ID: demo-123' \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hello"}]}'
```

### Error format

Errors are JSON with `Content-Type: application/json; charset=utf-8`.

```bash
curl -i http://localhost:8787/v1/chat/completions -d 'not-json'
# {"error":"invalid json: ..."}
```

### Timeouts, request body limit, graceful shutdown

- Dial timeout: `10s`
- TLS handshake timeout: `10s`
- Response header timeout: `30s`
- Overall upstream client timeout: `60s`
- Max request body: `50MB`
- `SIGTERM` / interrupt: graceful drain up to `30s`

```bash
# Request body limit example: avoid sending files larger than 50MB.
wc -c payload.json
curl -s http://localhost:8787/v1/chat/completions \
  -H 'Content-Type: application/json' \
  --data-binary @payload.json

# Graceful shutdown.
pkill -TERM headroom
```

---

## 🔤 Tokenizer Guide

| 后端 | 值 | 精度 | 依赖 | 适用场景 |
|------|-----|------|------|---------|
| Fallback | `fallback` | 粗略（pure-Go rune/word counting） | 零依赖 | 默认，通用场景 |
| tiktoken | `tiktoken` | 预留精确后端；v0.5.0 stub | 零外部依赖；可 fallback | OpenAI 模型规划 |
| HuggingFace | `huggingface` | 预留精确后端；v0.5.0 stub | 零外部依赖；可 fallback | 开源模型规划 |

Headroom Go remains **zero external dependencies** in v0.5.0. Non-fallback tokenizer backends currently return a warning and fall back when `AllowFallback=true`.

```go
tok, warnings, err := headroom.NewTokenizer(headroom.TokenizerConfig{
	Backend: headroom.TokenizerTiktoken,
	AllowFallback: true,
})
if err != nil { panic(err) }
count, _ := tok.Count("hello world")
fmt.Println(tok.Name(), count, warnings)
```

CLI example:

```bash
echo 'hello world' | headroom compress --tokenizer-backend fallback --stats --no-reversible
```

---

## 🔄 Pipeline Mode

| Capability | Legacy | Pipeline |
|------------|--------|----------|
| Activation | default | `EnablePipeline=true`, `TokenBudget>0`, or `Query!=""` |
| Policy | direct route + compress | `DefaultCompressionPolicy(Aggressiveness)` |
| Modes | implicit | Conservative / Standard / Aggressive policy modes |
| TokenBudget | ignored unless Pipeline activated | policy target token count |
| Query | ignored unless Pipeline activated | diff/search line retention signal |
| Observer | step notifications | detailed transform step notifications |
| Tag protection | post-processing oriented | protects custom XML-like tags before transforms |

Policy modes are derived from `Aggressiveness`: `<0.3` → `PolicyConservative`, `0.3-0.7` → `PolicyStandard`, `>=0.7` → `PolicyAggressive`.

```go
policy := headroom.DefaultCompressionPolicy(0.8)
fmt.Println(policy.Mode) // aggressive
```

Enable Pipeline explicitly:

```go
diffText := "diff --git a/main.go b/main.go\n@@ -1,3 +1,4 @@\n func main() {\n+\tpanic(\"boom\")\n }\n"
res, _ := headroom.Compress([]headroom.Message{{Role:"user", Content: diffText}}, headroom.Options{
	EnablePipeline: true,
	TokenBudget: 1000,
	Query: "panic",
	Reversible: false,
})
fmt.Println(res.CompressedTokens, res.Steps)
```

Observer callback:

```go
type logObserver struct{}
func (logObserver) ObserveCompressionStep(step headroom.CompressionStep) {
	fmt.Println(step.Name, step.Kind, step.TokensBefore, step.TokensAfter, step.Skipped, step.Reason)
}

_, _ = headroom.CompressString("INFO ok\nINFO ok\n", headroom.Options{
	EnablePipeline: true,
	Observer: logObserver{},
	Reversible: false,
})
```

---

## 🔙 Reversible Compression 详解

CCR (Context Compression Retrieval) stores original content in memory and appends a retrieval marker to compressed output when `Options.Reversible=true`.

Runtime behavior in v0.5.0:

- `Store(original, compressed, kind)` returns a deterministic ID.
- Actual ID format emitted by `Store`: `v2_{sha256前12字符}`.
- `CCRIDVersion = "v3"` exists as a public version constant, but the runtime store currently uses `LegacyCCRIDVersion = "v2"` for compatibility.
- TTL default: `24h`.
- MaxEntries default: `10000`.
- Background GC interval: every `30m`.
- `Stats()` returns active entry count and total original bytes.

```go
store := headroom.NewCCR(headroom.CCRConfig{}) // TTL 24h, MaxEntries 10000
id := store.Store("very long original", "short", headroom.KindText)
original, ok := store.Retrieve(id)
entries, bytes := store.Stats()
fmt.Println(id, original, ok, entries, bytes)
```

End-to-end via `Compress`:

```go
res, _ := headroom.Compress([]headroom.Message{{Role:"user", Content:"INFO repeat\nINFO repeat\nERROR keep\n"}}, headroom.DefaultOptions())
fmt.Println(res.Messages[0].Content) // may include [headroom:retrieve id=v2_...]
```

---

## 🔌 Custom Compressor

The compressor interface is:

```go
type Compressor interface {
	Kind() headroom.ContentKind
	Compress(content string, opts headroom.Options) (string, error)
}
```

Full implementation:

```go
package main

import (
	"fmt"
	"strings"

	headroom "github.com/superops-team/headroom-go"
)

type piiCompressor struct{}

func (piiCompressor) Kind() headroom.ContentKind { return headroom.KindText }

func (piiCompressor) Compress(content string, opts headroom.Options) (string, error) {
	content = strings.ReplaceAll(content, "alice@example.com", "[email]")
	return content, nil
}

func main() {
	registry := headroom.NewCompressorRegistry()
	registry.Register(piiCompressor{})
	out, err := registry.Compress(headroom.KindText, "contact alice@example.com", headroom.Options{})
	if err != nil { panic(err) }
	fmt.Println(out)
}
```

`CompressorFunc` shortcut:

```go
registry := headroom.NewCompressorRegistry()
registry.Register(headroom.NewCompressorFunc(headroom.KindJSON, func(content string, opts headroom.Options) (string, error) {
	return headroom.SmartCrushJSON(content, headroom.SmartCrushConfig{Aggressiveness: opts.Aggressiveness})
}))
```

---

## 📊 Observability

### Types

```go
type Observer interface {
	ObserveCompressionStep(step headroom.CompressionStep)
}

type CompressionStep struct {
	Name         string
	Kind         string
	TokensBefore int
	TokensAfter  int
	Skipped      bool
	Reason       string
}

type Warning struct {
	Code      string
	Component string
	Message   string
}
```

Use `Result.Warnings` and `Result.Steps` for post-run inspection:

```go
type observer struct{}
func (observer) ObserveCompressionStep(step headroom.CompressionStep) {
	fmt.Printf("step=%s kind=%s skipped=%v reason=%s\n", step.Name, step.Kind, step.Skipped, step.Reason)
}

res, _ := headroom.Compress([]headroom.Message{{Role:"user", Content:"short"}}, headroom.Options{
	TokenLimit: 100,
	Observer: observer{},
	Reversible: false,
})
for _, w := range res.Warnings { fmt.Println(w.Code, w.Component, w.Message) }
for _, s := range res.Steps { fmt.Println(s.Name, s.Skipped, s.Reason) }
```

---

## 📊 Real-World Performance

Benchmarks on Intel Xeon (32 cores), Go 1.22:

| Benchmark | Throughput | What It Means |
|-----------|-----------|---------------|
| Content Detection (1MB) | **390 MB/s** | 10 content types represented/detected in ~2.5ms |
| Tokenizer (1MB) | **95 MB/s** | Token counting at wire speed |
| Diff Compressor (5k lines) | **6.1M lines/s** | Entire PR diffs compressed instantly |
| Log Compressor (50k lines) | **2.3M lines/s** | Production log files in milliseconds |
| End-to-End (mixed) | **650 ops/s** | 650 messages compressed per second |

```bash
go test -bench=. -benchtime=1s ./...
```

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
            │SmartCrush│ │  Code    │ │  Text    │  ... more kinds
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

```bash
# Build the architecture into a single static-style binary (stdlib-only module).
go build -o /tmp/headroom ./cmd/headroom && /tmp/headroom version
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

```bash
# CI/CD error summarizer pre-compression.
go test ./... 2>&1 | headroom compress --enable-pipeline --query FAIL --stats --no-reversible
```

---

## 🚢 Deployment

### systemd

```ini
[Unit]
Description=Headroom Go OpenAI-compatible proxy
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
Environment=HEADROOM_API_KEY=replace-me
ExecStart=/usr/local/bin/headroom proxy --port 8787 --upstream https://api.openai.com/v1 --enable-pipeline --token-budget 4000
Restart=always
RestartSec=3
User=headroom
Group=headroom

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now headroom
curl -s http://localhost:8787/healthz
```

### Docker

```dockerfile
FROM golang:1.22 AS build
WORKDIR /src
COPY . .
RUN go build -o /headroom ./cmd/headroom

FROM gcr.io/distroless/base-debian12
COPY --from=build /headroom /headroom
EXPOSE 8787
ENTRYPOINT ["/headroom", "proxy", "--port", "8787"]
```

```bash
docker build -t headroom-go:v0.5.0 .
docker run --rm -p 8787:8787 -e HEADROOM_API_KEY="$OPENAI_API_KEY" headroom-go:v0.5.0
```

### Production recommendations

- Put the proxy behind your normal ingress/API gateway.
- Use `HEADROOM_API_KEY` via a secret manager, not shell history.
- Start with `--aggressiveness 0.5`, then raise only for noisy logs/search/diffs.
- Use `--enable-pipeline --token-budget <N>` for agent workloads with hard context budgets.
- Monitor 400/502 rates and upstream latency; the proxy timeout is 60s overall.

---

## 🔍 Troubleshooting

### 压缩后内容比原文长？

Headroom intentionally falls back to the original if output bytes are not shorter. This can happen with tiny inputs or when reversible CCR metadata would be larger than the savings.

```bash
echo short | headroom compress --stats
```

### 流式请求报错？

`stream:true` is not supported in v0.5.0 and returns 400.

```bash
curl -i http://localhost:8787/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"stream":true,"messages":[{"role":"user","content":"hello"}]}'
```

### CCR 检索返回 false？

Common causes: ID not stored in this process, TTL expired, entry was evicted by `MaxEntries`, or the process restarted because CCR is in-memory.

```go
store := headroom.NewCCR(headroom.CCRConfig{TTL: time.Second})
id := store.Store("original", "short", headroom.KindText)
time.Sleep(2 * time.Second)
_, ok := store.Retrieve(id)
fmt.Println(ok) // false
```

### 端口被占用？

```bash
headroom proxy --port 9876
curl -s http://localhost:9876/healthz
```

### Token 估算不准？

The default fallback tokenizer is approximate and dependency-free. `tiktoken` and `huggingface` are reserved backends in v0.5.0 and fall back when `AllowFallback=true`.

```bash
echo 'hello world' | headroom compress --tokenizer-backend fallback --stats --no-reversible
```

---

## 🔧 Development

```bash
# Clone & test
git clone https://github.com/superops-team/headroom-go.git
cd headroom-go

# Run all tests with race detection
go test -race -count=1 ./...

# Coverage report (target: 92.8%)
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

Please ensure tests pass (`go test -race ./...`) and coverage doesn't drop. See [CONTRIBUTING.md](CONTRIBUTING.md) for the full guide.

```bash
go test -race ./...
```

---

## 📄 License

MIT — see [LICENSE](LICENSE).

```bash
grep -n "MIT" LICENSE
```

---

<p align="center">
  <sub>Built with ❤️ by <a href="https://github.com/superops-team">superops-team</a> · Powered by pure Go · No snakes were harmed 🐍</sub>
</p>
