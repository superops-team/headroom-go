package tokenizer

import "testing"

func TestFallbackTokenizerCount(t *testing.T) {
	tok := FallbackTokenizer{}
	if got, _ := tok.Count(""); got != 0 {
		t.Fatalf("empty got %d", got)
	}
	if got, _ := tok.Count("hello world"); got != 2 {
		t.Fatalf("ascii got %d", got)
	}
	if got, _ := tok.Count("你好🙂"); got != 3 {
		t.Fatalf("cjk emoji got %d", got)
	}
	batch, err := tok.CountBatch([]string{"a b", "你好"})
	if err != nil {
		t.Fatal(err)
	}
	if len(batch) != 2 || batch[0] != 2 || batch[1] != 2 {
		t.Fatalf("bad batch %#v", batch)
	}
}

func TestTokenizerFactoryFallbackWarning(t *testing.T) {
	tok, warnings, err := NewTokenizer(TokenizerConfig{Backend: TokenizerTiktoken, AllowFallback: true})
	if err != nil {
		t.Fatal(err)
	}
	if tok.Name() != "fallback-rune" {
		t.Fatalf("got %s", tok.Name())
	}
	if len(warnings) == 0 {
		t.Fatal("expected fallback warning")
	}
	_, _, err = NewTokenizer(TokenizerConfig{Backend: TokenizerHF, AllowFallback: false})
	if err != ErrTokenizerNotImplemented {
		t.Fatalf("got %v", err)
	}
}

func TestCompressTokenizerNoFallbackReturnsError(t *testing.T) {
	_, _, err := NewTokenizer(TokenizerConfig{Backend: TokenizerBackend("missing"), AllowFallback: false})
	if err == nil || err.Error() != "unknown tokenizer backend" {
		t.Fatalf("expected tokenizer construction error, got %v", err)
	}
}
