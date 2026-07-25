//go:build darwin

// Package macperm reports on macOS Screen Recording access, which both
// capture and interactive selection depend on.
package macperm

import (
	"os/exec"
	"strings"
)

// paneURL deep-links System Settings to the Screen Recording pane.
const paneURL = "x-apple.systempreferences:com.apple.preference.security?Privacy_ScreenCapture"

// Hint tells the user how to grant access. TCC fixes a process's access at
// first use, so a fresh grant only takes hold once the terminal -- and any
// tmux or screen server under it -- restarts.
const Hint = "grant Screen Recording access to your terminal in " +
	"System Settings > Privacy & Security > Screen Recording, then restart it and retry"

// preflightScript asks CoreGraphics outright whether Screen Recording is
// granted. CGPreflightScreenCaptureAccess is missing from JXA's
// bridgesupport metadata, so it is bound by hand -- which keeps the check
// cgo-free and needs no new dependency.
const preflightScript = `
ObjC.import("CoreGraphics");
ObjC.bindFunction("CGPreflightScreenCaptureAccess", ["bool", []]);
String($.CGPreflightScreenCaptureAccess());
`

// Denied reports whether Screen Recording access is refused. Callers gate
// on this before showing any UI: screencapture draws its selection overlay
// without access and only fails once it reads the framebuffer.
//
// It asks CoreGraphics rather than inferring from a trial capture. A probe
// grab conflates "no access" with "that grab failed": `screencapture -R`
// has been seen refusing every rect on a machine whose full-screen grabs
// succeeded in the same second, which a probe reads as a denial and reports
// to a user who has already granted access.
func Denied() bool {
	out, err := exec.Command("osascript", "-l", "JavaScript", "-e", preflightScript).Output()
	if err != nil {
		// Nothing usable came back. Let the capture itself be the judge
		// rather than blocking on an inconclusive probe.
		return false
	}
	return strings.TrimSpace(string(out)) != "true"
}

// OpenSettings deep-links System Settings to the Screen Recording pane.
// Best effort -- callers report Hint whether or not the pane opens.
func OpenSettings() {
	_ = exec.Command("open", paneURL).Run()
}
