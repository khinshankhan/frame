//go:build darwin

package capture

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/khinshankhan/frame/macperm"
)

// darwinCapturer shells out to the system screencapture CLI -- frame is
// cgo-free on every platform, and kbinani/screenshot's darwin backend is
// cgo-only.
type darwinCapturer struct{}

func newCapturer() (Capturer, error) {
	return darwinCapturer{}, nil
}

// screenInfo mirrors one NSScreen: frame in points, origin bottom-left,
// Y-up, plus backingScaleFactor to convert to pixels.
type screenInfo struct {
	X, Y, W, H float64
	Scale      float64
}

// nsScreensScript reaches NSScreen via JXA (osascript -l JavaScript),
// which keeps the display query cgo-free.
const nsScreensScript = `
ObjC.import("AppKit");
var screens = $.NSScreen.screens;
var out = [];
for (var i = 0; i < screens.count; i++) {
  var s = screens.objectAtIndex(i);
  var f = s.frame;
  out.push({X:f.origin.x, Y:f.origin.y, W:f.size.width, H:f.size.height, Scale:s.backingScaleFactor});
}
JSON.stringify(out);
`

func nsScreens() ([]screenInfo, error) {
	out, err := exec.Command("osascript", "-l", "JavaScript", "-e", nsScreensScript).Output()
	if err != nil {
		return nil, fmt.Errorf("query displays: %w", err)
	}
	var screens []screenInfo
	if err := json.Unmarshal(out, &screens); err != nil {
		return nil, fmt.Errorf("parse display list: %w", err)
	}
	return screens, nil
}

func (darwinCapturer) Displays() ([]image.Rectangle, error) {
	screens, err := nsScreens()
	if err != nil {
		return nil, err
	}
	return screenBounds(screens), nil
}

// screenBounds converts AppKit's per-display, bottom-left-origin, Y-up
// frames into pixel rectangles in a single top-left-origin, Y-down space,
// matching the images CaptureDisplay returns.
//
// Only ever exercised against a single display. Each screen is scaled by
// its own backingScaleFactor, so a mixed-DPI arrangement composes into a
// pixel space that matches no real coordinate system -- revisit before
// trusting these bounds on multiple monitors.
func screenBounds(screens []screenInfo) []image.Rectangle {
	top := 0.0
	for _, s := range screens {
		if v := s.Y + s.H; v > top {
			top = v
		}
	}
	bounds := make([]image.Rectangle, len(screens))
	for i, s := range screens {
		minX := int(math.Round(s.X * s.Scale))
		minY := int(math.Round((top - (s.Y + s.H)) * s.Scale))
		w := int(math.Round(s.W * s.Scale))
		h := int(math.Round(s.H * s.Scale))
		bounds[i] = image.Rect(minX, minY, minX+w, minY+h)
	}
	return bounds
}

// CaptureDisplay shells out to `screencapture -D`, which numbers displays
// from 1 -- assumed to line up with the order Displays reports them in.
// The index goes unchecked here: screencapture rejects a bad one itself,
// and validating up front would cost an osascript round trip per capture.
func (darwinCapturer) CaptureDisplay(i int) (image.Image, error) {
	f, err := os.CreateTemp("", "frame-capture-*.png")
	if err != nil {
		return nil, err
	}
	path := f.Name()
	f.Close()
	os.Remove(path)
	defer os.Remove(path)

	cmd := exec.Command("screencapture", "-x", "-D", strconv.Itoa(i+1), path)
	if out, runErr := cmd.CombinedOutput(); runErr != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = runErr.Error()
		}
		if strings.Contains(msg, "could not create image from display") {
			macperm.OpenSettings()
			return nil, fmt.Errorf("capture display %d: %s (%s)", i, msg, macperm.Hint)
		}
		// screencapture counts displays from 1, so restate its complaint in
		// the 0-based numbering that -d uses.
		if strings.Contains(msg, "Invalid display specified") {
			if screens, nsErr := nsScreens(); nsErr == nil {
				return nil, fmt.Errorf("display %d out of range (have %d)", i, len(screens))
			}
		}
		return nil, fmt.Errorf("capture display %d: %s", i, msg)
	}

	imgFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer imgFile.Close()
	img, err := png.Decode(imgFile)
	if err != nil {
		return nil, fmt.Errorf("decode captured display %d: %w", i, err)
	}
	return img, nil
}
