package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
)

func TestErrorHandlingHomeworkLockedIs409WithCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	ErrorHandling(projectPage.ErrHomeworkLocked, c)

	var body struct{ Code string }
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusConflict || body.Code != "HOMEWORK_LOCKED" {
		t.Fatalf("status=%d code=%q, want 409 HOMEWORK_LOCKED", w.Code, body.Code)
	}
}
