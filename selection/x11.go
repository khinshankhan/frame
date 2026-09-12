//go:build linux || freebsd || openbsd || netbsd

package selection

import (
	"fmt"
	"image"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

const escapeKeysym xproto.Keysym = 0xff1b

// Scale reports pixels of img per display point. X11 addresses the screen
// in pixels already, so its rectangles need no conversion.
func Scale(*image.RGBA) float64 { return 1 }

func Select(img *image.RGBA) (image.Rectangle, bool, error) {
	conn, err := xgb.NewConn()
	if err != nil {
		return image.Rectangle{}, false, err
	}
	defer conn.Close()

	setup := xproto.Setup(conn)
	screen := setup.DefaultScreen(conn)
	width, height := int(screen.WidthInPixels), int(screen.HeightInPixels)
	escapeKeycode, err := keycodeForKeysym(conn, setup, escapeKeysym)
	if err != nil {
		return image.Rectangle{}, false, err
	}

	wid, err := xproto.NewWindowId(conn)
	if err != nil {
		return image.Rectangle{}, false, err
	}
	mask := uint32(xproto.CwBackPixel | xproto.CwOverrideRedirect | xproto.CwEventMask)
	values := []uint32{
		screen.BlackPixel,
		1,
		uint32(xproto.EventMaskButtonPress | xproto.EventMaskButtonRelease |
			xproto.EventMaskPointerMotion | xproto.EventMaskKeyPress | xproto.EventMaskExposure),
	}
	xproto.CreateWindow(conn, screen.RootDepth, wid, screen.Root,
		0, 0, uint16(width), uint16(height), 0,
		xproto.WindowClassInputOutput, screen.RootVisual, mask, values)
	xproto.MapWindow(conn, wid)

	gc, err := xproto.NewGcontextId(conn)
	if err != nil {
		return image.Rectangle{}, false, err
	}
	xproto.CreateGC(conn, gc, xproto.Drawable(wid), 0, nil)

	pix, err := xproto.NewPixmapId(conn)
	if err != nil {
		return image.Rectangle{}, false, err
	}
	xproto.CreatePixmap(conn, screen.RootDepth, pix, xproto.Drawable(wid), uint16(width), uint16(height))
	putImage(conn, xproto.Drawable(pix), gc, img, width, height, int(setup.MaximumRequestLength))

	xproto.GrabPointer(conn, false, wid,
		uint16(xproto.EventMaskButtonPress|xproto.EventMaskButtonRelease|xproto.EventMaskPointerMotion),
		xproto.GrabModeAsync, xproto.GrabModeAsync, wid, xproto.CursorNone, xproto.TimeCurrentTime)

	paint := func(sel *image.Rectangle) {
		xproto.CopyArea(conn, xproto.Drawable(pix), xproto.Drawable(wid), gc,
			0, 0, 0, 0, uint16(width), uint16(height))
		if sel != nil {
			xproto.PolyRectangle(conn, xproto.Drawable(wid), gc, []xproto.Rectangle{{
				X: int16(sel.Min.X), Y: int16(sel.Min.Y),
				Width: uint16(sel.Dx()), Height: uint16(sel.Dy()),
			}})
		}
	}

	drag := dragger{bounds: img.Rect}
	for {
		ev, xerr := conn.WaitForEvent()
		if xerr != nil {
			return image.Rectangle{}, false, xerr
		}
		if ev == nil {
			return image.Rectangle{}, false, fmt.Errorf("connection closed")
		}
		switch e := ev.(type) {
		case xproto.ExposeEvent:
			paint(nil)
		case xproto.KeyPressEvent:
			if e.Detail == escapeKeycode {
				return image.Rectangle{}, false, nil
			}
		case xproto.ButtonPressEvent:
			// Only the primary button drags; the wheel reports as buttons
			// 4 and 5 and would otherwise end the selection on a scroll.
			if e.Detail == xproto.ButtonIndex1 {
				drag.press(pointOf(e.EventX, e.EventY))
			}
		case xproto.MotionNotifyEvent:
			if r, ok := drag.move(pointOf(e.EventX, e.EventY)); ok {
				paint(&r)
			}
		case xproto.ButtonReleaseEvent:
			// A release with no drag behind it -- the tail of the click that
			// launched us, or a click that never moved -- is not a choice.
			// Keep waiting; Escape is how to back out.
			if e.Detail != xproto.ButtonIndex1 {
				continue
			}
			if r, ok := drag.release(pointOf(e.EventX, e.EventY)); ok {
				return r, true, nil
			}
			paint(nil)
		}
	}
}

func keycodeForKeysym(conn *xgb.Conn, setup *xproto.SetupInfo, keysym xproto.Keysym) (xproto.Keycode, error) {
	count := byte(setup.MaxKeycode - setup.MinKeycode + 1)
	reply, err := xproto.GetKeyboardMapping(conn, setup.MinKeycode, count).Reply()
	if err != nil {
		return 0, err
	}
	if reply == nil {
		return 0, fmt.Errorf("keyboard mapping unavailable")
	}

	perKeycode := int(reply.KeysymsPerKeycode)
	for i := 0; i < int(count); i++ {
		for j := 0; j < perKeycode; j++ {
			if reply.Keysyms[i*perKeycode+j] == keysym {
				return setup.MinKeycode + xproto.Keycode(i), nil
			}
		}
	}
	return 0, fmt.Errorf("keysym %#x not found", keysym)
}

func putImage(conn *xgb.Conn, drawable xproto.Drawable, gc xproto.Gcontext, img *image.RGBA, w, h, maxReq int) {
	iw := min(img.Rect.Dx(), w)
	ih := min(img.Rect.Dy(), h)
	maxRows := (maxReq*4 - 64) / (iw * 4)
	if maxRows < 1 {
		maxRows = 1
	}
	for y0 := 0; y0 < ih; y0 += maxRows {
		rows := maxRows
		if y0+rows > ih {
			rows = ih - y0
		}
		buf := make([]byte, iw*rows*4)
		for y := 0; y < rows; y++ {
			srcRow := img.Pix[(y0+y)*img.Stride:]
			for x := 0; x < iw; x++ {
				src := srcRow[x*4:]
				dst := buf[(y*iw+x)*4:]
				dst[0], dst[1], dst[2], dst[3] = src[2], src[1], src[0], 255
			}
		}
		xproto.PutImage(conn, xproto.ImageFormatZPixmap, drawable, gc,
			uint16(iw), uint16(rows), 0, int16(y0), 0, 24, buf)
	}
}

func pointOf(x, y int16) image.Point {
	return image.Pt(int(x), int(y))
}
