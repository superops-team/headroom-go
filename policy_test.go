package headroom

import "testing"

func TestCompressionPolicyModesAndDecision(t *testing.T) {
	if DefaultCompressionPolicy(0.1).Mode != PolicyConservative {
		t.Fatal("want conservative")
	}
	if DefaultCompressionPolicy(0.5).Mode != PolicyStandard {
		t.Fatal("want standard")
	}
	if DefaultCompressionPolicy(0.8).Mode != PolicyAggressive {
		t.Fatal("want aggressive")
	}
	p := DefaultCompressionPolicy(0.5)
	d, _ := p.Decide(CompressionContext{OriginalTokens: 10, TokenBudget: 20, Reversible: false})
	if !d.ShouldCompress || len(d.AllowedKinds) != 1 || d.AllowedKinds[0] != TransformReformat {
		t.Fatalf("bad decision %#v", d)
	}
	d, _ = p.Decide(CompressionContext{OriginalTokens: 100, Reversible: true})
	if !containsTransformKind(d.AllowedKinds, TransformOffload) {
		t.Fatalf("offload not allowed %#v", d)
	}
}
