package tagprotector

import (
	"strings"

	"github.com/superops-team/headroom-go/internal/types"
)

type ProtectedContent = types.ProtectedContent

type TagProtector struct{}

func NewTagProtector() TagProtector { return TagProtector{} }

func (p TagProtector) Protect(content string) ProtectedContent {
	placeholders := make(map[string]string)
	var out strings.Builder
	for i := 0; i < len(content); {
		if content[i] != '<' {
			out.WriteByte(content[i])
			i++
			continue
		}
		end := strings.IndexByte(content[i:], '>')
		if end < 0 {
			out.WriteByte(content[i])
			i++
			continue
		}
		end += i
		tag := content[i : end+1]
		name := tagName(tag)
		if name == "" || isStandardHTMLTag(name) || strings.HasPrefix(tag, "</") {
			out.WriteString(tag)
			i = end + 1
			continue
		}
		blockEnd := end + 1
		if !strings.HasSuffix(strings.TrimSpace(tag), "/>") {
			blockEnd = findCustomTagBlockEnd(content, end+1, name)
		}
		original := content[i:blockEnd]
		placeholder := p.placeholder(len(placeholders), content, placeholders)
		placeholders[placeholder] = original
		out.WriteString(placeholder)
		i = blockEnd
	}
	return ProtectedContent{Text: out.String(), Placeholders: placeholders}
}

func findCustomTagBlockEnd(content string, scanFrom int, name string) int {
	depth := 1
	for i := scanFrom; i < len(content); {
		lt := strings.IndexByte(content[i:], '<')
		if lt < 0 {
			return len(content)
		}
		lt += i
		gt := strings.IndexByte(content[lt:], '>')
		if gt < 0 {
			return len(content)
		}
		gt += lt
		tag := content[lt : gt+1]
		tagName := tagName(tag)
		if tagName == name {
			trimmed := strings.TrimSpace(tag)
			if strings.HasPrefix(trimmed, "</") {
				depth--
				if depth == 0 {
					return gt + 1
				}
			} else if !strings.HasSuffix(trimmed, "/>") {
				depth++
			}
		}
		i = gt + 1
	}
	return len(content)
}

func (p TagProtector) Restore(protected ProtectedContent) (string, []types.Warning) {
	out := protected.Text
	for placeholder, original := range protected.Placeholders {
		out = strings.Replace(out, placeholder, original, -1)
	}
	return out, protected.Warnings
}

func (p TagProtector) placeholder(i int, content string, used map[string]string) string {
	for {
		ph := "__HEADROOM_PROTECTED_TAG_" + intString(i) + "__"
		_, alreadyUsed := used[ph]
		if !strings.Contains(content, ph) && !alreadyUsed {
			return ph
		}
		i++
	}
}

func tagName(tag string) string {
	if len(tag) < 3 || tag[0] != '<' {
		return ""
	}
	i := 1
	if tag[i] == '/' {
		i++
	}
	start := i
	for i < len(tag) {
		c := tag[i]
		if c == '>' || c == '/' || c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			break
		}
		i++
	}
	return strings.ToLower(tag[start:i])
}

func isStandardHTMLTag(name string) bool {
	switch name {
	case "html", "head", "body", "title", "meta", "link", "script", "style", "div", "span", "p", "a", "img", "ul", "ol", "li", "table", "tr", "td", "th", "thead", "tbody", "article", "section", "nav", "footer", "header", "main", "br", "hr", "pre", "code", "h1", "h2", "h3", "h4", "h5", "h6":
		return true
	default:
		return false
	}
}

func intString(i int) string {
	if i == 0 {
		return "0"
	}
	digits := []byte{}
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	return string(digits)
}
