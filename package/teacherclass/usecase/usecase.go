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
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/edx"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/notifications"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

type UseCaseImpl struct {
	gw  teacherclass.Gateway
	edx edx.UseCase
	// files is "Мои файлы": file assignments are handed in from there.
	files userfiles.UseCase
	// notify tells the teacher about what happens in their class.
	notify notifications.Gateway
	// joins runs one join per class and student at a time. Without it, requests that overlap
	// (two tabs, a retry) all see a non-member: the teacher's count grows by each of them and
	// every assignment is copied once per request.
	joins singleflight.Group
}

type UseCaseModule struct {
	fx.Out
	teacherclass.UseCase
}

func SetupTeacherClassUseCase(gw teacherclass.Gateway, edxUC edx.UseCase, files userfiles.UseCase, notify notifications.Gateway) UseCaseModule {
	return UseCaseModule{UseCase: &UseCaseImpl{gw: gw, edx: edxUC, files: files, notify: notify}}
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
		LmsLinked:      inv.LmsLinked,
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

// canManage: the class owner, or SuperAdmin for any class. What belongs to the class itself
// (rename, archive, a new invite code, co-teachers) stops here.
func canManage(inv *models.ClassInviteDB, teacherID string, role models.Role) bool {
	return role == models.SuperAdmin || inv.OwnerTeacherID == teacherID
}

// canTeach adds a co-teacher with a running term to canManage: the roster, assignments, review.
func (u *UseCaseImpl) canTeach(inv *models.ClassInviteDB, teacherID string, role models.Role) (bool, error) {
	if canManage(inv, teacherID, role) {
		return true, nil
	}
	if role != models.Teacher {
		return false, nil
	}
	return u.gw.IsCoTeacher(inv.ID, teacherID)
}

// requireTeaching is requireOwner with canTeach.
func (u *UseCaseImpl) requireTeaching(teacherID string, role models.Role, classID string) (*models.ClassInviteDB, error) {
	inv, err := u.gw.GetInviteByID(classID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, teacherclass.ErrNotFound
		}
		return nil, err
	}
	teaches, err := u.canTeach(inv, teacherID, role)
	if err != nil {
		return nil, err
	}
	if !teaches {
		return nil, teacherclass.ErrForbidden
	}
	return inv, nil
}

// coTeacherTerms maps the classes the teacher co-teaches to the end of the term.
func (u *UseCaseImpl) coTeacherTerms(teacherID string) (map[string]time.Time, error) {
	rows, err := u.gw.ListCoTeachings(teacherID)
	if err != nil {
		return nil, err
	}
	out := make(map[string]time.Time, len(rows))
	for i := range rows {
		out[rows[i].ClassID] = rows[i].ExpiresAt
	}
	return out, nil
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

func (u *UseCaseImpl) CreateClass(teacherID string, role models.Role, in teacherclass.CreateClassInput) (*teacherclass.ClassDTO, error) {
	in.CourseID = strings.TrimSpace(in.CourseID)
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	if in.DisplayName == "" || teacherID == "" {
		return nil, teacherclass.ErrBadRequest
	}
	if !in.LinkLMS {
		return u.createLocalClass(teacherID, in.DisplayName)
	}
	if in.CourseID == "" {
		return nil, teacherclass.ErrBadRequest
	}
	allowed, err := teacherCourseIDs(teacherID, role)
	if err != nil {
		return nil, err
	}
	if allowed != nil && !allowed[in.CourseID] {
		return nil, teacherclass.ErrForbidden
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
		LmsLinked:      true,
	}
	if err := u.gw.CreateInvite(inv); err != nil {
		return nil, err
	}
	dto := u.toDTO(inv)
	u.fillOwnerName(dto)
	return dto, nil
}

// createLocalClass makes a class the LMS knows nothing about: no course, no cohort, the
// roster lives in class_members.
func (u *UseCaseImpl) createLocalClass(teacherID, displayName string) (*teacherclass.ClassDTO, error) {
	code, err := genInviteCode()
	if err != nil {
		return nil, err
	}
	slug, err := genSlug()
	if err != nil {
		return nil, err
	}
	id := uuid.NewString()
	inv := &models.ClassInviteDB{
		ID:             id,
		EdxCohortID:    models.UnlinkedCohortKey(id),
		OwnerTeacherID: teacherID,
		DisplayName:    displayName,
		InviteCode:     code,
		InviteSlug:     slug,
		IsActive:       true,
		LmsLinked:      false,
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

// ownerDisplayName returns the user's name from the LMS profile, "" when unknown.
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
		list, err = u.gw.ListInvitesByTeacher(teacherID)
	}
	if err != nil {
		return nil, err
	}
	terms, err := u.coTeacherTerms(teacherID)
	if err != nil {
		return nil, err
	}
	out := make([]teacherclass.ClassDTO, 0, len(list))
	for i := range list {
		dto := u.toDTO(&list[i])
		if until, ok := terms[dto.ID]; ok {
			dto.CoTeacherUntil = &until
		}
		out = append(out, *dto)
	}
	u.fillOwnerNames(out)
	return out, nil
}

// ListCourses: running Open edX courses for the "create class" picker; for a teacher who is
// only on some course teams, just those courses.
func (u *UseCaseImpl) ListCourses(teacherID string, role models.Role) ([]teacherclass.CourseDTO, error) {
	allowed, err := teacherCourseIDs(teacherID, role)
	if err != nil {
		return nil, err
	}
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
		if allowed != nil && !allowed[row.ID] {
			continue
		}
		out = append(out, teacherclass.CourseDTO{ID: row.ID, Name: row.Name})
	}
	return out, nil
}

// teacherCourseIDs returns the courses a teacher may link classes to: nil means every course
// (SuperAdmin, or LMS global staff); otherwise the courses where the teacher is staff or
// instructor. A teacher by course team must not get cohorts in someone else's course.
func teacherCourseIDs(teacherID string, role models.Role) (map[string]bool, error) {
	if role == models.SuperAdmin {
		return nil, nil
	}
	id, err := strconv.ParseInt(strings.TrimSpace(teacherID), 10, 64)
	if err != nil || id <= 0 {
		return nil, teacherclass.ErrForbidden
	}
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	profile, err := reader.LookupAuthUserProfileByID(id)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, teacherclass.ErrForbidden
	}
	if profile.IsStaff || profile.IsSuperuser {
		return nil, nil
	}
	ids, err := reader.ListCourseTeamCourseIDs(id)
	if err != nil {
		return nil, err
	}
	allowed := make(map[string]bool, len(ids))
	for _, courseID := range ids {
		allowed[courseID] = true
	}
	return allowed, nil
}

func (u *UseCaseImpl) GetClass(teacherID string, role models.Role, classID string) (*teacherclass.ClassDTO, error) {
	inv, err := u.requireTeaching(teacherID, role, classID)
	if err != nil {
		return nil, err
	}
	dto := u.toDTO(inv)
	u.fillOwnerName(dto)
	if role == models.Teacher && inv.OwnerTeacherID != teacherID {
		terms, err := u.coTeacherTerms(teacherID)
		if err != nil {
			return nil, err
		}
		if until, ok := terms[inv.ID]; ok {
			dto.CoTeacherUntil = &until
		}
	}
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
	if inv.LmsLinked {
		cohortID, _ := strconv.Atoi(inv.EdxCohortID)
		_, err = u.edx.PatchCohort(inv.EdxCourseID, cohortID, map[string]interface{}{"name": displayName})
		if err != nil {
			return nil, fmt.Errorf("%w: rename cohort", teacherclass.ErrInternal)
		}
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
		LmsLinked:   inv.LmsLinked,
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

// addToClass puts the user on the roster: class_members for a ЛК class, otherwise the
// course enrollment and the Open edX cohort.
func (u *UseCaseImpl) addToClass(inv *models.ClassInviteDB, userID string) error {
	id, err := lmsdb.ParseLMSUserID(userID)
	if err != nil {
		return teacherclass.ErrBadRequest
	}
	if !inv.LmsLinked {
		return u.gw.AddMember(inv.ID, strconv.FormatInt(id, 10))
	}
	username, err := lmsUsername(id)
	if err != nil {
		return err
	}
	if err := u.enrollAndAdd(username, inv.EdxCourseID, inv.EdxCohortID); err != nil {
		return fmt.Errorf("%w: join cohort: %v", teacherclass.ErrInternal, err)
	}
	return nil
}

// lmsUsername is the user's LMS login. A variable so tests can stand in for LMS MySQL.
var lmsUsername = func(id int64) (string, error) {
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		return "", err
	}
	defer reader.Close()
	user, err := reader.LookupAuthUserByID(id)
	if err != nil || user == nil {
		return "", teacherclass.ErrBadRequest
	}
	return user.Username, nil
}

func (u *UseCaseImpl) JoinClass(studentID string, codeOrSlug string) (*teacherclass.ClassDTO, error) {
	inv, err := u.resolveInvite(codeOrSlug)
	if err != nil {
		return nil, err
	}
	if !inv.IsActive {
		return nil, teacherclass.ErrNotFound
	}
	if was, err := u.gw.WasCoTeacher(inv.ID, studentID); err != nil {
		return nil, err
	} else if was {
		return nil, teacherclass.ErrCoTeacherCannotJoin
	}
	dto, err, _ := u.joins.Do(inv.ID+":"+studentID, func() (interface{}, error) {
		return u.join(studentID, inv)
	})
	if err != nil {
		return nil, err
	}
	return dto.(*teacherclass.ClassDTO), nil
}

func (u *UseCaseImpl) join(studentID string, inv *models.ClassInviteDB) (*teacherclass.ClassDTO, error) {
	// Opening the invite link again is not news to the teacher.
	wasMember, err := u.isClassMember(studentID, inv)
	if err != nil {
		log.Printf("teacherclass: class %s membership: %v", inv.ID, err)
		wasMember = true
	}
	if err := u.addToClass(inv, studentID); err != nil {
		return nil, err
	}
	// Auto-issue open assignments to late joiner
	assignments, _ := u.gw.ListAssignments(inv.EdxCourseID, inv.EdxCohortID)
	for i := range assignments {
		a := &assignments[i]
		if a.ClosedAt != nil {
			continue
		}
		if _, err := u.gw.FindStudentAssignmentProject(a.ID, studentID); err == nil {
			continue
		}
		tpl, err := u.assignmentTemplate(a)
		if err != nil {
			continue
		}
		_, _ = u.gw.ForkProjectForStudent(tpl, studentID, a.Title, a)
	}
	if !wasMember {
		u.notifyStudentJoined(studentID, inv)
	}
	dto := u.toDTO(inv)
	u.fillOwnerName(dto)
	return dto, nil
}

// LeaveClass takes the student off the roster. The course enrollment of a linked class stays,
// and so do the student's copies: the teacher still reviews handed-in work, and joining again
// picks the copies up.
func (u *UseCaseImpl) LeaveClass(studentID, classID string) error {
	inv, err := u.gw.GetInviteByID(classID)
	if err != nil {
		return teacherclass.ErrNotFound
	}
	member, err := u.isClassMember(studentID, inv)
	if err != nil {
		return err
	}
	if !member {
		return teacherclass.ErrNotFound
	}
	if err := u.removeFromRoster(studentID, inv); err != nil {
		return err
	}
	u.notifyStudentLeft(studentID, inv)
	return nil
}

func (u *UseCaseImpl) removeFromRoster(studentID string, inv *models.ClassInviteDB) error {
	if !inv.LmsLinked {
		return u.gw.RemoveMember(inv.ID, studentID)
	}
	id, err := lmsdb.ParseLMSUserID(studentID)
	if err != nil {
		return teacherclass.ErrBadRequest
	}
	username, err := lmsUsername(id)
	if err != nil {
		return err
	}
	cohortID, _ := strconv.Atoi(inv.EdxCohortID)
	if err := u.edx.RemoveCohortUser(username, inv.EdxCourseID, cohortID); err != nil {
		return fmt.Errorf("%w: leave cohort: %v", teacherclass.ErrInternal, err)
	}
	return nil
}

// classNotification is a row about the class that opens its board.
func classNotification(recipientID string, inv *models.ClassInviteDB, kind, dedupeKey string) *models.UserNotificationDB {
	actionURL := "/teacher/classes/" + inv.ID
	return &models.UserNotificationDB{
		RecipientUserID: recipientID,
		Kind:            kind,
		Severity:        "INFO",
		Source:          "system",
		ActionURL:       &actionURL,
		DedupeKey:       &dedupeKey,
	}
}

// teacherNotifications are rows for the owner and the co-teachers, without the student
// themselves; the roster has already changed by the time they are sent, so a failure to send one
// is only logged. dedupe_key is unique across recipients, so a co-teacher's row gets their own:
// with the owner's key it would take the row away from the owner.
func (u *UseCaseImpl) teacherNotifications(studentID string, inv *models.ClassInviteDB, kind, dedupeKey string) []*models.UserNotificationDB {
	var rows []*models.UserNotificationDB
	if inv.OwnerTeacherID != studentID {
		rows = append(rows, classNotification(inv.OwnerTeacherID, inv, kind, dedupeKey))
	}
	coTeachers, err := u.gw.ListCoTeachers(inv.ID)
	if err != nil {
		log.Printf("teacherclass: class %s co-teachers: %v", inv.ID, err)
	}
	for i := range coTeachers {
		if id := coTeachers[i].TeacherID; id != studentID {
			rows = append(rows, classNotification(id, inv, kind, dedupeKey+":"+id))
		}
	}
	return rows
}

func studentName(studentID string) string {
	if name := ownerDisplayName(studentID); name != "" {
		return name
	}
	return "Ученик"
}

// notifyStudentJoined: one row per class and UTC day, a class joining on its first lesson would
// otherwise fill the bell. It counts joins, not people: JoinClass calls it for someone who was
// off the roster, so a student who left and came back appears twice, after the "left" row.
func (u *UseCaseImpl) notifyStudentJoined(studentID string, inv *models.ClassInviteDB) {
	day := time.Now().UTC().Format("2006-01-02")
	for _, row := range u.teacherNotifications(studentID, inv, "class_members_joined", "class_members_joined:"+inv.ID+":"+day) {
		if err := u.notify.AddToDigest(row, "Вступлений в класс: ", "«"+inv.DisplayName+"»: ", studentName(studentID)); err != nil {
			log.Printf("teacherclass: notify class %s teacher: %v", inv.ID, err)
		}
	}
}

// notifyStudentLeft: one row per student and class, leaving again refreshes it.
func (u *UseCaseImpl) notifyStudentLeft(studentID string, inv *models.ClassInviteDB) {
	for _, row := range u.teacherNotifications(studentID, inv, "class_member_left", "class_member_left:"+inv.ID+":"+studentID) {
		row.Title = "Ученик покинул класс"
		row.Body = fmt.Sprintf("%s больше не в классе «%s». Сданные работы остались на доске класса.", studentName(studentID), inv.DisplayName)
		if err := u.notify.CreateOrUpdateByDedupe(row); err != nil {
			log.Printf("teacherclass: notify class %s teacher: %v", inv.ID, err)
		}
	}
}

func (u *UseCaseImpl) ListStudentClasses(studentID string) ([]teacherclass.ClassDTO, error) {
	if _, err := lmsdb.ParseLMSUserID(studentID); err != nil {
		return nil, teacherclass.ErrBadRequest
	}
	localIDs, err := u.gw.ListMemberClassIDs(studentID)
	if err != nil {
		return nil, err
	}
	local := make(map[string]bool, len(localIDs))
	for _, id := range localIDs {
		local[id] = true
	}
	// One query for the student's cohorts instead of one Open edX API call per class.
	// Without the LMS the student still sees the classes the ЛК keeps itself.
	cohorts, err := studentCohortIDs(studentID)
	if err != nil {
		if len(local) == 0 {
			return nil, err
		}
		log.Printf("teacherclass: student %s cohorts: %v", studentID, err)
		cohorts = map[string]bool{}
	}
	out := make([]teacherclass.ClassDTO, 0)
	if len(cohorts) == 0 && len(local) == 0 {
		return out, nil
	}
	all, err := u.gw.ListActiveInvites()
	if err != nil {
		return nil, err
	}
	for i := range all {
		inv := &all[i]
		if inv.LmsLinked && !cohorts[strings.TrimSpace(inv.EdxCohortID)] {
			continue
		}
		if !inv.LmsLinked && !local[inv.ID] {
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
	inv, err := u.requireTeaching(teacherID, role, classID)
	if err != nil {
		return nil, err
	}
	if !inv.LmsLinked {
		return u.listLocalMembers(inv.ID)
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

// listLocalMembers reads the ЛК roster; names and logins come from LMS MySQL when it is up.
func (u *UseCaseImpl) listLocalMembers(classID string) ([]teacherclass.CohortUserDTO, error) {
	rows, err := u.gw.ListMembers(classID)
	if err != nil {
		return nil, err
	}
	out := make([]teacherclass.CohortUserDTO, 0, len(rows))
	reader, rerr := lmsdb.NewReaderFromConfig()
	if rerr != nil {
		reader = nil
	} else if reader != nil {
		defer reader.Close()
	}
	for _, row := range rows {
		dto := teacherclass.CohortUserDTO{UserID: row.UserID}
		if reader != nil {
			if id, err := lmsdb.ParseLMSUserID(row.UserID); err == nil {
				if user, err := reader.LookupAuthUserByID(id); err == nil && user != nil {
					dto.Username = user.Username
					dto.Email = user.Email
					dto.Name = profileName(reader, id)
				}
			}
		}
		out = append(out, dto)
	}
	return out, nil
}

func (u *UseCaseImpl) AddMembersByEmail(teacherID string, role models.Role, classID string, emails []string) (added []string, missing []string, err error) {
	inv, err := u.requireTeaching(teacherID, role, classID)
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
		if inv.LmsLinked {
			err = u.enrollAndAdd(row.Username, inv.EdxCourseID, inv.EdxCohortID)
		} else {
			err = u.gw.AddMember(inv.ID, strconv.FormatInt(row.ID, 10))
		}
		if err != nil {
			missing = append(missing, email)
			continue
		}
		added = append(added, email)
	}
	return added, missing, nil
}

// RemoveMember: member is the LMS username; a ЛК class also takes the user id, so a
// member whose LMS profile could not be read can still be removed.
func (u *UseCaseImpl) RemoveMember(teacherID string, role models.Role, classID, username string) error {
	inv, err := u.requireTeaching(teacherID, role, classID)
	if err != nil {
		return err
	}
	if !inv.LmsLinked {
		return u.removeLocalMember(inv.ID, username)
	}
	cohortID, _ := strconv.Atoi(inv.EdxCohortID)
	return u.edx.RemoveCohortUser(username, inv.EdxCourseID, cohortID)
}

func (u *UseCaseImpl) removeLocalMember(classID, member string) error {
	member = strings.TrimSpace(member)
	if member == "" {
		return teacherclass.ErrBadRequest
	}
	if ok, err := u.gw.IsMemberOfAny(member, []string{classID}); err != nil {
		return err
	} else if ok {
		return u.gw.RemoveMember(classID, member)
	}
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		return err
	}
	defer reader.Close()
	row, err := reader.LookupAuthUserByUsername(member)
	if err != nil || row == nil {
		return teacherclass.ErrNotFound
	}
	return u.gw.RemoveMember(classID, strconv.FormatInt(row.ID, 10))
}

func assignmentDTO(a *models.ClassAssignmentDB) teacherclass.AssignmentDTO {
	kind := models.AssignmentScratch
	var maxFiles int
	if a.IsFileAssignment() {
		kind = models.AssignmentFile
		maxFiles = maxSubmissionFiles()
	}
	allowed := []string{}
	allowed = append(allowed, a.AllowedKinds...)
	return teacherclass.AssignmentDTO{
		Kind:               kind,
		AllowedKinds:       allowed,
		MaxFiles:           maxFiles,
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
	inv, err := u.requireTeaching(teacherID, role, classID)
	if err != nil {
		return nil, err
	}
	in.Title = strings.TrimSpace(in.Title)
	in.AssignmentKey = strings.TrimSpace(in.AssignmentKey)
	in.TemplateProjectID = strings.TrimSpace(in.TemplateProjectID)
	if in.Title == "" || in.AssignmentKey == "" {
		return nil, teacherclass.ErrBadRequest
	}
	a := &models.ClassAssignmentDB{
		EdxCourseID:        inv.EdxCourseID,
		EdxCohortID:        inv.EdxCohortID,
		CreatedByTeacherID: teacherID,
		Title:              in.Title,
		Instructions:       in.Instructions,
		AssignmentKey:      in.AssignmentKey,
		DueAt:              in.DueAt,
	}
	switch in.Kind {
	case "", models.AssignmentScratch:
	case models.AssignmentFile:
		// Students hand in files later: nothing is issued now.
		starter, err := u.prepareFileAssignment(teacherID, in, a)
		if err != nil {
			return nil, err
		}
		if err := u.gw.CreateAssignment(a); err != nil {
			return nil, err
		}
		dto := assignmentDTO(a)
		if starter != nil {
			ref := teacherclass.FileRef(starter)
			dto.TemplateFile = &ref
		}
		return &dto, nil
	default:
		return nil, teacherclass.ErrBadRequest
	}
	// The template is optional: without one every student starts from an empty project.
	var tpl *models.ScratchProjectDB
	if in.TemplateProjectID != "" {
		tpl, err = u.gw.GetProject(in.TemplateProjectID)
		if err != nil {
			return nil, teacherclass.ErrNotFound
		}
		if tpl.OwnerUserID != teacherID {
			return nil, teacherclass.ErrForbidden
		}
		a.TemplateProjectID = &in.TemplateProjectID
	}
	if err := u.gw.CreateAssignment(a); err != nil {
		return nil, err
	}
	members, err := u.ListMembers(teacherID, role, classID)
	if err != nil {
		return nil, err
	}
	dto := assignmentDTO(a)
	for _, m := range members {
		if m.UserID == "" {
			dto.CopiesSkipped++
			continue
		}
		if _, err := u.gw.ForkProjectForStudent(tpl, m.UserID, in.Title, a); err != nil {
			dto.CopiesFailed++
			log.Printf("teacherclass: assignment %s: copy for student %s: %v", a.ID, m.UserID, err)
			continue
		}
		dto.CopiesIssued++
	}
	return &dto, nil
}

// assignmentTemplate is the project students get a copy of; nil means an empty project.
func (u *UseCaseImpl) assignmentTemplate(a *models.ClassAssignmentDB) (*models.ScratchProjectDB, error) {
	if a.TemplateProjectID == nil || *a.TemplateProjectID == "" {
		return nil, nil
	}
	return u.gw.GetProject(*a.TemplateProjectID)
}

func (u *UseCaseImpl) ListAssignments(viewerID string, role models.Role, classID string) ([]teacherclass.AssignmentDTO, error) {
	var inv *models.ClassInviteDB
	var err error
	if role == models.Teacher || role == models.SuperAdmin {
		inv, err = u.gw.GetInviteByID(classID)
		if err != nil {
			return nil, teacherclass.ErrNotFound
		}
		if role == models.Teacher {
			teaches, err := u.canTeach(inv, viewerID, role)
			if err != nil {
				return nil, err
			}
			if !teaches {
				return nil, teacherclass.ErrForbidden
			}
		}
	} else {
		inv, err = u.gw.GetInviteByID(classID)
		if err != nil {
			return nil, teacherclass.ErrNotFound
		}
		member, err := u.isClassMember(viewerID, inv)
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
		out = append(out, assignmentDTO(&list[i]))
	}
	if err := u.fillTemplateFiles(out, list); err != nil {
		return nil, err
	}
	for i := range list {
		dto := &out[i]
		if studentView && list[i].IsFileAssignment() {
			if err := u.fillMyFileWork(dto, list[i].ID, viewerID); err != nil {
				return nil, err
			}
		} else if studentView {
			if p, err := u.gw.FindStudentAssignmentProject(list[i].ID, viewerID); err == nil && p != nil {
				dto.MyProjectID = p.ID
				dto.MyReviewStatus = deref(p.ReviewStatus)
				dto.MyReviewComment = deref(p.ReviewComment)
				dto.MySubmissionNote = p.SubmissionNote
			}
		}
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
	inv, err := u.requireTeaching(teacherID, role, classID)
	if err != nil {
		return 0, err
	}
	a, err := u.gw.GetAssignment(assignmentID)
	if err != nil {
		return 0, teacherclass.ErrNotFound
	}
	if a.EdxCohortID != inv.EdxCohortID {
		return 0, teacherclass.ErrForbidden
	}
	if a.ClosedAt != nil {
		return 0, teacherclass.ErrAssignmentClosed
	}
	// A file submission starts with the student's first file: nothing to issue.
	if a.IsFileAssignment() {
		return 0, teacherclass.ErrBadRequest
	}
	tpl, err := u.assignmentTemplate(a)
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

func (u *UseCaseImpl) SetAssignmentClosed(teacherID string, role models.Role, classID, assignmentID string, closed bool) (*teacherclass.AssignmentDTO, error) {
	inv, err := u.requireTeaching(teacherID, role, classID)
	if err != nil {
		return nil, err
	}
	a, err := u.gw.GetAssignment(assignmentID)
	if err != nil || a.EdxCohortID != inv.EdxCohortID {
		return nil, teacherclass.ErrNotFound
	}
	if err := u.gw.SetAssignmentClosed(a.ID, closed); err != nil {
		return nil, err
	}
	if a, err = u.gw.GetAssignment(a.ID); err != nil {
		return nil, err
	}
	dto := assignmentDTO(a)
	return &dto, nil
}

// openAssignmentOf is an assignment that still takes the student's work.
func (u *UseCaseImpl) openAssignmentOf(studentID, assignmentID string) (*models.ClassAssignmentDB, error) {
	a, err := u.gw.GetAssignment(assignmentID)
	if err != nil {
		return nil, teacherclass.ErrNotFound
	}
	// The due date only marks late work; closing the assignment stops hand-ins.
	if a.ClosedAt != nil {
		return nil, teacherclass.ErrAssignmentClosed
	}
	// The copy outlives membership: a student who left keeps it but hands in nothing.
	inv, err := u.gw.GetInviteByCourseCohort(a.EdxCourseID, a.EdxCohortID)
	if err != nil {
		return nil, teacherclass.ErrNotFound
	}
	member, err := u.isClassMember(studentID, inv)
	if err != nil {
		return nil, err
	}
	if !member {
		return nil, teacherclass.ErrForbidden
	}
	return a, nil
}

func (u *UseCaseImpl) UpdateProjectNote(studentID, projectID, note string) error {
	note, err := submissionNote(note)
	if err != nil {
		return err
	}
	p, err := u.gw.GetProject(projectID)
	if err != nil {
		return teacherclass.ErrNotFound
	}
	if p.OwnerUserID != studentID {
		return teacherclass.ErrForbidden
	}
	if p.AssignmentID == nil || strings.TrimSpace(*p.AssignmentID) == "" {
		return teacherclass.ErrBadRequest
	}
	if _, err := u.openAssignmentOf(studentID, *p.AssignmentID); err != nil {
		return err
	}
	return u.gw.SetProjectNote(projectID, note)
}

func (u *UseCaseImpl) SubmitAssignment(studentID, projectID, note string) error {
	note, err := submissionNote(note)
	if err != nil {
		return err
	}
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
	// Handing in again would put reviewed work back on review and drop the teacher's comment.
	switch deref(p.ReviewStatus) {
	case "", models.ReviewDraft, models.ReviewReturned:
	default:
		return teacherclass.ErrAlreadyHandedIn
	}
	a, err := u.openAssignmentOf(studentID, *p.AssignmentID)
	if err != nil {
		return err
	}
	unedited, err := u.workUnedited(p, a)
	if err != nil {
		return err
	}
	if unedited {
		return teacherclass.ErrWorkNotEdited
	}
	handIn := &teacherclass.HandIn{At: time.Now().UTC(), Note: note}
	if err := u.gw.UpdateReview(projectID, models.ReviewSubmitted, "", "", handIn); err != nil {
		return err
	}
	u.notifyWorkSubmitted(studentID, a)
	return nil
}

// maxSubmissionNoteChars limits the text a student adds to a hand-in.
const maxSubmissionNoteChars = 2000

// submissionNote cleans up the student's text; Postgres TEXT cannot hold a NUL.
func submissionNote(note string) (string, error) {
	note = strings.TrimSpace(note)
	if utf8.RuneCountInString(note) > maxSubmissionNoteChars || strings.ContainsRune(note, 0) {
		return "", teacherclass.ErrBadRequest
	}
	return note, nil
}

// reviewStatusAllowed: Submitted puts reviewed work back in the queue. Draft is the
// student's own state: set by the teacher it would hide the work from the teacher for good.
func reviewStatusAllowed(status string) bool {
	switch status {
	case models.ReviewAccepted, models.ReviewReturned, models.ReviewShownInClass, models.ReviewSubmitted:
		return true
	}
	return false
}

func (u *UseCaseImpl) ReviewSubmission(teacherID string, role models.Role, projectID, status, comment string) error {
	if !reviewStatusAllowed(status) {
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
	if err != nil {
		return teacherclass.ErrForbidden
	}
	if teaches, err := u.canTeach(inv, teacherID, role); err != nil {
		return err
	} else if !teaches {
		return teacherclass.ErrForbidden
	}
	if !draftVisible(role, p) {
		return teacherclass.ErrNotFound
	}
	if err := u.gw.UpdateReview(projectID, status, comment, teacherID, nil); err != nil {
		return err
	}
	u.notifyWorkReviewed(p.OwnerUserID, p.ID, p.Title, status, comment)
	return nil
}

// draftVisible: a SuperAdmin sees every copy; the teacher only handed-in work (as access.Resolve).
func draftVisible(role models.Role, p *models.ScratchProjectDB) bool {
	return role == models.SuperAdmin || models.IsHandedIn(p.ReviewStatus)
}

// draftVisibleStatus is draftVisible for a file submission.
func draftVisibleStatus(role models.Role, status string) bool {
	return role == models.SuperAdmin || models.IsHandedIn(&status)
}

// isLate: the work was handed in after the assignment's due date.
func isLate(a *models.ClassAssignmentDB, p *models.ScratchProjectDB) bool {
	return lateAt(a, p.SubmittedAt)
}

func lateAt(a *models.ClassAssignmentDB, submittedAt *time.Time) bool {
	return a != nil && a.DueAt != nil && submittedAt != nil && submittedAt.After(*a.DueAt)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (u *UseCaseImpl) ListSubmissions(teacherID string, role models.Role, classID, assignmentID string) ([]teacherclass.SubmissionDTO, error) {
	inv, err := u.requireTeaching(teacherID, role, classID)
	if err != nil {
		return nil, err
	}
	a, err := u.gw.GetAssignment(assignmentID)
	if err != nil || a.EdxCohortID != inv.EdxCohortID {
		return nil, teacherclass.ErrNotFound
	}
	if a.IsFileAssignment() {
		return u.fileSubmissionDTOs(role, a)
	}
	list, err := u.gw.ListSubmissions(assignmentID)
	if err != nil {
		return nil, err
	}
	out := make([]teacherclass.SubmissionDTO, 0, len(list))
	for i := range list {
		p := &list[i]
		status := models.ReviewDraft
		if p.ReviewStatus != nil && *p.ReviewStatus != "" {
			status = *p.ReviewStatus
		}
		dto := teacherclass.SubmissionDTO{
			OwnerUserID:  p.OwnerUserID,
			Title:        p.Title,
			ReviewStatus: status,
			UpdatedAt:    p.UpdatedAt,
		}
		// The lesson board shows who is working: a draft the teacher may not open still
		// carries its status and last save, without the project id that would lead to it.
		if draftVisible(role, p) {
			dto.ProjectID = p.ID
			dto.ReviewComment = deref(p.ReviewComment)
			dto.SubmissionNote = p.SubmissionNote
			dto.SubmittedAt = p.SubmittedAt
			dto.PreviewURL = "/projectPage/" + p.ID + "/preview"
			dto.Late = isLate(a, p)
			dto.ReviewedBy = deref(p.ReviewedByTeacherID)
			dto.ReviewedAt = p.ReviewedAt
		}
		out = append(out, dto)
	}
	fillReviewerNames(out)
	return out, nil
}

// fillReviewerNames looks each teacher up once, however many works they reviewed.
func fillReviewerNames(dtos []teacherclass.SubmissionDTO) {
	names := make(map[string]string)
	for i := range dtos {
		id := dtos[i].ReviewedBy
		if id == "" {
			continue
		}
		name, ok := names[id]
		if !ok {
			name = ownerDisplayName(id)
			names[id] = name
		}
		dtos[i].ReviewedByName = name
	}
}

// work is one student's work on an assignment for ProgressMatrix: a Scratch copy or a file
// submission. The ids stay empty for a draft the teacher may not open.
type work struct {
	owner, status           string
	projectID, submissionID string
	late                    bool
}

// ProgressMatrix returns each assignment's status per project owner. It does not call
// Open edX: the board joins these rows with the roster it already loaded from /members.
func (u *UseCaseImpl) ProgressMatrix(teacherID string, role models.Role, classID string) ([]teacherclass.AssignmentDTO, []teacherclass.ProgressRowDTO, error) {
	inv, err := u.requireTeaching(teacherID, role, classID)
	if err != nil {
		return nil, nil, err
	}
	assignments, err := u.gw.ListAssignments(inv.EdxCourseID, inv.EdxCohortID)
	if err != nil {
		return nil, nil, err
	}
	assignDTOs := make([]teacherclass.AssignmentDTO, 0, len(assignments))
	ids := make([]string, 0, len(assignments))
	for i := range assignments {
		assignDTOs = append(assignDTOs, assignmentDTO(&assignments[i]))
		ids = append(ids, assignments[i].ID)
	}
	subs, err := u.gw.ListSubmissionsForAssignments(ids)
	if err != nil {
		return nil, nil, err
	}
	byAssignment, err := u.fileWork(role, assignments)
	if err != nil {
		return nil, nil, err
	}
	byID := map[string]*models.ClassAssignmentDB{}
	for i := range assignments {
		byID[assignments[i].ID] = &assignments[i]
	}
	for j := range subs {
		p := &subs[j]
		if p.AssignmentID == nil {
			continue
		}
		status := models.ReviewDraft
		if p.ReviewStatus != nil && *p.ReviewStatus != "" {
			status = *p.ReviewStatus
		}
		w := work{owner: p.OwnerUserID, status: status}
		// A draft the teacher may not open shows as "draft" without a link to it.
		if draftVisible(role, p) {
			w.projectID = p.ID
			w.late = isLate(byID[*p.AssignmentID], p)
		}
		byAssignment[*p.AssignmentID] = append(byAssignment[*p.AssignmentID], w)
	}
	rows := []teacherclass.ProgressRowDTO{}
	rowOf := map[string]int{}
	for i := range assignments {
		a := &assignments[i]
		seen := map[string]struct{}{}
		// Newest first: the latest project wins if a student somehow has two.
		for _, w := range byAssignment[a.ID] {
			if _, ok := seen[w.owner]; ok {
				continue
			}
			seen[w.owner] = struct{}{}
			cell := teacherclass.ProgressCellDTO{
				AssignmentID: a.ID,
				Status:       w.status,
				ProjectID:    w.projectID,
				SubmissionID: w.submissionID,
				Late:         w.late,
			}
			idx, ok := rowOf[w.owner]
			if !ok {
				idx = len(rows)
				rowOf[w.owner] = idx
				rows = append(rows, teacherclass.ProgressRowDTO{UserID: w.owner})
			}
			rows[idx].Cells = append(rows[idx].Cells, cell)
		}
	}
	return assignDTOs, rows, nil
}
