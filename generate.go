package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"barcode-rest/internal/barcode"
)

// CLI mode: generate one PNG without running the server. Unlike the HTTP
// API (which never accepts file paths, by design), the CLI writes exactly
// where the user asks — a different trust boundary.
//
//	barcode-rest generate datamatrix --text ABC123 --size 256 --output dm.png

type symbology struct {
	gen       func(barcode.GenerateOptions) ([]byte, error)
	module    int  // default pixels per module
	quiet     int  // default quiet zone in modules
	oneD      bool // has height/label flags
	fullASCII bool // has fullascii flag (code39/93 only)
}

var symbologies = map[string]symbology{
	"datamatrix": {barcode.GenerateDataMatrixPNG, 10, 4, false, false},
	"qr":         {barcode.GenerateQRPNG, 10, 4, false, false},
	"aztec":      {barcode.GenerateAztecPNG, 10, 4, false, false},
	"pdf417":     {barcode.GeneratePDF417PNG, 3, 2, false, false},
	"code128":    {barcode.GenerateCode128PNG, 3, 10, true, false},
	"code39":     {barcode.GenerateCode39PNG, 3, 10, true, true},
	"code93":     {barcode.GenerateCode93PNG, 3, 10, true, true},
	"codabar":    {barcode.GenerateCodabarPNG, 3, 10, true, false},
	"itf":        {barcode.GenerateITFPNG, 3, 10, true, false},
	"code25":     {barcode.GenerateCode25PNG, 3, 10, true, false},
	"ean13":      {barcode.GenerateEAN13PNG, 3, 10, true, false},
	"ean8":       {barcode.GenerateEAN8PNG, 3, 10, true, false},
}

func generateUsage() int {
	names := make([]string, 0, len(symbologies))
	for n := range symbologies {
		names = append(names, n)
	}
	sort.Strings(names)
	fmt.Fprintln(os.Stderr, `usage: barcode-rest generate <symbology> --text <text> --output <path|->`)
	fmt.Fprintln(os.Stderr, "symbologies: "+strings.Join(names, ", "))
	return 2
}

func runGenerate(args []string) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return generateUsage()
	}
	name := args[0]
	sym, ok := symbologies[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown symbology: %s\n", name)
		return generateUsage()
	}

	// ContinueOnError: parse failures return instead of os.Exit, so
	// runGenerate stays testable and reusable from Go code.
	fs := flag.NewFlagSet("generate "+name, flag.ContinueOnError)
	var opt barcode.GenerateOptions
	fs.StringVar(&opt.Text, "text", "", "text to encode (required)")
	fs.IntVar(&opt.Module, "module", sym.module, "pixels per module (narrow-bar width for 1D)")
	fs.IntVar(&opt.Quiet, "quiet", sym.quiet, "quiet zone in modules")
	output := fs.String("output", "", `output file path, or "-" for stdout (required)`)
	if sym.oneD {
		fs.IntVar(&opt.Height, "height", 80, "bar height in px")
		fs.BoolVar(&opt.Label, "label", false, "draw human-readable text below the bars")
	} else if name != "pdf417" {
		fs.IntVar(&opt.Size, "size", 0, "exact output edge length in px (overrides module)")
	}
	if sym.fullASCII {
		fs.BoolVar(&opt.FullASCII, "fullascii", false, "extended mode: encode lowercase etc. as +N pairs")
	}
	switch name {
	case "qr":
		fs.StringVar(&opt.Level, "level", "M", "error correction level (L/M/Q/H)")
	case "pdf417":
		fs.StringVar(&opt.Level, "level", "2", "security level (0-8)")
	}
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2 // flag package already printed the error and usage to stderr
	}

	// ponytail: no HTTP-style range checks here — the CLI is the user's own
	// machine and the generator's pixel cap bounds memory. Only reject values
	// that would make image dimensions nonsensical.
	switch {
	case opt.Text == "":
		fmt.Fprintln(os.Stderr, "--text is required")
		return 2
	case *output == "":
		fmt.Fprintln(os.Stderr, "--output is required")
		return 2
	case opt.Module < 1 || opt.Quiet < 0 || opt.Size < 0 || (sym.oneD && opt.Height < 1):
		fmt.Fprintln(os.Stderr, "module and height must be >= 1, quiet and size >= 0")
		return 2
	}
	if name == "pdf417" {
		switch opt.Level {
		case "0", "1", "2", "3", "4", "5", "6", "7", "8":
		default:
			fmt.Fprintln(os.Stderr, "--level must be an integer between 0 and 8")
			return 2
		}
	}

	png, err := sym.gen(opt)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if *output == "-" {
		if _, err := os.Stdout.Write(png); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}
	if err := os.WriteFile(*output, png, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
