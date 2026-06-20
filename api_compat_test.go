package headroom

import "testing"

var _ func([]Message, Options) (*Result, error) = Compress
var _ func(string, Options) (string, error) = CompressString
var _ func(Options) (*CompressionEngine, []Warning) = NewCompressionEngine

func TestPublicAPICompatibility(t *testing.T) {
	opts := Options{Aggressiveness: 0.5, Reversible: true, AlignPrefix: false, TokenLimit: 10}
	msgs := []Message{{Role: "user", Content: "hello world", Name: "n"}}
	result, err := Compress(msgs, opts)
	if err != nil {
		t.Fatal(err)
	}
	var _ []Message = result.Messages
	var _ int = result.OriginalTokens
	var _ int = result.CompressedTokens
	var _ float64 = result.Savings
	var _ []Warning = result.Warnings
	var _ []CompressionStep = result.Steps
	var _ string = result.Messages[0].Role
	var _ string = result.Messages[0].Content
	var _ string = result.Messages[0].Name
	var warning Warning
	var _ string = warning.Code
	var _ string = warning.Component
	var _ string = warning.Message
	var step CompressionStep
	var _ string = step.Name
	var _ string = step.Kind
	var _ int = step.TokensBefore
	var _ int = step.TokensAfter
	var _ bool = step.Skipped
	var _ string = step.Reason
	if _, err := CompressString("hello", opts); err != nil {
		t.Fatal(err)
	}
}
