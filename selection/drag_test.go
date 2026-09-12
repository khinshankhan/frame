package selection

import (
	"image"
	"testing"
)

func TestDragClickIsNotASelection(t *testing.T) {
	d := dragger{bounds: image.Rect(0, 0, 1920, 1080)}
	d.press(image.Pt(506, 1032))
	if r, ok := d.release(image.Pt(506, 1032)); ok {
		t.Fatalf("click without a drag selected %v", r)
	}
}

func TestDragReleaseWithoutPress(t *testing.T) {
	d := dragger{bounds: image.Rect(0, 0, 1920, 1080)}
	if r, ok := d.release(image.Pt(10, 10)); ok {
		t.Fatalf("release with no press selected %v", r)
	}
}

func TestDragCanonicalises(t *testing.T) {
	d := dragger{bounds: image.Rect(0, 0, 1920, 1080)}
	d.press(image.Pt(50, 40))
	r, ok := d.release(image.Pt(10, 20))
	if want := image.Rect(10, 20, 50, 40); !ok || r != want {
		t.Fatalf("got %v, %v; want %v, true", r, ok, want)
	}
}

func TestDragClipsToImage(t *testing.T) {
	d := dragger{bounds: image.Rect(0, 0, 100, 100)}
	d.press(image.Pt(10, 10))
	if r, _ := d.move(image.Pt(300, 200)); r != image.Rect(10, 10, 100, 100) {
		t.Fatalf("move drew %v past the image", r)
	}
	r, ok := d.release(image.Pt(300, 200))
	if want := image.Rect(10, 10, 100, 100); !ok || r != want {
		t.Fatalf("got %v, %v; want %v, true", r, ok, want)
	}
}

func TestDragOutsideImageIsNotASelection(t *testing.T) {
	d := dragger{bounds: image.Rect(0, 0, 100, 100)}
	d.press(image.Pt(150, 150))
	if r, ok := d.release(image.Pt(300, 200)); ok {
		t.Fatalf("drag outside the image selected %v", r)
	}
}
