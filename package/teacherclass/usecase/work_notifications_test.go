package usecase

import (
	"errors"
	"strings"
	"testing"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/notifications"
)

// workNotifier records notifications; err makes them fail.
type workNotifier struct {
	notifications.Gateway
	err  error
	sent []models.UserNotificationDB
}

func (n *workNotifier) CreateOrUpdateByDedupe(row *models.UserNotificationDB) error {
	if n.err != nil {
		return n.err
	}
	n.sent = append(n.sent, *row)
	return nil
}

func submittedFileWork(t *testing.T, uc *UseCaseImpl, gw *fileGateway) string {
	t.Helper()
	dto := createFileAssignment(t, uc)
	if _, err := uc.AttachFile("797", dto.ID, "f1"); err != nil {
		t.Fatal(err)
	}
	if err := uc.SubmitFiles("797", dto.ID, ""); err != nil {
		t.Fatal(err)
	}
	for id := range gw.subs {
		return id
	}
	t.Fatal("no submission")
	return ""
}

func TestSubmitFilesNotifiesOwnerOncePerAssignment(t *testing.T) {
	uc, gw := newFileUseCase(t)
	gw.inv.DisplayName = "5 А"
	notify := uc.notify.(*workNotifier)
	submittedFileWork(t, uc, gw)

	if len(notify.sent) != 1 {
		t.Fatalf("notifications = %+v", notify.sent)
	}
	row := notify.sent[0]
	if row.RecipientUserID != "10" || row.Kind != "class_work_submitted" {
		t.Fatalf("row = %+v", row)
	}
	if !strings.Contains(row.Body, "«Деталь»") || !strings.Contains(row.Body, "«5 А»") {
		t.Fatalf("body = %q", row.Body)
	}
	if row.ActionURL == nil || *row.ActionURL != "/teacher/classes/class-1" {
		t.Fatalf("action = %v", row.ActionURL)
	}
	// The key has no student in it: a whole class handing in stays one row.
	if row.DedupeKey == nil || strings.Contains(*row.DedupeKey, "797") {
		t.Fatalf("dedupe = %v", row.DedupeKey)
	}
}

func TestReviewFilesNotifiesStudent(t *testing.T) {
	uc, gw := newFileUseCase(t)
	notify := uc.notify.(*workNotifier)
	subID := submittedFileWork(t, uc, gw)
	notify.sent = nil

	if err := uc.ReviewFileSubmission("10", models.Teacher, subID, models.ReviewReturned, "Добавь отверстие"); err != nil {
		t.Fatal(err)
	}
	if len(notify.sent) != 1 {
		t.Fatalf("notifications = %+v", notify.sent)
	}
	row := notify.sent[0]
	if row.RecipientUserID != "797" || row.Title != "Работа возвращена на доработку" {
		t.Fatalf("row = %+v", row)
	}
	if row.Body != "«Деталь»: Добавь отверстие" {
		t.Fatalf("body = %q", row.Body)
	}
	if row.ActionURL == nil || *row.ActionURL != "/myclasses" {
		t.Fatalf("action = %v", row.ActionURL)
	}

	notify.sent = nil
	if err := uc.ReviewFileSubmission("10", models.Teacher, subID, models.ReviewAccepted, ""); err != nil {
		t.Fatal(err)
	}
	if len(notify.sent) != 1 || notify.sent[0].Body != "«Деталь»" {
		t.Fatalf("accepted without a comment: %+v", notify.sent)
	}
}

func TestReviewBackToSubmittedTellsStudentNothing(t *testing.T) {
	uc, gw := newFileUseCase(t)
	notify := uc.notify.(*workNotifier)
	subID := submittedFileWork(t, uc, gw)
	notify.sent = nil

	if err := uc.ReviewFileSubmission("10", models.Teacher, subID, models.ReviewSubmitted, ""); err != nil {
		t.Fatal(err)
	}
	if len(notify.sent) != 0 {
		t.Fatalf("notifications = %+v", notify.sent)
	}
}

func TestFailedNotificationKeepsHandInAndReview(t *testing.T) {
	uc, gw := newFileUseCase(t)
	uc.notify.(*workNotifier).err = errors.New("bell down")
	subID := submittedFileWork(t, uc, gw)

	if err := uc.ReviewFileSubmission("10", models.Teacher, subID, models.ReviewReturned, "x"); err != nil {
		t.Fatal(err)
	}
	if gw.subs[subID].ReviewStatus != models.ReviewReturned {
		t.Fatalf("status = %q", gw.subs[subID].ReviewStatus)
	}
}

func handInGateway() *studentGateway {
	return &studentGateway{projects: map[string]*models.ScratchProjectDB{
		"a1/797": {ID: "p1", OwnerUserID: "797", AssignmentID: strPtr("a1")},
	}}
}

func TestSubmitAssignmentNotifiesOwner(t *testing.T) {
	withCohortMembers(t, "797")
	withoutLMSNames(t)
	notify := &workNotifier{}
	uc := &UseCaseImpl{gw: handInGateway(), notify: notify}

	if err := uc.SubmitAssignment("797", "p1", ""); err != nil {
		t.Fatal(err)
	}
	if len(notify.sent) != 1 {
		t.Fatalf("notifications = %+v", notify.sent)
	}
	row := notify.sent[0]
	if row.RecipientUserID != "10" || row.Kind != "class_work_submitted" {
		t.Fatalf("row = %+v", row)
	}
	if row.DedupeKey == nil || *row.DedupeKey != "class_work_submitted:a1" {
		t.Fatalf("dedupe = %v", row.DedupeKey)
	}
}

func TestRefusedHandInNotifiesNobody(t *testing.T) {
	withCohortMembers(t, "797")
	notify := &workNotifier{}
	uc := &UseCaseImpl{gw: handInGateway(), notify: notify}

	if err := uc.SubmitAssignment("798", "p1", ""); err == nil {
		t.Fatal("someone else's copy was handed in")
	}
	if len(notify.sent) != 0 {
		t.Fatalf("notifications = %+v", notify.sent)
	}
}

func TestFailedNotificationKeepsScratchHandIn(t *testing.T) {
	withCohortMembers(t, "797")
	withoutLMSNames(t)
	gw := handInGateway()
	uc := &UseCaseImpl{gw: gw, notify: &workNotifier{err: errors.New("bell down")}}

	if err := uc.SubmitAssignment("797", "p1", ""); err != nil {
		t.Fatal(err)
	}
	if gw.reviewed != "p1" {
		t.Fatalf("reviewed = %q", gw.reviewed)
	}
}

func TestReviewSubmissionNotifiesStudent(t *testing.T) {
	gw := newDraftsGateway()
	gw.projects["a1/798"].Title = "Лабиринт"
	notify := &workNotifier{}
	uc := &UseCaseImpl{gw: gw, notify: notify}

	if err := uc.ReviewSubmission("10", models.Teacher, "sent", models.ReviewReturned, "Добавь финиш"); err != nil {
		t.Fatal(err)
	}
	if len(notify.sent) != 1 {
		t.Fatalf("notifications = %+v", notify.sent)
	}
	row := notify.sent[0]
	if row.RecipientUserID != "798" || row.Title != "Работа возвращена на доработку" {
		t.Fatalf("row = %+v", row)
	}
	if row.Body != "«Лабиринт»: Добавь финиш" {
		t.Fatalf("body = %q", row.Body)
	}
	if row.DedupeKey == nil || *row.DedupeKey != "class_work_reviewed:sent" {
		t.Fatalf("dedupe = %v", row.DedupeKey)
	}

	notify.sent = nil
	if err := uc.ReviewSubmission("10", models.Teacher, "sent", models.ReviewSubmitted, ""); err != nil {
		t.Fatal(err)
	}
	if len(notify.sent) != 0 {
		t.Fatalf("back to the queue: %+v", notify.sent)
	}
}
