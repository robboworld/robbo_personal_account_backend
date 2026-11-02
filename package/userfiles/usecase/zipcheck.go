package usecase

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"io"
	"io/fs"
	"path"
	"regexp"
	"strings"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles"
)

// maxZipEntries caps the central directory before archive/zip parses it: a 50 MB zip of
// empty entries would otherwise allocate ~120 MB per request. Large Scratch projects and
// Office files stay far below it.
const maxZipEntries = 20000

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
}

// Python scripts: allowed in a zip or a FreeCAD model (homework, workbench macros), but the
// file is flagged and the ЛК asks to run them only from a trusted source.
var pythonScriptExts = map[string]bool{".py": true, ".pyc": true, ".pyo": true, ".pyw": true, ".fcmacro": true}

// Pages a browser would run, by name; a Scratch project keeps its costumes as SVG.
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
		{0xFD, '7', 'z', 'X', 'Z'}, []byte("MSCF"),
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

func entryHead(f *zip.File) ([]byte, error) {
	if f.UncompressedSize64 == 0 {
		return nil, nil
	}
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
	return head[:n], nil
}

func readEntry(f *zip.File, limit int64) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(io.LimitReader(rc, limit))
}

// Office relationships that load something from outside the file when it opens: a remote
// template (macros), an OLE object, a frame or a sub-document.
var externalRelPattern = regexp.MustCompile(`(?i)<Relationship\b[^>]*\bType="[^"]*/(attachedTemplate|oleObject|frame|subDocument|package)"[^>]*\bTargetMode="External"|<Relationship\b[^>]*\bTargetMode="External"[^>]*\bType="[^"]*/(attachedTemplate|oleObject|frame|subDocument|package)"`)

// inspectZip rejects archives whose contents could run code when the owner opens them, and
// reports Python scripts inside. It reads the directory and at most a few bytes of each
// entry; nothing is unpacked.
func inspectZip(kind string, ra io.ReaderAt, size int64) (hasScripts bool, err error) {
	zr, err := openZip(ra, size)
	if err != nil {
		return false, err
	}
	for _, f := range zr.File {
		name := strings.ToLower(f.Name)
		ext := path.Ext(name)
		if f.Flags&0x1 != 0 { // encrypted: nothing inside can be checked
			return false, userfiles.ErrFileContentNotAllowed
		}
		if unsafeZipPath(f.Name) || f.Mode()&fs.ModeSymlink != 0 {
			return false, userfiles.ErrFileContentNotAllowed
		}
		if executableExts[ext] || (!isScratchZip(kind) && webPageExts[ext]) {
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
				body, err := readEntry(f, 1<<20)
				if err != nil {
					return false, userfiles.ErrFileContentNotAllowed
				}
				if bytes.Contains(bytes.ToLower(body), []byte("macroenabled")) || externalRelPattern.Match(body) {
					return false, userfiles.ErrFileContentNotAllowed
				}
			}
		case kindODT:
			if strings.HasPrefix(name, "basic/") || strings.HasPrefix(name, "scripts/") {
				return false, userfiles.ErrFileContentNotAllowed
			}
		}
		if pythonScriptExts[ext] {
			hasScripts = true
		}
		head, err := entryHead(f)
		if err != nil {
			return false, userfiles.ErrFileContentNotAllowed
		}
		if hasMagic(head, executableMagic) {
			return false, userfiles.ErrFileContentNotAllowed
		}
		// Office keeps charts as embedded workbooks (zip); a plain zip may not nest archives.
		if isOpaqueZip(kind) && hasMagic(head, archiveMagic) {
			return false, userfiles.ErrFileContentNotAllowed
		}
	}
	if kind == kindFCSTD {
		return hasScripts, inspectFreeCAD(zr)
	}
	return hasScripts, nil
}

// unsafeZipPath: an entry that an unpacker would write outside the target folder
// ("zip slip"): an absolute path, a drive letter or a ".." step, with either slash.
func unsafeZipPath(name string) bool {
	name = strings.ReplaceAll(name, `\`, "/")
	if strings.HasPrefix(name, "/") || len(name) >= 2 && name[1] == ':' {
		return true
	}
	for _, seg := range strings.Split(name, "/") {
		if seg == ".." {
			return true
		}
	}
	return false
}

// Scratch keeps costumes as SVG; they are drawn by the editor, never served as a page.
func isScratchZip(kind string) bool { return kind == kindSB3 || kind == kindSB2 }

// isOpaqueZip: a plain zip, or a SketchUp 2021+ model whose layout is not published, gets
// the strictest checks (no nested archives either).
func isOpaqueZip(kind string) bool { return kind == kindZIP || kind == kindSKP }
