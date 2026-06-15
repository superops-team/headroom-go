# Headroom-Go 生产加固变更提案（v0.3.0）

## Purpose

基于 v0.2.0 代码审查结果，对 headroom-go 进行**最小化生产环境加固**。v0.2.0 核心压缩逻辑设计良好、测试覆盖达标，但存在 4 个生产部署硬阻塞项和若干代码卫生问题。本提案聚焦 **P0 阻塞项 + P1 代码卫生**，P2 可观测性做最小可用实现，P3 全部推迟到 v0.4.0。

## Why Now?

v0.2.0 在以下场景存在明确风险，每一项均有代码证据：

| # | 风险 | 代码位置 | 后果 |
|---|------|---------|------|
| 1 | HTTP Client 无超时 | `proxy/proxy.go:149` `http.DefaultClient.Do(req)` | 上游不可达时 goroutine 永久阻塞 →  goroutine 泄漏 → OOM |
| 2 | CCR 无内存上限 | `ccr.go:36` `collectExpired()` 仅在 `Store()` 时触发 | 只读场景下 map 无限增长 → OOM |
| 3 | 无优雅关闭 | `cmd/headroom/main.go:137` `http.ListenAndServe` | SIGTERM 时正在处理的请求被中断 |
| 4 | go.mod 版本号错误 | `go.mod:1` `go 1.25.1` | 不存在的 Go 版本，构建可能失败 |
| 5 | SHA1 弱哈希 | `ccr.go:96` `sha1.New()` | 安全审计工具告警，合规风险 |
| 6 | 无结构化日志 | 全局 `fmt.Fprintf` | 生产排障无法按级别/字段过滤 |
| 7 | 自定义 `itoa` | `codecompressor.go:227-247` | 自造轮子，标准库已有 `strconv.Itoa` |

## Scope（v0.3.0）

### P0 — 生产部署硬阻塞（必须）

| 变更项 | 改动量 | 涉及文件 |
|--------|--------|---------|
| Proxy HTTP Client 超时配置 | ~15 行 | `proxy/proxy.go` |
| HTTP Server 优雅关闭 | ~20 行 | `cmd/headroom/main.go` |
| CCR 内存上限 + 后台 GC | ~30 行 | `ccr.go` |
| 修正 go.mod 版本号 | 1 行 | `go.mod` |

### P1 — 代码卫生与安全（强烈建议）

| 变更项 | 改动量 | 涉及文件 |
|--------|--------|---------|
| SHA1 → SHA256 | ~5 行 | `ccr.go` |
| `fmt.Fprintf` → `log/slog` | ~15 行 | `cmd/headroom/main.go`, `proxy/proxy.go` |
| `itoa` → `strconv.Itoa` | ~5 行（删除 20 行） | `codecompressor.go`, `textcompressor.go` |

### P2 — 最小可观测性（仅 proxy 层）

| 变更项 | 改动量 | 涉及文件 |
|--------|--------|---------|
| Proxy Request ID 生成与传递 | ~10 行 | `proxy/proxy.go` |
| Proxy `/healthz` 扩展为简单状态页 | ~5 行 | `proxy/proxy.go` |

### 明确不包含（推迟到 v0.4.0+）

以下项目**不在 v0.3.0 范围内**，理由如下：

| 推迟项 | 理由 |
|--------|------|
| Prometheus metrics | 需要引入 `expvar` 或手写格式，增加 ~100 行非核心代码；当前阶段用日志 + Request ID 即可排障 |
| Rate limiting | 上游 API 本身有 rate limit；proxy 层加限流属于过早优化；需引入 `golang.org/x/time/rate` |
| `context.Context` API | 新增 `CompressContext` 需全套测试覆盖，属于 API 扩展而非加固 |
| SmartCrusher Schema 兼容选项 | 当前激进模式行为是设计决策（见 spec.md），非 bug；变更需独立提案 |
| 拆分 `collapseLongFunctions` | 纯重构，无功能变更，推迟到有足够测试护栏后 |
| Benchmark / Fuzz / cmd 测试 | 非阻塞项，推迟到 CI 流水线建立后 |
| YAML 配置文件 | 3 个子命令的 CLI 不需要配置文件 |
| Token 估算改进 | 需引入 tiktoken 或自研算法，属于功能增强 |
| `CompressStream` | 全新功能，非加固 |
| 压缩效果语义验证 | 研究级课题，非工程任务 |

## 关键设计决策

| 决策 | 理由 |
|------|------|
| **P0/P1 全部完成才标记 production-ready** | 硬阻塞项不可分割 |
| **P2 仅做 Request ID + 扩展 healthz** | 最小可观测性：能追踪请求链路 + 确认服务存活 |
| **Rate limiting 推迟** | 上游 API 自带 rate limit；proxy 层加限流需引入 x/time/rate，违背零依赖原则 |
| **slog 而非 zap/zerolog** | 零外部依赖；slog 是 Go 1.21+ 标准库；`go.mod` 修正为 `go 1.22` 后完全可用 |
| **SHA256 替代 SHA1，id 前缀 `v1_` → `v2_`** | 哈希算法变更必须伴随前缀变更，否则新旧 id 碰撞无法区分 |
| **CCR 后台 GC 用 `time.Ticker`，30 分钟间隔** | 平衡 CPU 开销与内存释放；30 分钟对 24h TTL 足够 |
| **CCR MaxEntries 默认 10000，FIFO 淘汰** | 简单可预测；不需要 LRU 的复杂度 |
| **Proxy HTTP Client 超时：Dial 10s + TLS 10s + ResponseHeader 30s + 整体 60s** | 覆盖 LLM API 典型延迟（30-60s），同时防止无限等待 |
| **优雅关闭超时 30s** | 给正在处理的 LLM 请求足够的完成时间 |

## Risk & Mitigation

| 风险 | 等级 | 缓解 |
|------|------|------|
| CCR id 前缀 `v1_` → `v2_` 导致旧缓存不可检索 | **中** | v0.3.0 是首个 production 版本，无存量生产数据；开发环境缓存可丢弃 |
| slog 默认 JSON 格式改变日志输出 | 低 | v0.2.0 无生产部署，无日志解析依赖；slog 支持 `TextHandler` 保持可读性 |
| CCR MaxEntries 淘汰导致 Retrieve 返回 false | 低 | 调用方已有 `bool` 返回值处理；10000 条目 × 平均 10KB = 100MB，足够 |
| 后台 GC goroutine 未随 CCR 生命周期停止 | 低 | CCR 是包级单例（`sync.Once`），进程生命周期内一直运行，无需停止 |
| 优雅关闭超时内未完成 → 请求被强制中断 | 低 | 30s 覆盖绝大多数 LLM API 响应时间 |

## 向下兼容分析

| 变更 | 兼容性 | 说明 |
|------|--------|------|
| CCR id 前缀 `v1_` → `v2_` | **不兼容** | 旧 id 无法检索；无存量生产数据，影响为零 |
| `sha1Prefix12` → `sha256Prefix12` | 内部函数 | 无公开 API 变更 |
| `itoa` → `strconv.Itoa` | 内部函数 | 行为完全一致 |
| `fmt.Fprintf` → `slog` | 输出格式变化 | CLI stderr 输出从纯文本变为 `time=... level=... msg=...` |
| Proxy HTTP Client 超时 | 行为变化 | 超时后返回 502 而非永久挂起，是修复 |
| `http.ListenAndServe` → `http.Server` | CLI 行为变化 | 收到 SIGTERM/SIGINT 后等待 30s 再退出 |
| CCR `MaxEntries` 新增 | 新增字段 | `CCRConfig` 增加字段，零值 = 不限制（向后兼容） |
| Proxy Request ID | 新增 header | 响应头增加 `X-Request-ID`，不影响现有客户端 |

## v0.3.0 vs v0.2.0 变更对照

| 变更点 | v0.2.0 | v0.3.0 |
|--------|--------|--------|
| Proxy HTTP Client | `http.DefaultClient`（零超时） | 自定义 `http.Client`：Dial 10s / TLS 10s / ResponseHeader 30s / 整体 60s |
| HTTP Server | `http.ListenAndServe` | `http.Server` + `signal.NotifyContext` + 30s `Shutdown` |
| CCR 内存管理 | 仅 `Store()` 时惰性 GC | MaxEntries=10000 + 30min Ticker 后台 GC |
| go.mod | `go 1.25.1` | `go 1.22` |
| 哈希算法 | SHA1，id 前缀 `v1_` | SHA256，id 前缀 `v2_` |
| 日志 | `fmt.Fprintf(os.Stderr, ...)` | `slog.Info/Warn/Error` + `slog.NewTextHandler` |
| 数字格式化 | 自定义 `itoa()` | `strconv.Itoa()` |
| Request ID | 无 | 生成 UUIDv4 → `X-Request-ID` 请求头 + 响应头 |
| `/healthz` | `{"status":"ok"}` | `{"status":"ok","version":"v0.3.0","uptime":"2h30m"}` |
