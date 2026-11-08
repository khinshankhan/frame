//go:build unix

package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeTesseract writes a shell script standing in for the tesseract binary.
func fakeTesseract(t *testing.T, script string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tesseract")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRecognizeOutput(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	tests := []struct {
		name    string
		script  string
		want    string
		wantErr string
	}{
		{"strips the page break tesseract ends with", `printf 'hello\nworld\n\f'`, "hello\nworld", ""},
		{"keeps the lines of a block", `printf 'a\n\nb\n'`, "a\n\nb", ""},
		{"passes the flags and language", `printf '%s ' "$@"`, "stdin stdout -l deu --oem 1 --psm 4", ""},
		{"reports what tesseract said", `echo 'no such language' >&2; exit 1`, "", "no such language"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := recognize(fakeTesseract(t, tt.script), "deu", img)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("got error %v, want it to mention %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRecognizeBlankIsNoText(t *testing.T) {
	for _, script := range []string{`printf ''`, `printf ' \n\f'`} {
		t.Run(script, func(t *testing.T) {
			_, err := recognize(fakeTesseract(t, script), "eng", image.NewRGBA(image.Rect(0, 0, 2, 2)))
			if !errors.Is(err, errNoText) {
				t.Fatalf("got %v, want errNoText", err)
			}
		})
	}
}

func TestRecognizeSendsTheImageOnStdin(t *testing.T) {
	saved := filepath.Join(t.TempDir(), "stdin.png")
	img := image.NewRGBA(image.Rect(0, 0, 3, 5))
	if _, err := recognize(fakeTesseract(t, fmt.Sprintf(`cat > %q; printf ok`, saved)), "eng", img); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(saved)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	got, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if got.Bounds().Dx() != 3 || got.Bounds().Dy() != 5 {
		t.Fatalf("got a %v image, want 3x5", got.Bounds())
	}
}

func TestCropAndUpscale(t *testing.T) {
	red := color.RGBA{255, 0, 0, 255}
	blue := color.RGBA{0, 0, 255, 255}
	src := image.NewRGBA(image.Rect(0, 0, 6, 6))
	for y := range 6 {
		for x := range 6 {
			src.SetRGBA(x, y, blue)
		}
	}
	selected := image.Rect(1, 2, 3, 3)
	for y := selected.Min.Y; y < selected.Max.Y; y++ {
		for x := selected.Min.X; x < selected.Max.X; x++ {
			src.SetRGBA(x, y, red)
		}
	}

	got := upscale(src.SubImage(selected), 3)

	if got.Bounds() != image.Rect(0, 0, 6, 3) {
		t.Fatalf("got bounds %v, want a 6x3 image at the origin", got.Bounds())
	}
	// Any blue means pixels from outside the selection leaked in.
	for y := range 3 {
		for x := range 6 {
			if got.RGBAAt(x, y) != red {
				t.Errorf("pixel %d,%d: got %v, want the selection's red", x, y, got.RGBAAt(x, y))
			}
		}
	}
}
