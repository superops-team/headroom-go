package compressors

import "github.com/superops-team/headroom-go/internal/types"

type ContentKind = types.ContentKind
type Options = types.Options
type Warning = types.Warning
type CompressionStep = types.CompressionStep
type Observer = types.Observer
type CompressionContext = types.CompressionContext
type ReformatOutput = types.ReformatOutput
type OffloadOutput = types.OffloadOutput
type ReformatTransform = types.ReformatTransform
type OffloadTransform = types.OffloadTransform

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
)

type CompressionConfig struct {
	Aggressiveness float64
}
