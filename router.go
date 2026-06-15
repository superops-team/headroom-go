package headroom

import (
	"encoding/json"
	"strings"
)

type ContentRouter struct{}

func NewContentRouter() *ContentRouter {
	return &ContentRouter{}
}

// Detect 返回内容类型（JSON / Code / Text）。
// O(n) 单次扫描，无额外大内存分配。
func (r *ContentRouter) Detect(content string) ContentKind {
	if content == "" {
		return KindText
	}

	trimmed := strings.TrimSpace(content)
	// 快速 JSON 预检（以 { 或 [ 开头）
	if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
		if json.Valid([]byte(trimmed)) {
			return KindJSON
		}
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
			return KindCode
		}
	}

	return KindText
}
