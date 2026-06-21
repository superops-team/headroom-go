package headroom

import (
	"github.com/superops-team/headroom-go/internal/ccr"
	eng "github.com/superops-team/headroom-go/internal/engine"
)

type CCRConfig = ccr.CCRConfig
type CCR = ccr.CCR

func NewCCR(cfg CCRConfig) *CCR {
	return ccr.NewCCR(cfg)
}

func getPackageCCR() *CCR {
	return eng.GetPackageCCR()
}
