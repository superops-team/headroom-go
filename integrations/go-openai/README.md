# headroom-go + OpenAI Go SDK

透明压缩 OpenAI Go SDK 的所有 ChatCompletion 请求。

## 安装

```bash
go get github.com/superops-team/headroom-go
go get github.com/sashabaranov/go-openai
```

## 使用

```go
import (
    "github.com/sashabaranov/go-openai"
    headroom "github.com/superops-team/headroom-go/integrations/go-openai"
)

func main() {
    client := openai.NewClient("sk-xxx")

    // 包装 client，自动压缩所有请求
    client = headroom.WrapClient(client, headroom.Config{
        Aggressiveness: 0.5,
        Reversible:     true,
    }).(*openai.Client)

    // 正常使用，messages 自动压缩
    resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
        Model: "gpt-4",
        Messages: []openai.ChatCompletionMessage{
            {Role: "user", Content: hugeContent},
        },
    })
}
```

## 配置

| 参数 | 默认值 | 说明 |
|------|--------|------|
| Aggressiveness | 0.5 | 压缩强度 0.0-1.0 |
| Reversible | true | 启用可逆压缩 |
| AlignPrefix | false | KV Cache 前缀对齐 |
| EnablePipeline | false | Pipeline 模式 |
