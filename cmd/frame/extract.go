package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os/exec"
	"strings"

	"github.com/khinshankhan/frame/selection"
	"github.com/khinshankhan/yui/lib/clipboard"
	"golang.org/x/image/draw"
)

// Experiment: OCR is meant to be a first-party plugin, so once the plugin
// host exists this moves there. Until then it is the way to try OCR on real
// screenshots, and a record of what the plugin needs from the host: the
// selected area as an image, a scale option, exec and a clipboard.

const (
	tesseractBinary = "tesseract"

	// Screen text is small for tesseract. 2x read browser and terminal text
	// far better than the original size, and 3x was no better.
	ocrUpscale = 2
)

var errNoText = errors.New("no text found")

func runExtract(args []string) error {
	fs := flag.NewFlagSet("extract", flag.ExitOnError)
	lang := fs.String("l", "eng", "tesseract language, see tesseract --list-langs")
	fs.Parse(args)

	// Checked first: finding out after picking an area wastes the pick.
	binary, err := exec.LookPath(tesseractBinary)
	if err != nil {
		return fmt.Errorf("extract needs tesseract installed: %w", err)
	}

	img, err := inputImage(fs)
	if err != nil {
		return err
	}
	r, ok, err := selection.Select(img)
	if err != nil {
		return err
	}
	if !ok {
		return nil // cancelled; declining to select is not an error
	}

	text, err := recognize(binary, *lang, upscale(img.SubImage(r), ocrUpscale))
	if err != nil {
		return err
	}
	// Printed first, so a clipboard failure doesn't lose the text.
	fmt.Println(text)
	return clipboard.Copy(text)
}

// recognize reads the text in img. The image goes over stdin so a screenshot,
// which may well hold something private, never lands in a temp file.
func recognize(binary, lang string, img image.Image) (string, error) {
	var in bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := encoder.Encode(&in, img); err != nil {
		return "", err
	}

	// psm 4 reads one column of text in mixed sizes. psm 3 read ls -l output
	// column by column, and psm 6 lost words from forms and video calls.
	var out, stderr bytes.Buffer
	cmd := exec.Command(binary, "stdin", "stdout", "-l", lang, "--oem", "1", "--psm", "4")
	cmd.Stdin = &in
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tesseract: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	// Trimming also drops the form feed tesseract ends each page with.
	text := strings.TrimSpace(out.String())
	if text == "" {
		return "", errNoText
	}
	return text, nil
}

// upscale returns img enlarged by a whole factor. A smooth filter matters:
// nearest neighbour read digits worse than no upscale at all, 0 turning to 8.
func upscale(img image.Image, by int) *image.RGBA {
	b := img.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx()*by, b.Dy()*by))
	draw.CatmullRom.Scale(out, out.Bounds(), img, b, draw.Src, nil)
	return out
}
