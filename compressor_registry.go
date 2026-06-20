package headroom

import (
	"errors"
	"sync"
)

type Compressor interface {
	Kind() ContentKind
	Compress(content string, opts Options) (string, error)
}

type CompressorFunc struct {
	kind ContentKind
	fn   func(string, Options) (string, error)
}

func NewCompressorFunc(kind ContentKind, fn func(string, Options) (string, error)) CompressorFunc {
	return CompressorFunc{kind: kind, fn: fn}
}

func (c CompressorFunc) Kind() ContentKind { return c.kind }
func (c CompressorFunc) Compress(content string, opts Options) (string, error) {
	if c.fn == nil {
		return content, errors.New("compressor function is nil")
	}
	return c.fn(content, opts)
}

type CompressorRegistry struct {
	mu          sync.RWMutex
	compressors map[ContentKind]Compressor
}

func NewCompressorRegistry() *CompressorRegistry {
	return &CompressorRegistry{compressors: make(map[ContentKind]Compressor)}
}

func (r *CompressorRegistry) Register(c Compressor) {
	if c == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.compressors[c.Kind()] = c
}

func (r *CompressorRegistry) Lookup(kind ContentKind) (Compressor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.compressors[kind]
	return c, ok
}

func (r *CompressorRegistry) Compress(kind ContentKind, content string, opts Options) (string, error) {
	if c, ok := r.Lookup(kind); ok {
		return c.Compress(content, opts)
	}
	if c, ok := r.Lookup(KindText); ok {
		return c.Compress(content, opts)
	}
	return content, nil
}

func compressionConfigFromOptions(opts Options) CompressionConfig {
	return CompressionConfig{Aggressiveness: opts.Aggressiveness}
}

func smartCrushConfigFromOptions(opts Options) SmartCrushConfig {
	return SmartCrushConfig{Aggressiveness: opts.Aggressiveness}
}

var (
	defaultCompressorRegistryOnce sync.Once
	defaultCompressorRegistry     *CompressorRegistry
)

func DefaultCompressorRegistry() *CompressorRegistry {
	defaultCompressorRegistryOnce.Do(func() {
		defaultCompressorRegistry = NewCompressorRegistry()
		defaultCompressorRegistry.Register(NewCompressorFunc(KindJSON, func(content string, opts Options) (string, error) {
			return SmartCrushJSON(content, smartCrushConfigFromOptions(opts))
		}))
		defaultCompressorRegistry.Register(NewCompressorFunc(KindCode, func(content string, opts Options) (string, error) {
			return CompressCode(content, compressionConfigFromOptions(opts)), nil
		}))
		defaultCompressorRegistry.Register(NewCompressorFunc(KindText, func(content string, opts Options) (string, error) {
			return CompressText(content, compressionConfigFromOptions(opts)), nil
		}))
	})
	return defaultCompressorRegistry
}
