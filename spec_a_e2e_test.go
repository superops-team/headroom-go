package headroom

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestSpecAE2EContentKindsAcrossPublicCompressAPIs(t *testing.T) {
	cases := []struct {
		name    string
		kind    ContentKind
		content string
	}{
		{name: "json", kind: KindJSON, content: prettyJSONFixture(24)},
		{name: "code", kind: KindCode, content: strings.Repeat("func computeValue(input int) int {\n\tif input > 10 {\n\t\treturn input * 2\n\t}\n\treturn input + 1\n}\n", 12)},
		{name: "text", kind: KindText, content: strings.Repeat("This repeated paragraph explains the same operational context and should be compressed.\n", 32)},
		{name: "diff", kind: KindDiff, content: "diff --git a/app.go b/app.go\n--- a/app.go\n+++ b/app.go\n@@ -1,3 +1,3 @@\n" + strings.Repeat("-old behavior line\n+new behavior line\n", 48)},
		{name: "log", kind: KindLog, content: strings.Repeat("[INFO] request completed status=200 latency=10ms\n", 30) + "[ERROR] request failed status=500 err=boom\n" + strings.Repeat("[DEBUG] retry scheduler idle\n", 30)},
		{name: "search", kind: KindSearch, content: strings.Repeat("internal/a.go:10:return nil\n", 24) + strings.Repeat("internal/b.go-22-func handler() error\n", 24)},
		{name: "tabular", kind: KindTabular, content: "name,state,count\nalpha,green,10\nbeta,yellow,20\ngamma,red,30\n" + strings.Repeat("delta,green,40\n", 20)},
		{name: "html", kind: KindHTML, content: "<!doctype html><html><head><style>.x { color: red; }</style><script>console.log('x')</script></head><body><!-- remove me --><article><h1>Hello</h1><p>World</p></article></body></html>"},
	}

	router := NewContentRouter()
	for _, tc := range cases {
		t.Run(tc.name+"/detect", func(t *testing.T) {
			if got := router.Detect(tc.content); got != tc.kind {
				t.Fatalf("Detect()=%s, want %s", got.String(), tc.kind.String())
			}
		})

		for _, pipeline := range []bool{false, true} {
			mode := "legacy"
			if pipeline {
				mode = "pipeline"
			}
			t.Run(tc.name+"/Compress/"+mode, func(t *testing.T) {
				opts := DefaultOptions()
				opts.EnablePipeline = pipeline
				opts.Reversible = false
				opts.Aggressiveness = 0.7
				res, err := Compress([]Message{{Role: "system", Content: "stable prefix"}, {Role: "user", Name: "case-" + tc.name, Content: tc.content}, {Role: "assistant", Content: "assistant output must pass through"}}, opts)
				if err != nil {
					t.Fatalf("Compress() error: %v", err)
				}
				if len(res.Messages) != 3 {
					t.Fatalf("messages len=%d, want 3", len(res.Messages))
				}
				if res.Messages[1].Role != "user" || res.Messages[1].Name != "case-"+tc.name || res.Messages[1].Content == "" {
					t.Fatalf("bad response message: %#v", res.Messages[1])
				}
				if res.Messages[2].Content != "assistant output must pass through" {
					t.Fatalf("assistant message changed: %q", res.Messages[2].Content)
				}
				if res.OriginalTokens < 0 || res.CompressedTokens < 0 {
					t.Fatalf("negative token counts: before=%d after=%d", res.OriginalTokens, res.CompressedTokens)
				}
				if len(res.Steps) == 0 {
					t.Fatalf("expected compression steps")
				}
			})

			t.Run(tc.name+"/CompressString/"+mode, func(t *testing.T) {
				opts := DefaultOptions()
				opts.EnablePipeline = pipeline
				opts.Reversible = false
				out, err := CompressString(tc.content, opts)
				if err != nil {
					t.Fatalf("CompressString() error: %v", err)
				}
				if out == "" {
					t.Fatalf("CompressString() returned empty output")
				}
			})
		}
	}
}

func TestSpecAE2EPipelineLegacyTokenizerCCRAlignerAndTags(t *testing.T) {
	t.Run("legacy reversible store retrieve", func(t *testing.T) {
		src := strings.Repeat("legacy reversible line\n", 80)
		opts := DefaultOptions()
		opts.Reversible = true
		res, err := Compress([]Message{{Role: "user", Content: src}}, opts)
		if err != nil {
			t.Fatalf("Compress() error: %v", err)
		}
		assertRetrieveMarkerRestores(t, res.Messages[0].Content, src)
	})

	t.Run("pipeline reversible store retrieve", func(t *testing.T) {
		src := strings.Repeat("pipeline reversible line\n", 80)
		opts := DefaultOptions()
		opts.EnablePipeline = true
		opts.Reversible = true
		res, err := Compress([]Message{{Role: "user", Content: src}}, opts)
		if err != nil {
			t.Fatalf("Compress() error: %v", err)
		}
		assertRetrieveMarkerRestores(t, res.Messages[0].Content, src)
	})

	t.Run("tokenizer fallback tiktoken stub hf stub", func(t *testing.T) {
		fallback, warnings, err := NewTokenizer(TokenizerConfig{Backend: TokenizerFallback})
		if err != nil || len(warnings) != 0 || fallback.Name() != "fallback-rune" {
			t.Fatalf("fallback tokenizer got tok=%v warnings=%#v err=%v", fallback, warnings, err)
		}
		for _, backend := range []TokenizerBackend{TokenizerTiktoken, TokenizerHF} {
			tok, warnings, err := NewTokenizer(TokenizerConfig{Backend: backend, AllowFallback: true})
			if err != nil {
				t.Fatalf("%s fallback error: %v", backend, err)
			}
			if tok.Name() != "fallback-rune" || len(warnings) == 0 || warnings[0].Code != "tokenizer_fallback" {
				t.Fatalf("%s did not downgrade to fallback with warning: tok=%s warnings=%#v", backend, tok.Name(), warnings)
			}
		}
	})

	t.Run("cache aligner prefix", func(t *testing.T) {
		aligned := NewCacheAligner(CacheAlignerConfig{Enabled: true, Version: "v-spec-a"}).Align("payload")
		if !strings.HasPrefix(aligned, "[headroom/v-spec-a]\n") {
			t.Fatalf("missing aligned prefix: %q", aligned)
		}
		if got := NewCacheAligner(CacheAlignerConfig{Enabled: false}).Align("payload"); got != "payload" {
			t.Fatalf("disabled aligner changed payload: %q", got)
		}
	})

	t.Run("tag protector preserves custom tags", func(t *testing.T) {
		src := "before <system-reminder>keep exact <tool_call>{}</tool_call></system-reminder> <div>html</div> after"
		protector := NewTagProtector()
		protected := protector.Protect(src)
		if strings.Contains(protected.Text, "system-reminder") || len(protected.Placeholders) == 0 {
			t.Fatalf("custom tag was not protected: %#v", protected)
		}
		restored, warnings := protector.Restore(protected)
		if len(warnings) != 0 || restored != src {
			t.Fatalf("restore got %q warnings=%#v, want original", restored, warnings)
		}
	})
}

func TestSpecAE2EBoundaryAndErrorScenarios(t *testing.T) {
	t.Run("empty and whitespace inputs", func(t *testing.T) {
		res, err := Compress([]Message{{Role: "user", Content: ""}, {Role: "user", Content: "   \n\t"}}, DefaultOptions())
		if err != nil {
			t.Fatalf("Compress() error: %v", err)
		}
		if len(res.Messages) != 2 || res.Messages[0].Content != "" || res.Messages[1].Content != "   \n\t" {
			t.Fatalf("empty/whitespace response mismatch: %#v", res.Messages)
		}
		out, err := CompressString("", DefaultOptions())
		if err != nil || out != "" {
			t.Fatalf("CompressString(empty) got %q err=%v", out, err)
		}
	})

	t.Run("extreme policy values and long input", func(t *testing.T) {
		src := strings.Repeat("extreme long input line with repeated words and symbols !!!\n", 5000)
		opts := DefaultOptions()
		opts.EnablePipeline = true
		opts.Reversible = false
		opts.Aggressiveness = 2.0
		res, err := Compress([]Message{{Role: "user", Content: src}}, opts)
		if err != nil {
			t.Fatalf("Compress(long) error: %v", err)
		}
		if len(res.Messages) != 1 || res.Messages[0].Content == "" || len(res.Messages[0].Content) > len(src) {
			t.Fatalf("bad long-input response length=%d original=%d", len(res.Messages[0].Content), len(src))
		}
		foundClampWarning := false
		for _, warning := range res.Warnings {
			if warning.Code == "aggressiveness_clamped" {
				foundClampWarning = true
			}
		}
		if !foundClampWarning {
			t.Fatalf("expected aggressiveness clamp warning, got %#v", res.Warnings)
		}
	})

	t.Run("invalid input and unsupported content kind degrade safely", func(t *testing.T) {
		pipeline := NewDefaultPipeline()
		pr := pipeline.Run(`{"a":1,}`, CompressionContext{ContentKind: KindJSON, Tokenizer: FallbackTokenizer{}, Aggressiveness: 0.5}, DefaultCompressionPolicy(0.5))
		if pr.Output != `{"a":1,}` || len(pr.Warnings) == 0 || pr.Warnings[0].Code != "transform_error_invalid_input" {
			t.Fatalf("invalid JSON should warn and keep input, got output=%q warnings=%#v", pr.Output, pr.Warnings)
		}

		registry := NewCompressorRegistry()
		registry.Register(NewCompressorFunc(KindText, func(content string, opts Options) (string, error) { return "fallback:" + content, nil }))
		out, err := registry.Compress(ContentKind(999), "payload", DefaultOptions())
		if err != nil || out != "fallback:payload" {
			t.Fatalf("unsupported ContentKind fallback got %q err=%v", out, err)
		}

		_, err = NewCompressorFunc(KindText, nil).Compress("payload", DefaultOptions())
		if err == nil {
			t.Fatalf("nil compressor function should return an error")
		}
	})

	t.Run("tokenizer error downgrade and hard failure", func(t *testing.T) {
		pipeline := NewDefaultPipeline()
		pr := pipeline.Run("hello world", CompressionContext{ContentKind: KindText, Tokenizer: errorTokenizer{}, Aggressiveness: 0.5}, DefaultCompressionPolicy(0.5))
		if pr.TokensBefore != 2 || len(pr.Warnings) == 0 || pr.Warnings[0].Code != "tokenizer_count_error" {
			t.Fatalf("pipeline tokenizer fallback mismatch: %#v", pr)
		}

		_, _, err := NewTokenizer(TokenizerConfig{Backend: TokenizerHF, AllowFallback: false})
		if err != ErrTokenizerNotImplemented {
			t.Fatalf("expected tokenizer stub hard failure, got %v", err)
		}

		opts := DefaultOptions()
		opts.TokenizerConfig = TokenizerConfig{Backend: TokenizerBackend("unsupported"), AllowFallback: false}
		if _, err := Compress([]Message{{Role: "user", Content: "hello"}}, opts); err == nil {
			t.Fatalf("Compress should fail for unsupported tokenizer without fallback")
		}
	})

	t.Run("public api construction response fields", func(t *testing.T) {
		opts := DefaultOptions()
		opts.TokenLimit = 99999
		engine, warnings := NewCompressionEngine(opts)
		if engine == nil || len(warnings) != 0 {
			t.Fatalf("NewCompressionEngine got engine=%v warnings=%#v", engine, warnings)
		}
		res, err := engine.Compress([]Message{{Role: "user", Name: "named", Content: "short public api request"}}, opts)
		if err != nil {
			t.Fatalf("engine.Compress() error: %v", err)
		}
		if len(res.Messages) != 1 || res.Messages[0].Name != "named" || res.Messages[0].Content != "short public api request" || res.Steps[0].Skipped != true {
			t.Fatalf("unexpected public API response: %#v", res)
		}

		ccr := NewCCR(CCRConfig{TTL: time.Hour, MaxEntries: 2})
		id := ccr.Store("original", "compressed", KindText)
		if got, ok := ccr.Retrieve(id); !ok || got != "original" {
			t.Fatalf("CCR Retrieve got %q ok=%v", got, ok)
		}
		count, bytes := ccr.Stats()
		if count != 1 || bytes != len("original") {
			t.Fatalf("CCR Stats got count=%d bytes=%d", count, bytes)
		}
	})
}

func prettyJSONFixture(n int) string {
	var b strings.Builder
	b.WriteString("{\n  \"status\": \"ok\",\n  \"items\": [\n")
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteString(",\n")
		}
		b.WriteString(fmt.Sprintf("    {\"id\": %d, \"name\": \"item-%d\", \"enabled\": true}", i, i))
	}
	b.WriteString("\n  ]\n}")
	return b.String()
}

func assertRetrieveMarkerRestores(t *testing.T, content, want string) {
	t.Helper()
	marker := "[headroom:retrieve id="
	idx := strings.LastIndex(content, marker)
	if idx < 0 {
		t.Fatalf("retrieve marker missing in %q", content)
	}
	id := strings.TrimSuffix(content[idx+len(marker):], "]")
	got, ok := getPackageCCR().Retrieve(id)
	if !ok {
		t.Fatalf("retrieve id %q not found", id)
	}
	if got != want {
		t.Fatalf("Retrieve(%q) got original length=%d, want %d", id, len(got), len(want))
	}
}
