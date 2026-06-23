// Package headroom 提供 Ollama 的透明压缩中间件。
//
// 使用方式：
//
//	import headroom "github.com/superops-team/headroom-go/integrations/ollama"
//
//	transport := headroom.NewTransport(http.DefaultTransport, headroom.Config{
//	    Aggressiveness: 0.5,
//	})
//	client := &http.Client{Transport: transport}
package headroom

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

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

// NewTransport 创建带压缩的 HTTP Transport。
func NewTransport(base http.RoundTripper, cfg Config) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &compressionTransport{base: base, cfg: cfg}
}

type compressionTransport struct {
	base http.RoundTripper
	cfg  Config
}

func (t *compressionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != "POST" || !isOllamaChatPath(req.URL.Path) {
		return t.base.RoundTrip(req)
	}

	body, err := io.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		return nil, err
	}

	compressed, err := compressOllamaBody(body, t.cfg)
	if err != nil {
		req.Body = io.NopCloser(bytes.NewReader(body))
	} else {
		req.Body = io.NopCloser(bytes.NewReader(compressed))
		req.ContentLength = int64(len(compressed))
	}

	return t.base.RoundTrip(req)
}

func isOllamaChatPath(path string) bool {
	return path == "/api/chat" || path == "/api/generate"
}

func compressOllamaBody(body []byte, cfg Config) ([]byte, error) {
	var req ollamaChatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}

	opts := headroom.DefaultOptions()
	opts.Aggressiveness = cfg.Aggressiveness
	opts.Reversible = cfg.Reversible
	opts.EnablePipeline = cfg.EnablePipeline

	compressed, err := headroom.CompressString(req.Prompt, opts)
	if err != nil {
		return nil, err
	}

	req.Prompt = compressed

	// 也压缩 messages（如果有）
	for i, m := range req.Messages {
		c, err := headroom.CompressString(m.Content, opts)
		if err == nil {
			req.Messages[i].Content = c
		}
	}

	return json.Marshal(req)
}

type ollamaChatRequest struct {
	Model    string           `json:"model"`
	Prompt   string           `json:"prompt,omitempty"`
	Messages []ollamaMessage  `json:"messages,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
