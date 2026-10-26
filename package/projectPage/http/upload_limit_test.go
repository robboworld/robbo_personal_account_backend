package http

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
)

type uploadAuthDelegate struct{ auth.Delegate }

func (uploadAuthDelegate) UserIdentity(*gin.Context) (string, models.Role, error) {
	return "1", models.Student, nil
}

func (uploadAuthDelegate) UserAccess(models.Role, []models.Role, context.Context) error {
	return nil
}

type uploadPageDelegate struct {
	projectPage.Delegate
	maxBytes int64
	uploaded []byte
}

func (d *uploadPageDelegate) MaxSb3UploadBytes(string) (int64, error) { return d.maxBytes, nil }

func (d *uploadPageDelegate) UploadProjectSb3(_ string, _ string, data []byte) error {
	d.uploaded = data
	return nil
}

func postSb3(t *testing.T, h *Handler, size int) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", "game.sb3")
	if err != nil {
		t.Fatal(err)
	}
	part.Write(bytes.Repeat([]byte{'x'}, size))
	mw.Close()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/projectPage/p1/upload", &body)
	c.Request.Header.Set("Content-Type", mw.FormDataContentType())
	c.Params = gin.Params{{Key: "projectPageId", Value: "p1"}}
	h.UploadProjectSb3(c)
	return w
}

func TestUploadProjectSb3RejectsOversizedBody(t *testing.T) {
	pages := &uploadPageDelegate{maxBytes: 1 << 20}
	h := &Handler{authDelegate: uploadAuthDelegate{}, projectPageDelegate: pages}

	// Larger than limit + multipart slack: cut off while streaming.
	w := postSb3(t, h, 3<<20)
	if w.Code != http.StatusConflict || pages.uploaded != nil {
		t.Fatalf("status=%d uploaded=%d bytes, want 409 and nothing saved", w.Code, len(pages.uploaded))
	}

	// Over the limit but within the slack: caught by the file size check.
	w = postSb3(t, h, (1<<20)+1)
	if w.Code != http.StatusConflict || pages.uploaded != nil {
		t.Fatalf("status=%d uploaded=%d bytes, want 409 and nothing saved", w.Code, len(pages.uploaded))
	}
}

func TestUploadProjectSb3AcceptsFileAtLimit(t *testing.T) {
	pages := &uploadPageDelegate{maxBytes: 1 << 20}
	h := &Handler{authDelegate: uploadAuthDelegate{}, projectPageDelegate: pages}

	w := postSb3(t, h, 1<<20)
	if w.Code != http.StatusOK || len(pages.uploaded) != 1<<20 {
		t.Fatalf("status=%d uploaded=%d bytes", w.Code, len(pages.uploaded))
	}
}
