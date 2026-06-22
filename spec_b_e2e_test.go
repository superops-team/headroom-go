package headroom

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSpecBE2EAllBusinessContentKindsAcrossPublicAPIs(t *testing.T) {
	cases := specBE2EContentKindFixtures()
	router := NewContentRouter()

	for _, tc := range cases {
		t.Run(tc.name+"/detect", func(t *testing.T) {
			if got := router.Detect(tc.content); got != tc.kind {
				t.Fatalf("Detect()=%s, want %s", got.String(), tc.kind.String())
			}
		})

		for _, mode := range []struct {
			name     string
			pipeline bool
		}{
			{name: "legacy", pipeline: false},
			{name: "pipeline", pipeline: true},
		} {
			t.Run(tc.name+"/Compress/"+mode.name, func(t *testing.T) {
				opts := DefaultOptions()
				opts.EnablePipeline = mode.pipeline
				opts.Reversible = false
				opts.Aggressiveness = 0.8
				res, err := Compress([]Message{
					{Role: "system", Content: "stable e2e prefix"},
					{Role: "user", Name: "spec-b-" + tc.name, Content: tc.content},
					{Role: "assistant", Content: "assistant pass-through"},
				}, opts)
				if err != nil {
					t.Fatalf("Compress() error: %v", err)
				}
				if len(res.Messages) != 3 {
					t.Fatalf("messages len=%d, want 3", len(res.Messages))
				}
				if res.Messages[1].Role != "user" || res.Messages[1].Name != "spec-b-"+tc.name || res.Messages[1].Content == "" {
					t.Fatalf("bad user response message: %#v", res.Messages[1])
				}
				if res.Messages[2].Content != "assistant pass-through" {
					t.Fatalf("assistant message changed: %q", res.Messages[2].Content)
				}
				if res.OriginalTokens < res.CompressedTokens && len(res.Messages[1].Content) > len(tc.content) {
					t.Fatalf("compression expanded both tokens and bytes: before=%d after=%d in=%d out=%d", res.OriginalTokens, res.CompressedTokens, len(tc.content), len(res.Messages[1].Content))
				}
				if len(res.Steps) == 0 {
					t.Fatal("expected at least one compression step")
				}
				if mode.pipeline {
					for _, step := range res.Steps {
						if step.Name == "legacy_compress" {
							t.Fatalf("pipeline path used legacy helper step: %#v", res.Steps)
						}
					}
				}
			})

			t.Run(tc.name+"/CompressString/"+mode.name, func(t *testing.T) {
				opts := DefaultOptions()
				opts.EnablePipeline = mode.pipeline
				opts.Reversible = false
				out, err := CompressString(tc.content, opts)
				if err != nil {
					t.Fatalf("CompressString() error: %v", err)
				}
				if out == "" {
					t.Fatal("CompressString() returned empty output")
				}
				if strings.Contains(out, "headroom:retrieve") {
					t.Fatalf("non-reversible CompressString output has retrieve marker: %q", out)
				}
			})
		}
	}
}

func TestSpecBE2EPipelineLegacyTokenizerCCRAlignerTags(t *testing.T) {
	t.Run("pipeline compression path records non legacy steps", func(t *testing.T) {
		opts := DefaultOptions()
		opts.EnablePipeline = true
		opts.Reversible = false
		res, err := Compress([]Message{{Role: "user", Content: strings.Repeat("[INFO] service=api heartbeat=ok latency=12ms\n", 120)}}, opts)
		if err != nil {
			t.Fatalf("pipeline Compress() error: %v", err)
		}
		if len(res.Steps) == 0 {
			t.Fatal("pipeline should produce steps")
		}
		for _, step := range res.Steps {
			if step.Name == "legacy_compress" {
				t.Fatalf("pipeline unexpectedly used legacy step: %#v", res.Steps)
			}
		}
	})

	t.Run("legacy compression path records legacy helper step", func(t *testing.T) {
		opts := DefaultOptions()
		opts.Reversible = false
		res, err := Compress([]Message{{Role: "user", Content: strings.Repeat("legacy repeated content line\n", 80)}}, opts)
		if err != nil {
			t.Fatalf("legacy Compress() error: %v", err)
		}
		if len(res.Steps) != 1 || res.Steps[0].Name != "legacy_compress" {
			t.Fatalf("legacy path step mismatch: %#v", res.Steps)
		}
	})

	t.Run("tokenizer backends fallback tiktoken stub hf stub and batch", func(t *testing.T) {
		fallback, warnings, err := NewTokenizer(TokenizerConfig{Backend: TokenizerFallback})
		if err != nil || len(warnings) != 0 || fallback.Name() != "fallback-rune" {
			t.Fatalf("fallback tokenizer got name=%q warnings=%#v err=%v", tokenizerName(fallback), warnings, err)
		}
		counts, err := fallback.CountBatch([]string{"hello world", "你好🙂"})
		if err != nil || len(counts) != 2 || counts[0] != 2 || counts[1] != 3 {
			t.Fatalf("fallback CountBatch got counts=%#v err=%v", counts, err)
		}
		for _, backend := range []TokenizerBackend{TokenizerTiktoken, TokenizerHF} {
			tok, warnings, err := NewTokenizer(TokenizerConfig{Backend: backend, AllowFallback: true})
			if err != nil {
				t.Fatalf("%s fallback error: %v", backend, err)
			}
			if tok.Name() != "fallback-rune" || len(warnings) != 1 || warnings[0].Code != "tokenizer_fallback" {
				t.Fatalf("%s did not degrade to fallback with warning: tok=%q warnings=%#v", backend, tok.Name(), warnings)
			}
			_, _, err = NewTokenizer(TokenizerConfig{Backend: backend, AllowFallback: false})
			if !errors.Is(err, ErrTokenizerNotImplemented) {
				t.Fatalf("%s without fallback got err=%v, want ErrTokenizerNotImplemented", backend, err)
			}
		}
	})

	t.Run("CCR reversible store retrieve", func(t *testing.T) {
		ccr := NewCCR(CCRConfig{TTL: time.Hour, MaxEntries: 4})
		id := ccr.Store("original payload", "compressed payload", KindText)
		if got, ok := ccr.Retrieve(id); !ok || got != "original payload" {
			t.Fatalf("Retrieve(%q) got %q ok=%v", id, got, ok)
		}
		count, bytes := ccr.Stats()
		if count != 1 || bytes != len("original payload") {
			t.Fatalf("Stats() count=%d bytes=%d", count, bytes)
		}
	})

	t.Run("CacheAligner prefix alignment", func(t *testing.T) {
		aligner := NewCacheAligner(CacheAlignerConfig{Enabled: true, Version: "spec-b"})
		if got := aligner.Align("payload"); got != "[headroom/spec-b]\npayload" {
			t.Fatalf("Align()=%q", got)
		}
		if got := NewCacheAligner(CacheAlignerConfig{Enabled: false, Version: "spec-b"}).Align("payload"); got != "payload" {
			t.Fatalf("disabled Align()=%q", got)
		}
	})

	t.Run("Tag Protector protects and restores custom tags", func(t *testing.T) {
		src := "before <system-reminder>keep <tool_call>{\"x\":1}</tool_call></system-reminder><div>html stays visible</div> after"
		protector := NewTagProtector()
		protected := protector.Protect(src)
		if len(protected.Placeholders) == 0 || strings.Contains(protected.Text, "system-reminder") {
			t.Fatalf("custom tag not protected: %#v", protected)
		}
		restored, warnings := protector.Restore(protected)
		if len(warnings) != 0 || restored != src {
			t.Fatalf("Restore() got %q warnings=%#v", restored, warnings)
		}
	})
}

func TestSpecBE2EBoundaryAndExceptionScenarios(t *testing.T) {
	t.Run("empty data and whitespace are preserved", func(t *testing.T) {
		res, err := Compress([]Message{{Role: "user", Content: ""}, {Role: "user", Content: "\n\t  "}}, DefaultOptions())
		if err != nil {
			t.Fatalf("Compress(empty) error: %v", err)
		}
		if len(res.Messages) != 2 || res.Messages[0].Content != "" || res.Messages[1].Content != "\n\t  " {
			t.Fatalf("empty/whitespace changed: %#v", res.Messages)
		}
		out, err := CompressString("", DefaultOptions())
		if err != nil || out != "" {
			t.Fatalf("CompressString(empty)=%q err=%v", out, err)
		}
	})

	t.Run("extreme values and very long input", func(t *testing.T) {
		src := strings.Repeat("extreme input line with repeated words and symbols !!!\n", 7000)
		opts := DefaultOptions()
		opts.EnablePipeline = true
		opts.Reversible = false
		opts.Aggressiveness = 99
		res, err := Compress([]Message{{Role: "user", Content: src}}, opts)
		if err != nil {
			t.Fatalf("Compress(long) error: %v", err)
		}
		if len(res.Messages) != 1 || res.Messages[0].Content == "" || len(res.Messages[0].Content) > len(src) {
			t.Fatalf("long input response invalid: out=%d in=%d", len(res.Messages[0].Content), len(src))
		}
		if !hasWarning(res.Warnings, "aggressiveness_clamped") {
			t.Fatalf("expected aggressiveness_clamped warning, got %#v", res.Warnings)
		}
	})

	t.Run("invalid input unsupported content kind and tokenizer degradation", func(t *testing.T) {
		pipeline := NewDefaultPipeline()
		pr := pipeline.Run(`{"a":1,}`, CompressionContext{ContentKind: KindJSON, Tokenizer: FallbackTokenizer{}, Aggressiveness: 0.5}, DefaultCompressionPolicy(0.5))
		if pr.Output != `{"a":1,}` || !hasWarning(pr.Warnings, "transform_error_invalid_input") {
			t.Fatalf("invalid JSON should warn and preserve input: output=%q warnings=%#v", pr.Output, pr.Warnings)
		}

		registry := NewCompressorRegistry()
		registry.Register(NewCompressorFunc(KindText, func(content string, opts Options) (string, error) { return "fallback:" + content, nil }))
		out, err := registry.Compress(ContentKind(4242), "payload", DefaultOptions())
		if err != nil || out != "fallback:payload" {
			t.Fatalf("unsupported ContentKind fallback got %q err=%v", out, err)
		}

		pr = NewDefaultPipeline().Run("hello world", CompressionContext{ContentKind: KindText, Tokenizer: specBErrorTokenizer{}, Aggressiveness: 0.5}, DefaultCompressionPolicy(0.5))
		if pr.TokensBefore != 2 || !hasWarning(pr.Warnings, "tokenizer_count_error") {
			t.Fatalf("tokenizer error should downgrade to fallback count: %#v", pr)
		}

		opts := DefaultOptions()
		opts.TokenizerConfig = TokenizerConfig{Backend: TokenizerBackend("not-supported"), AllowFallback: false}
		if _, err := Compress([]Message{{Role: "user", Content: "hello"}}, opts); err == nil {
			t.Fatal("unsupported tokenizer without fallback should fail")
		}
	})
}

func TestSpecBE2EBackwardCompatibilityAPIsCompileAndRun(t *testing.T) {
	jsonOut, err := SmartCrushJSON(`{"items":[1,2,3,4,5,6],"drop":null}`, SmartCrushConfig{Aggressiveness: 0.5})
	if err != nil || jsonOut == "" {
		t.Fatalf("SmartCrushJSON()=%q err=%v", jsonOut, err)
	}
	if out := CompressCode("// comment\nfunc main() {\n    return\n}\n", CodeConfig{Aggressiveness: 0.5}); out == "" {
		t.Fatal("CompressCode returned empty output")
	}
	if out := CompressText("hello hello hello", TextConfig{Aggressiveness: 0.5}); out == "" {
		t.Fatal("CompressText returned empty output")
	}
	aligner := NewCacheAligner(CacheAlignerConfig{Enabled: true, Version: PrefixVersion})
	var _ *CacheAligner = aligner
	if !strings.HasPrefix(aligner.Align("payload"), "[headroom/"+PrefixVersion+"]\n") {
		t.Fatalf("NewCacheAligner/CacheAligner prefix mismatch")
	}
	var _ func(string, SmartCrushConfig) (string, error) = SmartCrushJSON
	var _ func(string, CodeConfig) string = CompressCode
	var _ func(string, TextConfig) string = CompressText
	var _ func(CacheAlignerConfig) *CacheAligner = NewCacheAligner
}

func TestSpecBE2ECompressLegacyHelperBehavior(t *testing.T) {
	t.Run("skip helper keeps previous names and reasons", func(t *testing.T) {
		opts := DefaultOptions()
		opts.TokenLimit = 10
		cases := []struct {
			msg        Message
			tokens     int
			wantName   string
			wantReason string
		}{
			{msg: Message{Role: "assistant", Content: "reply"}, tokens: 1, wantName: "skip_assistant", wantReason: "assistant role"},
			{msg: Message{Role: "user", Content: " \n\t"}, tokens: 0, wantName: "skip_empty", wantReason: "empty content"},
			{msg: Message{Role: "user", Content: "short"}, tokens: 1, wantName: "skip_token_limit", wantReason: "below token limit"},
		}
		for _, tc := range cases {
			skipped, step := legacySkipMessage(tc.msg, opts, tc.tokens)
			if !skipped || step.Name != tc.wantName || step.Reason != tc.wantReason || !step.Skipped {
				t.Fatalf("legacySkipMessage(%#v) skipped=%v step=%#v", tc.msg, skipped, step)
			}
		}
	})

	t.Run("route helper preserves detected kind and wraps compressor errors", func(t *testing.T) {
		registry := NewCompressorRegistry()
		registry.Register(NewCompressorFunc(KindJSON, func(content string, opts Options) (string, error) { return "{}", nil }))
		kind, out, err := routeAndCompressLegacy(NewContentRouter(), registry, `{"a": 1}`, DefaultOptions())
		if err != nil || kind != KindJSON || out != "{}" {
			t.Fatalf("routeAndCompressLegacy json got kind=%s out=%q err=%v", kind.String(), out, err)
		}

		sentinel := errors.New("spec-b sentinel")
		registry.Register(NewCompressorFunc(KindText, func(content string, opts Options) (string, error) { return "", sentinel }))
		_, _, err = routeAndCompressLegacy(NewContentRouter(), registry, "plain text input", DefaultOptions())
		if !errors.Is(err, sentinel) || !strings.Contains(err.Error(), "compress Text:") {
			t.Fatalf("routeAndCompressLegacy should wrap sentinel, got %v", err)
		}
	})

	t.Run("post process helper preserves prefix reversible fallback behavior", func(t *testing.T) {
		orig := strings.Repeat("original payload ", 80)
		opts := DefaultOptions()
		opts.Reversible = true
		opts.AlignPrefix = true
		ccr := NewCCR(CCRConfig{TTL: time.Hour})
		content, step, err := postProcessLegacyCompression(orig, "tiny", KindText, opts, FallbackTokenizer{}, 100, NewCacheAligner(CacheAlignerConfig{Enabled: true, Version: "spec-b"}), ccr)
		if err != nil {
			t.Fatalf("postProcessLegacyCompression() error: %v", err)
		}
		if step.Name != "legacy_compress" || step.Kind != KindText.String() || step.Skipped {
			t.Fatalf("post process step mismatch: %#v", step)
		}
		if !strings.HasPrefix(content, "[headroom/spec-b]\n") || !strings.Contains(content, "[headroom:retrieve id=") {
			t.Fatalf("post content missing prefix or retrieve marker: %q", content)
		}
		id := strings.TrimSuffix(content[strings.LastIndex(content, "[headroom:retrieve id=")+len("[headroom:retrieve id="):], "]")
		if got, ok := ccr.Retrieve(id); !ok || got != orig {
			t.Fatalf("post process CCR retrieve got len=%d ok=%v", len(got), ok)
		}

		fallbackOpts := DefaultOptions()
		fallbackOpts.Reversible = false
		content, step, err = postProcessLegacyCompression("short", "this output is longer", KindText, fallbackOpts, FallbackTokenizer{}, 1, NewCacheAligner(CacheAlignerConfig{}), ccr)
		if err != nil {
			t.Fatalf("postProcessLegacyCompression fallback error: %v", err)
		}
		if content != "short" || !step.Skipped || step.Reason != "output not shorter" {
			t.Fatalf("fallback mismatch: %#v content=%q", step, content)
		}
	})

	t.Run("split helpers match compressLegacy externally observed behavior", func(t *testing.T) {
		opts := DefaultOptions()
		opts.Reversible = false
		content := strings.Repeat("same behavior line\n", 60)
		res, err := compressLegacy([]Message{{Role: "user", Content: content}}, opts, FallbackTokenizer{}, nil, nil)
		if err != nil {
			t.Fatalf("compressLegacy() error: %v", err)
		}
		kind, compressed, err := routeAndCompressLegacy(NewContentRouter(), DefaultCompressorRegistry(), content, opts)
		if err != nil {
			t.Fatalf("routeAndCompressLegacy() error: %v", err)
		}
		manualContent, manualStep, err := postProcessLegacyCompression(content, compressed, kind, opts, FallbackTokenizer{}, res.OriginalTokens, NewCacheAligner(CacheAlignerConfig{Enabled: false, Version: PrefixVersion}), getPackageCCR())
		if err != nil {
			t.Fatalf("postProcessLegacyCompression() error: %v", err)
		}
		if len(res.Messages) != 1 || res.Messages[0].Content != manualContent || res.Steps[0] != manualStep {
			t.Fatalf("split helper behavior drift: result=%#v manualContent=%q manualStep=%#v", res, manualContent, manualStep)
		}
	})
}

func specBE2EContentKindFixtures() []struct {
	name    string
	kind    ContentKind
	content string
} {
	return []struct {
		name    string
		kind    ContentKind
		content string
	}{
		{name: "json", kind: KindJSON, content: prettyJSONFixture(32)},
		{name: "code", kind: KindCode, content: strings.Repeat("func computeValue(input int) int {\n\tif input > 10 {\n\t\treturn input * 2\n\t}\n\treturn input + 1\n}\n", 14)},
		{name: "text", kind: KindText, content: strings.Repeat("This operational paragraph repeats context and can be compressed safely.\n", 48)},
		{name: "diff", kind: KindDiff, content: "diff --git a/app.go b/app.go\n--- a/app.go\n+++ b/app.go\n@@ -1,3 +1,3 @@\n" + strings.Repeat("-old behavior line\n+new behavior line\n", 60)},
		{name: "log", kind: KindLog, content: strings.Repeat("[INFO] request completed status=200 latency=10ms\n", 45) + "[ERROR] request failed status=500 err=boom\n" + strings.Repeat("[DEBUG] retry scheduler idle\n", 45)},
		{name: "search", kind: KindSearch, content: strings.Repeat("internal/a.go:10:return nil\n", 30) + strings.Repeat("internal/b.go-22-func handler() error\n", 30)},
		{name: "tabular", kind: KindTabular, content: "name,state,count\nalpha,green,10\nbeta,yellow,20\ngamma,red,30\n" + strings.Repeat("delta,green,40\n", 32)},
		{name: "html", kind: KindHTML, content: "<!doctype html><html><head><style>.x { color: red; }</style><script>console.log('x')</script></head><body><!-- remove me --><article><h1>Hello</h1><p>World</p></article></body></html>"},
	}
}

func hasWarning(warnings []Warning, code string) bool {
	for _, warning := range warnings {
		if warning.Code == code {
			return true
		}
	}
	return false
}

func tokenizerName(tok Tokenizer) string {
	if tok == nil {
		return "<nil>"
	}
	return tok.Name()
}

type specBErrorTokenizer struct{}

func (specBErrorTokenizer) Name() string { return "spec-b-error" }
func (specBErrorTokenizer) Count(string) (int, error) {
	return 0, errors.New("spec-b tokenizer count failed")
}
func (specBErrorTokenizer) CountBatch([]string) ([]int, error) {
	return nil, errors.New("spec-b tokenizer batch failed")
}
