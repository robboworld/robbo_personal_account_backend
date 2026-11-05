package usecase

import (
	"testing"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

const (
	blankProject = `{"targets":[
		{"isStage":true,"blocks":{},"costumes":[{"assetId":"cd21514d0531fdffb22204e0ec5ed84a"}],"sounds":[{"assetId":"83a9787d4cb6f3b7632b4ddfebf74367"}]},
		{"isStage":false,"blocks":{},"costumes":[{"assetId":"f8e72b8244738d0b448e46b38c5db6c2"}],"sounds":[{"assetId":"28c76b6bebd04be1383fe9ba4933d263"}]}
	],"meta":{"semver":"3.0.0"}}`
	templateProject = `{"targets":[{"isStage":true,"blocks":{"b1":{"opcode":"event_whenflagclicked"}},"costumes":[]}],
		"monitors":[{"id":"old"}],"meta":{"semver":"3.0.0","agent":"Teacher's browser"}}`
	// The same project after a save with no edits: another browser, new monitor ids, keys reordered.
	templateResaved = `{"meta":{"semver":"3.0.0","agent":"Student's browser"},"monitors":[{"id":"new"}],
		"targets":[{"costumes":[],"blocks":{"b1":{"opcode":"event_whenflagclicked"}},"isStage":true}]}`
	editedProject = `{"targets":[{"isStage":true,"blocks":{"b1":{"opcode":"event_whenflagclicked"},"b2":{"opcode":"motion_movesteps"}},"costumes":[]}],
		"meta":{"semver":"3.0.0"}}`
)

func TestWorkUnedited(t *testing.T) {
	tests := []struct {
		name     string
		template string
		copyJSON string
		saved    bool
		want     bool
	}{
		{name: "issued copy, never saved", template: templateProject, copyJSON: templateProject, want: true},
		{name: "never saved even if the template changed since", template: editedProject, copyJSON: templateProject, want: true},
		{name: "saved without edits", template: templateProject, copyJSON: templateResaved, saved: true, want: true},
		{name: "saved with edits", template: templateProject, copyJSON: editedProject, saved: true},
		{name: "no template, nothing saved", copyJSON: "{}", want: true},
		{name: "no template, default project saved", copyJSON: blankProject, saved: true, want: true},
		{name: "no template, scripts added", copyJSON: editedProject, saved: true},
		{name: "no template, own costume", copyJSON: `{"targets":[{"blocks":{},"costumes":[{"assetId":"0123"}]}]}`, saved: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &models.ClassAssignmentDB{ID: "a1", EdxCohortID: "7"}
			p := &models.ScratchProjectDB{ID: "p1", OwnerUserID: "797", AssignmentID: strPtr("a1"), ScratchVMJSON: tt.copyJSON}
			gw := &studentGateway{
				projects: map[string]*models.ScratchProjectDB{"a1/797": p},
				unsaved:  map[string]bool{"p1": !tt.saved},
			}
			if tt.template != "" {
				a.TemplateProjectID = strPtr("tpl")
				gw.projects["tpl"] = &models.ScratchProjectDB{ID: "tpl", ScratchVMJSON: tt.template}
			}
			got, err := (&UseCaseImpl{gw: gw}).workUnedited(p, a)
			if err != nil || got != tt.want {
				t.Fatalf("workUnedited = %v, %v; want %v", got, err, tt.want)
			}
		})
	}
}

// Work cannot be checked against a template the teacher deleted, so it is not held back.
func TestWorkEditedWithDeletedTemplate(t *testing.T) {
	a := &models.ClassAssignmentDB{ID: "a1", EdxCohortID: "7", TemplateProjectID: strPtr("gone")}
	p := &models.ScratchProjectDB{ID: "p1", OwnerUserID: "797", AssignmentID: strPtr("a1"), ScratchVMJSON: templateProject}
	gw := &studentGateway{projects: map[string]*models.ScratchProjectDB{"a1/797": p}}
	if got, err := (&UseCaseImpl{gw: gw}).workUnedited(p, a); err != nil || got {
		t.Fatalf("workUnedited = %v, %v; want false", got, err)
	}
}
