package usecase

import (
	"archive/zip"
	"bytes"
	"errors"
	"testing"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles"
)

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

func classify(name string, data []byte) (string, string, error) {
	head := data[:min(len(data), 512)]
	return Classify(name, head, bytes.NewReader(data), int64(len(data)))
}

func TestClassifyAccepts(t *testing.T) {
	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 32)...)
	docx := zipOf(t, map[string]string{"[Content_Types].xml": "<x/>", "word/document.xml": "<x/>"})
	xlsx := zipOf(t, map[string]string{"[Content_Types].xml": "<x/>", "xl/workbook.xml": "<x/>"})
	pptx := zipOf(t, map[string]string{"[Content_Types].xml": "<x/>", "ppt/presentation.xml": "<x/>"})
	odt := zipOf(t, map[string]string{"mimetype": odtMimetype, "content.xml": "<x/>"})
	sb3 := zipOf(t, map[string]string{"project.json": "{}"})
	plain := zipOf(t, map[string]string{"a.txt": "hi"})
	fcstd := zipOf(t, map[string]string{"Document.xml": "<Document/>", "GuiDocument.xml": "<Document/>"})
	// Binary STL: the header may start with "solid" too; the size decides.
	stl := append([]byte("solid exported by CAD"), make([]byte, 59)...)
	stl = append(stl, 2, 0, 0, 0)
	stl = append(stl, make([]byte, 100)...)
	skp := append([]byte("\xFF\xFE\xFF\x0ES\x00k\x00e\x00t\x00c\x00h\x00U\x00p\x00 \x00M\x00o\x00d\x00e\x00l\x00"), 0xFF, 0xFE, 0xFF, 0x0E, 0, 0)
	cases := []struct {
		name string
		data []byte
		kind string
	}{
		{"pic.PNG", png, kindPNG},
		{"doc.pdf", []byte("%PDF-1.7\n..."), kindPDF},
		{"notes.txt", []byte("Привет, мир\n"), kindTXT},
		{"song.mp3", append([]byte("ID3\x03\x00"), make([]byte, 32)...), kindMP3},
		{"raw.mp3", []byte{0xFF, 0xFB, 0x90, 0x64, 0, 0, 0, 0}, kindMP3},
		{"report.docx", docx, kindDOCX},
		{"table.xlsx", xlsx, kindXLSX},
		{"slides.pptx", pptx, kindPPTX},
		{"text.odt", odt, kindODT},
		{"game.sb3", sb3, kindSB3},
		{"archive.zip", plain, kindZIP},
		{"office-in-a.zip", docx, kindZIP},
		{"old.sb", append([]byte("ScratchV02"), 0, 0, 0, 0x10, 0xFF), kindSB},
		{"game.sb2", sb3, kindSB2},
		{"blink.ino", []byte("void setup() { pinMode(13, OUTPUT); }\n"), kindINO},
		{"part.stl", stl, kindSTL},
		{"part.STL", []byte("solid cube\n facet normal 0 0 1\n endfacet\nendsolid cube\n"), kindSTL},
		{"robot.FCStd", fcstd, kindFCSTD},
		{"model-in-a.zip", fcstd, kindZIP},
		{"house.skp", skp, kindSKP},
		{"house2024.skp", plain, kindSKP},
	}
	for _, c := range cases {
		kind, mime, err := classify(c.name, c.data)
		if err != nil || kind != c.kind || mime != mimeByKind[c.kind] {
			t.Errorf("%s: kind=%q mime=%q err=%v, want %q", c.name, kind, mime, err, c.kind)
		}
	}
}

func TestClassifyRejects(t *testing.T) {
	sb3 := zipOf(t, map[string]string{"project.json": "{}"})
	cases := []struct {
		name string
		data []byte
	}{
		{"page.html", []byte("<!DOCTYPE html><script>alert(1)</script>")},
		{"page.txt", []byte("<html><script>alert(1)</script></html>")},
		{"icon.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`)},
		{"icon.png", []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"></svg>`)},
		{"setup.exe", []byte("MZ\x90\x00")},
		{"setup.pdf", []byte("MZ\x90\x00\x03\x00\x00\x00")},
		{"game.docx", sb3},
		{"photo.jpg", []byte("%PDF-1.7\n")},
		{"binary.txt", []byte("text\x00more")},
		{"empty.txt", []byte{}},
		{"noext", []byte("hello")},
		{"cut.stl", append(make([]byte, 80), 5, 0, 0, 0, 1, 2, 3)},
		{"page.stl", []byte("<html><script>alert(1)</script></html>")},
		{"photo.sb", []byte("\x89PNG\r\n\x1a\n")},
		{"sketch.ino", []byte("MZ\x90\x00\x03\x00\x00\x00")},
		{"game.sb2", zipOf(t, map[string]string{"a.txt": "hi"})},
		{"model.fcstd", sb3},
		{"house.skp", []byte("%PDF-1.7\n")},
		{"house.skp", zipOf(t, map[string]string{"project.json": "{}"})},
	}
	for _, c := range cases {
		if _, _, err := Classify(c.name, c.data[:min(len(c.data), 512)], bytes.NewReader(c.data), int64(len(c.data))); !errors.Is(err, userfiles.ErrFileTypeNotAllowed) {
			t.Errorf("%s: err=%v, want FILE_TYPE_NOT_ALLOWED", c.name, err)
		}
	}
}
