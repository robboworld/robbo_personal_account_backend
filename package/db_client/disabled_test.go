package db_client

import (
	"errors"
	"testing"

	"gorm.io/gorm"
)

type legacyRow struct {
	ID   uint
	Name string
}

func TestDisabledDBRejectsEveryOperation(t *testing.T) {
	db, err := newDisabledDB()
	if err != nil {
		t.Fatal(err)
	}
	var rows []legacyRow
	ops := map[string]error{
		"find":   db.Find(&rows).Error,
		"first":  db.First(&legacyRow{}, 1).Error,
		"create": db.Create(&legacyRow{Name: "x"}).Error,
		"update": db.Model(&legacyRow{ID: 1}).Update("name", "y").Error,
		"delete": db.Delete(&legacyRow{}, 1).Error,
		"raw":    db.Raw("SELECT 1").Scan(&rows).Error,
		"tx": db.Transaction(func(tx *gorm.DB) error {
			return tx.Create(&legacyRow{Name: "z"}).Error
		}),
	}
	for name, opErr := range ops {
		if !errors.Is(opErr, ErrLegacyPostgresDisabled) {
			t.Errorf("%s: err=%v want ErrLegacyPostgresDisabled", name, opErr)
		}
	}
}
