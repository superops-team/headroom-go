package headroom

import (
	"errors"
	"testing"
)

func TestCompressorRegistryReplaceAndFallback(t *testing.T) {
	r := NewCompressorRegistry()
	r.Register(NewCompressorFunc(KindText, func(content string, opts Options) (string, error) { return "text:" + content, nil }))
	r.Register(NewCompressorFunc(KindJSON, func(content string, opts Options) (string, error) { return "json", nil }))
	if got, _ := r.Compress(KindJSON, "{}", Options{}); got != "json" {
		t.Fatalf("json got %q", got)
	}
	if got, _ := r.Compress(KindDiff, "abc", Options{}); got != "text:abc" {
		t.Fatalf("fallback got %q", got)
	}
}

func TestCompressorFuncNilReturnsError(t *testing.T) {
	c := NewCompressorFunc(KindText, nil)
	got, err := c.Compress("abc", Options{})
	if err == nil {
		t.Fatal("expected nil compressor function to return an error")
	}
	if got != "abc" {
		t.Fatalf("nil compressor should preserve input, got %q", got)
	}
}

func TestCompressLegacyWrapsCompressorErrorChain(t *testing.T) {
	sentinel := errors.New("sentinel compressor failure")
	registry := DefaultCompressorRegistry()
	original, ok := registry.Lookup(KindText)
	if !ok {
		t.Fatal("default text compressor missing")
	}
	registry.Register(NewCompressorFunc(KindText, func(content string, opts Options) (string, error) {
		return content, sentinel
	}))
	t.Cleanup(func() { registry.Register(original) })

	_, err := Compress([]Message{{Role: "user", Content: "plain text input"}}, DefaultOptions())
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected wrapped sentinel error, got %v", err)
	}
}
