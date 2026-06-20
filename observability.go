package headroom

type Warning struct {
	Code      string
	Component string
	Message   string
}

type CompressionStep struct {
	Name         string
	Kind         string
	TokensBefore int
	TokensAfter  int
	Skipped      bool
	Reason       string
}

type Observer interface {
	ObserveCompressionStep(step CompressionStep)
}

type NoopObserver struct{}

func (NoopObserver) ObserveCompressionStep(step CompressionStep) {}
