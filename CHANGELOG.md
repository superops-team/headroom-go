# Changelog

All notable changes to this project will be documented in this file.

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
