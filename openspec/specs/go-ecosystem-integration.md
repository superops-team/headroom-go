# Spec: Go 生态集成

**版本:** v0.7.0-ecosystem
**日期:** 2026-06-22
**优先级:** P1
**状态:** 待确认

---

## 1. 背景

Headroom (Python) 有 LangChain、Agno、Vercel AI SDK、LiteLLM 等 8+ 框架集成。headroom-go 作为 Go 项目，应优先覆盖 Go 生态的主流 AI/LLM 框架。

---

## 2. 目标

提供 3 个 Go 生态集成 + 2 个通用集成示例：

### 2.1 Go 生态集成

| 集成 | 框架 | 形式 | 优先级 |
|------|------|------|:------:|
| **OpenAI Go SDK** | `github.com/sashabaranov/go-openai` | HTTP 中间件 | P1 |
| **langchaingo** | `github.com/tmc/langchaingo` | Document Compressor | P1 |
| **Ollama** | `github.com/ollama/ollama` | HTTP 中间件 | P1 |

### 2.2 通用集成

| 集成 | 形式 | 优先级 |
|------|------|:------:|
| **HTTP 中间件** | 标准 `http.RoundTripper`，透明压缩任何 OpenAI 兼容请求 | P1 |
| **K8s Sidecar** | 与任意 LLM 客户端 sidecar 部署 | P1 |

---

## 3. 技术方案

### 3.1 目录结构

```
integrations/
├── go-openai/
│   ├── headroom.go          # HTTP RoundTripper 包装
│   ├── example_test.go      # 集成示例
│   └── README.md
├── langchaingo/
│   ├── compressor.go        # Document Compressor 实现
│   ├── example_test.go
│   └── README.md
├── ollama/
│   ├── middleware.go         # HTTP 中间件
│   ├── example_test.go
│   └── README.md
└── k8s/
    ├── sidecar.yaml          # K8s Sidecar 部署清单
    └── README.md
```

### 3.2 OpenAI Go SDK 集成

```go
import (
    "github.com/sashabaranov/go-openai"
    headroom "github.com/superops-team/headroom-go/integrations/go-openai"
)

func main() {
    client := openai.NewClient("sk-xxx")
    // 包装 HTTP client，透明压缩所有请求
    client = headroom.WrapClient(client, headroom.Config{
        Aggressiveness: 0.5,
        Reversible:     true,
    })
    // 后续所有 ChatCompletion 请求自动压缩
    resp, _ := client.CreateChatCompletion(ctx, req)
}
```

### 3.3 langchaingo 集成

```go
import (
    "github.com/tmc/langchaingo/chains"
    headroom "github.com/superops-team/headroom-go/integrations/langchaingo"
)

compressor := headroom.NewDocumentCompressor(headroom.Config{
    Aggressiveness: 0.5,
})
// 在 chain 中作为 document compressor 使用
```

### 3.4 HTTP RoundTripper（通用）

```go
// 核心：标准 http.RoundTripper，透明压缩任何 OpenAI 兼容 API 请求
transport := headroom.NewRoundTripper(http.DefaultTransport, headroom.Config{
    Aggressiveness: 0.5,
})
client := &http.Client{Transport: transport}
// 所有通过此 client 的请求自动压缩 messages
```

### 3.5 K8s Sidecar

```yaml
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: app
    image: my-llm-app
    env:
    - name: OPENAI_BASE_URL
      value: http://localhost:18787/v1
  - name: headroom
    image: ghcr.io/superops-team/headroom-go:latest
    args: ["proxy", "--port=18787"]
    ports:
    - containerPort: 18787
```

---

## 4. 文件变更

| 操作 | 文件 | 说明 |
|------|------|------|
| **新建** | `integrations/go-openai/headroom.go` | OpenAI Go SDK 集成 |
| **新建** | `integrations/go-openai/example_test.go` | 示例测试 |
| **新建** | `integrations/go-openai/README.md` | 文档 |
| **新建** | `integrations/langchaingo/compressor.go` | langchaingo 集成 |
| **新建** | `integrations/langchaingo/example_test.go` | 示例测试 |
| **新建** | `integrations/langchaingo/README.md` | 文档 |
| **新建** | `integrations/ollama/middleware.go` | Ollama 集成 |
| **新建** | `integrations/ollama/example_test.go` | 示例测试 |
| **新建** | `integrations/ollama/README.md` | 文档 |
| **新建** | `integrations/k8s/sidecar.yaml` | K8s 部署清单 |
| **新建** | `integrations/k8s/README.md` | 文档 |

---

## 5. 验收标准

- [ ] OpenAI Go SDK 集成：WrapClient 后请求自动压缩
- [ ] langchaingo 集成：DocumentCompressor 可正常压缩文档
- [ ] Ollama 集成：HTTP 中间件透明压缩
- [ ] K8s sidecar 清单可直接 `kubectl apply`
- [ ] 每个集成有 example_test.go 可运行
- [ ] 每个集成有 README.md

---

## 6. 时间估算

| 阶段 | 预估 |
|------|------|
| HTTP RoundTripper 核心 | 1h |
| go-openai 集成 | 1h |
| langchaingo 集成 | 1h |
| ollama 集成 | 0.5h |
| K8s sidecar | 0.5h |
| 文档 | 1h |
| **总计** | **~5h** |
