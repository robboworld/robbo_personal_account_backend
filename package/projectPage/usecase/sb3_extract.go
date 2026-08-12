package usecase

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
)

func extractProjectJSONFromSb3(data []byte) (string, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	for _, f := range reader.File {
		name := strings.ToLower(strings.TrimPrefix(f.Name, "./"))
		if name == "project.json" {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()
			body, err := io.ReadAll(rc)
			if err != nil {
				return "", err
			}
			return string(body), nil
		}
	}
	return "", nil
}

// validateScratchVMJSON ensures project.json looks like a loadable Scratch 3 project.
func validateScratchVMJSON(vmJSON string) error {
	if strings.TrimSpace(vmJSON) == "" {
		return projectPage.ErrInvalidProjectFile
	}
	var parsed struct {
		Targets []struct {
			IsStage bool `json:"isStage"`
		} `json:"targets"`
		Meta *struct {
			Semver string `json:"semver"`
		} `json:"meta"`
	}
	if err := json.Unmarshal([]byte(vmJSON), &parsed); err != nil {
		return projectPage.ErrInvalidProjectFile
	}
	if parsed.Meta == nil || strings.TrimSpace(parsed.Meta.Semver) == "" {
		return projectPage.ErrInvalidProjectFile
	}
	for _, t := range parsed.Targets {
		if t.IsStage {
			return nil
		}
	}
	return projectPage.ErrInvalidProjectFile
}

// validateSb3Archive ensures the payload is a ZIP with a loadable project.json.
// Empty/broken uploads otherwise make Scratch VM throw validation / FixedAsciiString errors.
func validateSb3Archive(data []byte) error {
	if len(data) < 4 || data[0] != 'P' || data[1] != 'K' {
		return projectPage.ErrInvalidProjectFile
	}
	vmJSON, err := extractProjectJSONFromSb3(data)
	if err != nil {
		return projectPage.ErrInvalidProjectFile
	}
	if strings.TrimSpace(vmJSON) == "" {
		return projectPage.ErrInvalidProjectFile
	}
	return validateScratchVMJSON(vmJSON)
}
