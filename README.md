# Headroom Go

[![Go Report Card](https://goreportcard.com/badge/github.com/chopratejas/headroom-go)](https://goreportcard.com/report/github.com/chopratejas/headroom-go)
[![Test Coverage](https://img.shields.io/badge/test%20coverage-100%25-brightgreen)](https://github.com/chopratejas/headroom-go)

A Go implementation of [headroom](https://github.com/chopratejas/headroom) - intelligent context compression for LLM applications. Reduce token usage by up to 70% while preserving semantic meaning.

---

## Features

- **Content-Type Aware Compression**: Automatically detects JSON, code, and plain text
- **Multi-level Compression Strategies**: Conservative, Standard, and Aggressive modes
- **Reversible Compression**: Store original content locally for lossless retrieval
- **Cache Alignment**: Prefix alignment to improve Provider KV cache hit rate
- **HTTP Proxy Mode**: Drop-in replacement for OpenAI Chat Completions API
- **Zero External Dependencies**: Uses only Go standard library

---

## Installation

```bash
go install github.com/chopratejas/headroom-go/cmd/headroom@latest
```

Or add to your project:
```bash
go get github.com/chopratejas/headroom-go
```

---

## Usage

### CLI

```bash
# Compress text from stdin
echo "Hello World" | headroom compress

# Compress with aggressive mode
echo '{"data": [1, 2, 3, 4, 5]}' | headroom compress -a 0.8

# Start proxy server
headroom proxy --upstream https://api.openai.com/v1 --port 8080
```

### API

```go
package main

import (
    "fmt"
    "github.com/chopratejas/headroom-go"
)

func main() {
    messages := []headroom.Message{
        {Role: "user", Content: "Your long text here..."},
    }
    
    result, err := headroom.Compress(messages, headroom.Options{
        Aggressiveness: 0.5,
        Reversible:     true,
        AlignPrefix:    true,
    })
    
    fmt.Printf("Compressed %d tokens to %d tokens (%.1f%% savings)",
        result.OriginalTokens,
        result.CompressedTokens,
        result.Savings*100,
    )
}
```

---

## Compression Strategies

| Mode | Aggressiveness | Description |
|------|----------------|-------------|
| Conservative | 0.0 - 0.3 | Remove whitespace, empty lines, comments |
| Standard | 0.3 - 0.7 | + Remove stopwords, collapse short arrays |
| Aggressive | 0.7 - 1.0 | + Truncate numbers, convert booleans to strings |

---

## Compression Algorithms

1. **SmartCrusher (JSON)**: Removes redundant fields, collapses arrays, truncates values
2. **CodeCompressor**: Removes comments, folds long function bodies, preserves error handling
3. **TextCompressor**: Deduplicates lines, removes stopwords, folds long paragraphs
4. **CCR**: Compress-Cache-Retrieve for reversible compression with ID-based retrieval

---

## Proxy Mode

Run as an HTTP proxy compatible with OpenAI Chat Completions API:

```bash
headroom proxy \
  --upstream https://api.openai.com/v1 \
  --port 8080 \
  --apikey your-api-key
```

Then use `http://localhost:8080/v1/chat/completions` instead of OpenAI's endpoint.

---

## Development

```bash
# Run tests
go test -race -v ./...

# Build
go build -o headroom ./cmd/headroom

# Format
gofmt -w .
```

---

## License

MIT

---

---

# Headroom Go (中文)

[![Go Report Card](https://goreportcard.com/badge/github.com/chopratejas/headroom-go)](https://goreportcard.com/report/github.com/chopratejas/headroom-go)

[headroom](https://github.com/chopratejas/headroom) 的 Go 语言实现 - 为 LLM 应用提供智能上下文压缩。在保留语义的同时，将 token 使用量减少多达 70%。

---

## 功能特性

- **内容类型感知压缩**: 自动检测 JSON、代码和纯文本
- **多级压缩策略**: 保守、标准和激进模式
- **可逆压缩**: 本地存储原始内容，支持无损检索
- **缓存对齐**: 前缀对齐提高 Provider KV 缓存命中率
- **HTTP 代理模式**: 即插即用替换 OpenAI Chat Completions API
- **零外部依赖**: 仅使用 Go 标准库

---

## 安装

```bash
go install github.com/chopratejas/headroom-go/cmd/headroom@latest
```

或添加到项目中：
```bash
go get github.com/chopratejas/headroom-go
```

---

## 使用方法

### CLI

```bash
# 从标准输入压缩文本
echo "Hello World" | headroom compress

# 使用激进模式压缩
echo '{"data": [1, 2, 3, 4, 5]}' | headroom compress -a 0.8

# 启动代理服务器
headroom proxy --upstream https://api.openai.com/v1 --port 8080
```

### API

```go
package main

import (
    "fmt"
    "github.com/chopratejas/headroom-go"
)

func main() {
    messages := []headroom.Message{
        {Role: "user", Content: "Your long text here..."},
    }
    
    result, err := headroom.Compress(messages, headroom.Options{
        Aggressiveness: 0.5,
        Reversible:     true,
        AlignPrefix:    true,
    })
    
    fmt.Printf("Compressed %d tokens to %d tokens (%.1f%% savings)",
        result.OriginalTokens,
        result.CompressedTokens,
        result.Savings*100,
    )
}
```

---

## 压缩策略

| 模式 | 激进程度 | 描述 |
|------|----------|------|
| Conservative（保守） | 0.0 - 0.3 | 移除空白、空行、注释 |
| Standard（标准） | 0.3 - 0.7 | + 移除停用词，折叠短数组 |
| Aggressive（激进） | 0.7 - 1.0 | + 截断数字，将布尔值转为字符串 |

---

## 压缩算法

1. **SmartCrusher (JSON)**: 移除冗余字段、折叠数组、截断值
2. **CodeCompressor**: 移除注释、折叠长函数体、保留错误处理
3. **TextCompressor**: 去重行、移除停用词、折叠长段落
4. **CCR**: 压缩-缓存-检索，支持基于 ID 的可逆压缩

---

## 代理模式

以 HTTP 代理方式运行，兼容 OpenAI Chat Completions API：

```bash
headroom proxy \
  --upstream https://api.openai.com/v1 \
  --port 8080 \
  --apikey your-api-key
```

然后使用 `http://localhost:8080/v1/chat/completions` 代替 OpenAI 的端点。

---

## 开发

```bash
# 运行测试
go test -race -v ./...

# 构建
go build -o headroom ./cmd/headroom

# 格式化
gofmt -w .
```

---

## 许可证

MIT