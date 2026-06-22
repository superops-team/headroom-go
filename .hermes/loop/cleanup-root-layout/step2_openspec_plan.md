# cleanup-root-layout Step 2 OpenSpec Plan

## 1. 需求目标

基于已修订 Spec `openspec/specs/cleanup-root-layout.md`，对 `headroom-go` 根包进行纯文件重组，消除根目录 shim 文件碎片化，同时保持公开 API、行为逻辑、测试结果和 build-tag 兼容性不变。

本计划覆盖以下目标：

1. 将 8 个根包 shim 文件合并为 `exports.go`，集中承载类型、常量、变量和构造函数 re-export。
2. 将历史兼容类型、函数和 helper 从 `headroom.go` 迁移到 `compat.go`。
3. 精简 `headroom.go`，保留公共 API 入口，消除与 `internal/engine/pipeline.go` 重复的 Pipeline 实现。
4. 保持 build-tag 桩 `tokenizer_hf.go` / `tokenizer_tiktoken.go` 独立不变。
5. 删除 8 个旧 shim 文件。
6. 确保 `go build`、`go test`、`go test -race`、build-tag 矩阵、外部 import smoke 等验证通过。

## 2. 非目标

本次变更不包含：

1. 不新增功能。
2. 不改变压缩算法、token 统计逻辑、JSON 处理逻辑、HTML 清理逻辑或 CCR 行为。
3. 不改变任何公开 API 的函数签名、类型名、常量名、变量名、返回值语义或方法集。
4. 不重写 `internal/engine` 的核心行为。
5. 不合并 `tokenizer_hf.go` / `tokenizer_tiktoken.go`。
6. 不做性能优化。
7. 不引入新的 goroutine、锁、共享状态或并发模型。
8. 不弱化现有测试断言。
9. `doc.go` 为可选项，不作为必须交付。

## 3. Spec 与代码事实依据

### 3.1 Spec 依据

| 依据 | 内容 |
| --- | --- |
| `openspec/specs/cleanup-root-layout.md:13` | 当前根目录存在 21 个 `.go` 文件。 |
| `openspec/specs/cleanup-root-layout.md:17` | 明确 8 个 shim 文件列表。 |
| `openspec/specs/cleanup-root-layout.md:25` | 根包只保留公共 API 入口。 |
| `openspec/specs/cleanup-root-layout.md:27` | 根目录非测试 `.go` 文件目标为 12 → ≤5。 |
| `openspec/specs/cleanup-root-layout.md:29` | shim 合并为 `exports.go` + `compat.go`。 |
| `openspec/specs/cleanup-root-layout.md:92` | `exports.go` 合并所有 shim 类型/常量。 |
| `openspec/specs/cleanup-root-layout.md:107` | `compat.go` 承载兼容函数。 |
| `openspec/specs/cleanup-root-layout.md:116` | build-tag 桩文件保持不变。 |
| `openspec/specs/cleanup-root-layout.md:120` | `headroom.go` 精简。 |
| `openspec/specs/cleanup-root-layout.md:216` | 最终验收标准。 |

### 3.2 待合并 shim 符号

| 当前文件 | 当前符号 | 目标归属 |
| --- | --- | --- |
| `cachealigner.go:10`、`cachealigner.go:26`、`cachealigner.go:30` | `CacheAlignerConfig`、`CacheAligner`、`NewCacheAligner` | `exports.go` |
| `ccr.go:13`、`ccr.go:30`、`ccr.go:34`、`ccr.go:38` | `CCRConfig`、`CCR`、`NewCCR`、`getPackageCCR` | `exports.go` |
| `ccrstore.go:5` | `CCRStore` | `exports.go` |
| `content_kind.go:12`、`content_kind.go:27` | `ContentKind`、`KindText` / `KindJSON` / `KindCode` / `KindDiff` / `KindLog` / `KindSearch` / `KindTabular` / `KindSpreadsheet` / `KindHTML` / `KindUnknown` | `exports.go` |
| `observability.go:14`、`observability.go:29`、`observability.go:42`、`observability.go:46` | `Warning`、`CompressionStep`、`Observer`、`NoopObserver` | `exports.go` |
| `router.go:15`、`router.go:18` | `ContentRouter`、`NewContentRouter` | `exports.go` |
| `tag_protector.go:10`、`tag_protector.go:26`、`tag_protector.go:29` | `ProtectedContent`、`TagProtector`、`NewTagProtector` | `exports.go` |
| `tokenizer.go:11`、`tokenizer.go:18`、`tokenizer.go:26`、`tokenizer.go:30`、`tokenizer.go:33`、`tokenizer.go:41`、`tokenizer.go:46`、`tokenizer.go:51` | `TokenizerBackend`、`TokenizerConfig`、`Tokenizer`、`FallbackTokenizer`、`TokenizerFallback` / `TokenizerTiktoken` / `TokenizerHF`、`ErrTokenizerNotImplemented`、`NewTokenizer`、`ResolveTokenizer` | `exports.go` |

### 3.3 `headroom.go` 保留的公共入口

| 符号 | 当前位置 | 规划归属 |
| --- | --- | --- |
| `Message` | `headroom.go:55` | `headroom.go` |
| `Options` | `headroom.go:73` | `headroom.go` |
| `Result` | `headroom.go:84` | `headroom.go` |
| `CompressionEngine` | `headroom.go:88` | `headroom.go` |
| `CompressionContext` | `headroom.go:91` | `headroom.go` |
| `TransformError` | `headroom.go:94` | `headroom.go` |
| `ReformatTransform` | `headroom.go:97` | `headroom.go` |
| `DefaultOptions` | `headroom.go:115` | `headroom.go` |
| `NewCompressionEngine` | `headroom.go:127` | `headroom.go` |
| `DefaultCompressionPolicy` | `headroom.go:137` | `headroom.go` |
| `NewTransformError` | `headroom.go:148` | `headroom.go` |
| `NewDefaultPipeline` | `headroom.go:156` | `headroom.go`，但委托 `internal/engine` |
| `Compress` | `headroom.go:181` | `headroom.go` |
| `CompressString` | `headroom.go:202` | `headroom.go` |

### 3.4 迁移到 `compat.go` 的兼容内容

| 符号 | 当前位置 | 目标归属 |
| --- | --- | --- |
| `errorTokenizer` | `headroom.go:531` | `compat.go` |
| `CompressionConfig` / `CodeConfig` / `TextConfig` / `SmartCrushConfig` | `headroom.go:541` | `compat.go` |
| `Compressor` / `CompressorFunc` / `CompressorRegistry` | `headroom.go:545` | `compat.go` |
| `SmartCrushJSON` | `headroom.go:549` | `compat.go` |
| `SmartCrushJSONWithSteps` | `headroom.go:553` | `compat.go` |
| `CompressCode` | `headroom.go:557` | `compat.go` |
| `CompressText` | `headroom.go:561` | `compat.go` |
| `lineIndent` | `headroom.go:565` | `compat.go` |
| `NewCompressorFunc` | `headroom.go:579` | `compat.go` |
| `NewCompressorRegistry` | `headroom.go:583` | `compat.go` |
| `DefaultCompressorRegistry` | `headroom.go:587` | `compat.go` |

### 3.5 Pipeline 去重依据

| 根包重复实现 | internal 权威实现 |
| --- | --- |
| `headroom.go:99` `Pipeline` | `internal/engine/pipeline.go:17` `Pipeline` |
| `headroom.go:354` `Pipeline.Run` | `internal/engine/pipeline.go:26` `Pipeline.Run` |
| `headroom.go:424` `countTokensForPipeline` | `internal/engine/pipeline.go:96` `countTokensForPipeline` |
| `headroom.go:438` `appliesTo` | `internal/engine/pipeline.go:166` `appliesTo` |
| `headroom.go:451` `legacyTextTransform` | `internal/engine/pipeline.go:175` `legacyTextTransform` |
| `headroom.go:460` `legacyCodeTransform` | `internal/engine/pipeline.go:184` `legacyCodeTransform` |
| `headroom.go:469` `jsonMinifierTransform` | `internal/engine/pipeline.go:193` `jsonMinifierTransform` |
| `headroom.go:482` `jsonOffloadTransform` | `internal/engine/pipeline.go:206` `jsonOffloadTransform` |
| `headroom.go:484` `NewJSONOffloadTransform` | `internal/engine/pipeline.go:208` `NewJSONOffloadTransform` |
| `headroom.go:508` `warningFromTransformError` | `internal/engine/pipeline.go:232` `warningFromTransformError` |
| `headroom.go:515` `htmlCleanTransform` | `internal/engine/pipeline.go:239` `htmlCleanTransform` |
| `headroom.go:523` `removeHTMLBlock` | `internal/engine/pipeline.go:247` 相关 HTML 清理 |
| `headroom.go:527` `removeHTMLComments` | `internal/engine/pipeline.go:251` 相关 HTML 清理 |

### 3.6 build-tag 约束

| 文件 | 约束 | 规划 |
| --- | --- | --- |
| `tokenizer_hf.go:1` | `//go:build tokenizer_hf` | 保持独立，不合并、不删除、不改 tag。 |
| `tokenizer_tiktoken.go:1` | `//go:build tokenizer_tiktoken` | 保持独立，不合并、不删除、不改 tag。 |

## 4. 模块边界

### 4.1 根包 `headroom`

职责：

1. 作为 `github.com/superops-team/headroom-go` 的唯一公共入口。
2. 暴露稳定 API 与历史兼容 API。
3. 通过 re-export 屏蔽 internal 包布局。
4. 委托 `internal/engine` 执行核心压缩与 Pipeline 行为。

文件边界：

| 文件 | 模块职责 | 允许内容 | 禁止内容 |
| --- | --- | --- | --- |
| `headroom.go` | 公共 API facade | `Message`、`Options`、`Result`、`Compress`、`CompressString`、engine/pipeline facade | shim re-export、compat 函数、大段 Pipeline 重复实现 |
| `exports.go` | re-export 聚合 | 类型别名、常量别名、错误变量、构造函数转发 | 压缩算法、Pipeline 逻辑、兼容函数实现、build-tag 桩 |
| `compat.go` | 历史兼容 API | `SmartCrushJSON`、`CompressCode`、`CompressText`、registry、兼容 helper | shim re-export、Pipeline 权威实现 |
| `version.go` | 版本常量 | `Version`、`PrefixVersion`、CCR ID version | 任何重组逻辑 |
| `tokenizer_hf.go` | `tokenizer_hf` tag 桩 | 当前 tag-specific stub | 与其他 tokenizer 桩合并 |
| `tokenizer_tiktoken.go` | `tokenizer_tiktoken` tag 桩 | 当前 tag-specific stub | 与其他 tokenizer 桩合并 |
| `doc.go` | 可选 package 文档 | package 注释 | 功能逻辑 |

### 4.2 `internal/engine`

职责：

1. 持有 Pipeline 权威实现。
2. 持有 `CompressionEngine` 核心执行逻辑。
3. 持有 policy、transform、warning、token 统计等内部行为。

契约：根包不得再复制 `internal/engine/pipeline.go` 的行为实现；根包如需保留 `NewDefaultPipeline`，必须以类型别名或薄 facade 委托 internal engine。

### 4.3 其他 internal 模块

| internal 模块 | 根包暴露方式 | 变更原则 |
| --- | --- | --- |
| `internal/types` | `headroom.go` / `exports.go` 类型别名 | 不改模型字段和行为。 |
| `internal/tokenizer` | `exports.go` re-export + build-tag 桩 | 不改 backend 枚举和 fallback 行为。 |
| `internal/router` | `exports.go` re-export | 不改路由行为。 |
| `internal/ccr` | `exports.go` re-export | 不改 reversible CCR 行为。 |
| `internal/cachealigner` | `exports.go` re-export | 不改 prefix align 行为。 |
| `internal/tagprotector` | `exports.go` re-export | 不改 tag 保护行为。 |
| `internal/compressors` | `compat.go` 兼容入口 | 不改 JSON/code/text 压缩行为。 |

## 5. 接口契约

### 5.1 公开 API 兼容契约

本次变更必须保持以下契约：

1. 所有当前导出的类型名保持不变。
2. 所有当前导出的函数名保持不变。
3. 所有当前导出的常量名保持不变。
4. 所有当前导出的变量名保持不变。
5. 所有公开函数签名保持不变。
6. 所有类型别名应优先使用 `type X = internal.X`，不得误改为新定义类型导致方法集或赋值兼容性变化。
7. 外部用户继续通过 `import "github.com/superops-team/headroom-go"` 使用根包，无需改代码。

### 5.2 行为兼容契约

以下行为必须保持不变：

1. `Compress`、`CompressString` 行为。
2. `DefaultOptions` 默认值。
3. `SmartCrushJSON` / `SmartCrushJSONWithSteps` 行为。
4. `CompressCode` / `CompressText` 行为。
5. `NewTokenizer` / `ResolveTokenizer` fallback 与错误行为。
6. CCR、router、tag protector、cache aligner 行为。
7. Pipeline transform 顺序、warning 语义、错误处理、token 统计和 fallback 语义。
8. build-tag 下 tokenizer stub 编译行为。

### 5.3 测试可见性契约

当前根包测试直接引用若干私有符号，需显式处理：

| 测试引用 | 风险 | 处理原则 |
| --- | --- | --- |
| `spec_b_e2e_test.go:257-347` 使用 legacy helpers / `getPackageCCR` | 删除或移动私有 helper 可能导致测试失败 | 保留根包测试可见性，或改为等价公开行为断言。 |
| `spec_a_e2e_test.go:184-207` 构造 `Pipeline` / `jsonMinifierTransform` | Pipeline 去重后私有 transform 可能消失 | 改为 `NewDefaultPipeline` 或公开压缩入口行为测试，避免复制旧实现。 |
| `spec_b_e2e_test.go:208-224` 构造 `Pipeline` / `jsonMinifierTransform` | 同上 | 同上。 |
| `spec_d_coverage_test.go:110-123` 调用 `lineIndent` | compat 迁移后需保持根包可见 | `lineIndent` 迁移到 `compat.go` 后仍在 `package headroom` 可见，或改为等价行为测试。 |

## 6. 数据模型规划

本需求不新增业务数据模型，仅调整现有模型的文件归属。

### 6.1 Re-export 数据模型

| 数据模型类别 | 符号 | 目标文件 |
| --- | --- | --- |
| 缓存对齐 | `CacheAlignerConfig`、`CacheAligner` | `exports.go` |
| CCR | `CCRConfig`、`CCR`、`CCRStore` | `exports.go` |
| 内容类型 | `ContentKind`、`KindText`、`KindJSON`、`KindCode`、`KindDiff`、`KindLog`、`KindSearch`、`KindTabular`、`KindSpreadsheet`、`KindHTML`、`KindUnknown` | `exports.go` |
| 可观测性 | `Warning`、`CompressionStep`、`Observer`、`NoopObserver` | `exports.go` |
| 内容路由 | `ContentRouter` | `exports.go` |
| 标签保护 | `ProtectedContent`、`TagProtector` | `exports.go` |
| Tokenizer | `TokenizerBackend`、`TokenizerConfig`、`Tokenizer`、`FallbackTokenizer`、`TokenizerFallback`、`TokenizerTiktoken`、`TokenizerHF`、`ErrTokenizerNotImplemented` | `exports.go` |

### 6.2 兼容数据模型

| 数据模型类别 | 符号 | 目标文件 |
| --- | --- | --- |
| 历史配置 | `CompressionConfig`、`CodeConfig`、`TextConfig`、`SmartCrushConfig` | `compat.go` |
| 历史压缩接口 | `Compressor`、`CompressorFunc`、`CompressorRegistry` | `compat.go` |
| 历史兼容函数 | `SmartCrushJSON`、`SmartCrushJSONWithSteps`、`CompressCode`、`CompressText` | `compat.go` |
| 历史 helper | `errorTokenizer`、`lineIndent` | `compat.go` |

### 6.3 Pipeline 数据模型

Pipeline 权威模型归属 `internal/engine/pipeline.go`：

1. `Pipeline`
2. `Pipeline.Run`
3. Transform 实现
4. warning 生成
5. token 统计
6. HTML / JSON / code / text 等 transform 细节

根包只保留必要的公开类型别名或 facade，避免两套实现并存。

## 7. 依赖关系与执行顺序

```text
T0 建立符号与验证基线
  ├─ T1 合并 shim re-export 到 exports.go
  ├─ T2 迁移兼容内容到 compat.go
  └─ T3 精简 headroom.go Pipeline 重复实现
        ↓
T4 处理根包测试私有符号依赖
        ↓
T5 删除 8 个旧 shim 文件并确认 build-tag 桩不变
        ↓
T6 执行全量回归验证
        ↓
T7 执行外部 import smoke
```

执行约束：

1. 先建立符号基线，避免遗漏公开 API。
2. 先补齐 `exports.go` / `compat.go`，再删除旧 shim。
3. 先处理测试可见性，再大幅精简 `headroom.go`。
4. build-tag 验证必须在旧 shim 删除后执行。
5. 外部 import smoke 必须最后执行。

## 8. 验证矩阵

| 验证维度 | 命令 | 验收结果 |
| --- | --- | --- |
| 编译 | `go build ./...` | 通过 |
| 全量测试 | `go test ./...` | 通过 |
| Race 回归 | `go test -race -count=1 ./...` | 通过 |
| 静态检查 | `go vet ./...` | 通过 |
| 覆盖率 | `go test -cover ./...` | 通过，覆盖率不下降；按 Spec 目标关注根包 ≥73%、proxy ≥91%。 |
| HF build-tag | `go test -run '^$' -tags tokenizer_hf ./...` | 通过 |
| Tiktoken build-tag | `go test -run '^$' -tags tokenizer_tiktoken ./...` | 通过 |
| 根目录文件数量 | `python3 - <<'PY' ...` | 根目录非测试 `.go` ≤5；根目录总 `.go` ≤14。 |
| 外部 import | 临时 module + `replace` + `go test ./...` | 通过，覆盖核心 API 与 shim re-export。 |
| CLI smoke | `go run ./cmd/headroom --help` 或 `go run ./cmd/headroom version` | 通过。 |
| proxy smoke | `go run ./cmd/headroom proxy --port=18787` + `curl localhost:18787/healthz` | healthz 正常。 |

根目录文件数量检查命令：

```bash
python3 - <<'PY'
from pathlib import Path
root = Path('.')
go_files = sorted(p.name for p in root.glob('*.go'))
non_test = [f for f in go_files if not f.endswith('_test.go')]
print('root .go:', len(go_files), go_files)
print('root non-test .go:', len(non_test), non_test)
assert len(non_test) <= 5
assert len(go_files) <= 14
PY
```

外部 import smoke 命令：

```bash
tmp="$(mktemp -d)" && \
cd "$tmp" && \
go mod init external-headroom-smoke && \
go mod edit -replace github.com/superops-team/headroom-go=/home/wanglichao.superops/workspace/superops-team/headroom-go && \
go get github.com/superops-team/headroom-go && \
cat > smoke_test.go <<'EOF'
package smoke

import (
    "testing"

    headroom "github.com/superops-team/headroom-go"
)

func TestPublicAPISmoke(t *testing.T) {
    opts := headroom.DefaultOptions()
    _, _ = headroom.Compress([]headroom.Message{{Role: "user", Content: "hello hello hello"}}, opts)
    _, _ = headroom.CompressString("hello hello hello", opts)

    _ = headroom.NewCompressionEngine
    _ = headroom.DefaultCompressionPolicy
    _ = headroom.NewTransformError
    _ = headroom.NewDefaultPipeline

    _ = headroom.KindJSON
    _ = headroom.NoopObserver{}
    _ = headroom.NewContentRouter()
    _ = headroom.NewTagProtector()
    _, _, _ = headroom.NewTokenizer(headroom.TokenizerConfig{Backend: headroom.TokenizerFallback, AllowFallback: true})
    _ = headroom.NewCacheAligner(headroom.CacheAlignerConfig{Enabled: true, Version: headroom.PrefixVersion})
    _ = headroom.NewCCR(headroom.CCRConfig{})
    _ = headroom.ErrTokenizerNotImplemented
}
EOF
go test ./...
```

## 9. 风险与缓解

| 风险 | 影响 | 缓解 |
| --- | --- | --- |
| re-export 从类型别名误改为新类型 | 破坏赋值兼容、方法集或外部编译 | 统一使用 `type X = internal.X`；外部 import smoke 验证。 |
| 私有测试符号被删除 | 现有测试失败 | T4 专门处理测试可见性；保留 helper 或改为等价行为测试。 |
| Pipeline 行为漂移 | 压缩结果、warning、token 统计变化 | 不重写逻辑，只委托 `internal/engine` 权威实现；运行 E2E 与 pipeline 测试。 |
| build-tag 桩误合并 | 单 tag 编译失败 | 明确禁止合并；运行 `tokenizer_hf` / `tokenizer_tiktoken` tag 测试。 |
| 删除 shim 过早 | 中间状态缺符号，难定位 | 先创建 `exports.go` / `compat.go`，初步通过后再删除旧 shim。 |
| 覆盖率下降 | Spec 验收失败 | 执行 `go test -cover ./...`；测试调整不得弱化断言。 |
| 行为无意改变 | 违反“零功能变更” | 所有迁移函数体保持不变；以全量测试、race、外部 smoke 验证。 |

## 10. 交付清单

### 10.1 必须交付

1. `exports.go`：合并 8 个 shim 的 re-export。
2. `compat.go`：迁移兼容类型、函数和 helper。
3. 精简后的 `headroom.go`：保留核心公开 API，移除重复 Pipeline 实现。
4. 删除旧 shim 文件：
   - `cachealigner.go`
   - `ccr.go`
   - `ccrstore.go`
   - `content_kind.go`
   - `observability.go`
   - `router.go`
   - `tag_protector.go`
   - `tokenizer.go`
5. 保留 build-tag 桩：
   - `tokenizer_hf.go`
   - `tokenizer_tiktoken.go`
6. 必要测试适配：仅限保持原测试意图与行为覆盖。
7. 验证记录：完整验证矩阵命令结果。

### 10.2 可选交付

1. `doc.go`：提取 package 文档注释。
2. `exports.go` / `compat.go` 文件级注释。

## 11. 最终验收标准

1. 根目录非测试 `.go` 文件从 12 个降至 ≤5 个。
2. 根目录总 `.go` 文件从 21 个降至 ≤14 个。
3. 8 个旧 shim 文件已删除。
4. `exports.go` 和 `compat.go` 存在且职责清晰。
5. `headroom.go` 不再包含与 `internal/engine/pipeline.go` 重复的大段 Pipeline 实现。
6. `tokenizer_hf.go` / `tokenizer_tiktoken.go` 保持独立且 build tag 不变。
7. 公开 API 签名不变。
8. 行为逻辑不变。
9. 所有现有测试通过。
10. build-tag 编译矩阵通过。
11. 外部 import smoke 通过。
12. CLI / proxy smoke 通过。
