package usecase

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/profanity"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
)

const maxCommentBodyRunes = 2000
const maxCommentsPerProject = 200

func (p *ProjectPageUseCaseImpl) requirePublicProject(projectPageId string) (*models.ScratchProjectDB, error) {
	row, err := p.projectPageGateway.GetScratchProjectById(projectPageId)
	if err != nil {
		return nil, err
	}
	if row.DeletedAt != nil || !row.IsPublic {
		return nil, auth.ErrNotAccess
	}
	return row, nil
}

func (p *ProjectPageUseCaseImpl) GetProjectComments(
	projectPageId string,
	viewerId string,
) (*models.ProjectCommentsHTTP, error) {
	if _, err := p.requirePublicProject(projectPageId); err != nil {
		return nil, err
	}
	rows, err := p.projectPageGateway.ListProjectComments(projectPageId, maxCommentsPerProject)
	if err != nil {
		return nil, err
	}
	tree, err := p.buildCommentTree(rows, viewerId)
	if err != nil {
		return nil, err
	}
	return &models.ProjectCommentsHTTP{Comments: tree}, nil
}

func (p *ProjectPageUseCaseImpl) CreateProjectComment(
	projectPageId string,
	userId string,
	body string,
	parentID *string,
) (*models.ProjectCommentHTTP, error) {
	userId = strings.TrimSpace(userId)
	body = strings.TrimSpace(body)
	if userId == "" || body == "" {
		return nil, projectPage.ErrBadRequest
	}
	if utf8.RuneCountInString(body) > maxCommentBodyRunes {
		return nil, projectPage.ErrBadRequest
	}
	if profanity.ContainsProfanity(body) {
		return nil, projectPage.ErrProfanityDetected
	}

	project, err := p.requirePublicProject(projectPageId)
	if err != nil {
		return nil, err
	}

	var parent *models.ScratchProjectCommentDB
	if parentID != nil && strings.TrimSpace(*parentID) != "" {
		pid := strings.TrimSpace(*parentID)
		parent, err = p.projectPageGateway.GetCommentByID(pid)
		if err != nil {
			return nil, err
		}
		if parent.ProjectID != project.ID {
			return nil, projectPage.ErrBadRequest
		}
		parentID = &pid
	} else {
		parentID = nil
	}

	created, err := p.projectPageGateway.CreateComment(&models.ScratchProjectCommentDB{
		ProjectID: project.ID,
		UserID:    userId,
		ParentID:  parentID,
		Body:      body,
	})
	if err != nil {
		return nil, err
	}

	p.notifyComment(project, created, parent)

	author := lookupCommentAuthor(userId)
	return &models.ProjectCommentHTTP{
		ID:        created.ID,
		Body:      created.Body,
		CreatedAt: created.CreatedAt.UTC().Format(time.RFC3339),
		Author:    author,
		Reactions: models.CommentReactionsHTTP{},
		Replies:   []models.ProjectCommentHTTP{},
	}, nil
}

func (p *ProjectPageUseCaseImpl) notifyComment(
	project *models.ScratchProjectDB,
	comment *models.ScratchProjectCommentDB,
	parent *models.ScratchProjectCommentDB,
) {
	if p.notificationGateway == nil {
		return
	}
	actionURL := "/projects/" + project.ID
	ownerID := strings.TrimSpace(project.OwnerUserID)
	authorID := strings.TrimSpace(comment.UserID)

	if parent == nil {
		if ownerID != "" && ownerID != authorID {
			title := strings.TrimSpace(project.Title)
			if title == "" {
				title = "Untitled"
			}
			dedupeKey := "project_comment:" + project.ID + ":" + comment.ID
			_ = p.notificationGateway.CreateOrUpdateByDedupe(&models.UserNotificationDB{
				RecipientUserID: ownerID,
				Title:           "Новый комментарий к проекту",
				Body:            fmt.Sprintf("К проекту «%s» оставили комментарий", title),
				Kind:            "project_comment",
				Severity:        "INFO",
				Source:          "system",
				ActionURL:       &actionURL,
				DedupeKey:       &dedupeKey,
			})
		}
		return
	}

	parentAuthor := strings.TrimSpace(parent.UserID)
	if parentAuthor != "" && parentAuthor != authorID {
		dedupeKey := "comment_reply:" + comment.ID
		_ = p.notificationGateway.CreateOrUpdateByDedupe(&models.UserNotificationDB{
			RecipientUserID: parentAuthor,
			Title:           "Ответ на ваш комментарий",
			Body:            "Вам ответили в комментариях к проекту",
			Kind:            "comment_reply",
			Severity:        "INFO",
			Source:          "system",
			ActionURL:       &actionURL,
			DedupeKey:       &dedupeKey,
		})
	}
	// Also notify project owner about replies (if different from author and parent author).
	if ownerID != "" && ownerID != authorID && ownerID != parentAuthor {
		title := strings.TrimSpace(project.Title)
		if title == "" {
			title = "Untitled"
		}
		dedupeKey := "project_comment:" + project.ID + ":" + comment.ID
		_ = p.notificationGateway.CreateOrUpdateByDedupe(&models.UserNotificationDB{
			RecipientUserID: ownerID,
			Title:           "Новый комментарий к проекту",
			Body:            fmt.Sprintf("К проекту «%s» оставили комментарий", title),
			Kind:            "project_comment",
			Severity:        "INFO",
			Source:          "system",
			ActionURL:       &actionURL,
			DedupeKey:       &dedupeKey,
		})
	}
}

func (p *ProjectPageUseCaseImpl) DeleteProjectComment(
	projectPageId string,
	commentId string,
	userId string,
	role models.Role,
) error {
	userId = strings.TrimSpace(userId)
	commentId = strings.TrimSpace(commentId)
	if userId == "" || commentId == "" {
		return projectPage.ErrBadRequest
	}
	project, err := p.requirePublicProject(projectPageId)
	if err != nil {
		return err
	}
	comment, err := p.projectPageGateway.GetCommentByID(commentId)
	if err != nil {
		return err
	}
	if comment.ProjectID != project.ID {
		return projectPage.ErrCommentNotFound
	}
	isAuthor := strings.TrimSpace(comment.UserID) == userId
	isOwner := strings.TrimSpace(project.OwnerUserID) == userId
	isAdmin := role == models.SuperAdmin
	if !isAuthor && !isOwner && !isAdmin {
		return auth.ErrNotAccess
	}
	return p.projectPageGateway.SoftDeleteComment(commentId)
}

func (p *ProjectPageUseCaseImpl) GetCommentReactions(
	projectPageId, commentId, viewerId string,
) (*models.CommentReactionsHTTP, error) {
	project, err := p.requirePublicProject(projectPageId)
	if err != nil {
		return nil, err
	}
	comment, err := p.projectPageGateway.GetCommentByID(commentId)
	if err != nil {
		return nil, err
	}
	if comment.ProjectID != project.ID {
		return nil, projectPage.ErrCommentNotFound
	}
	return p.projectPageGateway.GetCommentReactionSummary(commentId, viewerId)
}

func (p *ProjectPageUseCaseImpl) PutCommentReaction(
	projectPageId, commentId, userId, reactionCode string,
) (*models.CommentReactionsHTTP, error) {
	userId = strings.TrimSpace(userId)
	reactionCode = strings.TrimSpace(reactionCode)
	if userId == "" || (reactionCode != "like" && reactionCode != "dislike") {
		return nil, projectPage.ErrBadRequest
	}
	project, err := p.requirePublicProject(projectPageId)
	if err != nil {
		return nil, err
	}
	comment, err := p.projectPageGateway.GetCommentByID(commentId)
	if err != nil {
		return nil, err
	}
	if comment.ProjectID != project.ID {
		return nil, projectPage.ErrCommentNotFound
	}

	existing, _ := p.projectPageGateway.GetCommentReactionSummary(commentId, userId)
	if existing != nil && existing.MyReaction != nil && *existing.MyReaction == reactionCode {
		// Toggle off on same reaction.
		if err := p.projectPageGateway.DeleteCommentReaction(commentId, userId); err != nil {
			return nil, err
		}
	} else {
		if err := p.projectPageGateway.UpsertCommentReaction(commentId, userId, reactionCode); err != nil {
			return nil, err
		}
	}
	return p.projectPageGateway.GetCommentReactionSummary(commentId, userId)
}

func (p *ProjectPageUseCaseImpl) DeleteCommentReaction(
	projectPageId, commentId, userId string,
) (*models.CommentReactionsHTTP, error) {
	userId = strings.TrimSpace(userId)
	if userId == "" {
		return nil, projectPage.ErrBadRequest
	}
	project, err := p.requirePublicProject(projectPageId)
	if err != nil {
		return nil, err
	}
	comment, err := p.projectPageGateway.GetCommentByID(commentId)
	if err != nil {
		return nil, err
	}
	if comment.ProjectID != project.ID {
		return nil, projectPage.ErrCommentNotFound
	}
	if err := p.projectPageGateway.DeleteCommentReaction(commentId, userId); err != nil {
		return nil, err
	}
	return p.projectPageGateway.GetCommentReactionSummary(commentId, userId)
}

func (p *ProjectPageUseCaseImpl) buildCommentTree(
	rows []models.ScratchProjectCommentDB,
	viewerId string,
) ([]models.ProjectCommentHTTP, error) {
	if len(rows) == 0 {
		return []models.ProjectCommentHTTP{}, nil
	}
	ids := make([]string, 0, len(rows))
	authorsCache := map[string]models.CommentAuthorHTTP{}
	for _, row := range rows {
		ids = append(ids, row.ID)
		uid := strings.TrimSpace(row.UserID)
		if _, ok := authorsCache[uid]; !ok {
			authorsCache[uid] = lookupCommentAuthor(uid)
		}
	}
	reactions, err := p.projectPageGateway.GetCommentReactionSummaries(ids, viewerId)
	if err != nil {
		return nil, err
	}

	nodes := make(map[string]*models.ProjectCommentHTTP, len(rows))
	for _, row := range rows {
		react := reactions[row.ID]
		nodes[row.ID] = &models.ProjectCommentHTTP{
			ID:        row.ID,
			Body:      row.Body,
			CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339),
			Author:    authorsCache[strings.TrimSpace(row.UserID)],
			Reactions: react,
			Replies:   []models.ProjectCommentHTTP{},
		}
	}

	var rootIDs []string
	childrenOf := map[string][]string{}
	for _, row := range rows {
		if row.ParentID != nil && strings.TrimSpace(*row.ParentID) != "" {
			pid := strings.TrimSpace(*row.ParentID)
			if _, ok := nodes[pid]; ok {
				childrenOf[pid] = append(childrenOf[pid], row.ID)
				continue
			}
		}
		rootIDs = append(rootIDs, row.ID)
	}

	var materialize func(id string) models.ProjectCommentHTTP
	materialize = func(id string) models.ProjectCommentHTTP {
		n := *nodes[id]
		childIDs := childrenOf[id]
		n.Replies = make([]models.ProjectCommentHTTP, 0, len(childIDs))
		for _, cid := range childIDs {
			n.Replies = append(n.Replies, materialize(cid))
		}
		return n
	}

	roots := make([]models.ProjectCommentHTTP, 0, len(rootIDs))
	for _, id := range rootIDs {
		roots = append(roots, materialize(id))
	}
	return roots, nil
}

func lookupCommentAuthor(userID string) models.CommentAuthorHTTP {
	userID = strings.TrimSpace(userID)
	out := models.CommentAuthorHTTP{ID: userID, Nickname: "User " + userID, FullName: ""}
	if userID == "" {
		return out
	}
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil || id <= 0 {
		return out
	}
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		return out
	}
	defer reader.Close()
	profile, err := reader.LookupAuthUserProfileByID(id)
	if err != nil || profile == nil {
		return out
	}
	out.Nickname = strings.TrimSpace(profile.Username)
	name := strings.TrimSpace(profile.ProfileName)
	if name == "" {
		parts := []string{strings.TrimSpace(profile.FirstName), strings.TrimSpace(profile.LastName)}
		var nonEmpty []string
		for _, p := range parts {
			if p != "" {
				nonEmpty = append(nonEmpty, p)
			}
		}
		name = strings.Join(nonEmpty, " ")
	}
	out.FullName = name
	if out.Nickname == "" {
		out.Nickname = name
	}
	if out.Nickname == "" {
		out.Nickname = "User " + userID
	}
	return out
}
