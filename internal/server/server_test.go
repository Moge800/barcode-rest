package server

import (
	"bytes"
	"image"
	"image/png"
	"net/http/httptest"
	"strings"
	"testing"
)

var pngSig = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

// one valid request per barcode endpoint
var okPaths = []string{
	"/datamatrix?text=ABC123",
	"/qr?text=ABC123",
	"/aztec?text=ABC123",
	"/pdf417?text=ABC123",
	"/code128?text=ABC123",
	"/code39?text=ABC-123",
	"/code39?text=nullpo&fullascii=1",
	"/code93?text=nullpo&fullascii=1",
	"/code93?text=ABC123",
	"/codabar?text=A12345B",
	"/itf?text=1234567890",
	"/code25?text=12345",
	"/ean13?text=490123456789",
	"/ean8?text=4901234",
}

func do(t *testing.T, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	New(nil).ServeHTTP(rec, req)
	return rec
}

func TestOK(t *testing.T) {
	for _, path := range append([]string{"/health"}, okPaths...) {
		if rec := do(t, "GET", path); rec.Code != 200 {
			t.Errorf("GET %s = %d, want 200", path, rec.Code)
		}
	}
}

func TestPNGResponse(t *testing.T) {
	for _, path := range okPaths {
		rec := do(t, "GET", path)
		if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
			t.Errorf("GET %s Content-Type = %q, want image/png", path, ct)
		}
		if !bytes.HasPrefix(rec.Body.Bytes(), pngSig) {
			t.Errorf("GET %s body does not start with PNG signature", path)
			continue
		}
		// a barcode must contain black pixels — catches silently blank renders
		img, err := png.Decode(rec.Body)
		if err != nil {
			t.Errorf("GET %s: decode png: %v", path, err)
			continue
		}
		if !hasBlack(img) {
			t.Errorf("GET %s image is blank", path)
		}
	}
}

func hasBlack(img image.Image) bool {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if r, _, _, _ := img.At(x, y).RGBA(); r>>8 < 128 {
				return true
			}
		}
	}
	return false
}

func TestBadRequests(t *testing.T) {
	cases := []struct {
		path string
		want int
	}{
		{"/datamatrix", 400},
		{"/datamatrix?text=", 400},
		{"/datamatrix?text=" + strings.Repeat("a", maxDataMatrixBytes+1), 400}, // over byte cap
		// fits the byte cap but exceeds the symbol's capacity for this content:
		// must be a clean 400, not a 500.
		{"/datamatrix?text=" + strings.Repeat("A", 2000), 400},
		{"/qr?text=" + strings.Repeat("A", maxQRBytes+1), 400},          // over byte cap
		{"/qr?text=" + strings.Repeat("A", 2000) + "&level=H", 400},     // exceeds level-H capacity
		{"/aztec?text=" + strings.Repeat("A", maxAztecBytes+1), 400},    // over byte cap
		{"/pdf417?text=" + strings.Repeat("A", maxPDF417Bytes+1), 400},  // over byte cap
		{"/datamatrix?text=A&module=1", 400},
		{"/datamatrix?text=A&module=999", 400},
		{"/datamatrix?text=A&quiet=-1", 400},
		{"/qr?text=A&level=X", 400},
		{"/datamatrix?text=A&size=15", 400},
		{"/datamatrix?text=A&size=9999", 400},
		{"/datamatrix?text=ABC123456789&size=16", 400}, // barcode doesn't fit at 1px/module
		{"/code128?text=%E3%81%82", 400},               // non-ASCII not encodable
		{"/code128?text=A&size=255", 400},              // size unsupported for 1D
		{"/code128?text=A&height=5", 400},
		{"/ean13?text=hello", 400},
		{"/ean13?text=12345", 400},
		{"/ean13?text=4901234", 400}, // valid EAN-8 length must not fall through to EAN-8
		{"/ean8?text=490123456789", 400},
		{"/codabar?text=12345", 400}, // missing start/stop chars
		{"/itf?text=123", 400},       // odd number of digits
		{"/code39?text=%E3%81%82", 400},
		{"/code39?text=nullpo", 400}, // lowercase rejected unless fullascii=1
		{"/code39?text=A&fullascii=2", 400},
		{"/code128?text=A&label=2", 400},
		{"/code128?text=" + strings.Repeat("A", 81), 400}, // library limit is 80
		// max-parameter code39 would render ~55000px wide -> pixel cap
		{"/code39?text=" + strings.Repeat("a", 128) + "&fullascii=1&module=32&height=600", 400},
		// bars fit under the cap but the label re-allocation must not bypass it
		{"/code39?text=" + strings.Repeat("a", 128) + "&fullascii=1&module=32&height=150&quiet=0&label=1", 400},
		{"/ean13?text=490123456789&fullascii=1", 400}, // fullascii is code39/93 only
		{"/ean13?text=490123456789&fullascii=", 400},  // present-but-empty must also be rejected
		{"/code128?text=A&fullascii=0", 400},
		{"/pdf417?text=A&level=9", 400},
		{"/pdf417?text=A&size=255", 400},
		{"/nosuchpath", 404},
	}
	for _, c := range cases {
		if rec := do(t, "GET", c.path); rec.Code != c.want {
			t.Errorf("GET %s = %d, want %d", c.path, rec.Code, c.want)
		}
	}
}

func TestCode128MaxLength(t *testing.T) {
	if rec := do(t, "GET", "/code128?text="+strings.Repeat("A", 80)); rec.Code != 200 {
		t.Errorf("80 chars = %d, want 200", rec.Code)
	}
}

func TestSizeParam(t *testing.T) {
	for _, path := range []string{"/datamatrix?text=ABC123&size=255", "/qr?text=ABC123&size=255"} {
		rec := do(t, "GET", path)
		if rec.Code != 200 {
			t.Fatalf("GET %s = %d, want 200", path, rec.Code)
		}
		cfg, err := png.DecodeConfig(rec.Body)
		if err != nil {
			t.Fatalf("GET %s: decode png: %v", path, err)
		}
		if cfg.Width != 255 || cfg.Height != 255 {
			t.Errorf("GET %s image = %dx%d, want 255x255", path, cfg.Width, cfg.Height)
		}
	}
}

func TestLabel(t *testing.T) {
	plain := do(t, "GET", "/ean13?text=490123456789")
	labeled := do(t, "GET", "/ean13?text=490123456789&label=1")
	if labeled.Code != 200 {
		t.Fatalf("label=1 = %d, want 200", labeled.Code)
	}
	pc, err := png.DecodeConfig(plain.Body)
	if err != nil {
		t.Fatal(err)
	}
	lc, err := png.DecodeConfig(labeled.Body)
	if err != nil {
		t.Fatal(err)
	}
	if lc.Height <= pc.Height {
		t.Errorf("labeled height %d not taller than plain %d", lc.Height, pc.Height)
	}
	if lc.Width != pc.Width {
		t.Errorf("labeled width %d != plain width %d", lc.Width, pc.Width)
	}
}

// Large-but-valid payloads near each symbology's capacity must encode to a
// real PNG (200), not just be rejected — this is the case the old 128/256-byte
// caps could never reach.
func TestLargeInputEncodes(t *testing.T) {
	digits := func(n int) string { return strings.Repeat("1", n) }
	cases := []string{
		"/datamatrix?text=" + digits(3000),         // < 3116 numeric cap
		"/datamatrix?text=" + digits(3116),         // exactly the numeric cap
		"/qr?text=" + digits(3000),                 // default level M
		"/qr?text=" + digits(7089) + "&level=L",    // QR-L numeric cap
		"/aztec?text=" + digits(3000),              // < 3748
		"/pdf417?text=" + digits(2000),             // < 2610 at security level 2
	}
	for _, path := range cases {
		rec := do(t, "GET", path)
		if rec.Code != 200 {
			t.Errorf("GET %s = %d, want 200", path, rec.Code)
			continue
		}
		if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
			t.Errorf("GET %s Content-Type = %q, want image/png", path, ct)
		}
		if _, err := png.Decode(rec.Body); err != nil {
			t.Errorf("GET %s: decode png: %v", path, err)
		}
	}
}

func TestShutdown(t *testing.T) {
	called := 0
	h := New(func() { called++ })

	// GET must not stop the server (browser / preview / stray click).
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/shutdown", nil))
	if rec.Code != 405 {
		t.Errorf("GET /shutdown = %d, want 405", rec.Code)
	}
	if called != 0 {
		t.Error("GET /shutdown must not trigger shutdown")
	}

	// POST replies {"ok":true} and triggers shutdown exactly once.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/shutdown", nil))
	if rec.Code != 200 {
		t.Fatalf("POST /shutdown = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("POST /shutdown Content-Type = %q, want application/json; charset=utf-8", ct)
	}
	if body := strings.TrimSpace(rec.Body.String()); body != `{"ok":true}` {
		t.Errorf("POST /shutdown body = %q, want {\"ok\":true}", body)
	}
	if called != 1 {
		t.Errorf("POST /shutdown triggered shutdown %d times, want 1", called)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	for _, path := range []string{"/health", "/datamatrix?text=A", "/qr?text=A"} {
		if rec := do(t, "POST", path); rec.Code != 405 {
			t.Errorf("POST %s = %d, want 405", path, rec.Code)
		}
	}
}
