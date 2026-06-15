// Package proxy 提供 headroom 的 HTTP 代理层，兼容 OpenAI Chat Completions API。
//
// 代理将请求中的 messages 压缩后转发到上游 LLM API，降低 token 成本。
package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	headroom "github.com/superops-team/headroom-go"
)

// maxRequestBodyBytes 是代理可读取请求体的最大字节数（50MB）
const maxRequestBodyBytes = 50 * 1024 * 1024

// Config 配置 headroom HTTP 代理。
type Config struct {
	UpstreamBaseURL string          // 上游 API Base URL（例如 https://api.openai.com/v1）
	APIKey          string          // 上游 API key（通过 Authorization 头转发）
	ListenAddr      string          // 监听地址（默认 ":8787"）
	CompressOptions headroom.Options // 压缩选项
}

// NewProxy 创建一个 headroom HTTP Proxy。
// 支持：POST /v1/chat/completions（压缩后转发），GET /healthz
func NewProxy(cfg Config) http.Handler {
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":8787"
	}
	if cfg.UpstreamBaseURL == "" {
		cfg.UpstreamBaseURL = "https://api.openai.com/v1"
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"status":"ok"}`)
	})

	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodyBytes))
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		// 解析为通用 JSON（保留未知字段以便转发）
		var payload map[string]interface{}
		if err := json.Unmarshal(body, &payload); err != nil {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, fmt.Sprintf(`{"error":"invalid json: %s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		// 提取 messages
		messagesRaw, ok := payload["messages"]
		if !ok {
			// 没有 messages，原样转发
			forwardToUpstream(w, r, cfg, body)
			return
		}

		// 检查 stream: true → 拒绝流式（v0.1）
		// 兼容 bool 与 string 两种 JSON 值（上游 API 可能有不同表示）
		isStream := false
		switch s := payload["stream"].(type) {
		case bool:
			isStream = s
		case string:
			if s == "true" || s == "True" || s == "1" {
				isStream = true
			}
		case nil:
			// 不存在，不处理
		default:
			// 其他类型（如 float64=1）：安全起见，不视为流式
		}
		if isStream {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, `{"error":"streaming not supported in v0.1"}`)
			return
		}

		// 将 messages 反序列化为 []headroom.Message
		messagesJSON, err := json.Marshal(messagesRaw)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		var msgs []headroom.Message
		if err := json.Unmarshal(messagesJSON, &msgs); err != nil {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, fmt.Sprintf(`{"error":"invalid messages: %s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		// 压缩 messages
		compressed, err := headroom.Compress(msgs, cfg.CompressOptions)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, fmt.Sprintf(`{"error":"compression failed: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		// 替换 messages 字段并重新序列化
		payload["messages"] = compressed.Messages
		newBody, err := json.Marshal(payload)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		forwardToUpstream(w, r, cfg, newBody)
	})

	return mux
}

func forwardToUpstream(w http.ResponseWriter, r *http.Request, cfg Config, body []byte) {
	url := strings.TrimRight(cfg.UpstreamBaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, url,
		bytes.NewReader(body))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, fmt.Sprintf(`{"error":"upstream request failed: %s"}`, err.Error()), http.StatusBadGateway)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	// 转发客户端的 Authorization（或回落到配置的 APIKey）
	if auth := r.Header.Get("Authorization"); auth != "" {
		req.Header.Set("Authorization", auth)
	} else if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, fmt.Sprintf(`{"error":"upstream unreachable: %s"}`, err.Error()), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 透传响应头
	for k, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
