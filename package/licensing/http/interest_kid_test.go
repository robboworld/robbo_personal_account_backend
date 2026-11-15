package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/licensing"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

type kidTestAuth struct{ auth.Delegate }

func (kidTestAuth) UserIdentity(*gin.Context) (string, models.Role, error) {
	return "5", models.Student, nil
}

type kidTestLicensing struct {
	licensing.Delegate
	saved bool
}

func (l *kidTestLicensing) SaveFeatureInterest(models.FeatureInterestCore) error {
	l.saved = true
	return nil
}

// postInterest sends a valid survey answer from a session with the given email.
func postInterest(t *testing.T, email string) (*httptest.ResponseRecorder, *kidTestLicensing) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		auth.MarkKidAccountSession(c, email)
		c.Next()
	})
	l := &kidTestLicensing{}
	h := &Handler{authDelegate: kidTestAuth{}, licensingDelegate: l}
	h.InitLicensingRoutes(router)
	body := `{"role":"student","feature":"robots","org":"School 1","pain":"text","notify":true}`
	req := httptest.NewRequest(http.MethodPost, "/licensing/interest", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec, l
}

func TestSaveFeatureInterestRefusesKidAccount(t *testing.T) {
	rec, l := postInterest(t, "k-abc@kids.robbo.invalid")
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "kid_account") {
		t.Fatalf("want 403 kid_account, got %d %s", rec.Code, rec.Body.String())
	}
	if l.saved {
		t.Fatal("a kid account's answer was saved")
	}
}

func TestSaveFeatureInterestKeepsFullAccount(t *testing.T) {
	rec, l := postInterest(t, "parent@example.com")
	if rec.Code != http.StatusOK || !l.saved {
		t.Fatalf("full account: want 200 and a saved answer, got %d saved=%v", rec.Code, l.saved)
	}
}
