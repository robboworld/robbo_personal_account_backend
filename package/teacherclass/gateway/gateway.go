package gateway

import (
	"strings"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/db_client"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type GatewayImpl struct {
	db *gorm.DB
}

type GatewayModule struct {
	fx.Out
	teacherclass.Gateway
}

func SetupTeacherClassGateway(postgresClient db_client.PostgresClient) GatewayModule {
	_ = postgresClient
	dsn := viper.GetString("projectsPostgres.postgresDsn")
	if dsn == "" {
		panic("projectsPostgres.postgresDsn required for teacherclass")
	}
	db, err := db_client.OpenByDSN(dsn)
	if err != nil {
		panic(err)
	}
	return GatewayModule{Gateway: &GatewayImpl{db: db}}
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

func (g *GatewayImpl) ListInvitesByOwner(ownerTeacherID string) ([]models.ClassInviteDB, error) {
	var list []models.ClassInviteDB
	err := g.db.Where("owner_teacher_id = ? AND archived_at IS NULL", ownerTeacherID).
		Order("created_at DESC").Find(&list).Error
	return list, err
}

func (g *GatewayImpl) ListActiveInvites() ([]models.ClassInviteDB, error) {
	var list []models.ClassInviteDB
	err := g.db.Where("is_active = TRUE AND archived_at IS NULL").Find(&list).Error
	return list, err
}

func (g *GatewayImpl) CreateAssignment(a *models.ClassAssignmentDB) error {
	now := time.Now().UTC()
	a.CreatedAt = now
	a.UpdatedAt = now
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

func (g *GatewayImpl) CloseAssignment(id string) error {
	now := time.Now().UTC()
	return g.db.Model(&models.ClassAssignmentDB{}).Where("id = ?", id).
		Updates(map[string]interface{}{"closed_at": now, "updated_at": now}).Error
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
	tplID := template.ID
	row := models.ScratchProjectDB{
		OwnerUserID:       studentUserID,
		Title:             title,
		Instruction:       template.Instruction,
		Note:              template.Note,
		ScratchVMJSON:     template.ScratchVMJSON,
		IsPublic:          false,
		EdxCourseID:       &courseID,
		EdxCohortID:       &cohortID,
		AssignmentID:      &assignID,
		TemplateProjectID: &tplID,
		ReviewStatus:      &status,
	}
	if row.ScratchVMJSON == "" {
		row.ScratchVMJSON = "{}"
	}
	if err := g.db.Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (g *GatewayImpl) ListSubmissions(assignmentID string) ([]models.ScratchProjectDB, error) {
	var list []models.ScratchProjectDB
	err := g.db.Where("assignment_id = ? AND deleted_at IS NULL", assignmentID).
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

func (g *GatewayImpl) UpdateReview(projectID, status, comment string, submittedAt *time.Time) error {
	updates := map[string]interface{}{
		"review_status":  status,
		"review_comment": comment,
		"updated_at":     time.Now().UTC(),
	}
	if submittedAt != nil {
		updates["submitted_at"] = *submittedAt
	}
	res := g.db.Model(&models.ScratchProjectDB{}).
		Where("id = ? AND deleted_at IS NULL", projectID).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
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
