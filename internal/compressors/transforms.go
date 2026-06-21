package compressors

import "strings"

type diffOffloadTransform struct{}

func (diffOffloadTransform) Name() string             { return "diff_offload" }
func (diffOffloadTransform) AppliesTo() []ContentKind { return []ContentKind{KindDiff} }
func (diffOffloadTransform) EstimateBloat(content string, ctx CompressionContext) float64 {
	if len(content) > 200 {
		return 1
	}
	return 0
}
func (diffOffloadTransform) Confidence() float64 { return 0.7 }
func (diffOffloadTransform) Apply(content string, ctx CompressionContext) (OffloadOutput, error) {
	lines := strings.Split(content, "\n")
	kept := make([]string, 0, len(lines))
	omitted := 0
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		keep := strings.HasPrefix(line, "diff --git") || strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "@@") || strings.HasPrefix(trim, "Binary files") || strings.Contains(trim, "renamed") || strings.Contains(trim, "new file mode") || strings.Contains(trim, "deleted file mode") || i < 3 || i >= len(lines)-3
		if ctx.Query != "" && strings.Contains(strings.ToLower(line), strings.ToLower(ctx.Query)) {
			keep = true
		}
		if keep {
			kept = append(kept, line)
		} else {
			omitted++
		}
	}
	if omitted > 0 {
		kept = append(kept, "[... omitted diff lines ...]")
	}
	out := strings.Join(kept, "\n")
	id := ""
	if ctx.CCR != nil {
		id = ctx.CCR.Store(content, out, ctx.ContentKind)
	}
	return OffloadOutput{Output: out, BytesSaved: len(content) - len(out), CacheKey: id, Steps: []CompressionStep{{Name: "diff_offload", Kind: ctx.ContentKind.String()}}}, nil
}

type logTemplateTransform struct{}

func (logTemplateTransform) Name() string             { return "log_template" }
func (logTemplateTransform) AppliesTo() []ContentKind { return []ContentKind{KindLog} }
func (logTemplateTransform) Apply(content string, ctx CompressionContext) (ReformatOutput, error) {
	out := CompressText(content, TextConfig{Aggressiveness: ctx.Aggressiveness})
	return ReformatOutput{Output: out, BytesSaved: len(content) - len(out), Steps: []CompressionStep{{Name: "log_template", Kind: ctx.ContentKind.String()}}}, nil
}

type logOffloadTransform struct{}

func (logOffloadTransform) Name() string             { return "log_offload" }
func (logOffloadTransform) AppliesTo() []ContentKind { return []ContentKind{KindLog} }
func (logOffloadTransform) EstimateBloat(content string, ctx CompressionContext) float64 {
	if len(content) > 300 {
		return 1
	}
	return 0
}
func (logOffloadTransform) Confidence() float64 { return 0.7 }
func (logOffloadTransform) Apply(content string, ctx CompressionContext) (OffloadOutput, error) {
	lines := strings.Split(content, "\n")
	kept := make([]string, 0, len(lines))
	omitted := 0
	for i, line := range lines {
		upper := strings.ToUpper(line)
		keep := strings.Contains(upper, "ERROR") || strings.Contains(upper, "FAIL") || strings.Contains(upper, "FATAL") || strings.HasPrefix(strings.TrimSpace(upper), "TRACEBACK") || i < 5 || i >= len(lines)-5
		if keep {
			kept = append(kept, line)
		} else {
			omitted++
		}
	}
	if omitted > 0 {
		kept = append(kept, "[... omitted low-priority log lines ...]")
	}
	out := strings.Join(kept, "\n")
	id := ""
	if ctx.CCR != nil {
		id = ctx.CCR.Store(content, out, ctx.ContentKind)
	}
	return OffloadOutput{Output: out, BytesSaved: len(content) - len(out), CacheKey: id, Steps: []CompressionStep{{Name: "log_offload", Kind: ctx.ContentKind.String()}}}, nil
}

type searchOffloadTransform struct{}

func (searchOffloadTransform) Name() string             { return "search_offload" }
func (searchOffloadTransform) AppliesTo() []ContentKind { return []ContentKind{KindSearch} }
func (searchOffloadTransform) EstimateBloat(content string, ctx CompressionContext) float64 {
	if len(content) > 200 {
		return 1
	}
	return 0
}
func (searchOffloadTransform) Confidence() float64 { return 0.7 }
func (searchOffloadTransform) Apply(content string, ctx CompressionContext) (OffloadOutput, error) {
	groups := make(map[string][]string)
	order := []string{}
	for _, line := range strings.Split(content, "\n") {
		file, rest := splitSearchLine(line)
		if file == "" {
			continue
		}
		if _, ok := groups[file]; !ok {
			order = append(order, file)
		}
		if len(groups[file]) < 5 || (ctx.Query != "" && strings.Contains(strings.ToLower(rest), strings.ToLower(ctx.Query))) || strings.Contains(strings.ToUpper(rest), "ERROR") {
			groups[file] = append(groups[file], rest)
		}
	}
	if len(order) == 0 {
		return OffloadOutput{Output: content}, nil
	}
	outLines := []string{}
	for _, file := range order {
		outLines = append(outLines, file+":")
		for _, rest := range groups[file] {
			outLines = append(outLines, "  "+rest)
		}
	}
	out := strings.Join(outLines, "\n")
	id := ""
	if ctx.CCR != nil {
		id = ctx.CCR.Store(content, out, ctx.ContentKind)
	}
	return OffloadOutput{Output: out, BytesSaved: len(content) - len(out), CacheKey: id, Steps: []CompressionStep{{Name: "search_offload", Kind: ctx.ContentKind.String()}}}, nil
}

func splitSearchLine(line string) (string, string) {
	for i := len(line) - 1; i >= 0; i-- {
		if line[i] == ':' || line[i] == '-' {
			j := i - 1
			for j >= 0 && line[j] >= '0' && line[j] <= '9' {
				j--
			}
			if j >= 0 && (line[j] == ':' || line[j] == '-') && j > 0 {
				return line[:j], line[j+1:]
			}
		}
	}
	return "", ""
}

type htmlCleanTransform struct{}

func NewDiffOffloadTransform() OffloadTransform     { return diffOffloadTransform{} }
func NewLogTemplateTransform() ReformatTransform    { return logTemplateTransform{} }
func NewLogOffloadTransform() OffloadTransform      { return logOffloadTransform{} }
func NewSearchOffloadTransform() OffloadTransform   { return searchOffloadTransform{} }
func NewHTMLCleanTransform() ReformatTransform      { return htmlCleanTransform{} }

func (htmlCleanTransform) Name() string             { return "html_clean" }
func (htmlCleanTransform) AppliesTo() []ContentKind { return []ContentKind{KindHTML} }
func (htmlCleanTransform) Apply(content string, ctx CompressionContext) (ReformatOutput, error) {
	out := removeHTMLBlock(content, "script")
	out = removeHTMLBlock(out, "style")
	out = removeHTMLComments(out)
	out = strings.TrimSpace(out)
	return ReformatOutput{Output: out, BytesSaved: len(content) - len(out), Steps: []CompressionStep{{Name: "html_clean", Kind: ctx.ContentKind.String()}}}, nil
}

func removeHTMLBlock(s, tag string) string {
	lower := strings.ToLower(s)
	startNeedle := "<" + tag
	endNeedle := "</" + tag + ">"
	for {
		start := strings.Index(lower, startNeedle)
		if start < 0 {
			return s
		}
		end := strings.Index(lower[start:], endNeedle)
		if end < 0 {
			return s
		}
		end = start + end + len(endNeedle)
		s = s[:start] + s[end:]
		lower = strings.ToLower(s)
	}
}

func RemoveHTMLBlock(s, tag string) string {
	return removeHTMLBlock(s, tag)
}

func removeHTMLComments(s string) string {
	for {
		start := strings.Index(s, "<!--")
		if start < 0 {
			return s
		}
		end := strings.Index(s[start+4:], "-->")
		if end < 0 {
			return s
		}
		end = start + 4 + end + 3
		s = s[:start] + s[end:]
	}
}

func RemoveHTMLComments(s string) string {
	return removeHTMLComments(s)
}
