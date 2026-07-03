package barcode

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"strconv"

	bc "github.com/boombuler/barcode"
	"github.com/boombuler/barcode/aztec"
	"github.com/boombuler/barcode/codabar"
	"github.com/boombuler/barcode/code128"
	"github.com/boombuler/barcode/code39"
	"github.com/boombuler/barcode/code93"
	"github.com/boombuler/barcode/datamatrix"
	"github.com/boombuler/barcode/ean"
	"github.com/boombuler/barcode/pdf417"
	"github.com/boombuler/barcode/qr"
	"github.com/boombuler/barcode/twooffive"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// ErrTooSmall means the requested Size cannot fit the barcode at 1px/module.
var ErrTooSmall = errors.New("size is too small for the encoded barcode")

type GenerateOptions struct {
	Text   string
	Module int
	Quiet  int
	Level  string
	Size   int // if > 0, output is exactly Size x Size px and Module is ignored (2D only)
	Height int // bar height in px (1D only)
	// FullASCII enables Extended Code39/93 pair encoding (+N = n etc.).
	// Off by default: scanners without extended mode read the pairs literally.
	FullASCII bool
	Label     bool // draw human-readable content below the bars (1D only)
}

// ---- 2D (square modules, size param supported) ----

func GenerateDataMatrixPNG(opt GenerateOptions) ([]byte, error) {
	code, err := datamatrix.Encode(opt.Text)
	if err != nil {
		return nil, err
	}
	return renderPNG(code, opt)
}

func GenerateQRPNG(opt GenerateOptions) ([]byte, error) {
	level, err := qrLevel(opt.Level)
	if err != nil {
		return nil, err
	}
	code, err := qr.Encode(opt.Text, level, qr.Auto)
	if err != nil {
		return nil, err
	}
	return renderPNG(code, opt)
}

func GenerateAztecPNG(opt GenerateOptions) ([]byte, error) {
	code, err := aztec.Encode([]byte(opt.Text), 33, 0) // 33% ECC = library default
	if err != nil {
		return nil, err
	}
	return renderPNG(code, opt)
}

// GeneratePDF417PNG renders square modules; the library already emits rows
// 2 modules tall, giving a 2:1 row ratio.
// ponytail: spec prefers 3:1 rows — bump if scanners struggle.
func GeneratePDF417PNG(opt GenerateOptions) ([]byte, error) {
	sec := 2
	if opt.Level != "" {
		sec, _ = strconv.Atoi(opt.Level) // handler validates 0-8
	}
	code, err := pdf417.Encode(opt.Text, byte(sec))
	if err != nil {
		return nil, err
	}
	m := opt.Module
	return encodePNG(blockImage(code, m, m, opt.Quiet*m))
}

// ---- 1D (bar width Module px, bar height Height px) ----

func GenerateCode128PNG(opt GenerateOptions) ([]byte, error) {
	return generate1D(opt, code128.Encode)
}

func GenerateCode39PNG(opt GenerateOptions) ([]byte, error) {
	return generate1D(opt, func(s string) (bc.BarcodeIntCS, error) {
		return code39.Encode(s, false, opt.FullASCII) // no checksum
	})
}

func GenerateCode93PNG(opt GenerateOptions) ([]byte, error) {
	return generate1D(opt, func(s string) (bc.Barcode, error) {
		return code93.Encode(s, true, opt.FullASCII) // code93 checksum is mandatory
	})
}

func GenerateCodabarPNG(opt GenerateOptions) ([]byte, error) {
	return generate1D(opt, codabar.Encode)
}

// GenerateITFPNG encodes interleaved 2 of 5 (even number of digits).
func GenerateITFPNG(opt GenerateOptions) ([]byte, error) {
	return generate1D(opt, func(s string) (bc.Barcode, error) {
		return twooffive.Encode(s, true)
	})
}

// GenerateCode25PNG encodes standard (non-interleaved) 2 of 5.
func GenerateCode25PNG(opt GenerateOptions) ([]byte, error) {
	return generate1D(opt, func(s string) (bc.Barcode, error) {
		return twooffive.Encode(s, false)
	})
}

// ean.Encode picks EAN-8 or EAN-13 by length, so each endpoint pins its own
// valid lengths to avoid silently generating the other symbology.
func GenerateEAN13PNG(opt GenerateOptions) ([]byte, error) {
	if l := len(opt.Text); l != 12 && l != 13 {
		return nil, fmt.Errorf("ean13 needs 12 or 13 digits")
	}
	return generate1D(opt, ean.Encode)
}

func GenerateEAN8PNG(opt GenerateOptions) ([]byte, error) {
	if l := len(opt.Text); l != 7 && l != 8 {
		return nil, fmt.Errorf("ean8 needs 7 or 8 digits")
	}
	return generate1D(opt, ean.Encode)
}

func generate1D[T bc.Barcode](opt GenerateOptions, encode func(string) (T, error)) ([]byte, error) {
	code, err := encode(opt.Text)
	if err != nil {
		return nil, err
	}
	// 1D codes are 1 module tall, so scaling y by Height gives the bar height.
	img := blockImage(code, opt.Module, opt.Height, opt.Quiet*opt.Module)
	if opt.Label {
		// Content(), not opt.Text: includes computed check digits (EAN) and
		// is what a scanner will actually read.
		img = addLabel(img, code.Content(), opt.Module)
	}
	return encodePNG(img)
}

// addLabel appends the human-readable content below the bars, rendered with
// the built-in 7x13 bitmap font and integer-upscaled so it stays crisp.
func addLabel(bars *image.Gray, text string, module int) *image.Gray {
	face := basicfont.Face7x13
	textW := font.MeasureString(face, text).Ceil()
	if textW == 0 {
		return bars
	}
	scale := max(module, 2)
	for scale > 1 && textW*scale > bars.Rect.Dx() {
		scale--
	}

	small := image.NewGray(image.Rect(0, 0, textW, face.Height))
	whiteFill(small)
	(&font.Drawer{Dst: small, Src: image.Black, Face: face, Dot: fixed.P(0, face.Ascent)}).DrawString(text)

	barsH := bars.Rect.Dy()
	out := image.NewGray(image.Rect(0, 0, bars.Rect.Dx(), barsH+face.Height*scale+2*scale))
	whiteFill(out)
	copy(out.Pix, bars.Pix) // same width, so rows line up
	ox := (out.Rect.Dx() - textW*scale) / 2
	scaleInto(out, image.Rect(ox, barsH, ox+textW*scale, barsH+face.Height*scale), small)
	return out
}

func qrLevel(s string) (qr.ErrorCorrectionLevel, error) {
	switch s {
	case "", "M":
		return qr.M, nil
	case "L":
		return qr.L, nil
	case "Q":
		return qr.Q, nil
	case "H":
		return qr.H, nil
	}
	return 0, fmt.Errorf("invalid level: %s", s)
}

// blockImage draws the barcode with mx x my pixel modules surrounded by a
// white margin (px).
func blockImage(code bc.Barcode, mx, my, margin int) *image.Gray {
	src := toGray(code)
	b := src.Rect
	img := image.NewGray(image.Rect(0, 0, b.Dx()*mx+2*margin, b.Dy()*my+2*margin))
	whiteFill(img)
	scaleInto(img, image.Rect(margin, margin, margin+b.Dx()*mx, margin+b.Dy()*my), src)
	return img
}

// toGray copies the barcode into an image.Gray at 1px/module. The x/image
// scaler needs a Gray source: its dst-is-RGBA64Image path silently draws
// nothing for source types it doesn't know.
func toGray(code bc.Barcode) *image.Gray {
	b := code.Bounds()
	img := image.NewGray(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			img.Set(x, y, code.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return img
}

func whiteFill(img *image.Gray) {
	for i := range img.Pix {
		img.Pix[i] = 0xFF
	}
}

// scaleInto scales src into the dst rectangle with nearest-neighbor, which
// is exact for the integer factors used here — no interpolation, sharp edges.
func scaleInto(dst *image.Gray, r image.Rectangle, src image.Image) {
	xdraw.NearestNeighbor.Scale(dst, r, src, src.Bounds(), xdraw.Src, nil)
}

func encodePNG(img *image.Gray) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// renderPNG renders a square 2D code. With Size set, the largest integer
// module that fits is used and the barcode is centered on a Size x Size
// white canvas, so the output is exactly the requested size.
func renderPNG(code bc.Barcode, opt GenerateOptions) ([]byte, error) {
	if opt.Size <= 0 {
		m := opt.Module
		return encodePNG(blockImage(code, m, m, opt.Quiet*m))
	}
	src := toGray(code)
	b := src.Rect
	module := opt.Size / (max(b.Dx(), b.Dy()) + 2*opt.Quiet)
	if module < 1 {
		return nil, ErrTooSmall
	}
	img := image.NewGray(image.Rect(0, 0, opt.Size, opt.Size))
	whiteFill(img)
	ox := (opt.Size - b.Dx()*module) / 2
	oy := (opt.Size - b.Dy()*module) / 2
	scaleInto(img, image.Rect(ox, oy, ox+b.Dx()*module, oy+b.Dy()*module), src)
	return encodePNG(img)
}
