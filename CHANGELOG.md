# Changelog

All notable changes to this project will be documented in this file.

## [v0.8.0] - 2026-06-23

### Added

- **Go 生态集成**: 3 个框架集成 + K8s sidecar
  - `integrations/go-openai/` — OpenAI Go SDK 透明压缩
  - `integrations/langchaingo/` — Document Compressor
  - `integrations/ollama/` — Ollama HTTP 中间件
  - `integrations/k8s/` — Sidecar 部署清单
- **K8s Helm Chart**: `charts/headroom/` — standalone + sidecar 双模式
- **文档站**: `docs/` — Starlight 文档站（安装/快速开始/CLI/API 参考）
- **流式响应支持**: proxy 支持 SSE streaming（`stream:true`）
- **Homebrew formula**: `homebrew/headroom.rb`
- **Benchmark 页**: 自动化 benchmark + GitHub Pages 报告
- **CI 扩展**: docs.yml + benchmark.yml

### Changed

- proxy: 流式请求从返回 400 改为透传 SSE 流
- 版本号升级至 v0.8.0

## [v0.7.0] - 2026-06-23

### Added

- **MCP Server**: `headroom mcp serve` — stdio JSON-RPC MCP Server 提供 4 个工具
  - `headroom_compress` — 压缩文本内容（自动检测 10 种内容类型）
  - `headroom_retrieve` — 通过 ID 检索原始内容
  - `headroom_stats` — 会话压缩统计
  - `headroom_read` — 通过 CCR 缓存读取文件
- **wrap 命令**: `headroom wrap <agent>` — 启动代理 + 自动配置 IDE
  - 支持 Claude Code、Codex、Copilot CLI、Generic
  - `--apply` 自动修改配置文件，Ctrl+C 恢复原配置
- **CI/CD**: GitHub Actions 自动化流水线
  - CI: build + test + race + vet + lint + security（push/PR 触发）
  - Release: 多平台二进制构建 + 自动发布（tag 触发）
  - Docker: 多 arch 镜像自动推送到 ghcr.io
- **Docker 镜像**: 多阶段构建（FROM scratch），<10MB
- **golangci-lint 配置**: `.golangci.yml`

### Changed

- 版本号升级至 v0.7.0

## [v0.6.0] - 2026-06-22

### Changed

- **根包文件精简**：合并 8 个 shim 文件为 `exports.go` + `compat.go`
  - `exports.go` — 类型别名 + 常量 re-export（ContentKind/Observer/CacheAligner/CCR/ContentRouter/TagProtector/Tokenizer）
  - `compat.go` — 历史兼容 API（SmartCrushJSON/CompressCode/CompressText/CompressorRegistry 等）
- **headroom.go 精简**：消除与 `internal/engine/pipeline.go` 重复的 Pipeline 实现，根包 Pipeline 改为类型别名
- **version.go 合并**：版本常量合并到 `headroom.go`
- 根目录非测试 `.go` 文件从 12 → 5 个，总 `.go` 文件从 21 → 14 个
- 根包测试覆盖率从 73.3% → 85.2%

### Removed

- 删除 8 个旧 shim 文件：`cachealigner.go`、`ccr.go`、`ccrstore.go`、`content_kind.go`、`observability.go`、`router.go`、`tag_protector.go`、`tokenizer.go`
- 删除 `version.go`（合并到 `headroom.go`）

## [v0.5.0] - 2026-06-21

### Changed

- **项目结构规范化**：按 Go 社区标准将实现代码迁移到 `internal/` 子包
  - `internal/compressors/` — SmartCrusher、Code、Text、专用变换、CompressorRegistry
  - `internal/engine/` — CompressionEngine、Pipeline、Policy
  - `internal/router/` — ContentRouter 内容类型检测
  - `internal/tokenizer/` — Tokenizer 接口 + tiktoken/HuggingFace 后端
  - `internal/ccr/` — 可逆压缩存储
  - `internal/cachealigner/` — KV Cache 前缀对齐
  - `internal/tagprotector/` — XML Tag 保护
  - `internal/types/` — 共享类型定义（ContentKind、Observer、Warning）
- 根包精简为公共 API 层：`Message`、`Options`、`Result`、`Compress()`、`CompressString()`
- 通过类型别名和兼容 shim 保持 100% 向后兼容
- 版本号升级至 v0.5.0

### Fixed

- 修复 `internal/ccr/ccr.go` 编译错误（`Legacytypes` typo → `LegacyCCRIDVersion`）

## [v0.4.3] - 2026-06-21

### Changed

- README 全面重写：增加痛点开场、对比表、真实场景用例、成本节省估算
- 新增 Killer Features 专区（Tag Protector、CCR、CacheAligner、Proxy）
- 新增贡献指南和 Go Reference badge
- 视觉优化：居中布局、emoji 导航、底部趣味文案

## [v0.4.2] - 2026-06-21

### Fixed

- Fixed Spec C and Spec D brooks-lint issues and expanded coverage tests.

## [v0.4.1] - 2026-06-21

### Changed

- Refactored Spec B APIs to keep existing interface compatibility while reducing coupling across compressor entry points.
- Unified configuration loading and defaults so headroom, proxy, and compressor behavior share a consistent configuration surface.
- Split large compression and proxy functions into smaller units to improve maintainability and targeted test coverage.

### Added

- Added Spec B end-to-end coverage plus expanded API compatibility, registry, headroom, proxy, CCR, and text compressor tests.

## [v0.4.0] - 2026-06-21

### Added

- Added the Spec A core compression engine upgrade with a policy-driven compression pipeline, engine abstractions, and compressor registry.
- Added tokenizer integration points for built-in, HuggingFace, and tiktoken-compatible tokenizers.
- Added tag protection, specialized transforms, observability hooks, and CCR store support for safer reversible compression flows.
- Added API compatibility, registry, policy, pipeline, tokenizer, fuzz, benchmark, and end-to-end test coverage.

### Changed

- Upgraded SmartCrusher, code compression, text compression, content-kind detection, routing, CLI, and proxy behavior to use the new compression pipeline.
- Bumped package version metadata to `v0.4.0` and prefix metadata to `v0.4`.

### Fixed

- Improved preservation behavior for protected tags, structured content, proxy payload handling, and compression edge cases covered by the expanded test suite.
