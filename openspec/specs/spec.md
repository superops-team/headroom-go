---
title: Headroom Go 规范说明书
version: 0.2.0
status: draft
domain: ai-context-compression
owner: headroom-go
created: 2026-06-14
updated: 2026-06-14
changelog:
  - v0.2.0: 修正版 — 删除冗余字段、明确模糊规则、补全协议边界
  - v0.1.0: 初稿
---

# Headroom Go — AI 上下文压缩层（Go 实现）

## 概述

Headroom-Go 是 [headroom](https://github.com/chopratejas/headroom)（Python 版）的 Go 移植，核心目标：在发送给 LLM 之前，将 Agent 读取的一切（工具输出、日志、RAG 片段、文件、对话历史）压缩为更小的 token 量，同时保持语义准确性与可逆性。

v0.1.0 为**最小可上线版本**，目标：单二进制、零外部依赖、Library API + HTTP Proxy + CLI。

### 与 Python 原版的组件对照

| 原版组件 | Go 版替代（v0.1） |
|---------|-------------------|
| Kompress-base (HuggingFace ML) | 启发式文本压缩（关键词 + 句子评分，见 TextCompressor） |
| CodeCompressor (Tree-sitter AST) | 基于关键字/正则的代码结构折叠与注释剔除 |
| Cross-agent Memory | 进程内内存 CCR（sync.RWMutex + map，v0.2 可扩展为 SQLite） |
| Agent wrap / MCP / Copilot | v0.2+ 规划 |

## 功能范围

### 包含（v0.1）

- Library API — `Compress()` / `CompressString()`
- 内容类型自动检测（JSON / 代码 / 文本）
- SmartCrusher — JSON 结构压缩
- CodeCompressor — 代码启发式压缩
- TextCompressor — 文本启发式压缩
- CacheAligner — 可选前缀标记（提升 Provider KV Cache）
- CCR — 可逆压缩：原始内容本地缓存（Library API 层），支持按 ID 检索
- HTTP Proxy — OpenAI 兼容 `/v1/chat/completions` 透明代理（非流式）
- CLI — `compress` / `proxy` / `version` 子命令
- `go test` 单元测试（覆盖率 ≥ 60%）

### 不包含（v0.1 明确拒绝）

- ML 模型推理、Tree-sitter 绑定
- 流式响应（`stream: true` 请求返回 `{"error": "streaming not supported in v0.1"}`）
- 跨 Agent 持久化 Memory、Agent wrap 集成
- MCP Server、Copilot OAuth 路由
- `headroom learn`、`headroom perf`
- Proxy 层的可逆检索自动循环（v0.1 Proxy 不做 retrieve 处理，Library API 层支持手动 Retrieve）

## 架构总览

```
  ┌────────────────────── 调用方 ───────────────────────┐
  │   Go App (Library)   │   CLI compress   │   HTTP 客户端   │
  └───────┬──────────────┴───────┬──────────┴───────┬──────────┘
          ▼                       ▼                 ▼
  ┌─────────────────┐    ┌──────────────┐   ┌──────────────┐
  │  Compress([]Msg) │    │ compress 子命令 │   │ HTTP Proxy   │
  │  CompressString  │    └──────────────┘   │ (非流式)    │
  └────────┬─────────┘                        └──────┬───────┘
           ▼                                        ▼
  ┌─────────────────────────────────────────────────────────┐
  │                    ContentRouter.Detect                  │
  │         KindJSON (1) / KindCode (2) / KindText (0)     │
  └──────┬────────────┬────────────────┬────────────────────┘
         ▼            ▼                ▼
  ┌────────────┐ ┌──────────┐   ┌──────────────┐
  │ SmartCrusher│ │ CodeComp.│   │ TextCompressor│
  └──────┬─────┘ └────┬────┘   └──────┬───────┘
         ▼            ▼                ▼
  ┌──────────────────────────────────────────────┐
  │          CacheAligner.Align (可选)            │
  └──────────────────────────────────────────────┘
           ▼
  ┌──────────────────────────────────────────────┐
  │  CCR.Store (可逆) → SHA-1 id ("v1_"+12hex)   │
  │  CCR.Retrieve(id) → original (Library API层)  │
  └──────────────────────────────────────────────┘
```

**数据流：**

1. 输入消息进入 `ContentRouter`，按内容签名分类
2. 分类后进入对应 Compressor，生成压缩内容
3. 压缩内容经过 `CacheAligner`（可选）
4. 若 `Options.Reversible = true`：原始内容存入 CCR，生成 retrieve ID；压缩内容末尾追加 `[headroom:retrieve id=v1_xxx]`
5. 返回压缩消息

---

## 需求项

### Requirement: 顶层压缩 API

#### 描述

库 SHALL 暴露 `Compress(messages []Message, opts Options) (*Result, error)` 和 `CompressString(content string, opts Options) (string, error)`。

#### 验收标准

- [x] `Message{Role, Content, Name}` — JSON 兼容 OpenAI Messages 格式
- [x] `Options.Aggressiveness float64` ∈ [0.0, 1.0]，默认 **0.5**（注：与 v0.1 的 0.6 不同——0.5 是更安全的默认）
- [x] `Options.Reversible bool` — 默认 **true**；控制是否调用 CCR.Store
- [x] `Options.AlignPrefix bool` — 默认 **false**（CacheAligner 默认关闭，减少前缀开销）
- [x] `Options.TokenLimit int` — 当估算 token 数 < TokenLimit 时跳过压缩；默认 0（不限制，始终压缩）
- [x] `Result.Messages` — 压缩后的消息数组（**唯一输出**）
- [x] `Result.CompressedTokens` / `Result.OriginalTokens` — 基于 `runeCount / 4` 估算
- [x] `Result.Savings = (OriginalTokens - CompressedTokens) / OriginalTokens`，保留 2 位小数
- [x] `assistant` 角色消息原样透传，**不压缩**
- [x] `tool` 角色消息**视为 user 消息处理**（压缩）

#### 优先级

- [x] 高

---

### Requirement: ContentRouter — 内容类型检测

#### 描述

ContentRouter SHALL 实现 O(n) 检测，内存分配不超过 O(1) 额外空间（除输入字符串本身外）。

#### 验收标准

- [x] `Detect(content string) ContentKind`
- [x] `KindJSON(1)` / `KindCode(2)` / `KindText(0)` 枚举明确赋值为 0/1/2（不使用 iota 推导，避免后续插入破坏常量值）
- [x] JSON 判定：去掉首尾空白后以 `{` 或 `[` 开头，且 `encoding/json.Valid()` 返回 true
- [x] 代码判定：全文扫描，**任意 3 行以上**包含代码关键字（见下方列表），或包含 ``` 分隔符，或出现 `//` / `#` 单行注释与 `{}`/`;` 共存
- [x] 代码关键字列表：**`func` `return` `class` `def` `import` `export` `struct` `interface` `enum` `fn` `pub` `let` `const` `var` `async` `await` `throw` `try` `catch`**
- [x] 默认 → `KindText`
- [x] 空字符串 → `KindText`

#### 代码关键字检测实现约束

```go
// 正确：单次遍历，计数器
found := 0
for each line:
    if line contains any keyword: found++
    if found >= 3: return KindCode
// 错误：不缓存多行、不预扫描全文
```

#### Scenario

- **GIVEN** `"[1,2,3]"`（合法 JSON 数组）
- **WHEN** `router.Detect("[1,2,3]")`
- **THEN** 返回值 == `KindJSON`（值 1）

- **GIVEN** `"def foo():\n    pass\n    return 1"`
- **WHEN** `router.Detect(content)`
- **THEN** 返回值 == `KindCode`（值 2）

- **GIVEN** `"INFO 2026-06-14 service started on port 8080"`
- **WHEN** `router.Detect(content)`
- **THEN** 返回值 == `KindText`（值 0）

#### 优先级

- [x] 高

---

### Requirement: SmartCrusher — JSON 结构压缩

#### 描述

SmartCrusher SHALL 接收 JSON 字符串，输出**仍为合法 JSON**（`encoding/json.Valid()` 通过），语义保留。压缩策略按 `Aggressiveness` 分三级。

#### 压缩策略（明确规则）

| Aggressiveness | 策略 | 示例 |
|----------------|------|------|
| **0.0–0.3** 保守 | 移除空白、多余逗号、零值（`null`/`0`/`false`/空字符串`""`）、空对象`{}`、空数组`[]` | `{"a":null,"b":0}` → `{"a":null}` |
| **0.3–0.7** 标准 | 保守 + 数组折叠（>5 元素 → `[...N items]`）、保留字段名不变 | `[1,2,3,4,5,6,7]` → `"items":[...7 items]` |
| **0.7–1.0** 激进 | 标准 + 数字保留**小数点后最多 2 位**（`3.14159` → `"3.14"`）、布尔转 `T`/`F` | `{"ratio":3.14159}` → `{"ratio":"3.14"}` |

**注意**：激进模式将数字转为字符串（加引号）以保证 JSON 合法，避免截断 `1e-10` → `1e-1` 的语义破坏问题。

#### 验收标准

- [x] 非法 JSON 输入 → 返回原始内容 + `nil` error（降级）
- [x] 合法 JSON 输入 → 输出必须通过 `encoding/json.Valid()`
- [x] 数组折叠阈值：**>5 个元素**（含）折叠为 `[...N items]`
- [x] 零值移除：保守模式移除 `null`、`0`、`false`、空字符串 `""`、空对象 `{}`、空数组 `[]`
- [x] 数字激进模式：小数点后最多 2 位，**转为字符串**（`"3.14"` 而非裸数字 `3.14`）

#### Scenario

- **GIVEN** `{"items":[1,2,3,4,5,6,7]}`，`Aggressiveness = 0.5`
- **WHEN** `SmartCrushJSON(content, cfg)`
- **THEN** 输出为合法 JSON，且 `items` 值为 `"[...7 items]"`

- **GIVEN** `{"x":null,"y":[],"z":1}`，`Aggressiveness = 0.2`
- **WHEN** `SmartCrushJSON(content, cfg)`
- **THEN** 输出为 `{"z":1}`（x 和 y 被移除）

- **GIVEN** `{"v":3.1415926}`，`Aggressiveness = 0.8`
- **WHEN** `SmartCrushJSON(content, cfg)`
- **THEN** 输出为 `{"v":"3.14"}`（字符串形式，合法 JSON）

#### 优先级

- [x] 高

---

### Requirement: CodeCompressor — 代码启发式压缩

#### 描述

CodeCompressor SHALL 去除注释/空行/冗余空白，折叠过长函数体。

#### 压缩规则

| 操作 | 规则 |
|------|------|
| 单行注释 | 移除所有 `// ` 和 `# ` 开头的注释行 |
| 块注释 | 移除所有 `/* ... */` 块（跨行） |
| 空行 | 移除所有仅含空白的行 |
| 连续空白 | 替换为单个空格（行内） |
| 长函数折叠 | 函数体（`{` 到 `}` 之间的行数）**>20 行** → 保留签名行 + `// ... (N lines collapsed) ...` + 最后 **3 行** |
| 语义锚点保留 | 以下关键字所在行**不折叠**：`return` `throw` `func` `def` `class` `interface` `struct` `enum` `import` `export` |

#### 折叠操作定义

```
func Foo() {
  // line 1 ← 保留（函数签名）
  // line 2
  // ...
  // line 18
  return x        // ← 保留（语义锚点 return）
  // line 20
}                  ← 保留（闭合括号）
```

折叠后：
```
func Foo() {
// ... (18 lines collapsed) ...
return x
}
```

#### 验收标准

- [x] `CompressCode(content string, cfg CodeConfig) string`
- [x] `//` / `#` / `/* */` 注释必须完全移除（不残留注释符号）
- [x] 恰好 20 行的函数体**不折叠**（">20"才折叠）
- [x] 含 `err != nil` 的 if 语句属于语义锚点，**不折叠**
- [x] 折叠后的输出不改变行的相对顺序

#### Scenario

- **GIVEN** Go 代码含 `// this is a comment` 和 `/* block */`
- **WHEN** `CompressCode(content, CodeConfig{Aggressiveness: 0.5})`
- **THEN** 输出中不包含 `// this is a comment` 和 `/* block */`

- **GIVEN** 50 行函数体
- **WHEN** `CompressCode(content, CodeConfig{})`
- **THEN** 输出中函数体被折叠为 `// ... (N lines collapsed) ...` + 最后 3 行

- **GIVEN** `if err != nil { return nil, err }`
- **WHEN** `CompressCode(content, CodeConfig{})`
- **THEN** 该语句完整出现在输出中

#### 优先级

- [x] 高

---

### Requirement: TextCompressor — 文本启发式压缩

#### 描述

TextCompressor SHALL 对自然语言文本（日志、说明、RAG 片段）进行语义保留压缩。

#### 压缩规则（明确）

| 操作 | 规则 |
|------|------|
| 完全重复行 | 保留首行 + ` [xN]` 计数标记（精确计数） |
| 英文 stopwords（见下方列表，共 43 词） | 直接删除（不在原位留占位符） |
| FATAL / ERROR 行 | 完整保留，不去重，不折叠 |
| WARN 行 | 保留，不去重（仅对重复的 WARN 行计数） |
| INFO / DEBUG 行 | 完全重复时折叠为 ` [xN]` |
| 超长段落 | **>30 行**（非 spec.md 中错误的 50 行） → 保留前 **10 行** + `[...N more lines...]` + 最后 **5 行** |

**英文 Stopwords 列表（43 词）**：
`a, an, the, and, or, but, is, are, was, were, be, been, being, have, has, had, do, does, did, will, would, should, could, may, might, must, can, of, to, in, for, on, at, by, from, with, as, about, into, over, after, before, between, during, under, since, without, within, than, then, so`

#### 验收标准

- [x] `CompressText(content string, cfg TextConfig) string`
- [x] FATAL / ERROR 完整保留
- [x] `[INFO] heartbeat OK` 重复 200 次 → `[INFO] heartbeat OK [x200]`
- [x] stopwords 收缩后，典型英文句子缩短 ≥ 30%
- [x] 行顺序不变（不重排序）
- [x] **不处理中文/非拉丁文字**：字符不在 ASCII 范围内则跳过 stopwords 判断

#### Scenario

- **GIVEN** 200 行 `[INFO] heartbeat OK` + 1 行 `[FATAL] disk full`
- **WHEN** `CompressText(content, TextConfig{Aggressiveness: 0.5})`
- **THEN** `[FATAL] disk full` 完整保留；`[INFO]` 行折叠为 `[INFO] heartbeat OK [x200]`

- **GIVEN** `"the server is running in production mode for the client"`
- **WHEN** `CompressText(content, TextConfig{})`
- **THEN** 输出 `server running production mode client`（删除了 6 个 stopwords），长度缩短 >30%

#### 优先级

- [x] 高

---

### Requirement: CacheAligner — 前缀对齐

#### 描述

在压缩输出头部插入固定前缀，使相同配置的压缩请求产生相同前缀，提升 Provider KV Cache 命中率。

#### 验收标准

- [x] `CacheAlignerConfig{Enabled bool, Version string}` — **删除了 Kind 和 Aggressiveness 字段**（前缀不需要这些）
- [x] `Align(content string) string`
- [x] `Enabled = true` 时，输出格式固定为 `[headroom/{Version}]`（不超过 30 字符）
- [x] `Enabled = false` 时，输出 == 输入（零开销）
- [x] 默认 `Version = "v0.1"`，完整默认前缀为 `[headroom/v0.1]`

#### Scenario

- **GIVEN** `CacheAlignerConfig{Enabled: true, Version: "v0.1"}`，输入 `"some text"`
- **WHEN** `aligner.Align("some text")`
- **THEN** 输出 == `"[headroom/v0.1]\nsome text"`

#### 优先级

- [ ] 中

---

### Requirement: CCR — 可逆压缩缓存与检索

#### 描述

CCR（Compress-Cache-Retrieve）存储原始内容，生成版本化 ID，支持按 ID 检索。

#### 关键设计决策

1. **ID 格式**：`v1_{sha1(original)[0:12]}` — `v1_` 前缀保证未来算法切换时旧 id 仍可识别
2. **GC 机制**：惰性检查——每次 `Store()` 调用时，先扫描全量 map，删除所有 `CreatedAt` 早于 `now - TTL` 的条目
3. **线程安全**：`sync.RWMutex` 保护所有 map 操作

#### 验收标准

- [x] `NewCCR(config CCRConfig) *CCR`，`TTL` 默认 24h
- [x] `Store(original, compressed string, kind ContentKind) string` → 返回 `"v1_{12hex}"`
- [x] `Retrieve(id string) (string, bool)` → 找到返回 original + true；未找到返回 `""` + false
- [x] `Stats() (int, int)` → `(activeEntryCount, totalOriginalBytes)`，**不包含已过期条目**
- [x] GC：每次 `Store()` 时触发，检查并删除所有超时条目
- [x] 线程安全：读写操作全部加锁
- [x] Library API 层：压缩内容末尾追加 `[headroom:retrieve id=v1_xxx]`
- [x] **Proxy 层（v0.1）：不处理 retrieve ID**，Proxy 仅做压缩转发，不注入 Retrieve 逻辑

#### Scenario

- **GIVEN** `"the quick brown fox"`
- **WHEN** `id := ccr.Store(original, compressed, KindText)` → `ccr.Retrieve(id)`
- **THEN** 返回 `"the quick brown fox"` + `true`

- **GIVEN** 空 CCR 实例
- **WHEN** `ccr.Retrieve("v1_deadbeef1234")`
- **THEN** 返回 `""` + `false`

#### 优先级

- [x] 高

---

### Requirement: HTTP Proxy — OpenAI 兼容透明代理（非流式）

#### 描述

HTTP Proxy SHALL 拦截 `POST /v1/chat/completions` 请求体中的 messages，压缩后转发。

**重要约束（v0.1 明确边界）**：

- **仅支持非流式请求**（`"stream": false` 或无 stream 字段）
- **不支持流式**：`"stream": true` → 返回 HTTP 400 + `{"error": "streaming not supported in v0.1"}`
- **不处理 retrieve ID**：Proxy 不实现 Retrieve 逻辑，LLM 返回的 `[headroom:retrieve id=...]` 不会被自动解析

#### 验收标准

- [x] `NewProxy(config ProxyConfig) http.Handler`
- [x] 配置项：`UpstreamBaseURL`（默认 `https://api.openai.com/v1`）、`APIKey`（从环境变量 `HEADROOM_API_KEY` 读取）、`ListenAddr`（默认 `:8787`）、`CompressOptions`
- [x] `POST /v1/chat/completions`：解析 JSON → 压缩 messages → 转发上游 → 透传响应
- [x] `GET /healthz` → `{"status":"ok"}`
- [x] 上游不可达 → HTTP 502 + `{"error":"upstream unreachable"}`
- [x] 流式请求 → HTTP 400 + `{"error":"streaming not supported in v0.1"}`
- [x] Authorization header 原样转发

#### Scenario

- **GIVEN** `httptest.Server` 作为上游，`stream: false` 请求
- **WHEN** 发送包含 10 条长消息的请求
- **THEN** 转发请求中 content 长度明显小于原始

- **GIVEN** `{"stream": true, ...}` 请求
- **WHEN** 发送请求
- **THEN** 返回 400 `{"error":"streaming not supported in v0.1"}`

#### 优先级

- [ ] 中

---

### Requirement: CLI — 命令行工具

#### 描述

CLI SHALL 提供 3 个子命令：`compress`（标准流压缩）、`proxy`（启动代理）、`version`（版本信息）。

#### 验收标准

- [x] `headroom compress [--aggressiveness=0.5] [--no-reversible] [--no-align] [--input=<file>] [--output=<file>] [--stats]`
  - 无 `--input` → 从 stdin 读取
  - `--stats` → 打印 `Original: X tokens | Compressed: Y tokens | Savings: Z%` 到 stderr
- [x] `headroom proxy [--port=8787] [--upstream=<url>]`
  - 启动时打印 `Listening on :8787`
  - 端口被占用时打印 `address already in use` 并退出码 1
- [x] `headroom version` → `headroom-go v0.1.0`

#### Scenario

- **GIVEN** `echo "hello world the quick brown fox jumps over the lazy dog" | headroom compress --stats`
- **WHEN** 执行命令
- **THEN** stdout 输出压缩结果；stderr 输出 token 统计

#### 优先级

- [x] 高

---

### Requirement: go test 单元测试覆盖

#### 描述

每个模块独立可测试，零网络访问（Proxy 测试用 `httptest`）。

#### 验收标准

- [x] `go test ./...` 全通过（Go 1.22+）
- [x] 每个压缩器测试覆盖：空输入、典型输入、错误/降级路径
- [x] TextCompressor 额外测试：中文文本（stopwords 不处理）、恰好 30 行（不折叠）、31 行（折叠）
- [x] CodeCompressor 额外测试：恰好 20 行（不折叠）、21 行（折叠）
- [x] Proxy 测试：正常请求、流式拒绝、上游错误
- [x] **覆盖率 ≥ 60%**

#### 优先级

- [x] 高

---

## 接口定义（最终版）

```go
// content_kind.go
type ContentKind int
const (
    KindText ContentKind = 0
    KindJSON ContentKind = 1
    KindCode ContentKind = 2
)
func (k ContentKind) String() string

// headroom.go（根包，合并 proxy，无需子包）
type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
    Name    string `json:"name,omitempty"`
}

type Options struct {
    Aggressiveness float64 // 默认 0.5
    Reversible     bool    // 默认 true
    AlignPrefix    bool    // 默认 false
    TokenLimit     int     // 默认 0（不限制）
}

type Result struct {
    Messages        []Message
    CompressedTokens int
    OriginalTokens  int
    Savings        float64
}

func DefaultOptions() Options
func Compress(messages []Message, opts Options) (*Result, error)
func CompressString(content string, opts Options) (string, error)

// router.go
type ContentRouter struct{}
func NewContentRouter() *ContentRouter
func (r *ContentRouter) Detect(content string) ContentKind

// smartcrusher.go
type SmartCrushConfig struct{ Aggressiveness float64 }
func SmartCrushJSON(content string, cfg SmartCrushConfig) (string, error)

// codecompressor.go
type CodeConfig struct{ Aggressiveness float64 }
func CompressCode(content string, cfg CodeConfig) string

// textcompressor.go
type TextConfig struct{ Aggressiveness float64 }
func CompressText(content string, cfg TextConfig) string

// cachealigner.go
type CacheAlignerConfig struct {
    Enabled bool
    Version string // 默认 "v0.1"
}
type CacheAligner struct{ cfg CacheAlignerConfig }
func NewCacheAligner(cfg CacheAlignerConfig) *CacheAligner
func (a *CacheAligner) Align(content string) string

// ccr.go
type CCRConfig struct{ TTL time.Duration } // 默认 24h
type CCREntry struct {
    Original  string
    Kind      ContentKind
    CreatedAt time.Time
}
type CCR struct {
    mu   sync.RWMutex
    data map[string]CCREntry
    cfg  CCRConfig
}
func NewCCR(cfg CCRConfig) *CCR
func (c *CCR) Store(original, compressed string, kind ContentKind) string // → "v1_12hex"
func (c *CCR) Retrieve(id string) (string, bool)
func (c *CCR) Stats() (int, int)

// proxy.go（同包，无需子目录）
type ProxyConfig struct {
    UpstreamBaseURL  string
    APIKey           string
    ListenAddr       string
    CompressOptions  Options
}
func NewProxy(cfg ProxyConfig) http.Handler

// cmd/headroom/main.go
package main // 独立可执行入口
```

### CLI 接口

```bash
headroom compress [flags]
  --aggressiveness float  压缩强度 (default 0.5)
  --no-reversible         关闭可逆压缩
  --no-align              关闭前缀对齐
  --input string          输入文件 (默认 stdin)
  --output string         输出文件 (默认 stdout)
  --stats                 打印 token 统计

headroom proxy [flags]
  --port int              监听端口 (default 8787)
  --upstream string       上游 Base URL (default https://api.openai.com/v1)

headroom version           打印版本号
```

### 错误处理

| 场景 | 处理 |
|------|------|
| 非法 JSON（SmartCrusher） | 返回原文 + nil |
| 空输入 | 返回空字符串 + nil |
| 流式请求（Proxy） | 400 `{"error":"streaming not supported in v0.1"}` |
| 上游不可达（Proxy） | 502 `{"error":"upstream unreachable"}` |
| CCR id 不存在 | `("", false)` |
| 端口被占用（CLI proxy） | 打印错误 + 退出码 1 |

---

## 数据模型

| 结构体 | 字段 | 类型 | 约束 |
|--------|------|------|------|
| Message | role | string | not null |
| Message | content | string | not null |
| Message | name | string | optional |
| CCREntry | Original | string | not null |
| CCREntry | Kind | ContentKind | not null |
| CCREntry | CreatedAt | time.Time | not null |
| ContentKind | — | int | 0=Text, 1=JSON, 2=Code |

---

## 验收标准总览

- [ ] 所有 Requirement 的验收标准均已实现并测试通过
- [ ] `go test ./... -cover` 覆盖率 ≥ 60%
- [ ] `headroom compress` 可用，支持 `--stats`
- [ ] `headroom proxy --port=8787` + `curl localhost:8787/healthz` 返回 200
- [ ] `stream: true` 请求返回 400
- [ ] `go.mod` 仅包含标准库（`std`）
- [ ] spec status → `implemented`
