# headroom-go v0.4.0 技术重构规范

## 1. 概述

本规范是历史 v0.4.0 重构草案，保留用于追溯。当前基线为 v0.5.0：`version.go`、`CompressionEngine`、`CompressorRegistry` 已存在，且兼容策略要求保留旧公开 API。下文中“删除旧 API / 删除 CacheAligner / 回退到 v0.4.0”的条目已过期，不作为当前执行要求。

本规范基于 Brooks-Lint 代码审查方法论，针对 headroom-go v0.3.0 进行系统性重构。审查发现代码在正确性和测试覆盖方面表现优秀（50/50 测试通过，含 `-race` 竞态检测），但结构性技术债务正在积累，主要集中在以下六个维度：

- 认知过载（长函数、职责不清）
- 变更传播（版本号/配置散落多处）
- 知识重复（重复代码片段）
- 偶然复杂性（不必要的间接层）
- 依赖混乱（模块/包职责边界模糊）
- 领域模型失真（缺乏抽象接口）

**目标：** 在保持 100% 向后兼容的前提下，通过结构优化降低长期维护成本，提升代码可扩展性。

## 2. 范围

### 2.1 纳入范围

| 模块 | 改动内容 |
|------|----------|
| `headroom.go` | 拆分 `Compress()` 长函数、提取 Options 聚合接口 |
| `router.go` | 引入 `Compressor` 接口，消除 switch 硬编码路由 |
| `smartcrusher.go` | 实现 `Compressor` 接口，统一配置结构 |
| `codecompressor.go` | 实现 `Compressor` 接口，简化函数折叠逻辑 |
| `textcompressor.go` | 实现 `Compressor` 接口，消除 flushDup 歧义 |
| `ccr.go` | 明确单例策略，提取常量 |
| `cachealigner.go` | 简化为纯函数，消除不必要的 struct 封装 |
| `proxy/proxy.go` | 提取 HTTP 错误响应辅助函数，消除重复模板 |
| 所有测试文件 | 适配接口变更，确保回归通过 |

### 2.2 排除范围

- 不修改压缩算法逻辑（压缩效果不变）
- 不修改 CLI 接口（用户可见行为不变）
- 不修改 `go.mod` module path
- 不修改 `CCR` 的核心数据结构（TTL/maxEntries 行为不变）

## 3. 详细规范

### 3.1 任务 1：引入 Compressor 接口（重构路由层）

#### 3.1.1 问题描述

当前 `Compress()` 函数使用 switch-case 硬编码路由，新增内容类型需要修改多处代码：

```go
// headroom.go 现状
switch kind {
case KindJSON:
    out, err = SmartCrushJSON(content, SmartCrushConfig{...})
case KindCode:
    out = CompressCode(content, CodeConfig{...})
default:
    out = CompressText(content, TextConfig{...})
}
```

新增 `KindMarkdown` 需要同时修改：`ContentKind` 枚举、`router.go`、`headroom.go` 三处。

#### 3.1.2 设计方案

定义统一的 `Compressor` 接口，让每个压缩器实现该接口：

```go
// Compressor 压缩器接口。所有压缩器必须实现此接口。
type Compressor interface {
    // Compress 对 content 进行压缩，返回压缩后的内容。
    Compress(content string) (string, error)
    // Kind 返回此压缩器处理的内容类型。
    Kind() ContentKind
}

// compressRegistry 内容类型到压缩器的映射。
var compressRegistry = map[ContentKind]Compressor{
    KindJSON: NewSmartCrushCompressor(),
    KindCode: NewCodeCompressor(),
    KindText: NewTextCompressor(),
}
```

**优点：**
- 新增类型只需注册到 map 中，不修改 `Compress()` 主逻辑
- 每个压缩器独立实现，可单独测试
- `ContentKind` 枚举仍然保留（用于 CCR 存储标识），但路由逻辑解耦

#### 3.1.3 验收标准

- [ ] 定义 `Compressor` 接口，包含 `Compress(string) (string, error)` 和 `Kind() ContentKind`
- [ ] 创建 `compressRegistry map[ContentKind]Compressor`
- [ ] `SmartCrushCompressor` 实现 `Compressor` 接口
- [ ] `CodeCompressor` 实现 `Compressor` 接口
- [ ] `TextCompressor` 实现 `Compressor` 接口
- [ ] `router.go` 不变（仍返回 `ContentKind` 枚举）
- [ ] `Compress()` 使用 map 查找替代 switch-case
- [ ] 现有 50 个测试全部通过
- [ ] 新增接口单元测试（验证各压缩器返回正确的 `Kind()`）

#### 3.1.4 实现步骤

1. 在 `router.go` 同级创建 `compressor.go`，定义 `Compressor` 接口
2. 将 `smartcrusher.go` 包装为 `SmartCrushCompressor`（添加 `Kind()` 方法）
3. 将 `codecompressor.go` 包装为 `CodeCompressor`（添加 `Kind()` 方法）
4. 将 `textcompressor.go` 包装为 `TextCompressor`（添加 `Kind()` 方法）
5. 创建 `compressRegistry` 并在 `headroom.go` 中使用
6. 历史草案曾建议收敛独立函数导出；当前 v0.5.0 兼容基线要求继续保留 `SmartCrushJSON` / `CompressCode` / `CompressText` 函数导出。
7. 运行全量测试回归

---

### 3.2 任务 2：拆分 Compress() 长函数（认知过载治理）

#### 3.2.1 问题描述

`Compress()` 函数 81 行，包含 5 个守卫检查、3 种压缩器调用、2 个可选后处理（prefix/reversible），平均心智负担过高。

#### 3.2.2 设计方案

将函数拆分为以下步骤函数：

```go
// Compress 压缩消息，主入口（保持公开 API 不变）
func Compress(messages []Message, opts Options) (*Result, error) {
    compressor := newCompressorChain(opts)
    aligner := newCacheAligner(opts)
    ccr := getPackageCCR()

    compressedMsgs, stats := compressMessages(messages, compressor, aligner, ccr)
    return buildResult(compressedMsgs, stats)
}

// compressMessages 处理单条消息的循环（提取为独立函数）
func compressMessages(messages []Message, comp Compressor, aligner *CacheAligner, ccr *CCR) ([]Message, compressionStats)

// shouldSkip 返回是否跳过压缩（守卫条件提取）
func shouldSkip(msg Message, tokenLimit int) bool

// postProcess 应用可选后处理（对齐前缀、可逆压缩、良性降级）
func postProcess(msg Message, out string, origLen int, opts Options, ccr *CCR) string
```

#### 3.2.3 验收标准

- [ ] `Compress()` 主函数不超过 30 行
- [ ] `compressMessages()` 提取为包级私有函数
- [ ] `shouldSkip()` 提取为包级私有函数，测试覆盖率 100%
- [ ] `postProcess()` 提取为包级私有函数，处理 prefix/reversible/良性降级逻辑
- [ ] 现有 50 个测试全部通过

---

### 3.3 任务 3：统一压缩配置结构（变更传播治理）

#### 3.3.1 问题描述

Aggressiveness 配置在多个 Config 结构体中重复定义：

```go
type SmartCrushConfig  { Aggressiveness float64 }
type CodeConfig        { Aggressiveness float64 }
type TextConfig         { Aggressiveness float64 }
```

新增全局选项（如 `EnableStopwords bool`）需要修改 4 个结构体。

#### 3.3.2 设计方案

定义统一的 `CompressionConfig`：

```go
// CompressionConfig 是所有压缩器的统一配置。
type CompressionConfig struct {
    Aggressiveness  float64 // 0.0-1.0，默认 0.5
    EnableStopwords bool   // TextCompressor 专用：是否移除 stopwords，默认 true
    // 未来扩展字段...
}

// Compressor 接口更新为接收统一配置：
type Compressor interface {
    Compress(content string, cfg CompressionConfig) (string, error)
    Kind() ContentKind
}
```

#### 3.3.3 验收标准

- [ ] 定义 `CompressionConfig` 结构体
- [ ] 所有压缩器接受 `CompressionConfig` 而非各自的 Config
- [ ] 消除 `SmartCrushConfig`、`CodeConfig`、`TextConfig`（或降级为 `CompressionConfig` 别名）
- [ ] `Options` 中的 `Aggressiveness` 映射到 `CompressionConfig.Aggressiveness`
- [ ] 现有测试全部通过（内部调整兼容层即可）

---

### 3.4 任务 4：提取 HTTP 错误响应辅助函数（知识重复治理）

#### 3.4.1 问题描述

`proxy/proxy.go` 中 HTTP 错误响应模板重复 7 次：

```go
w.Header().Set("Content-Type", "application/json")
http.Error(w, fmt.Sprintf(`{"error":"%s"}`, msg), code)
return
```

#### 3.4.2 设计方案

```go
// writeError 写入 JSON 格式的错误响应。
func writeError(w http.ResponseWriter, msg string, code int) {
    w.Header().Set("Content-Type", "application/json")
    http.Error(w, fmt.Sprintf(`{"error":"%s"}`, msg), code)
}

// 使用示例：
writeError(w, err.Error(), http.StatusBadRequest)
return
```

#### 3.4.3 验收标准

- [ ] 定义 `writeError(w http.ResponseWriter, msg string, code int)` 辅助函数
- [ ] `proxy/proxy.go` 中所有 7 处重复模板替换为 `writeError()` 调用
- [ ] 新增 `proxy/proxy_test.go` 测试用例验证 `writeError` 输出格式正确（含 `Content-Type` 头）

---

### 3.5 任务 5：简化 CacheAligner 为纯函数（偶然复杂性治理）

#### 3.5.1 问题描述

`CacheAligner` 类型仅包含一个配置字段 + 一个方法，与无状态函数风格不一致：

```go
type CacheAligner struct{ cfg CacheAlignerConfig }
func NewCacheAligner(cfg CacheAlignerConfig) *CacheAligner
func (a *CacheAligner) Align(content string) string
```

#### 3.5.2 设计方案

```go
// AlignPrefix 为 content 添加版本前缀。
// 若 enabled 为 false 或 version 为空，直接返回原内容。
func AlignPrefix(content string, enabled bool, version string) string {
    if !enabled || version == "" {
        return content
    }
    return fmt.Sprintf("[headroom/%s]\n%s", version, content)
}

// headroom.go 中的调用：
if opts.AlignPrefix {
    out = AlignPrefix(out, true, "v0.3")
}
```

#### 3.5.3 验收标准

- [x] v0.5.0 兼容决策：保留 `CacheAligner` struct 和 `NewCacheAligner()` 构造函数
- [ ] 定义 `AlignPrefix(content string, enabled bool, version string) string` 纯函数
- [ ] `headroom.go` 中使用新的函数调用
- [x] v0.5.0 兼容决策：保留 `cachealigner.go` 与 `cachealigner_test.go`
- [ ] 现有测试全部通过

---

### 3.6 任务 6：提取版本常量和明确 CCR 单例策略

#### 3.6.1 问题描述

版本号散落 5 处，升级时需全局搜索替换：

| 位置 | 值 |
|------|-----|
| `headroom.go:71` | `"v0.3"` |
| `proxy/proxy.go:80` | `"v0.3.0"` |
| `ccr.go:60` | `"v2_"` |
| `cmd/headroom/main.go:38` | `"v0.3.0"` |
| `proxy/proxy_test.go` | `"streaming not supported in v0.3"` |

#### 3.6.2 设计方案

```go
// version.go — 版本常量统一管理
package headroom

const (
    // Version 是 headroom-go 的语义化版本。
    Version = "v0.5.0"

    // PrefixVersion 是缓存对齐前缀的版本。
    // 每次压缩算法升级导致输出变化时应递增。
    PrefixVersion = "v0.4"

    // CCRIDVersion 是可逆压缩 ID 的版本前缀。
    // 每次存储格式变化时应递增。
    CCRIDVersion = "v3"
)
```

CLI 和 proxy 的版本从 `headroom.Version` 导入使用。

#### 3.6.3 验收标准

- [ ] 创建 `version.go`，定义 `Version`、`PrefixVersion`、`CCRIDVersion` 三个常量
- [ ] `headroom.go` 中 `CacheAligner` 的 version 使用 `PrefixVersion`
- [ ] `ccr.go` 中的 ID 前缀使用 `CCRIDVersion`
- [ ] `cmd/headroom/main.go` 中的版本打印使用 `headroom.Version`
- [ ] `proxy/proxy.go` 中的 /healthz 响应使用 `headroom.Version`
- [ ] `proxy/proxy_test.go` 中的测试断言使用 `headroom.Version`
- [ ] 现有测试全部通过

---

### 3.7 任务 7：消除 textcompressor flushDup 歧义

#### 3.7.1 问题描述

`textcompressor.go` 中 `flushDup` 闭包在两个位置被调用，参数逻辑不一致：

```go
// 空行触发：硬编码 false
flushDup(removeStopwordsIfNeeded(origLine, cfg, false))

// 新行触发：动态计算 isHighPriority
prevProc := removeStopwordsIfNeeded(origLine, cfg, isHighPriority(origLine))
```

#### 3.7.2 设计方案

重构为显式函数，消除闭包歧义：

```go
// flushDuplicateGroup 将累积的重复行组写入 processed。
// highPriority 表示这组行是否包含 FATAL/ERROR（决定是否移除 stopwords）。
func flushDuplicateGroup(processed []string, line string, count int, cfg TextConfig, highPriority bool) []string {
    if line == "" {
        return processed
    }
    if count <= 0 {
        return processed
    }

    output := line
    if !highPriority && cfg.Aggressiveness >= 0.3 {
        output = removeStopwords(line)
    }

    if count == 1 {
        return append(processed, output)
    }
    return append(processed, fmt.Sprintf("%s [x%d]", output, count))
}
```

#### 3.7.3 验收标准

- [ ] 删除 `flushDup` 闭包
- [ ] 定义 `flushDuplicateGroup(processed, line, count, cfg, highPriority) []string`
- [ ] 空行处理：调用 `flushDuplicateGroup(processed, origLine, dupCount, cfg, false)` 后重置
- [ ] 新行处理：调用 `flushDuplicateGroup(processed, origLine, dupCount, cfg, isHighPriority(origLine))` 后重置
- [ ] 最后一组处理：同样调用 `flushDuplicateGroup`
- [ ] 新增 `TestTextCompressor_DuplicateGroupFlushing` 测试用例验证边界行为
- [ ] 现有 50 个测试全部通过

---

### 3.8 任务 8：简化 CCR Store 逻辑

#### 3.8.1 问题描述

`ccr.go:66-70` 的双重条件守卫语义隐晦：

```go
if cfg.MaxEntries > 0 && len(c.data) >= c.cfg.MaxEntries {
    if _, exists := c.data[id]; !exists {
        c.evictOldest()
    }
}
```

#### 3.8.2 设计方案

```go
// isFull 判断存储是否已达到最大条目数限制。
func (c *CCR) isFull() bool {
    return c.cfg.MaxEntries > 0 && len(c.data) >= c.cfg.MaxEntries
}

// Store 中的调用简化为：
if c.isFull() && !c.contains(id) {
    c.evictOldest()
}
```

#### 3.8.3 验收标准

- [ ] 定义 `isFull() bool` 方法替代内联条件
- [ ] 定义 `contains(id string) bool` 方法替代 map 查找
- [ ] `Store()` 中的逻辑可读性提升
- [ ] 新增 `TestCCR_IsFull` 和 `TestCCR_Contains` 测试用例
- [ ] 现有 CCR 测试全部通过

---

## 4. 实现顺序

```
Phase 1（独立可测，不破坏现有代码）
├── 任务 6：提取版本常量       (version.go)
├── 任务 5：简化 CacheAligner (AlignPrefix 纯函数)
└── 任务 4：HTTP 错误辅助函数   (writeError)

Phase 2（依赖 Phase 1）
├── 任务 1：引入 Compressor 接口
│   ├── 定义接口 + compressRegistry
│   ├── 包装各压缩器
│   └── 修改 headroom.go 使用 map 路由
├── 任务 3：统一压缩配置结构   (CompressionConfig)
└── 任务 2：拆分 Compress()    (依赖任务 1/3 完成后)

Phase 3（收尾）
├── 任务 7：消除 flushDup 歧义  (TextCompressor 重构)
└── 任务 8：简化 CCR Store 逻辑
```

**每个 Phase 完成后必须运行：**
```bash
go build ./... && go test -race -count=1 ./...
```

## 5. 向后兼容性保证

- 所有 `package headroom` 的公开 API（`Compress`、`CompressString`、`Options`、`Result`、`Message`）保持不变
- `go.mod` 的 module path 不变
- CLI 参数和输出格式不变
- CCR 的存储格式通过 `CCRIDVersion` 常量管理，版本升级时旧缓存自动失效

## 6. 测试策略

| 任务 | 新增测试用例 |
|------|-------------|
| 任务 1 | `TestCompressor_Interface_Compliance`（验证各压缩器 Kind() 正确） |
| 任务 2 | `TestCompress_shouldSkip`（守卫条件覆盖率 100%） |
| 任务 4 | `TestProxy_WriteError_Format`（验证错误响应格式） |
| 任务 7 | `TestTextCompressor_DuplicateGroupFlushing`（边界行为） |
| 任务 8 | `TestCCR_IsFull`、`TestCCR_Contains` |

## 7. 风险评估

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 接口重构导致现有用户代码编译失败 | 高 | 仅修改 `package headroom` 内部结构，不改变公开 API |
| 测试覆盖盲区引入回归 | 中 | 每个任务后运行完整测试套件 |
| Phase 顺序依赖导致返工 | 低 | Phase 1 的任务完全独立，可独立验证 |

## 8. 成功标准

- [ ] 50 个现有测试全部通过（含 `-race`）
- [ ] 新增测试覆盖所有重构后的新增函数
- [ ] `go vet` / `go vet ./...` 无警告
- [ ] 代码行数净减少（通过删除冗余结构体和重复代码）
- [ ] 每个公开 API 有完整的 godoc 注释
