package usecase

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles"
	"golang.org/x/image/webp"
)

// Images are decoded and encoded again before they are stored. Whatever was appended to or
// hidden in the original (an HTML or PHP payload, a crafted chunk aimed at a viewer bug,
// EXIF with the GPS position of a child's photo) does not survive: only the pixels do.
const (
	// One frame: 25 MP decodes to at most 100 MB (RGBA).
	maxImagePixels = 25_000_000
	// All frames of a GIF together.
	maxGIFPixels = 100_000_000
	maxGIFFrames = 2000
	jpegQuality  = 90
)

func isImageKind(kind string) bool {
	switch kind {
	case kindPNG, kindJPEG, kindGIF, kindWEBP:
		return true
	}
	return false
}

// storedImageKind: WebP has no encoder in Go, so it is stored as PNG.
func storedImageKind(kind string) string {
	if kind == kindWEBP {
		return kindPNG
	}
	return kind
}

var errImage = userfiles.ErrFileContentNotAllowed

func checkPixels(w, h int, limit int64) error {
	if w <= 0 || h <= 0 || int64(w)*int64(h) > limit {
		return errImage
	}
	return nil
}

// reencodeImage writes a clean copy of the image to w in storedImageKind(kind).
func reencodeImage(kind string, ra io.ReaderAt, size int64, w io.Writer) error {
	sr := func() *io.SectionReader { return io.NewSectionReader(ra, 0, size) }
	switch kind {
	case kindGIF:
		if err := checkGIF(sr()); err != nil {
			return err
		}
		g, err := gif.DecodeAll(sr())
		if err != nil {
			return errImage
		}
		return gif.EncodeAll(w, g)
	case kindPNG, kindWEBP, kindJPEG:
		var cfg image.Config
		var err error
		switch kind {
		case kindPNG:
			cfg, err = png.DecodeConfig(sr())
		case kindWEBP:
			cfg, err = webp.DecodeConfig(sr())
		default:
			cfg, err = jpeg.DecodeConfig(sr())
		}
		if err != nil {
			return errImage
		}
		if err := checkPixels(cfg.Width, cfg.Height, maxImagePixels); err != nil {
			return err
		}
		var img image.Image
		switch kind {
		case kindPNG:
			img, err = png.Decode(sr())
		case kindWEBP:
			img, err = webp.Decode(sr())
		default:
			img, err = jpeg.Decode(sr())
		}
		if err != nil {
			return errImage
		}
		if kind == kindJPEG {
			// EXIF goes away, so turn the pixels the way the camera meant them to be shown.
			img = applyOrientation(img, jpegOrientation(sr()))
			return jpeg.Encode(w, img, &jpeg.Options{Quality: jpegQuality})
		}
		return png.Encode(w, img)
	}
	return errImage
}

// checkGIF walks the GIF blocks without decoding: gif.DecodeAll allocates every frame, so a
// small file with thousands of large frames would otherwise exhaust memory.
func checkGIF(r io.Reader) error {
	br := bufio.NewReader(r)
	hdr := make([]byte, 13)
	if _, err := io.ReadFull(br, hdr); err != nil {
		return errImage
	}
	if err := checkPixels(int(binary.LittleEndian.Uint16(hdr[6:8])), int(binary.LittleEndian.Uint16(hdr[8:10])), maxImagePixels); err != nil {
		return err
	}
	skipTable := func(flags byte) error {
		if flags&0x80 == 0 {
			return nil
		}
		_, err := br.Discard(3 << ((flags & 7) + 1))
		return err
	}
	skipSubBlocks := func() error {
		for {
			n, err := br.ReadByte()
			if err != nil {
				return err
			}
			if n == 0 {
				return nil
			}
			if _, err := br.Discard(int(n)); err != nil {
				return err
			}
		}
	}
	if err := skipTable(hdr[10]); err != nil {
		return errImage
	}
	var frames, total int64
	for {
		b, err := br.ReadByte()
		if err != nil {
			return errImage
		}
		switch b {
		case 0x3B: // trailer
			return nil
		case 0x21: // extension
			if _, err := br.ReadByte(); err != nil {
				return errImage
			}
			if err := skipSubBlocks(); err != nil {
				return errImage
			}
		case 0x2C: // image descriptor
			desc := make([]byte, 9)
			if _, err := io.ReadFull(br, desc); err != nil {
				return errImage
			}
			fw, fh := int64(binary.LittleEndian.Uint16(desc[4:6])), int64(binary.LittleEndian.Uint16(desc[6:8]))
			frames++
			total += fw * fh
			if frames > maxGIFFrames || total > maxGIFPixels {
				return errImage
			}
			if err := skipTable(desc[8]); err != nil {
				return errImage
			}
			if _, err := br.ReadByte(); err != nil { // LZW minimum code size
				return errImage
			}
			if err := skipSubBlocks(); err != nil {
				return errImage
			}
		default:
			return errImage
		}
	}
}

// jpegOrientation reads the EXIF Orientation tag (1–8); 1 when there is none.
func jpegOrientation(r io.Reader) int {
	br := bufio.NewReader(r)
	if soi := make([]byte, 2); readFull(br, soi) != nil || soi[0] != 0xFF || soi[1] != 0xD8 {
		return 1
	}
	for {
		m := make([]byte, 4)
		if readFull(br, m) != nil || m[0] != 0xFF {
			return 1
		}
		marker, segLen := m[1], int(binary.BigEndian.Uint16(m[2:4]))
		if segLen < 2 || marker == 0xDA { // start of scan: no EXIF before the pixels
			return 1
		}
		seg := make([]byte, segLen-2)
		if readFull(br, seg) != nil {
			return 1
		}
		if marker == 0xE1 && bytes.HasPrefix(seg, []byte("Exif\x00\x00")) {
			return exifOrientation(seg[6:])
		}
	}
}

func readFull(r io.Reader, b []byte) error {
	_, err := io.ReadFull(r, b)
	return err
}

func exifOrientation(tiff []byte) int {
	if len(tiff) < 8 {
		return 1
	}
	var bo binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 1
	}
	ifd := int(bo.Uint32(tiff[4:8]))
	if ifd < 8 || ifd+2 > len(tiff) {
		return 1
	}
	n := int(bo.Uint16(tiff[ifd : ifd+2]))
	for i := 0; i < n; i++ {
		e := ifd + 2 + i*12
		if e+12 > len(tiff) {
			return 1
		}
		if bo.Uint16(tiff[e:e+2]) == 0x0112 {
			if v := int(bo.Uint16(tiff[e+8 : e+10])); v >= 1 && v <= 8 {
				return v
			}
			return 1
		}
	}
	return 1
}

// applyOrientation turns or mirrors img per EXIF Orientation.
func applyOrientation(img image.Image, o int) image.Image {
	if o <= 1 || o > 8 {
		return img
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	dw, dh := w, h
	if o >= 5 {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var dx, dy int
			switch o {
			case 2:
				dx, dy = w-1-x, y
			case 3:
				dx, dy = w-1-x, h-1-y
			case 4:
				dx, dy = x, h-1-y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = h-1-y, x
			case 7:
				dx, dy = h-1-y, w-1-x
			case 8:
				dx, dy = y, w-1-x
			}
			dst.Set(dx, dy, img.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}
