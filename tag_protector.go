package headroom

import "github.com/superops-team/headroom-go/internal/tagprotector"

type ProtectedContent = tagprotector.ProtectedContent
type TagProtector = tagprotector.TagProtector

func NewTagProtector() TagProtector {
	return tagprotector.NewTagProtector()
}
