package usecase

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"io"
	"regexp"
	"strings"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles"
)

// A FreeCAD model is a zip with Document.xml and GuiDocument.xml. Python objects of the
// workbenches (Draft, BIM, FEM…) are stored there as <Python module=… class=… value=…/>, and
// FreeCAD imports that module while it opens the file: before 1.1.2 any module, so a file
// could name os or subprocess (CVE-2026-34789); a legacy pickled value runs code by itself.
// The BIM project manager eval()s the wpposition/wpu/wpv/wpaxis metadata (CVE-2026-34398).
// This is a filter for those paths, not a guarantee: an old FreeCAD stays the risk.

var freecadXMLEntries = map[string]bool{"Document.xml": true, "GuiDocument.xml": true}

// Both document files together, uncompressed. Shapes live in .brp entries, so real
// documents are far smaller; a larger one is refused rather than left unread.
const maxFreeCADXML = 64 << 20

var pythonModuleName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*$`)

// Top-level modules that run commands, load code, open files or the network, or do
// something on import (antigravity opens a browser). Workbench modules are not here.
var freecadBlockedModules = map[string]bool{
	"os": true, "posix": true, "nt": true, "posixpath": true, "ntpath": true, "subprocess": true,
	"sys": true, "builtins": true, "__builtin__": true, "__main__": true, "importlib": true,
	"imp": true, "runpy": true, "pty": true, "shutil": true, "socket": true, "ctypes": true,
	"code": true, "codeop": true, "pickle": true, "cPickle": true, "_pickle": true, "marshal": true,
	"shelve": true, "multiprocessing": true, "webbrowser": true, "antigravity": true, "this": true,
	"urllib": true, "urllib2": true, "urllib3": true, "requests": true, "http": true, "httplib": true,
	"ftplib": true, "smtplib": true, "telnetlib": true, "poplib": true, "imaplib": true,
	"xmlrpc": true, "pdb": true, "bdb": true, "platform": true, "commands": true, "popen2": true,
	"tempfile": true, "pathlib": true, "zipimport": true, "site": true, "sysconfig": true,
	"distutils": true, "setuptools": true, "pip": true, "ensurepip": true, "venv": true,
	"shlex": true, "glob": true, "signal": true, "asyncio": true, "concurrent": true,
	"threading": true, "_thread": true, "thread": true, "io": true, "_io": true, "fileinput": true,
	"tarfile": true, "zipfile": true, "ssl": true, "select": true, "selectors": true, "mmap": true,
	"fcntl": true, "resource": true, "pwd": true, "grp": true, "getpass": true, "inspect": true,
	"types": true, "gc": true, "operator": true, "functools": true, "timeit": true,
	"profile": true, "cProfile": true, "trace": true, "tkinter": true, "Tkinter": true,
	"idlelib": true, "turtle": true, "_posixsubprocess": true, "_winapi": true, "msvcrt": true,
	"winreg": true, "_winreg": true,
	// Qt can start processes (QProcess) and open URLs.
	"PySide": true, "PySide2": true, "PySide6": true, "PyQt4": true, "PyQt5": true, "PyQt6": true,
	"shiboken2": true, "shiboken6": true,
}

// Metadata the BIM project manager passes to eval().
var freecadEvalKeys = map[string]bool{"wpposition": true, "wpu": true, "wpv": true, "wpaxis": true}

// inspectFreeCAD reads the document files of an FCStd; the zip-level checks run in inspectZip.
func inspectFreeCAD(zr *zip.Reader) error {
	budget := int64(maxFreeCADXML)
	for _, f := range zr.File {
		if !freecadXMLEntries[f.Name] {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return userfiles.ErrFileContentNotAllowed
		}
		counted := &countingReader{r: io.LimitReader(rc, budget+1)}
		err = checkFreeCADXML(counted)
		rc.Close()
		budget -= counted.n
		if err != nil || budget < 0 {
			return userfiles.ErrFileContentNotAllowed
		}
	}
	return nil
}

// checkFreeCADXML walks the elements; attribute values come back with entities decoded, so
// module="o&#115;" is checked as "os". Go's decoder never loads external entities.
func checkFreeCADXML(r io.Reader) error {
	dec := xml.NewDecoder(r)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		el, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		attr := func(name string) (string, bool) {
			for _, a := range el.Attr {
				if a.Name.Local == name {
					return a.Value, true
				}
			}
			return "", false
		}
		switch el.Name.Local {
		case "Python":
			if err := checkPythonObject(attr); err != nil {
				return err
			}
		case "Item":
			if key, _ := attr("key"); freecadEvalKeys[strings.ToLower(key)] {
				return userfiles.ErrFileContentNotAllowed
			}
		}
	}
}

func checkPythonObject(attr func(string) (string, bool)) error {
	for _, name := range []string{"module", "class"} {
		v, ok := attr(name)
		if !ok || v == "" {
			continue
		}
		if !pythonModuleName.MatchString(v) {
			return userfiles.ErrFileContentNotAllowed
		}
		if name == "module" && freecadBlockedModules[strings.SplitN(v, ".", 2)[0]] {
			return userfiles.ErrFileContentNotAllowed
		}
	}
	value, _ := attr("value")
	if enc, _ := attr("encoded"); enc == "yes" {
		raw, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(value), ""))
		if err != nil {
			return userfiles.ErrFileContentNotAllowed
		}
		value = string(raw)
	}
	// FreeCAD 0.17+ stores the state as JSON; anything else is a pickle.
	if v := bytes.TrimLeft([]byte(value), " \t\r\n"); len(v) > 0 && !strings.ContainsRune(`{["-0123456789tfn`, rune(v[0])) {
		return userfiles.ErrFileContentNotAllowed
	}
	return nil
}
