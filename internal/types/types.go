// Package types defines shared types used across internal packages.
package types

// Warning represents a non-fatal compression warning.
type Warning struct {
	Code      string
	Component string
	Message   string
}

// CompressionStep records a single step in the compression pipeline.
type CompressionStep struct {
	Name         string
	Kind         string
	TokensBefore int
	TokensAfter  int
	Skipped      bool
	Reason       string
}

// Observer receives compression step notifications.
type Observer interface {
	ObserveCompressionStep(step CompressionStep)
}

// NoopObserver is a no-op implementation of Observer.
type NoopObserver struct{}

func (NoopObserver) ObserveCompressionStep(step CompressionStep) {}

// TokenizerBackend identifies a tokenizer implementation.
type TokenizerBackend string

const (
	TokenizerFallback TokenizerBackend = "fallback"
	TokenizerTiktoken TokenizerBackend = "tiktoken"
	TokenizerHF       TokenizerBackend = "huggingface"
)

// TokenizerConfig configures tokenizer selection and fallback behavior.
type TokenizerConfig struct {
	Backend       TokenizerBackend
	Model         string
	TokenizerPath string
	AllowFallback bool
}

// Tokenizer counts model tokens for text inputs.
type Tokenizer interface {
	Name() string
	Count(text string) (int, error)
	CountBatch(texts []string) ([]int, error)
}

// Message represents a chat message compatible with OpenAI Messages format.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// Options controls compression behavior.
type Options struct {
	Aggressiveness float64
	Reversible     bool
	AlignPrefix    bool
	TokenLimit     int
	TokenizerConfig TokenizerConfig
	TokenBudget     int
	Query           string
	EnablePipeline  bool
	Observer        Observer
}

// Result is the output of compression.
type Result struct {
	Messages         []Message
	CompressedTokens int
	OriginalTokens   int
	Savings          float64
	Warnings         []Warning
	Steps            []CompressionStep
}

// ContentKind identifies the type of content for compression.
type ContentKind int

const (
	KindText        ContentKind = 0
	KindJSON        ContentKind = 1
	KindCode        ContentKind = 2
	KindDiff        ContentKind = 3
	KindLog         ContentKind = 4
	KindSearch      ContentKind = 5
	KindTabular     ContentKind = 6
	KindSpreadsheet ContentKind = 7
	KindHTML        ContentKind = 8
	KindUnknown     ContentKind = 9
)

func (k ContentKind) String() string {
	switch k {
	case KindText:
		return "Text"
	case KindJSON:
		return "JSON"
	case KindCode:
		return "Code"
	case KindDiff:
		return "Diff"
	case KindLog:
		return "Log"
	case KindSearch:
		return "Search"
	case KindTabular:
		return "Tabular"
	case KindSpreadsheet:
		return "Spreadsheet"
	case KindHTML:
		return "HTML"
	case KindUnknown:
		return "Unknown"
	default:
		return "Text"
	}
}

// Version constants for CCR ID generation.
const (
	LegacyCCRIDVersion = "v2"
	CCRIDVersion       = "v3"
)

// CCRStore stores original content for reversible compression retrieval.
type CCRStore interface {
	Store(original, compressed string, kind ContentKind) string
	Retrieve(id string) (string, bool)
}

type PolicyMode string

const (
	PolicyConservative PolicyMode = "conservative"
	PolicyStandard     PolicyMode = "standard"
	PolicyAggressive   PolicyMode = "aggressive"
)

type TransformKind string

const (
	TransformReformat TransformKind = "reformat"
	TransformOffload  TransformKind = "offload"
)

type CompressionContext struct {
	Query          string
	ContentKind    ContentKind
	OriginalTokens int
	TokenBudget    int
	Aggressiveness float64
	Reversible     bool
	AlignPrefix    bool
	Tokenizer      Tokenizer
	CCR            CCRStore
	Observer       Observer
}

type CompressionPolicy struct {
	Mode                     PolicyMode
	ReformatTargetRatio      float64
	BloatThreshold           float64
	OffloadFallbackRatio     float64
	MaxLossyRatio            float64
	MinPositiveSavingsTokens int
	PreserveErrors           bool
	PreserveTags             bool
}

type PolicyDecision struct {
	ShouldCompress  bool
	Reason          string
	AllowedKinds    []TransformKind
	TargetTokens    int
	MaxOutputTokens int
	RequireCCR      bool
}

func DefaultCompressionPolicy(aggressiveness float64) CompressionPolicy {
	a := clamp01(aggressiveness)
	mode := PolicyStandard
	if a < 0.3 {
		mode = PolicyConservative
	} else if a >= 0.7 {
		mode = PolicyAggressive
	}
	return CompressionPolicy{Mode: mode, ReformatTargetRatio: 0.85 - a*0.25, BloatThreshold: 0.25 + a*0.35, OffloadFallbackRatio: 0.65, MaxLossyRatio: 0.5 + a*0.3, MinPositiveSavingsTokens: 1, PreserveErrors: true, PreserveTags: true}
}

func (p CompressionPolicy) Decide(ctx CompressionContext) (PolicyDecision, []Warning) {
	warnings := []Warning{}
	if ctx.Aggressiveness < 0 || ctx.Aggressiveness > 1 {
		warnings = append(warnings, Warning{Code: "aggressiveness_clamped", Component: "policy", Message: "aggressiveness outside [0,1] was clamped"})
	}
	allowed := []TransformKind{TransformReformat}
	if ctx.Reversible {
		allowed = append(allowed, TransformOffload)
	}
	if ctx.TokenBudget > 0 && ctx.OriginalTokens <= ctx.TokenBudget {
		return PolicyDecision{ShouldCompress: true, Reason: "within budget; reformat only", AllowedKinds: []TransformKind{TransformReformat}, TargetTokens: ctx.OriginalTokens, MaxOutputTokens: ctx.OriginalTokens}, warnings
	}
	target := ctx.TokenBudget
	if target <= 0 {
		target = int(float64(ctx.OriginalTokens) * p.ReformatTargetRatio)
	}
	if target < 0 {
		target = 0
	}
	return PolicyDecision{ShouldCompress: true, Reason: "compressible", AllowedKinds: allowed, TargetTokens: target, MaxOutputTokens: ctx.OriginalTokens - p.MinPositiveSavingsTokens, RequireCCR: ctx.Reversible && ContainsTransformKind(allowed, TransformOffload)}, warnings
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func ContainsTransformKind(kinds []TransformKind, want TransformKind) bool {
	for _, kind := range kinds {
		if kind == want {
			return true
		}
	}
	return false
}

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

type ProtectedContent struct {
	Text         string
	Placeholders map[string]string
	Warnings     []Warning
}
