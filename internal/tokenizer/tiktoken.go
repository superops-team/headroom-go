//go:build tokenizer_tiktoken
// +build tokenizer_tiktoken

package tokenizer

func NewTiktokenTokenizerStub() Tokenizer {
	return notImplementedTokenizer{name: "tiktoken-stub"}
}

func init() {
	newTiktokenTokenizer = func(cfg TokenizerConfig) (Tokenizer, error) {
		return NewTiktokenTokenizerStub(), ErrTokenizerNotImplemented
	}
}
