package headroom

import "github.com/superops-team/headroom-go/internal/types"

// Warning represents a non-fatal issue encountered during compression.
//
// Warnings are collected in Result.Warnings. They do not cause compression
// to fail — the output is still valid, but may be suboptimal.
//
// Fields:
//   - Code: machine-readable warning code (e.g., "tokenizer_count_error")
//   - Component: the component that generated the warning (e.g., "pipeline")
//   - Message: human-readable description
type Warning = types.Warning

// CompressionStep records a single step in the compression pipeline.
//
// Each message goes through multiple steps: content detection, compression,
// optional post-processing. Steps are collected in Result.Steps for
// observability and debugging.
//
// Fields:
//   - Name: step identifier (e.g., "legacy_compress", "json_minifier")
//   - Kind: content type being processed
//   - TokensBefore: estimated token count before this step
//   - TokensAfter: estimated token count after this step
//   - Skipped: true if this step was bypassed
//   - Reason: why the step was skipped (if Skipped is true)
type CompressionStep = types.CompressionStep

// Observer receives compression step notifications.
//
// Implement this interface to monitor compression progress in real-time.
// The observer is called for each step in the compression pipeline.
//
// Example:
//
//	type myObserver struct{}
//	func (myObserver) ObserveCompressionStep(step headroom.CompressionStep) {
//	    log.Printf("[%s] %s: %d → %d tokens", step.Name, step.Kind, step.TokensBefore, step.TokensAfter)
//	}
type Observer = types.Observer

// NoopObserver is a no-op implementation of Observer.
// Use as a default when no observation is needed.
type NoopObserver = types.NoopObserver
