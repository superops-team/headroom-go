//go:build tokenizer_hf
// +build tokenizer_hf

package tokenizer

func NewHuggingFaceTokenizerStub() Tokenizer {
	return notImplementedTokenizer{name: "huggingface-stub"}
}

func init() {
	newHuggingFaceTokenizer = func(cfg TokenizerConfig) (Tokenizer, error) {
		return NewHuggingFaceTokenizerStub(), ErrTokenizerNotImplemented
	}
}
