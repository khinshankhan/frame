package main

import (
	"bufio"
	"fmt"
	"image"
	"image/draw"
	_ "image/png"
	"io"
	"os"

	"github.com/khinshankhan/frame/selection"
)

func main() {
	// Default to stdin so `frame capture - | select` needs no argument, but
	// bail with usage rather than blocking when stdin is a terminal.
	path := "-"
	if len(os.Args) > 1 {
		path = os.Args[1]
	} else if fi, err := os.Stdin.Stat(); err == nil && fi.Mode()&os.ModeCharDevice != 0 {
		fmt.Fprintln(os.Stderr, `usage: select [image.png]   reads a PNG on stdin when given "-" or no argument`)
		os.Exit(2)
	}

	r, ok, err := selectRect(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "select:", err)
		os.Exit(1)
	}
	if ok {
		fmt.Printf("%d %d %d %d\n", r.Min.X, r.Min.Y, r.Dx(), r.Dy())
	}
}

func selectRect(path string) (image.Rectangle, bool, error) {
	img, err := loadRGBA(path)
	if err != nil {
		return image.Rectangle{}, false, err
	}
	return selection.Select(img)
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
