package headroom

import (
	"strconv"
	"strings"
)

type CodeConfig struct {
	Aggressiveness float64
}

// CompressCode 压缩代码文本。
// 策略：移除注释/空行 → 折叠过长函数体 → 保留语义锚点。
func CompressCode(content string, cfg CodeConfig) string {
	// Step 1: 移除块注释
	content = removeBlockComments(content)

	// Step 2: 逐行处理（移除单行注释/空行/收缩空白）
	lines := strings.Split(content, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// 整行是注释 → 跳过
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		// 移除行尾 `// ` 注释（保留代码部分）。
		// 必须忽略字符串字面量内的 `// `，避免误删 `"a//b"`。
		// 采用状态机扫描：track 单双引号 + 转义符。
		if idx := indexLineComment(line); idx >= 0 {
			line = strings.TrimRight(line[:idx], " \t")
		}
		// 收缩连续空白为单个空格
		line = collapseWhitespace(line)
		filtered = append(filtered, line)
	}

	// Step 3: 函数体折叠检测（若 aggressiveness >= 0.3）
	if cfg.Aggressiveness >= 0.3 {
		filtered = collapseLongFunctions(filtered)
	}

	return strings.Join(filtered, "\n")
}

func removeBlockComments(content string) string {
	var b strings.Builder
	b.Grow(len(content))
	inSingle := false
	inDouble := false
	inRaw := false
	for i := 0; i < len(content); i++ {
		ch := content[i]
		if ch == '\\' && (inSingle || inDouble) && i+1 < len(content) {
			b.WriteByte(ch)
			i++
			b.WriteByte(content[i])
			continue
		}
		if !inSingle && !inDouble && ch == '`' {
			inRaw = !inRaw
			b.WriteByte(ch)
			continue
		}
		if !inRaw {
			switch ch {
			case '"':
				if !inSingle {
					inDouble = !inDouble
				}
			case '\'':
				if !inDouble {
					inSingle = !inSingle
				}
			case '/':
				if !inSingle && !inDouble && i+1 < len(content) && content[i+1] == '*' {
					j := i + 2
					closed := false
					for j < len(content) {
						if content[j] == '*' && j+1 < len(content) && content[j+1] == '/' {
							closed = true
							break
						}
						j++
					}
					if closed {
						for k := i + 2; k < j; k++ {
							if content[k] == '\n' || content[k] == '\r' {
								b.WriteByte(content[k])
							}
						}
						i = j + 1
						continue
					}
				}
			}
		}
		b.WriteByte(ch)
	}
	return b.String()
}

// indexLineComment 返回行中真正"行尾注释"开始的位置，-1 表示不存在。
// 它会忽略字符串字面量（单双引号，考虑转义符 \）内的 `//`，
// 因此不会误删 "http://x" 或 "path //a"。
func indexLineComment(line string) int {
	inSingle := false
	inDouble := false
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if ch == '\\' && (inSingle || inDouble) && i+1 < len(line) {
			// 字符串内的转义：跳过下一字节
			i++
			continue
		}
		switch ch {
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '/':
			if !inSingle && !inDouble && i+1 < len(line) && line[i+1] == '/' {
				return i
			}
		}
	}
	return -1
}

func collapseWhitespace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' {
			if !prevSpace {
				b.WriteRune(' ')
				prevSpace = true
			}
		} else {
			b.WriteRune(r)
			prevSpace = false
		}
	}
	return b.String()
}

// collapseLongFunctions 检测并折叠超过 20 行的函数体。
// 简化策略：找到函数签名行（以 `func ` / `def ` / `class ` 开头，且包含 `{` 或 `:`），
// 从下一行到匹配的闭合括号（或下一个同级签名）算作"函数体"。
// 超过 20 行则保留：签名行 + "// ... (N lines collapsed) ..." + 最后 3 行 + 闭合行。
func collapseLongFunctions(lines []string) []string {
	if len(lines) < 22 {
		return lines
	}

	out := make([]string, 0, len(lines))
	i := 0
	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// 是否是函数/类签名行？
		if isFuncOrClassSignature(trimmed) {
			// 找到匹配的闭合位置
			// 简化策略：从 i+1 开始，深度 = 括号 balance，遇到 depth = 0 的闭合行算作结束
			start := i + 1
			depth := 0
			// 初始化深度（基于签名行末尾的 { 或 Python 的缩进表示）
			if strings.Contains(trimmed, "{") {
				depth = 1
			} else {
				// Python 风格：以 ':' 结尾，按缩进检测
				// 找到下一个相同或更小缩进的行
				baseIndent := lineIndent(line)
				j := start
				for j < len(lines) && (lineIndent(lines[j]) > baseIndent || strings.TrimSpace(lines[j]) == "") {
					j++
				}
				// j 现在是函数体结束行 + 1
				bodyLen := j - start
				if bodyLen > 20 {
					// 保留签名行 + 折叠标记 + 最后 3 行
					out = append(out, line)
					out = append(out, "  // ... ("+strconv.Itoa(bodyLen)+" lines collapsed) ...")
					lastStart := j - 3
					if lastStart < start {
						lastStart = start
					}
					for k := lastStart; k < j; k++ {
						out = append(out, lines[k])
					}
					i = j
					continue
				}
			}

			// Go/C/Java 风格：用括号平衡检测
			if depth > 0 {
				j := start
				for j < len(lines) {
					for _, ch := range lines[j] {
						if ch == '{' {
							depth++
						} else if ch == '}' {
							depth--
						}
					}
					j++
					if depth <= 0 {
						break
					}
				}
				bodyLen := j - start - 1
				if bodyLen > 20 {
					out = append(out, line)
					out = append(out, "  // ... ("+strconv.Itoa(bodyLen)+" lines collapsed) ...")
					// 保留最后 3 行（加上闭合括号行 j）
					lastStart := j - 3
					if lastStart < start {
						lastStart = start
					}
					for k := lastStart; k < j; k++ {
						out = append(out, lines[k])
					}
					i = j
					continue
				}
			}

			out = append(out, line)
			i++
			continue
		}

		// 语义锚点：err / return / throw → 保留
		if strings.Contains(trimmed, "err") || strings.HasPrefix(trimmed, "return") || strings.HasPrefix(trimmed, "throw") {
			out = append(out, line)
			i++
			continue
		}

		out = append(out, line)
		i++
	}
	return out
}

func isFuncOrClassSignature(trimmed string) bool {
	switch {
	case strings.HasPrefix(trimmed, "func "),
		strings.HasPrefix(trimmed, "def "),
		strings.HasPrefix(trimmed, "class "),
		strings.HasPrefix(trimmed, "fn "):
		return true
	}
	return false
}

func lineIndent(s string) int {
	n := 0
	for _, r := range s {
		if r == ' ' {
			n++
		} else if r == '\t' {
			n += 4
		} else {
			break
		}
	}
	return n
}
