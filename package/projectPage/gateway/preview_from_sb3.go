package gateway

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"strings"
)

type scratchProjectFile struct {
	Targets []scratchTarget `json:"targets"`
}

type scratchTarget struct {
	IsStage        bool             `json:"isStage"`
	CurrentCostume int              `json:"currentCostume"`
	Costumes       []scratchCostume `json:"costumes"`
}

type scratchCostume struct {
	Md5ext     string `json:"md5ext"`
	DataFormat string `json:"dataFormat"`
}

func previewFromSb3Archive(archive []byte) (data []byte, mime string, ok bool) {
	if len(archive) == 0 {
		return nil, "", false
	}
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, "", false
	}
	files := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		files[f.Name] = f
	}
	readNamed := func(name string) ([]byte, bool) {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, false
		}
		f, found := files[name]
		if !found {
			return nil, false
		}
		rc, openErr := f.Open()
		if openErr != nil {
			return nil, false
		}
		defer rc.Close()
		b, readErr := io.ReadAll(rc)
		if readErr != nil || len(b) == 0 {
			return nil, false
		}
		return b, true
	}
	pickCostume := func(c scratchCostume) ([]byte, string, bool) {
		raw, found := readNamed(c.Md5ext)
		if !found {
			return nil, "", false
		}
		return raw, mimeFromScratchAsset(c.DataFormat, c.Md5ext), true
	}

	pj, found := readNamed("project.json")
	if found {
		var doc scratchProjectFile
		if json.Unmarshal(pj, &doc) == nil {
			if data, mime, ok = pickFromTargets(doc.Targets, true, pickCostume); ok {
				return data, mime, true
			}
			if data, mime, ok = pickFromTargets(doc.Targets, false, pickCostume); ok {
				return data, mime, true
			}
		}
	}
	for _, f := range zr.File {
		lower := strings.ToLower(f.Name)
		if strings.HasSuffix(lower, ".png") {
			if raw, found := readNamed(f.Name); found {
				return raw, "image/png", true
			}
		}
	}
	for _, f := range zr.File {
		lower := strings.ToLower(f.Name)
		if strings.HasSuffix(lower, ".svg") {
			if raw, found := readNamed(f.Name); found {
				return raw, "image/svg+xml", true
			}
		}
	}
	return nil, "", false
}

func pickFromTargets(
	targets []scratchTarget,
	stageOnly bool,
	pick func(scratchCostume) ([]byte, string, bool),
) ([]byte, string, bool) {
	preferPNG := []bool{true, false}
	for _, wantPNG := range preferPNG {
		for _, t := range targets {
			if stageOnly != t.IsStage {
				continue
			}
			if len(t.Costumes) == 0 {
				continue
			}
			idx := t.CurrentCostume
			if idx < 0 || idx >= len(t.Costumes) {
				idx = 0
			}
			ordered := []scratchCostume{t.Costumes[idx]}
			for i, c := range t.Costumes {
				if i == idx {
					continue
				}
				ordered = append(ordered, c)
			}
			for _, c := range ordered {
				isPNG := strings.EqualFold(c.DataFormat, "png") ||
					strings.HasSuffix(strings.ToLower(c.Md5ext), ".png")
				if isPNG != wantPNG {
					continue
				}
				if data, mime, ok := pick(c); ok {
					return data, mime, true
				}
			}
		}
	}
	return nil, "", false
}

func mimeFromScratchAsset(dataFormat, md5ext string) string {
	ext := strings.ToLower(strings.TrimSpace(dataFormat))
	if ext == "" {
		if i := strings.LastIndex(md5ext, "."); i >= 0 {
			ext = strings.ToLower(md5ext[i+1:])
		}
	}
	switch ext {
	case "png":
		return "image/png"
	case "jpg", "jpeg":
		return "image/jpeg"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	case "svg":
		return "image/svg+xml"
	default:
		return "application/octet-stream"
	}
}
