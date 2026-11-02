package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/admincontent"
	acusecase "github.com/skinnykaen/robbo_student_personal_account.git/package/admincontent/usecase"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
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

type audit struct{ n int }

func (a *audit) Write(*models.AdminContentAuditDB) error { a.n++; return nil }
func (a *audit) ListForTarget(string, int) ([]models.AdminContentAuditDB, error) {
	return nil, nil
}

type pages struct{ projectPage.Gateway }

func (pages) GetScratchProjectById(id string) (*models.ScratchProjectDB, error) {
	if id != "p1" {
		return nil, projectPage.ErrPageNotFound
	}
	return &models.ScratchProjectDB{ID: "p1", OwnerUserID: "7", Title: "Кот"}, nil
}
func (pages) GetLatestSb3Archive(string) ([]byte, error) { return []byte("PK sb3"), nil }

type contentGW struct{ admincontent.Gateway }

func router(a fakeAuth, au *audit) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	uc := acusecase.NewAdminContentUseCase(acusecase.Deps{
		Gateway: contentGW{}, Audit: au, Projects: pages{},
		Names: func([]string) map[string]acusecase.UserName { return nil },
	})
	h := NewAdminContentHandler(a, uc)
	h.InitRoutes(r)
	return r
}

func do(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestOnlySuperAdmin(t *testing.T) {
	paths := [][2]string{
		{http.MethodGet, "/api/admin/users/7/content"},
		{http.MethodGet, "/api/admin/storage"},
		{http.MethodPost, "/api/admin/users/7/projects/p1/download"},
		{http.MethodDelete, "/api/admin/users/7/files/f1"},
	}
	for _, p := range paths {
		if w := do(router(fakeAuth{noLogin: true}, &audit{}), p[0], p[1], ""); w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without login: %d", p[0], p[1], w.Code)
		}
		for _, role := range []models.Role{models.Student, models.Teacher, models.UnitAdmin} {
			if w := do(router(fakeAuth{role: role}, &audit{}), p[0], p[1], ""); w.Code != http.StatusForbidden {
				t.Errorf("%s %s as role %d: %d", p[0], p[1], role, w.Code)
			}
		}
	}
}

func TestDownloadProject(t *testing.T) {
	au := &audit{}
	r := router(fakeAuth{role: models.SuperAdmin}, au)
	if w := do(r, http.MethodPost, "/api/admin/users/7/projects/p1/download", `{"reason":""}`); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "REASON_REQUIRED") {
		t.Fatalf("no reason: %d %s", w.Code, w.Body)
	}
	if w := do(r, http.MethodPost, "/api/admin/users/8/projects/p1/download", `{"reason":"жалоба"}`); w.Code != http.StatusNotFound {
		t.Fatalf("project of another user: %d", w.Code)
	}
	w := do(r, http.MethodPost, "/api/admin/users/7/projects/p1/download", `{"reason":"жалоба"}`)
	if w.Code != http.StatusOK || w.Body.String() != "PK sb3" || au.n != 1 {
		t.Fatalf("status=%d body=%q audit=%d", w.Code, w.Body, au.n)
	}
	h := w.Header()
	if !strings.HasPrefix(h.Get("Content-Disposition"), "attachment;") || h.Get("X-Content-Type-Options") != "nosniff" ||
		h.Get("Content-Security-Policy") != "sandbox; default-src 'none'" {
		t.Fatalf("headers=%v", h)
	}
}
