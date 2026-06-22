# Spec: Headroom-Go 根包精简 — 消除根目录代码平铺

**版本:** v0.6.0-spec
**日期:** 2026-06-22
**状态:** 已修订（经 Spec Review 修正 P0×2 + P1×6）

---

## 1. 背景与动机

### 1.1 现状问题

当前根目录仍有 **21 个 `.go` 文件**（全部 `package headroom`），分为 4 类：

| 类别 | 文件数 | 行数 | 问题 |
|------|--------|------|------|
| **Shim 重导出** | 8 | ~265 | `cachealigner.go`/`ccr.go`/`ccrstore.go`/`content_kind.go`/`observability.go`/`router.go`/`tag_protector.go`/`tokenizer.go` — 每个文件 5-53 行，仅做类型别名/函数包装，碎片化严重 |
| **核心逻辑** | 1 | 589 | `headroom.go` — 包含 Compress/CompressString + legacy 路径 + Pipeline 变换 + 兼容函数，职责过多 |
| **Build-tag 桩** | 2 | 20 | `tokenizer_hf.go`/`tokenizer_tiktoken.go` — 可选后端的条件编译桩 |
| **测试** | 9 | ~1800 | E2E/兼容/benchmark/fuzz/fixtures 测试 |
| **版本** | 1 | 20 | `version.go` |

### 1.2 目标

按 Go 标准模块布局，根包只保留 **公共 API 入口**，消除碎片化 shim 文件：

- 根目录非测试 `.go` 文件从 **12 → ≤5**
- 根目录总 `.go` 文件从 **21 → ≤14**
- 所有 shim 合并为 `exports.go` + `compat.go` 两个文件
- `headroom.go` 精简：公共 API 入口，消除与 `internal/engine/` 的重复实现
- 保持 100% 向后兼容，外部 `import "github.com/superops-team/headroom-go"` 零改动

---

## 2. 目标目录结构

```
headroom-go/
├── headroom.go              # 公共 API 入口（Message/Options/Result/Compress/CompressString/DefaultOptions）
├── exports.go               # 类型别名 + 常量 re-export（合并 8 个 shim）
├── compat.go                # 兼容函数（SmartCrushJSON/CompressCode/CompressText 等）
├── version.go               # 版本常量（不变）
├── doc.go                   # Package 文档注释（可选，从 headroom.go 提取）
│
├── *_test.go                # 公共 API 测试（不变）
│   ├── headroom_test.go
│   ├── content_kind_test.go
│   ├── api_compat_test.go
│   ├── spec_a_e2e_test.go
│   ├── spec_b_e2e_test.go
│   ├── spec_d_coverage_test.go
│   ├── benchmark_test.go
│   ├── fixtures_test.go
│   └── fuzz_test.go
│
├── internal/                # 实现细节（已迁移，不变）
│   ├── types/               # Message/Options/Result/ContentKind/Observer 等
│   ├── compressors/         # SmartCrusher/Code/Text/Diff/Log/Search/HTML/Transforms
│   ├── engine/              # CompressionEngine/Pipeline/Policy + Pipeline 变换实现
│   ├── router/              # ContentRouter
│   ├── tokenizer/           # Tokenizer + 后端桩
│   ├── ccr/                 # 可逆压缩存储
│   ├── cachealigner/        # KV Cache 对齐
│   └── tagprotector/        # XML Tag 保护
│
├── proxy/                   # HTTP 代理（不变）
├── cmd/headroom/            # CLI 入口（不变）
├── testdata/                # 测试数据（不变）
├── go.mod
├── README.md
├── CHANGELOG.md
├── CONTRIBUTING.md
├── llms.txt
└── install.sh
```

---

## 3. 详细变更

### 3.1 新建 `doc.go` — Package 文档

将 `headroom.go` 顶部的 package 注释（~30 行）提取到独立的 `doc.go`，符合 Go 惯例（godoc 优先读取 `doc.go`）。

```go
// Package headroom provides intelligent context compression for AI agents.
//
// ... (existing package comment)
package headroom
```

### 3.2 新建 `exports.go` — 合并所有 Shim 类型/常量

合并以下 8 个 shim 文件的类型别名和常量到 `exports.go`（~200 行）：

| 原文件 | 内容 | 合并后 |
|--------|------|--------|
| `content_kind.go` | ContentKind 类型别名 + 10 个常量 | → `exports.go` |
| `observability.go` | Warning/CompressionStep/Observer/NoopObserver | → `exports.go` |
| `cachealigner.go` | CacheAlignerConfig/CacheAligner/NewCacheAligner | → `exports.go` |
| `ccr.go` | CCRConfig/CCR/NewCCR/getPackageCCR | → `exports.go` |
| `ccrstore.go` | CCRStore 类型别名 | → `exports.go` |
| `router.go` | ContentRouter/NewContentRouter | → `exports.go` |
| `tag_protector.go` | ProtectedContent/TagProtector/NewTagProtector | → `exports.go` |
| `tokenizer.go` | TokenizerBackend/TokenizerConfig/Tokenizer/FallbackTokenizer/常量/NewTokenizer/ResolveTokenizer | → `exports.go` |

### 3.3 新建 `compat.go` — 兼容函数

从 `headroom.go` 迁移兼容函数到 `compat.go`（~80 行）：

- `SmartCrushJSON` / `SmartCrushJSONWithSteps`
- `CompressCode` / `CompressText`
- `NewCompressorRegistry` / `NewCompressorFunc` / `DefaultCompressorRegistry`
- `lineIndent` / `errorTokenizer`

### 3.4 Build-tag 桩文件 — 保持不变

`tokenizer_hf.go` 和 `tokenizer_tiktoken.go` **不合并**。两个文件各自受独立 build tag 控制，分别引用对应的 internal stub，合并会破坏单 tag 编译。

### 3.5 `headroom.go` 精简

当前 `headroom.go`（589 行）包含：
- 公共 API：Compress/CompressString/DefaultOptions/NewCompressionEngine 等
- Legacy 路径实现：compressLegacy/legacySkipMessage/routeAndCompressLegacy 等
- Pipeline 变换实现：legacyTextTransform/legacyCodeTransform/jsonMinifierTransform 等
- 兼容函数：SmartCrushJSON/CompressCode/CompressText/lineIndent 等

**变更**：
- 兼容函数（~80 行）移到 `compat.go`
- 消除与 `internal/engine/pipeline.go` 重复的 Pipeline 变换实现（~200 行），根包仅保留 facade
- `headroom.go` 只保留公共 API 入口（Compress/CompressString/DefaultOptions/NewCompressionEngine/NewDefaultPipeline）+ Legacy 路径（compressLegacy 等私有函数）

目标：`headroom.go` 从 589 行 → ~300 行。

### 3.5 测试文件不变

所有 `*_test.go` 保持 `package headroom`，路径不变。测试的是公共 API，不需要迁移。

---

## 4. 向后兼容性保证

| 检查项 | 保证 |
|--------|------|
| `import "github.com/superops-team/headroom-go"` | ✅ 不变 |
| `headroom.Compress()` | ✅ 签名不变 |
| `headroom.CompressString()` | ✅ 签名不变 |
| `headroom.Message` / `Options` / `Result` | ✅ 类型别名不变 |
| `headroom.ContentKind` / `KindJSON` 等 | ✅ 常量不变 |
| `headroom.CacheAligner` / `NewCacheAligner` | ✅ 不变 |
| `headroom.CCR` / `NewCCR` | ✅ 不变 |
| `headroom.ContentRouter` / `NewContentRouter` | ✅ 不变 |
| `headroom.TagProtector` / `NewTagProtector` | ✅ 不变 |
| `headroom.Tokenizer` / `NewTokenizer` | ✅ 不变 |
| `headroom.SmartCrushJSON` / `CompressCode` / `CompressText` | ✅ 不变 |
| `headroom.Version` / `PrefixVersion` | ✅ 不变 |
| `go.mod` module path | ✅ 不变 |
| CLI 参数和输出 | ✅ 不变 |
| Proxy 行为 | ✅ 不变 |

---

## 5. 实施步骤

### Phase 1: 文件合并（无逻辑变更）

| 步骤 | 内容 | 验证 |
|------|------|------|
| 1.1 | 创建 `doc.go`（可选），提取 package 注释 | `go build` |
| 1.2 | 创建 `exports.go`，合并 8 个 shim 的类型别名和常量 | `go build` |
| 1.3 | 创建 `compat.go`，迁移兼容函数 | `go build` |
| 1.4 | 删除 8 个旧 shim 文件 | `go build` |
| 1.5 | 运行全量测试 + build-tag 矩阵 | `go test -race ./...` |

### Phase 2: headroom.go 精简

| 步骤 | 内容 | 验证 |
|------|------|------|
| 2.1 | 消除与 `internal/engine/pipeline.go` 重复的 Pipeline 变换实现 | `go build` |
| 2.2 | 根包仅保留 facade，委托 internal engine | `go build` |
| 2.3 | 运行全量测试 + 外部 import 编译 | `go test -race ./...` |

### Phase 3: 验证

| 步骤 | 内容 |
|------|------|
| 3.1 | `go test -race -count=1 ./...` 全部通过 |
| 3.2 | `go test -run '^$' -tags tokenizer_hf ./...` 通过 |
| 3.3 | `go test -run '^$' -tags tokenizer_tiktoken ./...` 通过 |
| 3.4 | `go vet ./...` 零警告 |
| 3.5 | 覆盖率不下降 |
| 3.6 | 根目录非测试 `.go` 文件 ≤5 个 |
| 3.7 | 外部 temp module import 编译通过 |

---

## 6. 文件变更清单

| 操作 | 文件 | 说明 |
|------|------|------|
| **新建** | `doc.go` | Package 文档（可选） |
| **新建** | `exports.go` | 合并 8 个 shim 的类型别名和常量 |
| **新建** | `compat.go` | 兼容函数（SmartCrushJSON/CompressCode/CompressText 等） |
| **修改** | `headroom.go` | 精简为公共 API 入口 + Legacy 路径 |
| **删除** | `cachealigner.go` | → exports.go |
| **删除** | `ccr.go` | → exports.go |
| **删除** | `ccrstore.go` | → exports.go |
| **删除** | `content_kind.go` | → exports.go |
| **删除** | `observability.go` | → exports.go |
| **删除** | `router.go` | → exports.go |
| **删除** | `tag_protector.go` | → exports.go |
| **删除** | `tokenizer.go` | → exports.go |

---

## 7. 验收标准

- [ ] 根目录非测试 `.go` 文件从 12 → ≤5 个
- [ ] 根目录总 `.go` 文件从 21 → ≤14 个
- [ ] `go build ./...` 编译成功
- [ ] `go test -race -count=1 ./...` 全部通过
- [ ] `go test -run '^$' -tags tokenizer_hf ./...` 通过
- [ ] `go test -run '^$' -tags tokenizer_tiktoken ./...` 通过
- [ ] `go vet ./...` 零警告
- [ ] 覆盖率不下降（根包 ≥ 73%，proxy ≥ 91%）
- [ ] 外部 temp module import 编译通过（覆盖 Compress/CompressString/所有 shim 类型/Pipeline 公开 API）
- [ ] `headroom compress --stats` CLI 功能正常
- [ ] `headroom proxy --port=18787` + `curl localhost:18787/healthz` 正常
- [ ] CHANGELOG 更新

---

## 8. 风险与缓解

| 风险 | 影响 | 缓解 |
|------|------|------|
| 合并文件导致 git blame 混乱 | 低 | 先复制合并（不改逻辑），后删除旧文件，避免同一 diff 混入格式化 |
| Pipeline 变换消除破坏测试 | 中 | 根包测试通过公共 API 覆盖；私有 helper 引用改为外部行为断言 |
| Build-tag 桩合并破坏单 tag 编译 | — | **已取消合并**，保留两个独立 tag 文件 |
| `exports.go` 职责混合 | 低 | 拆为 `exports.go`（类型/常量）+ `compat.go`（兼容函数） |
| 根包/internal legacy 双实现 | 中 | Phase 2 消除重复，根包仅 facade |

---

## 9. 时间估算

| Phase | 预估时间 |
|-------|---------|
| Phase 1: 文件合并 | 20 分钟 |
| Phase 2: headroom.go 精简 | 20 分钟 |
| Phase 3: 验证 | 10 分钟 |
| **总计** | **~50 分钟** |

---

*本 spec 待教主确认后进入执行阶段。*
