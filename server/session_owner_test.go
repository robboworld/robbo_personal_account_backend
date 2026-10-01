package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/licensing"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/oidc"
	"github.com/spf13/viper"
)

type fakeSessions struct {
	licensing.Gateway
	owners map[string]string // sid -> lms user id
}

func (f fakeSessions) GetActiveSession(sid string) (*models.UserSessionCore, error) {
	owner, ok := f.owners[sid]
	if !ok {
		return nil, errors.New("not found")
	}
	return &models.UserSessionCore{SessionKey: sid, LmsUserID: owner}, nil
}

func TestSessionOwnedBy(t *testing.T) {
	sessions := fakeSessions{owners: map[string]string{"sid-alice": "1"}}
	if !sessionOwnedBy(sessions, "sid-alice", "1") {
		t.Fatal("owner rejected")
	}
	for _, tc := range []struct{ sid, user string }{{"sid-alice", "2"}, {"", "1"}, {"sid-alice", ""}, {"missing", "1"}} {
		if sessionOwnedBy(sessions, tc.sid, tc.user) {
			t.Fatalf("sid=%q user=%q accepted", tc.sid, tc.user)
		}
	}
}

func TestApplyOidcSessionRejectsBorrowedSid(t *testing.T) {
	viper.Set("auth.access_signing_key", "0123456789abcdef0123456789abcdef")
	t.Cleanup(func() { viper.Set("auth.access_signing_key", "") })
	sessions := fakeSessions{owners: map[string]string{"sid-alice": "1"}}
	// Token claims user 99 (SuperAdmin) but reuses Alice's active sid.
	tok, err := oidc.IssueSessionToken("mallory", "99", "m@example.com", uint(models.SuperAdmin), "sid-alice")
	if err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.AddCookie(&http.Cookie{Name: oidc.SessionCookieName, Value: tok})
	if applyOidcSession(c, sessions) {
		t.Fatal("session with another user's sid accepted")
	}
}
