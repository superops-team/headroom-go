package proxy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	headroom "github.com/superops-team/headroom-go"
)

// newUpstreamMock 模拟一个上游 LLM API server（供测试使用）
func newUpstreamMock() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"test response"}}]}`)
	}))
}

// /healthz 返回 200 OK
func TestProxy_Healthz(t *testing.T) {
	mock := newUpstreamMock()
	defer mock.Close()

	p := NewProxy(Config{
		UpstreamBaseURL: mock.URL,
		CompressOptions: headroom.DefaultOptions(),
	})
	server := httptest.NewServer(p)
	defer server.Close()

	resp, err := http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("got status %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"status":"ok"`) {
		t.Errorf("unexpected body: %s", body)
	}
}

// POST /v1/chat/completions 转发到上游
func TestProxy_ChatCompletion(t *testing.T) {
	mock := newUpstreamMock()
	defer mock.Close()

	p := NewProxy(Config{
		UpstreamBaseURL: mock.URL,
		CompressOptions: headroom.Options{Aggressiveness: 0.5, Reversible: false},
	})
	server := httptest.NewServer(p)
	defer server.Close()

	// 用足够长的 messages 让压缩真实发生
	longContent := strings.Repeat("[INFO] heartbeat OK latency=12ms\n", 50)
	payload := map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]string{
			{"role": "system", "content": "You are a helpful assistant. Please be careful with the data and focus on the important details of the incoming request."},
			{"role": "user", "content": longContent},
		},
	}
	requestBody, _ := json.Marshal(payload)

	resp, err := http.Post(server.URL+"/v1/chat/completions", "application/json",
		strings.NewReader(string(requestBody)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("got status %d", resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		t.Logf("response body: %s", body)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "assistant") {
		t.Errorf("unexpected upstream response: %s", body)
	}
}

// stream: true → 返回 400
func TestProxy_StreamRejected(t *testing.T) {
	mock := newUpstreamMock()
	defer mock.Close()

	p := NewProxy(Config{
		UpstreamBaseURL: mock.URL,
		CompressOptions: headroom.DefaultOptions(),
	})
	server := httptest.NewServer(p)
	defer server.Close()

	resp, err := http.Post(server.URL+"/v1/chat/completions", "application/json",
		strings.NewReader(`{"model":"gpt-4","stream":true,"messages":[{"role":"user","content":"hi"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("got status %d, want 400", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "streaming not supported") {
		t.Errorf("unexpected body: %s", body)
	}
}

// 无效 JSON → 400
func TestProxy_InvalidJSON(t *testing.T) {
	mock := newUpstreamMock()
	defer mock.Close()

	p := NewProxy(Config{
		UpstreamBaseURL: mock.URL,
		CompressOptions: headroom.DefaultOptions(),
	})
	server := httptest.NewServer(p)
	defer server.Close()

	resp, err := http.Post(server.URL+"/v1/chat/completions", "application/json",
		strings.NewReader(`invalid json`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("got status %d, want 400", resp.StatusCode)
	}
}

// GET 其他路径 → 405 method not allowed
func TestProxy_MethodNotAllowed(t *testing.T) {
	mock := newUpstreamMock()
	defer mock.Close()

	p := NewProxy(Config{
		UpstreamBaseURL: mock.URL,
		CompressOptions: headroom.DefaultOptions(),
	})
	server := httptest.NewServer(p)
	defer server.Close()

	resp, err := http.Get(server.URL + "/v1/chat/completions")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 405 {
		t.Errorf("got status %d, want 405", resp.StatusCode)
	}
}

// 请求内容被真正压缩（验证：上游收到的 JSON 中 content 字段比原始短）
func TestProxy_ContentActuallyCompressed(t *testing.T) {
	var receivedBody string
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)
		io.WriteString(w, `{"choices":[]}`)
	}))
	defer mock.Close()

	p := NewProxy(Config{
		UpstreamBaseURL: mock.URL,
		CompressOptions: headroom.Options{Aggressiveness: 0.5, Reversible: false},
	})
	server := httptest.NewServer(p)
	defer server.Close()

	longContent := strings.Repeat("[INFO] service=api user=test status=ok latency=12ms\n", 50)
	requestBody := `{"messages":[{"role":"user","content":` + jsonEscape(longContent) + `}]}`

	_, err := http.Post(server.URL+"/v1/chat/completions", "application/json",
		strings.NewReader(requestBody))
	if err != nil {
		t.Fatal(err)
	}

	if len(receivedBody) >= len(requestBody) {
		t.Errorf("received body %d should be shorter than original %d", len(receivedBody), len(requestBody))
	}
	t.Logf("original %d bytes, forwarded %d bytes (%.1f%% savings)",
		len(requestBody), len(receivedBody), 100.0*float64(len(requestBody)-len(receivedBody))/float64(len(requestBody)))
}

// 便捷函数：JSON 字符串转义
func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
