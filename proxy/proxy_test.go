package proxy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	if !strings.Contains(string(body), `"version":"`+headroom.Version+`"`) {
		t.Errorf("version missing in body: %s", body)
	}
	if !strings.Contains(string(body), `"uptime"`) {
		t.Errorf("uptime missing in body: %s", body)
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
	var receivedPayload map[string]interface{}
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)
		_ = json.Unmarshal(body, &receivedPayload)
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
	messages, ok := receivedPayload["messages"].([]interface{})
	if !ok || len(messages) != 1 {
		t.Fatalf("upstream did not receive one messages array: %#v", receivedPayload)
	}
	msg, ok := messages[0].(map[string]interface{})
	if !ok {
		t.Fatalf("upstream message malformed: %#v", messages[0])
	}
	content, _ := msg["content"].(string)
	if content == "" || len(content) >= len(longContent) {
		t.Fatalf("upstream content was not compressed: %d >= %d content=%q", len(content), len(longContent), content)
	}
	t.Logf("original %d bytes, forwarded %d bytes (%.1f%% savings)",
		len(requestBody), len(receivedBody), 100.0*float64(len(requestBody)-len(receivedBody))/float64(len(requestBody)))
}

func TestProxy_E2ENonStreamingCompressesAndForwards(t *testing.T) {
	var receivedBody []byte
	upstreamErr := make(chan string, 1)
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			upstreamErr <- "unexpected upstream path: " + r.URL.Path
			http.Error(w, "bad path", http.StatusBadRequest)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			upstreamErr <- "authorization not forwarded: " + r.Header.Get("Authorization")
			http.Error(w, "bad auth", http.StatusUnauthorized)
			return
		}
		receivedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`)
	}))
	defer mock.Close()

	opts := headroom.DefaultOptions()
	opts.EnablePipeline = true
	opts.Reversible = false
	p := NewProxy(Config{UpstreamBaseURL: mock.URL + "/v1", APIKey: "test-key", CompressOptions: opts})
	server := httptest.NewServer(p)
	defer server.Close()

	longContent := strings.Repeat("[INFO] service=api heartbeat ok\n", 100)
	body := `{"model":"gpt-4","stream":false,"messages":[{"role":"user","content":` + jsonEscape(longContent) + `}]}`
	resp, err := http.Post(server.URL+"/v1/chat/completions", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, data)
	}
	select {
	case msg := <-upstreamErr:
		t.Fatal(msg)
	default:
	}
	var forwarded map[string]interface{}
	if err := json.Unmarshal(receivedBody, &forwarded); err != nil {
		t.Fatalf("forwarded body invalid JSON: %v", err)
	}
	messages := forwarded["messages"].([]interface{})
	content := messages[0].(map[string]interface{})["content"].(string)
	if len(content) >= len(longContent) {
		t.Fatalf("expected compressed forwarded content, got %d >= %d", len(content), len(longContent))
	}
}

// 上游超时 → proxy 返回 502
func TestProxy_UpstreamTimeout(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		io.WriteString(w, `{"choices":[]}`)
	}))
	defer mock.Close()

	// 注入短超时 HTTP Client，模拟上游超时
	shortClient := &http.Client{Timeout: 100 * time.Millisecond}
	p := NewProxy(Config{
		UpstreamBaseURL: mock.URL,
		CompressOptions: headroom.Options{Aggressiveness: 0.5, Reversible: false},
		HTTPClient:      shortClient,
	})
	server := httptest.NewServer(p)
	defer server.Close()

	// 测试客户端用长超时，确保是 proxy 内部超时触发
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(server.URL+"/v1/chat/completions", "application/json",
		strings.NewReader(`{"messages":[{"role":"user","content":"hi"}]}`))
	if err != nil {
		t.Fatalf("unexpected client error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("got status %d, want 502 Bad Gateway", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "upstream") {
		t.Errorf("expected upstream error in body, got: %s", body)
	}
}

// 便捷函数：JSON 字符串转义
func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// Request ID：不带 X-Request-ID 时自动生成
func TestProxy_RequestID_AutoGenerate(t *testing.T) {
	mock := newUpstreamMock()
	defer mock.Close()

	p := NewProxy(Config{
		UpstreamBaseURL: mock.URL,
		CompressOptions: headroom.Options{Aggressiveness: 0.5, Reversible: false},
	})
	server := httptest.NewServer(p)
	defer server.Close()

	resp, err := http.Post(server.URL+"/v1/chat/completions", "application/json",
		strings.NewReader(`{"messages":[{"role":"user","content":"hi"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	reqID := resp.Header.Get("X-Request-ID")
	if reqID == "" {
		t.Error("X-Request-ID header should be present")
	}
}

// Request ID：带 X-Request-ID 时透传
func TestProxy_RequestID_PassThrough(t *testing.T) {
	mock := newUpstreamMock()
	defer mock.Close()

	p := NewProxy(Config{
		UpstreamBaseURL: mock.URL,
		CompressOptions: headroom.Options{Aggressiveness: 0.5, Reversible: false},
	})
	server := httptest.NewServer(p)
	defer server.Close()

	req, _ := http.NewRequest("POST", server.URL+"/v1/chat/completions",
		strings.NewReader(`{"messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "my-custom-id")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	reqID := resp.Header.Get("X-Request-ID")
	if reqID != "my-custom-id" {
		t.Errorf("got X-Request-ID=%q, want %q", reqID, "my-custom-id")
	}
}
