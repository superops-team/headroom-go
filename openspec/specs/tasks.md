# Headroom-Go 实现任务清单（v0.2.0 修订版）

> 按 Phase 1→6 顺序执行。TDD 流程：①写测试（编译失败）→②写实现（测试通过）→③重构。每个任务完成后打勾。

---

## Phase 1: 项目骨架

- [ ] Task 1.1 — 清理临时代码，保留 `go.mod`，确认目录结构：
  - 根包文件：`content_kind.go`、`router.go`、`smartcrusher.go`、`codecompressor.go`、`textcompressor.go`、`cachealigner.go`、`ccr.go`、`proxy.go`、`headroom.go`
  - `cmd/headroom/main.go`
  - 每个 `.go` 文件对应 `.go` 的 `_test.go`
- [ ] Task 1.2 — 实现 [content_kind.go](file:///workspace/content_kind.go)：`ContentKind` 枚举（明确赋值 0/1/2）+ `String()` 方法

## Phase 2: 核心压缩器（测试驱动）

### Task 2.1 — ContentRouter（测试先行）

- [ ] 2.1a — 写 `router_test.go`：覆盖 JSON 数组/对象、Python 代码、纯文本、空字符串、恰好含 2 个关键字行（含 3 个才判定为代码）、Markdown 中的 JSON 代码块（```json）
- [ ] 2.1b — 实现 [router.go](file:///workspace/router.go)：单次遍历 + 计数器，关键字用 `strings.Contains` 检测

### Task 2.2 — SmartCrusher（测试先行）

- [ ] 2.2a — 写 `smartcrusher_test.go`：
  - 合法 JSON + Aggressiveness 0.2 → 空值移除
  - 合法 JSON + Aggressiveness 0.5 → 数组折叠（>5 元素）
  - 合法 JSON + Aggressiveness 0.8 → 数字截断为字符串
  - 非法 JSON → 返回原文 + nil
  - `[1,2,3]` + Aggressiveness 0.5 → 输出是合法 JSON
- [ ] 2.2b — 实现 [smartcrusher.go](file:///workspace/smartcrusher.go)：
  - 用 `json.Valid()` 判断合法性
  - 用 `json.Unmarshal` + `json.Marshal` 重建（自动清理空白/逗号）
  - 自定义遍历删除零值和空集合
  - 激进模式数字用 `fmt.Sprintf("%.2f", v)` + 加引号

### Task 2.3 — CodeCompressor（测试先行）

- [ ] 2.3a — 写 `codecompressor_test.go`：
  - `// comment` 和 `/* block */` 移除验证
  - 恰好 20 行函数 → 不折叠；21 行 → 折叠
  - `if err != nil { return nil, err }` 保留
  - Python `#` 注释移除
  - 空行收缩验证
- [ ] 2.3b — 实现 [codecompressor.go](file:///workspace/codecompressor.go)：
  - 正则去除块注释 `/\*[\s\S]*?\*/`
  - 逐行处理：去除单行注释、空白行
  - 保留语义锚点行（`return`/`throw`/`func` 等）
  - 函数体 >20 行折叠

### Task 2.4 — TextCompressor（测试先行）

- [ ] 2.4a — 写 `textcompressor_test.go`：
  - FATAL/ERROR 行完整保留验证
  - 重复 200 行 → ` [x200]` 计数
  - 恰好 30 行 → 不折叠；31 行 → 折叠
  - stopwords 句子缩短 ≥ 30%
  - 中文文本 stopwords 不处理
- [ ] 2.4b — 实现 [textcompressor.go](file:///workspace/textcompressor.go)：
  - 行级去重 map（保留出现次数）
  - stopwords 用 `map[string]struct{}` 查表（O(1)）
  - FATAL/ERROR 直接输出（不做去重检查）
  - >30 行段落：前 10 + `[...N more lines...]` + 最后 5

## Phase 3: 辅助模块

- [ ] Task 3.1 — 实现 [cachealigner.go](file:///workspace/cachealigner.go) + `cachealigner_test.go`
  - Config 只含 `Enabled` + `Version`
  - `Enabled=true` → `[headroom/{Version}]` 前缀
  - `Enabled=false` → 原样返回

- [ ] Task 3.2 — 实现 [ccr.go](file:///workspace/ccr.go) + `ccr_test.go`
  - `v1_{sha1(original)[0:12]}` id 格式
  - `sync.RWMutex` 保护所有操作
  - GC：每次 `Store()` 内触发惰性清理
  - `Stats()` 不统计过期条目

## Phase 4: 顶层 API

- [ ] Task 4.1 — 重写 [headroom.go](file:///workspace/headroom.go)：
  - `Options`：`Aggressiveness=0.5`，`Reversible=true`，`AlignPrefix=false`，`TokenLimit=0`
  - `Result`：删除 `Original` 字段
  - `tool` 角色 → 当 user 处理
  - token 估算：`utf8.RuneCountInString(s) / 4`
  - 可逆模式：末尾追加 `[headroom:retrieve id=v1_xxx]`
  - `TokenLimit>0` 时：估算 < TokenLimit 则跳过压缩

- [ ] Task 4.2 — 写 [headroom_test.go](file:///workspace/headroom_test.go)：
  - 混合 JSON + Code + Text 消息
  - 可逆 ON / OFF 对比
  - TokenLimit 跳过验证
  - Savings 精度（2 位小数）验证

## Phase 5: Transport 层

- [ ] Task 5.1 — 实现 [proxy.go](file:///workspace/proxy.go) + `proxy_test.go`：
  - `stream: true` → HTTP 400 + `{"error":"streaming not supported in v0.1"}`
  - 上游不可达 → HTTP 502 + `{"error":"upstream unreachable"}`
  - `GET /healthz` → `{"status":"ok"}`
  - 用 `httptest.NewServer` mock 上游

- [ ] Task 5.2 — 实现 [cmd/headroom/main.go](file:///workspace/cmd/headroom/main.go)：
  - `cobra` 或标准库 `flag` 实现子命令（优先标准库 flag，零依赖）
  - `compress`：stdin → compress → stdout；`--stats` → stderr 统计
  - `proxy`：`http.ListenAndServe`，端口冲突 → 退出码 1
  - `version`：`fmt.Println("headroom-go v0.1.0")`

## Phase 6: 验证与收尾

- [ ] Task 6.1 — `go test ./... -cover`，目标 ≥ 60%，修复所有失败
- [ ] Task 6.2 — 冒烟测试：
  - `go build -o headroom ./cmd/headroom`
  - `echo "test" | ./headroom compress`
  - `./headroom proxy --port=8787 &` → `curl localhost:8787/healthz`
- [ ] Task 6.3 — 更新 spec.md status → `implemented`

---

**总任务数：24 个原子步骤（比 v0.1 细化 4 个 TDD 子步骤）**
