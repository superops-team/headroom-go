// Package headroom provides intelligent context compression for AI agents.
//
// This file carries historical compatibility APIs that are kept for backward
// compatibility with existing callers. New code should prefer the primary
// API in headroom.go.
package headroom

import (
	"errors"

	"github.com/superops-team/headroom-go/internal/compressors"
)

// ── Historical Config Types ─────────────────────────────────────────────────

// CompressionConfig is the legacy compression configuration type.
type CompressionConfig = compressors.CompressionConfig

// CodeConfig configures code compression.
type CodeConfig = compressors.CodeConfig

// TextConfig configures text compression.
type TextConfig = compressors.TextConfig

// SmartCrushConfig configures SmartCrush JSON compression.
type SmartCrushConfig = compressors.SmartCrushConfig

// ── Historical Compressor Types ─────────────────────────────────────────────

// Compressor compresses content of a specific kind.
type Compressor = compressors.Compressor

// CompressorFunc wraps a function as a Compressor.
type CompressorFunc = compressors.CompressorFunc

// CompressorRegistry maps content kinds to compressors.
type CompressorRegistry = compressors.CompressorRegistry

// ── Historical Compression Functions ────────────────────────────────────────

// SmartCrushJSON compresses JSON content using SmartCrush.
func SmartCrushJSON(content string, cfg SmartCrushConfig) (string, error) {
	return compressors.SmartCrushJSON(content, cfg)
}

// SmartCrushJSONWithSteps compresses JSON and returns per-step details.
func SmartCrushJSONWithSteps(content string, cfg SmartCrushConfig) (string, []CompressionStep, error) {
	return compressors.SmartCrushJSONWithSteps(content, cfg)
}

// CompressCode compresses code content.
func CompressCode(content string, cfg CodeConfig) string {
	return compressors.CompressCode(content, cfg)
}

// CompressText compresses text content.
func CompressText(content string, cfg TextConfig) string {
	return compressors.CompressText(content, cfg)
}

// ── Registry Constructors ───────────────────────────────────────────────────

// NewCompressorFunc creates a CompressorFunc for the given content kind.
func NewCompressorFunc(kind ContentKind, fn func(string, Options) (string, error)) CompressorFunc {
	return compressors.NewCompressorFunc(kind, fn)
}

// NewCompressorRegistry creates a new empty CompressorRegistry.
func NewCompressorRegistry() *CompressorRegistry {
	return compressors.NewCompressorRegistry()
}

// DefaultCompressorRegistry returns a registry pre-configured with all built-in compressors.
func DefaultCompressorRegistry() *CompressorRegistry {
	return compressors.DefaultCompressorRegistry()
}

// ── Internal Helpers ────────────────────────────────────────────────────────

// errorTokenizer is a tokenizer that always returns an error (for testing).
type errorTokenizer struct{}

func (errorTokenizer) Name() string { return "error" }
func (errorTokenizer) Count(string) (int, error) {
	return 0, errors.New("tokenizer boom")
}
func (errorTokenizer) CountBatch([]string) ([]int, error) {
	return nil, errors.New("tokenizer boom")
}

// lineIndent returns the indentation width of the first line.
func lineIndent(s string) int {
	n := 0
	for _, r := range s {
		if r == ' ' {
			n++
		} else if r == '\t' {
			n += 4
		} else {
			break
		}
	}
	return n
}
