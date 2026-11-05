package gateway

import (
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
)

func (g *GatewayImpl) AddHistoryEvent(e *models.ClassHistoryEventDB) error {
	e.CreatedAt = time.Now().UTC()
	return g.db.Create(e).Error
}

func (g *GatewayImpl) ListHistoryEvents(classID string, actions []string, before *teacherclass.HistoryCursor, limit int) ([]models.ClassHistoryEventDB, error) {
	q := g.db.Where("class_id = ?", classID)
	if len(actions) > 0 {
		q = q.Where("action IN ?", actions)
	}
	if before != nil {
		q = q.Where("(created_at, id) < (?, ?::uuid)", before.At, before.ID)
	}
	var rows []models.ClassHistoryEventDB
	err := q.Order("created_at DESC, id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}
