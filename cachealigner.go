package headroom

import "strings"

type CacheAlignerConfig struct {
	Enabled bool
	Version string // 前缀版本，如 "v0.1"
}

type CacheAligner struct {
	cfg CacheAlignerConfig
}

func NewCacheAligner(cfg CacheAlignerConfig) *CacheAligner {
	if cfg.Version == "" {
		cfg.Version = "v0.1"
	}
	return &CacheAligner{cfg: cfg}
}

// Align 在内容前添加稳定的版本前缀。
// 目的：让 LLM provider 基于 token 前缀的 cache 有更高命中率。
// Enabled=false → 原样返回。
func (a *CacheAligner) Align(content string) string {
	if !a.cfg.Enabled {
		return content
	}
	var sb strings.Builder
	sb.Grow(len(content) + 32)
	sb.WriteString("[headroom/")
	sb.WriteString(a.cfg.Version)
	sb.WriteString("]\n")
	sb.WriteString(content)
	return sb.String()
}
