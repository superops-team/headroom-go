# cleanup-root-layout Step 2 OpenSpec Tasks

## 1. 任务拆解原则

本任务集遵循以下原则：

1. 单一职责：每个 Task 只处理一个明确变更面。
2. 可独立开发：每个 Task 有清晰输入和输出。
3. 可测试：每个 Task 至少提供一个验证命令。
4. 可验收：每个 Task 有明确 Acceptance Criteria。
5. 颗粒度 ≤ 1 天。
6. 纯文件重组，零功能变更。
7. 不改变任何公开 API 签名。
8. 不改变任何行为逻辑。
9. Build-tag 桩 `tokenizer_hf.go` / `tokenizer_tiktoken.go` 不合并。

## 2. 推荐执行顺序

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
T7 执行外部 import / CLI / proxy smoke
```

并行建议：

- `T1` 与 `T2` 在 `T0` 完成后可并行开发。
- `T3` 与 `T1` / `T2` 可部分并行分析，但落地应避免同改 `headroom.go` 冲突。
- `T4` 必须等待 `T2` 和 `T3`。
- `T5` 必须等待 `T1`、`T2`、`T3`、`T4`。
- `T6`、`T7` 必须最后串行执行。

## 3. 任务总览

| Task | 标题 | 单一职责 | 依赖 | 预计工时 | 可独立测试 |
| --- | --- | --- | --- | --- | --- |
| T0 | 建立符号与验证基线 | 锁定 API、shim、compat、Pipeline、测试依赖和初始验证状态 | 无 | 0.5 天 | 是 |
| T1 | 合并 shim re-export 到 `exports.go` | 将 8 个 shim 的 re-export 汇总到一个文件 | T0 | 0.5 天 | 是 |
| T2 | 迁移兼容内容到 `compat.go` | 从 `headroom.go` 迁移历史兼容类型、函数、helper | T0 | 0.5 天 | 是 |
| T3 | 精简 `headroom.go` Pipeline 重复实现 | 让 Pipeline 权威实现收敛到 `internal/engine` | T0 | 1 天 | 是 |
| T4 | 处理根包测试私有符号依赖 | 保证现有测试不因重组失效且不弱化断言 | T2、T3 | 0.5 天 | 是 |
| T5 | 删除 8 个旧 shim 文件 | 删除旧 shim 并确认 build-tag 桩保持独立 | T1、T2、T3、T4 | 0.25 天 | 是 |
| T6 | 执行全量回归验证 | gofmt/build/test/race/vet/coverage/build-tag 全通过 | T5 | 0.5 天 | 是 |
| T7 | 执行外部 import / CLI / proxy smoke | 验证外部用户入口与命令入口兼容 | T6 | 0.25 天 | 是 |

---

## T0：建立符号与验证基线

### 职责

锁定本次文件重组的基线，确认公开 API、shim 符号、compat 符号、Pipeline 重复实现、build-tag 文件和测试私有依赖，避免后续迁移遗漏或误删。

### 输入

1. `openspec/specs/cleanup-root-layout.md`
2. 根目录当前 21 个 `.go` 文件。
3. `headroom.go`
4. `internal/engine/pipeline.go`
5. 根包测试文件。

### 输出

1. shim 符号清单。
2. compat 符号清单。
3. `headroom.go` 保留公共入口清单。
4. Pipeline 去重清单。
5. build-tag 文件保留清单。
6. 测试私有符号依赖清单。
7. 初始验证结果。

### 依赖

无。

### 执行步骤

1. 阅读 Spec 关键位置：
   - `openspec/specs/cleanup-root-layout.md:13`
   - `openspec/specs/cleanup-root-layout.md:17`
   - `openspec/specs/cleanup-root-layout.md:25-31`
   - `openspec/specs/cleanup-root-layout.md:92-133`
   - `openspec/specs/cleanup-root-layout.md:216-229`
2. 确认 8 个 shim 文件的符号：
   - `cachealigner.go:10` / `cachealigner.go:26` / `cachealigner.go:30`
   - `ccr.go:13` / `ccr.go:30` / `ccr.go:34` / `ccr.go:38`
   - `ccrstore.go:5`
   - `content_kind.go:12` / `content_kind.go:27`
   - `observability.go:14` / `observability.go:29` / `observability.go:42` / `observability.go:46`
   - `router.go:15` / `router.go:18`
   - `tag_protector.go:10` / `tag_protector.go:26` / `tag_protector.go:29`
   - `tokenizer.go:11` / `tokenizer.go:18` / `tokenizer.go:26` / `tokenizer.go:30` / `tokenizer.go:33` / `tokenizer.go:41` / `tokenizer.go:46` / `tokenizer.go:51`
3. 确认 `headroom.go` 公共入口：`headroom.go:55`、`headroom.go:73`、`headroom.go:84`、`headroom.go:88`、`headroom.go:91`、`headroom.go:94`、`headroom.go:97`、`headroom.go:115`、`headroom.go:127`、`headroom.go:137`、`headroom.go:148`、`headroom.go:156`、`headroom.go:181`、`headroom.go:202`。
4. 确认 compat 迁移范围：`headroom.go:531`、`headroom.go:541-547`、`headroom.go:549`、`headroom.go:553`、`headroom.go:557`、`headroom.go:561`、`headroom.go:565`、`headroom.go:579`、`headroom.go:583`、`headroom.go:587`。
5. 确认 Pipeline 去重范围：`headroom.go:99`、`headroom.go:354`、`headroom.go:424`、`headroom.go:438`、`headroom.go:451`、`headroom.go:460`、`headroom.go:469`、`headroom.go:482`、`headroom.go:484`、`headroom.go:508`、`headroom.go:515`、`headroom.go:523`、`headroom.go:527`，对应 `internal/engine/pipeline.go:17`、`internal/engine/pipeline.go:22`、`internal/engine/pipeline.go:26`、`internal/engine/pipeline.go:96`、`internal/engine/pipeline.go:175`、`internal/engine/pipeline.go:184`、`internal/engine/pipeline.go:193`、`internal/engine/pipeline.go:206`、`internal/engine/pipeline.go:208`、`internal/engine/pipeline.go:232`、`internal/engine/pipeline.go:239`。
6. 确认 build-tag 文件：`tokenizer_hf.go:1` 与 `tokenizer_tiktoken.go:1`。
7. 确认测试私有依赖：
   - `spec_b_e2e_test.go:257-347`
   - `spec_a_e2e_test.go:184-207`
   - `spec_b_e2e_test.go:208-224`
   - `spec_d_coverage_test.go:110-123`
8. 执行初始验证并记录结果。

### Acceptance Criteria

1. 已形成完整符号基线，无待迁移公开符号遗漏。
2. 已明确每个符号的目标文件归属。
3. 已明确 build-tag 桩不参与合并。
4. 已明确测试私有依赖的处理策略。
5. 已记录初始测试状态。

### 验证命令

```bash
go build ./...
go test ./...
```

### 预计工时

0.5 天。

---

## T1：合并 shim re-export 到 `exports.go`

### 职责

创建或更新 `exports.go`，将 8 个 shim 文件中的类型别名、常量、变量和构造函数集中到一个文件。

### 输入

1. T0 的 shim 符号基线。
2. 8 个旧 shim 文件。
3. 对应 internal 包现有类型和构造函数。

### 输出

1. `exports.go`
2. 可编译的根包 re-export 定义。

### 依赖

T0。

### 执行步骤

1. 新建 `exports.go`，package 保持 `headroom`。
2. 添加文件职责注释：该文件只承载 public re-export。
3. 使用 `type X = internal.X` 形式迁移所有类型：
   - `CacheAlignerConfig`、`CacheAligner`
   - `CCRConfig`、`CCR`、`CCRStore`
   - `ContentKind`
   - `Warning`、`CompressionStep`、`Observer`、`NoopObserver`
   - `ContentRouter`
   - `ProtectedContent`、`TagProtector`
   - `TokenizerBackend`、`TokenizerConfig`、`Tokenizer`、`FallbackTokenizer`
4. 迁移 `Kind*` 常量。
5. 迁移 `Tokenizer*` 常量。
6. 迁移 `ErrTokenizerNotImplemented`。
7. 迁移构造函数：`NewCacheAligner`、`NewCCR`、`NewContentRouter`、`NewTagProtector`、`NewTokenizer`、`ResolveTokenizer`。
8. 迁移或保留 `getPackageCCR`，确保 `spec_b_e2e_test.go:257-347` 相关测试仍可访问。
9. 暂不删除旧 shim 文件，待 T5 统一删除。
10. 执行 gofmt 与初步验证。

### Acceptance Criteria

1. `exports.go` 覆盖全部 8 个 shim 文件目标符号。
2. 未新增非 Spec 要求的公开 API。
3. 类型 re-export 使用类型别名而不是新类型定义。
4. `getPackageCCR` 的根包测试可见性保持或已有明确等价测试方案。
5. `tokenizer_hf.go` / `tokenizer_tiktoken.go` 未被修改。
6. 除旧 shim 尚未删除导致的重复定义外，无缺失符号。

### 验证命令

```bash
gofmt -w exports.go
go test ./...
```

如因为旧 shim 尚未删除出现重复定义，应记录重复符号，并在 T5 删除旧 shim 后重新执行验证。

### 预计工时

0.5 天。

---

## T2：迁移兼容内容到 `compat.go`

### 职责

将 `headroom.go` 中历史兼容类型、函数和 helper 迁移到 `compat.go`，降低 `headroom.go` 职责复杂度。

### 输入

1. T0 的 compat 符号基线。
2. `headroom.go`

### 输出

1. `compat.go`
2. 移除兼容内容后的 `headroom.go`

### 依赖

T0。

### 执行步骤

1. 新建 `compat.go`，package 保持 `headroom`。
2. 添加文件职责注释：该文件只承载历史兼容 API。
3. 从 `headroom.go` 移动 `errorTokenizer`。
4. 从 `headroom.go` 移动兼容配置类型：`CompressionConfig`、`CodeConfig`、`TextConfig`、`SmartCrushConfig`。
5. 从 `headroom.go` 移动兼容接口/类型：`Compressor`、`CompressorFunc`、`CompressorRegistry`。
6. 从 `headroom.go` 移动兼容函数：`SmartCrushJSON`、`SmartCrushJSONWithSteps`、`CompressCode`、`CompressText`。
7. 从 `headroom.go` 移动 `lineIndent`，保持 `package headroom` 内测试可见。
8. 从 `headroom.go` 移动 registry 构造函数：`NewCompressorFunc`、`NewCompressorRegistry`、`DefaultCompressorRegistry`。
9. 保持函数体逻辑不变，不改错误处理、不改返回值、不改默认参数。
10. 执行 gofmt 与针对兼容函数的测试。

### Acceptance Criteria

1. `compat.go` 包含全部兼容符号。
2. `headroom.go` 不再承载已迁移的 compat 内容。
3. 所有公开兼容函数签名不变。
4. 迁移前后行为逻辑不变。
5. `lineIndent` 测试仍通过，或已改为等价行为断言。
6. `api_compat_test.go` 中兼容 API 断言通过。

### 验证命令

```bash
gofmt -w compat.go headroom.go
go test ./... -run 'SmartCrush|CompressCode|CompressText|lineIndent|Compat'
go test ./...
```

### 预计工时

0.5 天。

---

## T3：精简 `headroom.go` Pipeline 重复实现

### 职责

消除 `headroom.go` 与 `internal/engine/pipeline.go` 的重复 Pipeline 实现，使 Pipeline 行为权威收敛到 `internal/engine`。

### 输入

1. T0 的 Pipeline 去重清单。
2. `headroom.go`
3. `internal/engine/pipeline.go`
4. Pipeline 相关测试。

### 输出

1. 精简后的 `headroom.go`
2. 根包 `NewDefaultPipeline` 的兼容 facade 或类型别名方案。
3. 移除重复 Pipeline 行为实现后的代码结构。

### 依赖

T0。

### 执行步骤

1. 确认 `NewDefaultPipeline` 作为根包公开 API 必须保留。
2. 确认 `Pipeline` 类型对外兼容策略：
   - 优先使用 `type Pipeline = eng.Pipeline`；
   - 如无法别名，保留极薄 wrapper，但 `Run` 行为必须委托 `internal/engine`。
3. 将 `NewDefaultPipeline` 改为委托 `internal/engine.NewDefaultPipeline` 或等价 facade。
4. 移除或委托以下重复实现：
   - `Pipeline.Run`
   - `countTokensForPipeline`
   - `appliesTo`
   - `legacyTextTransform`
   - `legacyCodeTransform`
   - `jsonMinifierTransform`
   - `jsonOffloadTransform`
   - `NewJSONOffloadTransform`
   - `warningFromTransformError`
   - `htmlCleanTransform`
   - `removeHTMLBlock`
   - `removeHTMLComments`
5. 确保 `Compress` 和 `CompressString` 仍按现有路径委托 `NewCompressionEngine` 与 internal engine。
6. 不改 Pipeline transform 逻辑，不改变运行顺序、warning 语义或 token 统计。
7. 执行 gofmt 与 Pipeline 相关测试。

### Acceptance Criteria

1. `headroom.go` 不再保留与 `internal/engine/pipeline.go` 重复的大段 Pipeline 行为逻辑。
2. Pipeline 权威实现来自 `internal/engine/pipeline.go`。
3. `NewDefaultPipeline` 调用方式不变。
4. `Compress` / `CompressString` 行为不变。
5. Pipeline 相关现有测试通过，或已在 T4 中改为等价行为测试。
6. 未复制新的 Pipeline 实现到根包。

### 验证命令

```bash
gofmt -w headroom.go
go test ./... -run 'Pipeline|Compress|CompressString|SpecA|SpecB'
go test ./...
```

### 预计工时

1 天。

---

## T4：处理根包测试私有符号依赖

### 职责

保证现有测试在文件重组和 Pipeline 去重后继续通过，且测试断言不被弱化。

### 输入

1. T0 的测试私有依赖清单。
2. T2 后的 `compat.go`。
3. T3 后的 `headroom.go`。
4. `spec_a_e2e_test.go`
5. `spec_b_e2e_test.go`
6. `spec_d_coverage_test.go`

### 输出

1. 继续通过的现有测试；或
2. 经过等价行为适配的测试。

### 依赖

T2、T3。

### 执行步骤

1. 处理 `spec_b_e2e_test.go:257-347`：
   - 若 `legacySkipMessage`、`routeAndCompressLegacy`、`postProcessLegacyCompression`、`compressLegacy`、`getPackageCCR` 仍保留在根包，则测试可保持。
   - 若符号被移除或委托，改为公开 API 或 observable behavior 断言。
2. 处理 `spec_a_e2e_test.go:184-207` 与 `spec_b_e2e_test.go:208-224`：
   - 避免继续构造已被移除的根包私有 Pipeline internals。
   - 优先通过 `NewDefaultPipeline`、`Compress` 或 internal engine 的公开测试入口验证 JSON minify / offload 行为。
3. 处理 `spec_d_coverage_test.go:110-123`：
   - 若 `lineIndent` 保持在 `compat.go`，测试可保持。
   - 若不保留私有 helper，改为 `CompressCode` / `CompressText` 的等价行为断言。
4. 不允许将测试改成仅编译通过。
5. 不允许复制旧 Pipeline transform 实现到测试中。
6. 执行相关测试和全量测试。

### Acceptance Criteria

1. 所有受影响测试通过。
2. 测试不再依赖已删除文件。
3. 如修改测试，测试仍验证原行为。
4. 未弱化 E2E / coverage 测试断言。
5. 未为了测试保留第二套 Pipeline 行为实现。

### 验证命令

```bash
gofmt -w spec_a_e2e_test.go spec_b_e2e_test.go spec_d_coverage_test.go
go test ./... -run 'SpecA|SpecB|SpecD|Pipeline|CCR|lineIndent|SmartCrush'
go test ./...
```

### 预计工时

0.5 天。

---

## T5：删除 8 个旧 shim 文件并确认 build-tag 桩不变

### 职责

删除已被 `exports.go` 覆盖的旧 shim 文件，并确认两个 tokenizer build-tag 桩仍保持独立。

### 输入

1. T1 的 `exports.go`。
2. T2 的 `compat.go`。
3. T3 的 `headroom.go`。
4. T4 后的测试状态。
5. 8 个待删除 shim 文件。

### 输出

1. 删除以下文件：
   - `cachealigner.go`
   - `ccr.go`
   - `ccrstore.go`
   - `content_kind.go`
   - `observability.go`
   - `router.go`
   - `tag_protector.go`
   - `tokenizer.go`
2. 保留以下文件：
   - `tokenizer_hf.go`
   - `tokenizer_tiktoken.go`

### 依赖

T1、T2、T3、T4。

### 执行步骤

1. 确认 `exports.go` 已包含 8 个 shim 文件所有 re-export。
2. 确认 `compat.go` 已包含兼容内容。
3. 删除 8 个旧 shim 文件。
4. 确认 `tokenizer_hf.go` 存在且仍包含 `//go:build tokenizer_hf`。
5. 确认 `tokenizer_tiktoken.go` 存在且仍包含 `//go:build tokenizer_tiktoken`。
6. 执行编译和测试。

### Acceptance Criteria

1. 8 个旧 shim 文件不存在。
2. `exports.go` 提供等价 re-export。
3. `tokenizer_hf.go` / `tokenizer_tiktoken.go` 仍存在。
4. 两个 build-tag 桩未被合并、未被删除、未被改 tag。
5. 删除后无重复定义、无缺失符号。

### 验证命令

```bash
test ! -e cachealigner.go
test ! -e ccr.go
test ! -e ccrstore.go
test ! -e content_kind.go
test ! -e observability.go
test ! -e router.go
test ! -e tag_protector.go
test ! -e tokenizer.go

test -f tokenizer_hf.go
test -f tokenizer_tiktoken.go

go build ./...
go test ./...
```

### 预计工时

0.25 天。

---

## T6：执行全量回归验证

### 职责

对完成后的文件重组执行完整验证矩阵，确保编译、测试、race、vet、coverage 和 build-tag 编译全部通过。

### 输入

T5 后的完整代码状态。

### 输出

1. 完整验证结果。
2. 若失败，记录失败命令、失败原因和修复建议。

### 依赖

T5。

### 执行步骤

1. 执行 gofmt。
2. 执行 `go build ./...`。
3. 执行全量测试。
4. 执行 race 测试。
5. 执行 vet。
6. 执行 coverage。
7. 执行 build-tag 矩阵。
8. 执行根目录文件数量检查。

### Acceptance Criteria

1. `go build ./...` 通过。
2. `go test ./...` 通过。
3. `go test -race -count=1 ./...` 通过。
4. `go vet ./...` 通过。
5. `go test -cover ./...` 通过且覆盖率不下降。
6. `go test -run '^$' -tags tokenizer_hf ./...` 通过。
7. `go test -run '^$' -tags tokenizer_tiktoken ./...` 通过。
8. 根目录非测试 `.go` 文件 ≤5。
9. 根目录总 `.go` 文件 ≤14。

### 验证命令

```bash
gofmt -w exports.go compat.go headroom.go

go build ./...
go test ./...
go test -race -count=1 ./...
go vet ./...
go test -cover ./...
go test -run '^$' -tags tokenizer_hf ./...
go test -run '^$' -tags tokenizer_tiktoken ./...

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

若 T4 修改测试文件，也需包含：

```bash
gofmt -w spec_a_e2e_test.go spec_b_e2e_test.go spec_d_coverage_test.go
```

### 预计工时

0.5 天。

---

## T7：执行外部 import / CLI / proxy smoke

### 职责

验证外部用户通过根包 import 和命令入口使用时仍保持兼容。

### 输入

1. T6 验证通过后的代码状态。
2. 本地仓库路径 `/home/wanglichao.superops/workspace/superops-team/headroom-go`。

### 输出

1. 外部 import smoke 结果。
2. CLI smoke 结果。
3. proxy health smoke 结果。

### 依赖

T6。

### 执行步骤

1. 创建临时外部 Go module。
2. 使用 `replace` 指向本地仓库。
3. 编写 smoke test，覆盖：
   - `DefaultOptions`
   - `Compress`
   - `CompressString`
   - `NewCompressionEngine`
   - `DefaultCompressionPolicy`
   - `NewTransformError`
   - `NewDefaultPipeline`
   - `KindJSON`
   - `NoopObserver`
   - `NewContentRouter`
   - `NewTagProtector`
   - `NewTokenizer`
   - `ResolveTokenizer`
   - `NewCacheAligner`
   - `NewCCR`
   - `ErrTokenizerNotImplemented`
4. 执行外部 module `go test ./...`。
5. 执行 CLI smoke。
6. 执行 proxy health smoke。

### Acceptance Criteria

1. 外部 module 能成功 import 根包。
2. 核心公开 API 可编译和调用。
3. shim re-export API 可编译和调用。
4. CLI 命令可启动并输出帮助或版本。
5. proxy health endpoint 正常。

### 验证命令

外部 import smoke：

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

CLI smoke：

```bash
go run ./cmd/headroom --help
go run ./cmd/headroom version
```

proxy smoke：

```bash
go run ./cmd/headroom proxy --port=18787
curl localhost:18787/healthz
```

### 预计工时

0.25 天。

---

## 4. 总体验收标准

所有 Task 完成后，必须满足：

1. `exports.go` 存在并合并 8 个 shim 文件的 re-export。
2. `compat.go` 存在并承载历史兼容类型、函数、helper。
3. `headroom.go` 已精简，不再包含与 `internal/engine/pipeline.go` 重复的大段 Pipeline 实现。
4. 以下旧 shim 文件已删除：
   - `cachealigner.go`
   - `ccr.go`
   - `ccrstore.go`
   - `content_kind.go`
   - `observability.go`
   - `router.go`
   - `tag_protector.go`
   - `tokenizer.go`
5. `tokenizer_hf.go` / `tokenizer_tiktoken.go` 保持独立，build tag 不变。
6. 根目录非测试 `.go` 文件 ≤5。
7. 根目录总 `.go` 文件 ≤14。
8. 公开 API 签名不变。
9. 行为逻辑不变。
10. 所有现有测试通过。
11. build-tag 测试通过。
12. 外部 import smoke 通过。
13. CLI / proxy smoke 通过。

## 5. 执行注意事项

1. 不要先删除 shim 文件，应先补齐 `exports.go`。
2. 不要合并 `tokenizer_hf.go` / `tokenizer_tiktoken.go`。
3. 不要将 `type X = internal.X` 误写成 `type X internal.X`。
4. 不要复制新的 Pipeline 实现到根包。
5. 不要弱化测试断言。
6. 不要引入功能变更。
7. 处理私有测试依赖时，应先判断测试意图，再决定保留 helper 或改为行为测试。
8. 所有验证通过前，不应认为 Spec 已完成。
