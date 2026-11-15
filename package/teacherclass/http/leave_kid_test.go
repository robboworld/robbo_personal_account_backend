package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
)

type leaveTestAuth struct{ auth.Delegate }

func (leaveTestAuth) UserIdentity(*gin.Context) (string, models.Role, error) {
	return "5", models.Student, nil
}

func (leaveTestAuth) UserAccess(models.Role, []models.Role, context.Context) error { return nil }

type leaveTestUseCase struct {
	teacherclass.UseCase
	left bool
}

func (u *leaveTestUseCase) LeaveClass(string, string) error {
	u.left = true
	return nil
}

func postLeave(t *testing.T, email string) (*httptest.ResponseRecorder, *leaveTestUseCase) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		auth.MarkKidAccountSession(c, email)
		c.Next()
	})
	uc := &leaveTestUseCase{}
	h := &Handler{authDelegate: leaveTestAuth{}, uc: uc}
	h.InitRoutes(router)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/student/classes/c1/leave", nil))
	return rec, uc
}

// A kid account lives in its one class: leaving would close the class sign-in, so the server
// refuses it, not only the hidden button.
func TestLeaveClassRefusesKidAccount(t *testing.T) {
	rec, uc := postLeave(t, "k-abc@kids.robbo.invalid")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d %s", rec.Code, rec.Body.String())
	}
	if uc.left {
		t.Fatal("a kid account left its class")
	}
}

func TestLeaveClassKeepsFullAccount(t *testing.T) {
	rec, uc := postLeave(t, "parent@example.com")
	if rec.Code != http.StatusNoContent || !uc.left {
		t.Fatalf("full account: want 204 and a leave, got %d left=%v", rec.Code, uc.left)
	}
}
