---
title: Quick Start
---

# Quick Start

## Compress Text

```bash
echo "hello world hello world hello world" | headroom compress --stats
```

## Compress JSON

```bash
headroom compress --input=data.json --output=compressed.json --stats
```

## Start Proxy

```bash
headroom proxy --port=8787
```

Then configure your LLM client to use `http://localhost:8787/v1`.

## MCP Server

```bash
headroom mcp serve
```

Add to Claude Code config:

```json
{
  "mcpServers": {
    "headroom": {
      "command": "headroom",
      "args": ["mcp", "serve"]
    }
  }
}
```

## Wrap IDE

```bash
headroom wrap claude --apply
headroom wrap codex --apply
headroom wrap generic
```

## Go SDK

```go
import headroom "github.com/superops-team/headroom-go"

opts := headroom.DefaultOptions()
result, _ := headroom.Compress(messages, opts)
compressed, _ := headroom.CompressString(content, opts)
```
