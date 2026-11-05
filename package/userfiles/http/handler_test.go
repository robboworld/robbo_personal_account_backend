package http

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles"
)

type fakeAuth struct{ auth.Delegate }

func (fakeAuth) UserIdentity(*gin.Context) (string, models.Role, error) {
	return "1", models.Student, nil
}
func (fakeAuth) UserAccess(models.Role, []models.Role, context.Context) error { return nil }

type nopSeeker struct{ *bytes.Reader }

func (nopSeeker) Close() error { return nil }

type fakeUC struct {
	userfiles.UseCase
	enabled   bool
	maxBytes  int64
	uploaded  int64
	started   chan struct{}
	afterBody func(ctx context.Context) error
}

func (u *fakeUC) Enabled() bool                        { return u.enabled }
func (u *fakeUC) MaxUploadBytes(string) (int64, error) { return u.maxBytes, nil }
func (u *fakeUC) Upload(ctx context.Context, _ string, name string, r io.Reader) (*models.UserFileDB, error) {
	if u.started != nil {
		close(u.started)
	}
	n, err := io.Copy(io.Discard, r)
	if err != nil {
		return nil, err
	}
	u.uploaded = n
	if u.afterBody != nil {
		if err := u.afterBody(ctx); err != nil {
			return nil, err
		}
	}
	return &models.UserFileDB{ID: "f1", OriginalName: name, SizeBytes: n, CreatedAt: time.Now()}, nil
}
func (u *fakeUC) Open(owner, id string) (*models.UserFileDB, io.ReadSeekCloser, error) {
	if id != "f1" {
		return nil, nil, userfiles.ErrFileNotFound
	}
	row := &models.UserFileDB{ID: "f1", OriginalName: "Домашка.pdf", Mime: "application/pdf", SHA256: strings.Repeat("a", 64), CreatedAt: time.Now()}
	return row, nopSeeker{bytes.NewReader([]byte("%PDF-1.7"))}, nil
}

func router(uc *fakeUC) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewUserFilesHandler(fakeAuth{}, uc)
	h.InitRoutes(r)
	return r
}

func upload(t *testing.T, r *gin.Engine, size int) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, _ := mw.CreateFormFile("file", "notes.txt")
	part.Write(bytes.Repeat([]byte("a"), size))
	mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/files", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestDisabledIs404(t *testing.T) {
	r := router(&fakeUC{})
	for _, req := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "/api/files", nil),
		httptest.NewRequest(http.MethodPost, "/api/files", nil),
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("%s %s: %d, want 404", req.Method, req.URL, w.Code)
		}
	}
}

func TestUploadStreamsTheFile(t *testing.T) {
	uc := &fakeUC{enabled: true, maxBytes: 1 << 20}
	w := upload(t, router(uc), 1000)
	if w.Code != http.StatusCreated || uc.uploaded != 1000 {
		t.Fatalf("status=%d uploaded=%d body=%s", w.Code, uc.uploaded, w.Body)
	}
}

func TestUploadRefusesOversizedBody(t *testing.T) {
	uc := &fakeUC{enabled: true, maxBytes: 1 << 20}
	w := upload(t, router(uc), 3<<20)
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "FILE_SIZE_EXCEEDED") || uc.uploaded != 0 {
		t.Fatalf("status=%d uploaded=%d body=%s", w.Code, uc.uploaded, w.Body)
	}
}

// cancelServer is a real server: only a real connection tells the handler the client left.
func cancelServer(t *testing.T, uc *fakeUC) (url string, status <-chan int) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	statuses := make(chan int, 1)
	r.Use(func(c *gin.Context) {
		c.Next()
		statuses <- c.Writer.Status()
	})
	h := NewUserFilesHandler(fakeAuth{}, uc)
	h.InitRoutes(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv.URL + "/api/files", statuses
}

func multipartBody(t *testing.T, size int) (string, []byte) {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, _ := mw.CreateFormFile("file", "notes.txt")
	part.Write(bytes.Repeat([]byte("a"), size))
	mw.Close()
	return mw.FormDataContentType(), body.Bytes()
}

func waitStatus(t *testing.T, status <-chan int, want int) {
	t.Helper()
	select {
	case got := <-status:
		if got != want {
			t.Fatalf("status=%d, want %d", got, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the handler did not notice the client left")
	}
}

func TestCancelMidBodyIsNotAServerError(t *testing.T) {
	started := make(chan struct{})
	url, status := cancelServer(t, &fakeUC{enabled: true, maxBytes: 1 << 20, started: started})
	contentType, body := multipartBody(t, 1000)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pr, pw := io.Pipe()
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, pr)
	req.Header.Set("Content-Type", contentType)
	req.ContentLength = int64(len(body))

	done := make(chan struct{})
	go func() {
		defer close(done)
		if resp, err := http.DefaultClient.Do(req); err == nil {
			resp.Body.Close()
		}
	}()
	pw.Write(body[:500])
	<-started
	cancel()
	// The transport waits for its body read to end, cancelled or not.
	pw.CloseWithError(context.Canceled)
	<-done
	waitStatus(t, status, statusClientClosed)
}

func TestCancelAfterTheBodyReachesTheUseCase(t *testing.T) {
	received := make(chan struct{})
	uc := &fakeUC{enabled: true, maxBytes: 1 << 20, afterBody: func(ctx context.Context) error {
		close(received)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			return nil
		}
	}}
	url, status := cancelServer(t, uc)
	contentType, body := multipartBody(t, 1000)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", contentType)

	done := make(chan struct{})
	go func() {
		defer close(done)
		if resp, err := http.DefaultClient.Do(req); err == nil {
			resp.Body.Close()
		}
	}()
	<-received
	cancel()
	<-done
	waitStatus(t, status, statusClientClosed)
}

func TestDownloadHeaders(t *testing.T) {
	r := router(&fakeUC{enabled: true})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/files/f1/download", nil))
	if w.Code != http.StatusOK || w.Body.String() != "%PDF-1.7" {
		t.Fatalf("status=%d body=%q", w.Code, w.Body)
	}
	h := w.Header()
	for k, want := range map[string]string{
		"Content-Type":            "application/pdf",
		"X-Content-Type-Options":  "nosniff",
		"Content-Security-Policy": "sandbox; default-src 'none'",
		"Cache-Control":           "private, no-store",
	} {
		if h.Get(k) != want {
			t.Errorf("%s = %q, want %q", k, h.Get(k), want)
		}
	}
	if cd := h.Get("Content-Disposition"); !strings.HasPrefix(cd, `attachment; filename="file.pdf"; filename*=UTF-8''`) {
		t.Errorf("Content-Disposition = %q", cd)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/files/other/download", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("someone else's file: %d", w.Code)
	}
}
