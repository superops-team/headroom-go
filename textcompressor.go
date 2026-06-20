package headroom

import (
	"strconv"
	"strings"
)

type TextConfig struct {
	Aggressiveness float64
}

// stopwords（英文，43 词）。基于日志/技术文档高频无用词。
var stopwordSet = map[string]struct{}{
	"a": {}, "an": {}, "the": {}, "and": {}, "or": {}, "but": {},
	"is": {}, "are": {}, "was": {}, "were": {}, "be": {}, "been": {}, "being": {},
	"have": {}, "has": {}, "had": {},
	"do": {}, "does": {}, "did": {},
	"will": {}, "would": {}, "should": {}, "could": {},
	"may": {}, "might": {}, "must": {}, "can": {},
	"of": {}, "to": {}, "in": {}, "for": {}, "on": {}, "at": {}, "by": {},
	"from": {}, "with": {}, "as": {}, "about": {}, "into": {}, "over": {},
	"after": {}, "before": {}, "between": {}, "during": {}, "under": {},
	"since": {}, "without": {}, "within": {}, "than": {}, "then": {}, "so": {},
}

// CompressText 压缩自然语言文本（日志/说明/文档）。
// 策略：
// - FATAL/ERROR 行完整保留（也做重复计数）
// - 连续重复行 → "行内容 [xN]"
// - 英文 stopwords 删除（对非 FATAL/ERROR 行）
// - 超长段落（>30 行）→ 前10 + [...N more lines...] + 最后5
func CompressText(content string, cfg TextConfig) string {
	if content == "" {
		return ""
	}

	lines := strings.Split(content, "\n")

	// 设计：
	//   origLine - 存储原始行（用于重复检测、high-priority 检查）
	//   procLine - 输出到 processed 的行（可能被 stopwords 修改）
	processed := make([]string, 0, len(lines))
	var origLine string
	dupCount := 0

	flushDup := func(curProcLine string) {
		if dupCount <= 0 || origLine == "" {
			return
		}
		if dupCount == 1 {
			processed = append(processed, curProcLine)
		} else {
			processed = append(processed, curProcLine+" [x"+strconv.Itoa(dupCount)+"]")
		}
		dupCount = 0
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 空行：flush 之前的计数，不保留空行
		if trimmed == "" {
			flushDup(removeStopwordsIfNeeded(origLine, cfg, isHighPriority(origLine)))
			origLine = ""
			dupCount = 0
			continue
		}

		// 连续重复（用原始内容比较，避免被处理后的值影响）
		if trimmed == origLine {
			dupCount++
			continue
		}

		// 新行：flush 上一组，设置新的 origLine/dupCount
		// 注意：先用 prevLine 生成"上一组"的输出行
		if origLine != "" {
			prevProc := removeStopwordsIfNeeded(origLine, cfg, isHighPriority(origLine))
			flushDup(prevProc)
		}
		origLine = trimmed
		dupCount = 1
	}
	// flush 最后一组
	if origLine != "" {
		prevProc := removeStopwordsIfNeeded(origLine, cfg, isHighPriority(origLine))
		if dupCount == 1 {
			processed = append(processed, prevProc)
		} else {
			processed = append(processed, prevProc+" [x"+strconv.Itoa(dupCount)+"]")
		}
	}

	// 超长段落折叠（>30 行）
	if len(processed) > 30 {
		head := processed[:10]
		tail := processed[len(processed)-5:]
		middleCount := len(processed) - 15
		var sb strings.Builder
		sb.Grow(256)
		for _, l := range head {
			sb.WriteString(l)
			sb.WriteString("\n")
		}
		sb.WriteString("[...")
		sb.WriteString(strconv.Itoa(middleCount))
		sb.WriteString(" more lines...]\n")
		for _, l := range tail {
			sb.WriteString(l)
			sb.WriteString("\n")
		}
		return strings.TrimRight(sb.String(), "\n")
	}

	return strings.Join(processed, "\n")
}

// removeStopwords 从一行文本中移除英文 stopwords。
// 仅对纯拉丁字符为主的内容处理。中文/符号行为直接返回。
func removeStopwords(line string) string {
	// 快速判断：非拉丁字符占比 >25% → 跳过
	nonLatinCount := 0
	for _, r := range line {
		if r > 127 {
			nonLatinCount++
		}
	}
	if len(line) > 0 && nonLatinCount > len(line)/4 {
		return line
	}

	var sb strings.Builder
	sb.Grow(len(line))

	// 按词分割（空白）
	fields := strings.Fields(line)

	first := true
	for _, f := range fields {
		word, prefix, suffix := splitWordPunct(f)
		lower := strings.ToLower(word)
		if _, isStop := stopwordSet[lower]; isStop {
			continue
		}
		if !first {
			sb.WriteString(" ")
		}
		first = false
		sb.WriteString(prefix)
		sb.WriteString(word)
		sb.WriteString(suffix)
	}
	return sb.String()
}

// removeStopwordsIfNeeded 根据优先级和 aggressiveness 决定是否删除 stopwords。
func removeStopwordsIfNeeded(line string, cfg TextConfig, highPriority bool) string {
	if highPriority {
		return line
	}
	if cfg.Aggressiveness < 0.3 {
		return line
	}
	return removeStopwords(line)
}

// isHighPriority 判断某行是否包含 FATAL/ERROR（需要完整保留）。
func isHighPriority(line string) bool {
	upper := strings.ToUpper(line)
	return strings.Contains(upper, "FATAL") || strings.Contains(upper, "ERROR")
}

// splitWordPunct 把 "word," 拆成 ("word", "", ",")，把 "(word)" 拆成 ("word", "(", ")")
func splitWordPunct(s string) (word, prefix, suffix string) {
	i := 0
	for i < len(s) && !isLetterOrDigit(rune(s[i])) {
		i++
	}
	prefix = s[:i]
	j := len(s)
	for j > i && !isLetterOrDigit(rune(s[j-1])) {
		j--
	}
	word = s[i:j]
	suffix = s[j:]
	return
}

func isLetterOrDigit(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}
