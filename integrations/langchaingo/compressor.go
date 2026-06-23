// Package headroom 提供 langchaingo 的 Document Compressor 集成。
//
// 使用方式：
//
//	import headroom "github.com/superops-team/headroom-go/integrations/langchaingo"
//
//	compressor := headroom.NewDocumentCompressor(headroom.Config{
//	    Aggressiveness: 0.5,
//	})
//	docs, _ := compressor.CompressDocuments(ctx, documents)
package headroom

import (
	"context"

	headroom "github.com/superops-team/headroom-go"
)

// Config 配置压缩行为。
type Config struct {
	Aggressiveness float64
	Reversible     bool
	EnablePipeline bool
}

// DefaultConfig 返回推荐配置。
func DefaultConfig() Config {
	return Config{Aggressiveness: 0.5, Reversible: true}
}

// Document 表示一个可压缩的文档。
type Document struct {
	PageContent string
	Metadata    map[string]any
}

// DocumentCompressor 使用 headroom 压缩文档。
type DocumentCompressor struct {
	cfg Config
}

// NewDocumentCompressor 创建 DocumentCompressor。
func NewDocumentCompressor(cfg Config) *DocumentCompressor {
	return &DocumentCompressor{cfg: cfg}
}

// CompressDocuments 压缩文档列表。
func (c *DocumentCompressor) CompressDocuments(ctx context.Context, docs []Document) ([]Document, error) {
	opts := headroom.DefaultOptions()
	opts.Aggressiveness = c.cfg.Aggressiveness
	opts.Reversible = c.cfg.Reversible
	opts.EnablePipeline = c.cfg.EnablePipeline

	result := make([]Document, len(docs))
	for i, doc := range docs {
		compressed, err := headroom.CompressString(doc.PageContent, opts)
		if err != nil {
			return nil, err
		}
		result[i] = Document{
			PageContent: compressed,
			Metadata:    doc.Metadata,
		}
	}
	return result, nil
}

// CompressDocument 压缩单个文档。
func (c *DocumentCompressor) CompressDocument(ctx context.Context, doc Document) (Document, error) {
	docs, err := c.CompressDocuments(ctx, []Document{doc})
	if err != nil {
		return Document{}, err
	}
	return docs[0], nil
}
