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

func TestWriteErrCoTeacherStatuses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := map[error]int{
		teacherclass.ErrNotATeacher:         http.StatusNotFound,
		teacherclass.ErrBadCoTeacherTerm:    http.StatusBadRequest,
		teacherclass.ErrAlreadyCoTeacher:    http.StatusConflict,
		teacherclass.ErrTooManyCoTeachers:   http.StatusConflict,
		teacherclass.ErrCoTeacherCannotJoin: http.StatusForbidden,
	}
	for err, want := range cases {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/teacher/classes/c/co-teachers", nil)
		writeErr(c, err)
		if w.Code != want || !strings.Contains(w.Body.String(), err.Error()) {
			t.Errorf("%v: status=%d body=%s", err, w.Code, w.Body.String())
		}
	}
}

func TestSubmissionNoteBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	read := func(body string) (string, bool, int) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		note, ok := submissionNoteBody(c)
		return note, ok, w.Code
	}
	if note, ok, _ := read(""); !ok || note != "" {
		t.Fatalf("empty body: %q, %v", note, ok)
	}
	if note, ok, _ := read(`{"note":"готово"}`); !ok || note != "готово" {
		t.Fatalf("with a note: %q, %v", note, ok)
	}
	if _, ok, code := read(`{"note":`); ok || code != http.StatusBadRequest {
		t.Fatalf("broken body: ok=%v code=%d", ok, code)
	}
}

func TestNoteEditBodyNeedsTheNote(t *testing.T) {
	gin.SetMode(gin.TestMode)
	read := func(body string) (string, bool) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body))
		return noteEditBody(c)
	}
	for _, bad := range []string{"", "{}", `{"note":`} {
		if _, ok := read(bad); ok {
			t.Fatalf("body %q must not clear the note", bad)
		}
	}
	if note, ok := read(`{"note":""}`); !ok || note != "" {
		t.Fatalf("explicit empty note: %q, %v", note, ok)
	}
}
