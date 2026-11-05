package usecase

import (
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"io"

	"github.com/klauspost/compress/zstd"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles"
)

// Android apps are stored only as App Inventor builds, for class work: its build writes the
// runtime's classes into every manifest. This is a filter against arbitrary APKs reaching
// children, not proof of origin: a crafted manifest passes, ClamAV and the ЛК warning remain.
const appInventorRuntime = "com.google.appinventor.components.runtime"

// The compiled manifest; real ones are a few kilobytes per screen.
const maxAPKManifest = 4 << 20

// Permissions that take over a phone and that no App Inventor component asks for.
var apkBlockedPermissions = []string{
	"android.permission.BIND_ACCESSIBILITY_SERVICE",
	"android.permission.BIND_DEVICE_ADMIN",
	"android.permission.BIND_NOTIFICATION_LISTENER_SERVICE",
	"android.permission.BIND_VPN_SERVICE",
	"android.permission.MANAGE_DEVICE_ADMINS",
	"android.permission.PACKAGE_USAGE_STATS",
	"android.permission.REQUEST_DELETE_PACKAGES",
	"android.permission.SYSTEM_ALERT_WINDOW",
	"android.permission.WRITE_SECURE_SETTINGS",
}

func inspectAPK(zr *zip.Reader) error {
	for _, f := range zr.File {
		if f.Name != "AndroidManifest.xml" {
			continue
		}
		manifest, err := readEntry(f, maxAPKManifest)
		if err != nil || !axmlHasString(manifest, appInventorRuntime) {
			return userfiles.ErrFileContentNotAllowed
		}
		for _, permission := range apkBlockedPermissions {
			if axmlHasString(manifest, permission) {
				return userfiles.ErrFileContentNotAllowed
			}
		}
		return nil
	}
	return userfiles.ErrFileContentNotAllowed
}

// axmlHasString searches a compiled manifest's string pool, which is UTF-16 or UTF-8.
func axmlHasString(axml []byte, s string) bool {
	if bytes.Contains(axml, []byte(s)) {
		return true
	}
	wide := make([]byte, 0, len(s)*2)
	for i := range len(s) {
		wide = append(wide, s[i], 0)
	}
	return bytes.Contains(axml, wide)
}

var (
	blendMagic = []byte("BLENDER")
	gzipMagic  = []byte{0x1F, 0x8B}
	zstdMagic  = []byte{0x28, 0xB5, 0x2F, 0xFD}
)

// Blender's own window is far smaller; a stream that asks for more is not a .blend.
const maxBlendWindow = 128 << 20

// isBlend: a .blend starts with "BLENDER"; Blender can compress the whole file with gzip or,
// since 3.0, zstd, so only the unpacked start tells.
func isBlend(r io.Reader) bool {
	br := bufio.NewReader(r)
	magic, err := br.Peek(len(zstdMagic))
	if err != nil {
		return false
	}
	var body io.Reader = br
	switch {
	case bytes.HasPrefix(magic, gzipMagic):
		zr, err := gzip.NewReader(br)
		if err != nil {
			return false
		}
		defer zr.Close()
		body = zr
	case bytes.Equal(magic, zstdMagic):
		zr, err := zstd.NewReader(br, zstd.WithDecoderConcurrency(1), zstd.WithDecoderLowmem(true),
			zstd.WithDecoderMaxWindow(maxBlendWindow))
		if err != nil {
			return false
		}
		defer zr.Close()
		body = zr
	}
	head := make([]byte, len(blendMagic))
	_, err = io.ReadFull(body, head)
	return err == nil && bytes.Equal(head, blendMagic)
}
