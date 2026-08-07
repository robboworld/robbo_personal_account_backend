package usecase

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
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

// validateSb3Archive ensures the payload is a ZIP with project.json that has a Stage target.
// Empty/broken uploads otherwise make Scratch VM throw "Non-ascii character in FixedAsciiString".
func validateSb3Archive(data []byte) error {
	if len(data) < 4 || data[0] != 'P' || data[1] != 'K' {
		return fmt.Errorf("%w: not a valid .sb3 (zip) archive", projectPage.ErrBadRequest)
	}
	vmJSON, err := extractProjectJSONFromSb3(data)
	if err != nil {
		return fmt.Errorf("%w: not a valid .sb3 (zip) archive", projectPage.ErrBadRequest)
	}
	if strings.TrimSpace(vmJSON) == "" {
		return fmt.Errorf("%w: project.json missing in .sb3", projectPage.ErrBadRequest)
	}
	var parsed struct {
		Targets []struct {
			IsStage bool `json:"isStage"`
		} `json:"targets"`
	}
	if err := json.Unmarshal([]byte(vmJSON), &parsed); err != nil {
		return fmt.Errorf("%w: invalid project.json", projectPage.ErrBadRequest)
	}
	hasStage := false
	for _, t := range parsed.Targets {
		if t.IsStage {
			hasStage = true
			break
		}
	}
	if !hasStage {
		return fmt.Errorf("%w: project.json has no Stage target", projectPage.ErrBadRequest)
	}
	return nil
}
