package headroom

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

const defaultMaxEntries = 10000

type CCRConfig struct {
	TTL        time.Duration // 条目过期时间，默认 24 小时
	MaxEntries int           // 最大条目数，默认 10000
}

type ccrEntry struct {
	Original string
	Kind     ContentKind
	StoredAt time.Time
}

type CCR struct {
	mu   sync.RWMutex
	data map[string]*ccrEntry
	cfg  CCRConfig
}

func NewCCR(cfg CCRConfig) *CCR {
	if cfg.TTL <= 0 {
		cfg.TTL = 24 * time.Hour
	}
	if cfg.MaxEntries <= 0 {
		cfg.MaxEntries = defaultMaxEntries
	}
	c := &CCR{
		data: make(map[string]*ccrEntry, 128),
		cfg:  cfg,
	}
	// 后台 GC：每 30 分钟清理过期条目
	go c.backgroundGC()
	return c
}

// backgroundGC 定期清理过期条目。
func (c *CCR) backgroundGC() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		c.collectExpired()
	}
}

// Store 保存原始内容与压缩内容，返回可检索 id（legacy 格式 v2_SHA256前12字符）。
// 相同内容重复 Store 返回相同 id，且更新 StoredAt 时间。
func (c *CCR) Store(original, compressed string, kind ContentKind) string {
	// 惰性 GC：Store 前先清理过期条目
	c.collectExpired()

	id := LegacyCCRIDVersion + "_" + sha256Prefix12(original)

	c.mu.Lock()
	defer c.mu.Unlock()

	// MaxEntries 上限：FIFO 淘汰最旧条目
	if c.shouldEvictBeforeStoreLocked(id) {
		c.evictOldest()
	}

	c.data[id] = &ccrEntry{
		Original: original,
		Kind:     kind,
		StoredAt: time.Now(),
	}
	return id
}

func (c *CCR) isFullLocked() bool {
	return c.cfg.MaxEntries > 0 && len(c.data) >= c.cfg.MaxEntries
}

func (c *CCR) containsLocked(id string) bool {
	_, exists := c.data[id]
	return exists
}

func (c *CCR) shouldEvictBeforeStoreLocked(id string) bool {
	return c.isFullLocked() && !c.containsLocked(id)
}

// evictOldest 淘汰最旧的条目（调用方需持有 c.mu 写锁）。
func (c *CCR) evictOldest() {
	var oldestID string
	var oldestTime time.Time
	first := true
	for id, e := range c.data {
		if first || e.StoredAt.Before(oldestTime) {
			oldestID = id
			oldestTime = e.StoredAt
			first = false
		}
	}
	if oldestID != "" {
		delete(c.data, oldestID)
	}
}

// Retrieve 按 id 取回原始内容。
func (c *CCR) Retrieve(id string) (string, bool) {
	c.mu.RLock()
	e, ok := c.data[id]
	c.mu.RUnlock()
	if !ok {
		return "", false
	}
	if time.Since(e.StoredAt) > c.cfg.TTL {
		return "", false
	}
	return e.Original, true
}

// Stats 返回当前条目数与原始内容总字节数。
func (c *CCR) Stats() (int, int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	total := 0
	count := 0
	for _, e := range c.data {
		if time.Since(e.StoredAt) <= c.cfg.TTL {
			count++
			total += len(e.Original)
		}
	}
	return count, total
}

// collectExpired 惰性清理过期条目。
func (c *CCR) collectExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for id, e := range c.data {
		if now.Sub(e.StoredAt) > c.cfg.TTL {
			delete(c.data, id)
		}
	}
}

// sha256Prefix12 返回 SHA256 哈希的前 12 个十六进制字符。
func sha256Prefix12(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	sum := h.Sum(nil)
	hex := hex.EncodeToString(sum)
	if len(hex) > 12 {
		return hex[:12]
	}
	return hex
}
