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

func (d *dragger) release(p image.Point) (image.Rectangle, bool) {
	if !d.dragging {
		return image.Rectangle{}, false
	}
	d.dragging = false
	return rectOf(d.start, p), true
}

func rectOf(a, b image.Point) image.Rectangle {
	return image.Rectangle{Min: a, Max: b}.Canon()
}
