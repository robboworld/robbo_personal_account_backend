package usecase

import (
	"bufio"
	"compress/zlib"
	"io"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles"
)

// PDF names that run code, launch programs, carry other files or come from interactive
// forms; a document with any of them is refused. Encrypted PDFs are refused too: their
// object streams cannot be read.
var pdfBlockedNames = map[string]bool{
	"JavaScript": true, "JS": true, "Launch": true, "EmbeddedFile": true, "EmbeddedFiles": true,
	"RichMedia": true, "XFA": true, "GoToE": true, "Encrypt": true, "EF": true, "FileAttachment": true,
}

const (
	// Object streams hold dictionaries only; real ones are small.
	maxObjStmInflate = 8 << 20
	maxPDFInflate    = 64 << 20
)

func isPDFDelimiter(b byte) bool {
	switch b {
	case '(', ')', '<', '>', '[', ']', '{', '}', '/', '%':
		return true
	}
	return isPDFSpace(b)
}

func isPDFSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r' || b == '\n' || b == '\f' || b == 0
}

func unhex(b byte) (byte, bool) {
	switch {
	case b >= '0' && b <= '9':
		return b - '0', true
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10, true
	case b >= 'A' && b <= 'F':
		return b - 'A' + 10, true
	}
	return 0, false
}

// pdfScanner walks PDF tokens: names (with #xx escapes decoded) and bare keywords. It looks
// into compressed object streams, where PDF 1.5+ keeps most dictionaries.
type pdfScanner struct {
	inflated int64
	objStm   bool // the current object is an object stream
	params   bool // the current object has DecodeParms: a predictor changes the inflated bytes
	inType   bool // the last name was /Type
}

// scan returns ErrFileContentNotAllowed at the first blocked name. Comments and strings are
// scanned like everything else: skipping them needs a parser that agrees with every reader,
// and a "%" inside a string would hide the rest of its line.
func (s *pdfScanner) scan(r *bufio.Reader, nested bool) error {
	for {
		b, err := r.ReadByte()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		switch {
		case b == '/':
			name, err := readPDFName(r)
			if err != nil && err != io.EOF {
				return err
			}
			if pdfBlockedNames[name] {
				return userfiles.ErrFileContentNotAllowed
			}
			switch name {
			case "ObjStm":
				s.objStm = true
			case "DecodeParms", "DP":
				s.params = true
			}
			s.inType = name == "Type"
		case !isPDFDelimiter(b):
			word := readPDFWord(r, b)
			// "/Type 5 0 R": the type of a stream, hidden in another object, could be ObjStm.
			if s.inType && b >= '0' && b <= '9' {
				return userfiles.ErrFileContentNotAllowed
			}
			s.inType = false
			switch word {
			case "endobj":
				s.objStm, s.params = false, false
			case "stream":
				if s.objStm && !nested {
					if err := s.scanObjStm(r); err != nil {
						return err
					}
				}
			}
		}
	}
}

func readPDFName(r *bufio.Reader) (string, error) {
	var name []byte
	for len(name) < 127 {
		b, err := r.ReadByte()
		if err != nil {
			return string(name), err
		}
		if isPDFDelimiter(b) {
			return string(name), r.UnreadByte()
		}
		if b == '#' {
			pk, err := r.Peek(2)
			if err == nil {
				hi, ok1 := unhex(pk[0])
				lo, ok2 := unhex(pk[1])
				if ok1 && ok2 {
					r.Discard(2)
					b = hi<<4 | lo
				}
			}
		}
		name = append(name, b)
	}
	return string(name), nil
}

func readPDFWord(r *bufio.Reader, first byte) string {
	word := []byte{first}
	for len(word) < 16 {
		b, err := r.ReadByte()
		if err != nil {
			break
		}
		if isPDFDelimiter(b) {
			r.UnreadByte()
			break
		}
		word = append(word, b)
	}
	return string(word)
}

// scanObjStm inflates the object stream that starts after "stream" and scans its contents.
// Anything but plain Flate is refused: other filters and predictors would need decoding too,
// and real object streams are plain Flate.
func (s *pdfScanner) scanObjStm(r *bufio.Reader) error {
	if s.params {
		return userfiles.ErrFileContentNotAllowed
	}
	s.objStm = false
	// "stream" is followed by CRLF or LF.
	if b, _ := r.Peek(1); len(b) == 1 && b[0] == '\r' {
		r.Discard(1)
	}
	if b, _ := r.Peek(1); len(b) == 1 && b[0] == '\n' {
		r.Discard(1)
	}
	zr, err := zlib.NewReader(r)
	if err != nil {
		return userfiles.ErrFileContentNotAllowed
	}
	defer zr.Close()
	budget := min(int64(maxObjStmInflate), maxPDFInflate-s.inflated)
	if budget <= 0 {
		return userfiles.ErrFileContentNotAllowed
	}
	counted := &countingReader{r: io.LimitReader(zr, budget+1)}
	inner := &pdfScanner{}
	err = inner.scan(bufio.NewReader(counted), true)
	s.inflated += counted.n
	if err != nil {
		if err == userfiles.ErrFileContentNotAllowed {
			return err
		}
		// A broken stream: the reader cannot open it either.
		return nil
	}
	if counted.n > budget {
		return userfiles.ErrFileContentNotAllowed
	}
	return nil
}

type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

// checkPDF refuses PDFs with scripts, launch actions, attached files, XFA forms or encryption.
// It is a filter, not a parser: a PDF built to hide these from it may pass, which is why
// files are only ever served as attachments.
func checkPDF(ra io.ReaderAt, size int64) error {
	s := &pdfScanner{}
	err := s.scan(bufio.NewReaderSize(io.NewSectionReader(ra, 0, size), 64<<10), false)
	if err == userfiles.ErrFileContentNotAllowed {
		return err
	}
	if err != nil {
		return userfiles.ErrFileContentNotAllowed
	}
	return nil
}
