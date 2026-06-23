// Package mcp 实现 headroom-go 的 MCP (Model Context Protocol) Server。
//
// 通过 stdio JSON-RPC 2.0 传输，提供 4 个压缩相关工具：
//   - headroom_compress: 压缩文本内容
//   - headroom_retrieve: 通过 ID 检索原始内容
//   - headroom_stats: 获取压缩统计
//   - headroom_read: 通过 CCR 缓存读取文件
//
// 兼容 MCP 规范 2024-11-05。
package mcp

import (
	"encoding/json"
	"fmt"
)

const (
	protocolVersion = "2024-11-05"
	serverName      = "headroom-mcp"
	serverVersion   = "0.7.0"
)

// ── JSON-RPC 类型 ───────────────────────────────────────────────────────────

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ── MCP 协议类型 ────────────────────────────────────────────────────────────

type initializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	ServerInfo      serverInfo   `json:"serverInfo"`
	Capabilities    capabilities `json:"capabilities"`
}

type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type capabilities struct {
	Tools *toolsCapability `json:"tools,omitempty"`
}

type toolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

type listToolsResult struct {
	Tools []tool `json:"tools"`
}

type tool struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	InputSchema inputSchema `json:"inputSchema"`
}

type inputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]property `json:"properties"`
	Required   []string            `json:"required,omitempty"`
}

type property struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Default     any      `json:"default,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

type callToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type callToolResult struct {
	Content []contentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type contentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ── 工具定义 ────────────────────────────────────────────────────────────────

func buildTools() []tool {
	return []tool{
		{
			Name:        "headroom_compress",
			Description: "Compress text content to reduce token usage. Auto-detects content type (JSON/code/text/diff/log/search/tabular/spreadsheet/HTML) and applies specialized compression.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"content":        {Type: "string", Description: "Text content to compress"},
					"aggressiveness": {Type: "number", Description: "Compression strength 0.0-1.0", Default: 0.5},
					"reversible":     {Type: "boolean", Description: "Enable reversible compression (attach retrieval ID)", Default: true},
					"content_kind":   {Type: "string", Description: "Content type hint (auto for detection)", Default: "auto", Enum: []string{"auto", "text", "json", "code", "diff", "log", "search", "tabular", "spreadsheet", "html"}},
				},
				Required: []string{"content"},
			},
		},
		{
			Name:        "headroom_retrieve",
			Description: "Retrieve original content from a compression retrieval ID (format: v3_{sha256[:12]}). Only works for content compressed with reversible=true.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"retrieve_id": {Type: "string", Description: "The retrieval ID from compressed output (e.g., v3_a1b2c3d4e5f6)"},
				},
				Required: []string{"retrieve_id"},
			},
		},
		{
			Name:        "headroom_stats",
			Description: "Get session compression statistics including total compressions, token savings, and cache usage.",
			InputSchema: inputSchema{
				Type:       "object",
				Properties: map[string]property{},
			},
		},
		{
			Name:        "headroom_read",
			Description: "Read and compress a file through the CCR cache. Useful for reading large files with automatic compression.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"path": {Type: "string", Description: "Absolute path to the file to read and compress"},
				},
				Required: []string{"path"},
			},
		},
	}
}

// ── 响应辅助 ────────────────────────────────────────────────────────────────

func newResult(id any, result any) jsonRPCResponse {
	return jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: result}
}

func newError(id any, code int, message string) jsonRPCResponse {
	return jsonRPCResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: message}}
}

func textContent(text string) []contentItem {
	return []contentItem{{Type: "text", Text: text}}
}

func toJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func prettyStats(stats map[string]any) string {
	return fmt.Sprintf(
		"Headroom Compression Statistics\n"+
			"  Total compressions: %v\n"+
			"  Total original tokens: %v\n"+
			"  Total compressed tokens: %v\n"+
			"  Average savings: %.1f%%\n"+
			"  Cache entries: %v\n"+
			"  Cache size: %v bytes",
		stats["total_compressions"],
		stats["total_original_tokens"],
		stats["total_compressed_tokens"],
		stats["avg_savings"],
		stats["cache_entries"],
		stats["cache_bytes"],
	)
}
