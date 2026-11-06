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
	maxBytes      int64
	uploaded      []byte
	importedTitle string
}

func (d *uploadPageDelegate) MaxSb3UploadBytes(string) (int64, error) { return d.maxBytes, nil }

func (d *uploadPageDelegate) UploadProjectSb3(_ string, _ string, data []byte) error {
	d.uploaded = data
	return nil
}

func (d *uploadPageDelegate) ImportProjectSb3(_ string, title string, _ string, data []byte) (models.ProjectPageHTTP, error) {
	d.uploaded, d.importedTitle = data, title
	return models.ProjectPageHTTP{ProjectPageID: "p1", Title: title}, nil
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

func postImport(t *testing.T, h *Handler, size int, title string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("title", title)
	part, err := mw.CreateFormFile("file", "game.sb3")
	if err != nil {
		t.Fatal(err)
	}
	part.Write(bytes.Repeat([]byte{'x'}, size))
	mw.Close()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/projectPage/import", &body)
	c.Request.Header.Set("Content-Type", mw.FormDataContentType())
	h.ImportProjectSb3(c)
	return w
}

func TestImportProjectSb3PassesTitle(t *testing.T) {
	pages := &uploadPageDelegate{maxBytes: 1 << 20}
	h := &Handler{authDelegate: uploadAuthDelegate{}, projectPageDelegate: pages}

	w := postImport(t, h, 1024, "Моя игра")
	if w.Code != http.StatusCreated || pages.importedTitle != "Моя игра" || len(pages.uploaded) != 1024 {
		t.Fatalf("status=%d title=%q uploaded=%d bytes", w.Code, pages.importedTitle, len(pages.uploaded))
	}
}

func TestImportProjectSb3RejectsOversizedFile(t *testing.T) {
	pages := &uploadPageDelegate{maxBytes: 1 << 20}
	h := &Handler{authDelegate: uploadAuthDelegate{}, projectPageDelegate: pages}

	w := postImport(t, h, (1<<20)+1, "x")
	if w.Code != http.StatusConflict || pages.uploaded != nil {
		t.Fatalf("status=%d uploaded=%d bytes, want 409 and nothing created", w.Code, len(pages.uploaded))
	}
}
