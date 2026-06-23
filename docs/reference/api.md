---
title: API Reference
---

# Go API Reference

## Core Types

### Message

```go
type Message struct {
    Role    string // "system", "user", "assistant", "tool"
    Content string
    Name    string
}
```

### Options

```go
type Options struct {
    Aggressiveness  float64         // 0.0-1.0
    Reversible      bool            // Enable CCR
    AlignPrefix     bool            // KV cache alignment
    TokenLimit      int             // Skip below threshold
    TokenizerConfig TokenizerConfig
    TokenBudget     int             // Pipeline target
    Query           string          // Relevance scoring
    EnablePipeline  bool            // Use Pipeline path
    Observer        Observer        // Step notifications
}
```

### Result

```go
type Result struct {
    Messages         []Message
    CompressedTokens int
    OriginalTokens   int
    Savings          float64
    Warnings         []Warning
    Steps            []CompressionStep
}
```

## Core Functions

```go
func DefaultOptions() Options
func Compress(messages []Message, opts Options) (*Result, error)
func CompressString(content string, opts Options) (string, error)
func NewCompressionEngine(opts Options) (*CompressionEngine, []Warning)
func NewDefaultPipeline() *Pipeline
```

## Integrations

- [OpenAI Go SDK](/integrations/go-openai)
- [langchaingo](/integrations/langchaingo)
- [Ollama](/integrations/ollama)
- [Kubernetes](/integrations/k8s)
