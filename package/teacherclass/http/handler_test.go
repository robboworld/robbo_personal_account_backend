package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
)

func TestWriteErrHidesInternalDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/teacher/classes", nil)

	writeErr(c, errors.New(`Post "https://edx-test.ru/oauth2/access_token": dial tcp: server misbehaving`))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d", w.Code)
	}
	if strings.Contains(w.Body.String(), "edx-test.ru") {
		t.Fatalf("internal detail leaked: %s", w.Body.String())
	}
}

func TestWriteErrKeepsDomainErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/join", nil)

	writeErr(c, teacherclass.ErrAlreadyIn)

	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "already in class") {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
