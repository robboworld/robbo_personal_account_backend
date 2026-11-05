package usecase

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles"
)

func utf16le(s string) string {
	var b strings.Builder
	for i := range len(s) {
		b.WriteByte(s[i])
		b.WriteByte(0)
	}
	return b.String()
}

// aiManifest is the part of a compiled App Inventor manifest the checks read.
func aiManifest(extra ...string) string {
	return "\x03\x00\x08\x00" + utf16le("appinventor.ai_test.Hello") +
		utf16le("com.google.appinventor.components.runtime.multidex.MultiDexApplication") +
		utf16le(strings.Join(extra, ""))
}

func aiAPK(t *testing.T, extra ...zipEntry) []byte {
	t.Helper()
	return apkWith(t, aiManifest(), extra...)
}

func apkWith(t *testing.T, manifest string, extra ...zipEntry) []byte {
	t.Helper()
	return zipWith(t, append([]zipEntry{
		{name: "AndroidManifest.xml", body: manifest},
		{name: "classes.dex", body: "dex\n035\x00"},
		{name: "resources.arsc", body: "\x02\x00\x0c\x00"},
	}, extra...)...)
}

func aiProject(t *testing.T, extra ...zipEntry) []byte {
	t.Helper()
	return zipWith(t, append([]zipEntry{
		{name: "youngandroidproject/project.properties", body: "main=appinventor.ai_test.Hello.Screen1\n"},
		{name: "src/appinventor/ai_test/Hello/Screen1.scm", body: "#|\n$JSON\n{}\n|#"},
		{name: "src/appinventor/ai_test/Hello/Screen1.bky", body: "<xml/>"},
		{name: "assets/kitty.png", body: "\x89PNG"},
	}, extra...)...)
}

func gzipped(t *testing.T, data string) string {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	zw.Write([]byte(data))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func zstded(t *testing.T, data string) string {
	t.Helper()
	var buf bytes.Buffer
	zw, err := zstd.NewWriter(&buf)
	if err != nil {
		t.Fatal(err)
	}
	zw.Write([]byte(data))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

const blendData = "BLENDER-v405REND\x00\x00\x00\x00"

func TestCourseFormatsClassified(t *testing.T) {
	apk := aiAPK(t)
	aia := aiProject(t)
	cases := []struct {
		name string
		data []byte
		kind string
	}{
		{"main.py", []byte("#!/usr/bin/env python3\nprint('Привет')\n"), kindPY},
		{"robot.blend", []byte(blendData), kindBLEND},
		{"robot-gz.BLEND", []byte(gzipped(t, blendData)), kindBLEND},
		{"robot-zstd.blend", []byte(zstded(t, blendData)), kindBLEND},
		{"hello.aia", aia, kindAIA},
		{"project-in-a.zip", aia, kindZIP},
		{"hello.APK", apk, kindAPK},
	}
	for _, c := range cases {
		kind, mime, err := classify(c.name, c.data)
		if err != nil || kind != c.kind || mime != mimeByKind[c.kind] {
			t.Errorf("%s: kind=%q mime=%q err=%v, want %q", c.name, kind, mime, err, c.kind)
		}
	}
	sb3 := zipOf(t, map[string]string{"project.json": "{}"})
	refused := []struct {
		name string
		data []byte
	}{
		{"binary.py", []byte("print(1)\x00\x01")},
		{"page.py", []byte("<!DOCTYPE html><script>alert(1)</script>")},
		{"setup.py", []byte("MZ\x90\x00\x03\x00\x00\x00")},
		{"fake.blend", []byte("MZ\x90\x00\x03\x00\x00\x00")},
		{"fake-gz.blend", []byte(gzipped(t, "MZ\x90\x00 a program"))},
		{"fake-zstd.blend", []byte(zstded(t, "MZ\x90\x00 a program"))},
		{"short.blend", []byte("BLEN")},
		{"game.apk", sb3},
		{"plain.apk", zipOf(t, map[string]string{"a.txt": "hi"})},
		{"setup.apk", []byte("MZ\x90\x00\x03\x00\x00\x00")},
		{"janus.apk", append([]byte("dex\n035\x00"), apk...)},
		{"game.aia", sb3},
		{"app.aia", apk},
		// Renamed back after the download, these would skip the app checks.
		{"app.zip", apk},
		{"app.sb3", append([]byte{}, zipOf(t, map[string]string{"AndroidManifest.xml": "x", "classes.dex": "x", "project.json": "{}"})...)},
		{"app.docx", zipOf(t, map[string]string{"AndroidManifest.xml": "x", "resources.arsc": "x", "[Content_Types].xml": "<x/>", "word/a.xml": "<x/>"})},
	}
	for _, c := range refused {
		if _, _, err := classify(c.name, c.data); !errors.Is(err, userfiles.ErrFileTypeNotAllowed) {
			t.Errorf("%s: err=%v, want it refused", c.name, err)
		}
	}
}

func TestCodeFormatsAreFlagged(t *testing.T) {
	cases := []struct {
		name    string
		data    []byte
		scripts bool
	}{
		{"main.py", []byte("print(1)\n"), true},
		{"robot.blend", []byte(blendData), true},
		{"hello.apk", aiAPK(t), true},
		{"hello.aia", aiProject(t), false},
		{"with-script.aia", aiProject(t, zipEntry{name: "assets/run.py", body: "print(1)"}), true},
		{"with-extension.aia", aiProject(t,
			zipEntry{name: "assets/external_comps/edu.mit.Ext/classes.jar", body: "PK\x03\x04"},
			zipEntry{name: "assets/external_comps/edu.mit.Ext/files/AndroidRuntime.jar", body: "PK\x03\x04"}), true},
		{"folders.aia", aiProject(t, zipEntry{name: "assets/"}, zipEntry{name: "src/"}), false},
		{"homework.zip", zipWith(t, zipEntry{name: "main.py", body: "#!/usr/bin/env python3\nprint(1)"}), true},
		{"app.zip", zipWith(t, zipEntry{name: "build/hello.apk", body: string(aiAPK(t))}), true},
		{"project.zip", zipWith(t, zipEntry{name: "hello.aia", body: string(aiProject(t))}), false},
		{"model.zip", zipWith(t, zipEntry{name: "robot.blend", body: blendData}), true},
		{"model-gz.zip", zipWith(t, zipEntry{name: "robot.blend", body: gzipped(t, blendData)}), true},
		{"model-zstd.zip", zipWith(t, zipEntry{name: "robot.blend", body: zstded(t, blendData)}), true},
	}
	for _, c := range cases {
		kind, _, err := classify(c.name, c.data)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		scripts, err := inspectContent(kind, bytes.NewReader(c.data), int64(len(c.data)))
		if err != nil || scripts != c.scripts {
			t.Errorf("%s: scripts=%v err=%v, want scripts=%v", c.name, scripts, err, c.scripts)
		}
	}
}

func TestAPKChecks(t *testing.T) {
	ok := []struct {
		name string
		data []byte
	}{
		{"utf16.apk", aiAPK(t)},
		{"utf8.apk", apkWith(t, "\x03\x00\x08\x00"+appInventorRuntime+".multidex.MultiDexApplication")},
		{"native.apk", aiAPK(t, zipEntry{name: "lib/arm64-v8a/libjni.so", body: "\x7fELF"})},
		{"webviewer.apk", aiAPK(t, zipEntry{name: "assets/index.html", body: "<script>"})},
		{"sms.apk", apkWith(t, aiManifest("android.permission.SEND_SMS"))},
		{"multidex.apk", aiAPK(t, zipEntry{name: "classes2.dex", body: "dex\n035\x00"})},
	}
	for _, c := range ok {
		if err := check(c.name, c.data); err != nil {
			t.Errorf("%s refused: %v", c.name, err)
		}
	}
	bad := []struct {
		why  string
		data []byte
	}{
		{"not App Inventor", apkWith(t, "\x03\x00\x08\x00"+utf16le("com.evil.game"))},
		{"device admin", apkWith(t, aiManifest("android.permission.BIND_DEVICE_ADMIN"))},
		{"accessibility, UTF-8", apkWith(t, appInventorRuntime+"android.permission.BIND_ACCESSIBILITY_SERVICE")},
		{"oversized manifest", apkWith(t, aiManifest()+strings.Repeat(" ", maxAPKManifest))},
		{"an app in an app", aiAPK(t, zipEntry{name: "assets/update.apk", body: "x"})},
		{"an app renamed", aiAPK(t, zipEntry{name: "assets/update.dat", body: "PK\x03\x04"})},
		{"dex in assets", aiAPK(t, zipEntry{name: "assets/payload.dex", body: "dex\n"})},
		{"dex in a folder", aiAPK(t, zipEntry{name: "sub/classes.dex", body: "dex\n"})},
		{"ELF outside lib", aiAPK(t, zipEntry{name: "assets/payload", body: "\x7fELF"})},
		{"ELF in another folder", aiAPK(t, zipEntry{name: "lib/evil/libjni.so", body: "\x7fELF"})},
		{"Windows program in lib", aiAPK(t, zipEntry{name: "lib/x86/libjni.so", body: "MZ\x90\x00"})},
		{"page outside assets", aiAPK(t, zipEntry{name: "res/raw/index.html", body: "<script>"})},
		{"two classes.dex", aiAPK(t, zipEntry{name: "classes.dex", body: "dex\n036\x00"})},
		{"two manifests", aiAPK(t, zipEntry{name: "AndroidManifest.xml", body: "x"})},
	}
	for _, c := range bad {
		if err := check("a.apk", c.data); !errors.Is(err, userfiles.ErrFileContentNotAllowed) {
			t.Errorf("%s: err=%v", c.why, err)
		}
	}
}

func TestAIAChecks(t *testing.T) {
	if err := check("web.aia", aiProject(t, zipEntry{name: "assets/index.html", body: "<script>"})); err != nil {
		t.Errorf("a WebViewer page refused: %v", err)
	}
	bad := []struct {
		why  string
		data []byte
	}{
		{"an app inside", aiProject(t, zipEntry{name: "assets/app.apk", body: "x"})},
		{"a program", aiProject(t, zipEntry{name: "assets/setup.exe", body: "x"})},
		{"a jar outside extensions", aiProject(t, zipEntry{name: "assets/tool.jar", body: "PK\x03\x04"})},
		{"a nested zip", aiProject(t, zipEntry{name: "assets/more.zip", body: "x"})},
		{"a nested zip renamed", aiProject(t, zipEntry{name: "assets/kitty2.png", body: "PK\x03\x04"})},
		{"a page outside assets", aiProject(t, zipEntry{name: "src/index.html", body: "<script>"})},
	}
	for _, c := range bad {
		if err := check("a.aia", c.data); !errors.Is(err, userfiles.ErrFileContentNotAllowed) {
			t.Errorf("%s: err=%v", c.why, err)
		}
	}
}

func TestNestedAppsAreCheckedLikeUploads(t *testing.T) {
	notAI := apkWith(t, utf16le("com.evil.game"))
	bad := []struct {
		why  string
		data []byte
	}{
		{"an APK not from App Inventor", zipWith(t, zipEntry{name: "game.apk", body: string(notAI)})},
		{"not an APK at all", zipWith(t, zipEntry{name: "game.apk", body: "MZ\x90\x00"})},
		{"a jpg named apk", zipWith(t, zipEntry{name: "game.apk", body: "\xff\xd8\xff\xe0"})},
		{"a disguised APK", zipWith(t, zipEntry{name: "game.dat", body: string(aiAPK(t))})},
		{"trailing dot", zipWith(t, zipEntry{name: "game.dat.", body: string(notAI)})},
		{"project with an app", zipWith(t, zipEntry{name: "a.aia", body: string(aiProject(t, zipEntry{name: "assets/x.apk", body: "x"}))})},
		{"fake blend", zipWith(t, zipEntry{name: "a.blend", body: "MZ\x90\x00"})},
		{"gzip named blend", zipWith(t, zipEntry{name: "a.blend", body: gzipped(t, "MZ\x90\x00")})},
		{"zstd archive", zipWith(t, zipEntry{name: "a.dat", body: zstded(t, "x")})},
	}
	for _, c := range bad {
		if err := check("a.zip", c.data); !errors.Is(err, userfiles.ErrFileContentNotAllowed) {
			t.Errorf("%s: err=%v", c.why, err)
		}
	}
	// Office and Scratch files have no business holding an app.
	if err := check("a.sb3", zipWith(t, zipEntry{name: "project.json", body: "{}"}, zipEntry{name: "a.apk", body: string(aiAPK(t))})); !errors.Is(err, userfiles.ErrFileContentNotAllowed) {
		t.Errorf("an app in a Scratch project: err=%v", err)
	}
}

func TestNestedAppsAreBounded(t *testing.T) {
	data := zipWith(t, zipEntry{name: "a.apk", body: string(aiAPK(t))})
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	room := int64(10)
	if _, err := inspectNested(zr.File[0], ".apk", &room); !errors.Is(err, userfiles.ErrFileContentNotAllowed) {
		t.Errorf("an app past the room: err=%v", err)
	}
}

// setCentralSize overwrites the declared unpacked size of every entry in the central directory.
func setCentralSize(data []byte, size uint32) []byte {
	out := bytes.Clone(data)
	for i := bytes.Index(out, []byte("PK\x01\x02")); i >= 0; {
		binary.LittleEndian.PutUint32(out[i+24:], size)
		next := bytes.Index(out[i+4:], []byte("PK\x01\x02"))
		if next < 0 {
			break
		}
		i += 4 + next
	}
	return out
}

func TestZipHardening(t *testing.T) {
	sb3 := func(extra ...zipEntry) []byte {
		return zipWith(t, append([]zipEntry{{name: "project.json", body: "{}"}}, extra...)...)
	}
	docx := func(extra ...zipEntry) []byte {
		return zipWith(t, append([]zipEntry{{name: ct, body: "<Types/>"}, {name: "word/document.xml", body: "<w/>"}}, extra...)...)
	}
	const relType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/attachedTemplate"
	rels := "word/_rels/settings.xml.rels"
	odt := func(content string) []byte {
		return zipWith(t, zipEntry{name: "mimetype", body: odtMimetype}, zipEntry{name: "content.xml", body: content})
	}
	bad := []struct {
		why  string
		name string
		data []byte
	}{
		{"declared size 0 hides a program", "a.zip", setCentralSize(zipWith(t, zipEntry{name: "photo.jpg", body: "MZ\x90\x00"}), 0)},
		{"declared size 1 GB+", "a.zip", setCentralSize(zipWith(t, zipEntry{name: "a.txt", body: "x"}), maxZipUnpacked+1)},
		{"exe with a trailing dot", "a.zip", zipWith(t, zipEntry{name: "game.exe.", body: "x"})},
		{"bat with a trailing space", "a.zip", zipWith(t, zipEntry{name: "run.bat ", body: "x"})},
		{"NUL in a name", "a.zip", zipWith(t, zipEntry{name: "run.bat\x00.txt", body: "x"})},
		{"NTFS stream", "a.zip", zipWith(t, zipEntry{name: "a.txt:run.exe", body: "x"})},
		{"bidi override", "a.zip", zipWith(t, zipEntry{name: "exe‮txt.exe", body: "x"})},
		{"dotted segment", "a.zip", zipWith(t, zipEntry{name: "docs/.. /.bashrc", body: "x"})},
		{"new program types", "a.zip", zipWith(t, zipEntry{name: "app.msix", body: "x"})},
		{"zstd by magic", "a.zip", zipWith(t, zipEntry{name: "a.dat", body: "\x28\xB5\x2F\xFD"})},
		{"duplicate names", "a.zip", zipWith(t, zipEntry{name: "a.txt", body: "x"}, zipEntry{name: "a.txt", body: "y"})},
		{"page in a Scratch project", "a.sb3", sb3(zipEntry{name: "index.html", body: "<script>"})},
		{"page in a ScratchJr project", "a.sjr", zipWith(t, zipEntry{name: "project/data.json", body: "{}"}, zipEntry{name: "project/a.htm", body: "<script>"})},
		{"single quotes in rels", "a.docx", docx(zipEntry{name: rels, body: "<Relationships><Relationship Type='" + relType + "' Target='http://evil/t.dotm' TargetMode='External'/></Relationships>"})},
		{"entity in rels", "a.docx", docx(zipEntry{name: rels, body: `<Relationships><Relationship Type="` + relType + `" Target="http://evil/t.dotm" TargetMode="&#69;xternal"/></Relationships>`})},
		{"entity in content types", "a.docx", zipWith(t, zipEntry{name: ct, body: `<Types><Override ContentType="application/vnd.ms-word.document.macro&#69;nabled.main+xml"/></Types>`}, zipEntry{name: "word/document.xml", body: "x"})},
		{"rels padded past the limit", "a.docx", docx(zipEntry{name: rels, body: "<Relationships>" + strings.Repeat(" ", maxDocumentXML) + "</Relationships>"})},
		{"broken rels", "a.docx", docx(zipEntry{name: rels, body: "<Relationships"})},
		{"ODT macro event", "a.odt", odt(`<o xmlns:x="x"><x:listener x:href="vnd.sun.star.script:Standard.M.Main?language=Basic&amp;location=document"/></o>`)},
		{"ODT macro link", "a.odt", odt(`<o><a href="macro:///Tools.Run"/></o>`)},
	}
	for _, c := range bad {
		if err := check(c.name, c.data); !errors.Is(err, userfiles.ErrFileContentNotAllowed) {
			t.Errorf("%s: err=%v", c.why, err)
		}
	}
	ok := []struct {
		why  string
		name string
		data []byte
	}{
		{"an empty entry", "a.zip", zipWith(t, zipEntry{name: "empty.txt"})},
		{"a folder entry", "a.zip", zipWith(t, zipEntry{name: "docs/"}, zipEntry{name: "docs/a.txt", body: "x"})},
		{"dots inside a name", "a.zip", zipWith(t, zipEntry{name: "v1..2/a.b.txt", body: "x"})},
		{"a Russian name", "a.zip", zipWith(t, zipEntry{name: "Отчёт по проекту.txt", body: "x"})},
		{"a hyperlink", "a.docx", docx(zipEntry{name: rels, body: `<Relationships><Relationship Type="http://x/hyperlink" Target="https://robbo.ru" TargetMode="External"/></Relationships>`})},
		{"an internal template", "a.docx", docx(zipEntry{name: rels, body: `<Relationships><Relationship Type="` + relType + `" Target="t.dotx"/></Relationships>`})},
		{"an ODT link", "a.odt", odt(`<o><a href="https://robbo.ru/macro:1"/></o>`)},
	}
	for _, c := range ok {
		if err := check(c.name, c.data); err != nil {
			t.Errorf("%s refused: %v", c.why, err)
		}
	}
}
