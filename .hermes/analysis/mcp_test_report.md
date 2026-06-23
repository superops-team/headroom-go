# headroom-go MCP Server 全场景边界测试报告

**日期:** 2026-06-23
**版本:** v0.8.0
**测试脚本:** /tmp/mcp_test.py

---

## 测试结果: 37/37 PASS ✅

### Test 1: MCP 协议初始化 (5/5)
| 用例 | 结果 |
|------|:--:|
| initialize 返回 result | ✅ |
| protocolVersion = 2024-11-05 | ✅ |
| serverInfo.name = headroom-mcp | ✅ |
| capabilities.tools 存在 | ✅ |
| initialized 通知发送 | ✅ |

### Test 2: tools/list (9/9)
| 用例 | 结果 |
|------|:--:|
| 返回 4 个工具 | ✅ |
| headroom_compress 存在 | ✅ |
| headroom_retrieve 存在 | ✅ |
| headroom_stats 存在 | ✅ |
| headroom_read 存在 | ✅ |
| inputSchema 存在 | ✅ |
| content 为 required | ✅ |
| aggressiveness 参数存在 | ✅ |
| reversible 参数存在 | ✅ |

### Test 3: headroom_compress (10/10)
| 用例 | 结果 | 详情 |
|------|:--:|------|
| JSON 压缩 (aggressiveness=0.8) | ✅ | 正常压缩 |
| 空内容 | ✅ | 返回 "content is required" |
| 大日志 (50行 INFO + 3行 ERROR) | ✅ | 574→27 tokens, 95% 节省 |
| 代码压缩 | ✅ | 64→34 tokens, 47% 节省 |
| 缺少 content 参数 | ✅ | 返回 isError=true |
| aggressiveness=2.0 超范围 | ✅ | 正常处理（clamped） |
| 100KB 超长内容 | ✅ | 正常处理 |
| Unicode/特殊字符 | ✅ | 正常处理 |
| reversible=true 大 JSON | ✅ | 1348→113 tokens, 92% 节省, 含 retrieve ID |
| reversible=true 短内容 | ✅ | 内容过短不附加 ID（预期行为） |

### Test 4: headroom_retrieve (3/3)
| 用例 | 结果 | 详情 |
|------|:--:|------|
| 检索不存在的 ID | ✅ | 返回 "Content not found" |
| 缺少 retrieve_id | ✅ | 返回 "retrieve_id is required" |
| 空 retrieve_id | ✅ | 返回 "retrieve_id is required" |

### Test 5: headroom_stats (3/3)
| 用例 | 结果 |
|------|:--:|
| 返回文本 | ✅ |
| 包含 total_compressions | ✅ |
| 包含 avg_savings | ✅ |

### Test 6: headroom_read (4/4)
| 用例 | 结果 | 详情 |
|------|:--:|------|
| 读取存在的文件 | ✅ | 正常压缩 |
| 读取不存在的文件 | ✅ | 返回 "Failed to read file" |
| 缺少 path | ✅ | 返回 "path is required" |
| 空 path | ✅ | 返回 "path is required" |

### Test 7: 协议边界 (3/3)
| 用例 | 结果 |
|------|:--:|
| 未知方法 → error code -32601 | ✅ |
| 未知工具 → isError=true | ✅ |
| 连续 5 次快速请求 | ✅ |

---

## 🐛 发现的 Bug

### Bug 1: headroom_retrieve 无法检索之前压缩的内容 (P0)
**文件:** `internal/mcp/server.go:handleRetrieve`
**问题:** 每次调用 `headroom.NewCCR(headroom.CCRConfig{})` 创建新 CCR 实例，无法检索之前 `headroom_compress` 中压缩的内容。
**修复:** 使用共享的 CCR 实例（与 compress 共用 StatsTracker 中的 CCR）。

### Bug 2: headroom_stats cache_entries 始终为 0 (P1)
**文件:** `internal/mcp/server.go:StatsTracker.snapshot`
**问题:** `cache_entries` 和 `cache_bytes` 硬编码为 0，未从实际 CCR 实例读取。
**修复:** StatsTracker 持有 CCR 引用，调用 `store.Stats()` 获取真实数据。

### Bug 3: headroom_read 缺少 HEADROOM_MCP_READ 环境变量检查 (P2)
**文件:** `internal/mcp/server.go:handleRead`
**问题:** Spec 要求 `headroom_read` 仅在 `HEADROOM_MCP_READ=on` 时启用，当前无检查。
**修复:** 在 handleRead 开头检查环境变量，未设置时返回错误。

---

## 总结

- MCP 协议实现完整，4 个工具全部可用
- 边界处理健壮：空输入/超范围/超大内容/特殊字符均正确处理
- 错误响应符合 MCP 规范（isError + JSON-RPC error code）
- 3 个 bug 需修复，其中 P0 影响核心功能（retrieve 不可用）
