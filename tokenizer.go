package headroom

import "github.com/superops-team/headroom-go/internal/tokenizer"

type TokenizerBackend = tokenizer.TokenizerBackend
type TokenizerConfig = tokenizer.TokenizerConfig
type Tokenizer = tokenizer.Tokenizer
type FallbackTokenizer = tokenizer.FallbackTokenizer

const (
	TokenizerFallback = tokenizer.TokenizerFallback
	TokenizerTiktoken = tokenizer.TokenizerTiktoken
	TokenizerHF       = tokenizer.TokenizerHF
)

var ErrTokenizerNotImplemented = tokenizer.ErrTokenizerNotImplemented

func NewTokenizer(cfg TokenizerConfig) (Tokenizer, []Warning, error) {
	return tokenizer.NewTokenizer(cfg)
}

func ResolveTokenizer(cfg TokenizerConfig) (Tokenizer, []Warning, error) {
	return tokenizer.ResolveTokenizer(cfg)
}
