package headroom

import (
	"errors"
	"strconv"
	"strings"
	"testing"
)

type recordingObserver struct {
	steps []CompressionStep
}

func (r *recordingObserver) ObserveCompressionStep(step CompressionStep) {
	r.steps = append(r.steps, step)
}

// 混合消息：JSON + 代码 + 文本 + assistant 透传
func TestCompress_MixedMessages(t *testing.T) {
	// 构建长消息：JSON + 代码 + 文本，确保总长度足够触发真正压缩
	var longLogBuilder strings.Builder
	for i := 0; i < 50; i++ {
		longLogBuilder.WriteString("[INFO] service=api user=user" + strconv.Itoa(i) + " status=ok latency=12ms\n")
	}
	longLog := longLogBuilder.String()

	longCode := "def process_data(data):\n    if err != nil {\n        return nil, err\n    }\n    result = []\n    for i in range(100):\n        result.append(i * 2)\n    x = 1 + 1\n    y = 2 + 2\n    z = 3 + 3\n    w = 4 + 4\n    return result\n"

	longJSON := `{"items":[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15],"metadata":{"tags":["a","b","c","d","e","f","g","h","i"],"null":null,"empty":{}}}`

	msgs := []Message{
		{Role: "system", Content: "You are a helpful assistant. Please review the following data carefully. Focus on the important parts and ignore the filler text that repeats multiple times without adding value to the analysis."},
		{Role: "user", Content: longJSON},
		{Role: "assistant", Content: "I have processed the data successfully and prepared the analysis."},
		{Role: "user", Content: longCode},
		{Role: "user", Content: longLog},
	}

	opts := DefaultOptions()
	opts.Reversible = false // 关闭可逆，直接看压缩收益

	result, err := Compress(msgs, opts)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Messages) != len(msgs) {
		t.Errorf("got %d messages, want %d", len(result.Messages), len(msgs))
	}

	// assistant 角色消息应原样透传
	if result.Messages[2].Content != "I have processed the data successfully and prepared the analysis." {
		t.Errorf("assistant message changed: got %q", result.Messages[2].Content)
	}

	// Savings 应为正值
	if result.Savings <= 0 {
		t.Errorf("savings got %f, want >0", result.Savings)
	}

	t.Logf("Original: %d tokens, Compressed: %d tokens, Savings: %.1f%%",
		result.OriginalTokens, result.CompressedTokens, result.Savings*100)
}

// 可逆压缩模式：压缩输出中包含 [headroom:retrieve id=...]
func TestCompress_Reversible(t *testing.T) {
	// 用长消息，确保压缩收益 > retrieve id 开销
	var sb strings.Builder
	for i := 0; i < 50; i++ {
		sb.WriteString("[INFO] heartbeat OK service=api status=running latency=12ms\n")
	}
	msgs := []Message{
		{Role: "user", Content: sb.String()},
	}
	opts := DefaultOptions()
	opts.Reversible = true

	result, err := Compress(msgs, opts)
	if err != nil {
		t.Fatal(err)
	}
	content := result.Messages[0].Content
	if !strings.Contains(content, "headroom:retrieve id=v2_") {
		t.Errorf("reversible output missing retrieve marker, got: %s", content[:100])
	}
}

// 不可逆压缩模式：不包含 retrieve 标记
func TestCompress_NonReversible(t *testing.T) {
	msgs := []Message{
		{Role: "user", Content: "some content that will be compressed"},
	}
	opts := DefaultOptions()
	opts.Reversible = false

	result, err := Compress(msgs, opts)
	if err != nil {
		t.Fatal(err)
	}
	content := result.Messages[0].Content
	if strings.Contains(content, "headroom:retrieve") {
		t.Errorf("non-reversible output has retrieve marker: %s", content)
	}
}

// TokenLimit：短消息不压缩
func TestCompress_TokenLimit(t *testing.T) {
	short := "hi"
	msgs := []Message{{Role: "user", Content: short}}
	opts := DefaultOptions()
	opts.TokenLimit = 100 // 需要 > 估算 tokens 才跳过
	opts.Reversible = false

	result, err := Compress(msgs, opts)
	if err != nil {
		t.Fatal(err)
	}
	if result.Messages[0].Content != short {
		t.Errorf("token-limit skipped message modified: got %q, want %q",
			result.Messages[0].Content, short)
	}
}

func TestCompressLegacySkipStepsKeepNamesAndReasons(t *testing.T) {
	opts := DefaultOptions()
	opts.Reversible = false
	opts.TokenLimit = 100
	msgs := []Message{
		{Role: "assistant", Content: "assistant reply"},
		{Role: "user", Content: "   \n\t"},
		{Role: "user", Content: "short"},
	}
	res, err := Compress(msgs, opts)
	if err != nil {
		t.Fatal(err)
	}
	wants := []struct{ name, reason string }{
		{"skip_assistant", "assistant role"},
		{"skip_empty", "empty content"},
		{"skip_token_limit", "below token limit"},
	}
	for i, want := range wants {
		if res.Messages[i].Content != msgs[i].Content {
			t.Fatalf("message %d changed: %q", i, res.Messages[i].Content)
		}
		if res.Steps[i].Name != want.name || res.Steps[i].Reason != want.reason || !res.Steps[i].Skipped {
			t.Fatalf("step %d got %#v want name=%q reason=%q skipped", i, res.Steps[i], want.name, want.reason)
		}
	}
}

func TestCompressLegacyRoutesJSONCodeTextAndWrapsError(t *testing.T) {
	opts := DefaultOptions()
	opts.Reversible = false
	msgs := []Message{
		{Role: "user", Content: `{"a":1,"b":null,"items":[1,2,3,4,5,6,7,8,9,10]}`},
		{Role: "user", Content: "// comment {\nfunc main() {\n    var x = 1\n    return\n}"},
		{Role: "user", Content: strings.Repeat("repeat\n", 5)},
	}
	res, err := Compress(msgs, opts)
	if err != nil {
		t.Fatal(err)
	}
	wants := []string{KindJSON.String(), KindCode.String(), KindText.String()}
	for i, want := range wants {
		if res.Steps[i].Name != "legacy_compress" || res.Steps[i].Kind != want {
			t.Fatalf("step %d got %#v want kind %s", i, res.Steps[i], want)
		}
	}

	sentinel := errors.New("route sentinel")
	registry := DefaultCompressorRegistry()
	original, ok := registry.Lookup(KindText)
	if !ok {
		t.Fatal("default text compressor missing")
	}
	registry.Register(NewCompressorFunc(KindText, func(content string, opts Options) (string, error) {
		return content, sentinel
	}))
	t.Cleanup(func() { registry.Register(original) })
	_, err = Compress([]Message{{Role: "user", Content: "plain text input with enough tokens"}}, opts)
	if !errors.Is(err, sentinel) || !strings.Contains(err.Error(), "compress Text:") {
		t.Fatalf("expected wrapped route error, got %v", err)
	}
}

func TestCompressLegacyPostProcessPrefixReversibleAndFallback(t *testing.T) {
	longRepeated := strings.Repeat("[INFO] heartbeat OK service=api status=running latency=12ms\n", 80)
	prefixOpts := DefaultOptions()
	prefixOpts.Reversible = false
	prefixOpts.AlignPrefix = true
	res, err := Compress([]Message{{Role: "user", Content: longRepeated}}, prefixOpts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(res.Messages[0].Content, "[headroom/"+PrefixVersion+"]\n") {
		t.Fatalf("aligned prefix missing: %q", res.Messages[0].Content[:minLen(len(res.Messages[0].Content), 80)])
	}

	reversibleOpts := DefaultOptions()
	reversibleOpts.Reversible = true
	res, err = Compress([]Message{{Role: "user", Content: longRepeated}}, reversibleOpts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Messages[0].Content, "\n\n[headroom:retrieve id=v2_") {
		t.Fatalf("retrieve suffix missing: %q", res.Messages[0].Content)
	}

	fallbackOpts := DefaultOptions()
	fallbackOpts.Reversible = false
	short := "abc"
	res, err = Compress([]Message{{Role: "user", Content: short}}, fallbackOpts)
	if err != nil {
		t.Fatal(err)
	}
	if res.Messages[0].Content != short || !res.Steps[0].Skipped || res.Steps[0].Reason != "output not shorter" {
		t.Fatalf("fallback got msg=%q step=%#v", res.Messages[0].Content, res.Steps[0])
	}
}

func TestCompressLegacyStatsWarningsAndObserver(t *testing.T) {
	observer := &recordingObserver{}
	opts := DefaultOptions()
	opts.Reversible = false
	opts.Observer = observer
	opts.TokenizerConfig = TokenizerConfig{Backend: TokenizerTiktoken, AllowFallback: true}
	msg := strings.Repeat("repeat\n", 20)
	res, err := Compress([]Message{{Role: "user", Content: msg}}, opts)
	if err != nil {
		t.Fatal(err)
	}
	wantSavings := float64(res.OriginalTokens-res.CompressedTokens) / float64(res.OriginalTokens)
	if res.Savings != wantSavings {
		t.Fatalf("savings got %v want %v", res.Savings, wantSavings)
	}
	if len(observer.steps) != len(res.Steps) {
		t.Fatalf("observer saw %d steps want %d", len(observer.steps), len(res.Steps))
	}
	if len(res.Warnings) == 0 || res.Warnings[0].Code != "tokenizer_fallback" {
		t.Fatalf("initial warnings not preserved first: %#v", res.Warnings)
	}
}

func TestCompressLegacyJSONObserverMatchesResultSteps(t *testing.T) {
	observer := &recordingObserver{}
	opts := DefaultOptions()
	opts.Reversible = false
	opts.Observer = observer

	res, err := Compress([]Message{{Role: "user", Content: `{"items":[1,2,3,4,5,6,7,8,9,10],"drop":null}`}}, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(observer.steps) != len(res.Steps) {
		t.Fatalf("observer saw %d steps want %d", len(observer.steps), len(res.Steps))
	}
	for i := range res.Steps {
		if observer.steps[i] != res.Steps[i] {
			t.Fatalf("observer step %d = %#v, want %#v", i, observer.steps[i], res.Steps[i])
		}
	}
}

func TestCompressPipelineDispatchDoesNotUseLegacyStep(t *testing.T) {
	opts := DefaultOptions()
	opts.EnablePipeline = true
	opts.Reversible = false
	res, err := Compress([]Message{{Role: "user", Content: strings.Repeat("repeat\n", 20)}}, opts)
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range res.Steps {
		if step.Name == "legacy_compress" {
			t.Fatalf("pipeline dispatch should not use legacy step: %#v", res.Steps)
		}
	}
}

func minLen(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// CompressString：便捷方法
func TestCompressString(t *testing.T) {
	content := `{"a":1, "b":null, "data":[1,2,3,4,5,6,7,8,9,10]}`
	opts := DefaultOptions()
	opts.Reversible = false

	out, err := CompressString(content, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Error("empty output")
	}
	// 输出应比输入短（移除了 null，且数组被折叠）
	if len(out) >= len(content) {
		t.Errorf("output %d chars should be shorter than input %d chars", len(out), len(content))
	}
}

// 空消息数组：正常返回无错误
func TestCompress_Empty(t *testing.T) {
	result, err := Compress([]Message{}, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Messages) != 0 {
		t.Errorf("empty input should produce empty output, got %d messages", len(result.Messages))
	}
	if result.OriginalTokens != 0 || result.CompressedTokens != 0 {
		t.Errorf("empty token should both be 0, got %d/%d", result.OriginalTokens, result.CompressedTokens)
	}
}
