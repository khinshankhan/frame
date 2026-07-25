//go:build darwin

package selection

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"

	"github.com/khinshankhan/frame/macperm"
)

func Select(img *image.RGBA) (image.Rectangle, bool, error) {
	// Check access before the crosshair appears. macOS draws the selection
	// overlay without Screen Recording access and only enforces it when
	// screencapture reads the framebuffer, so leaving this until afterwards
	// makes the user drag out a region before learning it can't be taken.
	if macperm.Denied() {
		macperm.OpenSettings()
		return image.Rectangle{}, false, fmt.Errorf("screen recording access denied (%s)", macperm.Hint)
	}

	// Access is already confirmed, so an empty result is the user escaping
	// out. Declining to select is not an error.
	crop, ok, err := screencapture("-i", "-s", "-x")
	if err != nil {
		return image.Rectangle{}, false, err
	}
	if !ok {
		return image.Rectangle{}, false, nil
	}

	// screencapture -i grabs the live framebuffer at the moment the drag
	// ends, so match against a frame taken right after rather than img,
	// which the caller may have taken minutes ago. Anything that moved
	// since -- terminal output, the menu bar clock, a blinking cursor --
	// defeats an exact match. Re-grabbing narrows that window to the gap
	// between releasing the mouse and the capture below.
	if screen, ok, err := screencapture("-x", "-D", "1"); err == nil && ok {
		if r, found := findSubimage(screen, crop); found {
			return r, true, nil
		}
	}

	// img still answers when the screen genuinely held still, and is all
	// that is left if the re-grab failed or the selection came from a
	// display other than the main one.
	if r, found := findSubimage(img, crop); found {
		return r, true, nil
	}
	return image.Rectangle{}, false, fmt.Errorf("selected area not found on screen -- it likely changed between the capture and the selection")
}

// screencapture runs the tool with args plus a temp destination and decodes
// what it wrote. Reports false without an error when nothing was written,
// which is how an interactive grab signals cancellation.
func screencapture(args ...string) (*image.RGBA, bool, error) {
	f, err := os.CreateTemp("", "frame-selection-*.png")
	if err != nil {
		return nil, false, err
	}
	path := f.Name()
	if err := f.Close(); err != nil {
		return nil, false, err
	}
	// Remove it so a leftover empty file is never mistaken for a capture.
	if err := os.Remove(path); err != nil {
		return nil, false, err
	}
	defer os.Remove(path)

	cmd := exec.Command("screencapture", append(append([]string{}, args...), path)...)
	if err := cmd.Run(); err != nil && !fileExists(path) {
		return nil, false, nil
	}

	img, err := loadPNG(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if img.Rect.Empty() {
		return nil, false, nil
	}
	return img, true, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func loadPNG(path string) (*image.RGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	src, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	return rgbaOf(src), nil
}
