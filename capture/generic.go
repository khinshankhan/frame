//go:build !darwin

package capture

import (
	"fmt"
	"image"

	"github.com/kbinani/screenshot"
)

// genericCapturer wraps kbinani/screenshot: XGetImage on X11 (with a
// portal/dbus path on Wayland, unverified by us), GDI on Windows. Pure Go
// on those platforms; darwin gets its own cgo-free backend in darwin.go.
type genericCapturer struct{}

func newCapturer() (Capturer, error) {
	return genericCapturer{}, nil
}

func (genericCapturer) Displays() ([]image.Rectangle, error) {
	bounds := make([]image.Rectangle, screenshot.NumActiveDisplays())
	for i := range bounds {
		bounds[i] = screenshot.GetDisplayBounds(i)
	}
	return bounds, nil
}

func (genericCapturer) CaptureDisplay(i int) (image.Image, error) {
	// Validate here: the library panics rather than erroring on a bad index.
	if n := screenshot.NumActiveDisplays(); i < 0 || i >= n {
		return nil, fmt.Errorf("display %d out of range (have %d)", i, n)
	}
	img, err := screenshot.CaptureDisplay(i)
	if err != nil {
		return nil, fmt.Errorf("capture display %d: %w", i, err)
	}
	return img, nil
}
