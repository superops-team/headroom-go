package compressors_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	headroom "github.com/superops-team/headroom-go"
)

type customTestCompressor struct {
	kind headroom.ContentKind
	fn   func(string, headroom.Options) (string, error)
}

func (c customTestCompressor) Kind() headroom.ContentKind { return c.kind }
func (c customTestCompressor) Compress(content string, opts headroom.Options) (string, error) {
	return c.fn(content, opts)
}

func jsonEqual(t *testing.T, got, want string) bool {
	t.Helper()
	var gotValue interface{}
	if err := json.Unmarshal([]byte(got), &gotValue); err != nil {
		t.Fatalf("got invalid JSON %q: %v", got, err)
	}
	var wantValue interface{}
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatalf("want invalid JSON %q: %v", want, err)
	}
	return jsonNormalized(gotValue) == jsonNormalized(wantValue)
}

func jsonNormalized(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func TestCompressorRegistryReplaceAndFallback(t *testing.T) {
	r := headroom.NewCompressorRegistry()
	r.Register(headroom.NewCompressorFunc(headroom.KindText, func(content string, opts headroom.Options) (string, error) { return "text:" + content, nil }))
	r.Register(headroom.NewCompressorFunc(headroom.KindJSON, func(content string, opts headroom.Options) (string, error) { return "json", nil }))
	if got, _ := r.Compress(headroom.KindJSON, "{}", headroom.Options{}); got != "json" {
		t.Fatalf("json got %q", got)
	}
	if got, _ := r.Compress(headroom.KindDiff, "abc", headroom.Options{}); got != "text:abc" {
		t.Fatalf("fallback got %q", got)
	}
}

func TestCompressorFuncNilReturnsError(t *testing.T) {
	c := headroom.NewCompressorFunc(headroom.KindText, nil)
	got, err := c.Compress("abc", headroom.Options{})
	if err == nil {
		t.Fatal("expected nil compressor function to return an error")
	}
	if got != "abc" {
		t.Fatalf("nil compressor should preserve input, got %q", got)
	}
}

func TestCompressionConfigCompatibilityAdapters(t *testing.T) {
	cfg := headroom.CompressionConfig{Aggressiveness: 0.5}
	var codeCfg headroom.CodeConfig = cfg
	var textCfg headroom.TextConfig = cfg
	var smartCfg = headroom.SmartCrushConfig{Aggressiveness: cfg.Aggressiveness}

	if got := headroom.CompressCode("// comment\nfunc main() {}", codeCfg); strings.Contains(got, "comment") {
		t.Fatalf("code config adapter changed compression behavior: %q", got)
	}
	if got := headroom.CompressText("the server is running", textCfg); !strings.Contains(got, "server") {
		t.Fatalf("text config adapter changed compression behavior: %q", got)
	}
	if got, err := headroom.SmartCrushJSON(`{"a":1,"b":null}`, smartCfg); err != nil || !jsonEqual(t, got, `{"a":1}`) {
		t.Fatalf("smart config behavior changed: got=%q err=%v", got, err)
	}
}

func TestCompressorInterfaceAndDefaultRegistryCompatibility(t *testing.T) {
	var _ headroom.Compressor = customTestCompressor{}
	r := headroom.NewCompressorRegistry()
	r.Register(customTestCompressor{kind: headroom.KindText, fn: func(content string, opts headroom.Options) (string, error) {
		return "custom:" + content, nil
	}})
	got, err := r.Compress(headroom.KindText, "abc", headroom.Options{})
	if err != nil || got != "custom:abc" {
		t.Fatalf("custom compressor got=%q err=%v", got, err)
	}

	funcCompressor := headroom.NewCompressorFunc(headroom.KindText, func(content string, opts headroom.Options) (string, error) {
		return "func:" + content, nil
	})
	got, err = funcCompressor.Compress("abc", headroom.Options{})
	if err != nil || got != "func:abc" {
		t.Fatalf("compressor func got=%q err=%v", got, err)
	}

	jsonOut, err := headroom.DefaultCompressorRegistry().Compress(headroom.KindJSON, `{"items":[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20]}`, headroom.Options{Aggressiveness: 0.5})
	if err != nil || jsonOut == "" {
		t.Fatalf("default JSON compressor got=%q err=%v", jsonOut, err)
	}
}

func TestRegistryRoutesJSONCodeTextAndPropagatesErrors(t *testing.T) {
	r := headroom.NewCompressorRegistry()
	r.Register(headroom.NewCompressorFunc(headroom.KindJSON, func(content string, opts headroom.Options) (string, error) { return "json:" + content, nil }))
	r.Register(headroom.NewCompressorFunc(headroom.KindCode, func(content string, opts headroom.Options) (string, error) { return "code:" + content, nil }))
	r.Register(headroom.NewCompressorFunc(headroom.KindText, func(content string, opts headroom.Options) (string, error) { return "text:" + content, nil }))
	for _, tc := range []struct {
		kind headroom.ContentKind
		want string
	}{
		{headroom.KindJSON, "json:{}"},
		{headroom.KindCode, "code:func main() {}"},
		{headroom.KindText, "text:hello"},
	} {
		input := map[headroom.ContentKind]string{headroom.KindJSON: "{}", headroom.KindCode: "func main() {}", headroom.KindText: "hello"}[tc.kind]
		got, err := r.Compress(tc.kind, input, headroom.Options{})
		if err != nil || got != tc.want {
			t.Fatalf("kind %s got=%q err=%v want=%q", tc.kind.String(), got, err, tc.want)
		}
	}

	sentinel := errors.New("sentinel route error")
	r.Register(headroom.NewCompressorFunc(headroom.KindJSON, func(content string, opts headroom.Options) (string, error) { return "partial", sentinel }))
	got, err := r.Compress(headroom.KindJSON, "{}", headroom.Options{})
	if !errors.Is(err, sentinel) || got != "partial" {
		t.Fatalf("sentinel not propagated: got=%q err=%v", got, err)
	}
}

func TestRegistryExtendedKindsFallbackToTextOrOriginal(t *testing.T) {
	for _, kind := range []headroom.ContentKind{headroom.KindDiff, headroom.KindLog, headroom.KindSearch, headroom.KindTabular, headroom.KindSpreadsheet, headroom.KindHTML, headroom.KindUnknown} {
		t.Run(kind.String(), func(t *testing.T) {
			r := headroom.NewCompressorRegistry()
			called := false
			r.Register(headroom.NewCompressorFunc(headroom.KindText, func(content string, opts headroom.Options) (string, error) {
				called = true
				return "text:" + content, nil
			}))
			got, err := r.Compress(kind, "payload", headroom.Options{})
			if err != nil || got != "text:payload" || !called {
				t.Fatalf("fallback text got=%q called=%v err=%v", got, called, err)
			}

			empty := headroom.NewCompressorRegistry()
			got, err = empty.Compress(kind, "payload", headroom.Options{})
			if err != nil || got != "payload" {
				t.Fatalf("empty fallback got=%q err=%v", got, err)
			}
		})
	}
}

func TestCompressLegacyWrapsCompressorErrorChain(t *testing.T) {
	sentinel := errors.New("sentinel compressor failure")
	registry := headroom.DefaultCompressorRegistry()
	original, ok := registry.Lookup(headroom.KindText)
	if !ok {
		t.Fatal("default text compressor missing")
	}
	registry.Register(headroom.NewCompressorFunc(headroom.KindText, func(content string, opts headroom.Options) (string, error) {
		return content, sentinel
	}))
	t.Cleanup(func() { registry.Register(original) })

	_, err := headroom.Compress([]headroom.Message{{Role: "user", Content: "plain text input"}}, headroom.DefaultOptions())
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected wrapped sentinel error, got %v", err)
	}
}
