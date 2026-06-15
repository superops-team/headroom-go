package headroom

import (
	"strconv"
	"strings"
	"testing"
)

// FATAL 行保留
func TestTextCompressor_PreserveFatal(t *testing.T) {
	cfg := TextConfig{Aggressiveness: 0.5}
	src := "[INFO] heartbeat OK\n[FATAL] disk full on /mnt/data\n[INFO] heartbeat OK\n"
	out := CompressText(src, cfg)
	if !strings.Contains(out, "[FATAL] disk full on /mnt/data") {
		t.Errorf("FATAL line should be preserved, got: %s", out)
	}
}

// 重复 INFO 行计数
func TestTextCompressor_DuplicateLineCount(t *testing.T) {
	cfg := TextConfig{Aggressiveness: 0.5}
	src := ""
	for i := 0; i < 200; i++ {
		src += "[INFO] heartbeat OK\n"
	}
	out := CompressText(src, cfg)
	if !strings.Contains(out, "[x200]") && !strings.Contains(out, "200") {
		t.Errorf("duplicate line should be counted as x200, got: %s", out)
	}
	// 结果明显更短
	if len(out) > len(src)/2 {
		t.Errorf("output should be much shorter than input, got len=%d/%d", len(out), len(src))
	}
}

// stopwords 缩短典型英文句子（≥30%）
func TestTextCompressor_StopwordsShrink(t *testing.T) {
	cfg := TextConfig{Aggressiveness: 0.5}
	src := "the server is running in production mode for the client"
	out := CompressText(src, cfg)
	// must contain keywords
	for _, kw := range []string{"server", "production", "client"} {
		if !strings.Contains(out, kw) {
			t.Errorf("keyword %q missing from output: %s", kw, out)
		}
	}
	// must be shorter
	if len(out) == 0 || float64(len(out)) > float64(len(src))*0.7 {
		t.Errorf("output %d chars should be <70%% of input %d chars: %s", len(out), len(src), out)
	}
}

// 恰好 30 行段落 → 不折叠
func TestTextCompressor_30LinesNotCollapsed(t *testing.T) {
	cfg := TextConfig{Aggressiveness: 0.5}
	src := ""
	for i := 1; i <= 30; i++ {
		src += "line " + strconv.Itoa(i) + " content\n"
	}
	out := CompressText(src, cfg)
	if strings.Contains(out, "more lines") {
		t.Errorf("30-line para should NOT be collapsed, got: %s", out)
	}
}

// 31 行 → 折叠
func TestTextCompressor_31LinesCollapsed(t *testing.T) {
	cfg := TextConfig{Aggressiveness: 0.5}
	src := ""
	for i := 1; i <= 31; i++ {
		src += "line " + strconv.Itoa(i) + " content\n"
	}
	out := CompressText(src, cfg)
	if !strings.Contains(out, "more lines") {
		t.Errorf("31-line para should be collapsed, got: %s", out)
	}
}

// 中文文本：不做 stopwords 删除（避免误伤）
func TestTextCompressor_ChineseNotStopwordProcessed(t *testing.T) {
	cfg := TextConfig{Aggressiveness: 0.5}
	src := "这个是一个中文句子用于测试 headroom 的文本压缩。"
	out := CompressText(src, cfg)
	if out == "" {
		t.Error("chinese text should not become empty")
	}
	// 不应被 stopwords 处理
	if len(out) == 0 {
		t.Error("expected non-empty chinese output")
	}
}

// 行顺序保持
func TestTextCompressor_OrderPreserved(t *testing.T) {
	cfg := TextConfig{Aggressiveness: 0.5}
	src := "[ERROR] first error\n[INFO] middle info\n[FATAL] second fatal\n"
	out := CompressText(src, cfg)
	errIdx := strings.Index(out, "first error")
	fatalIdx := strings.Index(out, "second fatal")
	if errIdx == -1 || fatalIdx == -1 {
		t.Errorf("lines missing: %s", out)
	}
	if errIdx > fatalIdx {
		t.Errorf("order reversed: %s", out)
	}
}
