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

// fullAccountLicensing records which license operation reached the use case.
type fullAccountLicensing struct {
	licensing.Delegate
	called string
}

func (l *fullAccountLicensing) ListMyLicenses(string) ([]*models.LicenseCore, error) {
	l.called = "ListMyLicenses"
	return nil, nil
}

func (l *fullAccountLicensing) GetLicense(string, string) (*models.LicenseCore, error) {
	l.called = "GetLicense"
	return &models.LicenseCore{}, nil
}

func (l *fullAccountLicensing) RevokeSeat(string, string, string) error {
	l.called = "RevokeSeat"
	return nil
}

func (l *fullAccountLicensing) ConfirmDeviceLink(string, string, string) (*models.DeviceLinkSessionCore, error) {
	l.called = "ConfirmDeviceLink"
	return &models.DeviceLinkSessionCore{}, nil
}

var fullAccountLicenseRoutes = []struct{ method, path, body string }{
	{http.MethodGet, "/licensing/mine", ""},
	{http.MethodGet, "/licensing/lic-1", ""},
	{http.MethodDelete, "/licensing/lic-1/seats/seat-1", ""},
	{http.MethodPost, "/licensing/device/link/confirm", `{"userCode":"ABCD-EFGH","licenseId":"lic-1"}`},
}

func callLicenseRoute(t *testing.T, email, method, path, body string) (*httptest.ResponseRecorder, *fullAccountLicensing) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		auth.MarkKidAccountSession(c, email)
		c.Next()
	})
	l := &fullAccountLicensing{}
	h := &Handler{authDelegate: kidTestAuth{}, licensingDelegate: l}
	h.InitLicensingRoutes(router)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec, l
}

// A kid account has no licenses (architecture/ADR_CLASS_KID_LOGIN.md): the SPA locks the
// section, and a direct request gets 403 kid_account without reaching the use case.
func TestLicenseRoutesRefuseKidAccount(t *testing.T) {
	for _, r := range fullAccountLicenseRoutes {
		rec, l := callLicenseRoute(t, "k-abc@kids.robbo.invalid", r.method, r.path, r.body)
		if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "kid_account") {
			t.Errorf("%s %s: want 403 kid_account, got %d %s", r.method, r.path, rec.Code, rec.Body.String())
		}
		if l.called != "" {
			t.Errorf("%s %s: kid account reached %s", r.method, r.path, l.called)
		}
	}
}

func TestLicenseRoutesKeepFullAccount(t *testing.T) {
	for _, r := range fullAccountLicenseRoutes {
		rec, l := callLicenseRoute(t, "parent@example.com", r.method, r.path, r.body)
		if rec.Code != http.StatusOK || l.called == "" {
			t.Errorf("%s %s: full account: want 200 via the use case, got %d %s", r.method, r.path, rec.Code, rec.Body.String())
		}
	}
}
