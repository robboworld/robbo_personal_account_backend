package auth

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestIsKidAccountEmail(t *testing.T) {
	for email, want := range map[string]bool{
		"k-abc123@kids.robbo.invalid":      true,
		"  K-ABC@Kids.Robbo.Invalid ":      true,
		"parent@example.com":               false,
		"kids.robbo.invalid@example.com":   false,
		"k-abc@kids.robbo.invalid.evil.ru": false,
		"":                                 false,
	} {
		if got := IsKidAccountEmail(email); got != want {
			t.Errorf("IsKidAccountEmail(%q) = %v, want %v", email, got, want)
		}
	}
}

func TestKidAccountSessionMark(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if IsKidAccountSession(c) {
		t.Fatal("unmarked request taken for a kid session")
	}
	MarkKidAccountSession(c, "parent@example.com")
	if IsKidAccountSession(c) {
		t.Fatal("a full account's session marked as a kid one")
	}
	MarkKidAccountSession(c, "k-abc@kids.robbo.invalid")
	if !IsKidAccountSession(c) {
		t.Fatal("kid session not marked")
	}
	if IsKidAccountSession(nil) {
		t.Fatal("nil context taken for a kid session")
	}
}

func TestAbortIfKidAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	MarkKidAccountSession(c, "parent@example.com")
	if AbortIfKidAccount(c) || c.IsAborted() {
		t.Fatal("a full account's request aborted")
	}
	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	MarkKidAccountSession(c, "k-abc@kids.robbo.invalid")
	if !AbortIfKidAccount(c) || !c.IsAborted() {
		t.Fatal("a kid account's request went on")
	}
	if rec.Code != 403 || rec.Body.String() != `{"error":"kid_account"}` {
		t.Fatalf("want 403 kid_account, got %d %s", rec.Code, rec.Body.String())
	}
}
