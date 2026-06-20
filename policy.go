package headroom

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
	return PolicyDecision{ShouldCompress: true, Reason: "compressible", AllowedKinds: allowed, TargetTokens: target, MaxOutputTokens: ctx.OriginalTokens - p.MinPositiveSavingsTokens, RequireCCR: ctx.Reversible && containsTransformKind(allowed, TransformOffload)}, warnings
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

func containsTransformKind(kinds []TransformKind, want TransformKind) bool {
	for _, kind := range kinds {
		if kind == want {
			return true
		}
	}
	return false
}
