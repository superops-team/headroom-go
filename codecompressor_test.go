package headroom

import (
	"strings"
	"testing"
)

func TestCodeCompressor_LineCommentRemoval(t *testing.T) {
	cfg := CodeConfig{Aggressiveness: 0.5}
	src := "package main\n// this is a comment\nfunc main() {\n    println(\"hi\") // inline comment\n}\n"
	out := CompressCode(src, cfg)
	if strings.Contains(out, "this is a comment") {
		t.Errorf("line comment should be removed, got: %s", out)
	}
	// 但代码本身应该保留
	if !strings.Contains(out, "func main()") {
		t.Errorf("code should be preserved, got: %s", out)
	}
}

func TestCodeCompressor_BlockCommentRemoval(t *testing.T) {
	cfg := CodeConfig{Aggressiveness: 0.5}
	src := "package main\n/* block comment\n  spans multiple lines */\nfunc main() {\n    println(\"hi\")\n}\n"
	out := CompressCode(src, cfg)
	if strings.Contains(out, "block comment") {
		t.Errorf("block comment should be removed, got: %s", out)
	}
	if !strings.Contains(out, "func main()") {
		t.Errorf("code should be preserved")
	}
}

func TestCodeCompressor_BlockCommentInsideStringPreserved(t *testing.T) {
	cfg := CodeConfig{Aggressiveness: 0.5}
	src := `package main
func main() {
    println("a/*not comment*/b")
    raw := ` + "`" + `keep /* raw */ text` + "`" + `
}`
	out := CompressCode(src, cfg)
	if !strings.Contains(out, `"a/*not comment*/b"`) || !strings.Contains(out, "keep /* raw */ text") {
		t.Errorf("block comment markers inside strings should be preserved, got: %s", out)
	}
}

func TestCodeCompressor_UnclosedBlockCommentPreservesTail(t *testing.T) {
	cfg := CodeConfig{Aggressiveness: 0.5}
	src := "package main\nfunc main() {\n    println(\"before\")\n    /* truncated comment\n    println(\"after\")\n}\n"
	out := CompressCode(src, cfg)
	if !strings.Contains(out, "truncated comment") || !strings.Contains(out, `println("after")`) {
		t.Fatalf("unclosed block comment should preserve tail, got: %s", out)
	}
}

func TestCodeCompressor_HashCommentRemoval(t *testing.T) {
	cfg := CodeConfig{Aggressiveness: 0.5}
	src := "#!/usr/bin/env python\n# this is python comment\ndef foo():\n    return 1\n"
	out := CompressCode(src, cfg)
	if strings.Contains(out, "this is python comment") {
		t.Errorf("hash comment should be removed, got: %s", out)
	}
	if !strings.Contains(out, "def foo()") {
		t.Errorf("python code should be preserved")
	}
}

func TestCodeCompressor_EmptyLineRemoval(t *testing.T) {
	cfg := CodeConfig{Aggressiveness: 0.5}
	src := "a := 1\n\n\nb := 2\n\nc := 3\n"
	out := CompressCode(src, cfg)
	if strings.Contains(out, "\n\n") {
		t.Errorf("double empty lines should be collapsed, got: %s", out)
	}
}

// 恰好 20 行的函数体 → 不折叠
func TestCodeCompressor_20LinesNotCollapsed(t *testing.T) {
	cfg := CodeConfig{Aggressiveness: 0.5}
	// 20 行函数体（加上签名+闭合括号共22行）
	src := "func foo() {\n"
	for i := 1; i <= 20; i++ {
		src += "    x := 1\n"
	}
	src += "}\n"
	out := CompressCode(src, cfg)
	if strings.Contains(out, "lines collapsed") {
		t.Errorf("20-line function should NOT be collapsed, got: %s", out)
	}
}

// 21 行函数体 → 折叠
func TestCodeCompressor_21LinesCollapsed(t *testing.T) {
	cfg := CodeConfig{Aggressiveness: 0.5}
	src := "func foo() {\n"
	for i := 1; i <= 21; i++ {
		src += "    x := 1\n"
	}
	src += "}\n"
	out := CompressCode(src, cfg)
	if !strings.Contains(out, "lines collapsed") {
		t.Errorf("21-line function should be collapsed, got: %s", out)
	}
}

// err != nil 错误处理 → 不折叠
func TestCodeCompressor_PreserveErrorHandling(t *testing.T) {
	cfg := CodeConfig{Aggressiveness: 0.5}
	src := "if err != nil {\n    return nil, err\n}\n"
	out := CompressCode(src, cfg)
	if !strings.Contains(out, "err != nil") {
		t.Errorf("error handling should be preserved, got: %s", out)
	}
}

// return 语句 → 不折叠
func TestCodeCompressor_PreserveReturn(t *testing.T) {
	cfg := CodeConfig{Aggressiveness: 0.5}
	src := "func foo() {\n    a := 1\n    return a\n    b := 2\n}\n"
	out := CompressCode(src, cfg)
	if !strings.Contains(out, "return a") {
		t.Errorf("return statement should be preserved, got: %s", out)
	}
}
