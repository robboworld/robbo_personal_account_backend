package usecase

import (
	"encoding/json"
	"errors"
	"reflect"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"gorm.io/gorm"
)

// workUnedited: the copy holds nothing of the student's. A saved version does not prove work:
// the editor writes a new .sb3 on every save, changed or not, and the hand-in dialog itself
// tells the student to save. So the content decides, compared with the current template: the
// issued version is gone after the first save (retention), and a template changed since
// the issue only lets an unchanged copy through.
func (u *UseCaseImpl) workUnedited(p *models.ScratchProjectDB, a *models.ClassAssignmentDB) (bool, error) {
	saved, err := u.gw.HasOwnSave(p.ID)
	if err != nil {
		return false, err
	}
	if !saved {
		return true, nil
	}
	if isBlankScratchProject(p.ScratchVMJSON) {
		return true, nil
	}
	tpl, err := u.assignmentTemplate(a)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// The teacher deleted the template: nothing left to compare with.
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return tpl != nil && sameScratchProject(p.ScratchVMJSON, tpl.ScratchVMJSON), nil
}

// sameScratchProject compares two project.json. meta names the browser and the VM of the save;
// monitors get new ids at every load.
func sameScratchProject(a, b string) bool {
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

// scratchDefaultAssets are the costumes and sounds of the project RobboScratch opens with
// (robboscratch3_gui/src/lib/default-project).
var scratchDefaultAssets = map[string]bool{
	"cd21514d0531fdffb22204e0ec5ed84a": true,
	"83a9787d4cb6f3b7632b4ddfebf74367": true,
	"f8e72b8244738d0b448e46b38c5db6c2": true,
	"bb3a866c4db08353f6faf43c54990f10": true,
	"bb6b82c9fa7c432c552ca2f251ae2078": true,
	"be2345a4417ff516f9c1a5ece86a8c64": true,
	"28c76b6bebd04be1383fe9ba4933d263": true,
}

// isBlankScratchProject: the editor's default project, the only baseline of a copy issued without
// a template. Its names depend on the language, so it is told by having no scripts, no comments
// and no asset of its own. A student who only drew a costume has an asset of their own.
func isBlankScratchProject(vmJSON string) bool {
	type asset struct {
		AssetID string `json:"assetId"`
	}
	var project struct {
		Targets []struct {
			Blocks   map[string]json.RawMessage `json:"blocks"`
			Comments map[string]json.RawMessage `json:"comments"`
			Costumes []asset                    `json:"costumes"`
			Sounds   []asset                    `json:"sounds"`
		} `json:"targets"`
	}
	if err := json.Unmarshal([]byte(vmJSON), &project); err != nil || len(project.Targets) == 0 {
		return false
	}
	for _, t := range project.Targets {
		if len(t.Blocks) > 0 || len(t.Comments) > 0 {
			return false
		}
		for _, a := range append(t.Costumes, t.Sounds...) {
			if !scratchDefaultAssets[a.AssetID] {
				return false
			}
		}
	}
	return true
}
