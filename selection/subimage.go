package selection

import (
	"bytes"
	"image"
)

// findSubimage locates an interactively-selected crop inside a full-screen
// grab. screencapture returns the cropped image but never says where on
// screen it came from, so recovering the rectangle means searching for it.
// The X11 path never needs this -- it draws its own overlay, so it knew the
// rectangle the moment you let go of the mouse.
func findSubimage(img, needle *image.RGBA) (image.Rectangle, bool) {
	bounds := img.Bounds()
	nb := needle.Bounds()
	if nb.Dx() > bounds.Dx() || nb.Dy() > bounds.Dy() {
		return image.Rectangle{}, false
	}

	// Pack to alpha-stripped, tightly-strided RGB buffers so each
	// candidate row compares with one vectorized bytes.Equal instead of a
	// per-pixel loop -- large uniform-color regions (e.g. a solid editor
	// background) otherwise blow up the brute-force scan below.
	imgRGB, imgStride := packRGB(img)
	needleRGB, needleStride := packRGB(needle)
	rowBytes := nb.Dx() * 3
	rows := nb.Dy()

	rect := func(x, y int) image.Rectangle {
		p := image.Pt(bounds.Min.X+x, bounds.Min.Y+y)
		return image.Rectangle{Min: p, Max: p.Add(nb.Size())}
	}

	// Exact first. It is both the cheapest pass and the only unambiguous
	// one, so an unchanged selection never reaches the tolerant search
	// below and can never be placed by it.
	for y := 0; y <= bounds.Dy()-rows; y++ {
		for x := 0; x <= bounds.Dx()-nb.Dx(); x++ {
			if rowsEqual(imgRGB, imgStride, x*3, y, needleRGB, needleStride, rowBytes, rows) {
				return rect(x, y), true
			}
		}
	}

	// Nothing matched exactly, so the screen moved or composited
	// differently between the two grabs. Tolerate that, but score every
	// candidate and keep the closest rather than the first that passes:
	// with any slack at all, several positions clear the bar, and picking
	// whichever came first in scan order silently returns wrong
	// coordinates.
	//
	// Candidates come from a single row matching exactly, which costs one
	// vectorized compare per position and rejects almost everything.
	// Several anchors are tried because any one row may be the volatile
	// one.
	budget := rowBytes * rows / 50
	bestSum := -1
	var best image.Rectangle
	for _, anchor := range anchorRows(needleRGB, needleStride, rowBytes, rows) {
		anchorRow := needleRGB[anchor*needleStride : anchor*needleStride+rowBytes]
		for y := 0; y <= bounds.Dy()-rows; y++ {
			base := (y + anchor) * imgStride
			for x := 0; x <= bounds.Dx()-nb.Dx(); x++ {
				off := base + x*3
				if !bytes.Equal(imgRGB[off:off+rowBytes], anchorRow) {
					continue
				}
				sum, ok := diffScore(imgRGB, imgStride, x*3, y, needleRGB, needleStride, rowBytes, rows, budget, bestSum)
				if ok && (bestSum < 0 || sum < bestSum) {
					bestSum, best = sum, rect(x, y)
				}
			}
		}
	}
	if bestSum >= 0 {
		return best, true
	}
	return image.Rectangle{}, false
}

// diffScore totals the absolute channel difference between needle and the
// candidate position, reporting false if more than budget bytes differ by
// more than channelSlack, or if the total passes ceiling (a non-negative
// ceiling is the best score so far, letting hopeless candidates abort
// early). The raw total is what separates the true position from a merely
// similar one: both may sit inside the slack, but only the true position
// approaches zero.
func diffScore(imgRGB []byte, imgStride, xOff, y int, needleRGB []byte, needleStride, rowBytes, rows, budget, ceiling int) (int, bool) {
	sum := 0
	for row := 0; row < rows; row++ {
		imgStart := (y+row)*imgStride + xOff
		needleStart := row * needleStride
		a := imgRGB[imgStart : imgStart+rowBytes]
		b := needleRGB[needleStart : needleStart+rowBytes]
		if bytes.Equal(a, b) {
			continue
		}
		for i := range a {
			d := int(a[i]) - int(b[i])
			if d < 0 {
				d = -d
			}
			if d == 0 {
				continue
			}
			if d > channelSlack {
				if budget--; budget < 0 {
					return 0, false
				}
			}
			if sum += d; ceiling >= 0 && sum >= ceiling {
				return 0, false
			}
		}
	}
	return sum, true
}

// anchorRows picks one row per third of the selection, spread out so
// volatile content in one band cannot defeat them all. Within a band it
// takes the row nearest the middle carrying some detail: a flat row matches
// at thousands of positions and drags every one through the full
// comparison, while the very busiest row tends to be the one that changed
// between grabs -- text with a blinking cursor -- so aiming for either
// extreme is worse than settling for merely distinctive.
func anchorRows(needleRGB []byte, stride, rowBytes, rows int) []int {
	// Roughly "not a flat expanse": enough colour changes along the row to
	// be rare in the haystack, low enough that most textured rows qualify.
	// Counted per pixel, not per byte -- a solid run of any non-grey colour
	// is R,G,B,R,G,B..., so byte-to-byte it looks maximally varied while
	// being the flattest row there is, and anchoring on one matches at
	// every similarly-filled position on screen.
	minDetail := (rowBytes / 3) / 20

	detailed := func(r int) bool {
		row := needleRGB[r*stride : r*stride+rowBytes]
		n := 0
		for i := 3; i < len(row); i += 3 {
			if row[i] != row[i-3] || row[i+1] != row[i-2] || row[i+2] != row[i-1] {
				if n++; n >= minDetail {
					return true
				}
			}
		}
		return false
	}

	seen := map[int]bool{}
	var out []int
	for band := 0; band < 3; band++ {
		lo, hi := band*rows/3, (band+1)*rows/3
		if band == 2 {
			hi = rows
		}
		if lo >= hi {
			continue
		}
		mid := (lo + hi) / 2
		pick := mid
		// Walk outwards from the middle for the first row with detail.
		for d := 0; d < hi-lo; d++ {
			if r := mid + d; r < hi && detailed(r) {
				pick = r
				break
			}
			if r := mid - d; r >= lo && detailed(r) {
				pick = r
				break
			}
		}
		if !seen[pick] {
			seen[pick] = true
			out = append(out, pick)
		}
	}
	return out
}

// channelSlack is the per-channel difference treated as noise rather than
// changed content. Translucent surfaces -- the menu bar above all --
// composite slightly differently in a region grab than in a full-screen
// one: measured across a menu-bar selection, 3% of samples differed but
// never by more than 5. Genuinely changed pixels differ by far more (a
// repainting clock measures 197-214), so this separates them cleanly.
const channelSlack = 8

// rowsEqual reports whether needle sits byte for byte at the given offset.
func rowsEqual(imgRGB []byte, imgStride, xOff, y int, needleRGB []byte, needleStride, rowBytes, rows int) bool {
	for row := 0; row < rows; row++ {
		imgStart := (y+row)*imgStride + xOff
		needleStart := row * needleStride
		if !bytes.Equal(imgRGB[imgStart:imgStart+rowBytes], needleRGB[needleStart:needleStart+rowBytes]) {
			return false
		}
	}
	return true
}

func packRGB(img *image.RGBA) ([]byte, int) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	stride := w * 3
	out := make([]byte, stride*h)
	for y := 0; y < h; y++ {
		srcRow := img.Pix[y*img.Stride:]
		dstRow := out[y*stride:]
		for x := 0; x < w; x++ {
			si, di := x*4, x*3
			dstRow[di], dstRow[di+1], dstRow[di+2] = srcRow[si], srcRow[si+1], srcRow[si+2]
		}
	}
	return out, stride
}
