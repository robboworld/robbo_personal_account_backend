package usecase

import (
	"archive/zip"
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io/fs"
	"strings"
	"testing"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles"
)

type zipEntry struct {
	name, body string
	encrypted  bool
	symlink    bool
}

func zipWith(t *testing.T, entries ...zipEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		h := &zip.FileHeader{Name: e.name, Method: zip.Deflate}
		if e.encrypted {
			h.Flags |= 0x1
		}
		if e.symlink {
			h.SetMode(fs.ModeSymlink | 0o777)
		}
		w, err := zw.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(e.body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func check(name string, data []byte) error {
	kind, _, err := Classify(name, data[:min(len(data), 512)], bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	_, err = inspectContent(kind, bytes.NewReader(data), int64(len(data)))
	return err
}

const ct = "[Content_Types].xml"

func TestPythonScriptsAreFlaggedNotRefused(t *testing.T) {
	cases := []struct {
		name    string
		data    []byte
		scripts bool
	}{
		{"homework.zip", zipWith(t, zipEntry{name: "src/main.py", body: "print(1)"}), true},
		{"cache.zip", zipWith(t, zipEntry{name: "__pycache__/m.cpython-312.pyc", body: "\xcb\x0d\x0d\x0a"}), true},
		{"robot.FCStd", zipWith(t, zipEntry{name: "Document.xml", body: "<Document/>"}, zipEntry{name: "Macros/Run.FCMacro", body: "import FreeCAD"}), true},
		{"photos.zip", zipWith(t, zipEntry{name: "a.txt", body: "hi"}), false},
		{"robot.fcstd", fcstd(t), false},
	}
	for _, c := range cases {
		kind, _, err := Classify(c.name, c.data[:min(len(c.data), 512)], bytes.NewReader(c.data), int64(len(c.data)))
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		scripts, err := inspectContent(kind, bytes.NewReader(c.data), int64(len(c.data)))
		if err != nil || scripts != c.scripts {
			t.Errorf("%s: scripts=%v err=%v, want scripts=%v", c.name, scripts, err, c.scripts)
		}
	}
	// A script does not excuse the rest: a FreeCAD model that imports os is still refused.
	bad := fcstd(t, pyObj("os", "system", "", false))
	if err := check("evil.FCStd", bad); !errors.Is(err, userfiles.ErrFileContentNotAllowed) {
		t.Errorf("evil.FCStd: err=%v", err)
	}
}

// fcstd builds a FreeCAD model whose Document.xml holds the given elements.
func fcstd(t *testing.T, elements ...string) []byte {
	t.Helper()
	doc := `<?xml version='1.0' encoding='utf-8'?><Document SchemaVersion="4">` + strings.Join(elements, "") + `</Document>`
	return zipWith(t, zipEntry{name: "Document.xml", body: doc}, zipEntry{name: "GuiDocument.xml", body: "<Document/>"},
		zipEntry{name: "thumbnails/Thumbnail.png", body: "\x89PNG"}, zipEntry{name: "PartShape.brp", body: "DBRep_DrawableShape"})
}

func pyObj(module, class, value string, encoded bool) string {
	enc := ""
	if encoded {
		enc = ` encoded="yes"`
	}
	return fmt.Sprintf(`<Python value="%s"%s module="%s" class="%s"/>`, value, enc, module, class)
}

func TestZipContentChecks(t *testing.T) {
	docx := func(extra ...zipEntry) []byte {
		return zipWith(t, append([]zipEntry{{name: ct, body: "<Types/>"}, {name: "word/document.xml", body: "<w/>"}}, extra...)...)
	}
	ok := []struct {
		name string
		data []byte
	}{
		{"report.docx", docx(zipEntry{name: "word/embeddings/Microsoft_Excel_Worksheet.xlsx", body: "PK\x03\x04chart"})},
		{"report.docx", docx(zipEntry{name: "word/_rels/document.xml.rels", body: `<Relationship Type="http://x/hyperlink" Target="https://robbo.ru" TargetMode="External"/>`})},
		{"game.sb3", zipWith(t, zipEntry{name: "project.json", body: "{}"}, zipEntry{name: "cat.svg", body: "<svg/>"})},
		{"homework.zip", zipWith(t, zipEntry{name: "notes.txt", body: "hi"}, zipEntry{name: "pic.png", body: "\x89PNG"})},
		{"game.sb2", zipWith(t, zipEntry{name: "project.json", body: "{}"}, zipEntry{name: "1.svg", body: "<svg/>"})},
		{"pets.sjr", zipWith(t, zipEntry{name: "project/data.json", body: "{}"}, zipEntry{name: "project/characters/cat.svg", body: "<svg/>"})},
		{"pets-android.sjr", zipWith(t, zipEntry{name: "/project/data.json", body: "{}"}, zipEntry{name: "/project/characters/cat.svg", body: "<svg/>"})},
		{"house.skp", zipWith(t, zipEntry{name: "model.dat", body: "x"})},
		{"wall.FCStd", fcstd(t, pyObj("ArchWall", "_Wall", base64.StdEncoding.EncodeToString([]byte(`{"Type": "Wall"}`)), true),
			`<Map count="1"><Item key="Author" value="Robbo"/></Map>`)},
		{"plain.FCStd", fcstd(t, `<Python value="" encoded="yes" module="Draft" class="_Line"/>`, `<Python value="null"/>`)},
	}
	for _, c := range ok {
		if err := check(c.name, c.data); err != nil {
			t.Errorf("%s refused: %v", c.name, err)
		}
	}
	bad := []struct {
		why  string
		name string
		data []byte
	}{
		{"VBA macro", "a.docx", docx(zipEntry{name: "word/vbaProject.bin", body: "x"})},
		{"macro-enabled content type", "a.docx", zipWith(t, zipEntry{name: ct, body: `<Override ContentType="application/vnd.ms-word.document.macroEnabled.main+xml"/>`}, zipEntry{name: "word/document.xml", body: "x"})},
		{"OLE object", "a.docx", docx(zipEntry{name: "word/embeddings/oleObject1.bin", body: "x"})},
		{"OLE magic", "a.pptx", zipWith(t, zipEntry{name: ct, body: "x"}, zipEntry{name: "ppt/media/image1.png", body: "\xD0\xCF\x11\xE0\xA1\xB1\x1A\xE1"})},
		{"ActiveX", "a.xlsx", zipWith(t, zipEntry{name: ct, body: "x"}, zipEntry{name: "xl/activeX/activeX1.xml", body: "x"})},
		{"remote template", "a.docx", docx(zipEntry{name: "word/_rels/settings.xml.rels", body: `<Relationship Id="r1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/attachedTemplate" Target="http://evil/t.dotm" TargetMode="External"/>`})},
		{"ODT macro", "a.odt", zipWith(t, zipEntry{name: "mimetype", body: odtMimetype}, zipEntry{name: "Basic/Standard/Module1.xml", body: "x"})},
		{"exe by name", "a.zip", zipWith(t, zipEntry{name: "game.exe", body: "x"})},
		{"exe renamed", "a.zip", zipWith(t, zipEntry{name: "photo.jpg", body: "MZ\x90\x00"})},
		{"ELF in a project", "a.sb3", zipWith(t, zipEntry{name: "project.json", body: "{}"}, zipEntry{name: "a.wav", body: "\x7fELF"})},
		{"HTML in a zip", "a.zip", zipWith(t, zipEntry{name: "index.html", body: "<script>"})},
		{"nested zip", "a.zip", zipWith(t, zipEntry{name: "more.zip", body: "x"})},
		{"nested zip renamed", "a.zip", zipWith(t, zipEntry{name: "more.dat", body: "PK\x03\x04"})},
		{"encrypted entry", "a.zip", zipWith(t, zipEntry{name: "secret.txt", body: "x", encrypted: true})},
		{"zip slip", "a.zip", zipWith(t, zipEntry{name: "../../.bashrc", body: "x"})},
		{"zip slip, backslash", "a.zip", zipWith(t, zipEntry{name: `docs\..\..\autorun.txt`, body: "x"})},
		{"absolute path", "a.sb3", zipWith(t, zipEntry{name: "project.json", body: "{}"}, zipEntry{name: "/etc/cron.d/x", body: "x"})},
		{"absolute path in ScratchJr", "a.sjr", zipWith(t, zipEntry{name: "/project/data.json", body: "{}"}, zipEntry{name: "/etc/cron.d/x", body: "x"})},
		{"zip slip in ScratchJr", "a.sjr", zipWith(t, zipEntry{name: "/project/data.json", body: "{}"}, zipEntry{name: "/project/../../x", body: "x"})},
		{"exe in ScratchJr", "a.sjr", zipWith(t, zipEntry{name: "project/data.json", body: "{}"}, zipEntry{name: "project/sounds/a.exe", body: "x"})},
		{"drive letter", "a.zip", zipWith(t, zipEntry{name: "C:/Users/Public/x.txt", body: "x"})},
		{"symlink", "a.zip", zipWith(t, zipEntry{name: "link", body: "/etc/passwd", symlink: true})},
		{"HTML in a SketchUp zip", "a.skp", zipWith(t, zipEntry{name: "index.html", body: "<script>"})},
		{"nested zip in SketchUp", "a.skp", zipWith(t, zipEntry{name: "more.dat", body: "PK\x03\x04"})},
		{"FreeCAD imports os", "a.FCStd", fcstd(t, pyObj("os", "system", "e30=", true))},
		{"FreeCAD imports os.path", "a.FCStd", fcstd(t, pyObj("os.path", "join", "", false))},
		{"FreeCAD module as entity", "a.FCStd", fcstd(t, `<Python value="" module="o&#115;" class="x"/>`)},
		{"FreeCAD Qt process", "a.FCStd", fcstd(t, pyObj("PySide2.QtCore", "QProcess", "", false))},
		{"FreeCAD module injection", "a.FCStd", fcstd(t, pyObj("x;import os", "y", "", false))},
		{"FreeCAD legacy pickle", "a.FCStd", fcstd(t, pyObj("Draft", "_Line", base64.StdEncoding.EncodeToString([]byte("cos\nsystem\n(S'id'\ntR.")), true))},
		{"FreeCAD pickle protocol 2", "a.FCStd", fcstd(t, pyObj("Draft", "_Line", base64.StdEncoding.EncodeToString([]byte("\x80\x02cos\nsystem\n")), true))},
		{"FreeCAD bad base64", "a.FCStd", fcstd(t, pyObj("Draft", "_Line", "!!!", true))},
		{"FreeCAD BIM eval", "a.FCStd", fcstd(t, `<Map count="1"><Item key="WPPosition" value="__import__('os').system('id')"/></Map>`)},
		{"FreeCAD broken XML", "a.FCStd", zipWith(t, zipEntry{name: "Document.xml", body: "<Document><Python"})},
		{"FreeCAD unknown entity", "a.FCStd", zipWith(t, zipEntry{name: "Document.xml", body: `<Document a="&x;"/>`})},
	}
	for _, c := range bad {
		if err := check(c.name, c.data); !errors.Is(err, userfiles.ErrFileContentNotAllowed) {
			t.Errorf("%s: err=%v, want FILE_CONTENT_NOT_ALLOWED", c.why, err)
		}
	}
}

func TestZipEntryFloodRefusedBeforeParsing(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for i := 0; i <= maxZipEntries; i++ {
		zw.CreateHeader(&zip.FileHeader{Name: fmt.Sprint(i), Method: zip.Store})
	}
	zw.Close()
	data := buf.Bytes()
	if n, ok := zipEntryCount(bytes.NewReader(data), int64(len(data))); !ok || n != maxZipEntries+1 {
		t.Fatalf("count=%d ok=%v", n, ok)
	}
	if err := check("a.zip", data); !errors.Is(err, userfiles.ErrFileContentNotAllowed) {
		t.Fatalf("err=%v", err)
	}
}

func objStm(t *testing.T, body string) string {
	t.Helper()
	var z bytes.Buffer
	zw := zlib.NewWriter(&z)
	zw.Write([]byte(body))
	zw.Close()
	return fmt.Sprintf("5 0 obj\n<</Type/ObjStm/N 1/First 4/Filter/FlateDecode/Length %d>>\nstream\r\n%s\nendstream\nendobj\n", z.Len(), z.String())
}

func TestPDFChecks(t *testing.T) {
	head := "%PDF-1.7\n"
	if err := check("ok.pdf", []byte(head+"1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n"+objStm(t, "1 0 <</Type/Page>>")+"%%EOF")); err != nil {
		t.Fatalf("plain PDF refused: %v", err)
	}
	if err := check("percent.pdf", []byte(head+"1 0 obj<</Title(100%)/Type/Catalog>>endobj\n%%EOF")); err != nil {
		t.Fatalf("a percent sign refused: %v", err)
	}
	bad := map[string]string{
		"OpenAction JavaScript": head + "1 0 obj<</Type/Catalog/OpenAction<</S/JavaScript/JS(app.alert(1))>>>>endobj",
		"hex-escaped name":      head + "1 0 obj<</OpenAction<</S/Java#53cript/JS 3 0 R>>>>endobj",
		"launch action":         head + "1 0 obj<</OpenAction<</S/Launch/F(cmd.exe)>>>>endobj",
		"attached file":         head + "1 0 obj<</Names<</EmbeddedFiles 4 0 R>>>>endobj",
		"script in ObjStm":      head + objStm(t, "6 0 <</S/JavaScript/JS(app.alert(1))>>") + "%%EOF",
		"encrypted":             head + "trailer<</Encrypt 9 0 R>>",
		"percent in a string":   head + "1 0 obj<</Title(%)/OpenAction<</S/JavaScript/JS(app.alert(1))>>>>endobj",
		"comment ended by CR":   head + "%note\r1 0 obj<</OpenAction<</S/Launch/F(cmd.exe)>>>>endobj\n",
		"attachment by EF":      head + "1 0 obj<</Subtype/FileAttachment/FS<</EF<</F 4 0 R>>>>>>endobj",
		"hex ObjStm":            head + "5 0 obj<</Type/ObjStm/Filter/ASCIIHexDecode/Length 4>>\nstream\n2F4A>\nendstream\nendobj",
		"ObjStm with predictor": head + strings.Replace(objStm(t, "1 0 <<>>"), "/N 1", "/DecodeParms<</Predictor 12>>/N 1", 1),
		"indirect stream type":  head + "5 0 obj<</Type 6 0 R/Filter/FlateDecode/Length 9>>\nstream\nx\nendstream\nendobj",
	}
	for why, pdf := range bad {
		if err := check("a.pdf", []byte(pdf)); !errors.Is(err, userfiles.ErrFileContentNotAllowed) {
			t.Errorf("%s: err=%v", why, err)
		}
	}
}

func reencode(t *testing.T, kind string, data []byte) ([]byte, error) {
	t.Helper()
	var out bytes.Buffer
	err := reencodeImage(kind, bytes.NewReader(data), int64(len(data)), &out)
	return out.Bytes(), err
}

func TestImagesAreReencoded(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	var p bytes.Buffer
	png.Encode(&p, img)
	payload := []byte("<html><script>alert(document.cookie)</script></html>")
	polyglot := append(p.Bytes(), payload...)
	out, err := reencode(t, kindPNG, polyglot)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out, []byte("<script>")) {
		t.Fatal("the payload survived re-encoding")
	}
	if _, err := png.Decode(bytes.NewReader(out)); err != nil {
		t.Fatalf("output is not a PNG: %v", err)
	}
}

func TestWebPStoredAsPNG(t *testing.T) {
	// 1×1 lossless WebP.
	webp, _ := base64.StdEncoding.DecodeString("UklGRhoAAABXRUJQVlA4TA0AAAAvAAAAEAcQERGIiP4HAA==")
	kind, _, err := Classify("a.webp", webp, bytes.NewReader(webp), int64(len(webp)))
	if err != nil || kind != kindWEBP {
		t.Fatalf("kind=%s err=%v", kind, err)
	}
	out, err := reencode(t, kindWEBP, webp)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := png.Decode(bytes.NewReader(out)); err != nil || storedImageKind(kindWEBP) != kindPNG || withExt("cat.webp", kindPNG) != "cat.png" {
		t.Fatalf("webp is not stored as png: %v", err)
	}
}

func TestDecompressionBombsRefused(t *testing.T) {
	// A PNG header claiming 100 000 × 100 000 pixels.
	var p bytes.Buffer
	png.Encode(&p, image.NewGray(image.Rect(0, 0, 1, 1)))
	b := p.Bytes()
	binary.BigEndian.PutUint32(b[16:20], 100000)
	binary.BigEndian.PutUint32(b[20:24], 100000)
	if _, err := reencode(t, kindPNG, b); !errors.Is(err, userfiles.ErrFileContentNotAllowed) {
		t.Fatalf("huge PNG: err=%v", err)
	}
	// A GIF with five 5000×5000 frames: tiny file, 125 MP to decode.
	g := []byte("GIF89a\x88\x13\x88\x13\x00\x00\x00")
	for i := 0; i < 5; i++ {
		g = append(g, 0x2C, 0, 0, 0, 0, 0x88, 0x13, 0x88, 0x13, 0, 2, 0)
	}
	g = append(g, 0x3B)
	if _, err := reencode(t, kindGIF, g); !errors.Is(err, userfiles.ErrFileContentNotAllowed) {
		t.Fatalf("GIF frame bomb: err=%v", err)
	}
}

func withOrientation(t *testing.T, jpg []byte, o uint16) []byte {
	t.Helper()
	tiff := []byte("II*\x00\x08\x00\x00\x00\x01\x00")
	entry := make([]byte, 12)
	binary.LittleEndian.PutUint16(entry[0:], 0x0112)
	binary.LittleEndian.PutUint16(entry[2:], 3)
	binary.LittleEndian.PutUint32(entry[4:], 1)
	binary.LittleEndian.PutUint16(entry[8:], o)
	tiff = append(append(tiff, entry...), 0, 0, 0, 0)
	seg := append([]byte("Exif\x00\x00"), tiff...)
	app1 := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(app1[2:], uint16(len(seg)+2))
	return append(append(append([]byte{}, jpg[:2]...), append(app1, seg...)...), jpg[2:]...)
}

func TestJPEGKeepsOrientationWithoutEXIF(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 32, 16))
	for x := 0; x < 32; x++ {
		for y := 0; y < 16; y++ {
			img.Set(x, y, color.White)
		}
	}
	var j bytes.Buffer
	jpeg.Encode(&j, img, nil)
	photo := withOrientation(t, j.Bytes(), 6) // phone held upright
	if o := jpegOrientation(bytes.NewReader(photo)); o != 6 {
		t.Fatalf("orientation=%d", o)
	}
	out, err := reencode(t, kindJPEG, photo)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(out))
	if err != nil || cfg.Width != 16 || cfg.Height != 32 {
		t.Fatalf("size %dx%d err=%v, want 16x32", cfg.Width, cfg.Height, err)
	}
	if bytes.Contains(out, []byte("Exif")) {
		t.Fatal("EXIF survived")
	}
}
