# headroom-go + langchaingo

将 headroom 作为 langchaingo 的 Document Compressor 使用。

## 安装

```bash
go get github.com/superops-team/headroom-go
go get github.com/tmc/langchaingo
```

## 使用

```go
import (
    headroom "github.com/superops-team/headroom-go/integrations/langchaingo"
)

compressor := headroom.NewDocumentCompressor(headroom.Config{
    Aggressiveness: 0.5,
})

// 压缩文档
compressed, err := compressor.CompressDocuments(ctx, documents)

// 在 chain 中使用
chain := chains.NewStuffDocuments(llm)
chain.DocumentCompressor = compressor
```
