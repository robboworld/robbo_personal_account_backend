package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// UserIdentity returns a *gqlerror.Error wrapping the auth sentinel; a guest must get 401, not 500.
func TestErrorHandlingWrappedAuthErrorsAre401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, sentinel := range []error{auth.ErrTokenNotFound, auth.ErrInvalidAccessToken} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		ErrorHandling(&gqlerror.Error{Err: sentinel, Message: sentinel.Error()}, c)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("%v: status %d, want 401", sentinel, w.Code)
		}
	}
}
