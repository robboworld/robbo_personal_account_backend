package usecase

import (
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
)

// The other fakes drop the history: their tests are not about it.
func (g *localGateway) AddHistoryEvent(*models.ClassHistoryEventDB) error   { return nil }
func (g *studentGateway) AddHistoryEvent(*models.ClassHistoryEventDB) error { return nil }
func (g *fileGateway) AddHistoryEvent(*models.ClassHistoryEventDB) error    { return nil }

// historyLog keeps what a fake gateway is told to record; err makes the writes fail.
type historyLog struct {
	teacherclass.Gateway
	events []models.ClassHistoryEventDB
	err    error
	// Arguments of the last read.
	actions []string
	before  *teacherclass.HistoryCursor
}

func (g *historyLog) AddHistoryEvent(e *models.ClassHistoryEventDB) error {
	if g.err != nil {
		return g.err
	}
	g.events = append(g.events, *e)
	return nil
}

// UpdateInvite: the fakes hand out the invite itself, the change is already there.
func (g *historyLog) UpdateInvite(*models.ClassInviteDB) error { return nil }

func (g *historyLog) ListHistoryEvents(classID string, actions []string, before *teacherclass.HistoryCursor, limit int) ([]models.ClassHistoryEventDB, error) {
	g.actions, g.before = actions, before
	out := g.events
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (g *historyLog) actionsRecorded() []string {
	out := []string{}
	for i := range g.events {
		out = append(out, g.events[i].Action)
	}
	return out
}

func (g *historyLog) only(t *testing.T, action string) models.ClassHistoryEventDB {
	t.Helper()
	if len(g.events) != 1 || g.events[0].Action != action {
		t.Fatalf("events = %v, want one %s", g.actionsRecorded(), action)
	}
	return g.events[0]
}

// newHistoryClass is ЛК class c1 of teacher 10 with student 797 and co-teacher 20.
func newHistoryClass(t *testing.T) (*UseCaseImpl, *historyLog) {
	t.Helper()
	withoutLMSNames(t)
	local := newLocalGateway()
	local.invites["c1"] = &models.ClassInviteDB{ID: "c1", OwnerTeacherID: "10", DisplayName: "5А", InviteSlug: "s1", IsActive: true}
	local.members["c1"] = map[string]bool{"797": true}
	gw := &coTeacherGateway{localGateway: local}
	if err := gw.AddCoTeacher(&models.ClassCoTeacherDB{ClassID: "c1", TeacherID: "20", ExpiresAt: time.Now().Add(time.Hour)}, 3); err != nil {
		t.Fatal(err)
	}
	log := &historyLog{Gateway: gw}
	return &UseCaseImpl{gw: log, notify: &leaveNotifier{}}, log
}

func TestHistoryTellsWhoActedInTheClass(t *testing.T) {
	cases := []struct {
		name, teacher string
		role          models.Role
		want          string
	}{
		{"owner", "10", models.Teacher, models.HistoryActorOwner},
		{"co-teacher", "20", models.Teacher, models.HistoryActorCoTeacher},
		{"super admin", "99", models.SuperAdmin, models.HistoryActorAdmin},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc, log := newHistoryClass(t)
			if err := uc.RemoveMember(tc.teacher, tc.role, "c1", "797"); err != nil {
				t.Fatal(err)
			}
			e := log.only(t, models.HistoryMemberRemoved)
			if e.ClassID != "c1" || e.ActorUserID != tc.teacher || e.ActorRole != tc.want || e.TargetUserID != "797" {
				t.Fatalf("event = %+v", e)
			}
		})
	}
}

func TestHistoryRefusedActionIsNotRecorded(t *testing.T) {
	uc, log := newHistoryClass(t)
	if err := uc.RemoveMember("555", models.Teacher, "c1", "797"); !errors.Is(err, teacherclass.ErrForbidden) {
		t.Fatalf("stranger err = %v", err)
	}
	if _, err := uc.RenameClass("20", models.Teacher, "c1", "6А"); !errors.Is(err, teacherclass.ErrForbidden) {
		t.Fatalf("co-teacher renames: err = %v", err)
	}
	if len(log.events) != 0 {
		t.Fatalf("events = %v", log.actionsRecorded())
	}
}

func TestHistoryRename(t *testing.T) {
	uc, log := newHistoryClass(t)
	// The form sends the name it was opened with: nothing happened.
	if _, err := uc.RenameClass("10", models.Teacher, "c1", " 5А "); err != nil {
		t.Fatal(err)
	}
	if len(log.events) != 0 {
		t.Fatalf("same name recorded: %v", log.actionsRecorded())
	}
	if _, err := uc.RenameClass("10", models.Teacher, "c1", "6А"); err != nil {
		t.Fatal(err)
	}
	if e := log.only(t, models.HistoryClassRenamed); e.Title != "6А" || e.Detail != "5А" {
		t.Fatalf("event = %+v", e)
	}
}

func TestHistoryInviteRotationKeepsTheCodeOut(t *testing.T) {
	uc, log := newHistoryClass(t)
	dto, err := uc.RotateInvite("10", models.Teacher, "c1")
	if err != nil {
		t.Fatal(err)
	}
	e := log.only(t, models.HistoryInviteRotated)
	if e.Title != "" || e.Detail != "" || dto.InviteCode == "" {
		t.Fatalf("event = %+v", e)
	}
}

func TestHistoryArchive(t *testing.T) {
	uc, log := newHistoryClass(t)
	if err := uc.ArchiveClass("10", models.Teacher, "c1"); err != nil {
		t.Fatal(err)
	}
	log.only(t, models.HistoryClassArchived)
}

func TestHistoryRoster(t *testing.T) {
	uc, log := newHistoryClass(t)

	// Opening the invite link again is not news.
	if _, err := uc.JoinClass("797", "s1"); err != nil {
		t.Fatal(err)
	}
	if len(log.events) != 0 {
		t.Fatalf("second join recorded: %v", log.actionsRecorded())
	}
	if _, err := uc.JoinClass("798", "s1"); err != nil {
		t.Fatal(err)
	}
	if err := uc.LeaveClass("798", "c1"); err != nil {
		t.Fatal(err)
	}
	want := []string{models.HistoryMemberJoined, models.HistoryMemberLeft}
	if got := log.actionsRecorded(); !slices.Equal(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	for _, e := range log.events {
		if e.ActorUserID != "798" || e.ActorRole != models.HistoryActorStudent {
			t.Fatalf("event = %+v", e)
		}
	}
}

func TestHistoryRemoveFromLinkedClass(t *testing.T) {
	withoutLMSNames(t)
	prev := lmsUserID
	lmsUserID = func(username string) string { return map[string]string{"anya": "797"}[username] }
	t.Cleanup(func() { lmsUserID = prev })
	log := &historyLog{Gateway: &studentGateway{}}
	cohort := &cohortEdx{}
	uc := &UseCaseImpl{gw: log, edx: cohort}

	if err := uc.RemoveMember("10", models.Teacher, "class-1", "anya"); err != nil {
		t.Fatal(err)
	}
	if e := log.only(t, models.HistoryMemberRemoved); e.TargetUserID != "797" || e.Detail != "" {
		t.Fatalf("event = %+v", e)
	}

	// The login is all that is left of a user the LMS cannot find, and it must still say who.
	log.events = nil
	if err := uc.RemoveMember("10", models.Teacher, "class-1", "gone"); err != nil {
		t.Fatal(err)
	}
	if e := log.only(t, models.HistoryMemberRemoved); e.TargetUserID != "" || e.Detail != "gone" {
		t.Fatalf("event = %+v", e)
	}

	log.events = nil
	cohort.err = errors.New("dial tcp edx-test.ru")
	if err := uc.RemoveMember("10", models.Teacher, "class-1", "anya"); err == nil {
		t.Fatal("want the Open edX error")
	}
	if len(log.events) != 0 {
		t.Fatalf("a failed removal recorded: %v", log.actionsRecorded())
	}
}

func TestHistoryCoTeachers(t *testing.T) {
	uc, log := newHistoryClass(t)
	prev := lookupTeacherByEmail
	lookupTeacherByEmail = func(string) (string, error) { return "30", nil }
	t.Cleanup(func() { lookupTeacherByEmail = prev })

	until := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Second)
	if _, err := uc.AddCoTeacher("10", models.Teacher, "c1", "new@example.com", until); err != nil {
		t.Fatal(err)
	}
	e := log.only(t, models.HistoryCoTeacherAdded)
	if e.TargetUserID != "30" || e.Detail != until.Format(time.RFC3339) || e.ActorRole != models.HistoryActorOwner {
		t.Fatalf("event = %+v", e)
	}

	log.events = nil
	if err := uc.RemoveCoTeacher("10", models.Teacher, "c1", "30"); err != nil {
		t.Fatal(err)
	}
	if e := log.only(t, models.HistoryCoTeacherRemoved); e.ActorUserID != "10" || e.TargetUserID != "30" {
		t.Fatalf("event = %+v", e)
	}

	log.events = nil
	if err := uc.RemoveCoTeacher("20", models.Teacher, "c1", "20"); err != nil {
		t.Fatal(err)
	}
	if e := log.only(t, models.HistoryCoTeacherLeft); e.ActorRole != models.HistoryActorCoTeacher {
		t.Fatalf("event = %+v", e)
	}
}

func TestHistoryAssignmentClosedOnlyWhenItChanges(t *testing.T) {
	withoutLMSNames(t)
	log := &historyLog{Gateway: &studentGateway{assignments: map[string]*models.ClassAssignmentDB{
		"a1": {ID: "a1", EdxCohortID: "7", Title: "Лабиринт"},
	}}}
	uc := &UseCaseImpl{gw: log}

	for _, closed := range []bool{true, true, false} {
		if _, err := uc.SetAssignmentClosed("10", models.Teacher, "class-1", "a1", closed); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{models.HistoryAssignmentClosed, models.HistoryAssignmentReopened}
	if got := log.actionsRecorded(); !slices.Equal(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	if log.events[0].Title != "Лабиринт" {
		t.Fatalf("event = %+v", log.events[0])
	}
}

func TestHistoryScratchWork(t *testing.T) {
	withCohortMembers(t, "797")
	due := time.Now().Add(-time.Hour)
	log := &historyLog{Gateway: &studentGateway{
		assignments: map[string]*models.ClassAssignmentDB{"a1": {ID: "a1", EdxCohortID: "7", Title: "Лабиринт", DueAt: &due}},
		projects: map[string]*models.ScratchProjectDB{
			"a1/797": {
				ID: "p1", OwnerUserID: "797", Title: "Лабиринт", AssignmentID: strPtr("a1"),
				EdxCourseID: strPtr("c"), EdxCohortID: strPtr("7"), ReviewStatus: strPtr(models.ReviewDraft),
			},
		},
	}}
	uc := &UseCaseImpl{gw: log, notify: &workNotifier{}}

	if err := uc.SubmitAssignment("797", "p1", "Готово"); err != nil {
		t.Fatal(err)
	}
	e := log.only(t, models.HistoryWorkSubmitted)
	if e.ActorUserID != "797" || e.ActorRole != models.HistoryActorStudent || e.Title != "Лабиринт" || e.Detail != models.HistoryLate {
		t.Fatalf("event = %+v", e)
	}

	log.events = nil
	// The fake keeps the status it was given; the hand-in made it submitted.
	log.Gateway.(*studentGateway).projects["a1/797"].ReviewStatus = strPtr(models.ReviewSubmitted)
	if err := uc.ReviewSubmission("10", models.Teacher, "p1", models.ReviewAccepted, "Молодец, Аня"); err != nil {
		t.Fatal(err)
	}
	e = log.only(t, models.HistoryWorkReviewed)
	if e.TargetUserID != "797" || e.Detail != models.ReviewAccepted || e.Title != "Лабиринт" {
		t.Fatalf("event = %+v", e)
	}
}

// A student's work on a draft stays between them and the file: teachers do not see drafts.
func TestHistoryFileWorkSkipsTheDraft(t *testing.T) {
	uc, gw := newFileUseCase(t)
	log := &historyLog{Gateway: gw}
	uc.gw = log
	a := createFileAssignment(t, uc)
	log.only(t, models.HistoryAssignmentCreated)

	log.events = nil
	if _, err := uc.AttachFile("797", a.ID, "f1"); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.AttachFile("797", a.ID, "f2"); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.DetachFile("797", a.ID, "f2"); err != nil {
		t.Fatal(err)
	}
	if len(log.events) != 0 {
		t.Fatalf("draft work recorded: %v", log.actionsRecorded())
	}

	if err := uc.SubmitFiles("797", a.ID, ""); err != nil {
		t.Fatal(err)
	}
	if e := log.only(t, models.HistoryWorkSubmitted); e.Title != "Деталь" || e.Detail != "" {
		t.Fatalf("event = %+v", e)
	}

	log.events = nil
	sub, err := gw.FindFileSubmission(a.ID, "797")
	if err != nil {
		t.Fatal(err)
	}
	if err := uc.ReviewFileSubmission("10", models.Teacher, sub.ID, models.ReviewReturned, "Нет чертежа"); err != nil {
		t.Fatal(err)
	}
	if e := log.only(t, models.HistoryWorkReviewed); e.TargetUserID != "797" || e.Detail != models.ReviewReturned {
		t.Fatalf("event = %+v", e)
	}
}

func TestHistoryWriteFailureDoesNotUndoTheAction(t *testing.T) {
	uc, log := newHistoryClass(t)
	log.err = errors.New("projects db is down")
	local := log.Gateway.(*coTeacherGateway).localGateway

	if err := uc.RemoveMember("10", models.Teacher, "c1", "797"); err != nil {
		t.Fatalf("err = %v", err)
	}
	if local.members["c1"]["797"] {
		t.Fatal("still on the roster")
	}
}

func TestRecordClassMessage(t *testing.T) {
	uc, log := newHistoryClass(t)

	uc.RecordClassMessage("20", models.Teacher, "c1", []string{"797"})
	if e := log.only(t, models.HistoryMessageSent); e.TargetUserID != "797" || e.Detail != "1" {
		t.Fatalf("event = %+v", e)
	}

	log.events = nil
	uc.RecordClassMessage("10", models.Teacher, "c1", []string{"797", "798"})
	if e := log.only(t, models.HistoryMessageSent); e.TargetUserID != "" || e.Detail != "2" {
		t.Fatalf("event = %+v", e)
	}

	// classId comes from the client: it must not write into someone else's class.
	log.events = nil
	uc.RecordClassMessage("555", models.Teacher, "c1", []string{"797"})
	uc.RecordClassMessage("10", models.Teacher, "no-such-class", []string{"797"})
	if len(log.events) != 0 {
		t.Fatalf("events = %v", log.actionsRecorded())
	}
}

func TestListHistoryAccess(t *testing.T) {
	uc, _ := newHistoryClass(t)
	for _, tc := range []struct {
		id   string
		role models.Role
		want error
	}{
		{"10", models.Teacher, nil},
		{"20", models.Teacher, nil},
		{"99", models.SuperAdmin, nil},
		{"555", models.Teacher, teacherclass.ErrForbidden},
		// On the roster, but the history is for teachers.
		{"797", models.Student, teacherclass.ErrForbidden},
	} {
		if _, err := uc.ListHistory(tc.id, tc.role, "c1", "", ""); !errors.Is(err, tc.want) {
			t.Errorf("%s: err = %v, want %v", tc.id, err, tc.want)
		}
	}
}

func TestListHistoryPages(t *testing.T) {
	uc, log := newHistoryClass(t)
	lookups := 0
	prev := ownerDisplayName
	ownerDisplayName = func(id string) string {
		lookups++
		return map[string]string{"10": "Учитель", "797": "Аня"}[id]
	}
	t.Cleanup(func() { ownerDisplayName = prev })
	at := time.Date(2026, 11, 4, 10, 0, 0, 123456000, time.UTC)
	for i := 0; i < historyPageSize+1; i++ {
		log.events = append(log.events, models.ClassHistoryEventDB{
			ID: fmt.Sprintf("00000000-0000-0000-0000-%012d", i), ClassID: "c1", Action: models.HistoryMemberRemoved,
			ActorUserID: "10", ActorRole: models.HistoryActorOwner, TargetUserID: "797", CreatedAt: at.Add(-time.Duration(i) * time.Second),
		})
	}

	page, err := uc.ListHistory("10", models.Teacher, "c1", "members", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != historyPageSize || page.NextCursor == "" {
		t.Fatalf("items = %d, cursor = %q", len(page.Items), page.NextCursor)
	}
	if first := page.Items[0]; first.ActorName != "Учитель" || first.TargetName != "Аня" {
		t.Fatalf("first = %+v", first)
	}
	if lookups != 2 {
		t.Fatalf("name lookups = %d, want one per person", lookups)
	}
	if !slices.Contains(log.actions, models.HistoryMemberLeft) || slices.Contains(log.actions, models.HistoryWorkReviewed) {
		t.Fatalf("filter = %v", log.actions)
	}

	last := log.events[historyPageSize-1]
	log.events = log.events[historyPageSize:]
	page, err = uc.ListHistory("10", models.Teacher, "c1", "", page.NextCursor)
	if err != nil {
		t.Fatal(err)
	}
	if log.before == nil || !log.before.At.Equal(last.CreatedAt) || log.before.ID != last.ID {
		t.Fatalf("before = %+v, want %+v", log.before, last)
	}
	if len(log.actions) != 0 {
		t.Fatalf("no category must not filter: %v", log.actions)
	}
	if len(page.Items) != 1 || page.NextCursor != "" {
		t.Fatalf("last page: items = %d, cursor = %q", len(page.Items), page.NextCursor)
	}
}

func TestListHistoryBadInput(t *testing.T) {
	uc, _ := newHistoryClass(t)
	if _, err := uc.ListHistory("10", models.Teacher, "c1", "nope", ""); !errors.Is(err, teacherclass.ErrBadRequest) {
		t.Fatalf("category: err = %v", err)
	}
	for _, cursor := range []string{"abc", "abc_00000000-0000-0000-0000-000000000000", "1_not-a-uuid"} {
		if _, err := uc.ListHistory("10", models.Teacher, "c1", "", cursor); !errors.Is(err, teacherclass.ErrBadRequest) {
			t.Fatalf("cursor %q: err = %v", cursor, err)
		}
	}
}
