package headroom

import "github.com/superops-team/headroom-go/internal/cachealigner"

type CacheAlignerConfig = cachealigner.CacheAlignerConfig
type CacheAligner = cachealigner.CacheAligner

func NewCacheAligner(cfg CacheAlignerConfig) *CacheAligner {
	return cachealigner.NewCacheAligner(cfg)
}
