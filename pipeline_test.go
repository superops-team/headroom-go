package headroom

import (
	"errors"
	"strings"
	"testing"
)

type errorTokenizer struct{}

func (errorTokenizer) Name() string { return "error" }
func (errorTokenizer) Count(string) (int, error) {
	return 0, errors.New("tokenizer boom")
}
func (errorTokenizer) CountBatch([]string) ([]int, error) {
	return nil, errors.New("tokenizer boom")
}

func TestCompress_EnablePipelineTextAndJSON(t *testing.T) {
	longText := strings.Repeat("repeat\n", 20)
	msgs := []Message{{Role: "user", Content: longText}, {Role: "user", Content: "{\n  \"a\": 1,\n  \"b\": 2\n}"}}
	opts := DefaultOptions()
	opts.EnablePipeline = true
	opts.Reversible = false
	res, err := Compress(msgs, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Messages) != 2 {
		t.Fatalf("messages len %d", len(res.Messages))
	}
	if !strings.Contains(res.Messages[0].Content, "[x20]") {
		t.Fatalf("text not compressed: %q", res.Messages[0].Content)
	}
	if res.Messages[1].Content != `{"a":1,"b":2}` {
		t.Fatalf("json not minified: %q", res.Messages[1].Content)
	}
	if len(res.Steps) == 0 {
		t.Fatal("expected steps")
	}
}

func TestCompress_EnablePipelineTagProtection(t *testing.T) {
	src := "<system-reminder>must keep</system-reminder>\n" + strings.Repeat("same line\n", 10)
	opts := DefaultOptions()
	opts.EnablePipeline = true
	opts.Reversible = false
	res, err := Compress([]Message{{Role: "user", Content: src}}, opts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Messages[0].Content, "<system-reminder>must keep</system-reminder>") {
		t.Fatalf("tag lost: %s", res.Messages[0].Content)
	}
}

func TestCompress_EnablePipelineReversibleReformatStoresOriginal(t *testing.T) {
	src := strings.Repeat("repeat\n", 80)
	opts := DefaultOptions()
	opts.EnablePipeline = true
	opts.Reversible = true
	res, err := Compress([]Message{{Role: "user", Content: src}}, opts)
	if err != nil {
		t.Fatal(err)
	}
	content := res.Messages[0].Content
	marker := "[headroom:retrieve id="
	idx := strings.LastIndex(content, marker)
	if idx < 0 {
		t.Fatalf("expected retrieve id in pipeline reversible output: %q", content)
	}
	id := strings.TrimSuffix(content[idx+len(marker):], "]")
	restored, ok := getPackageCCR().Retrieve(id)
	if !ok {
		t.Fatalf("retrieve id %q not found", id)
	}
	if restored != src {
		t.Fatalf("retrieve restored %q want original %q", restored, src)
	}
}

func TestCompress_EnablePipelineDiffLogSearch(t *testing.T) {
	opts := DefaultOptions()
	opts.EnablePipeline = true
	opts.Reversible = true
	cases := []string{
		"diff --git a/a.go b/a.go\n--- a/a.go\n+++ b/a.go\n@@ -1 +1 @@\n" + strings.Repeat("-old\n+new\n", 40),
		strings.Repeat("[INFO] ok\n", 20) + "[ERROR] failed\n" + strings.Repeat("[DEBUG] skip\n", 20),
		strings.Repeat("a.go:10:func main\n", 30) + strings.Repeat("b.go-11-return nil\n", 30),
	}
	for _, src := range cases {
		res, err := Compress([]Message{{Role: "user", Content: src}}, opts)
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Messages[0].Content) >= len(src) {
			t.Fatalf("not shorter: %d >= %d", len(res.Messages[0].Content), len(src))
		}
	}
}

func TestTransformErrorWarningContract(t *testing.T) {
	terr := NewTransformError(TransformErrorInvalidInput, "json_minifier", "bad input", errors.New("boom"))
	if terr.Kind != TransformErrorInvalidInput || !strings.Contains(terr.Error(), "invalid_input") {
		t.Fatalf("unexpected transform error: %#v %s", terr, terr.Error())
	}
	for _, kind := range []TransformErrorKind{TransformErrorInvalidInput, TransformErrorSkipped, TransformErrorInternal} {
		if kind == "" {
			t.Fatalf("empty transform error kind")
		}
	}
}

func TestPipelineTransformErrorWarnsAndContinues(t *testing.T) {
	p := &Pipeline{reformats: []ReformatTransform{jsonMinifierTransform{}}, offloads: nil}
	ctx := CompressionContext{ContentKind: KindJSON, Tokenizer: FallbackTokenizer{}, Aggressiveness: 0.5}
	policy := DefaultCompressionPolicy(0.5)
	res := p.Run(`{"a":1,}`, ctx, policy)
	if res.Output != `{"a":1,}` {
		t.Fatalf("invalid transform should leave content unchanged, got %q", res.Output)
	}
	found := false
	for _, w := range res.Warnings {
		if w.Code == "transform_error_invalid_input" && w.Component == "json_minifier" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected transform warning, got %#v", res.Warnings)
	}
}

func TestPipelineRunTokenizerCountErrorWarnsAndFallsBack(t *testing.T) {
	p := &Pipeline{reformats: nil, offloads: nil}
	ctx := CompressionContext{ContentKind: KindText, Tokenizer: errorTokenizer{}, Aggressiveness: 0.5}
	res := p.Run("hello world", ctx, DefaultCompressionPolicy(0.5))
	if res.TokensBefore != 2 || res.TokensAfter != 2 {
		t.Fatalf("expected fallback token counts, got before=%d after=%d", res.TokensBefore, res.TokensAfter)
	}
	found := false
	for _, w := range res.Warnings {
		if w.Code == "tokenizer_count_error" && strings.Contains(w.Message, "used fallback tokenizer") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected tokenizer warning, got %#v", res.Warnings)
	}
}

func TestHTMLCleanMalformedBlocksDoNotTruncateTail(t *testing.T) {
	input := "<div>before</div><script>broken\n<div>after</div>"
	out := removeHTMLBlock(input, "script")
	if !strings.Contains(out, "<div>after</div>") {
		t.Fatalf("unterminated script truncated tail: %q", out)
	}
	commentInput := "<div>before</div><!-- broken\n<div>after</div>"
	out = removeHTMLComments(commentInput)
	if !strings.Contains(out, "<div>after</div>") {
		t.Fatalf("unterminated comment truncated tail: %q", out)
	}
}

func TestHTMLCleanPreservesPreCodeTextareaAndAttributeWhitespace(t *testing.T) {
	input := `<div title="a  b"><pre>x  y	z</pre><code>a  b</code><textarea>c  d</textarea></div>`
	out, err := htmlCleanTransform{}.Apply(input, CompressionContext{ContentKind: KindHTML})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`title="a  b"`, "x  y\tz", "a  b", "c  d"} {
		if !strings.Contains(out.Output, want) {
			t.Fatalf("html whitespace semantics lost; missing %q in %q", want, out.Output)
		}
	}
}
