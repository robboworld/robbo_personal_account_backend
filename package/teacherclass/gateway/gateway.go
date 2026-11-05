package gateway

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/db_client"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GatewayImpl struct {
	db *gorm.DB
}

type GatewayModule struct {
	fx.Out
	teacherclass.Gateway
}

func SetupTeacherClassGateway(postgresClient db_client.PostgresClient) (GatewayModule, error) {
	_ = postgresClient
	dsn := viper.GetString("projectsPostgres.postgresDsn")
	if dsn == "" {
		return GatewayModule{}, errors.New("projectsPostgres.postgresDsn required for teacherclass")
	}
	db, err := db_client.OpenByDSN(dsn)
	if err != nil {
		return GatewayModule{}, fmt.Errorf("SetupTeacherClassGateway: %w", err)
	}
	return GatewayModule{Gateway: &GatewayImpl{db: db}}, nil
}

func (g *GatewayImpl) CreateInvite(inv *models.ClassInviteDB) error {
	now := time.Now().UTC()
	inv.CreatedAt = now
	inv.UpdatedAt = now
	inv.IsActive = true
	return g.db.Create(inv).Error
}

func (g *GatewayImpl) UpdateInvite(inv *models.ClassInviteDB) error {
	inv.UpdatedAt = time.Now().UTC()
	return g.db.Save(inv).Error
}

func (g *GatewayImpl) GetInviteByID(id string) (*models.ClassInviteDB, error) {
	var inv models.ClassInviteDB
	err := g.db.Where("id = ? AND archived_at IS NULL", id).First(&inv).Error
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (g *GatewayImpl) GetInviteBySlug(slug string) (*models.ClassInviteDB, error) {
	var inv models.ClassInviteDB
	err := g.db.Where("invite_slug = ? AND is_active = TRUE AND archived_at IS NULL", strings.TrimSpace(slug)).First(&inv).Error
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (g *GatewayImpl) GetInviteByCode(code string) (*models.ClassInviteDB, error) {
	var inv models.ClassInviteDB
	err := g.db.Where("UPPER(invite_code) = UPPER(?) AND is_active = TRUE AND archived_at IS NULL", strings.TrimSpace(code)).First(&inv).Error
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (g *GatewayImpl) GetInviteByCourseCohort(courseID, cohortID string) (*models.ClassInviteDB, error) {
	var inv models.ClassInviteDB
	err := g.db.Where("edx_course_id = ? AND edx_cohort_id = ? AND archived_at IS NULL", courseID, cohortID).First(&inv).Error
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (g *GatewayImpl) ListInvitesByTeacher(teacherID string) ([]models.ClassInviteDB, error) {
	var list []models.ClassInviteDB
	err := g.db.Where(`archived_at IS NULL AND (owner_teacher_id = ? OR id IN (
			SELECT class_id FROM class_co_teachers
			WHERE teacher_id = ? AND revoked_at IS NULL AND expires_at > ?
		))`, teacherID, teacherID, time.Now().UTC()).
		Order("created_at DESC").Find(&list).Error
	return list, err
}

func (g *GatewayImpl) ListAllInvites() ([]models.ClassInviteDB, error) {
	var list []models.ClassInviteDB
	err := g.db.Where("archived_at IS NULL").Order("created_at DESC").Find(&list).Error
	return list, err
}

func (g *GatewayImpl) ListActiveInvites() ([]models.ClassInviteDB, error) {
	var list []models.ClassInviteDB
	err := g.db.Where("is_active = TRUE AND archived_at IS NULL").Find(&list).Error
	return list, err
}

func (g *GatewayImpl) AddMember(classID, userID string) error {
	row := models.ClassMemberDB{ClassID: classID, UserID: userID, JoinedAt: time.Now().UTC()}
	return g.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error
}

func (g *GatewayImpl) RemoveMember(classID, userID string) error {
	return g.db.Where("class_id = ? AND user_id = ?", classID, userID).Delete(&models.ClassMemberDB{}).Error
}

func (g *GatewayImpl) ListMembers(classID string) ([]models.ClassMemberDB, error) {
	var list []models.ClassMemberDB
	err := g.db.Where("class_id = ?", classID).Order("joined_at ASC").Find(&list).Error
	return list, err
}

func (g *GatewayImpl) ListMemberClassIDs(userID string) ([]string, error) {
	var ids []string
	err := g.db.Model(&models.ClassMemberDB{}).Where("user_id = ?", userID).Pluck("class_id", &ids).Error
	return ids, err
}

func (g *GatewayImpl) IsMemberOfAny(userID string, classIDs []string) (bool, error) {
	if len(classIDs) == 0 {
		return false, nil
	}
	var n int64
	err := g.db.Model(&models.ClassMemberDB{}).
		Where("user_id = ? AND class_id IN ?", userID, classIDs).Count(&n).Error
	return n > 0, err
}

func (g *GatewayImpl) CreateAssignment(a *models.ClassAssignmentDB) error {
	now := time.Now().UTC()
	a.CreatedAt = now
	a.UpdatedAt = now
	if a.Kind == "" {
		a.Kind = models.AssignmentScratch
	}
	// allowed_kinds is NOT NULL: a nil slice is written as NULL.
	if a.AllowedKinds == nil {
		a.AllowedKinds = pq.StringArray{}
	}
	return g.db.Create(a).Error
}

func (g *GatewayImpl) ListAssignments(courseID, cohortID string) ([]models.ClassAssignmentDB, error) {
	var list []models.ClassAssignmentDB
	err := g.db.Where("edx_course_id = ? AND edx_cohort_id = ?", courseID, cohortID).
		Order("created_at ASC").Find(&list).Error
	return list, err
}

func (g *GatewayImpl) GetAssignment(id string) (*models.ClassAssignmentDB, error) {
	var a models.ClassAssignmentDB
	err := g.db.Where("id = ?", id).First(&a).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// SetAssignmentClosed stops (closed) or resumes accepting work for the assignment.
func (g *GatewayImpl) SetAssignmentClosed(id string, closed bool) error {
	now := time.Now().UTC()
	var closedAt interface{}
	if closed {
		closedAt = now
	}
	res := g.db.Model(&models.ClassAssignmentDB{}).Where("id = ?", id).
		Updates(map[string]interface{}{"closed_at": closedAt, "updated_at": now})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (g *GatewayImpl) GetProject(id string) (*models.ScratchProjectDB, error) {
	var p models.ScratchProjectDB
	err := g.db.Where("id = ? AND deleted_at IS NULL", id).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (g *GatewayImpl) ForkProjectForStudent(template *models.ScratchProjectDB, studentUserID, title string, assignment *models.ClassAssignmentDB) (*models.ScratchProjectDB, error) {
	status := models.ReviewDraft
	courseID := assignment.EdxCourseID
	cohortID := assignment.EdxCohortID
	assignID := assignment.ID
	row := models.ScratchProjectDB{
		OwnerUserID:  studentUserID,
		Title:        title,
		IsPublic:     false,
		EdxCourseID:  &courseID,
		EdxCohortID:  &cohortID,
		AssignmentID: &assignID,
		ReviewStatus: &status,
		// tags is NOT NULL: a nil slice made every copy fail, so no student ever got the work.
		Tags: pq.StringArray{},
	}
	if template != nil {
		tplID := template.ID
		row.Instruction = template.Instruction
		row.Note = template.Note
		row.ScratchVMJSON = template.ScratchVMJSON
		row.TemplateProjectID = &tplID
		row.Tags = append(row.Tags, template.Tags...)
	}
	if row.ScratchVMJSON == "" {
		row.ScratchVMJSON = "{}"
	}
	err := g.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		if template == nil {
			return nil
		}
		return copyCurrentSb3(tx, template, &row)
	})
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// saveSourceAssignmentCopy marks the version a copy is issued with; the backfill script of old
// copies (robbo_projects_db/scripts/backfill_assignment_sb3) adds a suffix to it.
const saveSourceAssignmentCopy = "class.assignment"

// HasOwnSave: retention drops old versions, so the issued one may be gone; it is not the count
// of versions that tells, but a version from anywhere else. Versions older than save_source
// have none and count as the student's.
func (g *GatewayImpl) HasOwnSave(projectID string) (bool, error) {
	var n int64
	err := g.db.Model(&models.ScratchProjectVersionDB{}).
		Where("project_id = ? AND (save_source IS NULL OR save_source NOT LIKE ?)", projectID, saveSourceAssignmentCopy+"%").
		Count(&n).Error
	return n > 0, err
}

// copyCurrentSb3 gives the copy the template's .sb3 as its first version. Costumes and
// sounds live only in the archive: a JSON-only copy fails in the player with
// "costume.asset is null" for any asset not bundled with the editor.
func copyCurrentSb3(tx *gorm.DB, template, row *models.ScratchProjectDB) error {
	// The current version, the newest one if current is unset (as GetLatestSb3Archive).
	// Two queries: gorm's Order() silently drops a clause.Expr such as "(id = ?) DESC".
	var src models.ScratchProjectVersionDB
	err := gorm.ErrRecordNotFound
	if template.CurrentVersionID != nil && *template.CurrentVersionID != "" {
		err = tx.Where("id = ? AND project_id = ?", *template.CurrentVersionID, template.ID).First(&src).Error
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = tx.Where("project_id = ?", template.ID).Order("version_seq DESC").First(&src).Error
	}
	if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && len(src.Archive) == 0) {
		return nil
	}
	if err != nil {
		return err
	}
	saveSource := saveSourceAssignmentCopy
	ver := models.ScratchProjectVersionDB{
		ProjectID:       row.ID,
		VersionSeq:      1,
		Archive:         src.Archive,
		SizeBytes:       int64(len(src.Archive)),
		ChecksumSHA256:  src.ChecksumSHA256,
		CreatedByUserID: template.OwnerUserID,
		SaveSource:      &saveSource,
	}
	if err := tx.Create(&ver).Error; err != nil {
		return err
	}
	row.VersionCounter = 1
	row.CurrentVersionID = &ver.ID
	return tx.Model(&models.ScratchProjectDB{}).Where("id = ?", row.ID).Updates(map[string]interface{}{
		"version_counter":    1,
		"current_version_id": ver.ID,
	}).Error
}

func (g *GatewayImpl) ListSubmissions(assignmentID string) ([]models.ScratchProjectDB, error) {
	var list []models.ScratchProjectDB
	err := g.db.Where("assignment_id = ? AND deleted_at IS NULL", assignmentID).
		Order("updated_at DESC").Find(&list).Error
	return list, err
}

// ListSubmissionsForAssignments is ListSubmissions for several assignments in one query.
func (g *GatewayImpl) ListSubmissionsForAssignments(assignmentIDs []string) ([]models.ScratchProjectDB, error) {
	var list []models.ScratchProjectDB
	if len(assignmentIDs) == 0 {
		return list, nil
	}
	err := g.db.Where("assignment_id IN ? AND deleted_at IS NULL", assignmentIDs).
		Order("updated_at DESC").Find(&list).Error
	return list, err
}

func (g *GatewayImpl) ListProjectsByCohortAssignment(cohortID, assignmentID string) ([]models.ScratchProjectDB, error) {
	var list []models.ScratchProjectDB
	q := g.db.Where("edx_cohort_id = ? AND deleted_at IS NULL", cohortID)
	if assignmentID != "" {
		q = q.Where("assignment_id = ?", assignmentID)
	}
	err := q.Order("updated_at DESC").Find(&list).Error
	return list, err
}

// reviewUpdates: the columns a review or a hand-in changes in either kind of work.
func reviewUpdates(status, comment, reviewerID string, handIn *teacherclass.HandIn) map[string]interface{} {
	now := time.Now().UTC()
	updates := map[string]interface{}{
		"review_status":  status,
		"review_comment": comment,
		"updated_at":     now,
	}
	if handIn != nil {
		updates["submitted_at"] = handIn.At
		updates["submission_note"] = handIn.Note
		updates["reviewed_by_teacher_id"] = nil
		updates["reviewed_at"] = nil
	} else {
		updates["reviewed_by_teacher_id"] = reviewerID
		updates["reviewed_at"] = now
	}
	return updates
}

func (g *GatewayImpl) UpdateReview(projectID, status, comment, reviewerID string, handIn *teacherclass.HandIn) error {
	res := g.db.Model(&models.ScratchProjectDB{}).
		Where("id = ? AND deleted_at IS NULL", projectID).
		Updates(reviewUpdates(status, comment, reviewerID, handIn))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (g *GatewayImpl) SetProjectNote(projectID, note string) error {
	// In the WHERE, not checked before: a review arriving at the same moment wins.
	res := g.db.Model(&models.ScratchProjectDB{}).
		Where("id = ? AND deleted_at IS NULL AND review_status = ?", projectID, models.ReviewSubmitted).
		UpdateColumn("submission_note", note)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return teacherclass.ErrNotOnReview
	}
	return nil
}

func (g *GatewayImpl) FindStudentAssignmentProject(assignmentID, studentUserID string) (*models.ScratchProjectDB, error) {
	var p models.ScratchProjectDB
	err := g.db.Where("assignment_id = ? AND owner_user_id = ? AND deleted_at IS NULL", assignmentID, studentUserID).
		First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}
