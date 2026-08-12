package gateway

import (
	"errors"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const commentsListLimit = 200

func (r *ProjectPageGatewayImpl) ListProjectComments(projectId string, limit int) ([]models.ScratchProjectCommentDB, error) {
	if limit <= 0 || limit > commentsListLimit {
		limit = commentsListLimit
	}
	var rows []models.ScratchProjectCommentDB
	err := r.projectStorageDB.
		Where("project_id = ? AND deleted_at IS NULL", projectId).
		Order("created_at ASC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

func (r *ProjectPageGatewayImpl) GetCommentByID(commentId string) (*models.ScratchProjectCommentDB, error) {
	var row models.ScratchProjectCommentDB
	err := r.projectStorageDB.
		Where("id = ? AND deleted_at IS NULL", commentId).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, projectPage.ErrCommentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *ProjectPageGatewayImpl) CreateComment(comment *models.ScratchProjectCommentDB) (*models.ScratchProjectCommentDB, error) {
	now := time.Now().UTC()
	comment.CreatedAt = now
	comment.UpdatedAt = now
	if err := r.projectStorageDB.Create(comment).Error; err != nil {
		return nil, err
	}
	return comment, nil
}

func (r *ProjectPageGatewayImpl) SoftDeleteComment(commentId string) error {
	now := time.Now().UTC()
	res := r.projectStorageDB.Model(&models.ScratchProjectCommentDB{}).
		Where("id = ? AND deleted_at IS NULL", commentId).
		Updates(map[string]interface{}{
			"deleted_at": now,
			"updated_at": now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return projectPage.ErrCommentNotFound
	}
	return nil
}

func (r *ProjectPageGatewayImpl) GetCommentReactionSummary(commentId, viewerUserId string) (*models.CommentReactionsHTTP, error) {
	m, err := r.GetCommentReactionSummaries([]string{commentId}, viewerUserId)
	if err != nil {
		return nil, err
	}
	summary := m[commentId]
	return &summary, nil
}

func (r *ProjectPageGatewayImpl) GetCommentReactionSummaries(
	commentIds []string,
	viewerUserId string,
) (map[string]models.CommentReactionsHTTP, error) {
	out := make(map[string]models.CommentReactionsHTTP, len(commentIds))
	for _, id := range commentIds {
		out[id] = models.CommentReactionsHTTP{}
	}
	if len(commentIds) == 0 {
		return out, nil
	}

	type countRow struct {
		CommentID    string `gorm:"column:comment_id"`
		ReactionCode string `gorm:"column:reaction_code"`
		Count        int64  `gorm:"column:count"`
	}
	var counts []countRow
	if err := r.projectStorageDB.Raw(`
		SELECT comment_id, reaction_code, COUNT(*) AS count
		FROM scratch_comment_reactions
		WHERE comment_id IN ?
		GROUP BY comment_id, reaction_code
	`, commentIds).Scan(&counts).Error; err != nil {
		return nil, err
	}
	for _, row := range counts {
		s := out[row.CommentID]
		switch row.ReactionCode {
		case "like":
			s.Likes = row.Count
		case "dislike":
			s.Dislikes = row.Count
		}
		out[row.CommentID] = s
	}

	if viewerUserId != "" {
		var mine []models.ScratchCommentReactionDB
		if err := r.projectStorageDB.
			Where("comment_id IN ? AND user_id = ?", commentIds, viewerUserId).
			Find(&mine).Error; err != nil {
			return nil, err
		}
		for _, row := range mine {
			s := out[row.CommentID]
			code := row.ReactionCode
			s.MyReaction = &code
			out[row.CommentID] = s
		}
	}
	return out, nil
}

func (r *ProjectPageGatewayImpl) UpsertCommentReaction(commentId, userId, reactionCode string) error {
	reaction := models.ScratchCommentReactionDB{
		CommentID:    commentId,
		UserID:       userId,
		ReactionCode: reactionCode,
	}
	return r.projectStorageDB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "comment_id"}, {Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"reaction_code": reactionCode,
			"updated_at":    gorm.Expr("now()"),
		}),
	}).Create(&reaction).Error
}

func (r *ProjectPageGatewayImpl) DeleteCommentReaction(commentId, userId string) error {
	return r.projectStorageDB.
		Where("comment_id = ? AND user_id = ?", commentId, userId).
		Delete(&models.ScratchCommentReactionDB{}).Error
}
