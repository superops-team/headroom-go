// Package headroom provides intelligent context compression for AI agents.
//
// This file aggregates public re-exports from internal packages. All types,
// constants, and constructors here are type aliases or thin wrappers that
// preserve full backward compatibility with external importers.
package headroom

import (
	"github.com/superops-team/headroom-go/internal/cachealigner"
	"github.com/superops-team/headroom-go/internal/ccr"
	eng "github.com/superops-team/headroom-go/internal/engine"
	"github.com/superops-team/headroom-go/internal/router"
	"github.com/superops-team/headroom-go/internal/tagprotector"
	"github.com/superops-team/headroom-go/internal/tokenizer"
	"github.com/superops-team/headroom-go/internal/types"
)

// ── Content Kind ────────────────────────────────────────────────────────────

// ContentKind identifies the type of content for specialized compression.
type ContentKind = types.ContentKind

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

// ── Observability ───────────────────────────────────────────────────────────

// Warning represents a non-fatal issue encountered during compression.
type Warning = types.Warning

// CompressionStep records a single step in the compression pipeline.
type CompressionStep = types.CompressionStep

// Observer receives compression step notifications.
type Observer = types.Observer

// NoopObserver is a no-op implementation of Observer.
type NoopObserver = types.NoopObserver

// ── Cache Aligner ───────────────────────────────────────────────────────────

// CacheAlignerConfig configures prefix alignment for KV cache optimization.
type CacheAlignerConfig = cachealigner.CacheAlignerConfig

// CacheAligner adds a stable version prefix to compressed output.
type CacheAligner = cachealigner.CacheAligner

// NewCacheAligner creates a CacheAligner with the given configuration.
func NewCacheAligner(cfg CacheAlignerConfig) *CacheAligner {
	return cachealigner.NewCacheAligner(cfg)
}

// ── CCR (Compress-Cache-Retrieve) ───────────────────────────────────────────

// CCRConfig configures the reversible compression store.
type CCRConfig = ccr.CCRConfig

// CCR provides reversible compression with retrieval.
type CCR = ccr.CCR

// CCRStore is the interface for reversible compression storage.
type CCRStore = types.CCRStore

// NewCCR creates a new CCR store with the given configuration.
func NewCCR(cfg CCRConfig) *CCR {
	return ccr.NewCCR(cfg)
}

// getPackageCCR returns the package-level singleton CCR store.
func getPackageCCR() *CCR {
	return eng.GetPackageCCR()
}

// ── Content Router ──────────────────────────────────────────────────────────

// ContentRouter auto-detects the content type of a string.
type ContentRouter = router.ContentRouter

// NewContentRouter creates a new ContentRouter.
func NewContentRouter() *ContentRouter {
	return router.NewContentRouter()
}

// ── Tag Protector ───────────────────────────────────────────────────────────

// ProtectedContent holds content with protected XML tags extracted.
type ProtectedContent = tagprotector.ProtectedContent

// TagProtector preserves XML tags during compression.
type TagProtector = tagprotector.TagProtector

// NewTagProtector creates a new TagProtector with default protected tags.
func NewTagProtector() TagProtector {
	return tagprotector.NewTagProtector()
}

// ── Tokenizer ───────────────────────────────────────────────────────────────

// TokenizerBackend identifies the tokenizer implementation.
type TokenizerBackend = tokenizer.TokenizerBackend

// TokenizerConfig configures the tokenizer.
type TokenizerConfig = tokenizer.TokenizerConfig

// Tokenizer counts tokens in text.
type Tokenizer = tokenizer.Tokenizer

// FallbackTokenizer uses a simple ~4 chars/token heuristic.
type FallbackTokenizer = tokenizer.FallbackTokenizer

const (
	TokenizerFallback = tokenizer.TokenizerFallback
	TokenizerTiktoken = tokenizer.TokenizerTiktoken
	TokenizerHF       = tokenizer.TokenizerHF
)

// ErrTokenizerNotImplemented is returned when the requested tokenizer
// backend is unavailable and AllowFallback is false.
var ErrTokenizerNotImplemented = tokenizer.ErrTokenizerNotImplemented

// NewTokenizer creates a Tokenizer from the given configuration.
func NewTokenizer(cfg TokenizerConfig) (Tokenizer, []Warning, error) {
	return tokenizer.NewTokenizer(cfg)
}

// ResolveTokenizer resolves a tokenizer, returning warnings for fallbacks.
func ResolveTokenizer(cfg TokenizerConfig) (Tokenizer, []Warning, error) {
	return tokenizer.ResolveTokenizer(cfg)
}
