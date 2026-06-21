package ccr

import "github.com/superops-team/headroom-go/internal/types"

type CCRStore interface {
	Store(original, compressed string, kind types.ContentKind) string
	Retrieve(id string) (string, bool)
}
