---
title: CLI Reference
---

# CLI Reference

## Commands

| Command | Description |
|---------|-------------|
| `compress` | Compress stdin or file |
| `proxy` | Start HTTP proxy (OpenAI compatible) |
| `mcp serve` | Start MCP Server (stdio mode) |
| `wrap <agent>` | Start proxy + configure IDE |
| `version` | Print version |

## compress

```bash
headroom compress [flags]

Flags:
  --aggressiveness float   Compression strength 0.0-1.0 (default 0.5)
  --no-reversible          Disable reversible compression
  --no-align               Disable prefix alignment
  --tokenizer-backend string  Tokenizer: fallback/tiktoken/huggingface
  --token-budget int       Target token budget (0 = unlimited)
  --enable-pipeline        Use Pipeline compression path
  --query string           Query for diff/search scoring
  --input string           Input file (default stdin)
  --output string          Output file (default stdout)
  --stats                  Print token statistics to stderr
```

## proxy

```bash
headroom proxy [flags]

Flags:
  --port int               Listen port (default 8787)
  --upstream string        Upstream LLM API base URL
  --aggressiveness float   Compression strength (default 0.5)
  --no-reversible          Disable reversible compression
  --enable-pipeline        Use Pipeline mode
  --token-budget int       Target token budget
```

## mcp

```bash
headroom mcp serve          # Start MCP Server
```

## wrap

```bash
headroom wrap <agent> [flags]

Agents: claude, codex, copilot, generic

Flags:
  --port int       Proxy port (default 18787)
  --upstream string  Upstream LLM API URL
  --apply          Auto-apply configuration changes
```
