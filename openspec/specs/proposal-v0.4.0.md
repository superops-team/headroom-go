# headroom-go v0.4.0 重构提案

## 1. 背景与目的

### 1.1 为什么现在做

headroom-go v0.3.0 已完成核心功能开发，50 个测试用例全部通过，具备生产可用性。通过 Brooks-Lint 方法论进行代码审查后，发现以下问题：

1. **可扩展性瓶颈**：新增内容类型（如 Markdown）需要修改 3 个文件（ContentKind/router/headroom）
2. **维护成本上升**：版本号/配置分散在 5+ 处，升级时需全局搜索替换
3. **认知过载**：部分函数超过 80 行，新成员上手困难
4. **知识重复**：HTTP 错误响应模板重复 7 次，配置结构重复 4 份

这些问题在当前规模下影响有限，但随着项目增长，维护成本将指数上升。

### 1.2 目标

在不改变任何用户可见行为的前提下，重构代码结构，降低长期维护成本，为 v0.5.0 的新功能（如 Markdown 支持、流式压缩）打好基础。

## 2. 变更概述

### 2.1 核心改动

| 维度 | v0.3.0 现状 | v0.4.0 目标 |
|------|------------|------------|
| 压缩器路由 | switch-case 硬编码 | `Compressor` 接口 + map 注册 |
| 压缩配置 | 4 个独立 Config | 统一 `CompressionConfig` |
| 版本管理 | 5 处散落 | 3 个常量集中管理 |
| HTTP 错误 | 7 处重复模板 | `writeError()` 辅助函数 |
| CacheAligner | struct 封装 | `AlignPrefix()` 纯函数 |
| Compress() | 81 行 | ≤30 行 + 步骤函数 |
| TextCompressor | `flushDup` 闭包歧义 | `flushDuplicateGroup` 显式函数 |
| CCR Store | 双重条件守卫 | `isFull()` / `contains()` 语义方法 |

### 2.2 文件变更清单

| 操作 | 文件 | 说明 |
|------|------|------|
| 新建 | `version.go` | 版本常量统一管理 |
| 新建 | `compressor.go` | `Compressor` 接口 + `CompressionConfig` |
| 新建 | `*_compressor.go`（3个） | 各压缩器实现接口 |
| 修改 | `headroom.go` | 拆分函数 + 使用新接口 |
| 修改 | `textcompressor.go` | 重构 flushDuplicateGroup |
| 修改 | `ccr.go` | 简化 Store 逻辑 + 使用 CCRIDVersion |
| 修改 | `proxy/proxy.go` | 使用 writeError + headroom.Version |
| 修改 | `cmd/headroom/main.go` | 使用 headroom.Version |
| 删除 | `cachealigner.go` | 合并为 AlignPrefix 纯函数 |
| 删除 | `cachealigner_test.go` | 测试覆盖移至 headroom_test.go |

### 2.3 向后兼容性

**完全兼容，无破坏性变更：**

- `package headroom` 的公开 API 签名不变：`Compress`、`CompressString`、`Options`、`Result`、`Message`
- `go.mod` module path 不变
- CLI 参数和输出格式不变
- CCR 存储的 ID 格式升级：`"v2_xxx"` → `"v3_xxx"`（旧缓存自动失效，非破坏性）
- `AlignPrefix` 默认启用行为不变

## 3. 风险评估

### 3.1 技术风险

| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|----------|
| 重构引入回归 | 中 | 高 | 每个原子任务后立即运行全量测试 |
| 接口变更导致用户代码编译失败 | 低 | 高 | 仅修改 internal 实现，不改变公开 API |
| Phase 顺序依赖导致返工 | 低 | 低 | Phase 1 任务完全独立，可独立验证 |

### 3.2 兼容性风险

- **CCR ID 版本升级**：`"v2_"` → `"v3_"`，已存在的本地缓存无法检索。这是**预期行为**，非破坏性。
- **测试覆盖盲区**：新增的内部函数（`shouldSkip`、`postProcess` 等）需要补充测试用例。

## 4. 开发计划

### 4.1 阶段划分

```
Week 1（Phase 1：基础设施）
├── T1-001 ~ T1-008
├── 每日：go build + go test -race
└── 目标：独立验证，不影响现有代码

Week 2-3（Phase 2：接口抽象）
├── T1-009 ~ T1-013
├── 每日：全量测试 + 新增接口测试
└── 目标：引入 Compressor 接口，统一配置

Week 3-4（Phase 3：逻辑重构）
├── T1-014 ~ T1-018
├── 每日：测试覆盖 + 代码行数对比
└── 目标：消除歧义，提升可读性

Week 4（验收阶段）
├── T1-999：全量回归
├── 对比 v0.3.0 代码行数
└── 输出重构总结报告
```

### 4.2 测试要求

每个任务完成后必须：
1. `go build ./...` 通过
2. `go vet ./...` 无警告
3. `go test -race -count=1 ./...` 全部通过
4. 补充新测试用例（任务清单中指定）

### 4.3 代码审查要求

- 每次 commit 前进行 self-review
- Phase 完成后提交 PR，进行 peer review
- 重点审查：接口设计合理性、测试覆盖率、命名一致性

## 5. 成功标准

| 指标 | 当前值（v0.3.0） | 目标值（v0.4.0） |
|------|-----------------|-----------------|
| 测试覆盖率 | 50 个测试 | ≥60 个测试（新增≥10） |
| 代码行数（.go 源文件） | ~1200 行 | ≤1100 行（净减少≥100） |
| go vet 警告 | 0 | 0 |
| race 检测 | 通过 | 通过 |
| Compress() 函数行数 | 81 行 | ≤30 行 |
| 版本常量散落处数 | 5 处 | 3 处（version.go） |
| HTTP 错误模板重复 | 7 处 | 0 处 |

## 6. 文档更新

| 文档 | 更新内容 |
|------|----------|
| `README.md` | 版本号更新为 v0.4.0 |
| `openspec/specs/refactor-v0.4.0.md` | 本文档，作为变更依据 |
| `openspec/specs/tasks-v0.4.0.md` | 原子任务清单 |
| `openspec/config.yaml` | 版本配置更新 |
| `openspec/specs/spec.md` | 架构图更新（新增 Compressor 接口说明） |

## 7. 决策记录

| 日期 | 决策 | 理由 |
|------|------|------|
| 2026-06-19 | 不修改 module path | 当前 `github.com/superops-team/headroom-go` 已与仓库对齐，用户已熟悉 |
| 2026-06-19 | 不引入第三方依赖 | 保持零外部依赖原则 |
| 2026-06-19 | 不拆分为多个子模块 | 当前规模不需要，保持简单 |
| 2026-06-19 | CCR ID 版本从 v2 升级到 v3 | 存储格式不变，但通过版本前缀隔离新旧缓存 |
