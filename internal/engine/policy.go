package engine

import (
	"github.com/superops-team/headroom-go/internal/tokenizer"
	"github.com/superops-team/headroom-go/internal/types"
)

type ContentKind = types.ContentKind
type Message = types.Message
type Options = types.Options
type Result = types.Result
type Warning = types.Warning
type CompressionStep = types.CompressionStep
type Observer = types.Observer
type Tokenizer = types.Tokenizer
type FallbackTokenizer = tokenizer.FallbackTokenizer
type TokenizerConfig = types.TokenizerConfig
type TokenizerBackend = types.TokenizerBackend
type CCRStore = types.CCRStore
type PolicyMode = types.PolicyMode
type TransformKind = types.TransformKind
type CompressionContext = types.CompressionContext
type CompressionPolicy = types.CompressionPolicy
type PolicyDecision = types.PolicyDecision
type TransformErrorKind = types.TransformErrorKind
type TransformError = types.TransformError
type ReformatOutput = types.ReformatOutput
type OffloadOutput = types.OffloadOutput
type ReformatTransform = types.ReformatTransform
type OffloadTransform = types.OffloadTransform
type PipelineResult = types.PipelineResult
type ProtectedContent = types.ProtectedContent

const (
	KindText        = types.KindText
	KindJSON        = types.KindJSON
	KindCode        = types.KindCode
	KindDiff        = types.KindDiff
	KindLog         = types.KindLog
	KindSearch      = types.KindSearch
	KindTabular     = types.KindTabular
	KindSpreadsheet = types.KindSpreadsheet
	KindHTML        = types.KindHTML
	KindUnknown     = types.KindUnknown

	TokenizerFallback = types.TokenizerFallback
	TokenizerTiktoken = types.TokenizerTiktoken
	TokenizerHF       = types.TokenizerHF

	PolicyConservative = types.PolicyConservative
	PolicyStandard     = types.PolicyStandard
	PolicyAggressive   = types.PolicyAggressive

	TransformReformat = types.TransformReformat
	TransformOffload  = types.TransformOffload

	TransformErrorInvalidInput = types.TransformErrorInvalidInput
	TransformErrorSkipped      = types.TransformErrorSkipped
	TransformErrorInternal     = types.TransformErrorInternal
)

func DefaultCompressionPolicy(aggressiveness float64) CompressionPolicy {
	return types.DefaultCompressionPolicy(aggressiveness)
}

func NewTransformError(kind TransformErrorKind, transform, message string, cause error) TransformError {
	return types.NewTransformError(kind, transform, message, cause)
}

func containsTransformKind(kinds []TransformKind, want TransformKind) bool {
	return types.ContainsTransformKind(kinds, want)
}
