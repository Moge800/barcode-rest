package server

import (
	"log"
	"net/http"

	"barcode-rest/internal/barcode"
)

const Version = "0.1.0"

func New() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/datamatrix", handleDataMatrix)
	mux.HandleFunc("/qr", handleQR)
	mux.HandleFunc("/aztec", handleAztec)
	mux.HandleFunc("/pdf417", handlePDF417)
	mux.HandleFunc("/code128", handle1D(barcode.GenerateCode128PNG, 128, "text contains characters not encodable as code128"))
	mux.HandleFunc("/code39", handle1D(barcode.GenerateCode39PNG, 128, "text contains characters not encodable as code39 (try fullascii=1 for lowercase etc.)"))
	mux.HandleFunc("/code93", handle1D(barcode.GenerateCode93PNG, 128, "text contains characters not encodable as code93 (try fullascii=1 for lowercase etc.)"))
	mux.HandleFunc("/codabar", handle1D(barcode.GenerateCodabarPNG, 128, "text must be codabar format: start/stop A-D with digits or -$:/.+ between"))
	mux.HandleFunc("/itf", handle1D(barcode.GenerateITFPNG, 128, "text must be an even number of digits"))
	mux.HandleFunc("/code25", handle1D(barcode.GenerateCode25PNG, 128, "text must be digits"))
	mux.HandleFunc("/ean13", handle1D(barcode.GenerateEAN13PNG, 13, "text must be a valid 12 or 13 digit EAN-13/JAN code"))
	mux.HandleFunc("/ean8", handle1D(barcode.GenerateEAN8PNG, 8, "text must be a valid 7 or 8 digit EAN-8 code"))
	return logRequests(mux)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// logRequests logs method, path and status only — never query values,
// because text may contain shipping/production data.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("%s %s %d", r.Method, r.URL.Path, rec.status)
	})
}
