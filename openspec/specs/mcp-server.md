# Spec: MCP Server 集成

**版本:** v0.7.0-mcp
**日期:** 2026-06-22
**优先级:** P0
**状态:** 待确认

---

## 1. 背景与动机

MCP (Model Context Protocol) 是 2026 年 AI Agent 生态的核心协议。Claude Code、Codex、Cursor、Copilot 等主流 Agent 工具都通过 MCP 发现和调用外部工具。Headroom (Python) 已内置完整的 MCP Server（4 工具），headroom-go 目前仅提供 HTTP proxy，无法被 MCP 客户端直接发现。

Go 实现 MCP Server 的优势：
- 单二进制，启动 <10ms（Python 版 ~200ms）
- 内存占用 <5MB（Python 版 ~50MB）
- 零依赖，无需安装 Python/MCP 包

---

## 2. 目标

为 headroom-go CLI 添加 `mcp` 子命令，实现标准 MCP Server（stdio 传输），提供 4 个压缩相关工具。

### 2.1 子命令

```bash
headroom mcp serve              # 启动 MCP Server（stdio 模式）
headroom mcp install            # 生成 Claude Code / Codex 配置片段
headroom mcp install --client claude   # 仅 Claude Code
headroom mcp install --client codex    # 仅 Codex
headroom mcp install --client cursor   # 仅 Cursor
```

### 2.2 MCP 工具

| 工具名 | 功能 | 参数 | 返回值 |
|--------|------|------|--------|
| `headroom_compress` | 压缩文本内容 | `content` (string), `aggressiveness` (float, 0.5), `reversible` (bool, true), `content_kind` (string, "auto") | `{compressed, original_tokens, compressed_tokens, savings, steps, warnings}` |
| `headroom_retrieve` | 通过 ID 检索原始内容 | `retrieve_id` (string) | `{found, original, compressed, kind}` |
| `headroom_stats` | 获取压缩统计 | 无 | `{total_compressions, total_savings, avg_savings, cache_entries, cache_bytes}` |
| `headroom_read` | 通过 CCR 缓存读文件 | `path` (string) | `{content, compressed, tokens}` |

---

## 3. 技术方案

### 3.1 架构

```
cmd/headroom/main.go
  └── case "mcp":
        └── internal/mcp/          # 新增包
              ├── server.go        # MCP Server 主逻辑
              ├── tools.go         # 4 个工具实现
              ├── transport.go     # stdio JSON-RPC 传输
              └── install.go       # 配置生成
```

### 3.2 协议实现

遵循 MCP 规范 (2024-11-05)：
- JSON-RPC 2.0 over stdio
- `initialize` → `tools/list` → `tools/call` 生命周期
- 支持 `notifications/initialized`

### 3.3 工具实现细节

**headroom_compress:**
```json
{
  "name": "headroom_compress",
  "description": "Compress text content to reduce token usage. Supports auto-detection of 10 content types.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "content": {"type": "string", "description": "Text content to compress"},
      "aggressiveness": {"type": "number", "default": 0.5},
      "reversible": {"type": "boolean", "default": true},
      "content_kind": {"type": "string", "default": "auto", "enum": ["auto", "text", "json", "code", "diff", "log", "search", "tabular", "spreadsheet", "html"]}
    },
    "required": ["content"]
  }
}
```

**headroom_retrieve:**
```json
{
  "name": "headroom_retrieve",
  "description": "Retrieve original content from a compression ID (format: v3_{sha256[:12]})",
  "inputSchema": {
    "type": "object",
    "properties": {
      "retrieve_id": {"type": "string", "description": "The retrieval ID from compressed output"}
    },
    "required": ["retrieve_id"]
  }
}
```

**headroom_stats:**
```json
{
  "name": "headroom_stats",
  "description": "Get session compression statistics",
  "inputSchema": {"type": "object", "properties": {}}
}
```

**headroom_read:**
```json
{
  "name": "headroom_read",
  "description": "Read and compress a file through CCR cache",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path": {"type": "string", "description": "File path to read and compress"}
    },
    "required": ["path"]
  }
}
```

### 3.4 install 命令输出

```bash
$ headroom mcp install --client claude
# Add to ~/.claude/claude_desktop_config.json:
{
  "mcpServers": {
    "headroom": {
      "command": "headroom",
      "args": ["mcp", "serve"]
    }
  }
}
```

---

## 4. 文件变更

| 操作 | 文件 | 说明 |
|------|------|------|
| **新建** | `internal/mcp/server.go` | MCP Server 主逻辑 (~150 行) |
| **新建** | `internal/mcp/tools.go` | 4 个工具实现 (~200 行) |
| **新建** | `internal/mcp/transport.go` | stdio JSON-RPC 传输 (~80 行) |
| **新建** | `internal/mcp/install.go` | 配置生成 (~60 行) |
| **新建** | `internal/mcp/server_test.go` | 单元测试 (~150 行) |
| **修改** | `cmd/headroom/main.go` | 添加 `mcp` 子命令 (~30 行) |

---

## 5. 验收标准

- [ ] `headroom mcp serve` 启动 stdio MCP Server
- [ ] `tools/list` 返回 4 个工具
- [ ] `tools/call headroom_compress` 返回压缩结果
- [ ] `tools/call headroom_retrieve` 可检索已压缩内容
- [ ] `tools/call headroom_stats` 返回统计信息
- [ ] `headroom mcp install` 输出正确配置
- [ ] Claude Code 可成功连接并调用工具
- [ ] `go test -race ./internal/mcp/...` 通过
- [ ] 内存占用 <10MB（idle）
- [ ] 启动时间 <50ms

---

## 6. 风险

| 风险 | 缓解 |
|------|------|
| MCP 协议版本兼容 | 锁定 2024-11-05 版本，后续跟进升级 |
| stdio 传输可靠性 | 使用 bufio.Scanner 逐行读取，处理 EOF |
| CCR 跨进程共享 | MCP Server 使用独立 CCR 实例，不跨进程 |

---

## 7. 时间估算

| 阶段 | 预估 |
|------|------|
| transport.go + server.go | 1h |
| tools.go (4 工具) | 2h |
| install.go | 0.5h |
| cmd/headroom 集成 | 0.5h |
| 测试 | 1h |
| Claude Code 联调 | 0.5h |
| **总计** | **~5.5h** |
