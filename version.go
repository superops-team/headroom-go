package headroom

// Version is the semantic version of headroom-go.
// Used by CLI (headroom version) and proxy (/healthz).
const (
	// Version is the full semantic version string.
	Version = "v0.5.1"

	// PrefixVersion is the cache alignment prefix version.
	// Increment when compression algorithm changes would alter output.
	// Used by CacheAligner to generate [headroom/v0.5] prefixes.
	PrefixVersion = "v0.5"

	// LegacyCCRIDVersion is the ID prefix for legacy (SHA1-based) CCR entries.
	LegacyCCRIDVersion = "v2"

	// CCRIDVersion is the ID prefix for current (SHA256-based) CCR entries.
	// Format: v3_{sha256[:12]}
	CCRIDVersion = "v3"
)
