package headroom

import (
	"errors"
	"strings"
	"testing"

	"github.com/superops-team/headroom-go/internal/compressors"
	eng "github.com/superops-team/headroom-go/internal/engine"
)

func TestSpecD_EstimateTokensDirect(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{name: "empty", in: "", want: 0},
		{name: "ascii", in: "abcd efgh", want: len("abcd efgh") / 4},
		{name: "chinese", in: "你好世界", want: 4},
		{name: "emoji", in: "🙂🙃😉😊", want: 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := estimateTokens(tc.in); got != tc.want {
				t.Fatalf("estimateTokens(%q)=%d want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestSpecD_NoopObserverInterfaceAndNoPanic(t *testing.T) {
	var _ Observer = NoopObserver{}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NoopObserver.ObserveCompressionStep panicked: %v", r)
		}
	}()
	NoopObserver{}.ObserveCompressionStep(CompressionStep{Name: "spec_d", Kind: KindText.String(), TokensBefore: 4, TokensAfter: 2})
}

func TestSpecD_PipelineTokenBudgetZeroHasNoPipelineSummaryStep(t *testing.T) {
	opts := DefaultOptions()
	opts.EnablePipeline = true
	opts.TokenBudget = 0
	opts.Reversible = false

	res, err := Compress([]Message{{Role: "user", Content: strings.Repeat("same line\n", 40)}}, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Steps) == 0 {
		t.Fatal("expected compression steps")
	}
	for _, step := range res.Steps {
		if step.Name == "pipeline" {
			t.Fatalf("TokenBudget=0 should not emit pipeline summary step: %#v", res.Steps)
		}
	}
}

func TestSpecD_PipelineEmptyQueryDiffBehavior(t *testing.T) {
	src := buildSpecDDiff(30)
	p := NewDefaultPipeline()
	ctx := CompressionContext{Query: "", ContentKind: KindDiff, OriginalTokens: 642, Tokenizer: FallbackTokenizer{}, TokenBudget: 200, Aggressiveness: 0.5, Reversible: true}
	res := p.Run(src, ctx, DefaultCompressionPolicy(0.5))
	if !containsStepName(res.Steps, "diff_offload") {
		t.Fatalf("expected diff_offload step for empty Query, got %#v", res.Steps)
	}
	if !strings.Contains(res.Output, "[... omitted diff lines ...]") {
		t.Fatalf("expected omitted diff marker with empty Query, got %q", res.Output)
	}
}

func TestSpecD_NewTokenizerUnavailableBackendsExposeStub(t *testing.T) {
	cases := []struct {
		backend TokenizerBackend
		name    string
	}{
		{backend: TokenizerTiktoken, name: "tiktoken-stub"},
		{backend: TokenizerHF, name: "huggingface-stub"},
	}
	for _, tc := range cases {
		t.Run(string(tc.backend), func(t *testing.T) {
			tok, warnings, err := NewTokenizer(TokenizerConfig{Backend: tc.backend, AllowFallback: false})
			if !errors.Is(err, ErrTokenizerNotImplemented) {
				t.Fatalf("err=%v want ErrTokenizerNotImplemented", err)
			}
			if len(warnings) != 0 {
				t.Fatalf("unexpected warnings: %#v", warnings)
			}
			if tok == nil || tok.Name() != tc.name {
				t.Fatalf("tokenizer name=%v want %q", specDTokenizerName(tok), tc.name)
			}
			if _, err := tok.Count("hello"); !errors.Is(err, ErrTokenizerNotImplemented) {
				t.Fatalf("Count err=%v want ErrTokenizerNotImplemented", err)
			}
			if _, err := tok.CountBatch([]string{"hello", "world"}); !errors.Is(err, ErrTokenizerNotImplemented) {
				t.Fatalf("CountBatch err=%v want ErrTokenizerNotImplemented", err)
			}
		})
	}

	if tok, warnings, err := NewTokenizer(TokenizerConfig{Backend: TokenizerBackend("unavailable"), AllowFallback: false}); err == nil || tok != nil || len(warnings) != 0 {
		t.Fatalf("unknown unavailable backend got tok=%v warnings=%#v err=%v", tok, warnings, err)
	}
}

func TestSpecD_LineIndentDirect(t *testing.T) {
	cases := []struct {
		line string
		want int
	}{
		{line: "no indent", want: 0},
		{line: "  two spaces", want: 2},
		{line: "\ttab", want: 4},
	}
	for _, tc := range cases {
		if got := lineIndent(tc.line); got != tc.want {
			t.Fatalf("lineIndent(%q)=%d want %d", tc.line, got, tc.want)
		}
	}
}

func TestSpecD_DiffCompressionEndToEnd(t *testing.T) {
	opts := specDPipelineOptions()
	opts.Reversible = true
	cases := []struct {
		name        string
		content     string
		wantStep    string
		wantContain string
	}{
		{name: "normal unified diff", content: buildSpecDDiff(25), wantStep: "diff_offload", wantContain: "[... omitted diff lines ...]"},
		{name: "empty diff", content: "", wantStep: "pipeline_skip", wantContain: ""},
		{name: "large diff", content: buildSpecDDiff(120), wantStep: "diff_offload", wantContain: "[... omitted diff lines ...]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Compress([]Message{{Role: "user", Content: tc.content}}, opts)
			if err != nil {
				t.Fatal(err)
			}
			if len(res.Messages) != 1 {
				t.Fatalf("messages len=%d want 1", len(res.Messages))
			}
			if !containsStepName(res.Steps, tc.wantStep) {
				t.Fatalf("steps %#v do not contain %q", res.Steps, tc.wantStep)
			}
			if tc.wantContain != "" && !strings.Contains(res.Messages[0].Content, tc.wantContain) {
				t.Fatalf("compressed diff missing %q: %q", tc.wantContain, res.Messages[0].Content)
			}
		})
	}
}

func TestSpecD_HTMLCompressionEndToEnd(t *testing.T) {
	cases := []struct {
		name    string
		content string
		absent  []string
	}{
		{name: "normal HTML document", content: "<html><head><style>body{color:red}</style><script>alert(1)</script></head><body><main>keep</main></body></html>", absent: []string{"<style", "<script", "alert(1)"}},
		{name: "HTML with comments", content: "<html><body><!-- remove me --><div>keep</div></body></html>", absent: []string{"remove me", "<!--"}},
		{name: "empty HTML", content: "", absent: nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := compressors.NewHTMLCleanTransform().Apply(tc.content, CompressionContext{ContentKind: KindHTML, Aggressiveness: 0.7})
			if err != nil {
				t.Fatal(err)
			}
			if tc.content != "" && !containsStepName(out.Steps, "html_clean") {
				t.Fatalf("expected html_clean step, got %#v", out.Steps)
			}
			for _, absent := range tc.absent {
				if strings.Contains(out.Output, absent) {
					t.Fatalf("compressed HTML still contains %q: %q", absent, out.Output)
				}
			}
		})
	}
}

func TestSpecD_LogCompressionEndToEnd(t *testing.T) {
	opts := specDPipelineOptions()
	opts.Reversible = true
	cases := []struct {
		name        string
		content     string
		wantContain string
	}{
		{name: "mixed levels", content: strings.Repeat("[INFO] ok\n[INFO] ready\n[INFO] healthy\n[WARN] slow\n", 20) + "[ERROR] failed\n[FATAL] down\n" + strings.Repeat("[DEBUG] noisy\n[TRACE] detail\n", 20), wantContain: "more lines"},
		{name: "duplicate folding", content: strings.Repeat("[INFO] heartbeat\n", 12) + "[ERROR] failed once\n", wantContain: "[x12]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Compress([]Message{{Role: "user", Content: tc.content}}, opts)
			if err != nil {
				t.Fatal(err)
			}
			if !containsStepName(res.Steps, "log_template") && !containsStepName(res.Steps, "log_offload") {
				t.Fatalf("expected log step, got %#v", res.Steps)
			}
			if !strings.Contains(res.Messages[0].Content, tc.wantContain) {
				t.Fatalf("log output missing %q: %q", tc.wantContain, res.Messages[0].Content)
			}
		})
	}
}

func TestSpecD_SearchCompressionEndToEnd(t *testing.T) {
	opts := specDPipelineOptions()
	opts.Reversible = true
	content := strings.Repeat("pkg/a.go:10:func alpha()\n", 12) + strings.Repeat("pkg/b.go-22-return beta\n", 12)
	res, err := Compress([]Message{{Role: "user", Content: content}}, opts)
	if err != nil {
		t.Fatal(err)
	}
	if !containsStepName(res.Steps, "search_offload") {
		t.Fatalf("expected search_offload step, got %#v", res.Steps)
	}
	for _, want := range []string{"pkg/a.go:", "pkg/b.go:"} {
		if !strings.Contains(res.Messages[0].Content, want) {
			t.Fatalf("search output missing grouped file %q: %q", want, res.Messages[0].Content)
		}
	}
}

func TestSpecD_TabularSpreadsheetCompressionEndToEnd(t *testing.T) {
	opts := specDPipelineOptions()
	cases := []struct {
		name string
		in   string
	}{
		{name: "tsv", in: "name\tvalue\tstatus\napi\t1\tok\nworker\t2\twarn\n"},
		{name: "csv", in: "name,value,status\napi,1,ok\nworker,2,warn\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Compress([]Message{{Role: "user", Content: tc.in}}, opts)
			if err != nil {
				t.Fatal(err)
			}
			if !containsStepKind(res.Steps, KindTabular.String()) {
				t.Fatalf("expected tabular step, got %#v", res.Steps)
			}
			if res.Messages[0].Content != tc.in {
				t.Fatalf("tabular fallback should preserve current content, got %q want %q", res.Messages[0].Content, tc.in)
			}
		})
	}
}

func TestSpecD_CompressStringBoundaries(t *testing.T) {
	opts := DefaultOptions()
	opts.Reversible = false
	cases := []struct {
		name string
		in   string
	}{
		{name: "empty", in: ""},
		{name: "whitespace", in: " \n\t  "},
		{name: "very long single line", in: strings.Repeat("the quick brown fox jumps over the lazy dog ", 1000)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := CompressString(tc.in, opts)
			if err != nil {
				t.Fatal(err)
			}
			if tc.name != "very long single line" && out != tc.in {
				t.Fatalf("boundary content changed: got %q want %q", out, tc.in)
			}
			if tc.name == "very long single line" && (out == "" || len(out) > len(tc.in)) {
				t.Fatalf("long single line output len=%d input len=%d", len(out), len(tc.in))
			}
		})
	}
}

func TestSpecD_TransformNamesAndUnwrap(t *testing.T) {
	if compressors.NewDiffOffloadTransform().Name() != "diff_offload" {
		t.Fatal("unexpected diff transform name")
	}
	if compressors.NewLogOffloadTransform().Name() != "log_offload" {
		t.Fatal("unexpected log transform name")
	}
	if compressors.NewSearchOffloadTransform().Name() != "search_offload" {
		t.Fatal("unexpected search transform name")
	}
	if compressors.NewHTMLCleanTransform().Name() != "html_clean" {
		t.Fatal("unexpected html transform name")
	}
	cause := errors.New("cause")
	if !errors.Is(NewTransformError(TransformErrorInternal, "spec_d", "wrapped", cause), cause) {
		t.Fatal("TransformError should unwrap cause")
	}
	logOut, err := compressors.NewLogOffloadTransform().Apply(strings.Repeat("[INFO] noisy\n", 30)+"[ERROR] keep\n"+strings.Repeat("[DEBUG] noisy\n", 30), CompressionContext{ContentKind: KindLog})
	if err != nil || !strings.Contains(logOut.Output, "[... omitted low-priority log lines ...]") {
		t.Fatalf("log offload output=%q err=%v", logOut.Output, err)
	}
	jsonTransform := eng.NewJSONOffloadTransform()
	if jsonTransform.Name() != "json_offload" || jsonTransform.Confidence() <= 0 || jsonTransform.EstimateBloat(strings.Repeat("x", 201), CompressionContext{}) == 0 {
		t.Fatal("json offload metadata not covered")
	}
	jsonOut, err := jsonTransform.Apply(`[{"id":1,"status":"ok","drop":null},{"id":2,"status":"error","drop":null}]`, CompressionContext{ContentKind: KindJSON, Aggressiveness: 0.7, CCR: NewCCR(CCRConfig{})})
	if err != nil {
		t.Fatal(err)
	}
	if !containsStepName(jsonOut.Steps, "json_offload") || jsonOut.Output == "" || jsonOut.CacheKey == "" {
		t.Fatalf("unexpected json offload output: %#v", jsonOut)
	}
}

func specDPipelineOptions() Options {
	opts := DefaultOptions()
	opts.EnablePipeline = true
	opts.Reversible = false
	opts.Aggressiveness = 0.7
	return opts
}

func buildSpecDDiff(changes int) string {
	var b strings.Builder
	b.WriteString("diff --git a/a.go b/a.go\n")
	b.WriteString("--- a/a.go\n")
	b.WriteString("+++ b/a.go\n")
	b.WriteString("@@ -1,3 +1,3 @@\n")
	for i := 0; i < changes; i++ {
		b.WriteString("-old line with enough text to compress and omit\n")
		b.WriteString("+new line with enough text to compress and omit\n")
	}
	return b.String()
}

func containsStepName(steps []CompressionStep, name string) bool {
	for _, step := range steps {
		if step.Name == name {
			return true
		}
	}
	return false
}

func containsStepKind(steps []CompressionStep, kind string) bool {
	for _, step := range steps {
		if step.Kind == kind {
			return true
		}
	}
	return false
}

func specDTokenizerName(tok Tokenizer) string {
	if tok == nil {
		return "<nil>"
	}
	return tok.Name()
}
