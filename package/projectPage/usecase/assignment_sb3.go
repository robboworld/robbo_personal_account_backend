package usecase

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"log"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
)

// sb3Archive returns the project's .sb3 for the player and the editor.
//
// Assignment copies issued before the copy got the template's .sb3 (backend 74c2d7a) have only
// project.json. Costumes and sounds live in the archive, so the player fails on them with
// "costume.asset is null" and the editor opens the default project instead. Such a copy gets an
// archive built on the fly from its own project.json and the template's assets. Nothing is
// stored: the student's first save in the editor makes it a regular .sb3.
func (p *ProjectPageUseCaseImpl) sb3Archive(projectPageId string) ([]byte, error) {
	data, err := p.projectPageGateway.GetLatestSb3Archive(projectPageId)
	if !errors.Is(err, projectPage.ErrSb3ArchiveNotFound) {
		return data, err
	}
	row, rowErr := p.projectPageGateway.GetScratchProjectById(projectPageId)
	if rowErr != nil || row.TemplateProjectID == nil {
		return nil, err
	}
	tpl, tplErr := p.projectPageGateway.GetTemplateSb3Archive(*row.TemplateProjectID)
	if tplErr != nil {
		if errors.Is(tplErr, projectPage.ErrSb3ArchiveNotFound) {
			return nil, err
		}
		return nil, tplErr
	}
	built, missing, buildErr := sb3WithTemplateAssets(row.ScratchVMJSON, tpl)
	if buildErr != nil {
		log.Printf("projectPage: build .sb3 for assignment copy %s: %v", row.ID, buildErr)
		return nil, err
	}
	if missing > 0 {
		// Left to the player's bundled library; a custom asset among them still fails to load.
		log.Printf("projectPage: assignment copy %s: %d assets not in the template's .sb3", row.ID, missing)
	}
	return built, nil
}

type sb3AssetRefs struct {
	Targets []struct {
		Costumes []sb3AssetRef `json:"costumes"`
		Sounds   []sb3AssetRef `json:"sounds"`
	} `json:"targets"`
}

type sb3AssetRef struct {
	AssetID    string `json:"assetId"`
	DataFormat string `json:"dataFormat"`
	Md5ext     string `json:"md5ext"`
}

// sb3WithTemplateAssets zips vmJSON as project.json with every costume and sound it references
// that the template archive has. Assets are named by content hash, so a file with the same name
// is the same asset. missing counts the referenced files the template does not have.
func sb3WithTemplateAssets(vmJSON string, template []byte) (archive []byte, missing int, err error) {
	if err := validateScratchVMJSON(vmJSON); err != nil {
		return nil, 0, err
	}
	var refs sb3AssetRefs
	if err := json.Unmarshal([]byte(vmJSON), &refs); err != nil {
		return nil, 0, err
	}
	needed := map[string]bool{}
	need := func(assets []sb3AssetRef) {
		for _, a := range assets {
			name := a.Md5ext
			if name == "" && a.AssetID != "" && a.DataFormat != "" {
				name = a.AssetID + "." + a.DataFormat
			}
			if name != "" {
				needed[name] = true
			}
		}
	}
	for _, t := range refs.Targets {
		need(t.Costumes)
		need(t.Sounds)
	}

	tr, err := zip.NewReader(bytes.NewReader(template), int64(len(template)))
	if err != nil {
		return nil, 0, err
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("project.json")
	if err != nil {
		return nil, 0, err
	}
	if _, err := w.Write([]byte(vmJSON)); err != nil {
		return nil, 0, err
	}
	copied := map[string]bool{}
	for _, f := range tr.File {
		if !needed[f.Name] || copied[f.Name] {
			continue
		}
		if err := zw.Copy(f); err != nil {
			return nil, 0, err
		}
		copied[f.Name] = true
	}
	if err := zw.Close(); err != nil {
		return nil, 0, err
	}
	return buf.Bytes(), len(needed) - len(copied), nil
}
