package selection

import "image"

type dragger struct {
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
	return rectOf(d.start, p), true
}

// release ends the drag. A click that never moved encloses nothing, so it
// reports false like a release with no press before it: neither is a
// selection, and the caller should keep waiting for one.
func (d *dragger) release(p image.Point) (image.Rectangle, bool) {
	if !d.dragging {
		return image.Rectangle{}, false
	}
	d.dragging = false
	r := rectOf(d.start, p)
	return r, !r.Empty()
}

func rectOf(a, b image.Point) image.Rectangle {
	return image.Rectangle{Min: a, Max: b}.Canon()
}
