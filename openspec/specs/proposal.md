# Headroom-Go 变更提案（v0.2.0）

## Purpose

为 [headroom](https://github.com/chopratejas/headroom) 提供一个零外部依赖、可嵌入到 Go 应用或作为独立 HTTP 代理运行的 Go 实现。核心价值：在发送给 LLM 之前压缩上下文，保留语义准确性并支持可逆检索。

## Why Go?

- 单二进制部署 → 比 Python 更容易嵌入到 Agent / 工具链
- 原生并发 → Proxy 场景自然高效
- 零外部依赖 → 便于在受限环境（CI、容器最小镜像）中使用
- Go 社区对 LLM / Agent 生态的工具链存在空白

## What（v0.1 Scope）

| 交付项 | 说明 |
|--------|------|
| Library API | `Compress([]Message)` / `CompressString(string)` |
| SmartCrusher | JSON 三级压缩（保守/标准/激进），输出必为合法 JSON |
| CodeCompressor | 启发式代码压缩（去注释/空行 + >20 行函数折叠） |
| TextCompressor | 行级去重 + 43 词 stopwords + 超长段落折叠 |
| ContentRouter | O(n) 检测（JSON/Code/Text），零额外内存分配 |
| CacheAligner | 可选固定前缀（提升 Provider KV Cache） |
| CCR | 可逆存储（v1_{sha1} id + 惰性 GC + sync.RWMutex） |
| HTTP Proxy | 非流式 OpenAI 兼容代理（/v1/chat/completions + /healthz） |
| CLI | `compress` / `proxy` / `version` 子命令 |

## What NOT in v0.1

- ML 模型推理、Tree-sitter 绑定
- 流式响应（v0.1 明确拒绝）
- Proxy 层的自动 Retrieve 循环
- 跨 Agent 持久化 Memory（SQLite / Redis）
- MCP Server、Copilot OAuth、Agent wrap 集成
- `headroom learn`、`headroom perf`

## Approach

### 纯标准库（零外部依赖）

`encoding/json` / `regexp` / `sync` / `net/http` / `crypto/sha1` / `unicode/utf8`

### 纯函数优先

每个 Compressor：输入 `string` → 输出 `string`（+ error），无副作用。`CCR` 是唯一有状态模块（进程内 map）。

### TDD 驱动

每个模块：先写 `_test.go`（编译失败），再写实现（测试通过），最后重构。Tasks 按 `2.1a/2.1b` 命名体现 test-first。

### 包结构（平铺根包）

所有压缩器和 CCR、Proxy 同在根包 `headroom`，CLI 在 `cmd/headroom`（独立入口）。无需子包，简化 import 路径。

## 关键设计决策（v0.2 修订）

| 决策 | 理由 |
|------|------|
| ContentKind 枚举明确赋值 0/1/2，不用 iota | 避免后续插入新枚举值时意外破坏常量数值 |
| SmartCrusher 激进模式数字转为**字符串** `"3.14"` | 避免 `1e-10` 截断成 `1e-1` 语义破坏；加引号保证 JSON 合法 |
| CCR id 加版本前缀 `v1_` | 未来算法切换时旧 id 仍可识别 |
| TextCompressor 超长段落阈值 **30 行** | 兼顾日志场景（10-50 行 RAG 片段） |
| CacheAlignerConfig 只含 Enabled + Version | 前缀格式不需要 Kind/Aggressiveness，减少字段冗余 |
| TokenLimit 字段替代 Model | `Model` 在 spec v0.1 中从未被使用；`TokenLimit` 是实际有用的控制参数 |
| Proxy 拒绝流式请求（返回 400） | 流式场景下注入 Retrieve 逻辑极其复杂；v0.1 明确边界 |
| CLI 用标准库 `flag` 而非 cobra | 零依赖；子命令数量少（3 个）；标准库足够 |

## Risk & Mitigation

| 风险 | 等级 | 缓解 |
|------|------|------|
| 启发式压缩丢失语义 | 中 | Aggressiveness 默认 0.5（保守）、可逆模式默认开启 |
| JSON 激进模式数字截断语义破坏 | 高 | 改为字符串形式（已在 v0.2 中修正） |
| 代码压缩在冷门语言上效果差 | 低 | 注释/空行/空白收缩是通用规则；折叠逻辑对有 `{}` 的语言通用 |
| CCR 内存无限增长 | 中 | 惰性 GC（每次 Store 清理过期条目）+ TTL 默认 24h |
| 流式请求导致行为异常 | 高 | v0.1 明确返回 400，客户端不会收到意外响应 |
| Proxy 端口冲突 | 低 | CLI 检测占用并打印友好错误 + 退出码 1 |

## 开发排期规划

| Phase | 任务 | 依赖 | 预计工时 |
|-------|------|------|---------|
| Phase 1 | 项目骨架 + content_kind | 无 | 0.5d |
| Phase 2 | ContentRouter（测试先行） | Phase 1 | 0.5d |
| Phase 2 | SmartCrusher（测试先行） | Phase 1 | 1d |
| Phase 2 | CodeCompressor（测试先行） | Phase 1 | 1d |
| Phase 2 | TextCompressor（测试先行） | Phase 1 | 1d |
| Phase 3 | CacheAligner | Phase 2 | 0.25d |
| Phase 3 | CCR | Phase 2 | 0.5d |
| Phase 4 | 顶层 API + 端到端测试 | Phase 3 | 0.5d |
| Phase 5 | HTTP Proxy | Phase 4 | 0.5d |
| Phase 5 | CLI | Phase 4 | 0.5d |
| Phase 6 | 覆盖率达标 + 冒烟测试 | Phase 5 | 0.5d |

**总预计：约 6.5 人天**（单人实现，包含测试编写）

## v0.2 vs v0.1 关键变更

| 变更点 | v0.1 问题 | v0.2 修正 |
|--------|-----------|----------|
| 默认 Aggressiveness | 0.6（偏激进） | **0.5**（更安全） |
| CacheAligner 默认 | `Enabled=true` | **false**（减少前缀开销） |
| Stopwords 列表 | "40+" 未定义 | **43 词枚举** |
| 数字截断 | 3 位有效数字（语义破坏风险） | **小数点后 2 位 + 转字符串** |
| CCR id | 无版本前缀 | **`v1_` 前缀** |
| Proxy 流式 | 未定义 | **明确拒绝 + 400** |
| `Result.Original` | 冗余字段 | **删除** |
| Token 估算 | `Model` 字段死代码 | **`TokenLimit` 替代** |
| ContentKind 枚举 | iota 推导（易破坏） | **显式赋值 0/1/2** |
