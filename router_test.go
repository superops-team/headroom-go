package headroom

import "testing"

func TestContentRouter_JSONArray(t *testing.T) {
	r := NewContentRouter()
	got := r.Detect("[1, 2, 3, 4]")
	if got != KindJSON {
		t.Errorf("JSON array: got %s, want JSON", got)
	}
}

func TestContentRouter_JSONObject(t *testing.T) {
	r := NewContentRouter()
	got := r.Detect(`{"name": "headroom", "version": 1}`)
	if got != KindJSON {
		t.Errorf("JSON object: got %s, want JSON", got)
	}
}

func TestContentRouter_PythonCode(t *testing.T) {
	r := NewContentRouter()
	src := "def foo():\n    import json\n    data = {}\n    return data\n"
	got := r.Detect(src)
	if got != KindCode {
		t.Errorf("Python code: got %s, want Code", got)
	}
}

func TestContentRouter_PlainText(t *testing.T) {
	r := NewContentRouter()
	got := r.Detect("INFO 2026-06-14 service started on port 8080")
	if got != KindText {
		t.Errorf("plain text: got %s, want Text", got)
	}
}

func TestContentRouter_Empty(t *testing.T) {
	r := NewContentRouter()
	got := r.Detect("")
	if got != KindText {
		t.Errorf("empty: got %s, want Text", got)
	}
}

// 恰好 2 个关键字行 → 不应判定为代码（需要 ≥3）
func TestContentRouter_NotEnoughKeywords(t *testing.T) {
	r := NewContentRouter()
	src := "this is a return statement\nfollowed by another line\n"
	got := r.Detect(src)
	if got == KindCode {
		t.Errorf("2-keyword lines should NOT be KindCode, got KindCode")
	}
}

// Markdown 代码块 → KindCode
func TestContentRouter_CodeBlockMarker(t *testing.T) {
	r := NewContentRouter()
	src := "```go\npackage main\nfunc main(){}\n```"
	got := r.Detect(src)
	if got != KindCode {
		t.Errorf("markdown code block: got %s, want Code", got)
	}
}

// 非法 JSON（尾随逗号） → KindText
func TestContentRouter_InvalidJSON(t *testing.T) {
	r := NewContentRouter()
	got := r.Detect(`{"a": 1, "b": 2,}`)
	if got == KindJSON {
		t.Errorf("invalid JSON should NOT be KindJSON, got KindJSON")
	}
}
