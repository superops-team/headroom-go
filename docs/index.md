---
title: headroom-go
description: Intelligent context compression for AI agents
---

# headroom-go

**Zero-dependency, single-binary context compression for AI agents.**

headroom-go compresses everything an AI agent reads — tool outputs, logs, RAG snippets, code diffs, search results, and conversation history — before sending to an LLM. It auto-detects 10 content types and applies specialized compression strategies, achieving up to 70% token savings.

## Quick Start

```bash
# Install
curl -sSL https://raw.githubusercontent.com/superops-team/headroom-go/main/install.sh | bash

# Compress
echo '{"key":"value","items":[1,2,3,4,5]}' | headroom compress --stats

# Proxy
headroom proxy --port=8787

# MCP Server (Claude Code / Codex)
headroom mcp serve
```

## Features

- **10 Content Types**: JSON, Code, Text, Diff, Log, Search, Tabular, Spreadsheet, HTML, Unknown
- **Two Compression Paths**: Legacy (fast) + Pipeline (policy-driven)
- **Reversible Compression**: Store originals, retrieve by ID
- **KV Cache Alignment**: Boost provider-side cache hit rates
- **MCP Server**: Native MCP integration for Claude Code / Codex / Cursor
- **Wrap Command**: Auto-configure IDE proxy with one command
- **Zero Dependencies**: Pure Go standard library, single binary

## Next Steps

- [Installation Guide](/getting-started/installation)
- [Quick Start](/getting-started/quickstart)
- [CLI Reference](/reference/cli)
- [API Reference](/reference/api)
