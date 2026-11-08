//go:build linux || freebsd || openbsd || netbsd

package selection

import (
	"fmt"
	"image"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

const (
	escapeKeysym xproto.Keysym = 0xff1b

	// Together these wait up to a second for the keyboard to come free.
	keyboardGrabTries = 100
	keyboardGrabWait  = 10 * time.Millisecond
)

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

	gc, err := xproto.NewGcontextId(conn)
	if err != nil {
		return image.Rectangle{}, false, err
	}
	xproto.CreateGC(conn, gc, xproto.Drawable(wid),
		xproto.GcForeground|xproto.GcGraphicsExposures, []uint32{screen.BlackPixel, 0})

	newPixmap := func() (xproto.Drawable, error) {
		pix, err := xproto.NewPixmapId(conn)
		if err != nil {
			return 0, err
		}
		xproto.CreatePixmap(conn, screen.RootDepth, pix, xproto.Drawable(wid), uint16(width), uint16(height))
		return xproto.Drawable(pix), nil
	}
	bright, err := newPixmap()
	if err != nil {
		return image.Rectangle{}, false, err
	}
	dim, err := newPixmap()
	if err != nil {
		return image.Rectangle{}, false, err
	}
	// Each frame is composed here and copied to the window whole, so the
	// screen never shows a frame half drawn.
	back, err := newPixmap()
	if err != nil {
		return image.Rectangle{}, false, err
	}

	// An image smaller than the overlay leaves the rest of dim black, not
	// whatever the new pixmap's memory happened to hold.
	xproto.PolyFillRectangle(conn, dim, gc, []xproto.Rectangle{{Width: uint16(width), Height: uint16(height)}})
	maxReq := int(setup.MaximumRequestLength)
	putImage(conn, bright, gc, img, width, height, maxReq)
	putImage(conn, dim, gc, dimmed(img), width, height, maxReq)
	xproto.ChangeGC(conn, gc, xproto.GcForeground, []uint32{screen.WhitePixel})

	// Mapped only now, so the upload above never shows as a black screen.
	xproto.MapWindow(conn, wid)
	xproto.GrabPointer(conn, false, wid,
		uint16(xproto.EventMaskButtonPress|xproto.EventMaskButtonRelease|xproto.EventMaskPointerMotion),
		xproto.GrabModeAsync, xproto.GrabModeAsync, wid, xproto.CursorNone, xproto.TimeCurrentTime)
	if err := grabKeyboard(conn, wid); err != nil {
		return image.Rectangle{}, false, err
	}

	pointer, err := xproto.QueryPointer(conn, wid).Reply()
	if err != nil {
		return image.Rectangle{}, false, err
	}
	cursor := pointOf(pointer.WinX, pointer.WinY)
	var sel image.Rectangle

	paint := func() {
		xproto.CopyArea(conn, dim, back, gc, 0, 0, 0, 0, uint16(width), uint16(height))
		if sel.Empty() {
			if lines := guides(cursor, img.Rect); lines != nil {
				xproto.PolySegment(conn, back, gc, lines)
			}
		} else {
			x, y := int16(sel.Min.X), int16(sel.Min.Y)
			xproto.CopyArea(conn, bright, back, gc, x, y, x, y, uint16(sel.Dx()), uint16(sel.Dy()))
			xproto.PolyRectangle(conn, back, gc, []xproto.Rectangle{outline(sel)})
		}
		xproto.CopyArea(conn, back, xproto.Drawable(wid), gc, 0, 0, 0, 0, uint16(width), uint16(height))
		// Wait for the server to finish the frame, so motion that arrives
		// meanwhile queues up here for events.next to skip. A failure here
		// means the connection died, which the event loop reports.
		xproto.GetInputFocus(conn).Reply()
	}

	drag := dragger{bounds: img.Rect}
	queue := events{wait: conn.WaitForEvent, poll: conn.PollForEvent}
	for {
		ev, xerr := queue.next()
		if xerr != nil {
			return image.Rectangle{}, false, xerr
		}
		if ev == nil {
			return image.Rectangle{}, false, fmt.Errorf("connection closed")
		}
		switch e := ev.(type) {
		case xproto.ExposeEvent:
			paint()
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
			cursor = pointOf(e.EventX, e.EventY)
			sel, _ = drag.move(cursor)
			paint()
		case xproto.ButtonReleaseEvent:
			// A release with no drag behind it -- the tail of the click that
			// launched us, or a click that never moved -- is not a choice.
			// Keep waiting; Escape is how to back out.
			if e.Detail != xproto.ButtonIndex1 {
				continue
			}
			cursor = pointOf(e.EventX, e.EventY)
			if r, ok := drag.release(cursor); ok {
				return r, true, nil
			}
			sel = image.Rectangle{}
			paint()
		}
	}
}

// grabKeyboard takes the keyboard so Escape reaches the overlay: a window
// manager never focuses an override-redirect window. Launched from a hotkey,
// the keyboard stays held until that key comes up, so it gets a moment.
func grabKeyboard(conn *xgb.Conn, wid xproto.Window) error {
	var status byte
	for range keyboardGrabTries {
		reply, err := xproto.GrabKeyboard(conn, false, wid, xproto.TimeCurrentTime,
			xproto.GrabModeAsync, xproto.GrabModeAsync).Reply()
		if err != nil {
			return err
		}
		if reply.Status == xproto.GrabStatusSuccess {
			return nil
		}
		status = reply.Status
		time.Sleep(keyboardGrabWait)
	}
	return fmt.Errorf("could not grab the keyboard, is another program holding it? (X status %d)", status)
}

// events reads X events, passing over any motion that a newer motion has
// already replaced. Every motion repaints the screen, so without this a
// large screen falls further behind the faster the mouse moves.
type events struct {
	wait, poll func() (xgb.Event, xgb.Error)
	held       xgb.Event
}

func (q *events) next() (xgb.Event, xgb.Error) {
	ev := q.held
	q.held = nil
	if ev == nil {
		var err xgb.Error
		ev, err = q.wait()
		if err != nil {
			return nil, err
		}
	}
	if _, ok := ev.(xproto.MotionNotifyEvent); !ok {
		return ev, nil
	}
	for {
		newer, err := q.poll()
		if err != nil {
			return nil, err
		}
		if newer == nil {
			return ev, nil
		}
		if _, ok := newer.(xproto.MotionNotifyEvent); !ok {
			q.held = newer
			return ev, nil
		}
		ev = newer
	}
}

// dimmed returns a darkened copy of img, drawn everywhere outside the
// selection. Core X has no alpha, so darkening happens here, once, rather
// than per frame on the server.
func dimmed(img *image.RGBA) *image.RGBA {
	out := image.NewRGBA(img.Rect)
	for i := 0; i+3 < len(img.Pix); i += 4 {
		out.Pix[i] = dimChannel(img.Pix[i])
		out.Pix[i+1] = dimChannel(img.Pix[i+1])
		out.Pix[i+2] = dimChannel(img.Pix[i+2])
		out.Pix[i+3] = img.Pix[i+3]
	}
	return out
}

func dimChannel(v uint8) uint8 {
	return uint8(uint32(v) * 7 / 10)
}

// guides returns the crosshair through p, spanning the image. A pointer off
// the image has no guides, since nothing can be selected there.
func guides(p image.Point, bounds image.Rectangle) []xproto.Segment {
	if !p.In(bounds) {
		return nil
	}
	return []xproto.Segment{
		{X1: int16(bounds.Min.X), Y1: int16(p.Y), X2: int16(bounds.Max.X - 1), Y2: int16(p.Y)},
		{X1: int16(p.X), Y1: int16(bounds.Min.Y), X2: int16(p.X), Y2: int16(bounds.Max.Y - 1)},
	}
}

// outline returns the rectangle to stroke around sel. X draws one pixel
// wider and taller than asked, hence the minus one.
func outline(sel image.Rectangle) xproto.Rectangle {
	return xproto.Rectangle{
		X: int16(sel.Min.X), Y: int16(sel.Min.Y),
		Width: uint16(sel.Dx() - 1), Height: uint16(sel.Dy() - 1),
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
