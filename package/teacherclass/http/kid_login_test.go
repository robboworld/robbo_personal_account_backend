package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
)

func TestKeyedLimiterCountsPerKey(t *testing.T) {
	l := newKeyedLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		if !l.allow("class-a") {
			t.Fatalf("try %d refused", i+1)
		}
	}
	if l.allow("class-a") {
		t.Fatal("fourth try in a minute must be refused")
	}
	if !l.allow("class-b") {
		t.Fatal("another class has its own count")
	}
}

func TestKeyedLimiterWindowSlides(t *testing.T) {
	l := newKeyedLimiter(1, 20*time.Millisecond)
	if !l.allow("ip") || l.allow("ip") {
		t.Fatal("limit 1")
	}
	time.Sleep(30 * time.Millisecond)
	if !l.allow("ip") {
		t.Fatal("the window passed")
	}
}

func TestWriteKidErrStatuses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := map[error]int{
		teacherclass.ErrBadKidSecret:       http.StatusForbidden,
		teacherclass.ErrKidLoginClosed:     http.StatusForbidden,
		teacherclass.ErrKidLocked:          http.StatusForbidden,
		teacherclass.ErrKidConsentRequired: http.StatusForbidden,
		teacherclass.ErrLmsLinkedClass:     http.StatusConflict,
		teacherclass.ErrTooManyKids:        http.StatusConflict,
		teacherclass.ErrKidAccount:         http.StatusConflict, // JoinClass goes through writeErr
		teacherclass.ErrNotFound:           http.StatusNotFound,
		teacherclass.ErrBadRequest:         http.StatusBadRequest,
	}
	for err, want := range cases {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
		writeKidErr(c, err)
		if w.Code != want {
			t.Errorf("%v: status %d, want %d", err, w.Code, want)
		}
	}
}
