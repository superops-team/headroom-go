package engine

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/superops-team/headroom-go/internal/cachealigner"
	"github.com/superops-team/headroom-go/internal/ccr"
	"github.com/superops-team/headroom-go/internal/compressors"
	"github.com/superops-team/headroom-go/internal/router"
	"github.com/superops-team/headroom-go/internal/tokenizer"
	"github.com/superops-team/headroom-go/internal/types"
)

var (
	packageCCROnce sync.Once
	packageCCR     *ccr.CCR
)

func getPackageCCR() *ccr.CCR {
	packageCCROnce.Do(func() {
		packageCCR = ccr.NewCCR(ccr.CCRConfig{TTL: 24 * time.Hour})
	})
	return packageCCR
}

func GetPackageCCR() *ccr.CCR {
	return getPackageCCR()
}

func DefaultOptions() Options {
	return Options{
		Aggressiveness: 0.5,
		Reversible:     true,
		AlignPrefix:    false,
		TokenLimit:     0,
	}
}

func Compress(messages []Message, opts Options) (*Result, error) {
	engine, warnings := NewCompressionEngine(opts)
	result, err := engine.Compress(messages, opts)
	if result != nil && len(warnings) > 0 {
		result.Warnings = append(warnings, result.Warnings...)
	}
	return result, err
}

type CompressionEngine struct {
	tokenizer    Tokenizer
	tokenizerErr error
	detector     *router.ContentRouter
	policy       CompressionPolicy
	ccr          CCRStore
	observer     Observer
}

func NewCompressionEngine(opts Options) (*CompressionEngine, []Warning) {
	tok, warnings, err := tokenizer.ResolveTokenizer(opts.TokenizerConfig)
	if err != nil {
		warnings = append(warnings, Warning{Code: "tokenizer_error", Component: "tokenizer", Message: err.Error()})
		if opts.TokenizerConfig.AllowFallback {
			tok = tokenizer.FallbackTokenizer{}
		} else {
			tok = nil
		}
	}
	observer := opts.Observer
	if observer == nil {
		observer = types.NoopObserver{}
	}
	return &CompressionEngine{tokenizer: tok, tokenizerErr: err, detector: router.NewContentRouter(), policy: DefaultCompressionPolicy(opts.Aggressiveness), ccr: getPackageCCR(), observer: observer}, warnings
}

func (e *CompressionEngine) Compress(messages []Message, opts Options) (*Result, error) {
	if e.tokenizerErr != nil && !opts.TokenizerConfig.AllowFallback {
		return nil, e.tokenizerErr
	}
	if opts.EnablePipeline || opts.TokenBudget > 0 || opts.Query != "" {
		return e.compressWithPipeline(messages, opts)
	}
	return compressLegacy(messages, opts, e.tokenizer, nil, e.observer)
}

func (e *CompressionEngine) compressWithPipeline(messages []Message, opts Options) (*Result, error) {
	return runPipelineMessages(messages, opts, e)
}

func compressLegacy(messages []Message, opts Options, tok Tokenizer, initialWarnings []Warning, observer Observer) (*Result, error) {
	r := router.NewContentRouter()
	store := getPackageCCR()
	aligner := cachealigner.NewCacheAligner(cachealigner.CacheAlignerConfig{
		Enabled: opts.AlignPrefix,
		Version: PrefixVersion,
	})

	compressedMsgs := make([]Message, 0, len(messages))
	origTokens := 0
	compTokens := 0
	warnings := append([]Warning{}, initialWarnings...)
	steps := make([]CompressionStep, 0, len(messages))

	for _, m := range messages {
		msgTokens, err := countTokens(tok, m.Content)
		if err != nil {
			return nil, err
		}
		origTokens += msgTokens

		if skipped, step := legacySkipMessage(m, opts, msgTokens); skipped {
			compressedMsgs = append(compressedMsgs, m)
			compTokens += msgTokens
			steps = append(steps, step)
			continue
		}

		kind, out, err := routeAndCompressLegacy(r, compressors.DefaultCompressorRegistry(), m.Content, opts)
		if err != nil {
			return nil, err
		}

		out, step, err := postProcessLegacyCompression(m.Content, out, kind, opts, tok, msgTokens, aligner, store)
		if err != nil {
			return nil, err
		}
		steps = append(steps, step)

		compressedMsgs = append(compressedMsgs, Message{
			Role:    m.Role,
			Content: out,
			Name:    m.Name,
		})
		outTokens, err := countTokens(tok, out)
		if err != nil {
			return nil, err
		}
		compTokens += outTokens
	}

	return buildLegacyResult(compressedMsgs, origTokens, compTokens, warnings, steps, observer), nil
}

func legacySkipMessage(m Message, opts Options, msgTokens int) (bool, CompressionStep) {
	baseStep := CompressionStep{Kind: KindText.String(), TokensBefore: msgTokens, TokensAfter: msgTokens, Skipped: true}
	if m.Role == "assistant" {
		baseStep.Name = "skip_assistant"
		baseStep.Reason = "assistant role"
		return true, baseStep
	}
	if strings.TrimSpace(m.Content) == "" {
		baseStep.Name = "skip_empty"
		baseStep.Reason = "empty content"
		return true, baseStep
	}
	if opts.TokenLimit > 0 && msgTokens < opts.TokenLimit {
		baseStep.Name = "skip_token_limit"
		baseStep.Reason = "below token limit"
		return true, baseStep
	}
	return false, CompressionStep{}
}

func routeAndCompressLegacy(r *router.ContentRouter, registry *compressors.CompressorRegistry, content string, opts Options) (ContentKind, string, error) {
	kind := r.Detect(content)
	out, err := registry.Compress(kind, content, opts)
	if err != nil {
		return kind, "", fmt.Errorf("compress %s: %w", kind.String(), err)
	}
	return kind, out, nil
}

func postProcessLegacyCompression(original, compressed string, kind ContentKind, opts Options, tok Tokenizer, msgTokens int, aligner *cachealigner.CacheAligner, store *ccr.CCR) (string, CompressionStep, error) {
	out := applyAlignPrefix(compressed, opts, aligner)
	out = applyReversibleCCR(original, out, kind, opts, store)
	if fallbackContent, fallbackStep, fallback := applyFallbackIfLonger(original, out, kind, msgTokens); fallback {
		return fallbackContent, fallbackStep, nil
	}

	outTokens, err := countTokens(tok, out)
	if err != nil {
		return "", CompressionStep{}, err
	}
	return out, CompressionStep{Name: "legacy_compress", Kind: kind.String(), TokensBefore: msgTokens, TokensAfter: outTokens}, nil
}

func applyAlignPrefix(out string, opts Options, aligner *cachealigner.CacheAligner) string {
	if opts.AlignPrefix {
		out = aligner.Align(out)
	}
	return out
}

func applyReversibleCCR(original, out string, kind ContentKind, opts Options, store *ccr.CCR) string {
	if opts.Reversible {
		id := store.Store(original, out, kind)
		retrieveSuffix := "\n\n[headroom:retrieve id=" + id + "]"
		out += retrieveSuffix
	}
	return out
}

func applyFallbackIfLonger(original, out string, kind ContentKind, msgTokens int) (string, CompressionStep, bool) {
	if len(out) >= len(original) {
		return original, CompressionStep{Name: "legacy_compress", Kind: kind.String(), TokensBefore: msgTokens, TokensAfter: msgTokens, Skipped: true, Reason: "output not shorter"}, true
	}
	return out, CompressionStep{}, false
}

func buildLegacyResult(messages []Message, origTokens, compTokens int, warnings []Warning, steps []CompressionStep, observer Observer) *Result {
	savings := 0.0
	if origTokens > 0 {
		savings = float64(origTokens-compTokens) / float64(origTokens)
	}
	if observer != nil {
		for _, step := range steps {
			observer.ObserveCompressionStep(step)
		}
	}
	return &Result{Messages: messages, CompressedTokens: compTokens, OriginalTokens: origTokens, Savings: savings, Warnings: warnings, Steps: steps}
}

func countTokens(tok Tokenizer, content string) (int, error) {
	if tok == nil {
		tok = tokenizer.FallbackTokenizer{}
	}
	return tok.Count(content)
}
