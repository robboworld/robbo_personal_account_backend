package usecase

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"io"
	"io/fs"
	"path"
	"regexp"
	"strings"
	"unicode"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles"
)

// maxZipEntries caps the central directory before archive/zip parses it: a 50 MB zip of
// empty entries would otherwise allocate ~120 MB per request. Large Scratch projects and
// Office files stay far below it.
const maxZipEntries = 20000

const (
	// All entries unpacked: refuses zip bombs, and bounds the read in entryHead.
	maxZipUnpacked = 1 << 30
	// One XML part of an Office or ODT document.
	maxDocumentXML = 64 << 20
	// App Inventor projects and apps inside one zip, unpacked: they are held in memory.
	maxNestedApps = 64 << 20
)

const (
	zip64LocSignature  = 0x07064b50
	zip64EOCDSignature = 0x06064b50
	eocdLen            = 22
	zip64LocLen        = 20
	maxCommentLen      = 0xFFFF
)

// zipEntryCount reads the entry count from the end-of-central-directory record (and the
// ZIP64 one when the count overflows 16 bits) without parsing the directory.
func zipEntryCount(ra io.ReaderAt, size int64) (int64, bool) {
	tailLen := min(size, eocdLen+maxCommentLen)
	tail := make([]byte, tailLen)
	if _, err := ra.ReadAt(tail, size-tailLen); err != nil && err != io.EOF {
		return 0, false
	}
	i := bytes.LastIndex(tail, []byte{0x50, 0x4b, 0x05, 0x06})
	if i < 0 || len(tail)-i < eocdLen {
		return 0, false
	}
	eocd := tail[i:]
	count := int64(binary.LittleEndian.Uint16(eocd[10:12]))
	if count != 0xFFFF {
		return count, true
	}
	// ZIP64: the locator sits right before the EOCD and points at the ZIP64 EOCD record.
	eocdOff := size - tailLen + int64(i)
	if eocdOff < zip64LocLen {
		return 0, false
	}
	loc := make([]byte, zip64LocLen)
	if _, err := ra.ReadAt(loc, eocdOff-zip64LocLen); err != nil || binary.LittleEndian.Uint32(loc) != zip64LocSignature {
		return 0, false
	}
	recOff := int64(binary.LittleEndian.Uint64(loc[8:16]))
	rec := make([]byte, 40)
	if recOff < 0 || recOff+40 > size {
		return 0, false
	}
	if _, err := ra.ReadAt(rec, recOff); err != nil || binary.LittleEndian.Uint32(rec) != zip64EOCDSignature {
		return 0, false
	}
	n := binary.LittleEndian.Uint64(rec[32:40])
	if n > 1<<40 {
		return 0, false
	}
	return int64(n), true
}

// openZip refuses archives with too many entries before archive/zip allocates for them.
func openZip(ra io.ReaderAt, size int64) (*zip.Reader, error) {
	n, ok := zipEntryCount(ra, size)
	if !ok {
		return nil, userfiles.ErrFileTypeNotAllowed
	}
	if n > maxZipEntries {
		return nil, userfiles.ErrFileContentNotAllowed
	}
	zr, err := zip.NewReader(ra, size)
	if err != nil {
		return nil, userfiles.ErrFileTypeNotAllowed
	}
	if len(zr.File) > maxZipEntries {
		return nil, userfiles.ErrFileContentNotAllowed
	}
	return zr, nil
}

// Programs and scripts that run on a double click, by name.
var executableExts = map[string]bool{
	".exe": true, ".dll": true, ".com": true, ".scr": true, ".pif": true, ".cpl": true, ".msi": true,
	".msp": true, ".msc": true, ".bat": true, ".cmd": true, ".ps1": true, ".psm1": true, ".vbs": true,
	".vbe": true, ".js": true, ".jse": true, ".wsf": true, ".wsh": true, ".hta": true, ".lnk": true,
	".url": true, ".reg": true, ".jar": true, ".apk": true, ".app": true, ".sh": true, ".command": true,
	".dmg": true, ".pkg": true, ".deb": true, ".rpm": true, ".appimage": true, ".iso": true, ".img": true,
	".vhd": true, ".vhdx": true, ".chm": true, ".scf": true, ".inf": true, ".gadget": true,
	".docm": true, ".xlsm": true, ".pptm": true, ".dotm": true, ".xltm": true, ".xlam": true, ".ppam": true,
	".potm": true, ".ppsm": true, ".sldm": true, ".xll": true, ".wll": true, ".ade": true, ".adp": true,
	".mde": true, ".accde": true, ".mst": true, ".ws": true, ".wsc": true, ".sct": true, ".shs": true,
	".shb": true, ".ps1xml": true, ".psd1": true, ".psc1": true, ".msix": true, ".msixbundle": true,
	".appx": true, ".appxbundle": true, ".appinstaller": true, ".application": true, ".xbap": true,
	".jnlp": true, ".ins": true, ".isp": true, ".job": true, ".sys": true, ".drv": true, ".ocx": true,
	".hlp": true, ".diagcab": true, ".library-ms": true, ".searchconnector-ms": true,
	".settingcontent-ms": true, ".desktop": true, ".scpt": true, ".workflow": true, ".run": true,
	".ipa": true, ".aab": true, ".xapk": true, ".apks": true, ".dex": true,
}

// Python scripts: allowed in a zip or a FreeCAD model (homework, workbench macros), but the
// file is flagged and the ЛК asks to run them only from a trusted source.
var pythonScriptExts = map[string]bool{".py": true, ".pyc": true, ".pyo": true, ".pyw": true, ".fcmacro": true}

// Pages a browser would run, by name.
var webPageExts = map[string]bool{".html": true, ".htm": true, ".xhtml": true, ".svg": true, ".svgz": true, ".mht": true, ".mhtml": true}

// Archives inside a plain .zip cannot be checked without unpacking them.
var archiveExts = map[string]bool{
	".zip": true, ".rar": true, ".7z": true, ".tar": true, ".gz": true, ".tgz": true, ".bz2": true,
	".xz": true, ".cab": true, ".arj": true, ".lzh": true, ".zst": true,
}

// The first bytes of programs (PE, ELF, Mach-O, Java class / fat Mach-O, script) and archives.
var (
	executableMagic = [][]byte{
		[]byte("MZ"), []byte("\x7fELF"), []byte("#!"),
		{0xFE, 0xED, 0xFA, 0xCE}, {0xFE, 0xED, 0xFA, 0xCF}, {0xCE, 0xFA, 0xED, 0xFE}, {0xCF, 0xFA, 0xED, 0xFE},
		{0xCA, 0xFE, 0xBA, 0xBE},
		// OLE2 compound file: an embedded object or an old Office document, macros included.
		{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1},
	}
	archiveMagic = [][]byte{
		[]byte("PK\x03\x04"), []byte("Rar!"), {'7', 'z', 0xBC, 0xAF}, {0x1F, 0x8B}, []byte("BZh"),
		{0xFD, '7', 'z', 'X', 'Z'}, []byte("MSCF"), zstdMagic,
	}
)

func hasMagic(head []byte, magics [][]byte) bool {
	for _, m := range magics {
		if bytes.HasPrefix(head, m) {
			return true
		}
	}
	return false
}

// entryHead returns the first bytes of an entry and reads the rest of it. An unpacker trusts
// the data, not the directory, so a declared size of 0 or a wrong one must not hide an entry
// from the checks; the reader verifies the CRC at the end.
func entryHead(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	head := make([]byte, 8)
	n, err := io.ReadFull(rc, head)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, err
	}
	rest, err := io.Copy(io.Discard, io.LimitReader(rc, int64(f.UncompressedSize64)+1))
	if err != nil {
		return nil, err
	}
	if uint64(n)+uint64(rest) != f.UncompressedSize64 {
		return nil, userfiles.ErrFileContentNotAllowed
	}
	return head[:n], nil
}

// readEntry refuses an entry past limit: a check of only its start could be padded around.
func readEntry(f *zip.File, limit int64) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	body, err := io.ReadAll(io.LimitReader(rc, limit+1))
	if err == nil && int64(len(body)) > limit {
		err = userfiles.ErrFileContentNotAllowed
	}
	return body, err
}

// zipFitsUnpacked adds up the declared sizes; entryHead holds the entries to them.
func zipFitsUnpacked(zr *zip.Reader) bool {
	var total uint64
	for _, f := range zr.File {
		if f.UncompressedSize64 > maxZipUnpacked {
			return false
		}
		if total += f.UncompressedSize64; total > maxZipUnpacked {
			return false
		}
	}
	return true
}

// Office relationships that load something from outside the file when it opens: a remote
// template (macros), an OLE object, a frame or a sub-document.
var externalRelTypes = map[string]bool{
	"attachedtemplate": true, "oleobject": true, "frame": true, "subdocument": true, "package": true,
}

// entryHasElement parses an XML entry and reports whether bad matches any element. Parsed, not
// searched: single quotes and entities like "&#69;xternal" mean the same to Office.
func entryHasElement(f *zip.File, bad func(xml.StartElement) bool) error {
	body, err := readEntry(f, maxDocumentXML)
	if err != nil {
		return userfiles.ErrFileContentNotAllowed
	}
	dec := xml.NewDecoder(bytes.NewReader(body))
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return userfiles.ErrFileContentNotAllowed
		}
		if el, ok := tok.(xml.StartElement); ok && bad(el) {
			return userfiles.ErrFileContentNotAllowed
		}
	}
}

func unsafeOfficeElement(el xml.StartElement) bool {
	var relType, mode string
	for _, a := range el.Attr {
		value := strings.ToLower(a.Value)
		if strings.Contains(value, "macroenabled") {
			return true
		}
		switch a.Name.Local {
		case "Type":
			relType = path.Base(value)
		case "TargetMode":
			mode = value
		}
	}
	return el.Name.Local == "Relationship" && mode == "external" && externalRelTypes[relType]
}

// unsafeODTElement: an event listener or a link that calls a macro or a script of the office
// suite itself; nothing has to be embedded for it to run.
func unsafeODTElement(el xml.StartElement) bool {
	for _, a := range el.Attr {
		value := strings.ToLower(a.Value)
		if strings.Contains(value, "vnd.sun.star.script:") || strings.HasPrefix(value, "macro:") {
			return true
		}
	}
	return false
}

// inspectZip rejects archives whose contents could run code when the owner opens them, and
// reports the code a course may need (Python, App Inventor apps and extensions). It reads every
// entry through, within maxZipUnpacked, but writes nothing to disk.
func inspectZip(kind string, ra io.ReaderAt, size int64) (hasScripts bool, err error) {
	zr, err := openZip(ra, size)
	if err != nil {
		return false, err
	}
	if !zipFitsUnpacked(zr) {
		return false, userfiles.ErrFileContentNotAllowed
	}
	nestedRoom := int64(maxNestedApps)
	seen := make(map[string]bool, len(zr.File))
	for _, f := range zr.File {
		name := strings.ToLower(f.Name)
		ext := entryExt(name)
		if f.Flags&0x1 != 0 { // encrypted: nothing inside can be checked
			return false, userfiles.ErrFileContentNotAllowed
		}
		// Two entries with one name: this check would see one, an installer or unpacker the other.
		if seen[f.Name] {
			return false, userfiles.ErrFileContentNotAllowed
		}
		seen[f.Name] = true
		entryPath := f.Name
		// ScratchJr on Android writes "/project/data.json"; any other absolute path stays refused.
		if kind == kindSJR && strings.HasPrefix(entryPath, "/project/") {
			entryPath = entryPath[1:]
		}
		if unsafeZipPath(entryPath) || f.Mode()&fs.ModeSymlink != 0 {
			return false, userfiles.ErrFileContentNotAllowed
		}
		head, err := entryHead(f)
		if err != nil {
			return false, userfiles.ErrFileContentNotAllowed
		}
		if kind == kindZIP && nestedFormats[ext] {
			scripts, err := inspectNested(f, ext, &nestedRoom)
			if err != nil {
				return false, err
			}
			hasScripts = hasScripts || scripts
			continue
		}
		code := isAppCode(kind, name, ext)
		hasScripts = hasScripts || code
		if !code && executableExts[ext] || refusedWebPage(kind, name, ext) {
			return false, userfiles.ErrFileContentNotAllowed
		}
		if isOpaqueZip(kind) && archiveExts[ext] {
			return false, userfiles.ErrFileContentNotAllowed
		}
		switch kind {
		case kindDOCX, kindXLSX, kindPPTX:
			base := path.Base(name)
			if base == "vbaproject.bin" || base == "vbadata.xml" || strings.Contains(name, "/activex/") ||
				strings.HasPrefix(base, "oleobject") {
				return false, userfiles.ErrFileContentNotAllowed
			}
			if name == "[content_types].xml" || strings.HasSuffix(name, ".rels") {
				if err := entryHasElement(f, unsafeOfficeElement); err != nil {
					return false, err
				}
			}
		case kindODT:
			if strings.HasPrefix(name, "basic/") || strings.HasPrefix(name, "scripts/") {
				return false, userfiles.ErrFileContentNotAllowed
			}
			if ext == ".xml" && !strings.Contains(name, "/") {
				if err := entryHasElement(f, unsafeODTElement); err != nil {
					return false, err
				}
			}
		}
		if pythonScriptExts[ext] {
			hasScripts = true
		}
		if hasMagic(head, executableMagic) && !allowedProgram(kind, name, ext, head) {
			return false, userfiles.ErrFileContentNotAllowed
		}
		// Office keeps charts as embedded workbooks (zip); a plain zip may not nest archives.
		if isOpaqueZip(kind) && !code && hasMagic(head, archiveMagic) {
			return false, userfiles.ErrFileContentNotAllowed
		}
	}
	switch kind {
	case kindFCSTD:
		return hasScripts, inspectFreeCAD(zr)
	case kindAPK:
		return true, inspectAPK(zr)
	}
	return hasScripts, nil
}

// Files a plain zip may hold although they are archives or programs. Each gets the checks it
// would get on its own: homework often comes as a folder with the project and the built app.
var nestedFormats = map[string]bool{".apk": true, ".aia": true, ".blend": true}

// inspectNested goes one level deep: inside an app or a project, archives are refused.
func inspectNested(f *zip.File, ext string, room *int64) (hasScripts bool, err error) {
	if ext == ".blend" {
		rc, err := f.Open()
		if err != nil {
			return false, userfiles.ErrFileContentNotAllowed
		}
		defer rc.Close()
		if !isBlend(rc) {
			return false, userfiles.ErrFileContentNotAllowed
		}
		return true, nil
	}
	*room -= int64(f.UncompressedSize64)
	if *room < 0 {
		return false, userfiles.ErrFileContentNotAllowed
	}
	body, err := readEntry(f, int64(f.UncompressedSize64))
	if err != nil {
		return false, userfiles.ErrFileContentNotAllowed
	}
	r := bytes.NewReader(body)
	kind, _, err := Classify("entry"+ext, body[:min(len(body), 512)], r, r.Size())
	if err != nil {
		return false, userfiles.ErrFileContentNotAllowed
	}
	return inspectContent(kind, r, r.Size())
}

// entryExt is the extension the file gets on disk: Windows drops trailing dots and spaces.
func entryExt(name string) string { return path.Ext(strings.TrimRight(name, ". ")) }

// unsafeZipPath: an entry that an unpacker would write outside the target folder ("zip slip":
// an absolute path, a drive letter or a ".." step, with either slash, also as ".. " on
// Windows), or whose name is not what it seems: an NTFS stream, a control or bidi character.
func unsafeZipPath(name string) bool {
	name = strings.ReplaceAll(name, `\`, "/")
	if strings.HasPrefix(name, "/") {
		return true
	}
	for _, r := range name {
		if r == ':' || unicode.IsControl(r) || isBidiControl(r) {
			return true
		}
	}
	for _, seg := range strings.Split(name, "/") {
		if strings.Contains(seg, "..") && strings.TrimRight(seg, ". ") == "" {
			return true
		}
	}
	return false
}

// refusedWebPage: Scratch and ScratchJr keep costumes as SVG, drawn by the editor and never
// served as a page; an App Inventor app shows its own pages from assets in a WebViewer.
func refusedWebPage(kind, name, ext string) bool {
	if !webPageExts[ext] {
		return false
	}
	switch kind {
	case kindSB3, kindSB2, kindSJR:
		return ext != ".svg"
	case kindAIA, kindAPK:
		return !strings.HasPrefix(name, "assets/")
	}
	return true
}

// nativeLibPath: the ABIs the App Inventor build packs libraries for.
var nativeLibPath = regexp.MustCompile(`^lib/(armeabi|armeabi-v7a|arm64-v8a|x86|x86_64)/[^/]+\.so$`)

// isAppCode: code in the one place an App Inventor build puts it, the app's dex files at the
// root and a project's extensions. The same file anywhere else is a program like any other.
func isAppCode(kind, name, ext string) bool {
	switch kind {
	case kindAPK:
		return ext == ".dex" && !strings.Contains(name, "/")
	case kindAIA:
		return (ext == ".jar" || ext == ".aar") && strings.HasPrefix(name, "assets/external_comps/")
	}
	return false
}

// allowedProgram: an app's native libraries, and a Python script's "#!" line.
func allowedProgram(kind, name, ext string, head []byte) bool {
	if pythonScriptExts[ext] && bytes.HasPrefix(head, []byte("#!")) {
		return true
	}
	return kind == kindAPK && bytes.HasPrefix(head, []byte("\x7fELF")) && nativeLibPath.MatchString(name)
}

// isOpaqueZip: a plain zip, or a SketchUp 2021+ model whose layout is not published, gets the
// strictest checks (no nested archives either). So does an app or a project: an archive in its
// assets is how an app would carry another app.
func isOpaqueZip(kind string) bool {
	switch kind {
	case kindZIP, kindSKP, kindAIA, kindAPK:
		return true
	}
	return false
}
