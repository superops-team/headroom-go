package headroom

import "github.com/superops-team/headroom-go/internal/tokenizer"

// TokenizerBackend identifies the tokenizer implementation.
//
// Values:
//   - TokenizerFallback: ~4 chars/token heuristic (zero deps, default)
//   - TokenizerTiktoken: OpenAI tiktoken (precise, requires tiktoken)
//   - TokenizerHF: HuggingFace tokenizer (precise, requires HF)
type TokenizerBackend = tokenizer.TokenizerBackend

// TokenizerConfig configures the tokenizer.
//
// Fields:
//   - Backend: which tokenizer to use (fallback/tiktoken/huggingface)
//   - AllowFallback: if true, falls back to heuristic when backend is unavailable
type TokenizerConfig = tokenizer.TokenizerConfig

// Tokenizer counts tokens in text.
//
// Implementations:
//   - FallbackTokenizer: ~4 chars/token heuristic
//   - tiktoken backend: precise OpenAI token counting
//   - HuggingFace backend: precise HF tokenizer counting
type Tokenizer = tokenizer.Tokenizer

// FallbackTokenizer uses a simple ~4 chars/token heuristic.
// Always available, zero dependencies.
type FallbackTokenizer = tokenizer.FallbackTokenizer

// Tokenizer backend constants.
const (
	TokenizerFallback = tokenizer.TokenizerFallback
	TokenizerTiktoken = tokenizer.TokenizerTiktoken
	TokenizerHF       = tokenizer.TokenizerHF
)

// ErrTokenizerNotImplemented is returned when the requested tokenizer
// backend is unavailable and AllowFallback is false.
var ErrTokenizerNotImplemented = tokenizer.ErrTokenizerNotImplemented

// NewTokenizer creates a Tokenizer from the given configuration.
// Returns ErrTokenizerNotImplemented if the backend is unavailable
// and AllowFallback is false.
func NewTokenizer(cfg TokenizerConfig) (Tokenizer, []Warning, error) {
	return tokenizer.NewTokenizer(cfg)
}

// ResolveTokenizer resolves a tokenizer, returning warnings for fallbacks.
func ResolveTokenizer(cfg TokenizerConfig) (Tokenizer, []Warning, error) {
	return tokenizer.ResolveTokenizer(cfg)
}
