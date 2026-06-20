package headroom

type CompressionEngine struct {
	tokenizer    Tokenizer
	tokenizerErr error
	detector     *ContentRouter
	policy       CompressionPolicy
	ccr          CCRStore
	observer     Observer
}

func NewCompressionEngine(opts Options) (*CompressionEngine, []Warning) {
	tokenizer, warnings, err := ResolveTokenizer(opts.TokenizerConfig)
	if err != nil {
		warnings = append(warnings, Warning{Code: "tokenizer_error", Component: "tokenizer", Message: err.Error()})
		if opts.TokenizerConfig.AllowFallback {
			tokenizer = FallbackTokenizer{}
		} else {
			tokenizer = nil
		}
	}
	observer := opts.Observer
	if observer == nil {
		observer = NoopObserver{}
	}
	return &CompressionEngine{tokenizer: tokenizer, tokenizerErr: err, detector: NewContentRouter(), policy: DefaultCompressionPolicy(opts.Aggressiveness), ccr: getPackageCCR(), observer: observer}, warnings
}

func (e *CompressionEngine) Compress(messages []Message, opts Options) (*Result, error) {
	if e.tokenizerErr != nil && !opts.TokenizerConfig.AllowFallback {
		return nil, e.tokenizerErr
	}
	if opts.EnablePipeline || opts.TokenBudget > 0 || opts.Query != "" {
		return e.compressWithPipeline(messages, opts)
	}
	return compressLegacy(messages, opts, e.tokenizer, nil, e.observer)
}

func (e *CompressionEngine) compressWithPipeline(messages []Message, opts Options) (*Result, error) {
	return runPipelineMessages(messages, opts, e)
}
