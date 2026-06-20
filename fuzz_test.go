//go:build go1.18
// +build go1.18

package headroom

import (
	"strings"
	"testing"
)

func FuzzTagProtectorRestoreKeepsTags(f *testing.F) {
	f.Add("<system-reminder>keep me</system-reminder> body")
	f.Add("<tool_call>{\"name\":\"x\"}</tool_call> text <custom/> end")
	f.Fuzz(func(t *testing.T, s string) {
		wrapped := "<system-reminder>" + s + "</system-reminder>"
		p := NewTagProtector()
		protected := p.Protect(wrapped)
		restored, warnings := p.Restore(protected)
		if len(warnings) != 0 {
			t.Fatalf("unexpected warnings: %#v", warnings)
		}
		if restored != wrapped {
			t.Fatalf("restore changed protected content: got %q want %q", restored, wrapped)
		}
		if !strings.Contains(restored, "<system-reminder>") || !strings.Contains(restored, "</system-reminder>") {
			t.Fatalf("protected tag lost: %q", restored)
		}
	})
}

func FuzzCompressDoesNotPanic(f *testing.F) {
	f.Add("hello world")
	f.Add("{\"items\":[1,2,3]}")
	f.Add("diff --git a/a.go b/a.go\n--- a/a.go\n+++ b/a.go\n@@ -1 +1 @@\n-old\n+new")
	f.Fuzz(func(t *testing.T, s string) {
		opts := DefaultOptions()
		opts.EnablePipeline = true
		opts.Reversible = false
		if _, err := Compress([]Message{{Role: "user", Content: s}}, opts); err != nil {
			t.Fatalf("default compression should not fail: %v", err)
		}
	})
}
