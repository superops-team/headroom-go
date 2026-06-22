# cleanup-root-layout Spec 16 维度全方位合规分析报告

- 审查对象：`openspec/specs/cleanup-root-layout.md`
- 代码基线：当前工作区 `/home/wanglichao.superops/workspace/superops-team/headroom-go`
- 审查时间：2026-06-22
- 审查结论：**不通过**
- 判定依据：存在 **2 个 P0**；按标准“有 P0 = 不通过”。

> 说明：用户要求“加载 spec-review 技能”，但当前可用技能列表中没有 `spec-review`。本报告按该请求语义完成 Spec review，并已落盘到本文件。

---

## 0. 核对摘要

### 0.1 与代码基线一致的内容

| 核对项 | Spec 描述 | 实际核对 | 结论 |
|---|---:|---:|---|
| 根目录 `.go` 文件数 | 21 个，全部 `package headroom`，见 `openspec/specs/cleanup-root-layout.md:13` | 21 个；根包文件均为 `package headroom`，例如 `headroom.go:32`、`cachealigner.go:1` | ✅ 一致 |
| `headroom.go` 行数 | 589 行，见 `openspec/specs/cleanup-root-layout.md:18`、`openspec/specs/cleanup-root-layout.md:138` | 589 行；最后一个导出函数在 `headroom.go:587`，文件结束于 `headroom.go:589` | ✅ 一致 |
| `internal/engine/` 文件 | 用户基线声明已有 `engine.go`、`pipeline.go`、`policy.go`，无 `transforms.go` | 实际为 `engine.go`、`pipeline.go`、`policy.go`，另有测试文件；无 `transforms.go` | ✅ 一致 |
| internal 子包已存在 | Spec 目标结构列出 `internal/types`、`internal/compressors`、`internal/engine` 等，见 `openspec/specs/cleanup-root-layout.md:54` | 对应目录均存在；例如 `internal/engine/engine.go:1`、`internal/compressors/transforms.go:1` | ✅ 一致 |
| build-tag 桩数量 | 2 个：`tokenizer_hf.go`、`tokenizer_tiktoken.go`，见 `openspec/specs/cleanup-root-layout.md:19` | 当前两个根包桩分别在 `tokenizer_hf.go:1`、`tokenizer_tiktoken.go:1` | ✅ 一致 |

### 0.2 与当前代码冲突或已过期的内容

| 编号 | 冲突/过期内容 | 证据 | 影响 |
|---|---|---|---|
| C1 | “根目录 `.go` 文件从 21 → ≤8（仅公共 API + 测试）”不可达 | Spec 同时要求 9 个测试文件保留不变，见 `openspec/specs/cleanup-root-layout.md:43` 至 `openspec/specs/cleanup-root-layout.md:52`；即使非测试文件压到 4-5 个，根目录 `.go` 也会 ≥13 | **P0：验收标准自相矛盾，执行后无法满足** |
| C2 | Shim 数量描述自相矛盾：表头写 7 个，实际列 8 个 | `openspec/specs/cleanup-root-layout.md:17` 写“Shim 重导出 7”，但同一行列出 8 个文件；`openspec/specs/cleanup-root-layout.md:92` 又写“合并以下 7 个文件”，表格至 `openspec/specs/cleanup-root-layout.md:103` 实际 8 个 | P1：范围统计错误，容易导致执行漏删/漏并 |
| C3 | Phase 1 说“合并 build-tag 桩为 1 个 `tokenizer_build.go`”的示例会破坏单 tag 编译 | 当前两个 root 桩分别只在对应 tag 下声明函数：`tokenizer_hf.go:1`、`tokenizer_hf.go:8`；`tokenizer_tiktoken.go:1`、`tokenizer_tiktoken.go:8`。合并为 `tokenizer_hf || tokenizer_tiktoken` 后无法在单 tag 下同时引用另一个 internal stub | **P0：按 Spec 示例实现会导致 `go test -tags tokenizer_hf` 或 `go test -tags tokenizer_tiktoken` 编译失败** |
| C4 | “Pipeline 变换下沉到 `internal/engine/transforms.go`”描述已部分过期 | Pipeline 与 transform 代码已存在于 `internal/engine/pipeline.go:22`、`internal/engine/pipeline.go:175`、`internal/engine/pipeline.go:206`；根包 `headroom.go:354` 起也有重复实现 | P1：应改为“消除根包重复 Pipeline 实现，并视情况拆分 internal/engine/pipeline.go”，不是从零下沉 |
| C5 | `headroom.go` “只保留公共 API + Legacy 路径”与当前 internal engine 已有 legacy 路径重复 | internal engine 已有 legacy 实现在 `internal/engine/engine.go:91`；根包也有 `compressLegacy` 在 `headroom.go:225` | P1：继续保留根包 legacy 会保留重复实现，不符合架构收敛 |
| C6 | “创建 `exports.go`，合并 8 个 shim + 兼容函数”会生成过大混合职责文件 | 兼容函数位于 `headroom.go:541` 至 `headroom.go:589`；shim 文件总计 265 行，合并后再加兼容函数约 314 行 | P1：可维护性下降，建议分 `exports.go` 与 `compat.go` |
| C7 | 验收“140 个测试全部通过”与当前 `go test -json ./...` 统计口径不一致 | 当前实测 `go test -json ./...` 为 274 个 pass test events、9 个 package pass；无失败 | P1：验收应固定命令与统计口径，避免误判 |

---

## 1. 代码基线核对明细

### 1.1 根目录 `.go` 文件清单与行数

实测根目录 `.go` 文件 **21 个**：

| 文件 | 行数 | 分类建议 |
|---|---:|---|
| `api_compat_test.go` | 97 | 测试 |
| `benchmark_test.go` | 155 | 测试 |
| `cachealigner.go` | 32 | shim |
| `ccr.go` | 40 | shim |
| `ccrstore.go` | 5 | shim |
| `content_kind.go` | 38 | shim |
| `content_kind_test.go` | 33 | 测试 |
| `fixtures_test.go` | 61 | 测试 |
| `fuzz_test.go` | 43 | 测试 |
| `headroom.go` | 589 | 公共 API + legacy + pipeline facade/重复实现 + compat |
| `headroom_test.go` | 318 | 测试 |
| `observability.go` | 46 | shim |
| `router.go` | 20 | shim |
| `spec_a_e2e_test.go` | 278 | 测试 |
| `spec_b_e2e_test.go` | 394 | 测试 |
| `spec_d_coverage_test.go` | 361 | 测试 |
| `tag_protector.go` | 31 | shim |
| `tokenizer.go` | 53 | shim |
| `tokenizer_hf.go` | 10 | build-tag 桩 |
| `tokenizer_tiktoken.go` | 10 | build-tag 桩 |
| `version.go` | 20 | 版本 |

### 1.2 当前测试与 build-tag 验证

- `go test -json ./...`：274 个 test pass event，9 个 package pass，无失败。
- `go test -run '^$' -tags tokenizer_hf ./...`：通过。
- `go test -run '^$' -tags tokenizer_tiktoken ./...`：通过。
- 外部临时 module 编译了 `DefaultOptions`、`NewDefaultPipeline`、`Pipeline.Run`、`NewJSONOffloadTransform`、`CompressString`：通过。

---

## 2. 16 维度逐项审查

### D01 上下文逻辑连贯性

- 状态：⚠️ 条件不满足
- 问题：文件数、`headroom.go` 行数、internal 子包状态基本准确；但 shim 数量“7/8”自相矛盾，且“21 → ≤8（含测试）”与 9 个测试文件不变冲突。
- 证据：`openspec/specs/cleanup-root-layout.md:17`、`openspec/specs/cleanup-root-layout.md:43`、`openspec/specs/cleanup-root-layout.md:52`、`openspec/specs/cleanup-root-layout.md:234`。
- 建议：把验收拆成“根目录非测试 `.go` 文件数 ≤5”或“根目录非测试公共 API 文件 ≤4”；不要用包含测试文件的总 `.go` 数作为 ≤8 指标。

### D02 向下兼容隐患

- 状态：⚠️ 有风险
- 问题：合并 shim 本身不破坏 API，但移动/删除 `Pipeline`、`NewDefaultPipeline`、`NewJSONOffloadTransform`、compat 函数时容易改变导出符号或方法签名。
- 证据：`headroom.go:99` 定义公开 `Pipeline`；`headroom.go:156` 定义 `NewDefaultPipeline`；`headroom.go:354` 定义 `Pipeline.Run`；`headroom.go:484` 定义 `NewJSONOffloadTransform`；compat 函数在 `headroom.go:549` 至 `headroom.go:587`。
- 建议：新增外部兼容 fixture：在临时 module 中 import `github.com/superops-team/headroom-go`，编译所有导出符号；执行 `go test ./...` 前后对 `go doc` 或导出符号清单做 diff。

### D03 最小化实现原则

- 状态：❌ 不满足
- 问题：Spec 要新增 `internal/engine/transforms.go`，但 transform 已在 `internal/engine/pipeline.go` 中存在；从根包“下沉”不应再创造新的架构层级作为硬要求。
- 证据：`internal/engine/pipeline.go:22` 已有 `NewDefaultPipeline`；transform 从 `internal/engine/pipeline.go:175` 开始；根包重复实现从 `headroom.go:354` 开始。
- 建议：Phase 2 目标改为“删除根包重复 Pipeline 实现，根包仅保留 facade/type alias”；是否拆 `internal/engine/pipeline.go` 到 `transforms.go` 应作为可选重排，不作为验收前提。

### D04 风险点预判

- 状态：⚠️ 不完整
- 问题：Spec 只列 git blame、Pipeline 测试、build-tag、exports 长度；未覆盖验收指标不可达、单 tag 编译、导出 API、root/internal 重复实现、测试访问私有函数等风险。
- 证据：风险表见 `openspec/specs/cleanup-root-layout.md:246` 至 `openspec/specs/cleanup-root-layout.md:254`；root 测试直接访问私有 helper，如 `spec_b_e2e_test.go:331` 调用 `compressLegacy`。
- 建议：增加“build tag 矩阵”“导出符号 diff”“根包私有测试依赖迁移策略”“文件数验收口径”四类风险。

### D05 过度设计审查

- 状态：⚠️ 有过度合并倾向
- 问题：把 8 个 shim + compat 函数全部塞进 `exports.go` 会形成 >300 行混合职责文件；从“碎片化”变成“巨型导出仓库”。
- 证据：shim 文件分别为 `cachealigner.go:1`、`ccr.go:1`、`tokenizer.go:1` 等；compat 函数在 `headroom.go:541` 起。
- 建议：采用 `exports.go`（类型/常量 re-export）+ `compat.go`（兼容函数）两文件；如强制“1 文件”，至少按 domain 分区并保留 godoc。

### D06 架构一致性

- 状态：⚠️ 方向正确但方案需改
- 问题：当前 `internal/engine` 已有完整 engine/pipeline/policy；根包仍保留重复 pipeline 与 legacy 逻辑。Spec 若继续让 `headroom.go` 保留 legacy，会保留双实现。
- 证据：internal legacy 在 `internal/engine/engine.go:91`；根包 legacy 在 `headroom.go:225`；internal pipeline 在 `internal/engine/pipeline.go:26`；root pipeline 在 `headroom.go:354`。
- 建议：根包 `Compress`/`CompressString`/`NewCompressionEngine` 只委托 internal engine；root 私有 legacy helpers 若仅为测试覆盖，应迁移测试到 internal/engine 或改为外部行为测试。

### D07 文件数验收可达性

- 状态：❌ 不可达
- 问题：根目录 9 个 `*_test.go` 按 Spec 不迁移；因此总 `.go` 文件不可能 ≤8。
- 证据：测试文件列表在 `openspec/specs/cleanup-root-layout.md:43` 至 `openspec/specs/cleanup-root-layout.md:52`；验收在 `openspec/specs/cleanup-root-layout.md:234`。
- 建议：把验收改为：根目录非测试 `.go` 文件从 12 → ≤5；根目录总 `.go` 文件从 21 → ≤14。

### D08 Build-tag 语义正确性

- 状态：❌ 不通过
- 问题：Spec 示例的单文件 build constraint `tokenizer_hf || tokenizer_tiktoken` 无法在单 tag 下安全同时暴露两个 stub 函数；当前设计每个文件只引用对应 internal stub，语义正确。
- 证据：`tokenizer_hf.go:1`、`tokenizer_hf.go:8`；`tokenizer_tiktoken.go:1`、`tokenizer_tiktoken.go:8`；internal 对应文件也分别受 tag 控制：`internal/tokenizer/huggingface.go:1`、`internal/tokenizer/tiktoken.go:1`。
- 建议：不要合并 build-tag 桩；或保留两个 tiny 文件；若必须合并，需要多文件或生成方案，不能单文件 OR tag。

### D09 公共 API / Godoc 完整性

- 状态：⚠️ 需补强
- 问题：`doc.go` 符合 Go 惯例，但不是必要变更；拆 package comment 时如果 `headroom.go` 留下重复或漏掉注释，会影响 Godoc。
- 证据：现有 package comment 在 `headroom.go:1` 至 `headroom.go:31`。
- 建议：`doc.go` 可选；若创建，`headroom.go` 顶部应只保留 `package headroom`，避免重复 package doc。验收加 `go doc github.com/superops-team/headroom-go` 快照。

### D10 测试计划充分性

- 状态：⚠️ 不完整
- 问题：`go test -race ./...`、`go vet ./...` 是必要但不足；没有 build-tag 矩阵、外部 import 编译、CLI/proxy smoke 的具体命令。
- 证据：Phase 验证见 `openspec/specs/cleanup-root-layout.md:189`、`openspec/specs/cleanup-root-layout.md:197`、验收见 `openspec/specs/cleanup-root-layout.md:235` 至 `openspec/specs/cleanup-root-layout.md:241`。
- 建议：明确命令：`go test -race -count=1 ./...`、`go test -run '^$' -tags tokenizer_hf ./...`、`go test -run '^$' -tags tokenizer_tiktoken ./...`、外部 temp module compile、CLI/proxy smoke。

### D11 覆盖率验收可操作性

- 状态：⚠️ 口径不足
- 问题：要求“覆盖率不下降（根包 ≥73%，proxy ≥91%）”，但没有说明当前覆盖率命令、是否包含 build tags、是否 race、是否统计 statement coverage。
- 证据：验收写在 `openspec/specs/cleanup-root-layout.md:238`。
- 建议：固定为 `go test -coverprofile=coverage.out ./...` 并说明比较对象；或在纯文件搬迁阶段只要求不删除测试与关键行为 pass。

### D12 执行步骤原子性

- 状态：⚠️ 可改进
- 问题：Phase 1 同时创建 `exports.go`、合并 build-tag、删 10 个文件；失败定位困难。
- 证据：Phase 1 在 `openspec/specs/cleanup-root-layout.md:181` 至 `openspec/specs/cleanup-root-layout.md:189`。
- 建议：拆成 4 个可回滚原子提交：doc/comment、shim 合并、compat 拆出、可选 build-tag 保留/不动；每步都跑 `go test ./...`。

### D13 文件合并对 blame/审阅的影响

- 状态：✅ 可接受但需方法约束
- 问题：Spec 识别了 git blame 混乱，但缓解只写“纯文件移动+合并”，不足以降低审阅噪声。
- 证据：风险表见 `openspec/specs/cleanup-root-layout.md:250`。
- 建议：先“复制合并不改逻辑”，后“删除旧文件”，避免同一 diff 混入格式化和逻辑修改；必要时保留 `git mv` 风格路径。

### D14 性能与并发语义

- 状态：✅ 基本无新增性能风险
- 问题：文件搬迁理论上不改变运行时；但 CCR 全局 store 与 background GC 仍需确保不因 facade 改动产生多实例。
- 证据：root `getPackageCCR` 委托 internal：`ccr.go:38` 至 `ccr.go:40`；internal singleton 在 `internal/engine/engine.go:17` 至 `internal/engine/engine.go:30`。
- 建议：保留一个 package-level CCR singleton；外部兼容测试应覆盖 reversible retrieve。

### D15 可维护性与可读性

- 状态：⚠️ 目标正确，局部方案需修订
- 问题：减少根包平铺有价值；但将所有 shim 放到一个 `exports.go` 会降低局部可读性，build-tag tiny file 保留反而更清晰。
- 证据：当前 shim 文件很短，如 `ccrstore.go:5`、`router.go:17` 至 `router.go:20`。
- 建议：以“非测试根包文件 ≤6、职责清晰”为目标，不追求极限文件数。

### D16 发布/文档变更一致性

- 状态：⚠️ 范围需明确
- 问题：验收要求 CHANGELOG 更新，但实施步骤没有对应 Phase；纯内部重构是否需要用户可见 changelog 也未说明。
- 证据：验收项在 `openspec/specs/cleanup-root-layout.md:242`，Phase 1-3 未列该任务。
- 建议：若目标版本 v0.6.0，增加 Phase 4 文档/CHANGELOG；若只是内部重构，CHANGELOG 可作为可选 release checklist。

---

## 3. P0 / P1 问题清单

### P0（必须修复，否则 Spec 不可执行/执行即破坏）

1. **P0-1：根目录 `.go` 文件 ≤8 的验收标准不可达**
   - 证据：Spec 要保留 9 个测试文件不变，见 `openspec/specs/cleanup-root-layout.md:43` 至 `openspec/specs/cleanup-root-layout.md:52`，但验收要求总 `.go` ≤8，见 `openspec/specs/cleanup-root-layout.md:234`。
   - 后果：实现者即使正确合并所有非测试文件，也无法通过验收。
   - 修复：改为“根目录非测试 `.go` 文件 ≤5”或“根目录总 `.go` 文件 ≤14”。

2. **P0-2：按 Spec 合并 build-tag 桩会破坏单 tag 编译**
   - 证据：当前两个函数分别在各自 tag 文件中：`tokenizer_hf.go:1`、`tokenizer_hf.go:8`、`tokenizer_tiktoken.go:1`、`tokenizer_tiktoken.go:8`；internal 函数也分别受 tag 约束。
   - 后果：单个 `tokenizer_build.go` 若在 `tokenizer_hf || tokenizer_tiktoken` 下同时引用两个 internal stub，会在只启用一个 tag 时编译失败。
   - 修复：保留两个 build-tag 桩文件，不纳入合并；或采用两个 build-tag 文件分别承载两个函数。

### P1（建议修复，否则高概率引入返工/误判）

1. **P1-1：Shim 数量“7/8”不一致**
   - 证据：`openspec/specs/cleanup-root-layout.md:17`、`openspec/specs/cleanup-root-layout.md:92` 与表格 `openspec/specs/cleanup-root-layout.md:96` 至 `openspec/specs/cleanup-root-layout.md:103`。
   - 建议：统一为 8 个 shim。

2. **P1-2：Pipeline 下沉描述已过期**
   - 证据：internal 已有 `NewDefaultPipeline` 与 transform：`internal/engine/pipeline.go:22`、`internal/engine/pipeline.go:175`。
   - 建议：改为“删除 root 重复实现，root facade 到 internal engine”。

3. **P1-3：根包 legacy 与 internal legacy 重复**
   - 证据：`headroom.go:225` 与 `internal/engine/engine.go:91`。
   - 建议：最终只保留 internal engine legacy；根包测试改行为测试或迁移。

4. **P1-4：`exports.go` 范围过大**
   - 证据：shim 总计 265 行，compat 函数在 `headroom.go:541` 至 `headroom.go:589`。
   - 建议：拆 `exports.go` + `compat.go`。

5. **P1-5：测试数量验收口径不固定**
   - 证据：Spec 写 140 个测试，当前 `go test -json ./...` 统计为 274 个 pass event。
   - 建议：验收写“`go test -race -count=1 ./...` 全部通过”，不要写固定测试数量；或明确统计脚本。

6. **P1-6：缺少外部 import API 验证**
   - 证据：Spec 只写兼容表，未给验证命令，见 `openspec/specs/cleanup-root-layout.md:157` 至 `openspec/specs/cleanup-root-layout.md:175`。
   - 建议：加 temp module compile，覆盖 `Compress`、`CompressString`、shim 类型、Pipeline 公开 API、tokenizer stubs。

---

## 4. 修订后的 Phase 方案建议

### Phase 0：修订 Spec 与验收口径（阻塞执行）

1. 将“根目录 `.go` 文件 ≤8”改为：
   - 根目录**非测试** `.go` 文件从 12 → ≤5；或
   - 根目录总 `.go` 文件从 21 → ≤14。
2. 将 shim 数量统一为 8。
3. 删除“合并 build-tag 桩为单文件”的硬要求，保留两个 tag 文件。
4. 将“Pipeline 变换下沉到 `internal/engine/transforms.go`”改为“消除 root pipeline 重复实现；可选拆分 internal/engine/pipeline.go”。

### Phase 1：低风险文件整理（无逻辑变更）

1. 可选创建 `doc.go`，仅迁移 `headroom.go:1` 至 `headroom.go:31` 的 package comment。
2. 创建 `exports.go`，合并 8 个 shim 的类型别名、常量、构造函数。
3. 创建 `compat.go`，迁移 `SmartCrushJSON`、`CompressCode`、`CompressText`、`NewCompressorRegistry` 等兼容函数。
4. 保留 `tokenizer_hf.go` 与 `tokenizer_tiktoken.go` 不动。
5. 验证：`go test ./...`、`go test -run '^$' -tags tokenizer_hf ./...`、`go test -run '^$' -tags tokenizer_tiktoken ./...`。

### Phase 2：根包 facade 收敛

1. 根包 `Compress`、`CompressString`、`DefaultOptions`、`NewCompressionEngine` 继续作为公共 API facade。
2. 根包 `Pipeline` 优先设计为对 `internal/engine.Pipeline` 的兼容 facade/type alias；必须保持 `NewDefaultPipeline`、`Pipeline.Run`、`NewJSONOffloadTransform` 外部可编译。
3. 删除 root 中与 `internal/engine/pipeline.go` 重复的 transform 实现，避免双实现漂移。
4. 对 `spec_b_e2e_test.go` 中直接访问 root private helper 的用例做迁移或改为外部行为断言。
5. 验证：全量测试 + 外部 temp module compile。

### Phase 3：可选 internal 文件拆分

1. 若 `internal/engine/pipeline.go` 过长，再拆出 `internal/engine/transforms.go`。
2. 该步骤只做 internal 文件内搬迁，不作为根包精简的必要条件。
3. 验证：`go test -race -count=1 ./...`。

### Phase 4：发布/文档收尾

1. `go vet ./...`。
2. 覆盖率按固定命令比对。
3. CLI/proxy smoke：`headroom compress --stats`、`headroom proxy --port=18787` + `/healthz`。
4. 若作为 v0.6.0 发布，补 CHANGELOG；否则不强制。

---

## 5. 最终结论

- 结论：**不通过**。
- P0 数量：2。
- P1 数量：6。
- 可执行性：当前 Spec 不能直接进入实施；必须先修正验收口径和 build-tag 合并方案。
- 总体建议：保留“根包精简、公共 API facade、internal 承载实现”的方向，但把目标从“极限减少文件数”修正为“非测试根包文件清晰且可维护”，并优先消除 root/internal 重复实现。
