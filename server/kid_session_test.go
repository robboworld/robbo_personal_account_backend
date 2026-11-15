package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/oidc"
	"github.com/spf13/viper"
)

// The middleware marks a kid account's session by the email in the signed session token, for
// the cookie and for the Bearer header alike; a full account's session stays unmarked.
func TestApplyOidcSessionMarksKidAccount(t *testing.T) {
	viper.Set("auth.access_signing_key", "0123456789abcdef0123456789abcdef")
	t.Cleanup(func() { viper.Set("auth.access_signing_key", "") })
	sessions := fakeSessions{owners: map[string]string{"sid-kid": "5", "sid-full": "6"}}
	gin.SetMode(gin.TestMode)

	for _, tc := range []struct {
		name, sid, user, email string
		bearer, wantKid        bool
	}{
		{"kid cookie", "sid-kid", "5", "k-abc@kids.robbo.invalid", false, true},
		{"kid bearer", "sid-kid", "5", "k-abc@kids.robbo.invalid", true, true},
		{"full cookie", "sid-full", "6", "parent@example.com", false, false},
		{"full bearer", "sid-full", "6", "parent@example.com", true, false},
	} {
		tok, err := oidc.IssueSessionToken("u"+tc.user, tc.user, tc.email, uint(models.Student), tc.sid)
		if err != nil {
			t.Fatal(err)
		}
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/query", nil)
		if tc.bearer {
			c.Request.Header.Set("Authorization", "Bearer "+tok)
		} else {
			c.Request.AddCookie(&http.Cookie{Name: oidc.SessionCookieName, Value: tok})
		}
		if !applyOidcSession(c, sessions) {
			t.Fatalf("%s: session rejected", tc.name)
		}
		if got := auth.IsKidAccountSession(c); got != tc.wantKid {
			t.Errorf("%s: kid session = %v, want %v", tc.name, got, tc.wantKid)
		}
	}
}
