package headroom

import (
	"strconv"
	"strings"
	"testing"
)

func BenchmarkTokenizerFallback_1MB(b *testing.B) {
	tok := FallbackTokenizer{}
	s := strings.Repeat("hello 世界 ", 1024*64)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tok.Count(s)
	}
}

func BenchmarkDetector_Mixed_1MB(b *testing.B) {
	r := NewContentRouter()
	s := strings.Repeat("a.go:10:func main()\n", 1024*32)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Detect(s)
	}
}

func BenchmarkTagProtector_NestedTags(b *testing.B) {
	p := NewTagProtector()
	s := strings.Repeat("<system-reminder><tool_call/>keep</system-reminder> text ", 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pc := p.Protect(s)
		_, _ = p.Restore(pc)
	}
}

func BenchmarkCompress_EndToEnd_MixedMessages(b *testing.B) {
	opts := DefaultOptions()
	opts.EnablePipeline = true
	opts.Reversible = false
	msgs := []Message{{Role: "user", Content: strings.Repeat("[INFO] ok\n", 1000)}, {Role: "user", Content: strings.Repeat("a.go:10:func main\n", 1000)}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Compress(msgs, opts)
	}
}

func BenchmarkSmartCrusher_Array_10k(b *testing.B) {
	var sb strings.Builder
	sb.WriteString(`{"items":[`)
	for i := 0; i < 10000; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		status := "ok"
		message := "normal heartbeat"
		if i%997 == 0 {
			status = "ERROR"
			message = "critical failure"
		}
		sb.WriteString(`{"id":"item-`)
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString(`","status":"`)
		sb.WriteString(status)
		sb.WriteString(`","message":"`)
		sb.WriteString(message)
		sb.WriteString(`","empty":""}`)
	}
	sb.WriteString(`]}`)
	content := sb.String()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = SmartCrushJSON(content, SmartCrushConfig{Aggressiveness: 0.6})
	}
}

func BenchmarkDiffCompressor_5kLines(b *testing.B) {
	content := buildBenchmarkDiff(5000)
	ctx := CompressionContext{ContentKind: KindDiff, Query: "needle", CCR: getPackageCCR()}
	t := diffOffloadTransform{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = t.Apply(content, ctx)
	}
}

func BenchmarkLogCompressor_50kLines(b *testing.B) {
	content := buildBenchmarkLog(50000)
	ctx := CompressionContext{ContentKind: KindLog, CCR: getPackageCCR()}
	t := logOffloadTransform{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = t.Apply(content, ctx)
	}
}

func BenchmarkSearchCompressor_10kMatches(b *testing.B) {
	content := buildBenchmarkSearch(10000)
	ctx := CompressionContext{ContentKind: KindSearch, Query: "target", CCR: getPackageCCR()}
	t := searchOffloadTransform{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = t.Apply(content, ctx)
	}
}

func buildBenchmarkDiff(lines int) string {
	var sb strings.Builder
	sb.WriteString("diff --git a/a.go b/a.go\n--- a/a.go\n+++ b/a.go\n@@ -1 +1 @@\n")
	for i := 0; i < lines; i++ {
		if i%2 == 0 {
			sb.WriteString("-old line ")
		} else {
			sb.WriteString("+new line ")
		}
		sb.WriteString(strconv.Itoa(i))
		sb.WriteByte('\n')
	}
	return sb.String()
}

func buildBenchmarkLog(lines int) string {
	var sb strings.Builder
	for i := 0; i < lines; i++ {
		if i%4999 == 0 {
			sb.WriteString("[ERROR] service=api critical failure id=")
		} else {
			sb.WriteString("[INFO] service=api heartbeat ok id=")
		}
		sb.WriteString(strconv.Itoa(i))
		sb.WriteByte('\n')
	}
	return sb.String()
}

func buildBenchmarkSearch(lines int) string {
	var sb strings.Builder
	for i := 0; i < lines; i++ {
		sb.WriteString("pkg/file")
		sb.WriteString(strconv.Itoa(i % 100))
		sb.WriteString(".go:")
		sb.WriteString(strconv.Itoa(i + 1))
		sb.WriteString(":")
		if i%250 == 0 {
			sb.WriteString("target ERROR important match")
		} else {
			sb.WriteString("ordinary match")
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}
