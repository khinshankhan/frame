package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"time"

	"github.com/khinshankhan/frame/capture"
)

func runCapture(args []string) error {
	fs := flag.NewFlagSet("capture", flag.ExitOnError)
	displayIndex := fs.Int("d", 0, "display index to capture")
	fs.Parse(args)

	out := fmt.Sprintf("frame-%s.png", time.Now().Format("2006-01-02-150405"))
	if fs.NArg() > 0 {
		out = fs.Arg(0)
	}

	c, err := capture.New()
	if err != nil {
		return err
	}
	img, err := c.CaptureDisplay(*displayIndex)
	if err != nil {
		return err
	}

	// "-" streams the PNG on stdout so it can pipe into select; the summary
	// moves to stderr there to keep that stream decodable.
	summary := os.Stdout
	if out == "-" {
		if err := png.Encode(os.Stdout, img); err != nil {
			return err
		}
		summary = os.Stderr
	} else if err := writePNG(out, img); err != nil {
		return err
	}
	fmt.Fprintf(summary, "%s (%dx%d)\n", out, img.Bounds().Dx(), img.Bounds().Dy())
	return nil
}

// writePNG writes img to path, removing the file again if it cannot be
// written in full so a failed capture never leaves a truncated PNG behind.
func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	err = png.Encode(f, img)
	// Close can report the write failure (a full disk, NFS) that Encode
	// did not, so it decides success as much as Encode does.
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(path)
	}
	return err
}
