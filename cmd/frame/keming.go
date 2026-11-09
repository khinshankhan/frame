package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"os/exec"
	"strings"
)

// kemingBinary is the optional reader extract prefers to tesseract. It picks
// the best reader per selection, reads code exactly against the fonts
// installed here, and runs offline. It is a separate project,
// github.com/khinshankhan/keming, so frame only runs it.
const kemingBinary = "keming"

// kemingRun is keming started before the area is picked: it loads fonts and
// models while it waits for the image on stdin, which takes seconds.
type kemingRun struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
	out   bytes.Buffer
	done  bool
}

// kemingArgs are keming's arguments for extract's flags. The language is
// passed only when given, so keming keeps its own default.
func kemingArgs(lang, engine string, verbose bool) []string {
	var a []string
	if lang != "" {
		a = append(a, "-l", lang)
	}
	a = append(a, "-engine", engine)
	if verbose {
		a = append(a, "-v")
	}
	return a
}

// startKeming runs keming with args. Its stderr, where -v says which reader
// was used and errors are explained, goes to stderr.
func startKeming(binary string, args []string, stderr io.Writer) (*kemingRun, error) {
	k := &kemingRun{cmd: exec.Command(binary, args...)}
	k.cmd.Stdout = &k.out
	k.cmd.Stderr = stderr
	stdin, err := k.cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	k.stdin = stdin
	if err := k.cmd.Start(); err != nil {
		return nil, fmt.Errorf("keming: %w", err)
	}
	return k, nil
}

// read sends img and returns the text keming read. The image goes over stdin
// so a screenshot never lands in a temp file. keming enlarges it itself.
func (k *kemingRun) read(img image.Image) (string, error) {
	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	err := encoder.Encode(k.stdin, img)
	if cerr := k.stdin.Close(); err == nil {
		err = cerr
	}
	k.done = true
	if werr := k.cmd.Wait(); werr != nil {
		return "", fmt.Errorf("keming failed: %w", werr)
	}
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(k.out.String())
	if text == "" {
		return "", errNoText
	}
	return text, nil
}

// cancel ends a keming that won't be given an image: empty stdin tells it
// to exit quietly. It does nothing after read.
func (k *kemingRun) cancel() {
	if k.done {
		return
	}
	k.done = true
	k.stdin.Close()
	k.cmd.Wait()
}
