// Package headroom 提供 OpenAI Go SDK 的透明压缩集成。
//
// 通过包装 HTTP RoundTripper，对所有 ChatCompletion 请求自动压缩 messages。
//
// 使用方式：
//
//	import (
//	    "github.com/sashabaranov/go-openai"
//	    headroom "github.com/superops-team/headroom-go/integrations/go-openai"
//	)
//
//	client := openai.NewClient("sk-xxx")
//	client = headroom.WrapClient(client, headroom.Config{
//	    Aggressiveness: 0.5,
//	    Reversible:     true,
//	})
//	// 后续所有请求自动压缩
//	resp, _ := client.CreateChatCompletion(ctx, req)
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
	Aggressiveness float64 // 压缩强度 0.0-1.0
	Reversible     bool    // 启用可逆压缩
	AlignPrefix    bool    // 启用 KV Cache 前缀对齐
	EnablePipeline bool    // 使用 Pipeline 模式
}

// DefaultConfig 返回推荐配置。
func DefaultConfig() Config {
	return Config{
		Aggressiveness: 0.5,
		Reversible:     true,
	}
}

// WrapClient 包装 OpenAI client，透明压缩所有 ChatCompletion 请求。
func WrapClient(client *http.Client, cfg Config) *http.Client {
	if client == nil {
		client = &http.Client{}
	}
	original := client.Transport
	if original == nil {
		original = http.DefaultTransport
	}
	client.Transport = NewRoundTripper(original, cfg)
	return client
}

// NewRoundTripper 创建带压缩的 HTTP RoundTripper。
//
// 拦截 POST /chat/completions 请求，压缩 messages 后转发。
// 其他请求透传。
func NewRoundTripper(base http.RoundTripper, cfg Config) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &compressionRoundTripper{base: base, cfg: cfg}
}

type compressionRoundTripper struct {
	base http.RoundTripper
	cfg  Config
}

func (rt *compressionRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// 只处理 ChatCompletion 请求
	if req.Method != "POST" || !isChatCompletionsPath(req.URL.Path) {
		return rt.base.RoundTrip(req)
	}

	body, err := io.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		return nil, err
	}

	compressed, err := compressRequestBody(body, rt.cfg)
	if err != nil {
		// 压缩失败时透传原始请求
		req.Body = io.NopCloser(bytes.NewReader(body))
	} else {
		req.Body = io.NopCloser(bytes.NewReader(compressed))
		req.ContentLength = int64(len(compressed))
	}

	return rt.base.RoundTrip(req)
}

func isChatCompletionsPath(path string) bool {
	return len(path) > 0 && (path == "/chat/completions" ||
		path == "/v1/chat/completions" ||
		path[len(path)-len("/chat/completions"):] == "/chat/completions")
}

func compressRequestBody(body []byte, cfg Config) ([]byte, error) {
	var req chatCompletionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}

	opts := headroom.DefaultOptions()
	opts.Aggressiveness = cfg.Aggressiveness
	opts.Reversible = cfg.Reversible
	opts.AlignPrefix = cfg.AlignPrefix
	opts.EnablePipeline = cfg.EnablePipeline

	messages := make([]headroom.Message, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = headroom.Message{Role: m.Role, Content: m.Content, Name: m.Name}
	}

	result, err := headroom.Compress(messages, opts)
	if err != nil {
		return nil, err
	}

	req.Messages = make([]chatMessage, len(result.Messages))
	for i, m := range result.Messages {
		req.Messages[i] = chatMessage{Role: m.Role, Content: m.Content, Name: m.Name}
	}

	return json.Marshal(req)
}

type chatCompletionRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}
