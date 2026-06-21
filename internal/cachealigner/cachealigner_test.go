package cachealigner

import "testing"

// 启用 → 输出应包含 "[headroom/v0.1]" 前缀
func TestCacheAligner_Enabled(t *testing.T) {
	cfg := CacheAlignerConfig{Enabled: true, Version: "v0.1"}
	a := NewCacheAligner(cfg)
	out := a.Align("some compressed content")
	if len(out) == 0 {
		t.Fatal("empty output")
	}
	if len(out) < 15 || out[:15] != "[headroom/v0.1]" {
		t.Errorf("prefix missing, got: %s", out[:30])
	}
}

// 关闭 → 原样返回
func TestCacheAligner_Disabled(t *testing.T) {
	cfg := CacheAlignerConfig{Enabled: false}
	a := NewCacheAligner(cfg)
	in := "unchanged content"
	out := a.Align(in)
	if out != in {
		t.Errorf("disabled mode should return input verbatim: got %q, want %q", out, in)
	}
}
