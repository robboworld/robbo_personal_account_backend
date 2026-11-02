package usecase

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
)

// copyJSON references a costume by md5ext, a sound by assetId + dataFormat, and a costume the
// template no longer has.
const copyJSON = `{"targets":[
	{"isStage":true,"costumes":[{"assetId":"aa","md5ext":"aa.png","dataFormat":"png"}],"sounds":[]},
	{"isStage":false,"costumes":[{"assetId":"gone","md5ext":"gone.svg","dataFormat":"svg"}],
	 "sounds":[{"assetId":"bb","dataFormat":"wav"}]}
],"meta":{"semver":"3.0.0"}}`

func zipOf(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func unzip(t *testing.T, data []byte) map[string]string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(rc)
		rc.Close()
		out[f.Name] = string(b)
	}
	return out
}

func templateSb3(t *testing.T) []byte {
	return zipOf(t, map[string]string{
		"project.json": `{"targets":[{"isStage":true}],"meta":{"semver":"3.0.0"}}`,
		"aa.png":       "costume bytes",
		"bb.wav":       "sound bytes",
		"unused.png":   "not referenced by the copy",
	})
}

func TestSb3WithTemplateAssets(t *testing.T) {
	built, missing, err := sb3WithTemplateAssets(copyJSON, templateSb3(t))
	if err != nil {
		t.Fatal(err)
	}
	if missing != 1 {
		t.Errorf("missing = %d, want 1 (gone.svg)", missing)
	}
	got := unzip(t, built)
	want := map[string]string{"project.json": copyJSON, "aa.png": "costume bytes", "bb.wav": "sound bytes"}
	if len(got) != len(want) {
		t.Errorf("files = %v, want only %v", keys(got), keys(want))
	}
	for name, body := range want {
		if got[name] != body {
			t.Errorf("%s = %q, want %q", name, got[name], body)
		}
	}
	if err := validateSb3Archive(built); err != nil {
		t.Errorf("built archive is not a valid .sb3: %v", err)
	}
}

func TestSb3WithTemplateAssetsRejectsBareJSON(t *testing.T) {
	if _, _, err := sb3WithTemplateAssets("{}", templateSb3(t)); err == nil {
		t.Error(`"{}" is not a loadable project and must not become an .sb3`)
	}
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// archivePages serves one project and, optionally, its own and its template's .sb3.
type archivePages struct {
	projectPage.Gateway
	row         *models.ScratchProjectDB
	own         []byte
	tpl         []byte
	tplErr      error
	tplRequests int
}

func (g *archivePages) GetLatestSb3Archive(string) ([]byte, error) {
	if g.own == nil {
		return nil, projectPage.ErrSb3ArchiveNotFound
	}
	return g.own, nil
}
func (g *archivePages) GetScratchProjectById(string) (*models.ScratchProjectDB, error) {
	return g.row, nil
}
func (g *archivePages) GetTemplateSb3Archive(string) ([]byte, error) {
	g.tplRequests++
	if g.tplErr != nil {
		return nil, g.tplErr
	}
	if g.tpl == nil {
		return nil, projectPage.ErrSb3ArchiveNotFound
	}
	return g.tpl, nil
}

func assignmentCopy() *models.ScratchProjectDB {
	assignment, template := "assignment-1", "template-1"
	return &models.ScratchProjectDB{ID: "copy-1", ScratchVMJSON: copyJSON, AssignmentID: &assignment, TemplateProjectID: &template}
}

func TestSb3ArchiveOwnArchiveWins(t *testing.T) {
	g := &archivePages{row: assignmentCopy(), own: []byte("PK own"), tpl: templateSb3(t)}
	uc := &ProjectPageUseCaseImpl{projectPageGateway: g}
	data, err := uc.sb3Archive("copy-1")
	if err != nil || string(data) != "PK own" {
		t.Fatalf("got %q, %v; want the copy's own archive", data, err)
	}
	if g.tplRequests != 0 {
		t.Error("a project with its own .sb3 must not touch the template")
	}
}

func TestSb3ArchiveBuildsLegacyAssignmentCopy(t *testing.T) {
	uc := &ProjectPageUseCaseImpl{projectPageGateway: &archivePages{row: assignmentCopy(), tpl: templateSb3(t)}}
	data, err := uc.sb3Archive("copy-1")
	if err != nil {
		t.Fatal(err)
	}
	got := unzip(t, data)
	if got["project.json"] != copyJSON || got["aa.png"] != "costume bytes" {
		t.Errorf("built archive = %v, want the copy's JSON with the template's assets", keys(got))
	}
}

func TestSb3ArchiveNotFound(t *testing.T) {
	notCopy := assignmentCopy()
	notCopy.AssignmentID = nil
	cases := map[string]*archivePages{
		"not an assignment copy":   {row: notCopy, tpl: templateSb3(t)},
		"template without an .sb3": {row: assignmentCopy()},
	}
	for name, g := range cases {
		t.Run(name, func(t *testing.T) {
			uc := &ProjectPageUseCaseImpl{projectPageGateway: g}
			if _, err := uc.sb3Archive("copy-1"); !errors.Is(err, projectPage.ErrSb3ArchiveNotFound) {
				t.Errorf("err = %v, want ErrSb3ArchiveNotFound", err)
			}
		})
	}
}

func TestSb3ArchiveTemplateErrorPropagates(t *testing.T) {
	boom := errors.New("db down")
	uc := &ProjectPageUseCaseImpl{projectPageGateway: &archivePages{row: assignmentCopy(), tplErr: boom}}
	if _, err := uc.sb3Archive("copy-1"); !errors.Is(err, boom) {
		t.Errorf("err = %v, want the storage error", err)
	}
}
