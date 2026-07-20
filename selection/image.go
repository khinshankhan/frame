package selection

import (
	"image"
	"image/draw"
)

func rgbaOf(src image.Image) *image.RGBA {
	if img, ok := src.(*image.RGBA); ok && img.Rect.Min == (image.Point{}) {
		return img
	}
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst
}
