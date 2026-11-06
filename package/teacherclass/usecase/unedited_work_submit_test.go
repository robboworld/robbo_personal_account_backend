package usecase

import (
	"errors"
	"testing"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
)

// The hand-in itself stops at unedited work and lets edited work through.
func TestSubmitNeedsEditedWork(t *testing.T) {
	withCohortMembers(t, "797")
	gw := &studentGateway{
		projects: map[string]*models.ScratchProjectDB{
			"a1/797": {ID: "p1", OwnerUserID: "797", AssignmentID: strPtr("a1"), ScratchVMJSON: editedProject},
		},
		unsaved: map[string]bool{"p1": true},
	}
	uc := &UseCaseImpl{gw: gw, notify: &workNotifier{}}

	if err := uc.SubmitAssignment("797", "p1", ""); !errors.Is(err, teacherclass.ErrWorkNotEdited) {
		t.Fatalf("unedited: err = %v, want ErrWorkNotEdited", err)
	}
	if gw.reviewed != "" {
		t.Fatal("unedited work was handed in")
	}
	gw.unsaved = nil
	if err := uc.SubmitAssignment("797", "p1", ""); err != nil || gw.reviewed != "p1" {
		t.Fatalf("edited: err = %v, reviewed = %q", err, gw.reviewed)
	}
}
