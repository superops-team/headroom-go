package headroom

import "github.com/superops-team/headroom-go/internal/router"

// ContentRouter auto-detects the content type of a string.
//
// Uses O(n) single-pass detection with O(1) extra memory.
// Detection order: JSON → Code → Diff → Log → Search → Tabular → Spreadsheet → HTML → Text (default).
//
// Example:
//
//	router := headroom.NewContentRouter()
//	kind := router.Detect(`{"key": "value"}`)
//	// kind == headroom.KindJSON
type ContentRouter = router.ContentRouter

// NewContentRouter creates a new ContentRouter.
func NewContentRouter() *ContentRouter {
	return router.NewContentRouter()
}
