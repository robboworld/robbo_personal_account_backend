package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
	"gorm.io/gorm"
)

// coTeacherGateway adds terms to the in-memory ЛК classes.
type coTeacherGateway struct {
	*localGateway
	terms []*models.ClassCoTeacherDB
}

func (g *coTeacherGateway) running(classID string) []*models.ClassCoTeacherDB {
	var out []*models.ClassCoTeacherDB
	for _, row := range g.terms {
		if row.ClassID == classID && row.RevokedAt == nil && row.ExpiresAt.After(time.Now()) {
			out = append(out, row)
		}
	}
	return out
}

func (g *coTeacherGateway) AddCoTeacher(row *models.ClassCoTeacherDB, maxActive int) error {
	running := g.running(row.ClassID)
	for _, r := range running {
		if r.TeacherID == row.TeacherID {
			return teacherclass.ErrAlreadyCoTeacher
		}
	}
	if len(running) >= maxActive {
		return teacherclass.ErrTooManyCoTeachers
	}
	g.terms = append(g.terms, row)
	return nil
}

func (g *coTeacherGateway) IsCoTeacher(classID, teacherID string) (bool, error) {
	for _, r := range g.running(classID) {
		if r.TeacherID == teacherID {
			return true, nil
		}
	}
	return false, nil
}

func (g *coTeacherGateway) WasCoTeacher(classID, teacherID string) (bool, error) {
	for _, r := range g.terms {
		if r.ClassID == classID && r.TeacherID == teacherID {
			return true, nil
		}
	}
	return false, nil
}

func (g *coTeacherGateway) ListCoTeachers(classID string) ([]models.ClassCoTeacherDB, error) {
	var out []models.ClassCoTeacherDB
	for _, r := range g.running(classID) {
		out = append(out, *r)
	}
	return out, nil
}

func (g *coTeacherGateway) ListCoTeachings(teacherID string) ([]models.ClassCoTeacherDB, error) {
	var out []models.ClassCoTeacherDB
	for classID := range g.invites {
		for _, r := range g.running(classID) {
			if r.TeacherID == teacherID {
				out = append(out, *r)
			}
		}
	}
	return out, nil
}

func (g *coTeacherGateway) ListInvitesByTeacher(teacherID string) ([]models.ClassInviteDB, error) {
	var out []models.ClassInviteDB
	for id, inv := range g.invites {
		if is, _ := g.IsCoTeacher(id, teacherID); is || inv.OwnerTeacherID == teacherID {
			out = append(out, *inv)
		}
	}
	return out, nil
}

func (g *coTeacherGateway) SetCoTeacherExpiry(classID, teacherID string, expiresAt time.Time, remindedAt *time.Time) (*models.ClassCoTeacherDB, error) {
	for _, r := range g.running(classID) {
		if r.TeacherID == teacherID {
			r.ExpiresAt, r.RemindedAt = expiresAt, remindedAt
			return r, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (g *coTeacherGateway) RevokeCoTeacher(classID, teacherID, revokedBy string) error {
	for _, r := range g.running(classID) {
		if r.TeacherID == teacherID {
			now := time.Now()
			r.RevokedAt, r.RevokedBy = &now, &revokedBy
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (g *coTeacherGateway) ClaimExpiringCoTeachers(before time.Time) ([]models.ClassCoTeacherDB, error) {
	var out []models.ClassCoTeacherDB
	for _, r := range g.terms {
		if r.RevokedAt == nil && r.RemindedAt == nil && r.ExpiresAt.After(time.Now()) && !r.ExpiresAt.After(before) {
			now := time.Now()
			r.RemindedAt = &now
			out = append(out, *r)
		}
	}
	return out, nil
}

// Teachers of the LMS in these tests: 20 and 21; 30 is a student, the owner is 10.
func withLMSTeachers(t *testing.T) {
	t.Helper()
	prev := lookupTeacherByEmail
	lookupTeacherByEmail = func(email string) (string, error) {
		return map[string]string{"sub@x.ru": "20", "sub2@x.ru": "21", "sub3@x.ru": "22", "sub4@x.ru": "23", "owner@x.ru": "10"}[email], nil
	}
	t.Cleanup(func() { lookupTeacherByEmail = prev })
}

func coTeacherClass(t *testing.T) (*UseCaseImpl, *coTeacherGateway, *leaveNotifier, string) {
	t.Helper()
	withoutLMSNames(t)
	withLMSTeachers(t)
	gw := &coTeacherGateway{localGateway: newLocalGateway()}
	notify := &leaveNotifier{}
	uc := &UseCaseImpl{gw: gw, notify: notify}
	dto, err := uc.CreateClass("10", models.Teacher, teacherclass.CreateClassInput{DisplayName: "5 А"})
	if err != nil {
		t.Fatal(err)
	}
	return uc, gw, notify, dto.ID
}

func inDays(n int) time.Time { return time.Now().UTC().AddDate(0, 0, n) }

func appoint(t *testing.T, uc *UseCaseImpl, classID string) {
	t.Helper()
	if _, err := uc.AddCoTeacher("10", models.Teacher, classID, "Sub@x.ru ", inDays(14)); err != nil {
		t.Fatal(err)
	}
}

func TestCoTeacherRunsTheClassNotItsSettings(t *testing.T) {
	uc, gw, _, id := coTeacherClass(t)
	appoint(t, uc, id)
	gw.members[id] = map[string]bool{"797": true}

	dto, err := uc.GetClass("20", models.Teacher, id)
	if err != nil {
		t.Fatal(err)
	}
	if dto.CoTeacherUntil == nil || dto.InviteCode == "" {
		t.Fatalf("a co-teacher invites students and sees the term: %+v", dto)
	}
	if _, err := uc.ListMembers("20", models.Teacher, id); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.ListAssignments("20", models.Teacher, id); err != nil {
		t.Fatal(err)
	}
	if err := uc.RemoveMember("20", models.Teacher, id, "797"); err != nil {
		t.Fatalf("a co-teacher removes students: %v", err)
	}

	forbidden := map[string]func() error{
		"rename":  func() error { _, err := uc.RenameClass("20", models.Teacher, id, "x"); return err },
		"archive": func() error { return uc.ArchiveClass("20", models.Teacher, id) },
		"rotate":  func() error { _, err := uc.RotateInvite("20", models.Teacher, id); return err },
		"appoint": func() error {
			_, err := uc.AddCoTeacher("20", models.Teacher, id, "sub2@x.ru", inDays(3))
			return err
		},
		"extend": func() error {
			_, err := uc.ExtendCoTeacher("20", models.Teacher, id, "20", inDays(30))
			return err
		},
	}
	for name, call := range forbidden {
		if err := call(); !errors.Is(err, teacherclass.ErrForbidden) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

func TestStrangerAndEndedTermGetNoClass(t *testing.T) {
	uc, gw, _, id := coTeacherClass(t)
	appoint(t, uc, id)

	if _, err := uc.GetClass("21", models.Teacher, id); !errors.Is(err, teacherclass.ErrForbidden) {
		t.Fatalf("stranger: %v", err)
	}
	// A student never gets the right, whatever the table says.
	if _, err := uc.GetClass("20", models.Student, id); !errors.Is(err, teacherclass.ErrForbidden) {
		t.Fatalf("not a teacher any more: %v", err)
	}

	gw.terms[0].ExpiresAt = time.Now().Add(-time.Minute)
	if _, err := uc.GetClass("20", models.Teacher, id); !errors.Is(err, teacherclass.ErrForbidden) {
		t.Fatalf("ended term: %v", err)
	}
	if list, _ := uc.ListMyClasses("20", models.Teacher); len(list) != 0 {
		t.Fatalf("ended term still listed: %+v", list)
	}
}

func TestAddCoTeacherRefusals(t *testing.T) {
	uc, _, notify, id := coTeacherClass(t)
	cases := []struct {
		name, email string
		until       time.Time
		want        error
	}{
		{"past", "sub@x.ru", inDays(-1), teacherclass.ErrBadCoTeacherTerm},
		{"too long", "sub@x.ru", inDays(defaultCoTeacherMaxDays + 1), teacherclass.ErrBadCoTeacherTerm},
		{"not a teacher", "student@x.ru", inDays(7), teacherclass.ErrNotATeacher},
		{"no email", " ", inDays(7), teacherclass.ErrBadRequest},
		{"owner", "owner@x.ru", inDays(7), teacherclass.ErrAlreadyCoTeacher},
	}
	for _, c := range cases {
		if _, err := uc.AddCoTeacher("10", models.Teacher, id, c.email, c.until); !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v", c.name, err)
		}
	}
	if len(notify.sent) != 0 {
		t.Fatalf("a refused appointment notified: %+v", notify.sent)
	}

	appoint(t, uc, id)
	if _, err := uc.AddCoTeacher("10", models.Teacher, id, "sub@x.ru", inDays(7)); !errors.Is(err, teacherclass.ErrAlreadyCoTeacher) {
		t.Fatalf("twice: %v", err)
	}
	for _, email := range []string{"sub2@x.ru", "sub3@x.ru"} {
		if _, err := uc.AddCoTeacher("10", models.Teacher, id, email, inDays(7)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := uc.AddCoTeacher("10", models.Teacher, id, "sub4@x.ru", inDays(7)); !errors.Is(err, teacherclass.ErrTooManyCoTeachers) {
		t.Fatalf("over the limit: %v", err)
	}
}

func TestSuperAdminAppointsAndOwnerHears(t *testing.T) {
	uc, gw, notify, id := coTeacherClass(t)
	if _, err := uc.AddCoTeacher("1", models.SuperAdmin, id, "sub@x.ru", inDays(7)); err != nil {
		t.Fatal(err)
	}
	if gw.terms[0].GrantedBy != "1" {
		t.Fatalf("granted by %s", gw.terms[0].GrantedBy)
	}
	if len(notify.sent) != 2 || *notify.sent[0].DedupeKey == *notify.sent[1].DedupeKey {
		t.Fatalf("both sides, each with their own row: %+v", notify.sent)
	}
}

func TestRemoveCoTeacher(t *testing.T) {
	uc, gw, notify, id := coTeacherClass(t)
	appoint(t, uc, id)
	notify.sent = nil

	if err := uc.RemoveCoTeacher("21", models.Teacher, id, "20"); !errors.Is(err, teacherclass.ErrForbidden) {
		t.Fatalf("another teacher: %v", err)
	}
	if err := uc.RemoveCoTeacher("20", models.Teacher, id, "20"); err != nil {
		t.Fatalf("stepping down: %v", err)
	}
	if len(notify.sent) != 1 || notify.sent[0].RecipientUserID != "10" {
		t.Fatalf("stepping down tells the owner: %+v", notify.sent)
	}
	if *gw.terms[0].RevokedBy != "20" {
		t.Fatalf("revoked by %v", gw.terms[0].RevokedBy)
	}
	if _, err := uc.GetClass("20", models.Teacher, id); !errors.Is(err, teacherclass.ErrForbidden) {
		t.Fatalf("access is gone at once: %v", err)
	}
	if err := uc.RemoveCoTeacher("10", models.Teacher, id, "20"); !errors.Is(err, teacherclass.ErrNotFound) {
		t.Fatalf("nothing left to end: %v", err)
	}

	appoint(t, uc, id)
	notify.sent = nil
	if err := uc.RemoveCoTeacher("10", models.Teacher, id, "20"); err != nil {
		t.Fatal(err)
	}
	if len(notify.sent) != 1 || notify.sent[0].RecipientUserID != "20" {
		t.Fatalf("the owner's removal tells the co-teacher: %+v", notify.sent)
	}
}

func TestExtendCoTeacher(t *testing.T) {
	uc, gw, notify, id := coTeacherClass(t)
	appoint(t, uc, id)
	reminded := time.Now()
	gw.terms[0].RemindedAt = &reminded
	notify.sent = nil

	dto, err := uc.ExtendCoTeacher("10", models.Teacher, id, "20", inDays(30))
	if err != nil {
		t.Fatal(err)
	}
	if !dto.ExpiresAt.After(time.Now().AddDate(0, 0, 29)) || gw.terms[0].RemindedAt != nil {
		t.Fatalf("a later end needs a new reminder: %+v", gw.terms[0])
	}
	if len(notify.sent) != 1 || notify.sent[0].RecipientUserID != "20" {
		t.Fatalf("notified: %+v", notify.sent)
	}
	if _, err := uc.ExtendCoTeacher("10", models.Teacher, id, "20", inDays(defaultCoTeacherMaxDays+1)); !errors.Is(err, teacherclass.ErrBadCoTeacherTerm) {
		t.Fatalf("too long: %v", err)
	}
	if _, err := uc.ExtendCoTeacher("10", models.Teacher, id, "21", inDays(3)); !errors.Is(err, teacherclass.ErrNotFound) {
		t.Fatalf("no term: %v", err)
	}
}

// The invite code stays with a co-teacher: it must not bring them back as a student.
func TestCoTeacherCannotJoinByCode(t *testing.T) {
	uc, gw, _, id := coTeacherClass(t)
	appoint(t, uc, id)
	slug := gw.invites[id].InviteSlug

	if _, err := uc.JoinClass("20", slug); !errors.Is(err, teacherclass.ErrCoTeacherCannotJoin) {
		t.Fatalf("running term: %v", err)
	}
	if err := uc.RemoveCoTeacher("10", models.Teacher, id, "20"); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.JoinClass("20", slug); !errors.Is(err, teacherclass.ErrCoTeacherCannotJoin) {
		t.Fatalf("after the term: %v", err)
	}
	if gw.members[id]["20"] {
		t.Fatal("joined")
	}
	if _, err := uc.JoinClass("797", slug); err != nil {
		t.Fatalf("a student joins as before: %v", err)
	}
}

// dedupe_key is unique across recipients: a shared key would move the owner's row to a co-teacher.
func TestClassNewsReachEveryTeacherWithOwnRow(t *testing.T) {
	uc, gw, notify, id := coTeacherClass(t)
	appoint(t, uc, id)
	notify.sent = nil
	inv := gw.invites[id]

	uc.notifyStudentLeft("797", inv)

	if len(notify.sent) != 2 {
		t.Fatalf("notifications = %+v", notify.sent)
	}
	keys := map[string]string{}
	for _, row := range notify.sent {
		keys[row.RecipientUserID] = *row.DedupeKey
	}
	base := "class_member_left:" + id + ":797"
	if keys["10"] != base {
		t.Fatalf("the owner's key is unchanged: %q", keys["10"])
	}
	if keys["20"] != base+":20" {
		t.Fatalf("the co-teacher's key: %q", keys["20"])
	}
}

func TestListMyClassesShowsCoTaught(t *testing.T) {
	uc, _, _, id := coTeacherClass(t)
	appoint(t, uc, id)

	list, err := uc.ListMyClasses("20", models.Teacher)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != id || list[0].CoTeacherUntil == nil {
		t.Fatalf("co-teacher's list: %+v", list)
	}
	list, err = uc.ListMyClasses("10", models.Teacher)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].CoTeacherUntil != nil {
		t.Fatalf("owner's list: %+v", list)
	}
}

func TestRemindExpiringCoTeachersOnce(t *testing.T) {
	uc, gw, notify, id := coTeacherClass(t)
	appoint(t, uc, id)
	notify.sent = nil

	if n, err := uc.RemindExpiringCoTeachers(time.Now()); err != nil || n != 0 {
		t.Fatalf("14 days ahead: %d, %v", n, err)
	}
	gw.terms[0].ExpiresAt = time.Now().Add(20 * time.Hour)
	if n, err := uc.RemindExpiringCoTeachers(time.Now()); err != nil || n != 1 {
		t.Fatalf("ends tomorrow: %d, %v", n, err)
	}
	if len(notify.sent) != 2 {
		t.Fatalf("both sides: %+v", notify.sent)
	}
	if n, _ := uc.RemindExpiringCoTeachers(time.Now()); n != 0 {
		t.Fatalf("reminded twice: %d", n)
	}
}

// A term shorter than the reminder window is not reminded of right after it was granted.
func TestShortTermGetsNoReminder(t *testing.T) {
	uc, gw, _, id := coTeacherClass(t)
	if _, err := uc.AddCoTeacher("10", models.Teacher, id, "sub@x.ru", time.Now().Add(5*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if gw.terms[0].RemindedAt == nil {
		t.Fatal("a short term counts as reminded")
	}
}

func TestReviewRecordsTheCoTeacherAndABoardNamesThem(t *testing.T) {
	uc, gw := newFileUseCase(t)
	gw.coTeachers = map[string]bool{"20": true}
	prev := ownerDisplayName
	ownerDisplayName = func(id string) string { return map[string]string{"20": "Замена"}[id] }
	t.Cleanup(func() { ownerDisplayName = prev })
	subID := submittedFileWork(t, uc, gw)
	assignmentID := gw.subs[subID].AssignmentID

	if err := uc.ReviewFileSubmission("20", models.Teacher, subID, models.ReviewAccepted, "ок"); err != nil {
		t.Fatal(err)
	}
	if got := gw.subs[subID].ReviewedByTeacherID; got == nil || *got != "20" {
		t.Fatalf("reviewer = %v", got)
	}
	list, err := uc.ListSubmissions("20", models.Teacher, "class-1", assignmentID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list = %+v, %v", list, err)
	}
	if list[0].ReviewedBy != "20" || list[0].ReviewedByName != "Замена" {
		t.Fatalf("submission = %+v", list[0])
	}
	if err := uc.ReviewFileSubmission("21", models.Teacher, subID, models.ReviewReturned, ""); !errors.Is(err, teacherclass.ErrForbidden) {
		t.Fatalf("a teacher of no standing: %v", err)
	}
}

func TestHandInClearsTheReviewer(t *testing.T) {
	uc, gw := newFileUseCase(t)
	subID := submittedFileWork(t, uc, gw)
	assignmentID := gw.subs[subID].AssignmentID
	if err := uc.ReviewFileSubmission("10", models.Teacher, subID, models.ReviewReturned, "ещё"); err != nil {
		t.Fatal(err)
	}
	if err := uc.SubmitFiles("797", assignmentID, ""); err != nil {
		t.Fatal(err)
	}
	if got := gw.subs[subID].ReviewedByTeacherID; got != nil {
		t.Fatalf("reviewer of work back on review = %s", *got)
	}
}

func TestScratchReviewRecordsTheReviewer(t *testing.T) {
	withCohortMembers(t, "797")
	withoutLMSNames(t)
	gw := handInGateway()
	uc := &UseCaseImpl{gw: gw, notify: &workNotifier{}}
	if err := uc.SubmitAssignment("797", "p1", ""); err != nil {
		t.Fatal(err)
	}
	if gw.reviewer != "" {
		t.Fatalf("a hand-in has no reviewer: %q", gw.reviewer)
	}
	gw.projects["a1/797"].ReviewStatus = strPtr(models.ReviewSubmitted)
	gw.projects["a1/797"].EdxCohortID = strPtr("7")
	if err := uc.ReviewSubmission("10", models.Teacher, "p1", models.ReviewAccepted, ""); err != nil {
		t.Fatal(err)
	}
	if gw.reviewer != "10" {
		t.Fatalf("reviewer = %q", gw.reviewer)
	}
}
