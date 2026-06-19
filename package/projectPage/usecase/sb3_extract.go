package usecase

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
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
