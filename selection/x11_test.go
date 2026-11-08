//go:build linux || freebsd || openbsd || netbsd

package selection

import (
	"image"
	"image/color"
	"reflect"
	"testing"

	"github.com/jezek/xgb/xproto"
)

func TestDimmed(t *testing.T) {
	tests := []struct {
		name string
		in   color.RGBA
		want color.RGBA
	}{
		{"black stays black", color.RGBA{0, 0, 0, 255}, color.RGBA{0, 0, 0, 255}},
		{"white drops to 70 percent", color.RGBA{255, 255, 255, 255}, color.RGBA{178, 178, 178, 255}},
		{"channels scale apart", color.RGBA{100, 50, 10, 255}, color.RGBA{70, 35, 7, 255}},
		{"alpha is untouched", color.RGBA{100, 100, 100, 40}, color.RGBA{70, 70, 70, 40}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := image.NewRGBA(image.Rect(0, 0, 1, 1))
			img.SetRGBA(0, 0, tt.in)
			if got := dimmed(img).RGBAAt(0, 0); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			if got := img.RGBAAt(0, 0); got != tt.in {
				t.Fatalf("changed the source image to %v", got)
			}
		})
	}
}

func TestGuides(t *testing.T) {
	bounds := image.Rect(0, 0, 100, 50)
	tests := []struct {
		name string
		p    image.Point
		want []xproto.Segment
	}{
		{
			"span the image through the pointer",
			image.Pt(30, 20),
			[]xproto.Segment{{X1: 0, Y1: 20, X2: 99, Y2: 20}, {X1: 30, Y1: 0, X2: 30, Y2: 49}},
		},
		{
			"last pixel of the image still has guides",
			image.Pt(99, 49),
			[]xproto.Segment{{X1: 0, Y1: 49, X2: 99, Y2: 49}, {X1: 99, Y1: 0, X2: 99, Y2: 49}},
		},
		{"none past the right edge", image.Pt(100, 20), nil},
		{"none past the bottom edge", image.Pt(30, 50), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := guides(tt.p, bounds); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// X draws a rectangle one pixel wider and taller than asked, so the outline
// must ask for one less to land on the selection's last row and column.
func TestOutlineStaysInsideSelection(t *testing.T) {
	tests := []struct {
		name string
		sel  image.Rectangle
		want xproto.Rectangle
	}{
		{"regular", image.Rect(10, 20, 110, 70), xproto.Rectangle{X: 10, Y: 20, Width: 99, Height: 49}},
		{"one pixel", image.Rect(5, 5, 6, 6), xproto.Rectangle{X: 5, Y: 5, Width: 0, Height: 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := outline(tt.sel); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
