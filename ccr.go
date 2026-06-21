package headroom

import (
	"github.com/superops-team/headroom-go/internal/ccr"
	eng "github.com/superops-team/headroom-go/internal/engine"
)

// CCRConfig configures the reversible compression store.
//
// Fields:
//   - TTL: entry expiration duration (default 24h)
//   - MaxEntries: maximum entries before FIFO eviction (default 10000)
type CCRConfig = ccr.CCRConfig

// CCR (Compress-Cache-Retrieve) provides reversible compression.
//
// When Reversible mode is enabled, original content is stored in CCR
// and a retrieval ID (format: v3_{sha256[:12]}) is appended to the
// compressed output. Call Retrieve(id) to recover the original content.
//
// CCR is thread-safe (sync.RWMutex). A background goroutine runs GC
// every 30 minutes to clean expired entries.
//
// Example:
//
//	store := headroom.NewCCR(headroom.CCRConfig{TTL: 1 * time.Hour})
//	id := store.Store("original text", "compressed", headroom.KindText)
//	original, found := store.Retrieve(id)
//	count, bytes := store.Stats()
type CCR = ccr.CCR

// NewCCR creates a new CCR store with the given configuration.
// Starts a background GC goroutine that runs every 30 minutes.
func NewCCR(cfg CCRConfig) *CCR {
	return ccr.NewCCR(cfg)
}

func getPackageCCR() *CCR {
	return eng.GetPackageCCR()
}
