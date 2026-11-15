package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/usersearch"
)

type fakeAuth struct {
	auth.Delegate
	role    models.Role
	noLogin bool
}

func (a fakeAuth) UserIdentity(*gin.Context) (string, models.Role, error) {
	if a.noLogin {
		return "", models.Anonymous, errors.New("token not found")
	}
	return "1", a.role, nil
}

func (a fakeAuth) UserAccess(role models.Role, roles []models.Role, _ context.Context) error {
	for _, r := range roles {
		if r == role {
			return nil
		}
	}
	return errors.New("no access")
}

func get(a fakeAuth, path string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// A service with no LMS reader behind it: any call that reaches MySQL fails with 500.
	h := NewHandler(a, &usersearch.Service{})
	h.InitRoutes(r)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

func TestEmptyQueryListsUsersOnlyForSuperAdmin(t *testing.T) {
	w := get(fakeAuth{role: models.UnitAdmin}, "/api/users/search?beforeId=50&includeInactive=true")
	if w.Code != http.StatusOK || w.Body.String() != `{"items":[]}` {
		t.Fatalf("UnitAdmin without a query: got %d %s, want 200 and no users", w.Code, w.Body.String())
	}
}

func TestSearchNeedsAdmin(t *testing.T) {
	if w := get(fakeAuth{noLogin: true}, "/api/users/search"); w.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous: got %d, want 401", w.Code)
	}
	if w := get(fakeAuth{role: models.Teacher}, "/api/users/search"); w.Code != http.StatusForbidden {
		t.Fatalf("teacher: got %d, want 403", w.Code)
	}
}
