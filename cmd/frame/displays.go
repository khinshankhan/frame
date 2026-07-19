package main

import (
	"fmt"

	"github.com/khinshankhan/frame/capture"
)

func runDisplays() error {
	c, err := capture.New()
	if err != nil {
		return err
	}
	displays, err := c.Displays()
	if err != nil {
		return err
	}
	for i, b := range displays {
		fmt.Printf("%d: %dx%d at (%d,%d)\n", i, b.Dx(), b.Dy(), b.Min.X, b.Min.Y)
	}
	return nil
}
