package selection

import "image"

type dragger struct {
	// bounds is the image being selected on. The overlay can be larger
	// than the image (a small file, or one display of several), so drags
	// are clipped to it rather than reporting pixels the image lacks.
	bounds   image.Rectangle
	dragging bool
	start    image.Point
}

func (d *dragger) press(p image.Point) {
	d.dragging = true
	d.start = p
}

func (d *dragger) move(p image.Point) (image.Rectangle, bool) {
	if !d.dragging {
		return image.Rectangle{}, false
	}
	return rectOf(d.start, p).Intersect(d.bounds), true
}

// release ends the drag. A click that never moved, or a drag wholly off
// the image, encloses nothing, so it reports false just like a release
// with no press before it: none is a selection, and the caller should
// keep waiting for one.
func (d *dragger) release(p image.Point) (image.Rectangle, bool) {
	if !d.dragging {
		return image.Rectangle{}, false
	}
	d.dragging = false
	r := rectOf(d.start, p).Intersect(d.bounds)
	return r, !r.Empty()
}

func rectOf(a, b image.Point) image.Rectangle {
	return image.Rectangle{Min: a, Max: b}.Canon()
}
