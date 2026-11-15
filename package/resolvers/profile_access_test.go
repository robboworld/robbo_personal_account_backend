package resolvers

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

func studentContext(email string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("user_id", "5")
	c.Set("user_role", models.Student)
	auth.MarkKidAccountSession(c, email)
	return c
}

// A kid account's session may not update even its own profile; a full account still may.
func TestProfileSelfAccessRefusesKidAccount(t *testing.T) {
	admins := []models.Role{models.UnitAdmin, models.SuperAdmin}
	err := profileSelfOrAdminAccess(context.Background(), studentContext("k-abc@kids.robbo.invalid"),
		nil, "5", models.Student, admins)
	if !errors.Is(err, auth.ErrKidAccount) {
		t.Fatalf("kid account: want ErrKidAccount, got %v", err)
	}
	err = profileSelfOrAdminAccess(context.Background(), studentContext("pupil@example.com"),
		nil, "5", models.Student, admins)
	if err != nil {
		t.Fatalf("full account, own profile: want access, got %v", err)
	}
}
