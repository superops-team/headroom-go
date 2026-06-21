package headroom

import "github.com/superops-team/headroom-go/internal/types"

// ContentKind identifies the type of content for specialized compression.
//
// The ContentRouter auto-detects content types. Each type has a dedicated
// compressor with strategies optimized for that format.
//
// Values are explicitly assigned (not iota) to prevent accidental renumbering
// when new types are added.
type ContentKind = types.ContentKind

// Content type constants.
//
// Detection rules:
//   - KindJSON (1): starts with { or [ and passes encoding/json.Valid()
//   - KindCode (2): 3+ lines containing code keywords (func, class, def, etc.)
//   - KindText (0): default fallback for unrecognized content
//   - KindDiff (3): contains @@ hunk headers
//   - KindLog (4): timestamp + log level pattern
//   - KindSearch (5): filename:line: format
//   - KindTabular (6): TSV/CSV structured data
//   - KindSpreadsheet (7): multi-column cell data
//   - KindHTML (8): HTML tag structure
//   - KindUnknown (9): explicitly unclassified content
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
