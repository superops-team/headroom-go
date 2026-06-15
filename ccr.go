package headroom

import (
	"crypto/sha1"
	"encoding/hex"
	"sync"
	"time"
)

type CCRConfig struct {
	TTL time.Duration // 条目过期时间，默认 24 小时
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
	return &CCR{
		data: make(map[string]*ccrEntry, 128),
		cfg:  cfg,
	}
}

// Store 保存原始内容与压缩内容，返回可检索 id（格式 v1_SHA1前12字符）。
// 相同内容重复 Store 返回相同 id，且更新 StoredAt 时间。
func (c *CCR) Store(original, compressed string, kind ContentKind) string {
	// 惰性 GC：Store 前先清理过期条目
	c.collectExpired()

	id := "v1_" + sha1Prefix12(original)

	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[id] = &ccrEntry{
		Original: original,
		Kind:     kind,
		StoredAt: time.Now(),
	}
	return id
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
		// 理论上已过期：惰性清理会在 Store 时删除，这里做二次检查
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

// sha1Prefix12 返回 SHA1 哈希的前 12 个十六进制字符。
func sha1Prefix12(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	sum := h.Sum(nil)
	hex := hex.EncodeToString(sum)
	if len(hex) > 12 {
		return hex[:12]
	}
	return hex
}
