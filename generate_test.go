package main

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestRunGenerate(t *testing.T) {
	out := filepath.Join(t.TempDir(), "dm.png")
	if code := runGenerate([]string{"datamatrix", "--text", "ABC123", "--size", "256", "--output", out}); code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(b, []byte{0x89, 0x50, 0x4E, 0x47}) {
		t.Fatal("output is not a PNG")
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width != 256 || cfg.Height != 256 {
		t.Errorf("image = %dx%d, want 256x256", cfg.Width, cfg.Height)
	}
}

func TestRunGenerateErrors(t *testing.T) {
	out := filepath.Join(t.TempDir(), "x.png")
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"no args", nil, 2},
		{"unknown symbology", []string{"nope", "--text", "A", "--output", out}, 2},
		{"missing text", []string{"qr", "--output", out}, 2},
		{"missing output", []string{"qr", "--text", "A"}, 2},
		{"encode failure", []string{"ean13", "--text", "hello", "--output", out}, 1},
		// must return, not os.Exit — ContinueOnError keeps runGenerate reusable
		{"unknown flag", []string{"qr", "--nope", "--text", "A", "--output", out}, 2},
		{"help", []string{"qr", "--help"}, 0},
	}
	for _, c := range cases {
		if code := runGenerate(c.args); code != c.want {
			t.Errorf("%s: exit code %d, want %d", c.name, code, c.want)
		}
	}
}
