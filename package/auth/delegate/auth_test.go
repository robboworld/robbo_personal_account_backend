package delegate

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth/usecase"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/oidc"
	"github.com/spf13/viper"
)

const testSigningKey = "test-access-signing-key-0123456789abcdef"

func identityContext(t *testing.T, bearer string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/api/users/search", nil)
	if bearer != "" {
		c.Request.Header.Set("Authorization", "Bearer "+bearer)
	}
	return c
}

func newDelegate() *AuthDelegateImpl {
	viper.Set("auth.access_signing_key", testSigningKey)
	return &AuthDelegateImpl{&usecase.AuthUseCaseImpl{}}
}

func superAdminBFFToken(t *testing.T) string {
	t.Helper()
	viper.Set("auth.access_signing_key", testSigningKey)
	token, err := oidc.IssueSessionToken("admin", "800", "admin@example.test", uint(models.SuperAdmin), "revoked-sid")
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// A revoked BFF token: the middleware marked the request Anonymous; the header must not
// be re-read to recover the role.
func TestUserIdentityTrustsMiddlewareAnonymousVerdict(t *testing.T) {
	d := newDelegate()
	c := identityContext(t, superAdminBFFToken(t))
	c.Set("user_id", "0")
	c.Set("user_role", models.Anonymous)

	id, role, err := d.UserIdentity(c)
	if err == nil {
		t.Fatalf("got id=%q role=%v, want an error", id, role)
	}
	if role != models.Anonymous {
		t.Fatalf("role=%v want Anonymous", role)
	}
}

func TestUserIdentityReturnsMiddlewareIdentity(t *testing.T) {
	d := newDelegate()
	c := identityContext(t, "")
	c.Set("user_id", "797")
	c.Set("user_role", models.Student)

	id, role, err := d.UserIdentity(c)
	if err != nil || id != "797" || role != models.Student {
		t.Fatalf("got id=%q role=%v err=%v", id, role, err)
	}
}

// On paths the middleware skips, a BFF token carries no Id and must not grant its role.
func TestUserIdentityRejectsBFFTokenInHeaderFallback(t *testing.T) {
	d := newDelegate()
	c := identityContext(t, superAdminBFFToken(t))

	id, role, err := d.UserIdentity(c)
	if err == nil {
		t.Fatalf("got id=%q role=%v, want an error", id, role)
	}
}

func TestUserIdentityAcceptsLegacyJWTInHeaderFallback(t *testing.T) {
	d := newDelegate()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, &models.UserClaims{Id: "42", Role: models.Teacher}).
		SignedString([]byte(testSigningKey))
	if err != nil {
		t.Fatal(err)
	}
	c := identityContext(t, token)

	id, role, err := d.UserIdentity(c)
	if err != nil || id != "42" || role != models.Teacher {
		t.Fatalf("got id=%q role=%v err=%v", id, role, err)
	}
}
