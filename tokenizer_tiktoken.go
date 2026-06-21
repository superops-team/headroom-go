//go:build tokenizer_tiktoken
// +build tokenizer_tiktoken

package headroom

import "github.com/superops-team/headroom-go/internal/tokenizer"

func NewTiktokenTokenizerStub() Tokenizer {
	return tokenizer.NewTiktokenTokenizerStub()
}
