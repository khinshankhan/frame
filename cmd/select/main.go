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

func main() {
	fs := flag.NewFlagSet("select", flag.ExitOnError)
	points := fs.Bool("points", false, "report display points rather than image pixels")
	fs.Usage = usage
	fs.Parse(os.Args[1:])

	// Default to stdin so `frame capture - | select` works with no args,
	// but print usage rather than blocking on a terminal nobody is piping.
	path := "-"
	if fs.NArg() > 0 {
		path = fs.Arg(0)
	} else if fi, err := os.Stdin.Stat(); err == nil && fi.Mode()&os.ModeCharDevice != 0 {
		usage()
		os.Exit(2)
	}

	img, err := loadRGBA(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "select:", err)
		os.Exit(1)
	}
	r, ok, err := selection.Select(img)
	if err != nil {
		fmt.Fprintln(os.Stderr, "select:", err)
		os.Exit(1)
	}
	if !ok {
		return
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
}

func scale(v int, by float64) int {
	return int(math.Round(float64(v) / by))
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: select [-points] [image.png]

Reads a PNG on stdin when given "-" or no argument, so it pairs with
"frame capture - | select". Prints "x y width height" of the selected
area, in the pixels of that image -- on a Retina display that is twice
the points shown on screen while dragging. Pass -points for those.
Prints nothing if the selection is cancelled.`)
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
	// Name an empty stream for what it is; a capture that failed upstream
	// would otherwise surface here as a puzzling "unknown format".
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
