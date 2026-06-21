package headroom

import "github.com/superops-team/headroom-go/internal/cachealigner"

// CacheAlignerConfig configures prefix alignment for KV cache optimization.
//
// Fields:
//   - Enabled: if true, prefixes output with [headroom/{Version}]
//   - Version: the version string used in the prefix (default "v0.1")
type CacheAlignerConfig = cachealigner.CacheAlignerConfig

// CacheAligner adds a stable version prefix to compressed output.
//
// When enabled, all output is prefixed with [headroom/{Version}]\n.
// This makes identical configurations produce identical prefixes,
// boosting provider-side KV cache hit rates and reducing token costs.
//
// Example:
//
//	aligner := headroom.NewCacheAligner(headroom.CacheAlignerConfig{
//	    Enabled: true,
//	    Version: headroom.PrefixVersion,
//	})
//	output := aligner.Align("compressed content")
//	// → "[headroom/v0.5]\ncompressed content"
type CacheAligner = cachealigner.CacheAligner

// NewCacheAligner creates a CacheAligner with the given configuration.
// If Version is empty, defaults to "v0.1".
func NewCacheAligner(cfg CacheAlignerConfig) *CacheAligner {
	return cachealigner.NewCacheAligner(cfg)
}
