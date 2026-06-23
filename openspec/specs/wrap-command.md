# Spec: headroom wrap 命令

**版本:** v0.7.0-wrap
**日期:** 2026-06-22
**优先级:** P0
**状态:** 待确认

---

## 1. 背景

Headroom (Python) 的 `headroom wrap` 是最核心的生态入口——启动本地代理服务器，修改 IDE/Agent 的 API endpoint 配置，透明拦截并压缩所有 LLM 请求。headroom-go 已有 HTTP proxy 基础能力，但缺少自动配置 IDE 的 wrap 命令。

---

## 2. 目标

```bash
headroom wrap <agent>          # 启动代理 + 打印配置指令
headroom wrap <agent> --apply  # 启动代理 + 自动修改配置文件
headroom wrap <agent> --port 18787  # 指定端口
```

### 2.1 支持的 Agent

| Agent | 配置方式 | 自动 apply |
|-------|---------|:----------:|
| `claude` (Claude Code) | `~/.claude/settings.json` 修改 API endpoint | ✅ |
| `codex` (OpenAI Codex) | `~/.codex/config.yaml` 修改 base_url | ✅ |
| `cursor` | 打印手动配置步骤 | ❌ (无标准配置文件) |
| `copilot` (GitHub Copilot CLI) | 环境变量 `OPENAI_BASE_URL` | ✅ |
| `aider` | `--openai-api-base` 参数 | ❌ |
| `generic` | 打印通用配置指令 | ❌ |

### 2.2 工作流程

```
1. 启动本地 headroom proxy（已有能力）
2. 检测目标 Agent 是否安装
3. 生成/修改 Agent 配置，指向本地 proxy
4. 打印启动指令
5. 监听 SIGINT，退出时恢复原配置
```

---

## 3. 技术方案

### 3.1 架构

```
cmd/headroom/main.go
  └── case "wrap":
        └── internal/wrap/          # 新增包
              ├── wrap.go           # 主逻辑
              ├── claude.go         # Claude Code 配置
              ├── codex.go          # Codex 配置
              ├── copilot.go        # Copilot CLI 配置
              └── config.go         # 通用配置管理
```

### 3.2 配置备份与恢复

```
wrap 启动时:
  1. 检测目标配置文件是否存在
  2. 备份原配置 → {config}.headroom.bak
  3. 修改配置指向 http://127.0.0.1:{port}/v1
  4. 启动 proxy goroutine

wrap 退出时 (SIGINT/SIGTERM):
  1. 恢复原配置文件
  2. 删除备份
  3. 关闭 proxy
```

### 3.3 Claude Code 配置示例

```json
// ~/.claude/settings.json (修改后)
{
  "api": {
    "baseUrl": "http://127.0.0.1:18787/v1"
  }
}
```

### 3.4 Codex 配置示例

```yaml
# ~/.codex/config.yaml (修改后)
api_base: http://127.0.0.1:18787/v1
```

---

## 4. 文件变更

| 操作 | 文件 | 说明 |
|------|------|------|
| **新建** | `internal/wrap/wrap.go` | 主逻辑 (~120 行) |
| **新建** | `internal/wrap/claude.go` | Claude Code 配置 (~60 行) |
| **新建** | `internal/wrap/codex.go` | Codex 配置 (~60 行) |
| **新建** | `internal/wrap/copilot.go` | Copilot CLI 配置 (~40 行) |
| **新建** | `internal/wrap/config.go` | 通用配置管理 (~80 行) |
| **新建** | `internal/wrap/wrap_test.go` | 测试 (~100 行) |
| **修改** | `cmd/headroom/main.go` | 添加 `wrap` 子命令 (~40 行) |

---

## 5. 验收标准

- [ ] `headroom wrap claude` 启动 proxy + 打印配置指令
- [ ] `headroom wrap claude --apply` 自动修改 Claude Code 配置
- [ ] `headroom wrap codex --apply` 自动修改 Codex 配置
- [ ] Ctrl+C 退出时恢复原配置
- [ ] 异常退出（kill -9）后下次启动检测并提示残留备份
- [ ] `headroom wrap generic` 打印通用配置指令
- [ ] 端口冲突时给出明确错误提示

---

## 6. 时间估算

| 阶段 | 预估 |
|------|------|
| config.go 备份恢复 | 1h |
| claude.go + codex.go + copilot.go | 2h |
| wrap.go 主逻辑 | 1h |
| cmd 集成 | 0.5h |
| 测试 | 1h |
| **总计** | **~5.5h** |
