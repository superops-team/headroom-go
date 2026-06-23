# Spec: 流式响应支持

**版本:** v0.8.0-streaming
**日期:** 2026-06-22
**优先级:** P2
**状态:** 待确认

---

## 1. 背景

当前 proxy 明确不支持流式响应（`stream:true` 返回 400）。但大量 LLM 应用依赖 SSE streaming 实现打字机效果。Headroom (Python) 已支持流式响应。

---

## 2. 目标

为 proxy 添加 SSE (Server-Sent Events) 流式响应支持：

- 接收 `stream:true` 请求
- 压缩 messages 后转发到上游
- 将上游 SSE 流透传回客户端
- 支持 `stream_options` (include_usage 等)

---

## 3. 技术方案

### 3.1 流程

```
Client (stream:true)
  → proxy 接收请求
  → 压缩 messages
  → 转发到上游 (stream:true)
  → 读取上游 SSE 流
  → 逐 chunk 透传 (可选注入压缩统计)
  → Client 收到流式响应
```

### 3.2 实现要点

```go
func (p *Proxy) handleStream(w http.ResponseWriter, r *http.Request, body []byte) {
    // 1. 解析请求，压缩 messages
    compressed, err := p.compressMessages(body)
    
    // 2. 转发到上游
    upstreamReq, _ := http.NewRequest("POST", p.upstream, bytes.NewReader(compressed))
    upstreamReq.Header.Set("Accept", "text/event-stream")
    
    resp, err := p.client.Do(upstreamReq)
    defer resp.Body.Close()
    
    // 3. 设置 SSE 响应头
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.WriteHeader(resp.StatusCode)
    
    // 4. 透传 SSE 流
    flusher := w.(http.Flusher)
    scanner := bufio.NewScanner(resp.Body)
    for scanner.Scan() {
        line := scanner.Text()
        fmt.Fprintf(w, "%s\n", line)
        flusher.Flush()
    }
}
```

### 3.3 压缩统计注入（可选）

在 SSE 流的 `[DONE]` 之前注入压缩统计：

```
data: {"choices":[{"delta":{"content":"..."}}]}
...
data: {"headroom_stats":{"original_tokens":1500,"compressed_tokens":450,"savings":0.7}}
data: [DONE]
```

---

## 4. 文件变更

| 操作 | 文件 | 说明 |
|------|------|------|
| **修改** | `proxy/proxy.go` | 添加流式处理逻辑 (~80 行) |
| **新建** | `proxy/stream_test.go` | 流式测试 (~60 行) |

---

## 5. 验收标准

- [ ] `stream:true` 请求返回 SSE 流
- [ ] 压缩后 messages 正确转发
- [ ] 上游 SSE 流完整透传
- [ ] `stream_options.include_usage` 正确处理
- [ ] 上游错误时正确返回错误 SSE
- [ ] 客户端断开时清理上游连接
- [ ] `go test -race ./proxy/...` 通过

---

## 6. 时间估算

| 阶段 | 预估 |
|------|------|
| 流式处理核心 | 1.5h |
| 上游 SSE 透传 | 1h |
| 错误处理 + 断开清理 | 0.5h |
| 测试 | 1h |
| **总计** | **~4h** |
