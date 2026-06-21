package headroom

import "github.com/superops-team/headroom-go/internal/router"

type ContentRouter = router.ContentRouter

func NewContentRouter() *ContentRouter {
	return router.NewContentRouter()
}
