//go:build unix

package main

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeKeming writes a shell script standing in for keming, in dir.
func fakeKeming(t *testing.T, dir, script string) string {
	t.Helper()
	path := filepath.Join(dir, "keming")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestKemingRead(t *testing.T) {
	dir := t.TempDir()
	sent := filepath.Join(dir, "sent.png")
	k, err := startKeming(fakeKeming(t, dir, `cat > `+sent+`; printf 'args: %s\n' "$*"`), []string{"-l", "deu", "-v"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	src := image.NewRGBA(image.Rect(0, 0, 10, 10))
	got, err := k.read(src.SubImage(image.Rect(2, 3, 5, 5)))
	if err != nil {
		t.Fatal(err)
	}
	if got != "args: -l deu -v" {
		t.Fatalf("got %q", got)
	}
	f, err := os.Open(sent)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	// The selection itself, not enlarged: keming enlarges what it needs to.
	if img.Bounds().Dx() != 3 || img.Bounds().Dy() != 2 {
		t.Fatalf("keming got a %v image, want the 3x2 selection", img.Bounds())
	}
	k.cancel() // after read, does nothing
}

func TestKemingErrors(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	tests := []struct {
		name, script string
		want         string
	}{
		{"keming fails", `cat >/dev/null; echo 'keming: no fonts' >&2; exit 1`, "keming failed"},
		{"nothing read", `cat >/dev/null; printf '\n'`, errNoText.Error()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			k, err := startKeming(fakeKeming(t, t.TempDir(), tt.script), nil, &stderr)
			if err != nil {
				t.Fatal(err)
			}
			_, err = k.read(img)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %v, want it to mention %q", err, tt.want)
			}
			if tt.name == "keming fails" && !strings.Contains(stderr.String(), "no fonts") {
				t.Fatalf("keming's explanation was lost: %q", stderr.String())
			}
		})
	}
	k, err := startKeming(fakeKeming(t, t.TempDir(), `cat >/dev/null; printf '\n'`), nil, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := k.read(img); !errors.Is(err, errNoText) {
		t.Fatalf("got %v, want errNoText", err)
	}
}

func TestKemingCancelSendsNothing(t *testing.T) {
	dir := t.TempDir()
	got := filepath.Join(dir, "bytes")
	k, err := startKeming(fakeKeming(t, dir, `wc -c > `+got), nil, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	k.cancel()
	b, err := os.ReadFile(got)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(b)) != "0" {
		t.Fatalf("keming was sent %s bytes, want 0", strings.TrimSpace(string(b)))
	}
}

func TestKemingArgs(t *testing.T) {
	tests := []struct {
		name, lang, engine string
		verbose            bool
		want               string
	}{
		{"no language leaves keming's default", "", "auto", false, "-engine auto"},
		{"a language is passed on", "fr", "auto", false, "-l fr -engine auto"},
		{"verbose", "", "cells", true, "-engine cells -v"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strings.Join(kemingArgs(tt.lang, tt.engine, tt.verbose), " ")
			if got != tt.want {
				t.Fatalf("kemingArgs = %q, want %q", got, tt.want)
			}
		})
	}
}
