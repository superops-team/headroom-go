# Spec D: 核心功能端到端覆盖测试

## 1. 概述

当前覆盖率：根包 88.7%，proxy 87.0%，整体 82.0%（cmd/headroom CLI 入口 0% 拉低）。虽然整体已超 80%，但存在覆盖盲区需补强。

## 2. 覆盖盲区分析

### 2.1 核心路径未覆盖（P0）

| 函数 | 文件 | 当前覆盖 | 影响 |
|------|------|----------|------|
| `estimateTokens` | headroom.go:228 | 0% | 所有压缩路径的 token 估算均依赖此函数，0% 意味着无直接测试 |
| `NoopObserver.ObserveCompressionStep` | observability.go:24 | 0% | Observer 接口的空实现未被测试 |
| `Compress()` pipeline 分发路径 | engine.go:33 | 部分 | `EnablePipeline=true` 路径有测试但部分子函数 0% |
| `NewTokenizer` fallback 分支 | tokenizer.go:86-90 | 0% | tiktoken/hf stub 的 Name/Count/CountBatch 未被调用 |

### 2.2 专项能力未覆盖（P1）

| 函数 | 文件 | 当前覆盖 | 影响 |
|------|------|----------|------|
| `DiffCompressor.Apply` | specialized_transforms.go:63 | 0% | Diff 压缩从未被端到端触发 |
| `HTMLCompressor.Name` | specialized_transforms.go:148 | 0% | HTML 压缩器名称未验证 |
| `LogCompressor.Name` | specialized_transforms.go:54 | 0% | Log 压缩器名称未验证 |
| `lineIndent` | codecompressor.go:265 | 0% | 代码缩进辅助函数未被直接测试 |

### 2.3 边界/异常路径未覆盖（P1）

| 场景 | 当前状态 |
|------|----------|
| `CompressString` 空输入 | 部分覆盖 |
| `CompressString` 纯空白输入 | 未覆盖 |
| `ContentRouter.Detect` 空字符串 | 已有测试 |
| `ContentRouter.Detect` 边界：恰好 2 个关键字行 | 已有测试 |
| `SmartCrushJSON` 非法 JSON 降级 | 已有测试 |
| `CCR.Retrieve` 过期条目 | 已有测试 |
| `CCR.backgroundGC` 触发清理 | 已有测试 |
| `CacheAligner.Align` disabled 模式 | 已有测试 |
| `TagProtector.Protect` 无标签内容 | 已有测试 |
| `TagProtector.Restore` 空内容 | 已有测试 |
| Pipeline `TokenBudget=0` 降级 | 未覆盖 |
| Pipeline `Query` 为空 | 未覆盖 |
| `TokenizerConfig.AllowFallback=false` + 不可用后端 | 部分覆盖 |

## 3. 测试策略

### Phase 1：核心路径补强（目标：根包 ≥ 92%）

| Task | 内容 | 预计新增测试 |
|------|------|-------------|
| D-P1-T01 | `estimateTokens` 直接测试：空字符串、ASCII、中文、emoji | 4 |
| D-P1-T02 | `NoopObserver` 接口合规测试 | 2 |
| D-P1-T03 | Pipeline `TokenBudget=0` 降级到 legacy 路径 | 2 |
| D-P1-T04 | Pipeline `Query=""` 行为验证 | 1 |
| D-P1-T05 | `NewTokenizer` tiktoken/hf stub 覆盖 | 3 |
| D-P1-T06 | `lineIndent` 直接测试 | 3 |

### Phase 2：专项能力端到端（目标：根包 ≥ 95%）

| Task | 内容 | 预计新增测试 |
|------|------|-------------|
| D-P2-T01 | Diff 压缩端到端：正常 diff、空 diff、大 diff | 3 |
| D-P2-T02 | HTML 压缩端到端：正常 HTML、含注释、空 HTML | 3 |
| D-P2-T03 | Log 压缩端到端：混合日志级别、重复行 | 2 |
| D-P2-T04 | Search 结果压缩端到端 | 2 |
| D-P2-T05 | Tabular/Spreadsheet 压缩端到端 | 2 |

### Phase 3：异常路径加固（目标：proxy ≥ 90%）

| Task | 内容 | 预计新增测试 |
|------|------|-------------|
| D-P3-T01 | Proxy `TokenizerConfig.AllowFallback=false` + 不可用后端 | 1 |
| D-P3-T02 | Proxy 超大请求体（>1MB）拒绝 | 1 |
| D-P3-T03 | Proxy 上游返回非 200 状态码透传 | 1 |
| D-P3-T04 | `CompressString` 边界：空输入、纯空白、超长单行 | 3 |

## 4. 约束

- 不改变任何生产代码逻辑
- 不改变公开 API 签名
- 纯测试补充，零功能变更
- 每个 Task 独立可运行、可验证

## 5. 验收标准

- [ ] `go test -coverprofile=/tmp/coverage.out ./...` 根包 ≥ 92%
- [ ] `go test -coverprofile=/tmp/coverage.out ./...` proxy ≥ 90%
- [ ] `go test -race -count=1 ./...` 全部通过
- [ ] `go vet ./...` 无警告
- [ ] 新增测试用例 ≥ 30 个
