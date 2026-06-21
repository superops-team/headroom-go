package tokenizer

import (
	"errors"
	"unicode"

	"github.com/superops-team/headroom-go/internal/types"
)

type TokenizerBackend = types.TokenizerBackend

const (
	TokenizerFallback = types.TokenizerFallback
	TokenizerTiktoken = types.TokenizerTiktoken
	TokenizerHF       = types.TokenizerHF
)

type TokenizerConfig = types.TokenizerConfig

type Tokenizer = types.Tokenizer

var ErrTokenizerNotImplemented = errors.New("tokenizer backend not implemented")

var (
	newTiktokenTokenizer = func(cfg TokenizerConfig) (Tokenizer, error) {
		return notImplementedTokenizer{name: "tiktoken-stub"}, ErrTokenizerNotImplemented
	}
	newHuggingFaceTokenizer = func(cfg TokenizerConfig) (Tokenizer, error) {
		return notImplementedTokenizer{name: "huggingface-stub"}, ErrTokenizerNotImplemented
	}
)

type FallbackTokenizer struct{}

func (FallbackTokenizer) Name() string { return "fallback-rune" }

func (FallbackTokenizer) Count(text string) (int, error) {
	if text == "" {
		return 0, nil
	}
	count := 0
	inWord := false
	for _, r := range text {
		if unicode.IsSpace(r) {
			inWord = false
			continue
		}
		if r <= 127 && (unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') {
			if !inWord {
				count++
				inWord = true
			}
			continue
		}
		inWord = false
		if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			count++
			continue
		}
		count++
	}
	return count, nil
}

func (t FallbackTokenizer) CountBatch(texts []string) ([]int, error) {
	out := make([]int, len(texts))
	for i, text := range texts {
		count, err := t.Count(text)
		if err != nil {
			return nil, err
		}
		out[i] = count
	}
	return out, nil
}

type notImplementedTokenizer struct{ name string }

func (t notImplementedTokenizer) Name() string { return t.name }
func (t notImplementedTokenizer) Count(text string) (int, error) {
	return 0, ErrTokenizerNotImplemented
}
func (t notImplementedTokenizer) CountBatch(texts []string) ([]int, error) {
	return nil, ErrTokenizerNotImplemented
}

func NewTokenizer(cfg TokenizerConfig) (Tokenizer, []types.Warning, error) {
	backend := cfg.Backend
	if backend == "" || backend == TokenizerFallback {
		return FallbackTokenizer{}, nil, nil
	}
	var tok Tokenizer
	var err error
	switch backend {
	case TokenizerTiktoken:
		tok, err = newTiktokenTokenizer(cfg)
	case TokenizerHF:
		tok, err = newHuggingFaceTokenizer(cfg)
	default:
		return nil, nil, errors.New("unknown tokenizer backend")
	}
	if err != nil && cfg.AllowFallback {
		return FallbackTokenizer{}, []types.Warning{{Code: "tokenizer_fallback", Component: "tokenizer", Message: string(backend) + " tokenizer is not implemented; using fallback"}}, nil
	}
	return tok, nil, err
}

func ResolveTokenizer(cfg TokenizerConfig) (Tokenizer, []types.Warning, error) {
	return NewTokenizer(cfg)
}
