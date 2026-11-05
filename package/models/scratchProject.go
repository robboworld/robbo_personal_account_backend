package models

import (
	"encoding/json"
	"reflect"
	"time"

	"github.com/lib/pq"
)

type ScratchProjectDB struct {
	ID                string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:id"`
	OwnerUserID       string         `gorm:"column:owner_user_id"`
	Title             string         `gorm:"column:title"`
	Instruction       string         `gorm:"column:instruction"`
	Note              string         `gorm:"column:note"`
	ScratchVMJSON     string         `gorm:"column:scratch_vm_json;type:text"`
	IsPublic          bool           `gorm:"column:is_public"`
	LandingFeatured   bool           `gorm:"column:landing_featured"`
	LandingSortOrder  int            `gorm:"column:landing_sort_order"`
	Tags              pq.StringArray `gorm:"column:tags;type:text[];not null"`
	PreviewImage      []byte         `gorm:"column:preview_image"`
	PreviewMime       string         `gorm:"column:preview_mime"`
	PreviewUpdatedAt  *time.Time     `gorm:"column:preview_updated_at"`
	VersionCounter    int64          `gorm:"column:version_counter"`
	CurrentVersionID  *string        `gorm:"column:current_version_id"`
	EdxCourseID       *string        `gorm:"column:edx_course_id"`
	EdxCohortID       *string        `gorm:"column:edx_cohort_id"`
	AssignmentID      *string        `gorm:"column:assignment_id;type:uuid"`
	TemplateProjectID *string        `gorm:"column:template_project_id;type:uuid"`
	ReviewStatus      *string        `gorm:"column:review_status"`
	ReviewComment     *string        `gorm:"column:review_comment"`
	SubmittedAt       *time.Time     `gorm:"column:submitted_at"`
	SubmissionNote    string         `gorm:"column:submission_note"`
	// ReviewedByTeacherID and ReviewedAt: who gave the current review; a new hand-in clears them.
	ReviewedByTeacherID *string    `gorm:"column:reviewed_by_teacher_id"`
	ReviewedAt          *time.Time `gorm:"column:reviewed_at"`
	// TeacherEditedBy, TeacherEditedAt: the teacher who last changed the blocks of handed-in work.
	// StudentVersionID: the .sb3 from before that change, kept from retention. A new hand-in
	// clears all three.
	TeacherEditedBy  *string    `gorm:"column:teacher_edited_by"`
	TeacherEditedAt  *time.Time `gorm:"column:teacher_edited_at"`
	StudentVersionID *string    `gorm:"column:student_version_id;type:uuid"`
	CreatedAt           time.Time  `gorm:"column:created_at"`
	UpdatedAt           time.Time  `gorm:"column:updated_at"`
	DeletedAt           *time.Time `gorm:"column:deleted_at"`
}

func (ScratchProjectDB) TableName() string {
	return "scratch_projects"
}

// SameScratchContent compares two project.json: the editor writes a new .sb3 on every save,
// changed or not. meta names the browser and the VM of the save; monitors get new ids at every load.
func SameScratchContent(a, b string) bool {
	content := func(vmJSON string) (map[string]interface{}, bool) {
		var project map[string]interface{}
		if err := json.Unmarshal([]byte(vmJSON), &project); err != nil || project["targets"] == nil {
			return nil, false
		}
		delete(project, "meta")
		delete(project, "monitors")
		return project, true
	}
	left, ok := content(a)
	if !ok {
		return false
	}
	right, ok := content(b)
	return ok && reflect.DeepEqual(left, right)
}
