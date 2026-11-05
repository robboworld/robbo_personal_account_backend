package gateway

import (
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (g *GatewayImpl) GetFileSubmission(id string) (*models.ClassFileSubmissionDB, error) {
	var s models.ClassFileSubmissionDB
	if err := g.db.Where("id = ?", id).First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (g *GatewayImpl) FindFileSubmission(assignmentID, studentUserID string) (*models.ClassFileSubmissionDB, error) {
	var s models.ClassFileSubmissionDB
	err := g.db.Where("assignment_id = ? AND student_user_id = ?", assignmentID, studentUserID).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (g *GatewayImpl) EnsureFileSubmission(assignmentID, studentUserID string) (*models.ClassFileSubmissionDB, error) {
	now := time.Now().UTC()
	row := models.ClassFileSubmissionDB{
		AssignmentID:  assignmentID,
		StudentUserID: studentUserID,
		ReviewStatus:  models.ReviewDraft,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	// Two attaches at once both insert; the unique index keeps one row.
	err := g.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error
	if err != nil {
		return nil, err
	}
	return g.FindFileSubmission(assignmentID, studentUserID)
}

func (g *GatewayImpl) ListFileSubmissions(assignmentIDs []string) ([]models.ClassFileSubmissionDB, error) {
	var list []models.ClassFileSubmissionDB
	if len(assignmentIDs) == 0 {
		return list, nil
	}
	err := g.db.Where("assignment_id IN ?", assignmentIDs).Order("updated_at DESC").Find(&list).Error
	return list, err
}

type submissionFileRow struct {
	SubmissionID      string `gorm:"column:submission_id"`
	models.UserFileDB `gorm:"embedded"`
}

func (g *GatewayImpl) ListSubmissionFiles(submissionIDs []string) (map[string][]models.UserFileDB, error) {
	out := map[string][]models.UserFileDB{}
	if len(submissionIDs) == 0 {
		return out, nil
	}
	var rows []submissionFileRow
	err := g.db.Raw(`
		SELECT sf.submission_id, f.*
		FROM class_submission_files sf
		JOIN user_files f ON f.id = sf.user_file_id
		WHERE sf.submission_id IN ? AND f.deleted_at IS NULL
		ORDER BY sf.added_at, f.original_name`, submissionIDs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.SubmissionID] = append(out[r.SubmissionID], r.UserFileDB)
	}
	return out, nil
}

func (g *GatewayImpl) AttachSubmissionFile(submissionID, fileID string, maxFiles int) error {
	return g.db.Transaction(func(tx *gorm.DB) error {
		// The row lock makes the count and the insert one step for parallel attaches.
		var sub models.ClassFileSubmissionDB
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", submissionID).First(&sub).Error; err != nil {
			return err
		}
		var already int64
		if err := tx.Model(&models.ClassSubmissionFileDB{}).
			Where("submission_id = ? AND user_file_id = ?", submissionID, fileID).
			Count(&already).Error; err != nil {
			return err
		}
		if already > 0 {
			return nil
		}
		var n int64
		if err := tx.Raw(`
			SELECT count(*) FROM class_submission_files sf
			JOIN user_files f ON f.id = sf.user_file_id
			WHERE sf.submission_id = ? AND f.deleted_at IS NULL`, submissionID).
			Scan(&n).Error; err != nil {
			return err
		}
		if n >= int64(maxFiles) {
			return teacherclass.ErrTooManySubmissionFiles
		}
		now := time.Now().UTC()
		link := models.ClassSubmissionFileDB{SubmissionID: submissionID, UserFileID: fileID, AddedAt: now}
		if err := tx.Create(&link).Error; err != nil {
			return err
		}
		return tx.Model(&models.ClassFileSubmissionDB{}).Where("id = ?", submissionID).
			Update("updated_at", now).Error
	})
}

func (g *GatewayImpl) DetachSubmissionFile(submissionID, fileID string) error {
	return g.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Where("submission_id = ? AND user_file_id = ?", submissionID, fileID).
			Delete(&models.ClassSubmissionFileDB{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Model(&models.ClassFileSubmissionDB{}).Where("id = ?", submissionID).
			Update("updated_at", time.Now().UTC()).Error
	})
}

func (g *GatewayImpl) UpdateFileReview(submissionID, status, comment string, handIn *teacherclass.HandIn) error {
	res := g.db.Model(&models.ClassFileSubmissionDB{}).Where("id = ?", submissionID).
		Updates(reviewUpdates(status, comment, handIn))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (g *GatewayImpl) SetFileSubmissionNote(submissionID, note string) error {
	res := g.db.Model(&models.ClassFileSubmissionDB{}).
		Where("id = ? AND review_status = ?", submissionID, models.ReviewSubmitted).
		UpdateColumn("submission_note", note)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return teacherclass.ErrNotOnReview
	}
	return nil
}

func (g *GatewayImpl) GetUserFiles(ids []string) (map[string]models.UserFileDB, error) {
	out := map[string]models.UserFileDB{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []models.UserFileDB
	if err := g.db.Where("id IN ? AND deleted_at IS NULL", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.ID] = r
	}
	return out, nil
}

func (g *GatewayImpl) FileInUse(ownerUserID, fileID string) (bool, error) {
	var inUse bool
	err := g.db.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM class_submission_files sf
			JOIN class_file_submissions s ON s.id = sf.submission_id
			JOIN class_assignments a ON a.id = s.assignment_id
			JOIN class_invites ci ON ci.edx_course_id = a.edx_course_id
				AND ci.edx_cohort_id = a.edx_cohort_id AND ci.archived_at IS NULL
			WHERE sf.user_file_id = ? AND s.student_user_id = ?
				AND s.review_status IN ?
		) OR EXISTS (
			SELECT 1
			FROM class_assignments a
			JOIN class_invites ci ON ci.edx_course_id = a.edx_course_id
				AND ci.edx_cohort_id = a.edx_cohort_id AND ci.archived_at IS NULL
			WHERE a.template_file_id = ? AND a.created_by_teacher_id = ? AND a.closed_at IS NULL
		)`,
		fileID, ownerUserID,
		[]string{models.ReviewSubmitted, models.ReviewAccepted, models.ReviewShownInClass},
		fileID, ownerUserID,
	).Scan(&inUse).Error
	return inUse, err
}
