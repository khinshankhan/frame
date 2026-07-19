package main

import (
	"flag"
	"fmt"
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

	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return err
	}
	fmt.Printf("%s (%dx%d)\n", out, img.Bounds().Dx(), img.Bounds().Dy())
	return nil
}
