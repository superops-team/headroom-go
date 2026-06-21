package proxy

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	headroom "github.com/superops-team/headroom-go"
)

type errorReadCloser struct{}

func (errorReadCloser) Read([]byte) (int, error) { return 0, errors.New("bad \"json\"\n错误") }
func (errorReadCloser) Close() error             { return nil }

type failingRoundTripper struct{}

func (failingRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("upstream \"offline\"\n错误")
}

func assertJSONError(t *testing.T, resp *http.Response, wantStatus int, wantContains string) {
	t.Helper()
	if resp.StatusCode != wantStatus {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d want=%d body=%s", resp.StatusCode, wantStatus, data)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("Content-Type %q should contain application/json", ct)
	}
	var body struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("error body is not valid JSON: %v", err)
	}
	if !strings.Contains(body.Error, wantContains) {
		t.Fatalf("error %q should contain %q", body.Error, wantContains)
	}
}

func TestWriteErrorEscapesSpecialCharacters(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, http.StatusBadRequest, "bad \"json\"\n错误")
	resp := rec.Result()
	defer resp.Body.Close()
	assertJSONError(t, resp, http.StatusBadRequest, "bad \"json\"\n错误")
}

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
	assertJSONError(t, resp, http.StatusBadRequest, "streaming not supported")
}

func TestProxy_SpecDStreamStringRejected(t *testing.T) {
	p := NewProxy(Config{CompressOptions: headroom.DefaultOptions()})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"stream":"true","messages":[{"role":"user","content":"hi"}]}`))
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)
	resp := rec.Result()
	defer resp.Body.Close()
	assertJSONError(t, resp, http.StatusBadRequest, "streaming not supported")
}

func TestProxy_SpecDInvalidMessagesReturnsJSON(t *testing.T) {
	p := NewProxy(Config{CompressOptions: headroom.DefaultOptions()})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"messages":"not-an-array"}`))
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)
	resp := rec.Result()
	defer resp.Body.Close()
	assertJSONError(t, resp, http.StatusBadRequest, "invalid messages")
}

func TestProxy_SpecDMissingMessagesForwardsOriginalBody(t *testing.T) {
	var received string
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received = string(body)
		w.WriteHeader(http.StatusAccepted)
		io.WriteString(w, `{"ok":true}`)
	}))
	defer mock.Close()
	p := NewProxy(Config{UpstreamBaseURL: mock.URL, CompressOptions: headroom.DefaultOptions()})
	original := `{"model":"gpt-4","stream":1}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(original))
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)
	resp := rec.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status=%d want %d", resp.StatusCode, http.StatusAccepted)
	}
	if received != original {
		t.Fatalf("forwarded body=%q want %q", received, original)
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
	assertJSONError(t, resp, http.StatusBadRequest, "invalid json")
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
	assertJSONError(t, resp, http.StatusMethodNotAllowed, "method not allowed")
}

func TestProxy_ReadBodyErrorReturnsJSON(t *testing.T) {
	p := NewProxy(Config{CompressOptions: headroom.DefaultOptions()})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", errorReadCloser{})
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)
	resp := rec.Result()
	defer resp.Body.Close()
	assertJSONError(t, resp, http.StatusBadRequest, "bad \"json\"\n错误")
}

func TestProxy_CompressionFailedReturnsJSON(t *testing.T) {
	opts := headroom.DefaultOptions()
	opts.TokenizerConfig = headroom.TokenizerConfig{Backend: "missing", AllowFallback: false}
	p := NewProxy(Config{CompressOptions: opts})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"messages":[{"role":"user","content":"hi"}]}`))
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)
	resp := rec.Result()
	defer resp.Body.Close()
	assertJSONError(t, resp, http.StatusInternalServerError, "compression failed")
}

func TestProxy_SpecDUnavailableTokenizerWithoutFallbackReturnsJSON(t *testing.T) {
	opts := headroom.DefaultOptions()
	opts.TokenizerConfig = headroom.TokenizerConfig{Backend: headroom.TokenizerTiktoken, AllowFallback: false}
	p := NewProxy(Config{CompressOptions: opts})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"messages":[{"role":"user","content":"hello"}]}`))
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)
	resp := rec.Result()
	defer resp.Body.Close()
	assertJSONError(t, resp, http.StatusInternalServerError, "tokenizer backend not implemented")
}

func TestProxy_SpecDOversizedRequestBodyIsRejectedAsBadJSON(t *testing.T) {
	p := NewProxy(Config{CompressOptions: headroom.DefaultOptions()})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(strings.Repeat("{", maxRequestBodyBytes+1024)))
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)
	resp := rec.Result()
	defer resp.Body.Close()
	assertJSONError(t, resp, http.StatusBadRequest, "invalid json")
}

func TestProxy_SpecDUpstreamNon200StatusPassthrough(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Upstream-Reason", "rate-limited")
		w.WriteHeader(http.StatusTooManyRequests)
		io.WriteString(w, `{"error":"rate limit"}`)
	}))
	defer mock.Close()
	p := NewProxy(Config{UpstreamBaseURL: mock.URL, CompressOptions: headroom.Options{Aggressiveness: 0.5, Reversible: false}})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"messages":[{"role":"user","content":"hello"}]}`))
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)
	resp := rec.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("status=%d want %d", resp.StatusCode, http.StatusTooManyRequests)
	}
	if resp.Header.Get("X-Upstream-Reason") != "rate-limited" {
		t.Fatalf("upstream header was not passed through: %#v", resp.Header)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "rate limit") {
		t.Fatalf("upstream body was not passed through: %s", body)
	}
}

func TestProxy_UpstreamRequestFailedReturnsJSON(t *testing.T) {
	p := NewProxy(Config{UpstreamBaseURL: "http://[::1", CompressOptions: headroom.DefaultOptions()})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4"}`))
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)
	resp := rec.Result()
	defer resp.Body.Close()
	assertJSONError(t, resp, http.StatusBadGateway, "upstream request failed")
}

func TestProxy_UpstreamUnreachableReturnsJSON(t *testing.T) {
	client := &http.Client{Transport: failingRoundTripper{}}
	p := NewProxy(Config{UpstreamBaseURL: "http://example.invalid/v1", CompressOptions: headroom.DefaultOptions(), HTTPClient: client})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4"}`))
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)
	resp := rec.Result()
	defer resp.Body.Close()
	assertJSONError(t, resp, http.StatusBadGateway, "upstream unreachable")
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
