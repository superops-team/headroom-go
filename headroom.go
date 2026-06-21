package headroom

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// packageLevelCCR 是整个包共享的可逆压缩存储实例。
// 每次调用 Compress/CompressString 时使用同一实例，确保 id 跨调用可检索。
var (
	packageCCROnce sync.Once
	packageCCR     *CCR
)

func getPackageCCR() *CCR {
	packageCCROnce.Do(func() {
		packageCCR = NewCCR(CCRConfig{TTL: 24 * time.Hour})
	})
	return packageCCR
}

// Message 表示聊天消息，与 OpenAI Messages 格式兼容。
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// Options 控制压缩强度与可选项。
type Options struct {
	// Aggressiveness 控制压缩激进程度。0.0-0.3保守、0.3-0.7标准、0.7-1.0激进。
	// 默认 0.5。
	Aggressiveness float64
	// Reversible 是否启用可逆压缩：原始内容本地缓存，压缩输出末尾附加检索 id。
	// 默认 true。
	Reversible bool
	// AlignPrefix 是否在输出前加版本化前缀（提升 Provider side cache 命中率）。
	// 默认 false。
	AlignPrefix bool
	// TokenLimit 可选：估算 token 数低于该阈值时跳过压缩。0 表示不限制。
	// 默认 0。
	TokenLimit      int
	TokenizerConfig TokenizerConfig
	TokenBudget     int
	Query           string
	EnablePipeline  bool
	Observer        Observer
}

// Result 是 Compress 的输出。
type Result struct {
	Messages         []Message
	CompressedTokens int
	OriginalTokens   int
	Savings          float64
	Warnings         []Warning
	Steps            []CompressionStep
}

// DefaultOptions 返回推荐的默认选项。
func DefaultOptions() Options {
	return Options{
		Aggressiveness: 0.5,
		Reversible:     true,
		AlignPrefix:    false,
		TokenLimit:     0,
	}
}

// Compress 压缩一组聊天消息。assistant 角色消息原样透传。
func Compress(messages []Message, opts Options) (*Result, error) {
	engine, warnings := NewCompressionEngine(opts)
	result, err := engine.Compress(messages, opts)
	if result != nil && len(warnings) > 0 {
		result.Warnings = append(warnings, result.Warnings...)
	}
	return result, err
}

func compressLegacy(messages []Message, opts Options, tokenizer Tokenizer, initialWarnings []Warning, observer Observer) (*Result, error) {
	router := NewContentRouter()
	ccr := getPackageCCR()
	aligner := NewCacheAligner(CacheAlignerConfig{
		Enabled: opts.AlignPrefix,
		Version: PrefixVersion,
	})

	compressedMsgs := make([]Message, 0, len(messages))
	origTokens := 0
	compTokens := 0
	warnings := append([]Warning{}, initialWarnings...)
	steps := make([]CompressionStep, 0, len(messages))

	for _, m := range messages {
		msgTokens, err := countTokens(tokenizer, m.Content)
		if err != nil {
			return nil, err
		}
		origTokens += msgTokens

		if skipped, step := legacySkipMessage(m, opts, msgTokens); skipped {
			compressedMsgs = append(compressedMsgs, m)
			compTokens += msgTokens
			steps = append(steps, step)
			continue
		}

		kind, out, err := routeAndCompressLegacy(router, DefaultCompressorRegistry(), m.Content, opts)
		if err != nil {
			return nil, err
		}

		out, step, err := postProcessLegacyCompression(m.Content, out, kind, opts, tokenizer, msgTokens, aligner, ccr)
		if err != nil {
			return nil, err
		}
		steps = append(steps, step)

		compressedMsgs = append(compressedMsgs, Message{
			Role:    m.Role,
			Content: out,
			Name:    m.Name,
		})
		outTokens, err := countTokens(tokenizer, out)
		if err != nil {
			return nil, err
		}
		compTokens += outTokens
	}

	return buildLegacyResult(compressedMsgs, origTokens, compTokens, warnings, steps, observer), nil
}

func legacySkipMessage(m Message, opts Options, msgTokens int) (bool, CompressionStep) {
	baseStep := CompressionStep{Kind: KindText.String(), TokensBefore: msgTokens, TokensAfter: msgTokens, Skipped: true}
	if m.Role == "assistant" {
		baseStep.Name = "skip_assistant"
		baseStep.Reason = "assistant role"
		return true, baseStep
	}
	if strings.TrimSpace(m.Content) == "" {
		baseStep.Name = "skip_empty"
		baseStep.Reason = "empty content"
		return true, baseStep
	}
	if opts.TokenLimit > 0 && msgTokens < opts.TokenLimit {
		baseStep.Name = "skip_token_limit"
		baseStep.Reason = "below token limit"
		return true, baseStep
	}
	return false, CompressionStep{}
}

func routeAndCompressLegacy(router *ContentRouter, registry *CompressorRegistry, content string, opts Options) (ContentKind, string, error) {
	kind := router.Detect(content)
	out, err := registry.Compress(kind, content, opts)
	if err != nil {
		return kind, "", fmt.Errorf("compress %s: %w", kind.String(), err)
	}
	return kind, out, nil
}

func postProcessLegacyCompression(original, compressed string, kind ContentKind, opts Options, tokenizer Tokenizer, msgTokens int, aligner *CacheAligner, ccr *CCR) (string, CompressionStep, error) {
	out := applyAlignPrefix(compressed, opts, aligner)
	out = applyReversibleCCR(original, out, kind, opts, ccr)
	if fallbackContent, fallbackStep, fallback := applyFallbackIfLonger(original, out, kind, msgTokens); fallback {
		return fallbackContent, fallbackStep, nil
	}

	outTokens, err := countTokens(tokenizer, out)
	if err != nil {
		return "", CompressionStep{}, err
	}
	return out, CompressionStep{Name: "legacy_compress", Kind: kind.String(), TokensBefore: msgTokens, TokensAfter: outTokens}, nil
}

func applyAlignPrefix(out string, opts Options, aligner *CacheAligner) string {
	if opts.AlignPrefix {
		out = aligner.Align(out)
	}
	return out
}

func applyReversibleCCR(original, out string, kind ContentKind, opts Options, ccr *CCR) string {
	if opts.Reversible {
		id := ccr.Store(original, out, kind)
		retrieveSuffix := "\n\n[headroom:retrieve id=" + id + "]"
		out += retrieveSuffix
	}
	return out
}

func applyFallbackIfLonger(original, out string, kind ContentKind, msgTokens int) (string, CompressionStep, bool) {
	if len(out) >= len(original) {
		return original, CompressionStep{Name: "legacy_compress", Kind: kind.String(), TokensBefore: msgTokens, TokensAfter: msgTokens, Skipped: true, Reason: "output not shorter"}, true
	}
	return out, CompressionStep{}, false
}

func buildLegacyResult(messages []Message, origTokens, compTokens int, warnings []Warning, steps []CompressionStep, observer Observer) *Result {
	savings := 0.0
	if origTokens > 0 {
		savings = float64(origTokens-compTokens) / float64(origTokens)
	}
	if observer != nil {
		for _, step := range steps {
			observer.ObserveCompressionStep(step)
		}
	}
	return &Result{Messages: messages, CompressedTokens: compTokens, OriginalTokens: origTokens, Savings: savings, Warnings: warnings, Steps: steps}
}

// CompressString 压缩单段文本。适合快速测试或单次内容压缩。
func CompressString(content string, opts Options) (string, error) {
	r, err := Compress([]Message{{Role: "user", Content: content}}, opts)
	if err != nil {
		return "", err
	}
	if len(r.Messages) == 0 {
		return "", nil
	}
	return r.Messages[0].Content, nil
}

// estimateTokens 估算 token 数（按 ~4 chars/token 的粗略估算）。
func estimateTokens(s string) int {
	n, _ := FallbackTokenizer{}.Count(s)
	return n
}

func countTokens(tokenizer Tokenizer, content string) (int, error) {
	if tokenizer == nil {
		tokenizer = FallbackTokenizer{}
	}
	return tokenizer.Count(content)
}
