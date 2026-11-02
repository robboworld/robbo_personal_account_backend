package usecase

import (
	"bytes"
	"encoding/binary"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles"
)

// The whitelist. A file is accepted only when its contents are one of these kinds and its
// extension names that same kind. HTML, SVG, XML, scripts and executables are never stored:
// they are a way to run code in a child's browser or on their computer.
const (
	kindPNG  = "png"
	kindJPEG = "jpeg"
	kindGIF  = "gif"
	kindWEBP = "webp"
	kindPDF  = "pdf"
	kindTXT  = "txt"
	kindMP3  = "mp3"
	kindWAV  = "wav"
	kindOGG  = "ogg"
	kindZIP  = "zip"
	kindDOCX = "docx"
	kindXLSX = "xlsx"
	kindPPTX = "pptx"
	kindODT  = "odt"
	kindSB3  = "sb3"
	// Robbo course formats: Scratch 1.4 and 2, Arduino sketches, 3D models.
	kindSB    = "sb"
	kindSB2   = "sb2"
	kindINO   = "ino"
	kindSTL   = "stl"
	kindFCSTD = "fcstd"
	kindSKP   = "skp"
)

var kindByExt = map[string]string{
	".png": kindPNG, ".jpg": kindJPEG, ".jpeg": kindJPEG, ".gif": kindGIF, ".webp": kindWEBP,
	".pdf": kindPDF, ".txt": kindTXT, ".mp3": kindMP3, ".wav": kindWAV, ".ogg": kindOGG,
	".zip": kindZIP, ".docx": kindDOCX, ".xlsx": kindXLSX, ".pptx": kindPPTX, ".odt": kindODT,
	".sb3": kindSB3, ".sb": kindSB, ".sb2": kindSB2, ".ino": kindINO, ".stl": kindSTL,
	".fcstd": kindFCSTD, ".skp": kindSKP,
}

// The stored and served type comes from here, never from the client.
var mimeByKind = map[string]string{
	kindPNG:  "image/png",
	kindJPEG: "image/jpeg",
	kindGIF:  "image/gif",
	kindWEBP: "image/webp",
	kindPDF:  "application/pdf",
	kindTXT:  "text/plain",
	kindMP3:  "audio/mpeg",
	kindWAV:  "audio/wav",
	kindOGG:  "audio/ogg",
	kindZIP:  "application/zip",
	kindDOCX: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	kindXLSX: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	kindPPTX: "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	kindODT:  "application/vnd.oasis.opendocument.text",
	kindSB3:  "application/x.scratch.sb3",
	kindSB:   "application/x.scratch.sb",
	kindSB2:  "application/x.scratch.sb2",
	kindINO:  "text/plain",
	kindSTL:  "model/stl",
	// FreeCAD has no registered type; this one is never sniffed into something a browser runs.
	kindFCSTD: "application/x.freecad.fcstd",
	kindSKP:   "application/vnd.sketchup.skp",
}

// kindForName is the whitelist kind the file name claims, "" when it is not allowed.
func kindForName(name string) string {
	return kindByExt[strings.ToLower(filepath.Ext(name))]
}

// Classify checks the contents against the kind the extension claims and returns the kind and
// its MIME type. head is the start of the file (up to 512 bytes), ra/size the whole of it.
func Classify(name string, head []byte, ra io.ReaderAt, size int64) (kind, mime string, err error) {
	claimed := kindForName(name)
	if claimed == "" || size == 0 {
		return "", "", userfiles.ErrFileTypeNotAllowed
	}
	// STL has no signature: a binary one may start with anything, even "solid".
	if claimed == kindSTL {
		if !isSTL(head, ra, size) {
			return "", "", userfiles.ErrFileTypeNotAllowed
		}
		return claimed, mimeByKind[claimed], nil
	}
	detected, err := detectKind(head, ra, size)
	if err != nil {
		return "", "", err
	}
	if detected == "" || !compatible(claimed, detected) {
		return "", "", userfiles.ErrFileTypeNotAllowed
	}
	return claimed, mimeByKind[claimed], nil
}

// compatible: an Office document, a Scratch project or a FreeCAD model is also a zip, so
// .zip accepts them. Scratch 2 and 3 projects look alike (project.json at the root); an
// Arduino sketch is plain text; SketchUp 2021+ saves a zip whose layout is not published.
func compatible(claimed, detected string) bool {
	if claimed == detected {
		return true
	}
	switch claimed {
	case kindZIP:
		switch detected {
		case kindDOCX, kindXLSX, kindPPTX, kindODT, kindSB3, kindFCSTD:
			return true
		}
	case kindSB2:
		return detected == kindSB3
	case kindINO:
		return detected == kindTXT
	case kindSKP:
		return detected == kindZIP
	}
	return false
}

func detectKind(head []byte, ra io.ReaderAt, size int64) (string, error) {
	ct := http.DetectContentType(head)
	if ct == "application/zip" {
		return zipKind(ra, size)
	}
	return plainKind(ct, head, ra, size), nil
}

var (
	// Scratch 1.4: "ScratchV01" or "ScratchV02", then a Squeak object stream.
	sbMagic = []byte("ScratchV0")
	// SketchUp up to 2020: FF FE FF, the name length (14), then "SketchUp Model" in UTF-16LE.
	skpMagic = []byte("\xFF\xFE\xFF\x0ES\x00k\x00e\x00t\x00c\x00h\x00U\x00p\x00 \x00M\x00o\x00d\x00e\x00l\x00")
)

func plainKind(ct string, head []byte, ra io.ReaderAt, size int64) string {
	switch {
	// Before the text check: the SketchUp header reads as a UTF-16 BOM.
	case bytes.HasPrefix(head, sbMagic):
		return kindSB
	case bytes.HasPrefix(head, skpMagic):
		return kindSKP
	case ct == "image/png":
		return kindPNG
	case ct == "image/jpeg":
		return kindJPEG
	case ct == "image/gif":
		return kindGIF
	case ct == "image/webp":
		return kindWEBP
	case ct == "application/pdf":
		return kindPDF
	case ct == "audio/wave":
		return kindWAV
	case ct == "application/ogg":
		return kindOGG
	case ct == "audio/mpeg" || isMP3Frame(head):
		return kindMP3
	case strings.HasPrefix(ct, "text/plain"):
		if isPlainText(ra, size) {
			return kindTXT
		}
	}
	return ""
}

// isMP3Frame: an MP3 without an ID3 tag starts with an MPEG audio frame sync.
func isMP3Frame(head []byte) bool {
	return len(head) >= 2 && head[0] == 0xFF && head[1]&0xE0 == 0xE0 && head[1]&0x06 != 0
}

// isPlainText scans the whole file: DetectContentType sees only 512 bytes.
func isPlainText(ra io.ReaderAt, size int64) bool {
	buf := make([]byte, 64<<10)
	for off := int64(0); off < size; {
		n, err := ra.ReadAt(buf, off)
		if bytes.IndexByte(buf[:n], 0) >= 0 {
			return false
		}
		off += int64(n)
		if err != nil {
			return err == io.EOF && off >= size
		}
	}
	return true
}

const odtMimetype = "application/vnd.oasis.opendocument.text"

// zipKind reads only the central directory (and the 39-byte ODF mimetype entry).
func zipKind(ra io.ReaderAt, size int64) (string, error) {
	zr, err := openZip(ra, size)
	if err != nil {
		return "", err
	}
	has := func(name string) bool {
		for _, f := range zr.File {
			if f.Name == name {
				return true
			}
		}
		return false
	}
	hasPrefix := func(prefix string) bool {
		for _, f := range zr.File {
			if strings.HasPrefix(f.Name, prefix) {
				return true
			}
		}
		return false
	}
	if has("[Content_Types].xml") {
		switch {
		case hasPrefix("word/"):
			return kindDOCX, nil
		case hasPrefix("xl/"):
			return kindXLSX, nil
		case hasPrefix("ppt/"):
			return kindPPTX, nil
		}
	}
	for _, f := range zr.File {
		if f.Name != "mimetype" {
			continue
		}
		if rc, err := f.Open(); err == nil {
			b, _ := io.ReadAll(io.LimitReader(rc, 64))
			rc.Close()
			if strings.TrimSpace(string(b)) == odtMimetype {
				return kindODT, nil
			}
		}
	}
	if has("Document.xml") {
		return kindFCSTD, nil
	}
	if has("project.json") {
		return kindSB3, nil
	}
	return kindZIP, nil
}

// isSTL: a binary STL is an 80-byte header, a triangle count and 50 bytes per triangle, so
// its size is exact; an ASCII one is plain text that starts with "solid".
func isSTL(head []byte, ra io.ReaderAt, size int64) bool {
	if size >= stlBinaryHeader {
		var count [4]byte
		if _, err := ra.ReadAt(count[:], 80); err == nil &&
			int64(binary.LittleEndian.Uint32(count[:]))*stlTriangleLen+stlBinaryHeader == size {
			return true
		}
	}
	text := bytes.TrimLeft(head, " \t\r\n")
	return len(text) >= 5 && strings.EqualFold(string(text[:5]), "solid") && isPlainText(ra, size)
}

const (
	stlBinaryHeader = 84
	stlTriangleLen  = 50
)
