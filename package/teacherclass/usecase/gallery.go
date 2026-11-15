package usecase

import (
	"errors"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

// The lesson gallery: the files a class shares on one assignment. Anyone in the class adds files
// from their "Мои файлы" (no copy, the file stays in their quota); everyone in the class sees and
// downloads all of them. A student removes only what they shared; a teacher of the class removes
// any file, captions, pins and orders them, and closes the gallery to students.

const (
	defaultMaxGalleryFilesPerUser = 10
	maxGalleryCaptionRunes        = 200
)

// maxGalleryFilesPerUser is how many files one student may share in one gallery.
func maxGalleryFilesPerUser() int {
	if n := viper.GetInt("classes.maxGalleryFilesPerUser"); n > 0 {
		return n
	}
	return defaultMaxGalleryFilesPerUser
}

// galleryViewer is who looks at a gallery: the assignment, its class, and whether they teach it.
type galleryViewer struct {
	id      string
	role    models.Role
	a       *models.ClassAssignmentDB
	inv     *models.ClassInviteDB
	teaches bool
}

// galleryAccess lets in the class's teachers and the members of its roster. Files cross
// accounts here, so the gallery needs what file assignments need: a required antivirus scan.
func (u *UseCaseImpl) galleryAccess(viewerID string, role models.Role, assignmentID string) (*galleryViewer, error) {
	a, err := u.gw.GetAssignment(assignmentID)
	if err != nil {
		return nil, teacherclass.ErrNotFound
	}
	inv, err := u.gw.GetInviteByCourseCohort(a.EdxCourseID, a.EdxCohortID)
	if err != nil {
		return nil, teacherclass.ErrNotFound
	}
	teaches, err := u.canTeach(inv, viewerID, role)
	if err != nil {
		return nil, err
	}
	if !teaches {
		member, err := u.isClassMember(viewerID, inv)
		if err != nil {
			return nil, err
		}
		if !member {
			return nil, teacherclass.ErrForbidden
		}
	}
	if !u.FileAssignmentsEnabled() {
		return nil, teacherclass.ErrFileAssignmentsDisabled
	}
	return &galleryViewer{id: viewerID, role: role, a: a, inv: inv, teaches: teaches}, nil
}

// galleryManager is galleryAccess for what only the class's teachers do.
func (u *UseCaseImpl) galleryManager(teacherID string, role models.Role, assignmentID string) (*galleryViewer, error) {
	v, err := u.galleryAccess(teacherID, role, assignmentID)
	if err != nil {
		return nil, err
	}
	if !v.teaches {
		return nil, teacherclass.ErrForbidden
	}
	return v, nil
}

// canUpload: students add files while the gallery is open; teachers always, as long as the
// class is not archived.
func (v *galleryViewer) canUpload() bool {
	if v.inv.ArchivedAt != nil {
		return false
	}
	return v.teaches || v.a.GalleryClosedAt == nil
}

func (v *galleryViewer) canRemove(it *models.ClassGalleryItemDB) bool {
	return v.teaches || it.UploaderUserID == v.id
}

// galleryItem is an item of the viewer's assignment; one of another gallery reads as missing.
func (u *UseCaseImpl) galleryItem(v *galleryViewer, itemID string) (*models.ClassGalleryItemDB, error) {
	it, err := u.gw.GetGalleryItem(itemID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, teacherclass.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if it.AssignmentID != v.a.ID {
		return nil, teacherclass.ErrNotFound
	}
	return it, nil
}

func (u *UseCaseImpl) galleryDTO(v *galleryViewer) (*teacherclass.GalleryDTO, error) {
	items, err := u.gw.ListGalleryItems(v.a.ID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(items))
	for i := range items {
		ids = append(ids, items[i].UserFileID)
	}
	files, err := u.gw.GetUserFiles(ids)
	if err != nil {
		return nil, err
	}
	names := map[string]string{}
	out := &teacherclass.GalleryDTO{
		AssignmentID:    v.a.ID,
		Title:           v.a.Title,
		Items:           make([]teacherclass.GalleryItemDTO, 0, len(items)),
		Closed:          v.a.GalleryClosedAt != nil,
		CanManage:       v.teaches,
		CanUpload:       v.canUpload(),
		MaxFilesPerUser: maxGalleryFilesPerUser(),
	}
	for i := range items {
		it := &items[i]
		f, ok := files[it.UserFileID]
		if !ok {
			// Deleted between the two reads.
			continue
		}
		name, seen := names[it.UploaderUserID]
		if !seen {
			name = ownerDisplayName(it.UploaderUserID)
			names[it.UploaderUserID] = name
		}
		out.Items = append(out.Items, teacherclass.GalleryItemDTO{
			ID:           it.ID,
			File:         teacherclass.FileRef(&f),
			UploaderID:   it.UploaderUserID,
			UploaderName: name,
			Caption:      it.Caption,
			Pinned:       it.Pinned,
			CreatedAt:    it.CreatedAt,
			Mine:         it.UploaderUserID == v.id,
			CanRemove:    v.canRemove(it),
		})
	}
	return out, nil
}

func (u *UseCaseImpl) GetGallery(viewerID string, role models.Role, assignmentID string) (*teacherclass.GalleryDTO, error) {
	v, err := u.galleryAccess(viewerID, role, assignmentID)
	if err != nil {
		return nil, err
	}
	return u.galleryDTO(v)
}

func (u *UseCaseImpl) AddGalleryFile(viewerID string, role models.Role, assignmentID, fileID string) (*teacherclass.GalleryDTO, error) {
	v, err := u.galleryAccess(viewerID, role, assignmentID)
	if err != nil {
		return nil, err
	}
	if !v.canUpload() {
		return nil, teacherclass.ErrGalleryClosed
	}
	// Only the viewer's own file; one uploaded before the scanner is scanned now.
	f, err := u.files.EnsureScanned(viewerID, strings.TrimSpace(fileID))
	if err != nil {
		return nil, fileErr(err)
	}
	limit := maxGalleryFilesPerUser()
	if v.teaches {
		limit = 0
	}
	item := &models.ClassGalleryItemDB{AssignmentID: v.a.ID, UserFileID: f.ID, UploaderUserID: viewerID}
	created, err := u.gw.AddGalleryItem(item, limit)
	if err != nil {
		return nil, err
	}
	if created {
		u.recordGalleryAction(v, models.HistoryGalleryFileAdded, models.ClassHistoryEventDB{Title: v.a.Title, Detail: f.OriginalName})
	}
	return u.galleryDTO(v)
}

func (u *UseCaseImpl) RemoveGalleryItem(viewerID string, role models.Role, assignmentID, itemID string) (*teacherclass.GalleryDTO, error) {
	v, err := u.galleryAccess(viewerID, role, assignmentID)
	if err != nil {
		return nil, err
	}
	it, err := u.galleryItem(v, itemID)
	if err != nil {
		return nil, err
	}
	if !v.canRemove(it) {
		return nil, teacherclass.ErrForbidden
	}
	var name string
	if files, err := u.gw.GetUserFiles([]string{it.UserFileID}); err == nil {
		name = files[it.UserFileID].OriginalName
	}
	// The file stays in its owner's "Мои файлы": only the link goes.
	if err := u.gw.DeleteGalleryItem(it.ID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, teacherclass.ErrNotFound
		}
		return nil, err
	}
	e := models.ClassHistoryEventDB{Title: v.a.Title, Detail: name}
	if it.UploaderUserID != viewerID {
		e.TargetUserID = it.UploaderUserID
	}
	u.recordGalleryAction(v, models.HistoryGalleryFileRemoved, e)
	return u.galleryDTO(v)
}

func (u *UseCaseImpl) OpenGalleryFile(viewerID string, role models.Role, assignmentID, itemID string) (*models.UserFileDB, io.ReadSeekCloser, error) {
	v, err := u.galleryAccess(viewerID, role, assignmentID)
	if err != nil {
		return nil, nil, err
	}
	it, err := u.galleryItem(v, itemID)
	if err != nil {
		return nil, nil, err
	}
	row, rc, err := u.files.Open(it.UploaderUserID, it.UserFileID)
	if err != nil {
		return nil, nil, fileErr(err)
	}
	return row, rc, nil
}

func galleryCaption(s string) (string, error) {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > maxGalleryCaptionRunes {
		return "", teacherclass.ErrBadRequest
	}
	return s, nil
}

func (u *UseCaseImpl) UpdateGalleryItem(teacherID string, role models.Role, assignmentID, itemID string, in teacherclass.GalleryItemUpdate) (*teacherclass.GalleryDTO, error) {
	if in.Caption == nil && in.Pinned == nil {
		return nil, teacherclass.ErrBadRequest
	}
	if in.Caption != nil {
		c, err := galleryCaption(*in.Caption)
		if err != nil {
			return nil, err
		}
		in.Caption = &c
	}
	v, err := u.galleryManager(teacherID, role, assignmentID)
	if err != nil {
		return nil, err
	}
	it, err := u.galleryItem(v, itemID)
	if err != nil {
		return nil, err
	}
	if err := u.gw.UpdateGalleryItem(it.ID, in); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, teacherclass.ErrNotFound
		}
		return nil, err
	}
	return u.galleryDTO(v)
}

func (u *UseCaseImpl) ReorderGallery(teacherID string, role models.Role, assignmentID string, itemIDs []string) (*teacherclass.GalleryDTO, error) {
	if len(itemIDs) == 0 {
		return nil, teacherclass.ErrBadRequest
	}
	seen := make(map[string]bool, len(itemIDs))
	for _, id := range itemIDs {
		if id == "" || seen[id] {
			return nil, teacherclass.ErrBadRequest
		}
		seen[id] = true
	}
	v, err := u.galleryManager(teacherID, role, assignmentID)
	if err != nil {
		return nil, err
	}
	if err := u.gw.ReorderGallery(v.a.ID, itemIDs); err != nil {
		return nil, err
	}
	return u.galleryDTO(v)
}

func (u *UseCaseImpl) SetGalleryClosed(teacherID string, role models.Role, assignmentID string, closed bool) (*teacherclass.GalleryDTO, error) {
	v, err := u.galleryManager(teacherID, role, assignmentID)
	if err != nil {
		return nil, err
	}
	if (v.a.GalleryClosedAt != nil) == closed {
		return u.galleryDTO(v)
	}
	if err := u.gw.SetGalleryClosed(v.a.ID, closed); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, teacherclass.ErrNotFound
		}
		return nil, err
	}
	a, err := u.gw.GetAssignment(v.a.ID)
	if err != nil {
		return nil, teacherclass.ErrNotFound
	}
	v.a = a
	action := models.HistoryGalleryReopened
	if closed {
		action = models.HistoryGalleryClosed
	}
	u.recordGalleryAction(v, action, models.ClassHistoryEventDB{Title: a.Title})
	return u.galleryDTO(v)
}

// recordGalleryAction puts a gallery change into the class history as a teacher's or a
// student's action.
func (u *UseCaseImpl) recordGalleryAction(v *galleryViewer, action string, e models.ClassHistoryEventDB) {
	if v.teaches {
		u.recordTeacherAction(v.inv, v.id, v.role, action, e)
		return
	}
	u.recordStudentAction(v.inv, v.id, action, e)
}

// fillGalleryCounts adds the size of each lesson gallery to the assignments of a class.
func (u *UseCaseImpl) fillGalleryCounts(dtos []teacherclass.AssignmentDTO, list []models.ClassAssignmentDB) error {
	if !u.FileAssignmentsEnabled() || len(list) == 0 {
		return nil
	}
	ids := make([]string, 0, len(list))
	for i := range list {
		ids = append(ids, list[i].ID)
	}
	counts, err := u.gw.CountGalleryItems(ids)
	if err != nil {
		return err
	}
	for i := range list {
		dtos[i].GalleryEnabled = true
		dtos[i].GalleryCount = counts[list[i].ID]
		dtos[i].GalleryClosed = list[i].GalleryClosedAt != nil
	}
	return nil
}
