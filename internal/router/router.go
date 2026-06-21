package router

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/superops-team/headroom-go/internal/types"
)

var searchLinePattern = regexp.MustCompile(`^(.+?)(:|-)[0-9]+(:|-).+`)

type ContentRouter struct{}

func NewContentRouter() *ContentRouter {
	return &ContentRouter{}
}

// Detect 返回内容类型（JSON / Code / Text）。
// O(n) 单次扫描，无额外大内存分配。
func (r *ContentRouter) Detect(content string) types.ContentKind {
	if content == "" {
		return types.KindText
	}

	trimmed := strings.TrimSpace(content)
	// 快速 JSON 预检（以 { 或 [ 开头）
	if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
		if json.Valid([]byte(trimmed)) {
			return types.KindJSON
		}
	}

	if looksLikeHTML(trimmed) {
		return types.KindHTML
	}

	if looksLikeDiff(content) {
		return types.KindDiff
	}
	if looksLikeSearch(content) {
		return types.KindSearch
	}
	if looksLikeLog(content) {
		return types.KindLog
	}
	if looksLikeTabular(content) {
		return types.KindTabular
	}

	// 代码关键字检测：任意 3 行以上包含关键字
	keywords := []string{
		"func ", "func(", "def ", "return ",
		"class ", "import ", "export ", "struct ",
		"interface ", "enum ", "fn ", "const ", "var ",
		"throw ", "try ", "catch ", "async ", "await ",
	}

	keywordHits := 0
	hasCodeBlockMarker := false
	hasCommentAndBraces := false
	hasBraceOrSemicolon := false

	for _, line := range strings.Split(content, "\n") {
		l := strings.TrimSpace(line)
		if l == "" {
			continue
		}
		if strings.HasPrefix(l, "```") {
			hasCodeBlockMarker = true
		}
		if strings.HasPrefix(l, "//") || strings.HasPrefix(l, "#") {
			if hasBraceOrSemicolon || strings.Contains(l, "{") || strings.Contains(l, ";") {
				hasCommentAndBraces = true
			}
		}
		if strings.Contains(l, "{") || strings.Contains(l, ";") {
			hasBraceOrSemicolon = true
		}
		for _, kw := range keywords {
			if strings.Contains(l, kw) {
				keywordHits++
				break
			}
		}
		if keywordHits >= 3 || hasCodeBlockMarker || hasCommentAndBraces {
			return types.KindCode
		}
	}

	return types.KindText
}

func looksLikeHTML(s string) bool {
	l := strings.ToLower(s)
	return strings.HasPrefix(l, "<!doctype html") || strings.Contains(l, "<html") || (strings.Contains(l, "<head") && strings.Contains(l, "<body"))
}

func looksLikeDiff(s string) bool {
	if strings.Contains(s, "diff --git ") || strings.Contains(s, "\n@@ ") || strings.HasPrefix(s, "@@ ") {
		return true
	}
	return strings.Contains(s, "\n--- ") && strings.Contains(s, "\n+++ ")
}

func looksLikeSearch(s string) bool {
	hits := 0
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if searchLinePattern.MatchString(line) {
			hits++
		}
		if hits >= 2 {
			return true
		}
	}
	return false
}

func looksLikeLog(s string) bool {
	hits := 0
	for _, line := range strings.Split(s, "\n") {
		l := strings.ToUpper(strings.TrimSpace(line))
		if l == "" {
			continue
		}
		if strings.Contains(l, "ERROR") || strings.Contains(l, "WARN") || strings.Contains(l, "FAIL") || strings.Contains(l, "FATAL") || strings.HasPrefix(l, "TRACEBACK") || strings.Contains(l, " STACK ") {
			hits++
		}
		if strings.HasPrefix(l, "[") && (strings.Contains(l, "INFO]") || strings.Contains(l, "DEBUG]") || strings.Contains(l, "TRACE]")) {
			hits++
		}
		if hits >= 2 {
			return true
		}
	}
	return false
}

func looksLikeTabular(s string) bool {
	lines := nonEmptyLines(s)
	if len(lines) < 2 {
		return false
	}
	if strings.Contains(lines[0], "\t") && strings.Count(lines[0], "\t") == strings.Count(lines[1], "\t") {
		return true
	}
	if strings.Contains(lines[0], ",") && strings.Count(lines[0], ",") == strings.Count(lines[1], ",") {
		return true
	}
	if len(lines) >= 3 && strings.Contains(lines[0], "|") && strings.Contains(lines[1], "---") && strings.Contains(lines[2], "|") {
		return true
	}
	return false
}

func nonEmptyLines(s string) []string {
	parts := strings.Split(s, "\n")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
