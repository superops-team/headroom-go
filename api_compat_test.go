package headroom

import "testing"

var _ func([]Message, Options) (*Result, error) = Compress
var _ func(string, Options) (string, error) = CompressString
var _ func(Options) (*CompressionEngine, []Warning) = NewCompressionEngine
var _ func(string, SmartCrushConfig) (string, error) = SmartCrushJSON
var _ func(string, CodeConfig) string = CompressCode
var _ func(string, TextConfig) string = CompressText
var _ func(CacheAlignerConfig) *CacheAligner = NewCacheAligner

type compatCompressor struct{}

func (compatCompressor) Kind() ContentKind { return KindText }
func (compatCompressor) Compress(content string, opts Options) (string, error) {
	return content, nil
}

var _ Compressor = compatCompressor{}

func TestPublicAPICompatibility(t *testing.T) {
	opts := Options{Aggressiveness: 0.5, Reversible: true, AlignPrefix: false, TokenLimit: 10}
	msgs := []Message{{Role: "user", Content: "hello world", Name: "n"}}
	result, err := Compress(msgs, opts)
	if err != nil {
		t.Fatal(err)
	}
	var _ []Message = result.Messages
	var _ int = result.OriginalTokens
	var _ int = result.CompressedTokens
	var _ float64 = result.Savings
	var _ []Warning = result.Warnings
	var _ []CompressionStep = result.Steps
	var _ string = result.Messages[0].Role
	var _ string = result.Messages[0].Content
	var _ string = result.Messages[0].Name
	var warning Warning
	var _ string = warning.Code
	var _ string = warning.Component
	var _ string = warning.Message
	var step CompressionStep
	var _ string = step.Name
	var _ string = step.Kind
	var _ int = step.TokensBefore
	var _ int = step.TokensAfter
	var _ bool = step.Skipped
	var _ string = step.Reason
	if _, err := CompressString("hello", opts); err != nil {
		t.Fatal(err)
	}

	compressionCfg := CompressionConfig{Aggressiveness: 0.5}
	codeCfg := CodeConfig{Aggressiveness: 0.5}
	textCfg := TextConfig{Aggressiveness: 0.5}
	smartCfg := SmartCrushConfig{Aggressiveness: 0.5, Observer: NoopObserver{}}
	var _ CompressionConfig = compressionCfg
	var _ CodeConfig = codeCfg
	var _ TextConfig = textCfg
	var _ SmartCrushConfig = smartCfg
	if _, err := SmartCrushJSON(`{"items":[1,2,3,4,5,6]}`, smartCfg); err != nil {
		t.Fatal(err)
	}
	if CompressCode("// comment\nfunc main() {}", codeCfg) == "" {
		t.Fatal("CompressCode returned empty output")
	}
	if CompressText("hello hello", textCfg) == "" {
		t.Fatal("CompressText returned empty output")
	}

	alignerCfg := CacheAlignerConfig{Enabled: true, Version: PrefixVersion}
	aligner := NewCacheAligner(alignerCfg)
	var _ *CacheAligner = aligner
	if aligner.Align("content") == "" {
		t.Fatal("CacheAligner returned empty output")
	}

	registry := NewCompressorRegistry()
	registry.Register(compatCompressor{})
	registry.Register(NewCompressorFunc(KindJSON, func(content string, opts Options) (string, error) {
		return content, nil
	}))
	if out, err := registry.Compress(KindText, "abc", opts); err != nil || out != "abc" {
		t.Fatalf("registry compatibility got=%q err=%v", out, err)
	}
	if DefaultCompressorRegistry() == nil {
		t.Fatal("DefaultCompressorRegistry returned nil")
	}

	var _ ContentKind = KindDiff
	var _ ContentKind = KindLog
	var _ ContentKind = KindSearch
	var _ ContentKind = KindTabular
	var _ ContentKind = KindSpreadsheet
	var _ ContentKind = KindHTML
	var _ ContentKind = KindUnknown
}
