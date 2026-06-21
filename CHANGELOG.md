# Changelog

All notable changes to this project will be documented in this file.

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
