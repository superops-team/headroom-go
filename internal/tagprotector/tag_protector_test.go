package tagprotector

import (
	"strings"
	"testing"
)

func TestTagProtectorProtectRestore(t *testing.T) {
	p := NewTagProtector()
	src := "before <system-reminder>keep <tool_call/> nested</system-reminder> <div>html</div>"
	protected := p.Protect(src)
	if strings.Contains(protected.Text, "system-reminder") {
		t.Fatalf("tag not protected: %s", protected.Text)
	}
	if !strings.Contains(protected.Text, "<div>html</div>") {
		t.Fatalf("html should remain: %s", protected.Text)
	}
	restored, warnings := p.Restore(protected)
	if len(warnings) != 0 {
		t.Fatalf("warnings %#v", warnings)
	}
	if restored != src {
		t.Fatalf("got %q want %q", restored, src)
	}
}

func TestTagProtectorPlaceholderCollision(t *testing.T) {
	p := NewTagProtector()
	src := "__HEADROOM_PROTECTED_TAG_0__ <tool_call/>"
	protected := p.Protect(src)
	if strings.Contains(protected.Text, "__HEADROOM_PROTECTED_TAG_0__ <tool_call") {
		t.Fatalf("collision not avoided: %s", protected.Text)
	}
	restored, _ := p.Restore(protected)
	if restored != src {
		t.Fatalf("got %q", restored)
	}
}

func TestTagProtectorMultiplePlaceholderCollision(t *testing.T) {
	p := NewTagProtector()
	src := "__HEADROOM_PROTECTED_TAG_0__ <tool_call/> <system-reminder>keep</system-reminder>"
	protected := p.Protect(src)
	if len(protected.Placeholders) != 2 {
		t.Fatalf("expected two placeholders, got %#v", protected.Placeholders)
	}
	restored, _ := p.Restore(protected)
	if restored != src {
		t.Fatalf("got %q want %q", restored, src)
	}
}

func TestTagProtectorMixedCaseAndNestedTags(t *testing.T) {
	p := NewTagProtector()
	src := "before <Tool_Call>outer <tool_call>inner</tool_call> tail</Tool_Call> after"
	protected := p.Protect(src)
	if strings.Contains(protected.Text, "outer") || strings.Contains(protected.Text, "inner") || strings.Contains(protected.Text, "Tool_Call") {
		t.Fatalf("nested mixed-case tag was not fully protected: %s", protected.Text)
	}
	restored, _ := p.Restore(protected)
	if restored != src {
		t.Fatalf("got %q want %q", restored, src)
	}
}

func TestTagProtectorUnclosedCustomTagProtectsTail(t *testing.T) {
	p := NewTagProtector()
	src := "before <system-reminder>keep tail"
	protected := p.Protect(src)
	if strings.Contains(protected.Text, "keep tail") {
		t.Fatalf("unclosed custom tag tail was not protected: %s", protected.Text)
	}
	restored, _ := p.Restore(protected)
	if restored != src {
		t.Fatalf("got %q want %q", restored, src)
	}
}
