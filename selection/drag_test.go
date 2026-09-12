package selection

import (
	"image"
	"testing"
)

func TestDragClickIsNotASelection(t *testing.T) {
	var d dragger
	d.press(image.Pt(506, 1032))
	if r, ok := d.release(image.Pt(506, 1032)); ok {
		t.Fatalf("click without a drag selected %v", r)
	}
}

func TestDragReleaseWithoutPress(t *testing.T) {
	var d dragger
	if r, ok := d.release(image.Pt(10, 10)); ok {
		t.Fatalf("release with no press selected %v", r)
	}
}

func TestDragCanonicalises(t *testing.T) {
	var d dragger
	d.press(image.Pt(50, 40))
	r, ok := d.release(image.Pt(10, 20))
	if want := image.Rect(10, 20, 50, 40); !ok || r != want {
		t.Fatalf("got %v, %v; want %v, true", r, ok, want)
	}
}
