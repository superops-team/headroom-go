# Changelog

All notable changes to this project will be documented in this file.

## [v0.5.0] - 2026-06-21

### Added

- Added the Spec A core compression engine upgrade with a policy-driven compression pipeline, engine abstractions, and compressor registry.
- Added tokenizer integration points for built-in, HuggingFace, and tiktoken-compatible tokenizers.
- Added tag protection, specialized transforms, observability hooks, and CCR store support for safer reversible compression flows.
- Added API compatibility, registry, policy, pipeline, tokenizer, fuzz, benchmark, and end-to-end test coverage.

### Changed

- Upgraded SmartCrusher, code compression, text compression, content-kind detection, routing, CLI, and proxy behavior to use the new compression pipeline.
- Bumped package version metadata to `v0.5.0` and prefix metadata to `v0.5`.

### Fixed

- Improved preservation behavior for protected tags, structured content, proxy payload handling, and compression edge cases covered by the expanded test suite.
