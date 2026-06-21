// Package headroom provides intelligent context compression for AI agents.
//
// Headroom Go compresses everything an AI agent reads — tool outputs, logs,
// RAG snippets, code diffs, search results, and conversation history — before
// sending to an LLM. It auto-detects 10 content types and applies specialized
// compression strategies, achieving up to 70% token savings while preserving
// semantic accuracy.
//
// # Quick Start
//
//	result, _ := headroom.Compress(messages, headroom.Options{
//	    Aggressiveness: 0.5,
//	    Reversible:     true,
//	})
//	fmt.Printf("Saved %.0f%% tokens\n", result.Savings*100)
//
// # Architecture
//
// Headroom Go offers two compression paths:
//
//   - Legacy Path (default): Simple, fast — router → compressor → aligner → CCR.
//     Best for straightforward compression without policy decisions.
//
//   - Pipeline Path (EnablePipeline=true): Policy-driven — analyzes content,
//     applies token budgets, scores with query relevance, and chains multiple
//     reformat/offload transforms. Best when you need fine-grained control.
//
// # Zero Dependencies
//
// Headroom Go uses only the Go standard library. No CGo, no third-party
// packages, no runtime dependencies. Single binary deployment.
package headroom

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/superops-team/headroom-go/internal/cachealigner"
	"github.com/superops-team/headroom-go/internal/compressors"
	eng "github.com/superops-team/headroom-go/internal/engine"
	"github.com/superops-team/headroom-go/internal/router"
	"github.com/superops-team/headroom-go/internal/tokenizer"
	"github.com/superops-team/headroom-go/internal/types"
)

// Message represents a chat message compatible with OpenAI Messages format.
//
// Fields:
//   - Role: "system", "user", "assistant", or "tool"
//   - Content: the message text
//   - Name: optional participant name
type Message = types.Message

// Options controls compression behavior.
//
// Fields:
//   - Aggressiveness: compression strength 0.0-1.0 (default 0.5).
//     0.0-0.3 conservative, 0.3-0.7 standard, 0.7-1.0 aggressive.
//   - Reversible: if true, original content is cached and a retrieval ID
//     is appended to the output (default true).
//   - AlignPrefix: if true, prefixes output with [headroom/version] for
//     better provider-side KV cache hit rates (default false).
//   - TokenLimit: skip compression for messages with fewer tokens than
//     this threshold. 0 means always compress (default 0).
//   - TokenizerConfig: configures the tokenizer backend.
//   - TokenBudget: target token count for Pipeline mode (0 = unlimited).
//   - Query: search/diff relevance scoring query for Pipeline mode.
//   - EnablePipeline: use the policy-driven Pipeline path instead of Legacy.
//   - Observer: receives compression step notifications.
type Options = types.Options

// Result is the output of Compress.
//
// Fields:
//   - Messages: the compressed message array.
//   - CompressedTokens: estimated token count after compression.
//   - OriginalTokens: estimated token count before compression.
//   - Savings: token savings ratio (OriginalTokens-CompressedTokens)/OriginalTokens.
//   - Warnings: non-fatal warnings encountered during compression.
//   - Steps: detailed per-message compression steps for observability.
type Result = types.Result

// CompressionEngine compresses message batches with resolved dependencies.
// Created via NewCompressionEngine(opts).
type CompressionEngine = eng.CompressionEngine

// CompressionContext carries per-compression metadata for Pipeline transforms.
type CompressionContext = types.CompressionContext

// TransformError represents an error from a Pipeline transform step.
type TransformError = types.TransformError

// ReformatTransform is a Pipeline transform that rewrites content in-place.
type ReformatTransform = types.ReformatTransform

type Pipeline struct {
	reformats []ReformatTransform
	offloads  []types.OffloadTransform
}

const (
	TransformErrorInternal = types.TransformErrorInternal
)

// DefaultOptions returns the recommended default compression options.
//
// Defaults:
//   - Aggressiveness: 0.5 (standard)
//   - Reversible: true
//   - AlignPrefix: false
//   - TokenLimit: 0 (always compress)
func DefaultOptions() Options {
	return Options{
		Aggressiveness: 0.5,
		Reversible:     true,
		AlignPrefix:    false,
		TokenLimit:     0,
	}
}

// NewCompressionEngine creates a new CompressionEngine with the given options.
// The engine resolves tokenizer, CCR store, and pipeline dependencies.
// Returns the engine and any non-fatal warnings from dependency resolution.
func NewCompressionEngine(opts Options) (*CompressionEngine, []Warning) {
	return eng.NewCompressionEngine(opts)
}

// DefaultCompressionPolicy returns a Pipeline policy based on aggressiveness.
//
// Mapping:
//   - 0.0-0.3 → PolicyConservative
//   - 0.3-0.7 → PolicyStandard
//   - 0.7-1.0 → PolicyAggressive
func DefaultCompressionPolicy(aggressiveness float64) types.CompressionPolicy {
	return types.DefaultCompressionPolicy(aggressiveness)
}

// NewTransformError creates a TransformError for Pipeline transform failures.
//
// Parameters:
//   - kind: error category (TransformErrorInvalidInput, TransformErrorSkipped, TransformErrorInternal)
//   - transform: name of the transform that failed
//   - message: human-readable error description
//   - cause: underlying error (can be nil)
func NewTransformError(kind types.TransformErrorKind, transform, message string, cause error) TransformError {
	return types.NewTransformError(kind, transform, message, cause)
}

// NewDefaultPipeline creates a Pipeline pre-configured with all built-in transforms.
//
// Reformats (in-place): legacy_text, legacy_code, json_minifier, log_template, html_clean.
// Offloads (cache-and-replace): diff, log, search, json.
func NewDefaultPipeline() *Pipeline {
	return &Pipeline{reformats: []ReformatTransform{legacyTextTransform{}, legacyCodeTransform{}, jsonMinifierTransform{}, compressors.NewLogTemplateTransform(), compressors.NewHTMLCleanTransform()}, offloads: []types.OffloadTransform{compressors.NewDiffOffloadTransform(), compressors.NewLogOffloadTransform(), compressors.NewSearchOffloadTransform(), jsonOffloadTransform{}}}
}

// Compress compresses a batch of chat messages.
//
// Assistant role messages are passed through unchanged. Tool role messages
// are treated as user messages and compressed. The compression path is
// chosen automatically: Legacy path by default, Pipeline path when
// opts.EnablePipeline is true.
//
// Returns ErrTokenizerNotImplemented if the configured tokenizer backend
// is unavailable and AllowFallback is false.
//
// Example:
//
//	messages := []headroom.Message{
//	    {Role: "user", Content: "What does this error mean?"},
//	    {Role: "tool", Content: hugeJSONResponse},
//	}
//	result, err := headroom.Compress(messages, headroom.DefaultOptions())
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Saved %.0f%% tokens\n", result.Savings*100)
func Compress(messages []Message, opts Options) (*Result, error) {
	engine, warnings := NewCompressionEngine(opts)
	result, err := engine.Compress(messages, opts)
	if result != nil && len(warnings) > 0 {
		result.Warnings = append(warnings, result.Warnings...)
	}
	return result, err
}

// CompressString compresses a single text string.
//
// Convenience wrapper around Compress. Wraps the content in a user-role
// message, compresses it, and returns the compressed text.
//
// Example:
//
//	compressed, err := headroom.CompressString(hugeLogContent, headroom.DefaultOptions())
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(compressed)
func CompressString(content string, opts Options) (string, error) {
	r, err := Compress([]Message{{Role: "user", Content: content}}, opts)
	if err != nil {
		return "", err
	}
	if len(r.Messages) == 0 {
		return "", nil
	}
	return r.Messages[0].Content, nil
}

func estimateTokens(s string) int {
	n, _ := tokenizer.FallbackTokenizer{}.Count(s)
	return n
}

func countTokens(tok Tokenizer, content string) (int, error) {
	if tok == nil {
		tok = tokenizer.FallbackTokenizer{}
	}
	return tok.Count(content)
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

		compressedMsgs = append(compressedMsgs, Message{Role: m.Role, Content: out, Name: m.Name})
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

func routeAndCompressLegacy(r *ContentRouter, registry *CompressorRegistry, content string, opts Options) (ContentKind, string, error) {
	kind := r.Detect(content)
	out, err := registry.Compress(kind, content, opts)
	if err != nil {
		return kind, "", fmt.Errorf("compress %s: %w", kind.String(), err)
	}
	return kind, out, nil
}

func postProcessLegacyCompression(original, compressed string, kind ContentKind, opts Options, tok Tokenizer, msgTokens int, aligner *CacheAligner, store *CCR) (string, CompressionStep, error) {
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

func applyAlignPrefix(out string, opts Options, aligner *CacheAligner) string {
	if opts.AlignPrefix {
		out = aligner.Align(out)
	}
	return out
}

func applyReversibleCCR(original, out string, kind ContentKind, opts Options, store *CCR) string {
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

func (p *Pipeline) Run(content string, ctx CompressionContext, policy types.CompressionPolicy) types.PipelineResult {
	before, beforeWarning := countTokensForPipeline(ctx.Tokenizer, content, "before")
	current := content
	result := types.PipelineResult{Output: content, TokensBefore: before, TokensAfter: before}
	if beforeWarning != nil {
		result.Warnings = append(result.Warnings, *beforeWarning)
	}
	decision, warnings := policy.Decide(ctx)
	result.Warnings = append(result.Warnings, warnings...)
	if !decision.ShouldCompress {
		result.Steps = append(result.Steps, CompressionStep{Name: "policy", Kind: ctx.ContentKind.String(), TokensBefore: before, TokensAfter: before, Skipped: true, Reason: decision.Reason})
		return result
	}
	for _, t := range p.reformats {
		if !appliesTo(t.AppliesTo(), ctx.ContentKind) {
			continue
		}
		out, err := t.Apply(current, ctx)
		if err != nil {
			result.Warnings = append(result.Warnings, warningFromTransformError(t.Name(), err))
			continue
		}
		if out.Output != "" && len(out.Output) < len(current) {
			current = out.Output
			result.StepsApplied = append(result.StepsApplied, t.Name())
		}
		result.Warnings = append(result.Warnings, out.Warnings...)
		result.Steps = append(result.Steps, out.Steps...)
	}
	if containsTransformKind(decision.AllowedKinds, types.TransformOffload) {
		for _, t := range p.offloads {
			if !appliesTo(t.AppliesTo(), ctx.ContentKind) || t.Confidence() < 0.5 || t.EstimateBloat(current, ctx) < policy.BloatThreshold {
				continue
			}
			out, err := t.Apply(current, ctx)
			if err != nil {
				result.Warnings = append(result.Warnings, warningFromTransformError(t.Name(), err))
				continue
			}
			if out.Output != "" && len(out.Output) < len(current) {
				current = out.Output
				result.StepsApplied = append(result.StepsApplied, t.Name())
				if out.CacheKey != "" {
					result.CacheKeys = append(result.CacheKeys, out.CacheKey)
				}
			}
			result.Warnings = append(result.Warnings, out.Warnings...)
			result.Steps = append(result.Steps, out.Steps...)
		}
	}
	after, afterWarning := countTokensForPipeline(ctx.Tokenizer, current, "after")
	if afterWarning != nil {
		result.Warnings = append(result.Warnings, *afterWarning)
	}
	if len(current) >= len(content) {
		current = content
		after = before
		result.Steps = append(result.Steps, CompressionStep{Name: "pipeline", Kind: ctx.ContentKind.String(), TokensBefore: before, TokensAfter: after, Skipped: true, Reason: "output not shorter"})
	}
	result.Output = current
	result.BytesSaved = len(content) - len(current)
	result.TokensAfter = after
	if ctx.Observer != nil {
		for _, step := range result.Steps {
			ctx.Observer.ObserveCompressionStep(step)
		}
	}
	return result
}

func countTokensForPipeline(tok Tokenizer, content, phase string) (int, *Warning) {
	count, err := countTokens(tok, content)
	if err == nil {
		return count, nil
	}
	fallbackCount, fallbackErr := tokenizer.FallbackTokenizer{}.Count(content)
	message := err.Error()
	if fallbackErr != nil {
		message += "; fallback count failed: " + fallbackErr.Error()
		return 0, &Warning{Code: "tokenizer_count_error", Component: "pipeline", Message: message}
	}
	return fallbackCount, &Warning{Code: "tokenizer_count_error", Component: "pipeline", Message: "token count " + phase + " failed; used fallback tokenizer: " + message}
}

func appliesTo(kinds []ContentKind, want ContentKind) bool {
	for _, kind := range kinds {
		if kind == want {
			return true
		}
	}
	return false
}

func containsTransformKind(kinds []types.TransformKind, want types.TransformKind) bool {
	return types.ContainsTransformKind(kinds, want)
}

type legacyTextTransform struct{}

func (legacyTextTransform) Name() string             { return "legacy_text" }
func (legacyTextTransform) AppliesTo() []ContentKind { return []ContentKind{KindText} }
func (legacyTextTransform) Apply(content string, ctx CompressionContext) (types.ReformatOutput, error) {
	out := compressors.CompressText(content, compressors.TextConfig{Aggressiveness: ctx.Aggressiveness})
	return types.ReformatOutput{Output: out, BytesSaved: len(content) - len(out), Steps: []CompressionStep{{Name: "legacy_text", Kind: ctx.ContentKind.String()}}}, nil
}

type legacyCodeTransform struct{}

func (legacyCodeTransform) Name() string             { return "legacy_code" }
func (legacyCodeTransform) AppliesTo() []ContentKind { return []ContentKind{KindCode} }
func (legacyCodeTransform) Apply(content string, ctx CompressionContext) (types.ReformatOutput, error) {
	out := compressors.CompressCode(content, compressors.CodeConfig{Aggressiveness: ctx.Aggressiveness})
	return types.ReformatOutput{Output: out, BytesSaved: len(content) - len(out), Steps: []CompressionStep{{Name: "legacy_code", Kind: ctx.ContentKind.String()}}}, nil
}

type jsonMinifierTransform struct{}

func (jsonMinifierTransform) Name() string             { return "json_minifier" }
func (jsonMinifierTransform) AppliesTo() []ContentKind { return []ContentKind{KindJSON} }
func (jsonMinifierTransform) Apply(content string, ctx CompressionContext) (types.ReformatOutput, error) {
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(content)); err != nil {
		return types.ReformatOutput{}, NewTransformError(types.TransformErrorInvalidInput, "json_minifier", "invalid JSON", err)
	}
	out := buf.String()
	return types.ReformatOutput{Output: out, BytesSaved: len(content) - len(out), Steps: []CompressionStep{{Name: "json_minifier", Kind: ctx.ContentKind.String()}}}, nil
}

type jsonOffloadTransform struct{}

func NewJSONOffloadTransform() types.OffloadTransform { return jsonOffloadTransform{} }

func (jsonOffloadTransform) Name() string             { return "json_offload" }
func (jsonOffloadTransform) AppliesTo() []ContentKind { return []ContentKind{KindJSON} }
func (jsonOffloadTransform) EstimateBloat(content string, ctx CompressionContext) float64 {
	if len(content) > 200 {
		return 1
	}
	return 0
}
func (jsonOffloadTransform) Confidence() float64 { return 0.7 }
func (jsonOffloadTransform) Apply(content string, ctx CompressionContext) (types.OffloadOutput, error) {
	crushed, steps, err := compressors.SmartCrushJSONWithSteps(content, compressors.SmartCrushConfig{Aggressiveness: ctx.Aggressiveness})
	if err != nil {
		return types.OffloadOutput{}, NewTransformError(TransformErrorInternal, "json_offload", "smart crusher failed", err)
	}
	id := ""
	if ctx.CCR != nil {
		id = ctx.CCR.Store(content, crushed, ctx.ContentKind)
	}
	steps = append([]CompressionStep{{Name: "json_offload", Kind: ctx.ContentKind.String()}}, steps...)
	return types.OffloadOutput{Output: crushed, BytesSaved: len(content) - len(crushed), CacheKey: id, Steps: steps}, nil
}

func warningFromTransformError(component string, err error) Warning {
	if te, ok := err.(TransformError); ok {
		return Warning{Code: "transform_error_" + string(te.Kind), Component: component, Message: te.Error()}
	}
	return Warning{Code: "transform_error", Component: component, Message: fmt.Sprint(err)}
}

type htmlCleanTransform struct{}

func (htmlCleanTransform) Name() string             { return "html_clean" }
func (htmlCleanTransform) AppliesTo() []ContentKind { return []ContentKind{KindHTML} }
func (htmlCleanTransform) Apply(content string, ctx CompressionContext) (types.ReformatOutput, error) {
	return compressors.NewHTMLCleanTransform().Apply(content, ctx)
}

func removeHTMLBlock(s, tag string) string {
	return compressors.RemoveHTMLBlock(s, tag)
}

func removeHTMLComments(s string) string {
	return compressors.RemoveHTMLComments(s)
}

type errorTokenizer struct{}

func (errorTokenizer) Name() string { return "error" }
func (errorTokenizer) Count(string) (int, error) {
	return 0, errors.New("tokenizer boom")
}
func (errorTokenizer) CountBatch([]string) ([]int, error) {
	return nil, errors.New("tokenizer boom")
}

type CompressionConfig = compressors.CompressionConfig
type CodeConfig = compressors.CodeConfig
type TextConfig = compressors.TextConfig
type SmartCrushConfig = compressors.SmartCrushConfig
type Compressor = compressors.Compressor
type CompressorFunc = compressors.CompressorFunc
type CompressorRegistry = compressors.CompressorRegistry

func SmartCrushJSON(content string, cfg SmartCrushConfig) (string, error) {
	return compressors.SmartCrushJSON(content, cfg)
}

func SmartCrushJSONWithSteps(content string, cfg SmartCrushConfig) (string, []CompressionStep, error) {
	return compressors.SmartCrushJSONWithSteps(content, cfg)
}

func CompressCode(content string, cfg CodeConfig) string {
	return compressors.CompressCode(content, cfg)
}

func CompressText(content string, cfg TextConfig) string {
	return compressors.CompressText(content, cfg)
}

func lineIndent(s string) int {
	n := 0
	for _, r := range s {
		if r == ' ' {
			n++
		} else if r == '	' {
			n += 4
		} else {
			break
		}
	}
	return n
}

func NewCompressorFunc(kind ContentKind, fn func(string, Options) (string, error)) CompressorFunc {
	return compressors.NewCompressorFunc(kind, fn)
}

func NewCompressorRegistry() *CompressorRegistry {
	return compressors.NewCompressorRegistry()
}

func DefaultCompressorRegistry() *CompressorRegistry {
	return compressors.DefaultCompressorRegistry()
}
