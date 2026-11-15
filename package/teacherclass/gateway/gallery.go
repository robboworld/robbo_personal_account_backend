package gateway

import (
	"errors"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
	"gorm.io/gorm"
)

// liveGalleryItems are the items whose file is still in its owner's "Мои файлы".
const liveGalleryItems = `
	FROM class_gallery_items gi
	JOIN user_files f ON f.id = gi.user_file_id AND f.deleted_at IS NULL`

func (g *GatewayImpl) ListGalleryItems(assignmentID string) ([]models.ClassGalleryItemDB, error) {
	var list []models.ClassGalleryItemDB
	err := g.db.Raw(`SELECT gi.*`+liveGalleryItems+`
		WHERE gi.assignment_id = ?
		ORDER BY gi.pinned DESC, gi.position, gi.created_at, gi.id`, assignmentID).
		Scan(&list).Error
	return list, err
}

func (g *GatewayImpl) CountGalleryItems(assignmentIDs []string) (map[string]int, error) {
	out := map[string]int{}
	if len(assignmentIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		AssignmentID string `gorm:"column:assignment_id"`
		N            int    `gorm:"column:n"`
	}
	err := g.db.Raw(`SELECT gi.assignment_id, count(*) AS n`+liveGalleryItems+`
		WHERE gi.assignment_id IN ?
		GROUP BY gi.assignment_id`, assignmentIDs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.AssignmentID] = r.N
	}
	return out, nil
}

func (g *GatewayImpl) GetGalleryItem(id string) (*models.ClassGalleryItemDB, error) {
	var it models.ClassGalleryItemDB
	err := g.db.Raw(`SELECT gi.*`+liveGalleryItems+` WHERE gi.id = ?`, id).Scan(&it).Error
	if err != nil {
		return nil, err
	}
	if it.ID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	return &it, nil
}

func (g *GatewayImpl) AddGalleryItem(item *models.ClassGalleryItemDB, maxPerUploader int) (bool, error) {
	created := false
	err := g.db.Transaction(func(tx *gorm.DB) error {
		// One lock per gallery: the count, the next position and the insert are one step for
		// parallel uploads, and two uploads never get the same position.
		if err := tx.Exec(`SELECT pg_advisory_xact_lock(hashtext(?))`, "gallery:"+item.AssignmentID).Error; err != nil {
			return err
		}
		var existing models.ClassGalleryItemDB
		err := tx.Where("assignment_id = ? AND user_file_id = ?", item.AssignmentID, item.UserFileID).
			First(&existing).Error
		if err == nil {
			*item = existing
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if maxPerUploader > 0 {
			var n int64
			if err := tx.Raw(`SELECT count(*)`+liveGalleryItems+`
				WHERE gi.assignment_id = ? AND gi.uploader_user_id = ?`,
				item.AssignmentID, item.UploaderUserID).Scan(&n).Error; err != nil {
				return err
			}
			if n >= int64(maxPerUploader) {
				return teacherclass.ErrTooManyGalleryFiles
			}
		}
		var last int
		if err := tx.Raw(`SELECT COALESCE(max(position), 0) FROM class_gallery_items WHERE assignment_id = ?`,
			item.AssignmentID).Scan(&last).Error; err != nil {
			return err
		}
		item.Position = last + 1
		if item.CreatedAt.IsZero() {
			item.CreatedAt = time.Now().UTC()
		}
		if err := tx.Create(item).Error; err != nil {
			return err
		}
		created = true
		return nil
	})
	return created, err
}

func (g *GatewayImpl) DeleteGalleryItem(id string) error {
	res := g.db.Where("id = ?", id).Delete(&models.ClassGalleryItemDB{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (g *GatewayImpl) UpdateGalleryItem(id string, in teacherclass.GalleryItemUpdate) error {
	fields := map[string]any{}
	if in.Caption != nil {
		fields["caption"] = *in.Caption
	}
	if in.Pinned != nil {
		fields["pinned"] = *in.Pinned
	}
	if len(fields) == 0 {
		return nil
	}
	res := g.db.Model(&models.ClassGalleryItemDB{}).Where("id = ?", id).Updates(fields)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (g *GatewayImpl) ReorderGallery(assignmentID string, itemIDs []string) error {
	return g.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`SELECT pg_advisory_xact_lock(hashtext(?))`, "gallery:"+assignmentID).Error; err != nil {
			return err
		}
		for i, id := range itemIDs {
			if err := tx.Model(&models.ClassGalleryItemDB{}).
				Where("id = ? AND assignment_id = ?", id, assignmentID).
				Update("position", i+1).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (g *GatewayImpl) SetGalleryClosed(assignmentID string, closed bool) error {
	var at *time.Time
	if closed {
		now := time.Now().UTC()
		at = &now
	}
	// UpdateColumn: closing the gallery is not a change to the assignment's own fields.
	res := g.db.Model(&models.ClassAssignmentDB{}).Where("id = ?", assignmentID).
		UpdateColumn("gallery_closed_at", at)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
