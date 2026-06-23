# headroom-go + Ollama

透明压缩 Ollama API 请求。

## 安装

```bash
go get github.com/superops-team/headroom-go
```

## 使用

```go
import (
    "net/http"
    headroom "github.com/superops-team/headroom-go/integrations/ollama"
)

transport := headroom.NewTransport(http.DefaultTransport, headroom.Config{
    Aggressiveness: 0.5,
})
client := &http.Client{Transport: transport}

// 所有 /api/chat 和 /api/generate 请求自动压缩
resp, _ := client.Post("http://localhost:11434/api/chat", "application/json", body)
```
