//go:build darwin

package selection

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
)

func Select(img *image.RGBA) (image.Rectangle, bool, error) {
	f, err := os.CreateTemp("", "frame-selection-*.png")
	if err != nil {
		return image.Rectangle{}, false, err
	}
	path := f.Name()
	if err := f.Close(); err != nil {
		return image.Rectangle{}, false, err
	}
	if err := os.Remove(path); err != nil {
		return image.Rectangle{}, false, err
	}
	defer os.Remove(path)

	cmd := exec.Command("screencapture", "-i", "-s", "-x", path)
	if err := cmd.Run(); err != nil && !fileExists(path) {
		return image.Rectangle{}, false, nil
	}

	crop, err := loadPNG(path)
	if err != nil {
		if os.IsNotExist(err) {
			return image.Rectangle{}, false, nil
		}
		return image.Rectangle{}, false, err
	}
	if crop.Rect.Empty() {
		return image.Rectangle{}, false, nil
	}

	r, ok := findSubimage(img, crop)
	if !ok {
		return image.Rectangle{}, false, fmt.Errorf("selected area not found in source image")
	}
	return r, true, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func loadPNG(path string) (*image.RGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	src, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	return rgbaOf(src), nil
}

func findSubimage(img, needle *image.RGBA) (image.Rectangle, bool) {
	bounds := img.Bounds()
	nb := needle.Bounds()
	if nb.Dx() > bounds.Dx() || nb.Dy() > bounds.Dy() {
		return image.Rectangle{}, false
	}

	for y := bounds.Min.Y; y <= bounds.Max.Y-nb.Dy(); y++ {
		for x := bounds.Min.X; x <= bounds.Max.X-nb.Dx(); x++ {
			p := image.Pt(x, y)
			if subimageAt(img, needle, p) {
				return image.Rectangle{Min: p, Max: p.Add(nb.Size())}, true
			}
		}
	}
	return image.Rectangle{}, false
}

func subimageAt(img, needle *image.RGBA, at image.Point) bool {
	nb := needle.Bounds()
	for y := 0; y < nb.Dy(); y++ {
		imgRow := img.Pix[(at.Y+y-img.Rect.Min.Y)*img.Stride:]
		needleRow := needle.Pix[(nb.Min.Y+y-needle.Rect.Min.Y)*needle.Stride:]
		for x := 0; x < nb.Dx(); x++ {
			i := (at.X + x - img.Rect.Min.X) * 4
			j := (nb.Min.X + x - needle.Rect.Min.X) * 4
			if imgRow[i] != needleRow[j] || imgRow[i+1] != needleRow[j+1] || imgRow[i+2] != needleRow[j+2] {
				return false
			}
		}
	}
	return true
}
