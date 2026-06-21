package headroom

import "github.com/superops-team/headroom-go/internal/tagprotector"

// ProtectedContent holds content with protected XML tags extracted.
//
// Tags like <thinking>, <tool_call>, <function_result> are extracted
// before compression and restored after, preventing compression from
// mangling structured agent outputs.
type ProtectedContent = tagprotector.ProtectedContent

// TagProtector preserves XML tags during compression.
//
// Extracts protected tags before compression and restores them after.
// Built-in protected tags: thinking, tool_call, tool_result, function_call,
// function_result, scratchpad, reasoning, reflection.
//
// Example:
//
//	tp := headroom.NewTagProtector()
//	pc := tp.Protect("<thinking>Let me analyze...</thinking>\nSome text")
//	// pc.Content = "Some text" (tags extracted)
//	// Compress pc.Content here...
//	restored := tp.Restore(pc)
//	// restored = "<thinking>Let me analyze...</thinking>\nCompressed text"
type TagProtector = tagprotector.TagProtector

// NewTagProtector creates a new TagProtector with default protected tags.
func NewTagProtector() TagProtector {
	return tagprotector.NewTagProtector()
}
