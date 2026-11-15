package gateway

import (
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lib/pq"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Kid accounts and class sign-in pages: architecture/ADR_CLASS_KID_LOGIN.md.

func notFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, sql.ErrNoRows) {
		return teacherclass.ErrNotFound
	}
	return err
}

func (g *GatewayImpl) GetKidRoom(classID string) (*models.ClassKidLoginRoomDB, error) {
	var room models.ClassKidLoginRoomDB
	if err := g.db.Where("class_id = ?", classID).First(&room).Error; err != nil {
		return nil, notFound(err)
	}
	return &room, nil
}

func (g *GatewayImpl) GetKidRoomBySlug(slug string) (*models.ClassKidLoginRoomDB, error) {
	var room models.ClassKidLoginRoomDB
	if err := g.db.Where("login_slug = ?", slug).First(&room).Error; err != nil {
		return nil, notFound(err)
	}
	return &room, nil
}

func (g *GatewayImpl) GetKidRoomByCode(code string) (*models.ClassKidLoginRoomDB, error) {
	var room models.ClassKidLoginRoomDB
	if err := g.db.Where("room_code = ?", code).First(&room).Error; err != nil {
		return nil, notFound(err)
	}
	return &room, nil
}

func (g *GatewayImpl) SaveKidRoom(room *models.ClassKidLoginRoomDB) error {
	room.UpdatedAt = time.Now().UTC()
	err := g.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "class_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"login_slug", "open_until", "room_code", "updated_at"}),
	}).Create(room).Error
	// Another class holds the room code (or, once in a lifetime, the slug): pick another.
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return teacherclass.ErrConflict
	}
	return err
}

func (g *GatewayImpl) CreateKidLogin(row *models.ClassKidLoginDB) error {
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now().UTC()
	}
	return g.db.Create(row).Error
}

func (g *GatewayImpl) ListKidLogins(classID string) ([]models.ClassKidLoginDB, error) {
	var rows []models.ClassKidLoginDB
	err := g.db.Where("class_id = ?", classID).Order("created_at ASC, display_name ASC").Find(&rows).Error
	return rows, err
}

func (g *GatewayImpl) CountKidLogins(classID string) (int64, error) {
	var n int64
	err := g.db.Model(&models.ClassKidLoginDB{}).Where("class_id = ?", classID).Count(&n).Error
	return n, err
}

func (g *GatewayImpl) GetKidLogin(userID string) (*models.ClassKidLoginDB, error) {
	var row models.ClassKidLoginDB
	if err := g.db.Where("user_id = ?", userID).First(&row).Error; err != nil {
		return nil, notFound(err)
	}
	return &row, nil
}

func (g *GatewayImpl) GetKidLoginByCardHash(hash string) (*models.ClassKidLoginDB, error) {
	var row models.ClassKidLoginDB
	if err := g.db.Where("card_token_hash = ?", hash).First(&row).Error; err != nil {
		return nil, notFound(err)
	}
	return &row, nil
}

// SetKidPictures gives new pictures; a locked child starts over.
func (g *GatewayImpl) SetKidPictures(userID string, pictures []int) error {
	arr := make(pq.Int64Array, len(pictures))
	for i, p := range pictures {
		arr[i] = int64(p)
	}
	return g.db.Model(&models.ClassKidLoginDB{}).Where("user_id = ?", userID).Updates(map[string]interface{}{
		"pictures":        arr,
		"failed_attempts": 0,
		"locked_at":       nil,
	}).Error
}

func (g *GatewayImpl) SetKidCardHash(userID, hash string) error {
	return g.db.Model(&models.ClassKidLoginDB{}).Where("user_id = ?", userID).
		Update("card_token_hash", hash).Error
}

func (g *GatewayImpl) UnlockKid(userID string) error {
	return g.db.Model(&models.ClassKidLoginDB{}).Where("user_id = ?", userID).Updates(map[string]interface{}{
		"failed_attempts": 0,
		"locked_at":       nil,
	}).Error
}

// RecordKidLoginFailure is one statement, so parallel wrong tries all count.
func (g *GatewayImpl) RecordKidLoginFailure(userID string, lockAfter int) (bool, error) {
	var lockedAt *time.Time
	err := g.db.Raw(`
		UPDATE class_kid_logins
		   SET failed_attempts = failed_attempts + 1,
		       locked_at = CASE WHEN locked_at IS NULL AND failed_attempts + 1 >= ? THEN now() ELSE locked_at END
		 WHERE user_id = ?
		RETURNING locked_at`, lockAfter, userID).Row().Scan(&lockedAt)
	if err != nil {
		return false, notFound(err)
	}
	return lockedAt != nil, nil
}

func (g *GatewayImpl) RecordKidLoginSuccess(userID string, at time.Time) error {
	return g.db.Model(&models.ClassKidLoginDB{}).Where("user_id = ?", userID).Updates(map[string]interface{}{
		"failed_attempts": 0,
		"last_login_at":   at,
	}).Error
}
