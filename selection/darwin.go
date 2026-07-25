//go:build darwin

package selection

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"os/exec"

	"github.com/khinshankhan/frame/macperm"
)

// screenScaleScript reports each screen's point size and backing scale.
const screenScaleScript = `
ObjC.import("AppKit");
var out = [];
var screens = $.NSScreen.screens;
for (var i = 0; i < screens.count; i++) {
  var s = screens.objectAtIndex(i);
  out.push({W:s.frame.size.width, H:s.frame.size.height, Scale:s.backingScaleFactor});
}
JSON.stringify(out);
`

// Scale reports how many pixels of img make up one display point, for
// converting Select's rectangles into the point space macOS shows on
// screen. The factor is per-display -- 1 on a non-Retina monitor, 2 on
// Retina, and potentially both at once on a mixed setup -- so it is matched
// against whichever screen img was captured from rather than assumed from
// the main one. Falls back to 1, which is right whenever no scaling is in
// play and merely leaves coordinates in pixels when it is not.
func Scale(img *image.RGBA) float64 {
	out, err := exec.Command("osascript", "-l", "JavaScript", "-e", screenScaleScript).Output()
	if err != nil {
		return 1
	}
	var screens []struct{ W, H, Scale float64 }
	if err := json.Unmarshal(out, &screens); err != nil {
		return 1
	}

	w, h := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
	for _, s := range screens {
		if s.Scale > 0 && math.Round(s.W*s.Scale) == w && math.Round(s.H*s.Scale) == h {
			return s.Scale
		}
	}
	// img is not a whole screen (a pre-cropped frame, say), so there is
	// nothing to match it against. The main display is the best guess left,
	// but say so: on a mixed-DPI setup it is the wrong display's factor, and
	// silently reporting points that are off by 2x is worse than a warning.
	if len(screens) > 0 && screens[0].Scale > 0 {
		fmt.Fprintf(os.Stderr,
			"select: %dx%d matches no display; assuming the main display's %gx scale\n",
			img.Bounds().Dx(), img.Bounds().Dy(), screens[0].Scale)
		return screens[0].Scale
	}
	return 1
}

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
		// A re-grab of a different size means the display was resized or
		// rearranged since the caller captured img, so img describes a
		// screen that no longer exists. Searching it could only produce a
		// rectangle in stale coordinates.
		if screen.Bounds().Size() != img.Bounds().Size() {
			return image.Rectangle{}, false, fmt.Errorf(
				"display changed size since the capture (%v then, %v now) -- capture again",
				img.Bounds().Size(), screen.Bounds().Size())
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
