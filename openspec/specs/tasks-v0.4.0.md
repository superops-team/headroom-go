# headroom-go v0.4.0 重构任务清单

> 每个任务为原子任务，可独立开发、测试、验收。

## Phase 1：基础设施重构（不破坏现有代码）

---

### [T1-001] 提取版本常量为统一管理

**文件：** `version.go`（新建）

**内容：**
```go
package headroom

const (
    // Version 是 headroom-go 的语义化版本。
    Version = "v0.4.0"

    // PrefixVersion 是缓存对齐前缀的版本。
    // 每次压缩算法升级导致输出变化时应递增。
    PrefixVersion = "v0.4"

    // CCRIDVersion 是可逆压缩 ID 的版本前缀。
    // 每次存储格式变化时应递增。
    CCRIDVersion = "v3"
)
```

**验收标准：**
- [ ] `version.go` 文件创建，定义 `Version`、`PrefixVersion`、`CCRIDVersion`
- [ ] `go build ./...` 通过

---

### [T1-002] 替换 headroom.go 中的版本号

**文件：** `headroom.go`

**变更：**
- `headroom.go:71`：`Version: "v0.3"` → `Version: PrefixVersion`

**验收标准：**
- [ ] 编译通过
- [ ] `go test ./...` 通过

---

### [T1-003] 替换 ccr.go 中的 ID 版本前缀

**文件：** `ccr.go`

**变更：**
- `ccr.go:60`：`id := "v2_" + sha256Prefix12(original)` → `id := CCRIDVersion + "_" + sha256Prefix12(original)`

**验收标准：**
- [ ] `TestCCR_StoreAndRetrieve` 测试通过
- [ ] `TestCCR_DuplicateStoreReturnsSameID` 测试通过

---

### [T1-004] 替换 CLI 版本打印

**文件：** `cmd/headroom/main.go`

**变更：**
```go
import headroom "github.com/superops-team/headroom-go"
// ...
fmt.Println("headroom-go " + headroom.Version)
// printUsage 中的 "v0.3.0" → headroom.Version
```

**验收标准：**
- [ ] `headroom version` 输出 `headroom-go v0.4.0`
- [ ] `headroom --help` 显示正确版本

---

### [T1-005] 替换 proxy.go /healthz 版本

**文件：** `proxy/proxy.go`

**变更：**
```go
import "github.com/superops-team/headroom-go"
// ...
fmt.Fprintf(w, `{"status":"ok","version":"%s","uptime":"%s"}`, headroom.Version, time.Since(startTime).String())
```

**验收标准：**
- [ ] `curl localhost:8787/healthz` 返回包含 `"version":"v0.4.0"`
- [ ] `TestProxy_Healthz` 通过

---

### [T1-006] 简化 CacheAligner 为 AlignPrefix 纯函数

**文件：** `prefix.go`（新建）或合并到 `headroom.go`

**内容：**
```go
// AlignPrefix 为 content 添加版本前缀，提升 Provider KV 缓存命中率。
// 若 enabled 为 false 或 version 为空，直接返回原内容。
func AlignPrefix(content string, enabled bool, version string) string {
    if !enabled || version == "" {
        return content
    }
    return fmt.Sprintf("[headroom/%s]\n%s", version, content)
}
```

**验收标准：**
- [ ] `AlignPrefix("hello", true, "v0.4")` 返回 `"[headroom/v0.4]\nhello"`
- [ ] `AlignPrefix("hello", false, "v0.4")` 返回 `"hello"`
- [ ] `AlignPrefix("hello", true, "")` 返回 `"hello"`

---

### [T1-007] 更新 headroom.go 使用 AlignPrefix 函数

**文件：** `headroom.go`

**变更：**
- 删除 `import "github.com/superops-team/headroom-go/cachealigner"`（不存在了）
- 删除 `aligner := NewCacheAligner(...)` 调用
- 将 `if opts.AlignPrefix { out = aligner.Align(out) }` 替换为 `if opts.AlignPrefix { out = AlignPrefix(out, true, PrefixVersion) }`

**验收标准：**
- [ ] `headroom.go` 中不再引用 `CacheAligner` 类型
- [ ] 删除 `cachealigner.go` 和 `cachealigner_test.go`
- [ ] `go test ./...` 全部通过

---

### [T1-008] 提取 HTTP 错误响应辅助函数

**文件：** `proxy/proxy.go`

**新增：**
```go
// writeError 写入 JSON 格式的错误响应。
func writeError(w http.ResponseWriter, msg string, code int) {
    w.Header().Set("Content-Type", "application/json")
    http.Error(w, fmt.Sprintf(`{"error":"%s"}`, msg), code)
}
```

**替换：** 将 `proxy/proxy.go` 中所有 7 处重复模板：
```go
w.Header().Set("Content-Type", "application/json")
http.Error(w, fmt.Sprintf(`{"error":"%s"}`, ...), code)
return
```
统一替换为：
```go
writeError(w, ..., code)
return
```

**验收标准：**
- [ ] `writeError` 函数存在
- [ ] `TestProxy_InvalidJSON` 验证错误响应含 `Content-Type: application/json`
- [ ] `TestProxy_StreamRejected` 验证错误响应含 `Content-Type: application/json`
- [ ] 所有 7 处替换完成

---

## Phase 2：接口抽象与配置统一

---

### [T1-009] 定义 Compressor 接口

**文件：** `compressor.go`（新建）

**内容：**
```go
package headroom

// CompressionConfig 是所有压缩器的统一配置。
type CompressionConfig struct {
    Aggressiveness  float64
    EnableStopwords bool // TextCompressor 专用
}

// Compressor 压缩器接口。
type Compressor interface {
    Compress(content string, cfg CompressionConfig) (string, error)
    Kind() ContentKind
}
```

**验收标准：**
- [ ] `CompressionConfig` 和 `Compressor` 类型定义完成
- [ ] 编译通过（无实现，接口定义阶段）

---

### [T1-010] 实现 SmartCrushCompressor

**文件：** `smartcrusher.go` 或 `smartcrush_compressor.go`（新建）

**内容：** 创建一个实现 `Compressor` 接口的类型：

```go
type smartCrushCompressor struct{}

func (s *smartCrushCompressor) Kind() ContentKind { return KindJSON }

func (s *smartCrushCompressor) Compress(content string, cfg CompressionConfig) (string, error) {
    return SmartCrushJSON(content, SmartCrushConfig{Aggressiveness: cfg.Aggressiveness})
}

func NewSmartCrushCompressor() Compressor {
    return &smartCrushCompressor{}
}
```

**验收标准：**
- [ ] `SmartCrushCompressor` 实现 `Compressor` 接口（编译时验证）
- [ ] `TestSmartCrusher_*` 测试全部通过

---

### [T1-011] 实现 CodeCompressor

**文件：** `codecompressor.go` 或 `code_compressor.go`（新建）

**内容：** 创建一个实现 `Compressor` 接口的类型：

```go
type codeCompressor struct{}

func (c *codeCompressor) Kind() ContentKind { return KindCode }

func (c *codeCompressor) Compress(content string, cfg CompressionConfig) (string, error) {
    return CompressCode(content, CodeConfig{Aggressiveness: cfg.Aggressiveness}), nil
}

func NewCodeCompressor() Compressor {
    return &codeCompressor{}
}
```

**验收标准：**
- [ ] `CodeCompressor` 实现 `Compressor` 接口（编译时验证）
- [ ] `TestCodeCompressor_*` 测试全部通过

---

### [T1-012] 实现 TextCompressor

**文件：** `textcompressor.go` 或 `text_compressor.go`（新建）

**内容：** 创建一个实现 `Compressor` 接口的类型：

```go
type textCompressor struct{}

func (t *textCompressor) Kind() ContentKind { return KindText }

func (t *textCompressor) Compress(content string, cfg CompressionConfig) (string, error) {
    return CompressText(content, TextConfig{Aggressiveness: cfg.Aggressiveness}), nil
}

func NewTextCompressor() Compressor {
    return &textCompressor{}
}
```

**验收标准：**
- [ ] `TextCompressor` 实现 `Compressor` 接口（编译时验证）
- [ ] `TestTextCompressor_*` 测试全部通过

---

### [T1-013] 创建 compressRegistry 并修改 Compress() 使用 map 路由

**文件：** `compressor.go`（扩展）

**新增：**
```go
// compressRegistry 内容类型到压缩器的映射。
var compressRegistry = map[ContentKind]Compressor{
    KindJSON: NewSmartCrushCompressor(),
    KindCode: NewCodeCompressor(),
    KindText: NewTextCompressor(),
}

// compressWithRegistry 使用注册表路由到对应压缩器。
func compressWithRegistry(content string, kind ContentKind, cfg CompressionConfig) (string, error) {
    comp, ok := compressRegistry[kind]
    if !ok {
        return content, nil
    }
    return comp.Compress(content, cfg)
}
```

**文件：** `headroom.go`（修改）

**变更：** `Compress()` 函数中的 switch-case 路由：
```go
// 旧：
switch kind {
case KindJSON:
    out, err = SmartCrushJSON(m.Content, SmartCrushConfig{...})
case KindCode:
    out = CompressCode(m.Content, CodeConfig{...})
default:
    out = CompressText(m.Content, TextConfig{...})
}

// 新：
cfg := CompressionConfig{Aggressiveness: opts.Aggressiveness}
out, err = compressWithRegistry(m.Content, kind, cfg)
```

**验收标准：**
- [ ] `Compress()` 不再包含 switch-case 路由
- [ ] 所有现有测试通过
- [ ] 新增 `TestCompressor_RegistryComplete` 测试：验证所有 `ContentKind` 都能路由到对应压缩器

---

## Phase 3：函数拆分与逻辑重构

---

### [T1-014] 提取 shouldSkip() 守卫函数

**文件：** `headroom.go`

**新增：**
```go
// shouldSkip 判断是否应跳过压缩（空内容/assistant角色/短于TokenLimit）。
func shouldSkip(msg Message, tokenLimit int) bool {
    if msg.Role == "assistant" {
        return true
    }
    if strings.TrimSpace(msg.Content) == "" {
        return true
    }
    if tokenLimit > 0 && estimateTokens(msg.Content) < tokenLimit {
        return true
    }
    return false
}
```

**验收标准：**
- [ ] `shouldSkip()` 函数存在且不超过 10 行
- [ ] `TestCompress_shouldSkip_*` 测试覆盖所有分支（assistant/空内容/TokenLimit/正常）

---

### [T1-015] 提取 postProcess() 后处理函数

**文件：** `headroom.go`

**新增：**
```go
// postProcess 应用可选后处理：前缀对齐、可逆压缩、良性降级。
// 返回最终输出内容。
func postProcess(msg Message, out string, origLen int, opts Options, ccr *CCR) string {
    // 1. 前缀对齐
    if opts.AlignPrefix {
        out = AlignPrefix(out, true, PrefixVersion)
    }

    // 2. 可逆压缩
    if opts.Reversible {
        id := ccr.Store(msg.Content, out, KindForContent(msg.Content))
        retrieveSuffix := fmt.Sprintf("\n\n[headroom:retrieve id=%s]", id)
        out = out + retrieveSuffix
    }

    // 3. 良性降级：压缩后更长则用原文
    if len(out) >= origLen {
        return msg.Content
    }
    return out
}
```

**验收标准：**
- [ ] `postProcess()` 函数存在，逻辑清晰
- [ ] `headroom.go` 中的 `Compress()` 调用 `shouldSkip()` 和 `postProcess()`
- [ ] 所有现有测试通过

---

### [T1-016] 重构 Compress() 主函数

**文件：** `headroom.go`

**目标：** `Compress()` 函数不超过 30 行，主逻辑清晰：

```go
func Compress(messages []Message, opts Options) (*Result, error) {
    ccr := getPackageCCR()

    compressedMsgs := make([]Message, 0, len(messages))
    origTokens := 0
    compTokens := 0

    for _, m := range messages {
        msgTokens := estimateTokens(m.Content)
        origTokens += msgTokens

        if shouldSkip(m, opts.TokenLimit) {
            compressedMsgs = append(compressedMsgs, m)
            compTokens += msgTokens
            continue
        }

        kind := NewContentRouter().Detect(m.Content)
        cfg := CompressionConfig{Aggressiveness: opts.Aggressiveness}
        out, err := compressWithRegistry(m.Content, kind, cfg)
        if err != nil {
            return nil, fmt.Errorf("compression: %w", err)
        }

        out = postProcess(m, out, len(m.Content), opts, ccr)
        compTokens += len(out) / 4

        compressedMsgs = append(compressedMsgs, Message{
            Role:    m.Role,
            Content: out,
            Name:    m.Name,
        })
    }

    return buildResult(compressedMsgs, origTokens, compTokens), nil
}
```

**验收标准：**
- [ ] `Compress()` 不超过 30 行
- [ ] 所有 50 个测试通过

---

### [T1-017] 重构 TextCompressor 消除 flushDup 歧义

**文件：** `textcompressor.go`

**变更：**
- 删除 `flushDup` 闭包
- 定义 `flushDuplicateGroup(processed []string, line string, count int, cfg TextConfig, highPriority bool) []string`
- 统一调用点逻辑

**验收标准：**
- [ ] `flushDup` 闭包不存在
- [ ] 新增 `TestTextCompressor_DuplicateGroupFlushing` 测试边界行为
- [ ] 所有 `TestTextCompressor_*` 通过

---

### [T1-018] 简化 CCR Store 逻辑

**文件：** `ccr.go`

**新增：**
```go
// isFull 判断存储是否已达到最大条目数限制。
func (c *CCR) isFull() bool {
    return c.cfg.MaxEntries > 0 && len(c.data) >= c.cfg.MaxEntries
}

// contains 判断 id 是否已存在于存储中。
func (c *CCR) contains(id string) bool {
    _, ok := c.data[id]
    return ok
}
```

**修改 Store() 函数：**
```go
// 旧：
if cfg.MaxEntries > 0 && len(c.data) >= c.cfg.MaxEntries {
    if _, exists := c.data[id]; !exists {
        c.evictOldest()
    }
}

// 新：
if c.isFull() && !c.contains(id) {
    c.evictOldest()
}
```

**验收标准：**
- [ ] `isFull()` 和 `contains()` 方法存在
- [ ] 新增 `TestCCR_IsFull` 和 `TestCCR_Contains` 测试
- [ ] `TestCCR_StoreAndRetrieve` 等现有测试通过

---

## 验收阶段

---

### [T1-999] 全量回归测试

**执行：**
```bash
go build ./...
go vet ./...
go test -race -count=1 -v ./...
```

**验收标准：**
- [ ] 编译通过，无错误
- [ ] `go vet` 无警告
- [ ] 所有测试通过（含 `-race` 竞态检测）
- [ ] 代码行数净减少（对比 v0.3.0）
