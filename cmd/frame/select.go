package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/draw"
	_ "image/png"
	"io"
	"math"
	"os"

	"github.com/khinshankhan/frame/selection"
)

func runSelect(args []string) error {
	fs := flag.NewFlagSet("select", flag.ExitOnError)
	points := fs.Bool("points", false, "report display points rather than image pixels")
	fs.Parse(args)

	// Default to stdin so `frame capture - | frame select` needs no
	// argument, but say so rather than blocking when stdin is a terminal.
	path := "-"
	if fs.NArg() > 0 {
		path = fs.Arg(0)
	} else if fi, err := os.Stdin.Stat(); err == nil && fi.Mode()&os.ModeCharDevice != 0 {
		return fmt.Errorf(`select needs a PNG: pass a file, or pipe one in with "frame capture - | frame select"`)
	}

	img, err := loadRGBA(path)
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

	x, y, w, h := r.Min.X, r.Min.Y, r.Dx(), r.Dy()
	if *points {
		// Selection happens in img's pixel space; the conversion factor
		// belongs to whichever display img came from, so it is read from
		// the image rather than assumed.
		if s := selection.Scale(img); s > 0 {
			x, y, w, h = scale(x, s), scale(y, s), scale(w, s), scale(h, s)
		}
	}
	fmt.Printf("%d %d %d %d\n", x, y, w, h)
	return nil
}

func scale(v int, by float64) int {
	return int(math.Round(float64(v) / by))
}

func loadRGBA(path string) (*image.RGBA, error) {
	var in io.Reader = os.Stdin
	if path != "-" {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		in = f
	}
	// Tell an empty stream apart from a malformed one: a failed upstream
	// capture would otherwise read as "unknown format".
	br := bufio.NewReader(in)
	if _, err := br.Peek(1); err == io.EOF {
		return nil, fmt.Errorf("no image data on input (did the capture upstream fail?)")
	}
	src, _, err := image.Decode(br)
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst, nil
}
