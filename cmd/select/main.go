package main

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/png"
	"os"

	"github.com/khinshankhan/frame/selection"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: select <image.png>")
		os.Exit(2)
	}
	r, ok, err := selectRect(os.Args[1])
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
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	src, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst, nil
}
