package http

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/storage"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles"
	"github.com/spf13/viper"
)

const (
	// Covers the multipart headers and boundaries around the file.
	multipartSlackBytes  = 1 << 20
	defaultUploadTimeout = 10 * time.Minute
)

type Handler struct {
	authDelegate auth.Delegate
	uc           userfiles.UseCase
}

func NewUserFilesHandler(authDelegate auth.Delegate, uc userfiles.UseCase) Handler {
	return Handler{authDelegate: authDelegate, uc: uc}
}

// Every real account has its own files; parents and listeners too.
func fileRoles() []models.Role {
	return []models.Role{models.Student, models.Teacher, models.Parent, models.FreeListener, models.UnitAdmin, models.SuperAdmin}
}

func (h *Handler) InitRoutes(router *gin.Engine) {
	api := router.Group("/api/files")
	{
		api.GET("", h.List)
		api.POST("", h.Upload)
		api.GET("/:fileId/download", h.Download)
		api.DELETE("/:fileId", h.Delete)
	}
}

func (h *Handler) identity(c *gin.Context) (string, bool) {
	if h.uc == nil || !h.uc.Enabled() {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
		return "", false
	}
	userID, role, err := h.authDelegate.UserIdentity(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return "", false
	}
	if err := h.authDelegate.UserAccess(role, fileRoles(), c); err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return "", false
	}
	return userID, true
}

func (h *Handler) List(c *gin.Context) {
	userID, ok := h.identity(c)
	if !ok {
		return
	}
	rows, err := h.uc.List(userID)
	if err != nil {
		errorHandling(err, c)
		return
	}
	out := make([]models.UserFileHTTP, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.ToHTTP())
	}
	c.JSON(http.StatusOK, gin.H{"files": out})
}

func uploadTimeout() time.Duration {
	if d := viper.GetDuration("userFiles.uploadTimeout"); d > 0 {
		return d
	}
	return defaultUploadTimeout
}

// Upload streams the one multipart part "file" to the blob store: the body is never held in
// memory, and an oversized one is cut off while it streams in.
func (h *Handler) Upload(c *gin.Context) {
	userID, ok := h.identity(c)
	if !ok {
		return
	}
	maxBytes, err := h.uc.MaxUploadBytes(userID)
	if err != nil {
		errorHandling(err, c)
		return
	}
	limit := maxBytes + multipartSlackBytes
	if c.Request.ContentLength > limit {
		// Answer before reading: the browser then shows this error, not a reset connection.
		c.Header("Connection", "close")
		errorHandling(userfiles.ErrFileSizeExceeded, c)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
	// server.ReadTimeout (120s) is too short for 50 MB on a slow line; give this route longer.
	if err := http.NewResponseController(c.Writer).SetReadDeadline(time.Now().Add(uploadTimeout())); err != nil {
		log.Printf("userfiles: extend the upload read deadline: %v", err)
	}
	mr, err := c.Request.MultipartReader()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid multipart form"})
		return
	}
	part, err := mr.NextPart()
	if err != nil {
		if isTooLarge(err) {
			errorHandling(userfiles.ErrFileSizeExceeded, c)
			return
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer part.Close()
	if part.FormName() != "file" || part.FileName() == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	row, err := h.uc.Upload(userID, part.FileName(), part)
	if err != nil {
		if isTooLarge(err) {
			err = userfiles.ErrFileSizeExceeded
		}
		errorHandling(err, c)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"file": row.ToHTTP()})
}

func isTooLarge(err error) bool {
	var tooLarge *http.MaxBytesError
	return errors.As(err, &tooLarge)
}

// Download serves the file as an attachment that the browser never renders or sniffs.
// http.ServeContent answers Range and If-None-Match requests.
func (h *Handler) Download(c *gin.Context) {
	userID, ok := h.identity(c)
	if !ok {
		return
	}
	row, rc, err := h.uc.Open(userID, c.Param("fileId"))
	if err != nil {
		errorHandling(err, c)
		return
	}
	defer rc.Close()
	ServeFile(c, row, rc)
}

// SetDownloadHeaders makes the browser save the response as a file and never render or sniff
// it: a user's file is served from the ЛК origin.
func SetDownloadHeaders(w http.ResponseWriter, mime, name, etag string) {
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Disposition", attachmentDisposition(name))
	if etag != "" {
		w.Header().Set("ETag", `"`+etag+`"`)
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
	w.Header().Set("Referrer-Policy", "no-referrer")
}

// ServeFile sends a stored file with SetDownloadHeaders; http.ServeContent answers Range and
// If-None-Match requests.
func ServeFile(c *gin.Context, row *models.UserFileDB, rc io.ReadSeeker) {
	SetDownloadHeaders(c.Writer, row.Mime, row.OriginalName, strings.TrimSpace(row.SHA256))
	http.ServeContent(c.Writer, c.Request, "", row.CreatedAt, rc)
}

func (h *Handler) Delete(c *gin.Context) {
	userID, ok := h.identity(c)
	if !ok {
		return
	}
	if err := h.uc.Delete(userID, c.Param("fileId")); err != nil {
		errorHandling(err, c)
		return
	}
	c.Status(http.StatusNoContent)
}

// attachmentDisposition: an ASCII fallback name for old clients plus the real name (RFC 5987).
func attachmentDisposition(name string) string {
	ext := filepath.Ext(name)
	var b strings.Builder
	for _, r := range strings.TrimSuffix(name, ext) {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	ascii := b.String()
	if ascii == "" {
		ascii = "file"
	}
	return fmt.Sprintf(`attachment; filename="%s%s"; filename*=UTF-8''%s`, ascii, ext, url.PathEscape(name))
}

func errorHandling(err error, c *gin.Context) {
	coded := func(status int, code string) {
		c.AbortWithStatusJSON(status, gin.H{"error": code, "code": code})
	}
	switch {
	case errors.Is(err, userfiles.ErrDisabled), errors.Is(err, userfiles.ErrFileNotFound):
		coded(http.StatusNotFound, "FILE_NOT_FOUND")
	case errors.Is(err, userfiles.ErrFileTypeNotAllowed):
		coded(http.StatusUnsupportedMediaType, "FILE_TYPE_NOT_ALLOWED")
	case errors.Is(err, userfiles.ErrFileContentNotAllowed):
		coded(http.StatusUnsupportedMediaType, "FILE_CONTENT_NOT_ALLOWED")
	case errors.Is(err, userfiles.ErrTooManyUploads):
		coded(http.StatusTooManyRequests, "TOO_MANY_UPLOADS")
	case errors.Is(err, userfiles.ErrUploadRateLimited):
		coded(http.StatusTooManyRequests, "UPLOAD_RATE_LIMITED")
	case errors.Is(err, userfiles.ErrStorageUnavailable):
		coded(http.StatusInsufficientStorage, "STORAGE_UNAVAILABLE")
	case errors.Is(err, userfiles.ErrFileSizeExceeded):
		coded(http.StatusConflict, "FILE_SIZE_EXCEEDED")
	case errors.Is(err, userfiles.ErrFileCountExceeded):
		coded(http.StatusConflict, "FILE_COUNT_EXCEEDED")
	case errors.Is(err, storage.ErrStorageQuotaExceeded):
		coded(http.StatusConflict, "STORAGE_QUOTA_EXCEEDED")
	default:
		log.Printf("userfiles: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
