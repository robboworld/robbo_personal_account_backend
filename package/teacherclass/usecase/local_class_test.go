package usecase

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/edx"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/notifications"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
	"gorm.io/gorm"
)

// localGateway keeps classes and the ЛК roster in memory. The use case has no edx client,
// so any Open edX call from a ЛК class panics the test.
type localGateway struct {
	teacherclass.Gateway
	invites map[string]*models.ClassInviteDB
	members map[string]map[string]bool
}

func newLocalGateway() *localGateway {
	return &localGateway{invites: map[string]*models.ClassInviteDB{}, members: map[string]map[string]bool{}}
}

func (g *localGateway) CreateInvite(inv *models.ClassInviteDB) error {
	g.invites[inv.ID] = inv
	return nil
}

func (g *localGateway) GetInviteByID(id string) (*models.ClassInviteDB, error) {
	if inv, ok := g.invites[id]; ok {
		return inv, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (g *localGateway) GetInviteBySlug(slug string) (*models.ClassInviteDB, error) {
	for _, inv := range g.invites {
		if inv.InviteSlug == slug {
			return inv, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (g *localGateway) ListActiveInvites() ([]models.ClassInviteDB, error) {
	out := []models.ClassInviteDB{}
	for _, inv := range g.invites {
		out = append(out, *inv)
	}
	return out, nil
}

func (g *localGateway) AddMember(classID, userID string) error {
	if g.members[classID] == nil {
		g.members[classID] = map[string]bool{}
	}
	g.members[classID][userID] = true
	return nil
}

func (g *localGateway) RemoveMember(classID, userID string) error {
	delete(g.members[classID], userID)
	return nil
}

func (g *localGateway) ListMembers(classID string) ([]models.ClassMemberDB, error) {
	out := []models.ClassMemberDB{}
	for id := range g.members[classID] {
		out = append(out, models.ClassMemberDB{ClassID: classID, UserID: id})
	}
	return out, nil
}

func (g *localGateway) ListMemberClassIDs(userID string) ([]string, error) {
	var out []string
	for classID, m := range g.members {
		if m[userID] {
			out = append(out, classID)
		}
	}
	return out, nil
}

func (g *localGateway) IsMemberOfAny(userID string, classIDs []string) (bool, error) {
	for _, id := range classIDs {
		if g.members[id][userID] {
			return true, nil
		}
	}
	return false, nil
}

func (g *localGateway) ListAssignments(string, string) ([]models.ClassAssignmentDB, error) {
	return nil, nil
}

func withoutLMSNames(t *testing.T) {
	t.Helper()
	prev := ownerDisplayName
	ownerDisplayName = func(string) string { return "" }
	t.Cleanup(func() { ownerDisplayName = prev })
}

func TestLocalClassNeedsNoCourse(t *testing.T) {
	withoutLMSNames(t)
	gw := newLocalGateway()
	uc := &UseCaseImpl{gw: gw}

	dto, err := uc.CreateClass("10", models.Teacher, teacherclass.CreateClassInput{DisplayName: "Кружок", LinkLMS: false})
	if err != nil {
		t.Fatal(err)
	}
	inv := gw.invites[dto.ID]
	if inv == nil || inv.LmsLinked || inv.EdxCourseID != "" || inv.EdxCohortID != "lk:"+dto.ID {
		t.Fatalf("invite = %+v", inv)
	}
	if dto.LmsLinked {
		t.Fatal("dto says linked")
	}
}

func TestLinkedClassStillNeedsCourse(t *testing.T) {
	uc := &UseCaseImpl{gw: newLocalGateway()}
	if _, err := uc.CreateClass("10", models.Teacher, teacherclass.CreateClassInput{DisplayName: "5A", LinkLMS: true}); !errors.Is(err, teacherclass.ErrBadRequest) {
		t.Fatalf("err = %v, want ErrBadRequest", err)
	}
}

func TestLocalClassRosterStaysInLK(t *testing.T) {
	withoutLMSNames(t)
	gw := newLocalGateway()
	uc := &UseCaseImpl{gw: gw, notify: &leaveNotifier{}}
	dto, err := uc.CreateClass("10", models.Teacher, teacherclass.CreateClassInput{DisplayName: "Кружок"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := uc.JoinClass("797", dto.InviteSlug); err != nil {
		t.Fatal(err)
	}
	if !gw.members[dto.ID]["797"] {
		t.Fatal("student not on the ЛК roster")
	}
	if _, err := uc.ListAssignments("797", models.Student, dto.ID); err != nil {
		t.Fatalf("member err = %v", err)
	}
	if _, err := uc.ListAssignments("798", models.Student, dto.ID); !errors.Is(err, teacherclass.ErrForbidden) {
		t.Fatalf("non-member err = %v, want ErrForbidden", err)
	}
	if err := uc.RemoveMember("10", models.Teacher, dto.ID, "797"); err != nil {
		t.Fatal(err)
	}
	if gw.members[dto.ID]["797"] {
		t.Fatal("student still on the roster")
	}
}

func TestStudentSeesLocalClassesWithoutLMS(t *testing.T) {
	withoutLMSNames(t)
	prev := studentCohortIDs
	studentCohortIDs = func(string) (map[string]bool, error) { return nil, errors.New("lms down") }
	t.Cleanup(func() { studentCohortIDs = prev })

	gw := newLocalGateway()
	gw.invites["cohort"] = &models.ClassInviteDB{ID: "cohort", EdxCohortID: "7", LmsLinked: true}
	gw.invites["local"] = &models.ClassInviteDB{ID: "local", EdxCohortID: "lk:local"}
	gw.members["local"] = map[string]bool{"797": true}
	uc := &UseCaseImpl{gw: gw}

	list, err := uc.ListStudentClasses("797")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != "local" {
		t.Fatalf("classes = %+v, want only local", list)
	}
}

// assignGateway: a ЛК class whose student "bad" cannot get a copy.
type assignGateway struct {
	*localGateway
	forked    []string
	templates []*models.ScratchProjectDB
}

func (g *assignGateway) GetProject(id string) (*models.ScratchProjectDB, error) {
	return &models.ScratchProjectDB{ID: id, OwnerUserID: "10"}, nil
}

func (g *assignGateway) CreateAssignment(a *models.ClassAssignmentDB) error {
	a.ID = "a1"
	return nil
}

func (g *assignGateway) ForkProjectForStudent(tpl *models.ScratchProjectDB, studentUserID, _ string, _ *models.ClassAssignmentDB) (*models.ScratchProjectDB, error) {
	if studentUserID == "bad" {
		return nil, errors.New("insert failed")
	}
	g.forked = append(g.forked, studentUserID)
	g.templates = append(g.templates, tpl)
	return &models.ScratchProjectDB{}, nil
}

func TestCreateAssignmentReportsCopies(t *testing.T) {
	withoutLMSNames(t)
	gw := &assignGateway{localGateway: newLocalGateway()}
	uc := &UseCaseImpl{gw: gw}
	class, err := uc.CreateClass("10", models.Teacher, teacherclass.CreateClassInput{DisplayName: "Кружок"})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"797", "798", "bad"} {
		if err := gw.AddMember(class.ID, id); err != nil {
			t.Fatal(err)
		}
	}

	dto, err := uc.CreateAssignment("10", models.Teacher, class.ID, teacherclass.CreateAssignmentInput{
		Title: "Maze", AssignmentKey: "maze", TemplateProjectID: "tpl",
	})
	if err != nil {
		t.Fatal(err)
	}
	if dto.CopiesIssued != 2 || dto.CopiesFailed != 1 || dto.CopiesSkipped != 0 {
		t.Fatalf("copies issued=%d failed=%d skipped=%d, want 2/1/0", dto.CopiesIssued, dto.CopiesFailed, dto.CopiesSkipped)
	}
}

func TestCreateAssignmentWithoutTemplateIssuesEmptyProjects(t *testing.T) {
	withoutLMSNames(t)
	gw := &assignGateway{localGateway: newLocalGateway()}
	uc := &UseCaseImpl{gw: gw}
	class, err := uc.CreateClass("10", models.Teacher, teacherclass.CreateClassInput{DisplayName: "Кружок"})
	if err != nil {
		t.Fatal(err)
	}
	if err := gw.AddMember(class.ID, "797"); err != nil {
		t.Fatal(err)
	}

	dto, err := uc.CreateAssignment("10", models.Teacher, class.ID, teacherclass.CreateAssignmentInput{
		Title: "Free", AssignmentKey: "free",
	})
	if err != nil {
		t.Fatal(err)
	}
	if dto.TemplateProjectID != nil {
		t.Fatalf("template = %q, want none", *dto.TemplateProjectID)
	}
	if dto.CopiesIssued != 1 || len(gw.templates) != 1 || gw.templates[0] != nil {
		t.Fatalf("issued=%d templates=%v, want one empty project", dto.CopiesIssued, gw.templates)
	}
}

func TestLeaveLocalClass(t *testing.T) {
	gw := newLocalGateway()
	gw.invites["c1"] = &models.ClassInviteDB{ID: "c1", OwnerTeacherID: "10", DisplayName: "Кружок"}
	gw.members["c1"] = map[string]bool{"797": true, "798": true}
	notify := &leaveNotifier{}
	uc := &UseCaseImpl{gw: gw, notify: notify}

	if err := uc.LeaveClass("797", "c1"); err != nil {
		t.Fatal(err)
	}
	if gw.members["c1"]["797"] || !gw.members["c1"]["798"] {
		t.Fatalf("members = %v", gw.members["c1"])
	}
	if len(notify.sent) != 1 {
		t.Fatalf("notifications = %+v", notify.sent)
	}
	if err := uc.LeaveClass("797", "c1"); !errors.Is(err, teacherclass.ErrNotFound) {
		t.Fatalf("second leave err = %v, want ErrNotFound", err)
	}
	if err := uc.LeaveClass("797", "missing"); !errors.Is(err, teacherclass.ErrNotFound) {
		t.Fatalf("unknown class err = %v, want ErrNotFound", err)
	}
	// A refused leave tells the teacher nothing.
	if len(notify.sent) != 1 {
		t.Fatalf("notifications after refused leaves = %d", len(notify.sent))
	}
}

// leaveNotifier records notifications; err makes them fail.
type leaveNotifier struct {
	notifications.Gateway
	sent    []models.UserNotificationDB
	digests []digestEvent
	err     error
}

type digestEvent struct {
	row                     models.UserNotificationDB
	titlePrefix, bodyPrefix string
	item                    string
}

func (n *leaveNotifier) CreateOrUpdateByDedupe(row *models.UserNotificationDB) error {
	n.sent = append(n.sent, *row)
	return n.err
}

func (n *leaveNotifier) AddToDigest(row *models.UserNotificationDB, titlePrefix, bodyPrefix, item string) error {
	n.digests = append(n.digests, digestEvent{row: *row, titlePrefix: titlePrefix, bodyPrefix: bodyPrefix, item: item})
	return n.err
}

func TestLeaveNotifiesClassOwner(t *testing.T) {
	prev := ownerDisplayName
	ownerDisplayName = func(id string) string {
		if id == "797" {
			return "Аня Иванова"
		}
		return ""
	}
	t.Cleanup(func() { ownerDisplayName = prev })

	gw := newLocalGateway()
	gw.invites["c1"] = &models.ClassInviteDB{ID: "c1", OwnerTeacherID: "10", DisplayName: "Кружок"}
	gw.members["c1"] = map[string]bool{"797": true, "798": true}
	notify := &leaveNotifier{}
	uc := &UseCaseImpl{gw: gw, notify: notify}

	if err := uc.LeaveClass("797", "c1"); err != nil {
		t.Fatal(err)
	}
	row := notify.sent[0]
	if row.RecipientUserID != "10" || row.Kind != "class_member_left" || row.Source != "system" {
		t.Fatalf("row = %+v", row)
	}
	if !strings.Contains(row.Body, "Аня Иванова") || !strings.Contains(row.Body, "«Кружок»") {
		t.Fatalf("body = %q", row.Body)
	}
	if row.ActionURL == nil || *row.ActionURL != "/teacher/classes/c1" {
		t.Fatalf("action url = %v", row.ActionURL)
	}
	if row.DedupeKey == nil || *row.DedupeKey != "class_member_left:c1:797" {
		t.Fatalf("dedupe key = %v", row.DedupeKey)
	}

	// A student whose LMS profile cannot be read is still reported.
	if err := uc.LeaveClass("798", "c1"); err != nil {
		t.Fatal(err)
	}
	if body := notify.sent[1].Body; !strings.HasPrefix(body, "Ученик больше не в классе") {
		t.Fatalf("body without a name = %q", body)
	}
}

func TestJoinGoesToTheDaysDigest(t *testing.T) {
	prev := ownerDisplayName
	ownerDisplayName = func(id string) string {
		if id == "797" {
			return "Аня Иванова"
		}
		return ""
	}
	t.Cleanup(func() { ownerDisplayName = prev })

	gw := newLocalGateway()
	gw.invites["c1"] = &models.ClassInviteDB{ID: "c1", OwnerTeacherID: "10", DisplayName: "Кружок", InviteSlug: "s1", IsActive: true}
	notify := &leaveNotifier{}
	uc := &UseCaseImpl{gw: gw, notify: notify}

	if _, err := uc.JoinClass("797", "s1"); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.JoinClass("798", "s1"); err != nil {
		t.Fatal(err)
	}
	if len(notify.digests) != 2 || len(notify.sent) != 0 {
		t.Fatalf("digests = %+v, sent = %+v", notify.digests, notify.sent)
	}
	first, second := notify.digests[0], notify.digests[1]
	if first.row.RecipientUserID != "10" || first.row.Kind != "class_members_joined" {
		t.Fatalf("row = %+v", first.row)
	}
	if first.titlePrefix != "Вступлений в класс: " || first.bodyPrefix != "«Кружок»: " {
		t.Fatalf("digest = %+v", first)
	}
	if first.item != "Аня Иванова" || second.item != "Ученик" {
		t.Fatalf("items = %q, %q", first.item, second.item)
	}
	if first.row.ActionURL == nil || *first.row.ActionURL != "/teacher/classes/c1" {
		t.Fatalf("action url = %v", first.row.ActionURL)
	}
	// Both students land on one row: the key names the class and the day, not the student.
	if *first.row.DedupeKey != *second.row.DedupeKey || !strings.HasPrefix(*first.row.DedupeKey, "class_members_joined:c1:20") {
		t.Fatalf("dedupe keys = %q, %q", *first.row.DedupeKey, *second.row.DedupeKey)
	}

	// The invite link opened again: already on the roster, nothing to report.
	if _, err := uc.JoinClass("797", "s1"); err != nil {
		t.Fatal(err)
	}
	if len(notify.digests) != 2 {
		t.Fatalf("digests after joining twice = %d", len(notify.digests))
	}

	// Left and came back the same day: that is a join again, and it follows the "left" row.
	if err := uc.LeaveClass("797", "c1"); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.JoinClass("797", "s1"); err != nil {
		t.Fatal(err)
	}
	if len(notify.digests) != 3 || notify.digests[2].item != "Аня Иванова" {
		t.Fatalf("digests after coming back = %+v", notify.digests)
	}
}

// slowRoster holds the membership check open, so overlapping joins really overlap.
type slowRoster struct {
	*localGateway
	release chan struct{}
	checks  atomic.Int32
}

func (g *slowRoster) IsMemberOfAny(userID string, classIDs []string) (bool, error) {
	g.checks.Add(1)
	<-g.release
	return g.localGateway.IsMemberOfAny(userID, classIDs)
}

func TestOverlappingJoinsCountOnce(t *testing.T) {
	withoutLMSNames(t)
	local := newLocalGateway()
	local.invites["c1"] = &models.ClassInviteDB{ID: "c1", OwnerTeacherID: "10", InviteSlug: "s1", IsActive: true}
	gw := &slowRoster{localGateway: local, release: make(chan struct{})}
	notify := &leaveNotifier{}
	uc := &UseCaseImpl{gw: gw, notify: notify}

	const requests = 5
	var wg sync.WaitGroup
	errs := make(chan error, requests)
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dto, err := uc.JoinClass("797", "s1")
			if err == nil && dto.ID != "c1" {
				err = fmt.Errorf("class = %q", dto.ID)
			}
			errs <- err
		}()
	}
	// Every request gets to the check or waits for the one that did.
	time.Sleep(100 * time.Millisecond)
	close(gw.release)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if n := gw.checks.Load(); n != 1 {
		t.Fatalf("membership checks = %d, want 1", n)
	}
	if len(notify.digests) != 1 {
		t.Fatalf("digests = %d, want 1", len(notify.digests))
	}
}

func TestOwnerJoiningOwnClassIsNotNotified(t *testing.T) {
	gw := newLocalGateway()
	gw.invites["c1"] = &models.ClassInviteDB{ID: "c1", OwnerTeacherID: "10", InviteSlug: "s1", IsActive: true}
	notify := &leaveNotifier{}

	if _, err := (&UseCaseImpl{gw: gw, notify: notify}).JoinClass("10", "s1"); err != nil {
		t.Fatal(err)
	}
	if len(notify.digests) != 0 {
		t.Fatalf("digests = %+v", notify.digests)
	}
}

func TestJoinSurvivesNotificationFailure(t *testing.T) {
	withoutLMSNames(t)
	gw := newLocalGateway()
	gw.invites["c1"] = &models.ClassInviteDB{ID: "c1", OwnerTeacherID: "10", InviteSlug: "s1", IsActive: true}
	uc := &UseCaseImpl{gw: gw, notify: &leaveNotifier{err: errors.New("db down")}}

	if _, err := uc.JoinClass("797", "s1"); err != nil {
		t.Fatalf("err = %v", err)
	}
	if !gw.members["c1"]["797"] {
		t.Fatal("student is not on the roster")
	}
}

func TestLeaveSurvivesNotificationFailure(t *testing.T) {
	withoutLMSNames(t)
	gw := newLocalGateway()
	gw.invites["c1"] = &models.ClassInviteDB{ID: "c1", OwnerTeacherID: "10"}
	gw.members["c1"] = map[string]bool{"797": true}
	uc := &UseCaseImpl{gw: gw, notify: &leaveNotifier{err: errors.New("db down")}}

	if err := uc.LeaveClass("797", "c1"); err != nil {
		t.Fatalf("err = %v", err)
	}
	if gw.members["c1"]["797"] {
		t.Fatal("student is still on the roster")
	}
}

func TestOwnerLeavingOwnClassIsNotNotified(t *testing.T) {
	gw := newLocalGateway()
	gw.invites["c1"] = &models.ClassInviteDB{ID: "c1", OwnerTeacherID: "10"}
	gw.members["c1"] = map[string]bool{"10": true}
	notify := &leaveNotifier{}

	if err := (&UseCaseImpl{gw: gw, notify: notify}).LeaveClass("10", "c1"); err != nil {
		t.Fatal(err)
	}
	if len(notify.sent) != 0 {
		t.Fatalf("notifications = %+v", notify.sent)
	}
}

// cohortEdx records the cohort removal; any other Open edX call panics the test.
type cohortEdx struct {
	edx.UseCase
	removed string
	err     error
}

func (e *cohortEdx) RemoveCohortUser(username, courseID string, cohortID int) error {
	e.removed = fmt.Sprintf("%s %s %d", username, courseID, cohortID)
	return e.err
}

func TestLeaveLinkedClass(t *testing.T) {
	withCohortMembers(t, "797")
	prev := lmsUsername
	lmsUsername = func(id int64) (string, error) { return fmt.Sprintf("user%d", id), nil }
	t.Cleanup(func() { lmsUsername = prev })

	withoutLMSNames(t)
	cohort := &cohortEdx{}
	notify := &leaveNotifier{}
	uc := &UseCaseImpl{gw: &studentGateway{}, edx: cohort, notify: notify}
	if err := uc.LeaveClass("797", "class-1"); err != nil {
		t.Fatal(err)
	}
	if cohort.removed != "user797 course-v1:R+S+1 7" {
		t.Fatalf("removed = %q", cohort.removed)
	}
	if len(notify.sent) != 1 || notify.sent[0].RecipientUserID != "10" {
		t.Fatalf("notifications = %+v", notify.sent)
	}

	// Someone outside the cohort never reaches Open edX.
	cohort.removed = ""
	if err := uc.LeaveClass("798", "class-1"); !errors.Is(err, teacherclass.ErrNotFound) {
		t.Fatalf("non-member err = %v, want ErrNotFound", err)
	}
	if cohort.removed != "" {
		t.Fatalf("non-member removed from cohort: %q", cohort.removed)
	}

	cohort.err = errors.New("dial tcp edx-test.ru")
	if err := uc.LeaveClass("797", "class-1"); !errors.Is(err, teacherclass.ErrInternal) {
		t.Fatalf("Open edX failure err = %v, want ErrInternal", err)
	}
	// The student is still in the cohort: the teacher hears nothing.
	if len(notify.sent) != 1 {
		t.Fatalf("notifications after a failed leave = %d", len(notify.sent))
	}
}
