# Headroom-Go v0.3.0 任务拆解与测试规划

> 按 Phase 1→3 顺序执行。TDD 流程：先写测试（编译失败）→ 写实现（测试通过）→ 重构。
> 总改动量：~100 行新增 + ~20 行删除 = 净增 ~80 行。

---

## Phase 1: P0 生产阻塞项修复（预计 2h）

### Task 1.1 — 修正 go.mod 版本号

- [ ] **文件**：`go.mod`
- [ ] **改动**：`go 1.25.1` → `go 1.22`
- [ ] **验证**：`go build ./... && go vet ./... && go test -race ./...`

### Task 1.2 — Proxy HTTP Client 超时配置

- [ ] **文件**：`proxy/proxy.go`
- [ ] **改动**：
  - 在 `NewProxy` 中创建包级 `http.Client` 替代 `http.DefaultClient`
  - 配置：`Timeout: 60s`，`Transport.DialContext` 超时 10s，`TLSHandshakeTimeout: 10s`，`ResponseHeaderTimeout: 30s`
  - `forwardToUpstream` 使用该 client
- [ ] **测试**：`proxy/proxy_test.go` 新增 `TestProxy_UpstreamTimeout`
  - 用 `httptest.Server` 模拟慢响应（`time.Sleep(2s)`），验证 1s 超时后返回 502
- [ ] **验证**：`go test -race -v ./proxy/`

### Task 1.3 — HTTP Server 优雅关闭

- [ ] **文件**：`cmd/headroom/main.go`
- [ ] **改动**：
  - `runProxy` 中 `http.ListenAndServe` → `http.Server{Addr, Handler}`
  - 添加 `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)`
  - `srv.Shutdown(shutdownCtx)` with 30s timeout
  - 日志改用 `slog`（Phase 2 统一改）
- [ ] **测试**：手动验证（无法在单元测试中验证信号处理）
  - `go build -o headroom ./cmd/headroom && ./headroom proxy --port=19999 & sleep 1 && kill -TERM $!`
  - 预期：打印 "shutting down..." 后正常退出
- [ ] **验证**：`go build ./cmd/headroom`

### Task 1.4 — CCR 内存上限 + 后台 GC

- [ ] **文件**：`ccr.go`
- [ ] **改动**：
  - `CCRConfig` 新增 `MaxEntries int` 字段（默认 10000）
  - `NewCCR` 中启动后台 goroutine：`go c.backgroundGC()`
  - `backgroundGC()`：`time.NewTicker(30 * time.Minute)` 循环调用 `collectExpired()`
  - `Store()` 中：`len(c.data) >= c.cfg.MaxEntries` 时删除最旧条目（遍历找最小 `StoredAt`）
  - 保留现有惰性 GC（`collectExpired` 在 `Store` 时仍触发）
- [ ] **测试**：`ccr_test.go` 新增：
  - `TestCCR_MaxEntries`：Store 10001 条，验证 count ≤ 10000
  - `TestCCR_BackgroundGC`：用短 TTL（1ms），等待 Ticker 触发，验证过期条目被清理
- [ ] **验证**：`go test -race -v -run TestCCR`

---

## Phase 2: P1 代码卫生与安全（预计 1.5h）

### Task 2.1 — SHA1 → SHA256

- [ ] **文件**：`ccr.go`
- [ ] **改动**：
  - `import "crypto/sha1"` → `import "crypto/sha256"`
  - `sha1Prefix12` → `sha256Prefix12`，内部 `sha1.New()` → `sha256.New()`
  - `Store()` 中 id 前缀 `"v1_"` → `"v2_"`
- [ ] **测试**：更新 `ccr_test.go` 中所有 `"v1_"` 前缀断言 → `"v2_"`
  - `TestCCR_StoreAndRetrieve`：验证 id 以 `v2_` 开头
  - `TestCCR_RetrieveMissing`：用 `v2_` 前缀
- [ ] **验证**：`go test -race -v -run TestCCR`

### Task 2.2 — `itoa` → `strconv.Itoa`

- [ ] **文件**：`codecompressor.go`, `textcompressor.go`
- [ ] **改动**：
  - `codecompressor.go`：删除 `itoa` 函数（第 227-247 行），`import "strconv"`，所有 `itoa(n)` → `strconv.Itoa(n)`
  - `textcompressor.go`：`import "strconv"`，所有 `itoa(n)` → `strconv.Itoa(n)`
- [ ] **测试**：现有测试全部通过即可（行为完全一致）
- [ ] **验证**：`go test -race -v ./...`

### Task 2.3 — `fmt.Fprintf` → `log/slog`

- [ ] **文件**：`cmd/headroom/main.go`, `proxy/proxy.go`
- [ ] **改动**：
  - `cmd/headroom/main.go`：
    - 创建包级 `logger := slog.New(slog.NewTextHandler(os.Stderr, nil))`
    - 所有 `fmt.Fprintf(os.Stderr, ...)` → `logger.Info(...)` / `logger.Error(...)`
    - `fmt.Fprintln(os.Stderr, ...)` → `logger.Error(...)`
  - `proxy/proxy.go`：
    - `NewProxy` 接受 `*slog.Logger` 参数（或使用包级 logger）
    - 压缩失败时 `logger.Error("compression failed", "error", err)`
- [ ] **测试**：现有测试全部通过（日志输出到 stderr，不影响功能）
- [ ] **验证**：`go test -race -v ./... && go build ./cmd/headroom`

---

## Phase 3: P2 最小可观测性（预计 0.5h）

### Task 3.1 — Proxy Request ID

- [ ] **文件**：`proxy/proxy.go`
- [ ] **改动**：
  - `forwardToUpstream` 中：从请求头读取 `X-Request-ID`，若无则用 `crypto/rand` 生成 UUIDv4
  - 转发到上游时设置 `X-Request-ID` 头
  - 响应头中设置 `X-Request-ID`
  - 日志中附带 `request_id` 字段
- [ ] **测试**：`proxy_test.go` 新增 `TestProxy_RequestID`
  - 发送不带 `X-Request-ID` 的请求，验证响应头包含 `X-Request-ID`
  - 发送带 `X-Request-ID: my-id` 的请求，验证响应头为 `my-id`
- [ ] **验证**：`go test -race -v ./proxy/`

### Task 3.2 — `/healthz` 扩展

- [ ] **文件**：`proxy/proxy.go`
- [ ] **改动**：
  - `/healthz` handler 返回 `{"status":"ok","version":"v0.3.0","uptime":"..."}`
  - `NewProxy` 中记录 `startTime := time.Now()`
  - `uptime` 用 `time.Since(startTime).String()`
- [ ] **测试**：更新 `TestProxy_Healthz` 断言
  - 验证响应包含 `"version":"v0.3.0"` 和 `"uptime"`
- [ ] **验证**：`go test -race -v ./proxy/`

---

## Phase 4: 回归验证（预计 0.5h）

- [ ] **Task 4.1**：`go test -race -v ./...` 全部通过
- [ ] **Task 4.2**：`go vet ./...` 无警告
- [ ] **Task 4.3**：`go build -o headroom ./cmd/headroom` 成功
- [ ] **Task 4.4**：冒烟测试
  - `echo '{"a":1,"b":null}' | ./headroom compress --stats`
  - `./headroom proxy --port=18787 & sleep 1 && curl localhost:18787/healthz && kill %1`
- [ ] **Task 4.5**：更新 `openspec/config.yaml` 版本号为 `0.3.0`

---

## 测试规划总览

| 测试类型 | 新增 | 修改 | 说明 |
|---------|------|------|------|
| 单元测试 | 4 个 | 2 个 | MaxEntries / BackgroundGC / UpstreamTimeout / RequestID |
| 现有测试 | — | 2 个 | CCR v1_→v2_ 前缀 / Healthz 扩展 |
| 冒烟测试 | 2 个 | — | CLI compress + proxy 手动验证 |
| 覆盖率目标 | — | — | 保持 ≥ 85%（核心包） |

## 开发排期

| Phase | 内容 | 预计 |
|-------|------|------|
| Phase 1 | P0 修复（go.mod + 超时 + 优雅关闭 + CCR 内存） | 2h |
| Phase 2 | P1 卫生（SHA256 + itoa + slog） | 1.5h |
| Phase 3 | P2 可观测性（Request ID + healthz） | 0.5h |
| Phase 4 | 回归验证 + 冒烟测试 | 0.5h |
| **合计** | | **4.5h（约 0.6 人天）** |
