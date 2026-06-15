package headroom

import (
	"strconv"
	"strings"
	"testing"
)

// 混合消息：JSON + 代码 + 文本 + assistant 透传
func TestCompress_MixedMessages(t *testing.T) {
	// 构建长消息：JSON + 代码 + 文本，确保总长度足够触发真正压缩
	var longLogBuilder strings.Builder
	for i := 0; i < 50; i++ {
		longLogBuilder.WriteString("[INFO] service=api user=user" + strconv.Itoa(i) + " status=ok latency=12ms\n")
	}
	longLog := longLogBuilder.String()

	longCode := "def process_data(data):\n    if err != nil {\n        return nil, err\n    }\n    result = []\n    for i in range(100):\n        result.append(i * 2)\n    x = 1 + 1\n    y = 2 + 2\n    z = 3 + 3\n    w = 4 + 4\n    return result\n"

	longJSON := `{"items":[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15],"metadata":{"tags":["a","b","c","d","e","f","g","h","i"],"null":null,"empty":{}}}`

	msgs := []Message{
		{Role: "system", Content: "You are a helpful assistant. Please review the following data carefully. Focus on the important parts and ignore the filler text that repeats multiple times without adding value to the analysis."},
		{Role: "user", Content: longJSON},
		{Role: "assistant", Content: "I have processed the data successfully and prepared the analysis."},
		{Role: "user", Content: longCode},
		{Role: "user", Content: longLog},
	}

	opts := DefaultOptions()
	opts.Reversible = false // 关闭可逆，直接看压缩收益

	result, err := Compress(msgs, opts)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Messages) != len(msgs) {
		t.Errorf("got %d messages, want %d", len(result.Messages), len(msgs))
	}

	// assistant 角色消息应原样透传
	if result.Messages[2].Content != "I have processed the data successfully and prepared the analysis." {
		t.Errorf("assistant message changed: got %q", result.Messages[2].Content)
	}

	// Savings 应为正值
	if result.Savings <= 0 {
		t.Errorf("savings got %f, want >0", result.Savings)
	}

	t.Logf("Original: %d tokens, Compressed: %d tokens, Savings: %.1f%%",
		result.OriginalTokens, result.CompressedTokens, result.Savings*100)
}

// 可逆压缩模式：压缩输出中包含 [headroom:retrieve id=...]
func TestCompress_Reversible(t *testing.T) {
	// 用长消息，确保压缩收益 > retrieve id 开销
	var sb strings.Builder
	for i := 0; i < 50; i++ {
		sb.WriteString("[INFO] heartbeat OK service=api status=running latency=12ms\n")
	}
	msgs := []Message{
		{Role: "user", Content: sb.String()},
	}
	opts := DefaultOptions()
	opts.Reversible = true

	result, err := Compress(msgs, opts)
	if err != nil {
		t.Fatal(err)
	}
	content := result.Messages[0].Content
	if !strings.Contains(content, "headroom:retrieve id=v2_") {
		t.Errorf("reversible output missing retrieve marker, got: %s", content[:100])
	}
}

// 不可逆压缩模式：不包含 retrieve 标记
func TestCompress_NonReversible(t *testing.T) {
	msgs := []Message{
		{Role: "user", Content: "some content that will be compressed"},
	}
	opts := DefaultOptions()
	opts.Reversible = false

	result, err := Compress(msgs, opts)
	if err != nil {
		t.Fatal(err)
	}
	content := result.Messages[0].Content
	if strings.Contains(content, "headroom:retrieve") {
		t.Errorf("non-reversible output has retrieve marker: %s", content)
	}
}

// TokenLimit：短消息不压缩
func TestCompress_TokenLimit(t *testing.T) {
	short := "hi"
	msgs := []Message{{Role: "user", Content: short}}
	opts := DefaultOptions()
	opts.TokenLimit = 100 // 需要 > 估算 tokens 才跳过
	opts.Reversible = false

	result, err := Compress(msgs, opts)
	if err != nil {
		t.Fatal(err)
	}
	if result.Messages[0].Content != short {
		t.Errorf("token-limit skipped message modified: got %q, want %q",
			result.Messages[0].Content, short)
	}
}

// CompressString：便捷方法
func TestCompressString(t *testing.T) {
	content := `{"a":1, "b":null, "data":[1,2,3,4,5,6,7,8,9,10]}`
	opts := DefaultOptions()
	opts.Reversible = false

	out, err := CompressString(content, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Error("empty output")
	}
	// 输出应比输入短（移除了 null，且数组被折叠）
	if len(out) >= len(content) {
		t.Errorf("output %d chars should be shorter than input %d chars", len(out), len(content))
	}
}

// 空消息数组：正常返回无错误
func TestCompress_Empty(t *testing.T) {
	result, err := Compress([]Message{}, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Messages) != 0 {
		t.Errorf("empty input should produce empty output, got %d messages", len(result.Messages))
	}
	if result.OriginalTokens != 0 || result.CompressedTokens != 0 {
		t.Errorf("empty token should both be 0, got %d/%d", result.OriginalTokens, result.CompressedTokens)
	}
}
