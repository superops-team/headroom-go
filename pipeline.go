package headroom

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type TransformErrorKind string

const (
	TransformErrorInvalidInput TransformErrorKind = "invalid_input"
	TransformErrorSkipped      TransformErrorKind = "skipped"
	TransformErrorInternal     TransformErrorKind = "internal"
)

type TransformError struct {
	Kind      TransformErrorKind
	Transform string
	Message   string
	Cause     error
}

func (e TransformError) Error() string {
	if e.Transform == "" {
		if e.Cause != nil {
			return string(e.Kind) + ": " + e.Message + ": " + e.Cause.Error()
		}
		return string(e.Kind) + ": " + e.Message
	}
	if e.Cause != nil {
		return e.Transform + " " + string(e.Kind) + ": " + e.Message + ": " + e.Cause.Error()
	}
	return e.Transform + " " + string(e.Kind) + ": " + e.Message
}

func (e TransformError) Unwrap() error { return e.Cause }

func NewTransformError(kind TransformErrorKind, transform, message string, cause error) TransformError {
	return TransformError{Kind: kind, Transform: transform, Message: message, Cause: cause}
}

type ReformatOutput struct {
	Output     string
	BytesSaved int
	Warnings   []Warning
	Steps      []CompressionStep
}

type OffloadOutput struct {
	Output     string
	BytesSaved int
	CacheKey   string
	Warnings   []Warning
	Steps      []CompressionStep
}

type ReformatTransform interface {
	Name() string
	AppliesTo() []ContentKind
	Apply(content string, ctx CompressionContext) (ReformatOutput, error)
}

type OffloadTransform interface {
	Name() string
	AppliesTo() []ContentKind
	EstimateBloat(content string, ctx CompressionContext) float64
	Apply(content string, ctx CompressionContext) (OffloadOutput, error)
	Confidence() float64
}

type PipelineResult struct {
	Output       string
	BytesSaved   int
	TokensBefore int
	TokensAfter  int
	StepsApplied []string
	CacheKeys    []string
	Warnings     []Warning
	Steps        []CompressionStep
}

type Pipeline struct {
	reformats []ReformatTransform
	offloads  []OffloadTransform
}

func NewDefaultPipeline() *Pipeline {
	return &Pipeline{reformats: []ReformatTransform{legacyTextTransform{}, legacyCodeTransform{}, jsonMinifierTransform{}, logTemplateTransform{}, htmlCleanTransform{}}, offloads: []OffloadTransform{diffOffloadTransform{}, logOffloadTransform{}, searchOffloadTransform{}, jsonOffloadTransform{}}}
}

func (p *Pipeline) Run(content string, ctx CompressionContext, policy CompressionPolicy) PipelineResult {
	before, beforeWarning := countTokensForPipeline(ctx.Tokenizer, content, "before")
	current := content
	result := PipelineResult{Output: content, TokensBefore: before, TokensAfter: before}
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
	if containsTransformKind(decision.AllowedKinds, TransformOffload) {
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

func countTokensForPipeline(tokenizer Tokenizer, content, phase string) (int, *Warning) {
	count, err := countTokens(tokenizer, content)
	if err == nil {
		return count, nil
	}
	fallbackCount, fallbackErr := FallbackTokenizer{}.Count(content)
	message := err.Error()
	if fallbackErr != nil {
		message += "; fallback count failed: " + fallbackErr.Error()
		return 0, &Warning{Code: "tokenizer_count_error", Component: "pipeline", Message: message}
	}
	return fallbackCount, &Warning{Code: "tokenizer_count_error", Component: "pipeline", Message: "token count " + phase + " failed; used fallback tokenizer: " + message}
}

func runPipelineMessages(messages []Message, opts Options, e *CompressionEngine) (*Result, error) {
	p := NewDefaultPipeline()
	out := make([]Message, 0, len(messages))
	warnings := []Warning{}
	steps := []CompressionStep{}
	origTokens := 0
	compTokens := 0
	protector := NewTagProtector()
	for _, m := range messages {
		msgTokens, err := countTokens(e.tokenizer, m.Content)
		if err != nil {
			return nil, err
		}
		origTokens += msgTokens
		if m.Role == "assistant" || strings.TrimSpace(m.Content) == "" || (opts.TokenLimit > 0 && msgTokens < opts.TokenLimit) {
			out = append(out, m)
			compTokens += msgTokens
			steps = append(steps, CompressionStep{Name: "pipeline_skip", Kind: KindText.String(), TokensBefore: msgTokens, TokensAfter: msgTokens, Skipped: true, Reason: "message not eligible"})
			continue
		}
		kind := e.detector.Detect(m.Content)
		protected := protector.Protect(m.Content)
		ctx := CompressionContext{Query: opts.Query, ContentKind: kind, OriginalTokens: msgTokens, TokenBudget: opts.TokenBudget, Aggressiveness: opts.Aggressiveness, Reversible: opts.Reversible, AlignPrefix: opts.AlignPrefix, Tokenizer: e.tokenizer, CCR: nil, Observer: e.observer}
		policy := DefaultCompressionPolicy(opts.Aggressiveness)
		pr := p.Run(protected.Text, ctx, policy)
		restored, restoreWarnings := protector.Restore(ProtectedContent{Text: pr.Output, Placeholders: protected.Placeholders, Warnings: append(protected.Warnings, pr.Warnings...)})
		warnings = append(warnings, restoreWarnings...)
		steps = append(steps, pr.Steps...)
		outLen := len(restored)
		if opts.AlignPrefix {
			restored = NewCacheAligner(CacheAlignerConfig{Enabled: true, Version: PrefixVersion}).Align(restored)
			outLen = len(restored)
		}
		if opts.Reversible && restored != m.Content && e.ccr != nil {
			id := e.ccr.Store(m.Content, restored, kind)
			retrieveSuffix := "\n\n[headroom:retrieve id=" + id + "]"
			restored += retrieveSuffix
			outLen += len(retrieveSuffix)
		}
		if outLen >= len(m.Content) {
			restored = m.Content
		}
		out = append(out, Message{Role: m.Role, Content: restored, Name: m.Name})
		ct, err := countTokens(e.tokenizer, restored)
		if err != nil {
			return nil, err
		}
		compTokens += ct
	}
	savings := 0.0
	if origTokens > 0 {
		savings = float64(origTokens-compTokens) / float64(origTokens)
	}
	return &Result{Messages: out, OriginalTokens: origTokens, CompressedTokens: compTokens, Savings: savings, Warnings: warnings, Steps: steps}, nil
}

func appliesTo(kinds []ContentKind, want ContentKind) bool {
	for _, kind := range kinds {
		if kind == want {
			return true
		}
	}
	return false
}

type legacyTextTransform struct{}

func (legacyTextTransform) Name() string             { return "legacy_text" }
func (legacyTextTransform) AppliesTo() []ContentKind { return []ContentKind{KindText} }
func (legacyTextTransform) Apply(content string, ctx CompressionContext) (ReformatOutput, error) {
	out := CompressText(content, TextConfig{Aggressiveness: ctx.Aggressiveness})
	return ReformatOutput{Output: out, BytesSaved: len(content) - len(out), Steps: []CompressionStep{{Name: "legacy_text", Kind: ctx.ContentKind.String()}}}, nil
}

type legacyCodeTransform struct{}

func (legacyCodeTransform) Name() string             { return "legacy_code" }
func (legacyCodeTransform) AppliesTo() []ContentKind { return []ContentKind{KindCode} }
func (legacyCodeTransform) Apply(content string, ctx CompressionContext) (ReformatOutput, error) {
	out := CompressCode(content, CodeConfig{Aggressiveness: ctx.Aggressiveness})
	return ReformatOutput{Output: out, BytesSaved: len(content) - len(out), Steps: []CompressionStep{{Name: "legacy_code", Kind: ctx.ContentKind.String()}}}, nil
}

type jsonMinifierTransform struct{}

func (jsonMinifierTransform) Name() string             { return "json_minifier" }
func (jsonMinifierTransform) AppliesTo() []ContentKind { return []ContentKind{KindJSON} }
func (jsonMinifierTransform) Apply(content string, ctx CompressionContext) (ReformatOutput, error) {
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(content)); err != nil {
		return ReformatOutput{}, NewTransformError(TransformErrorInvalidInput, "json_minifier", "invalid JSON", err)
	}
	out := buf.String()
	return ReformatOutput{Output: out, BytesSaved: len(content) - len(out), Steps: []CompressionStep{{Name: "json_minifier", Kind: ctx.ContentKind.String()}}}, nil
}

type jsonOffloadTransform struct{}

func (jsonOffloadTransform) Name() string             { return "json_offload" }
func (jsonOffloadTransform) AppliesTo() []ContentKind { return []ContentKind{KindJSON} }
func (jsonOffloadTransform) EstimateBloat(content string, ctx CompressionContext) float64 {
	if len(content) > 200 {
		return 1
	}
	return 0
}
func (jsonOffloadTransform) Confidence() float64 { return 0.7 }
func (jsonOffloadTransform) Apply(content string, ctx CompressionContext) (OffloadOutput, error) {
	crushed, steps, err := SmartCrushJSONWithSteps(content, SmartCrushConfig{Aggressiveness: ctx.Aggressiveness})
	if err != nil {
		return OffloadOutput{}, NewTransformError(TransformErrorInternal, "json_offload", "smart crusher failed", err)
	}
	id := ""
	if ctx.CCR != nil {
		id = ctx.CCR.Store(content, crushed, ctx.ContentKind)
	}
	steps = append([]CompressionStep{{Name: "json_offload", Kind: ctx.ContentKind.String()}}, steps...)
	return OffloadOutput{Output: crushed, BytesSaved: len(content) - len(crushed), CacheKey: id, Steps: steps}, nil
}

func warningFromTransformError(component string, err error) Warning {
	if te, ok := err.(TransformError); ok {
		return Warning{Code: "transform_error_" + string(te.Kind), Component: component, Message: te.Error()}
	}
	return Warning{Code: "transform_error", Component: component, Message: fmt.Sprint(err)}
}
