// Package capture provides screen capture behind a backend-neutral interface.
package capture

import "image"

// Capturer is the per-OS capture abstraction. Backends capture a full
// display frame; cropping to a region happens above this layer.
type Capturer interface {
	// Displays reports the bounds of each attached display in the virtual
	// screen space. An empty slice (no displays) is data, not an error.
	Displays() ([]image.Rectangle, error)
	// CaptureDisplay captures the display at index i, as ordered by
	// Displays, erroring if i is out of range.
	CaptureDisplay(i int) (image.Image, error)
}

// New returns the best available Capturer for this machine.
func New() (Capturer, error) {
	return genericCapturer{}, nil
}
