package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	headroom "github.com/superops-team/headroom-go"
)

// StatsTracker 跟踪会话压缩统计，持有共享 CCR 实例。
type StatsTracker struct {
	mu                    sync.Mutex
	ccr                   *headroom.CCR
	totalCompressions     int64
	totalOriginalTokens   int64
	totalCompressedTokens int64
}

func newStatsTracker() *StatsTracker {
	return &StatsTracker{
		ccr: headroom.NewCCR(headroom.CCRConfig{}),
	}
}

func (s *StatsTracker) record(orig, comp int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.totalCompressions++
	s.totalOriginalTokens += int64(orig)
	s.totalCompressedTokens += int64(comp)
}

func (s *StatsTracker) snapshot() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	avg := 0.0
	if s.totalOriginalTokens > 0 {
		avg = float64(s.totalOriginalTokens-s.totalCompressedTokens) / float64(s.totalOriginalTokens) * 100
	}
	entries, bytes := s.ccr.Stats()
	return map[string]any{
		"total_compressions":      s.totalCompressions,
		"total_original_tokens":   s.totalOriginalTokens,
		"total_compressed_tokens": s.totalCompressedTokens,
		"avg_savings":             avg,
		"cache_entries":           entries,
		"cache_bytes":             bytes,
	}
}

func (s *StatsTracker) retrieve(id string) (string, bool) {
	return s.ccr.Retrieve(id)
}

// ── 工具实现 ────────────────────────────────────────────────────────────────

func handleToolCall(params callToolParams, stats *StatsTracker) callToolResult {
	switch params.Name {
	case "headroom_compress":
		return handleCompress(params.Arguments, stats)
	case "headroom_retrieve":
		return handleRetrieve(params.Arguments, stats)
	case "headroom_stats":
		return handleStats(stats)
	case "headroom_read":
		return handleRead(params.Arguments, stats)
	default:
		return callToolResult{
			Content: textContent(fmt.Sprintf("Unknown tool: %s", params.Name)),
			IsError: true,
		}
	}
}

func handleCompress(args json.RawMessage, stats *StatsTracker) callToolResult {
	var in struct {
		Content        string  `json:"content"`
		Aggressiveness float64 `json:"aggressiveness"`
		Reversible     *bool   `json:"reversible"`
		ContentKind    string  `json:"content_kind"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return callToolResult{Content: textContent(fmt.Sprintf("Invalid arguments: %v", err)), IsError: true}
	}
	if in.Content == "" {
		return callToolResult{Content: textContent("content is required"), IsError: true}
	}

	opts := headroom.DefaultOptions()
	if in.Aggressiveness > 0 {
		opts.Aggressiveness = in.Aggressiveness
	}
	if in.Reversible != nil {
		opts.Reversible = *in.Reversible
	}

	result, err := headroom.CompressString(in.Content, opts)
	if err != nil {
		return callToolResult{Content: textContent(fmt.Sprintf("Compression failed: %v", err)), IsError: true}
	}

	origTokens := estimateTokens(in.Content)
	compTokens := estimateTokens(result)
	stats.record(origTokens, compTokens)

	savings := 0.0
	if origTokens > 0 {
		savings = float64(origTokens-compTokens) / float64(origTokens) * 100
	}

	output := fmt.Sprintf(
		"Compressed (%d → %d tokens, %.0f%% saved):\n%s",
		origTokens, compTokens, savings, result,
	)
	return callToolResult{Content: textContent(output)}
}

func handleRetrieve(args json.RawMessage, stats *StatsTracker) callToolResult {
	var in struct {
		RetrieveID string `json:"retrieve_id"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return callToolResult{Content: textContent(fmt.Sprintf("Invalid arguments: %v", err)), IsError: true}
	}
	if in.RetrieveID == "" {
		return callToolResult{Content: textContent("retrieve_id is required"), IsError: true}
	}

	original, found := stats.retrieve(in.RetrieveID)
	if !found {
		return callToolResult{Content: textContent(fmt.Sprintf("Content not found for ID: %s", in.RetrieveID)), IsError: true}
	}

	return callToolResult{Content: textContent(original)}
}

func handleStats(stats *StatsTracker) callToolResult {
	s := stats.snapshot()
	return callToolResult{Content: textContent(prettyStats(s))}
}

func handleRead(args json.RawMessage, stats *StatsTracker) callToolResult {
	// 安全检查：仅在 HEADROOM_MCP_READ=on 时启用文件读取
	if os.Getenv("HEADROOM_MCP_READ") != "on" {
		return callToolResult{
			Content: textContent("headroom_read is disabled. Set HEADROOM_MCP_READ=on to enable file reading."),
			IsError: true,
		}
	}

	var in struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return callToolResult{Content: textContent(fmt.Sprintf("Invalid arguments: %v", err)), IsError: true}
	}
	if in.Path == "" {
		return callToolResult{Content: textContent("path is required"), IsError: true}
	}

	data, err := os.ReadFile(in.Path)
	if err != nil {
		return callToolResult{Content: textContent(fmt.Sprintf("Failed to read file: %v", err)), IsError: true}
	}

	content := string(data)
	opts := headroom.DefaultOptions()
	compressed, err := headroom.CompressString(content, opts)
	if err != nil {
		origTokens := estimateTokens(content)
		stats.record(origTokens, origTokens)
		return callToolResult{Content: textContent(content)}
	}

	origTokens := estimateTokens(content)
	compTokens := estimateTokens(compressed)
	stats.record(origTokens, compTokens)

	savings := 0.0
	if origTokens > 0 {
		savings = float64(origTokens-compTokens) / float64(origTokens) * 100
	}

	output := fmt.Sprintf(
		"File: %s (%d tokens)\nCompressed (%d tokens, %.0f%% saved):\n%s",
		in.Path, origTokens, compTokens, savings, compressed,
	)
	return callToolResult{Content: textContent(output)}
}

func estimateTokens(s string) int {
	return len(strings.Fields(s)) + len(s)/4
}

// ── Server ──────────────────────────────────────────────────────────────────

// Serve 启动 MCP Server（stdio 模式）。
func Serve() error {
	stats := newStatsTracker()
	transport := newStdioTransport()
	defer transport.Close()

	req, err := transport.Read()
	if err != nil {
		return fmt.Errorf("read initialize: %w", err)
	}

	if req.Method != "initialize" {
		transport.Write(newError(req.ID, -32601, fmt.Sprintf("expected initialize, got %s", req.Method)))
		return fmt.Errorf("expected initialize, got %s", req.Method)
	}

	initResult := initializeResult{
		ProtocolVersion: protocolVersion,
		ServerInfo:      serverInfo{Name: serverName, Version: serverVersion},
		Capabilities: capabilities{
			Tools: &toolsCapability{ListChanged: false},
		},
	}
	transport.Write(newResult(req.ID, initResult))

	transport.Write(jsonRPCResponse{
		JSONRPC: "2.0",
		Result:  struct{}{},
	})

	for {
		req, err := transport.Read()
		if err != nil {
			if err.Error() == "EOF" {
				return nil
			}
			return fmt.Errorf("read: %w", err)
		}

		switch req.Method {
		case "tools/list":
			result := listToolsResult{Tools: buildTools()}
			transport.Write(newResult(req.ID, result))

		case "tools/call":
			var params callToolParams
			if err := json.Unmarshal(req.Params, &params); err != nil {
				transport.Write(newError(req.ID, -32602, fmt.Sprintf("invalid params: %v", err)))
				continue
			}
			result := handleToolCall(params, stats)
			transport.Write(newResult(req.ID, result))

		case "notifications/initialized":

		default:
			transport.Write(newError(req.ID, -32601, fmt.Sprintf("unknown method: %s", req.Method)))
		}
	}
}

// ── stdio 传输 ──────────────────────────────────────────────────────────────

type stdioTransport struct {
	encoder *json.Encoder
	decoder *json.Decoder
}

func newStdioTransport() *stdioTransport {
	return &stdioTransport{
		encoder: json.NewEncoder(os.Stdout),
		decoder: json.NewDecoder(os.Stdin),
	}
}

func (t *stdioTransport) Read() (jsonRPCRequest, error) {
	var req jsonRPCRequest
	if err := t.decoder.Decode(&req); err != nil {
		return req, err
	}
	return req, nil
}

func (t *stdioTransport) Write(resp jsonRPCResponse) {
	_ = t.encoder.Encode(resp)
}

func (t *stdioTransport) Close() {}

// Version 返回 MCP Server 版本。
func Version() string {
	return serverVersion
}

var _ = time.Now
