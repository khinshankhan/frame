package main

import (
	"image"
	"os"
	"path/filepath"
	"testing"
)

func TestWritePNGLeavesNoFileOnFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.png")
	// png.Encode rejects an empty image, standing in for any failed write.
	if err := writePNG(path, image.NewRGBA(image.Rect(0, 0, 0, 0))); err == nil {
		t.Fatal("writePNG of an empty image succeeded")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("failed write left %s behind (stat: %v)", path, err)
	}
}

func TestWritePNG(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.png")
	if err := writePNG(path, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(path); err != nil || fi.Size() == 0 {
		t.Fatalf("stat %s: %v", path, err)
	}
}
