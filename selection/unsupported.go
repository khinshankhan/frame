//go:build !linux && !freebsd && !openbsd && !netbsd && !darwin

package selection

import (
	"fmt"
	"image"
)

func Select(*image.RGBA) (image.Rectangle, bool, error) {
	return image.Rectangle{}, false, fmt.Errorf("interactive area selection is not supported on this OS yet")
}

// Scale reports pixels of img per display point.
func Scale(*image.RGBA) float64 { return 1 }
