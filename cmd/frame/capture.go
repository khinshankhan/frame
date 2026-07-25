package main

import (
	"flag"
	"fmt"
	"image/png"
	"io"
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
	var w io.Writer = os.Stdout
	summary := os.Stderr
	if out != "-" {
		f, err := os.Create(out)
		if err != nil {
			return err
		}
		defer f.Close()
		w, summary = f, os.Stdout
	}
	if err := png.Encode(w, img); err != nil {
		return err
	}
	fmt.Fprintf(summary, "%s (%dx%d)\n", out, img.Bounds().Dx(), img.Bounds().Dy())
	return nil
}
