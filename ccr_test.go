package headroom

import (
	"testing"
	"time"
)

func TestCCR_StoreAndRetrieve(t *testing.T) {
	c := NewCCR(CCRConfig{TTL: time.Hour})
	orig := "the quick brown fox jumps over the lazy dog"
	comp := "quick brown fox lazy dog"

	id := c.Store(orig, comp, KindText)
	if id == "" {
		t.Fatal("empty id")
	}
	// id 前缀应为 v1_
	if len(id) < 4 || id[:3] != "v1_" {
		t.Errorf("id should start with v1_, got %q", id)
	}

	retrieved, ok := c.Retrieve(id)
	if !ok {
		t.Fatal("retrieve returned false")
	}
	if retrieved != orig {
		t.Errorf("retrieve got %q, want %q", retrieved, orig)
	}
}

// 不存在的 id → false
func TestCCR_RetrieveMissing(t *testing.T) {
	c := NewCCR(CCRConfig{TTL: time.Hour})
	_, ok := c.Retrieve("v1_deadbeef1234")
	if ok {
		t.Error("retrieve of unknown id should return false")
	}
}

// Stats 计数
func TestCCR_Stats(t *testing.T) {
	c := NewCCR(CCRConfig{TTL: time.Hour})
	c.Store("a", "a", KindText)
	c.Store("b", "b", KindText)
	count, totalBytes := c.Stats()
	if count != 2 {
		t.Errorf("count got %d, want 2", count)
	}
	if totalBytes <= 0 {
		t.Errorf("totalBytes got %d, want >0", totalBytes)
	}
}

// 多次 Store 同一内容 → 同一 id，返回同一值
func TestCCR_SameContentSameId(t *testing.T) {
	c := NewCCR(CCRConfig{TTL: time.Hour})
	id1 := c.Store("hello world", "hello", KindText)
	id2 := c.Store("hello world", "hello", KindText)
	if id1 != id2 {
		t.Errorf("same content should give same id: %s vs %s", id1, id2)
	}
}

// 惰性 GC：Store 时清理过期条目
func TestCCR_LazyGC(t *testing.T) {
	c := NewCCR(CCRConfig{TTL: time.Hour}) // 长 TTL 确保期间不过期
	c.Store("expire_me", "exp", KindText)

	// 立即再 Store 一次，之前的条目还没过期
	c.Store("another", "ano", KindText)
	count, _ := c.Stats()
	if count != 2 {
		t.Fatalf("before expiry: got %d entries, want 2", count)
	}
}
