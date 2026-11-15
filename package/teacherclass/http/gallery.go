package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
	ufhttp "github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles/http"
)

func (h *Handler) GetGallery(c *gin.Context) {
	userID, role, ok := h.identity(c, studentRoles())
	if !ok {
		return
	}
	dto, err := h.uc.GetGallery(userID, role, c.Param("assignmentId"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, dto)
}

// AddGalleryFile shares a file of the viewer's "Мои файлы": {"fileId": "…"}.
func (h *Handler) AddGalleryFile(c *gin.Context) {
	userID, role, ok := h.identity(c, studentRoles())
	if !ok {
		return
	}
	var body struct {
		FileID string `json:"fileId"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.FileID) == "" {
		writeErr(c, teacherclass.ErrBadRequest)
		return
	}
	dto, err := h.uc.AddGalleryFile(userID, role, c.Param("assignmentId"), strings.TrimSpace(body.FileID))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, dto)
}

func (h *Handler) RemoveGalleryItem(c *gin.Context) {
	userID, role, ok := h.identity(c, studentRoles())
	if !ok {
		return
	}
	dto, err := h.uc.RemoveGalleryItem(userID, role, c.Param("assignmentId"), c.Param("itemId"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, dto)
}

func (h *Handler) DownloadGalleryFile(c *gin.Context) {
	userID, role, ok := h.identity(c, studentRoles())
	if !ok {
		return
	}
	row, rc, err := h.uc.OpenGalleryFile(userID, role, c.Param("assignmentId"), c.Param("itemId"))
	if err != nil {
		writeErr(c, err)
		return
	}
	defer rc.Close()
	ufhttp.ServeFile(c, row, rc)
}

// UpdateGalleryItem takes {"caption": "…"} and/or {"pinned": true}.
func (h *Handler) UpdateGalleryItem(c *gin.Context) {
	userID, role, ok := h.identity(c, teacherRoles())
	if !ok {
		return
	}
	var body struct {
		Caption *string `json:"caption"`
		Pinned  *bool   `json:"pinned"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, teacherclass.ErrBadRequest)
		return
	}
	dto, err := h.uc.UpdateGalleryItem(userID, role, c.Param("assignmentId"), c.Param("itemId"),
		teacherclass.GalleryItemUpdate{Caption: body.Caption, Pinned: body.Pinned})
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, dto)
}

// ReorderGallery takes {"itemIds": [...]} in the new order.
func (h *Handler) ReorderGallery(c *gin.Context) {
	userID, role, ok := h.identity(c, teacherRoles())
	if !ok {
		return
	}
	var body struct {
		ItemIDs []string `json:"itemIds"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, teacherclass.ErrBadRequest)
		return
	}
	dto, err := h.uc.ReorderGallery(userID, role, c.Param("assignmentId"), body.ItemIDs)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, dto)
}

func (h *Handler) CloseGallery(c *gin.Context)  { h.setGalleryClosed(c, true) }
func (h *Handler) ReopenGallery(c *gin.Context) { h.setGalleryClosed(c, false) }

func (h *Handler) setGalleryClosed(c *gin.Context, closed bool) {
	userID, role, ok := h.identity(c, teacherRoles())
	if !ok {
		return
	}
	dto, err := h.uc.SetGalleryClosed(userID, role, c.Param("assignmentId"), closed)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, dto)
}
