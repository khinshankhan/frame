//go:build !linux && !freebsd && !openbsd && !netbsd

package selection

import (
	"fmt"
	"image"
)

func Select(*image.RGBA) (image.Rectangle, bool, error) {
	return image.Rectangle{}, false, fmt.Errorf("interactive area selection is not supported on this OS yet")
}
