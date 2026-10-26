package usecase

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/edx"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type UseCaseImpl struct {
	gw  teacherclass.Gateway
	edx edx.UseCase
}

type UseCaseModule struct {
	fx.Out
	teacherclass.UseCase
}

func SetupTeacherClassUseCase(gw teacherclass.Gateway, edxUC edx.UseCase) UseCaseModule {
	return UseCaseModule{UseCase: &UseCaseImpl{gw: gw, edx: edxUC}}
}

func (u *UseCaseImpl) toDTO(inv *models.ClassInviteDB) *teacherclass.ClassDTO {
	base := strings.TrimRight(viper.GetString("frontend.publicUrl"), "/")
	if base == "" {
		base = "http://localhost:3030"
	}
	return &teacherclass.ClassDTO{
		ID:             inv.ID,
		EdxCourseID:    inv.EdxCourseID,
		EdxCohortID:    inv.EdxCohortID,
		OwnerTeacherID: inv.OwnerTeacherID,
		DisplayName:    inv.DisplayName,
		InviteCode:     inv.InviteCode,
		InviteSlug:     inv.InviteSlug,
		JoinURL:        base + "/join/" + inv.InviteSlug,
		IsActive:       inv.IsActive,
		CreatedAt:      inv.CreatedAt,
	}
}

func (u *UseCaseImpl) requireOwner(teacherID string, role models.Role, classID string) (*models.ClassInviteDB, error) {
	inv, err := u.gw.GetInviteByID(classID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, teacherclass.ErrNotFound
		}
		return nil, err
	}
	if !canManage(inv, teacherID, role) {
		return nil, teacherclass.ErrForbidden
	}
	return inv, nil
}

// canManage: the class owner, or SuperAdmin for any class.
func canManage(inv *models.ClassInviteDB, teacherID string, role models.Role) bool {
	return role == models.SuperAdmin || inv.OwnerTeacherID == teacherID
}

func genInviteCode() (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	part := func(n int) (string, error) {
		b := make([]byte, n)
		for i := 0; i < n; i++ {
			v, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
			if err != nil {
				return "", err
			}
			b[i] = alphabet[v.Int64()]
		}
		return string(b), nil
	}
	a, err := part(4)
	if err != nil {
		return "", err
	}
	b, err := part(2)
	if err != nil {
		return "", err
	}
	return a + "-" + b, nil
}

func genSlug() (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 10)
	for i := range b {
		v, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		b[i] = alphabet[v.Int64()]
	}
	return string(b), nil
}

func (u *UseCaseImpl) ensureCohorted(courseID string) error {
	body, err := u.edx.GetCohortSettings(courseID)
	if err == nil {
		var settings map[string]interface{}
		_ = json.Unmarshal(body, &settings)
		if v, ok := settings["is_cohorted"].(bool); ok && v {
			return nil
		}
	}
	_, err = u.edx.SetCohortSettings(courseID, map[string]interface{}{"is_cohorted": true})
	return err
}

func (u *UseCaseImpl) CreateClass(teacherID string, in teacherclass.CreateClassInput) (*teacherclass.ClassDTO, error) {
	in.CourseID = strings.TrimSpace(in.CourseID)
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	if in.CourseID == "" || in.DisplayName == "" || teacherID == "" {
		return nil, teacherclass.ErrBadRequest
	}
	if err := u.ensureCohorted(in.CourseID); err != nil {
		return nil, fmt.Errorf("%w: enable cohorts: %v", teacherclass.ErrInternal, err)
	}
	body, err := u.edx.CreateCohort(in.CourseID, map[string]interface{}{
		"name":            in.DisplayName,
		"assignment_type": "manual",
	})
	if err != nil {
		// EdX accepts "Manual" in older APIs
		body, err = u.edx.CreateCohort(in.CourseID, map[string]interface{}{
			"name":            in.DisplayName,
			"assignment_type": "Manual",
		})
		if err != nil {
			return nil, fmt.Errorf("%w: create cohort: %v", teacherclass.ErrInternal, err)
		}
	}
	var cohort models.CohortHTTP
	if err := json.Unmarshal(body, &cohort); err != nil {
		return nil, teacherclass.ErrInternal
	}
	code, err := genInviteCode()
	if err != nil {
		return nil, err
	}
	slug, err := genSlug()
	if err != nil {
		return nil, err
	}
	inv := &models.ClassInviteDB{
		EdxCourseID:    in.CourseID,
		EdxCohortID:    strconv.FormatUint(uint64(cohort.ID), 10),
		OwnerTeacherID: teacherID,
		DisplayName:    in.DisplayName,
		InviteCode:     code,
		InviteSlug:     slug,
		IsActive:       true,
	}
	if err := u.gw.CreateInvite(inv); err != nil {
		return nil, err
	}
	dto := u.toDTO(inv)
	u.fillOwnerName(dto)
	return dto, nil
}

func (u *UseCaseImpl) fillOwnerName(dto *teacherclass.ClassDTO) {
	dto.OwnerName = ownerDisplayName(dto.OwnerTeacherID)
}

// fillOwnerNames looks each teacher up once, however many of their classes are listed.
func (u *UseCaseImpl) fillOwnerNames(dtos []teacherclass.ClassDTO) {
	names := make(map[string]string)
	for i := range dtos {
		id := dtos[i].OwnerTeacherID
		name, ok := names[id]
		if !ok {
			name = ownerDisplayName(id)
			names[id] = name
		}
		dtos[i].OwnerName = name
	}
}

// ownerDisplayName returns the teacher's name from the LMS profile, "" when unknown.
// A variable so tests can stand in for LMS MySQL.
var ownerDisplayName = func(ownerID string) string {
	id, err := lmsdb.ParseLMSUserID(ownerID)
	if err != nil {
		return ""
	}
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil || reader == nil {
		return ""
	}
	defer reader.Close()
	profile, err := reader.LookupAuthUserProfileByID(id)
	if err != nil || profile == nil {
		return ""
	}
	name := strings.TrimSpace(profile.ProfileName)
	if name == "" {
		name = strings.TrimSpace(strings.TrimSpace(profile.FirstName) + " " + strings.TrimSpace(profile.LastName))
	}
	if name == "" {
		name = profile.Username
	}
	return name
}

func (u *UseCaseImpl) ListMyClasses(teacherID string, role models.Role) ([]teacherclass.ClassDTO, error) {
	var list []models.ClassInviteDB
	var err error
	if role == models.SuperAdmin {
		list, err = u.gw.ListAllInvites()
	} else {
		list, err = u.gw.ListInvitesByOwner(teacherID)
	}
	if err != nil {
		return nil, err
	}
	out := make([]teacherclass.ClassDTO, 0, len(list))
	for i := range list {
		out = append(out, *u.toDTO(&list[i]))
	}
	u.fillOwnerNames(out)
	return out, nil
}

// ListCourses: running Open edX courses for the "create class" picker.
func (u *UseCaseImpl) ListCourses() ([]teacherclass.CourseDTO, error) {
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	rows, err := reader.ListCourses(500)
	if err != nil {
		return nil, err
	}
	out := make([]teacherclass.CourseDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, teacherclass.CourseDTO{ID: row.ID, Name: row.Name})
	}
	return out, nil
}

func (u *UseCaseImpl) GetClass(teacherID string, role models.Role, classID string) (*teacherclass.ClassDTO, error) {
	inv, err := u.requireOwner(teacherID, role, classID)
	if err != nil {
		return nil, err
	}
	dto := u.toDTO(inv)
	u.fillOwnerName(dto)
	return dto, nil
}

func (u *UseCaseImpl) RenameClass(teacherID string, role models.Role, classID, displayName string) (*teacherclass.ClassDTO, error) {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return nil, teacherclass.ErrBadRequest
	}
	inv, err := u.requireOwner(teacherID, role, classID)
	if err != nil {
		return nil, err
	}
	cohortID, _ := strconv.Atoi(inv.EdxCohortID)
	_, err = u.edx.PatchCohort(inv.EdxCourseID, cohortID, map[string]interface{}{"name": displayName})
	if err != nil {
		return nil, fmt.Errorf("%w: rename cohort", teacherclass.ErrInternal)
	}
	inv.DisplayName = displayName
	if err := u.gw.UpdateInvite(inv); err != nil {
		return nil, err
	}
	return u.toDTO(inv), nil
}

func (u *UseCaseImpl) ArchiveClass(teacherID string, role models.Role, classID string) error {
	inv, err := u.requireOwner(teacherID, role, classID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	inv.ArchivedAt = &now
	inv.IsActive = false
	return u.gw.UpdateInvite(inv)
}

func (u *UseCaseImpl) RotateInvite(teacherID string, role models.Role, classID string) (*teacherclass.ClassDTO, error) {
	inv, err := u.requireOwner(teacherID, role, classID)
	if err != nil {
		return nil, err
	}
	code, err := genInviteCode()
	if err != nil {
		return nil, err
	}
	slug, err := genSlug()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	inv.InviteCode = code
	inv.InviteSlug = slug
	inv.RotatedAt = &now
	inv.IsActive = true
	if err := u.gw.UpdateInvite(inv); err != nil {
		return nil, err
	}
	return u.toDTO(inv), nil
}

func (u *UseCaseImpl) resolveInvite(codeOrSlug string) (*models.ClassInviteDB, error) {
	codeOrSlug = strings.TrimSpace(codeOrSlug)
	if codeOrSlug == "" {
		return nil, teacherclass.ErrBadRequest
	}
	inv, err := u.gw.GetInviteBySlug(codeOrSlug)
	if err == nil {
		return inv, nil
	}
	inv, err = u.gw.GetInviteByCode(codeOrSlug)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, teacherclass.ErrNotFound
		}
		return nil, err
	}
	return inv, nil
}

func (u *UseCaseImpl) PreviewJoin(codeOrSlug string) (*teacherclass.JoinPreviewDTO, error) {
	inv, err := u.resolveInvite(codeOrSlug)
	if err != nil {
		return nil, err
	}
	dto := &teacherclass.JoinPreviewDTO{
		DisplayName: inv.DisplayName,
		EdxCourseID: inv.EdxCourseID,
		InviteSlug:  inv.InviteSlug,
		IsActive:    inv.IsActive,
	}
	classDTO := u.toDTO(inv)
	u.fillOwnerName(classDTO)
	dto.OwnerName = classDTO.OwnerName
	return dto, nil
}

func (u *UseCaseImpl) enrollAndAdd(username, courseID, cohortIDStr string) error {
	cohortID, err := strconv.Atoi(cohortIDStr)
	if err != nil {
		return teacherclass.ErrBadRequest
	}
	_, _ = u.edx.PostEnrollment(map[string]interface{}{
		"course_details": map[string]interface{}{"course_id": courseID},
		"user":           username,
	})
	_, err = u.edx.AddStudent(username, courseID, cohortID)
	return err
}

func (u *UseCaseImpl) JoinClass(studentID string, codeOrSlug string) (*teacherclass.ClassDTO, error) {
	inv, err := u.resolveInvite(codeOrSlug)
	if err != nil {
		return nil, err
	}
	if !inv.IsActive {
		return nil, teacherclass.ErrNotFound
	}
	id, err := lmsdb.ParseLMSUserID(studentID)
	if err != nil {
		return nil, teacherclass.ErrBadRequest
	}
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	user, err := reader.LookupAuthUserByID(id)
	if err != nil || user == nil {
		return nil, teacherclass.ErrBadRequest
	}
	if err := u.enrollAndAdd(user.Username, inv.EdxCourseID, inv.EdxCohortID); err != nil {
		return nil, fmt.Errorf("%w: join cohort: %v", teacherclass.ErrInternal, err)
	}
	// Auto-issue open assignments to late joiner
	assignments, _ := u.gw.ListAssignments(inv.EdxCourseID, inv.EdxCohortID)
	for i := range assignments {
		a := &assignments[i]
		if a.ClosedAt != nil || a.TemplateProjectID == nil {
			continue
		}
		if _, err := u.gw.FindStudentAssignmentProject(a.ID, studentID); err == nil {
			continue
		}
		tpl, err := u.gw.GetProject(*a.TemplateProjectID)
		if err != nil {
			continue
		}
		_, _ = u.gw.ForkProjectForStudent(tpl, studentID, a.Title, a)
	}
	dto := u.toDTO(inv)
	u.fillOwnerName(dto)
	return dto, nil
}

func (u *UseCaseImpl) ListStudentClasses(studentID string) ([]teacherclass.ClassDTO, error) {
	if _, err := lmsdb.ParseLMSUserID(studentID); err != nil {
		return nil, teacherclass.ErrBadRequest
	}
	// One query for the student's cohorts instead of one Open edX API call per class.
	cohorts, err := studentCohortIDs(studentID)
	if err != nil {
		return nil, err
	}
	out := make([]teacherclass.ClassDTO, 0)
	if len(cohorts) == 0 {
		return out, nil
	}
	all, err := u.gw.ListActiveInvites()
	if err != nil {
		return nil, err
	}
	for i := range all {
		inv := &all[i]
		if !cohorts[strings.TrimSpace(inv.EdxCohortID)] {
			continue
		}
		out = append(out, *u.toDTO(inv))
	}
	u.fillOwnerNames(out)
	return out, nil
}

// studentCohortIDs returns the Open edX cohort ids the LMS user belongs to.
// A variable so tests can stand in for LMS MySQL.
var studentCohortIDs = func(userID string) (map[string]bool, error) {
	uid, err := lmsdb.ParseLMSUserID(userID)
	if err != nil {
		return nil, err
	}
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	ids, err := reader.ListUserCohortIDs(uid)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(ids))
	for _, id := range ids {
		out[strconv.FormatInt(id, 10)] = true
	}
	return out, nil
}

// profileName is the LMS display name: profile name, else "first last"; empty when unknown.
func profileName(reader *lmsdb.Reader, id int64) string {
	profile, err := reader.LookupAuthUserProfileByID(id)
	if err != nil || profile == nil {
		return ""
	}
	if name := strings.TrimSpace(profile.ProfileName); name != "" {
		return name
	}
	return strings.TrimSpace(strings.TrimSpace(profile.FirstName) + " " + strings.TrimSpace(profile.LastName))
}

func (u *UseCaseImpl) parseCohortUsers(body []byte) []teacherclass.CohortUserDTO {
	extract := func(items []map[string]interface{}) []teacherclass.CohortUserDTO {
		out := make([]teacherclass.CohortUserDTO, 0, len(items))
		reader, _ := lmsdb.NewReaderFromConfig()
		if reader != nil {
			defer reader.Close()
		}
		for _, item := range items {
			username := fmt.Sprint(item["username"])
			email := fmt.Sprint(item["email"])
			if email == "<nil>" {
				email = ""
			}
			dto := teacherclass.CohortUserDTO{Username: username, Email: email}
			if reader != nil && username != "" {
				if row, err := reader.LookupAuthUserByUsername(username); err == nil && row != nil {
					dto.UserID = strconv.FormatInt(row.ID, 10)
					dto.Email = row.Email
					dto.Name = profileName(reader, row.ID)
				}
			}
			out = append(out, dto)
		}
		return out
	}
	var asList []map[string]interface{}
	if err := json.Unmarshal(body, &asList); err == nil && asList != nil {
		return extract(asList)
	}
	var paginated struct {
		Results []map[string]interface{} `json:"results"`
	}
	if err := json.Unmarshal(body, &paginated); err == nil {
		return extract(paginated.Results)
	}
	return nil
}

func (u *UseCaseImpl) ListMembers(teacherID string, role models.Role, classID string) ([]teacherclass.CohortUserDTO, error) {
	inv, err := u.requireOwner(teacherID, role, classID)
	if err != nil {
		return nil, err
	}
	cohortID, _ := strconv.Atoi(inv.EdxCohortID)
	type edxResult struct {
		body []byte
		err  error
	}
	ch := make(chan edxResult, 1)
	go func() {
		body, err := u.edx.ListCohortUsers(inv.EdxCourseID, cohortID)
		ch <- edxResult{body: body, err: err}
	}()
	select {
	case res := <-ch:
		if res.err != nil {
			// Open edX unreachable (local/dev): keep board usable without roster.
			log.Printf("teacherclass: list cohort users %s/%s: %v", inv.EdxCourseID, inv.EdxCohortID, res.err)
			return []teacherclass.CohortUserDTO{}, nil
		}
		return u.parseCohortUsers(res.body), nil
	case <-time.After(2 * time.Second):
		log.Printf("teacherclass: list cohort users %s/%s: timeout", inv.EdxCourseID, inv.EdxCohortID)
		return []teacherclass.CohortUserDTO{}, nil
	}
}

func (u *UseCaseImpl) AddMembersByEmail(teacherID string, role models.Role, classID string, emails []string) (added []string, missing []string, err error) {
	inv, err := u.requireOwner(teacherID, role, classID)
	if err != nil {
		return nil, nil, err
	}
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		return nil, nil, err
	}
	defer reader.Close()
	for _, raw := range emails {
		email := strings.TrimSpace(strings.ToLower(raw))
		if email == "" {
			continue
		}
		row, err := reader.LookupAuthUserByEmail(email)
		if err != nil || row == nil {
			missing = append(missing, email)
			continue
		}
		if err := u.enrollAndAdd(row.Username, inv.EdxCourseID, inv.EdxCohortID); err != nil {
			missing = append(missing, email)
			continue
		}
		added = append(added, email)
	}
	return added, missing, nil
}

func (u *UseCaseImpl) RemoveMember(teacherID string, role models.Role, classID, username string) error {
	inv, err := u.requireOwner(teacherID, role, classID)
	if err != nil {
		return err
	}
	cohortID, _ := strconv.Atoi(inv.EdxCohortID)
	return u.edx.RemoveCohortUser(username, inv.EdxCourseID, cohortID)
}

func assignmentDTO(a *models.ClassAssignmentDB) teacherclass.AssignmentDTO {
	return teacherclass.AssignmentDTO{
		ID:                 a.ID,
		EdxCourseID:        a.EdxCourseID,
		EdxCohortID:        a.EdxCohortID,
		CreatedByTeacherID: a.CreatedByTeacherID,
		Title:              a.Title,
		Instructions:       a.Instructions,
		AssignmentKey:      a.AssignmentKey,
		TemplateProjectID:  a.TemplateProjectID,
		DueAt:              a.DueAt,
		ClosedAt:           a.ClosedAt,
		CreatedAt:          a.CreatedAt,
	}
}

func (u *UseCaseImpl) CreateAssignment(teacherID string, role models.Role, classID string, in teacherclass.CreateAssignmentInput) (*teacherclass.AssignmentDTO, error) {
	inv, err := u.requireOwner(teacherID, role, classID)
	if err != nil {
		return nil, err
	}
	in.Title = strings.TrimSpace(in.Title)
	in.AssignmentKey = strings.TrimSpace(in.AssignmentKey)
	if in.Title == "" || in.AssignmentKey == "" || in.TemplateProjectID == "" {
		return nil, teacherclass.ErrBadRequest
	}
	tpl, err := u.gw.GetProject(in.TemplateProjectID)
	if err != nil {
		return nil, teacherclass.ErrNotFound
	}
	if tpl.OwnerUserID != teacherID {
		return nil, teacherclass.ErrForbidden
	}
	tplID := in.TemplateProjectID
	a := &models.ClassAssignmentDB{
		EdxCourseID:        inv.EdxCourseID,
		EdxCohortID:        inv.EdxCohortID,
		CreatedByTeacherID: teacherID,
		Title:              in.Title,
		Instructions:       in.Instructions,
		AssignmentKey:      in.AssignmentKey,
		TemplateProjectID:  &tplID,
		DueAt:              in.DueAt,
	}
	if err := u.gw.CreateAssignment(a); err != nil {
		return nil, err
	}
	members, err := u.ListMembers(teacherID, role, classID)
	if err != nil {
		return nil, err
	}
	for _, m := range members {
		if m.UserID == "" {
			continue
		}
		if _, err := u.gw.ForkProjectForStudent(tpl, m.UserID, in.Title, a); err != nil {
			log.Printf("teacherclass: assignment %s: copy for student %s: %v", a.ID, m.UserID, err)
		}
	}
	dto := assignmentDTO(a)
	return &dto, nil
}

func (u *UseCaseImpl) ListAssignments(viewerID string, role models.Role, classID string) ([]teacherclass.AssignmentDTO, error) {
	var inv *models.ClassInviteDB
	var err error
	if role == models.Teacher || role == models.SuperAdmin {
		inv, err = u.gw.GetInviteByID(classID)
		if err != nil {
			return nil, teacherclass.ErrNotFound
		}
		if role == models.Teacher && inv.OwnerTeacherID != viewerID {
			return nil, teacherclass.ErrForbidden
		}
	} else {
		inv, err = u.gw.GetInviteByID(classID)
		if err != nil {
			return nil, teacherclass.ErrNotFound
		}
		member, err := studentInCohort(viewerID, inv.EdxCohortID)
		if err != nil {
			return nil, err
		}
		if !member {
			return nil, teacherclass.ErrForbidden
		}
	}
	list, err := u.gw.ListAssignments(inv.EdxCourseID, inv.EdxCohortID)
	if err != nil {
		return nil, err
	}
	studentView := role != models.Teacher && role != models.SuperAdmin
	out := make([]teacherclass.AssignmentDTO, 0, len(list))
	for i := range list {
		dto := assignmentDTO(&list[i])
		if studentView {
			if p, err := u.gw.FindStudentAssignmentProject(list[i].ID, viewerID); err == nil && p != nil {
				dto.MyProjectID = p.ID
				dto.MyReviewStatus = deref(p.ReviewStatus)
				dto.MyReviewComment = deref(p.ReviewComment)
			}
		}
		out = append(out, dto)
	}
	return out, nil
}

// studentInCohort reports whether the LMS user is a member of the Open edX cohort.
// A variable so tests can stand in for LMS MySQL.
var studentInCohort = func(userID, cohortID string) (bool, error) {
	uid, err := lmsdb.ParseLMSUserID(userID)
	if err != nil {
		return false, nil
	}
	cid, err := strconv.ParseInt(strings.TrimSpace(cohortID), 10, 64)
	if err != nil {
		return false, nil
	}
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		return false, err
	}
	defer reader.Close()
	return reader.IsUserInAnyCohort(uid, []int64{cid})
}

func (u *UseCaseImpl) IssueToLatecomers(teacherID string, role models.Role, classID, assignmentID string) (int, error) {
	inv, err := u.requireOwner(teacherID, role, classID)
	if err != nil {
		return 0, err
	}
	a, err := u.gw.GetAssignment(assignmentID)
	if err != nil || a.TemplateProjectID == nil {
		return 0, teacherclass.ErrNotFound
	}
	if a.EdxCohortID != inv.EdxCohortID {
		return 0, teacherclass.ErrForbidden
	}
	tpl, err := u.gw.GetProject(*a.TemplateProjectID)
	if err != nil {
		return 0, err
	}
	members, err := u.ListMembers(teacherID, role, classID)
	if err != nil {
		return 0, err
	}
	issued := 0
	for _, m := range members {
		if m.UserID == "" {
			continue
		}
		if _, err := u.gw.FindStudentAssignmentProject(a.ID, m.UserID); err == nil {
			continue
		}
		if _, err := u.gw.ForkProjectForStudent(tpl, m.UserID, a.Title, a); err == nil {
			issued++
		} else {
			log.Printf("teacherclass: assignment %s: late copy for student %s: %v", a.ID, m.UserID, err)
		}
	}
	return issued, nil
}

func (u *UseCaseImpl) SubmitAssignment(studentID, projectID string) error {
	p, err := u.gw.GetProject(projectID)
	if err != nil {
		return teacherclass.ErrNotFound
	}
	if p.OwnerUserID != studentID {
		return teacherclass.ErrForbidden
	}
	// Only a copy issued for an assignment goes to the teacher's review queue.
	if p.AssignmentID == nil || strings.TrimSpace(*p.AssignmentID) == "" {
		return teacherclass.ErrBadRequest
	}
	now := time.Now().UTC()
	return u.gw.UpdateReview(projectID, models.ReviewSubmitted, "", &now)
}

func (u *UseCaseImpl) ReviewSubmission(teacherID string, role models.Role, projectID, status, comment string) error {
	switch status {
	case models.ReviewAccepted, models.ReviewReturned, models.ReviewShownInClass, models.ReviewDraft, models.ReviewSubmitted:
	default:
		return teacherclass.ErrBadRequest
	}
	p, err := u.gw.GetProject(projectID)
	if err != nil {
		return teacherclass.ErrNotFound
	}
	if p.EdxCohortID == nil {
		return teacherclass.ErrForbidden
	}
	inv, err := u.gw.GetInviteByCourseCohort(deref(p.EdxCourseID), *p.EdxCohortID)
	if err != nil || !canManage(inv, teacherID, role) {
		return teacherclass.ErrForbidden
	}
	return u.gw.UpdateReview(projectID, status, comment, nil)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (u *UseCaseImpl) ListSubmissions(teacherID string, role models.Role, classID, assignmentID string) ([]teacherclass.SubmissionDTO, error) {
	inv, err := u.requireOwner(teacherID, role, classID)
	if err != nil {
		return nil, err
	}
	a, err := u.gw.GetAssignment(assignmentID)
	if err != nil || a.EdxCohortID != inv.EdxCohortID {
		return nil, teacherclass.ErrNotFound
	}
	list, err := u.gw.ListSubmissions(assignmentID)
	if err != nil {
		return nil, err
	}
	out := make([]teacherclass.SubmissionDTO, 0, len(list))
	for i := range list {
		p := &list[i]
		status := ""
		if p.ReviewStatus != nil {
			status = *p.ReviewStatus
		}
		comment := ""
		if p.ReviewComment != nil {
			comment = *p.ReviewComment
		}
		out = append(out, teacherclass.SubmissionDTO{
			ProjectID:     p.ID,
			OwnerUserID:   p.OwnerUserID,
			Title:         p.Title,
			ReviewStatus:  status,
			ReviewComment: comment,
			UpdatedAt:     p.UpdatedAt,
			SubmittedAt:   p.SubmittedAt,
			PreviewURL:    "/projectPage/" + p.ID + "/preview",
		})
	}
	return out, nil
}

// ProgressMatrix returns each assignment's status per project owner. It does not call
// Open edX: the board joins these rows with the roster it already loaded from /members.
func (u *UseCaseImpl) ProgressMatrix(teacherID string, role models.Role, classID string) ([]teacherclass.AssignmentDTO, []teacherclass.ProgressRowDTO, error) {
	inv, err := u.requireOwner(teacherID, role, classID)
	if err != nil {
		return nil, nil, err
	}
	assignments, err := u.gw.ListAssignments(inv.EdxCourseID, inv.EdxCohortID)
	if err != nil {
		return nil, nil, err
	}
	assignDTOs := make([]teacherclass.AssignmentDTO, 0, len(assignments))
	rows := []teacherclass.ProgressRowDTO{}
	rowOf := map[string]int{}
	for i := range assignments {
		a := &assignments[i]
		assignDTOs = append(assignDTOs, assignmentDTO(a))
		subs, err := u.gw.ListSubmissions(a.ID)
		if err != nil {
			return nil, nil, err
		}
		seen := map[string]struct{}{}
		// Newest first: the latest project wins if a student somehow has two.
		for j := range subs {
			p := &subs[j]
			if _, ok := seen[p.OwnerUserID]; ok {
				continue
			}
			seen[p.OwnerUserID] = struct{}{}
			status := "draft"
			if p.ReviewStatus != nil {
				status = *p.ReviewStatus
			}
			idx, ok := rowOf[p.OwnerUserID]
			if !ok {
				idx = len(rows)
				rowOf[p.OwnerUserID] = idx
				rows = append(rows, teacherclass.ProgressRowDTO{UserID: p.OwnerUserID})
			}
			rows[idx].Cells = append(rows[idx].Cells, teacherclass.ProgressCellDTO{
				AssignmentID: a.ID,
				Status:       status,
				ProjectID:    p.ID,
			})
		}
	}
	return assignDTOs, rows, nil
}
