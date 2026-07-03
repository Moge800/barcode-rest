package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"barcode-rest/internal/barcode"
	"barcode-rest/internal/response"
)

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]any{
		"ok":      true,
		"name":    "barcode-rest",
		"version": Version,
	})
}

func handleDataMatrix(w http.ResponseWriter, r *http.Request) {
	opt, ok := parseOptions(w, r, 128, 10, 4)
	if !ok {
		return
	}
	writePNG(w, opt, barcode.GenerateDataMatrixPNG, "failed to encode datamatrix", http.StatusInternalServerError)
}

func handleQR(w http.ResponseWriter, r *http.Request) {
	opt, ok := parseOptions(w, r, 256, 10, 4)
	if !ok {
		return
	}
	opt.Level = r.URL.Query().Get("level")
	switch opt.Level {
	case "", "L", "M", "Q", "H":
	default:
		response.WriteError(w, http.StatusBadRequest, "level must be one of L, M, Q, H")
		return
	}
	writePNG(w, opt, barcode.GenerateQRPNG, "failed to encode qr", http.StatusInternalServerError)
}

func handleAztec(w http.ResponseWriter, r *http.Request) {
	opt, ok := parseOptions(w, r, 256, 10, 4)
	if !ok {
		return
	}
	writePNG(w, opt, barcode.GenerateAztecPNG, "failed to encode aztec", http.StatusInternalServerError)
}

// PDF417 is not square, so size is unsupported; level is the security level 0-8.
func handlePDF417(w http.ResponseWriter, r *http.Request) {
	opt, ok := parseOptions(w, r, 256, 3, 2)
	if !ok {
		return
	}
	if opt.Size > 0 {
		response.WriteError(w, http.StatusBadRequest, "size is not supported for pdf417, use module")
		return
	}
	opt.Level = r.URL.Query().Get("level")
	switch opt.Level {
	case "", "0", "1", "2", "3", "4", "5", "6", "7", "8":
	default:
		response.WriteError(w, http.StatusBadRequest, "level must be an integer between 0 and 8")
		return
	}
	writePNG(w, opt, barcode.GeneratePDF417PNG, "text is too long for pdf417", http.StatusBadRequest)
}

// 1D barcodes: module is bar width, quiet default 10 modules per spec,
// encode failures are caused by the input so they map to 400.
func handle1D(gen func(barcode.GenerateOptions) ([]byte, error), maxTextBytes int, errMsg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		opt, ok := parse1DOptions(w, r, maxTextBytes)
		if !ok {
			return
		}
		q := r.URL.Query()
		if opt.FullASCII, ok = parseBoolParam(w, q.Get("fullascii"), "fullascii"); !ok {
			return
		}
		if opt.Label, ok = parseBoolParam(w, q.Get("label"), "label"); !ok {
			return
		}
		writePNG(w, opt, gen, errMsg, http.StatusBadRequest)
	}
}

func parse1DOptions(w http.ResponseWriter, r *http.Request, maxTextBytes int) (barcode.GenerateOptions, bool) {
	opt, ok := parseOptions(w, r, maxTextBytes, 3, 10)
	if !ok {
		return opt, false
	}
	if opt.Size > 0 {
		response.WriteError(w, http.StatusBadRequest, "size is not supported for 1D barcodes, use module and height")
		return opt, false
	}
	if opt.Height, ok = parseIntParam(w, r.URL.Query().Get("height"), "height", 80, 20, 600); !ok {
		return opt, false
	}
	return opt, true
}

func parseOptions(w http.ResponseWriter, r *http.Request, maxTextBytes, moduleDef, quietDef int) (barcode.GenerateOptions, bool) {
	var opt barcode.GenerateOptions
	if r.Method != http.MethodGet {
		response.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return opt, false
	}
	q := r.URL.Query()

	opt.Text = q.Get("text")
	if opt.Text == "" {
		response.WriteError(w, http.StatusBadRequest, "text is required")
		return opt, false
	}
	if len(opt.Text) > maxTextBytes {
		response.WriteError(w, http.StatusBadRequest, fmt.Sprintf("text must be at most %d bytes", maxTextBytes))
		return opt, false
	}

	var ok bool
	if opt.Module, ok = parseIntParam(w, q.Get("module"), "module", moduleDef, 2, 32); !ok {
		return opt, false
	}
	if opt.Quiet, ok = parseIntParam(w, q.Get("quiet"), "quiet", quietDef, 0, 16); !ok {
		return opt, false
	}
	if opt.Size, ok = parseIntParam(w, q.Get("size"), "size", 0, 16, 2048); !ok {
		return opt, false
	}
	return opt, true
}

func parseBoolParam(w http.ResponseWriter, s, name string) (bool, bool) {
	switch s {
	case "", "0":
		return false, true
	case "1":
		return true, true
	}
	response.WriteError(w, http.StatusBadRequest, name+" must be 0 or 1")
	return false, false
}

func parseIntParam(w http.ResponseWriter, s, name string, def, min, max int) (int, bool) {
	if s == "" {
		return def, true
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < min || v > max {
		response.WriteError(w, http.StatusBadRequest, fmt.Sprintf("%s must be an integer between %d and %d", name, min, max))
		return 0, false
	}
	return v, true
}

// genSem bounds concurrent generations: with the pixel cap each render can
// still allocate ~16MB, so unbounded parallelism could exhaust memory.
var genSem = make(chan struct{}, 4)

func writePNG(w http.ResponseWriter, opt barcode.GenerateOptions, gen func(barcode.GenerateOptions) ([]byte, error), errMsg string, errStatus int) {
	genSem <- struct{}{}
	png, err := gen(opt)
	<-genSem
	if errors.Is(err, barcode.ErrTooSmall) || errors.Is(err, barcode.ErrTooLarge) {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		response.WriteError(w, errStatus, errMsg)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(png)
}
