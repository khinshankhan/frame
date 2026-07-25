package selection

import "image"

// findSubimage locates an interactively-selected crop inside a full-screen
// grab. screencapture returns the cropped image but never says where on
// screen it came from, so recovering the rectangle means searching for it.
// The X11 path never needs this -- it draws its own overlay, so it knew the
// rectangle the moment you let go of the mouse.
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
