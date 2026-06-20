package headroom

import (
	"strconv"
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
	// id 前缀应为 v2_
	if len(id) < 4 || id[:3] != "v2_" {
		t.Errorf("id should start with v2_, got %q", id)
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
	_, ok := c.Retrieve("v2_deadbeef1234")
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

// MaxEntries：超过上限时淘汰最旧条目
func TestCCR_MaxEntries(t *testing.T) {
	c := NewCCR(CCRConfig{TTL: time.Hour, MaxEntries: 100})
	// Store 101 条不同内容
	for i := 0; i < 101; i++ {
		content := "entry-" + strconv.Itoa(i)
		c.Store(content, content, KindText)
	}
	count, _ := c.Stats()
	if count > 100 {
		t.Errorf("MaxEntries=100 but got %d entries", count)
	}
}

func TestCCR_MaxEntriesEvictsOldestWhenStoringNewID(t *testing.T) {
	c := NewCCR(CCRConfig{TTL: time.Hour, MaxEntries: 1})
	idA := c.Store("A", "a", KindText)
	idB := c.Store("B", "b", KindText)
	if idA == idB {
		t.Fatal("different content should have different ids")
	}
	if _, ok := c.Retrieve(idA); ok {
		t.Fatalf("oldest id %q should have been evicted", idA)
	}
	if got, ok := c.Retrieve(idB); !ok || got != "B" {
		t.Fatalf("new id %q should remain, got=%q ok=%v", idB, got, ok)
	}
}

func TestCCR_MaxEntriesRepeatStoreDoesNotEvictSameID(t *testing.T) {
	c := NewCCR(CCRConfig{TTL: time.Hour, MaxEntries: 1})
	id1 := c.Store("A", "a", KindText)
	id2 := c.Store("A", "a2", KindText)
	if id1 != id2 {
		t.Fatalf("repeat store should return same id: %q vs %q", id1, id2)
	}
	if got, ok := c.Retrieve(id1); !ok || got != "A" {
		t.Fatalf("repeat-stored id should remain, got=%q ok=%v", got, ok)
	}
	count, _ := c.Stats()
	if count != 1 {
		t.Fatalf("repeat store should keep one entry, got %d", count)
	}
}

// 后台 GC：过期条目被 Ticker 清理
func TestCCR_BackgroundGC(t *testing.T) {
	c := NewCCR(CCRConfig{TTL: 50 * time.Millisecond, MaxEntries: 1000})
	c.Store("will-expire", "x", KindText)

	// 等待 TTL 过期 + Ticker 触发（30min 默认 ticker 太长，这里验证惰性 GC 即可）
	// 后台 GC 的 Ticker 间隔为 30min，单元测试中不等待 Ticker。
	// 验证：过期后 Retrieve 返回 false
	time.Sleep(100 * time.Millisecond)
	_, ok := c.Retrieve("v2_" + sha256Prefix12("will-expire"))
	if ok {
		t.Error("expired entry should not be retrievable")
	}
	// Store 触发惰性 GC，清理过期条目
	c.Store("new-entry", "y", KindText)
	count, _ := c.Stats()
	if count != 1 {
		t.Errorf("after GC: got %d entries, want 1", count)
	}
}
