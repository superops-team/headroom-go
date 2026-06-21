# Spec C: brooks-lint 反馈优化

## 概述

基于 brooks-lint PR Review 对 Spec B (v0.4.1) 的审查结果，修复 3 个 Warning 和 1 个 Suggestion。

## 问题清单

### P1 - Warning

1. **CompressionConfig 位置不当**：定义在 `codecompressor.go`，被全项目引用，应移至独立文件
2. **legacySkipDecision / legacyPostProcessResult 冗余**：两个私有结构体仅用于单次返回，Go 多返回值已足够
3. **compressionConfigFromOptions / smartCrushConfigFromOptions 重复**：两个仅差一行的辅助函数

### P2 - Suggestion

4. **postProcessLegacyCompression 可进一步拆分**：三个独立后处理步骤在一个函数内

## 约束

- 不改变任何公开 API 签名
- 不改变任何行为逻辑
- 所有现有测试必须通过
- 纯重构，零功能变更
