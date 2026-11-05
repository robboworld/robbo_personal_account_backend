package gateway

import (
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const coTeacherRunning = "revoked_at IS NULL AND expires_at > ?"

func (g *GatewayImpl) AddCoTeacher(row *models.ClassCoTeacherDB, maxActive int) error {
	return g.db.Transaction(func(tx *gorm.DB) error {
		// Serialises appointments to the class: two at once would both pass the limit.
		var inv models.ClassInviteDB
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND archived_at IS NULL", row.ClassID).First(&inv).Error
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		// A finished term still holds the unique index until it is closed.
		err = tx.Model(&models.ClassCoTeacherDB{}).
			Where("class_id = ? AND teacher_id = ? AND revoked_at IS NULL AND expires_at <= ?", row.ClassID, row.TeacherID, now).
			Update("revoked_at", gorm.Expr("expires_at")).Error
		if err != nil {
			return err
		}
		var running []models.ClassCoTeacherDB
		if err := tx.Where("class_id = ? AND "+coTeacherRunning, row.ClassID, now).Find(&running).Error; err != nil {
			return err
		}
		for i := range running {
			if running[i].TeacherID == row.TeacherID {
				return teacherclass.ErrAlreadyCoTeacher
			}
		}
		if len(running) >= maxActive {
			return teacherclass.ErrTooManyCoTeachers
		}
		row.GrantedAt = now
		return tx.Create(row).Error
	})
}

func (g *GatewayImpl) IsCoTeacher(classID, teacherID string) (bool, error) {
	var n int64
	err := g.db.Model(&models.ClassCoTeacherDB{}).
		Where("class_id = ? AND teacher_id = ? AND "+coTeacherRunning, classID, teacherID, time.Now().UTC()).
		Count(&n).Error
	return n > 0, err
}

func (g *GatewayImpl) WasCoTeacher(classID, teacherID string) (bool, error) {
	var n int64
	err := g.db.Model(&models.ClassCoTeacherDB{}).
		Where("class_id = ? AND teacher_id = ?", classID, teacherID).Count(&n).Error
	return n > 0, err
}

func (g *GatewayImpl) ListCoTeachers(classID string) ([]models.ClassCoTeacherDB, error) {
	var list []models.ClassCoTeacherDB
	err := g.db.Where("class_id = ? AND "+coTeacherRunning, classID, time.Now().UTC()).
		Order("granted_at ASC").Find(&list).Error
	return list, err
}

func (g *GatewayImpl) ListCoTeachings(teacherID string) ([]models.ClassCoTeacherDB, error) {
	var list []models.ClassCoTeacherDB
	err := g.db.Where("teacher_id = ? AND "+coTeacherRunning, teacherID, time.Now().UTC()).
		Where("class_id IN (SELECT id FROM class_invites WHERE archived_at IS NULL)").
		Find(&list).Error
	return list, err
}

func (g *GatewayImpl) SetCoTeacherExpiry(classID, teacherID string, expiresAt time.Time, remindedAt *time.Time) (*models.ClassCoTeacherDB, error) {
	var row models.ClassCoTeacherDB
	res := g.db.Model(&row).Clauses(clause.Returning{}).
		Where("class_id = ? AND teacher_id = ? AND "+coTeacherRunning, classID, teacherID, time.Now().UTC()).
		Updates(map[string]interface{}{"expires_at": expiresAt, "reminded_at": remindedAt})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (g *GatewayImpl) RevokeCoTeacher(classID, teacherID, revokedBy string) error {
	now := time.Now().UTC()
	res := g.db.Model(&models.ClassCoTeacherDB{}).
		Where("class_id = ? AND teacher_id = ? AND "+coTeacherRunning, classID, teacherID, now).
		Updates(map[string]interface{}{"revoked_at": now, "revoked_by": revokedBy})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (g *GatewayImpl) ClaimExpiringCoTeachers(before time.Time) ([]models.ClassCoTeacherDB, error) {
	var list []models.ClassCoTeacherDB
	now := time.Now().UTC()
	err := g.db.Raw(`
		UPDATE class_co_teachers c SET reminded_at = ?
		FROM class_invites i
		WHERE i.id = c.class_id AND i.archived_at IS NULL
			AND c.revoked_at IS NULL AND c.reminded_at IS NULL
			AND c.expires_at > ? AND c.expires_at <= ?
		RETURNING c.*`, now, now, before).Scan(&list).Error
	return list, err
}
