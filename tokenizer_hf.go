//go:build tokenizer_hf
// +build tokenizer_hf

package headroom

import "github.com/superops-team/headroom-go/internal/tokenizer"

func NewHuggingFaceTokenizerStub() Tokenizer {
	return tokenizer.NewHuggingFaceTokenizerStub()
}
