package http

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

// Projects are saved only as .sb3 via /projectPage/:id/upload, which enforces the
// project size limit and the account quota; /project must stay read-only.
func TestProjectRoutesAreReadOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := &Handler{}
	h.InitProjectRoutes(router)

	for _, r := range router.Routes() {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected write route %s %s", r.Method, r.Path)
		}
	}
}
