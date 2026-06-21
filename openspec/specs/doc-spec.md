# Headroom-Go 文档编写 Spec

**版本:** v1.0.0
**日期:** 2026-06-21
**状态:** 待执行

---

## 1. 背景

headroom-go v0.5.0 已完成核心功能开发（Spec A/B/C/D/E），140 个测试全通过，项目结构规范化完成。当前 README.md 已有基础内容（270 行），但缺少大量关键文档。需要编写完备的文档覆盖所有功能和用法。

## 2. 现有文档评估

| 文档 | 状态 | 行数 | 评分 |
|------|------|------|------|
| README.md | ✅ 存在 | 270 行 | B+ — 有痛点/对比/Quick Start/架构图，缺 TOC/CLI 参考/API 参考/高级用法 |
| CHANGELOG.md | ✅ 存在 | 69 行 | A — 格式规范，内容完整 |
| install.sh | ✅ 存在 | 99 行 | A — 功能完整 |
| LICENSE | ✅ 存在 | — | A |
| CONTRIBUTING.md | ❌ 缺失 | — | — |
| llms.txt | ❌ 缺失 | — | — |
| 代码注释 | ⚠️ 部分 | — | C — 多数导出符号无 doc comment |
| CLI --help | ⚠️ 简陋 | 199 行 | C — 只列了子命令名，无 flag 说明 |

## 3. 文档交付清单

### Phase 1: README.md 重写（核心）

| # | 章节 | 内容 | 优先级 |
|---|------|------|--------|
| 1 | Title + Badges | 保留现有，更新测试数 138→140 | P0 |
| 2 | TOC | 新增目录导航 | P0 |
| 3 | The Problem | 保留现有 | P0 |
| 4 | Why Headroom Go? | 保留对比表 | P0 |
| 5 | Quick Start | 保留，增加 install.sh 说明 | P0 |
| 6 | **CLI Reference** | 🆕 完整 CLI 命令参考，所有 flag 说明 | P0 |
| 7 | **Go Library API** | 🆕 核心 API 参考：Compress/CompressString/Options/Result/Message | P0 |
| 8 | How It Works | 保留架构图，增加 Pipeline vs Legacy 路径说明 | P0 |
| 9 | **Content Types** | 🆕 10 种内容类型详细说明 + 压缩策略 + 示例 | P0 |
| 10 | **Compression Modes** | 🆕 Aggressiveness 三级说明 + TokenLimit + TokenBudget | P1 |
| 11 | Killer Features | 保留，增加 TagProtector/CCR/CacheAligner 详细用法 | P0 |
| 12 | **Proxy Guide** | 🆕 完整代理配置：端点/认证/流式限制/健康检查/Request ID/错误格式 | P0 |
| 13 | **Tokenizer Guide** | 🆕 三种后端对比 + 配置方法 | P1 |
| 14 | **Pipeline Mode** | 🆕 Pipeline vs Legacy 对比 + 启用方法 + Policy 配置 | P1 |
| 15 | **Reversible Compression** | 🆕 CCR 详细用法：Store/Retrieve/Stats/TTL/MaxEntries | P1 |
| 16 | **Custom Compressor** | 🆕 自定义压缩器：Compressor 接口 + Registry 注册 | P1 |
| 17 | **Observability** | 🆕 Observer 接口 + CompressionStep + Warning | P2 |
| 18 | Real-World Performance | 保留 benchmark 表 | P1 |
| 19 | Architecture | 保留 ASCII 图 | P1 |
| 20 | Use Cases | 保留场景表 | P1 |
| 21 | **Deployment** | 🆕 systemd/Docker/生产部署指南 | P2 |
| 22 | **Troubleshooting** | 🆕 常见问题 FAQ | P2 |
| 23 | Development | 保留 | P1 |
| 24 | Contributing | 保留 + 链接 CONTRIBUTING.md | P1 |
| 25 | License | 保留 | P1 |

### Phase 2: 配套文档

| 文档 | 内容 | 优先级 |
|------|------|--------|
| CONTRIBUTING.md | 开发环境/构建/测试/PR 流程/代码规范 | P1 |
| llms.txt | AI Agent 可消费的结构化项目概览 | P1 |

### Phase 3: 代码注释补全

| 文件 | 需补注释的导出符号 | 优先级 |
|------|-------------------|--------|
| headroom.go | Message/Options/Result/Compress/CompressString/DefaultOptions | P0 |
| content_kind.go | ContentKind 所有常量 | P0 |
| observability.go | Warning/CompressionStep/Observer/NoopObserver | P1 |
| version.go | Version/PrefixVersion/CCRIDVersion | P1 |
| cachealigner.go | CacheAligner/CacheAlignerConfig/NewCacheAligner/Align | P1 |
| ccr.go | CCR/CCRConfig/NewCCR/Store/Retrieve/Stats | P1 |
| router.go | ContentRouter/NewContentRouter/Detect | P1 |
| tag_protector.go | TagProtector/ProtectedContent/NewTagProtector | P1 |
| tokenizer.go | Tokenizer/TokenizerConfig/TokenizerBackend/FallbackTokenizer/NewTokenizer | P1 |
| proxy/proxy.go | Config/NewProxy/所有端点 | P1 |
| internal/compressors/ | Compressor/CompressionConfig/CompressorRegistry/SmartCrushJSON/CompressCode/CompressText | P1 |
| internal/engine/ | CompressionEngine/NewCompressionEngine/Pipeline/Policy | P1 |

## 4. 关键设计决策

| 决策 | 理由 |
|------|------|
| README 为主文档，不拆多文件 | 项目规模适中，单文件 README 足够 |
| CLI 参考用表格而非 man-page 风格 | Markdown 表格在 GitHub/pkg.go.dev 渲染更好 |
| 代码示例全部可运行 | 从实际测试用例提取，确保不腐化 |
| llms.txt 独立文件 | AI Agent 消费场景需要结构化纯文本 |
| 代码注释用 godoc 标准格式 | pkg.go.dev 自动渲染 |

## 5. 验收标准

- [ ] README.md 覆盖所有 P0/P1 章节，每个章节有可运行代码示例
- [ ] CLI 所有 flag 有文档说明
- [ ] 所有公开 API 有 godoc 注释
- [ ] CONTRIBUTING.md 可让新贡献者 10 分钟内开始开发
- [ ] llms.txt 覆盖项目概述/快速开始/核心 API/常见模式
- [ ] `go build ./...` 通过
- [ ] `go test -race ./...` 全部通过
- [ ] `go vet ./...` 零警告

## 6. 执行计划

| Phase | 内容 | 方式 |
|-------|------|------|
| Phase 1 | README.md 完整重写 | Coco 生成初稿 → 屠龙刀审核修正 |
| Phase 2 | CONTRIBUTING.md + llms.txt | Coco 生成 |
| Phase 3 | 代码注释补全 | Coco 逐文件补全 |
| Phase 4 | 验证 | 屠龙刀验证编译/测试/vet |
