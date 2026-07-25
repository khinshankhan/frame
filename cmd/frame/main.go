package main

import (
	"fmt"
	"os"

	"github.com/khinshankhan/frame/version"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "capture":
		err = runCapture(os.Args[2:])
	case "displays":
		err = runDisplays()
	case "version":
		fmt.Println(version.String())
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "frame:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage:
  frame capture [-d display] [out.png]   capture a display to a PNG ("-" for stdout)
  frame displays                         list displays
  frame version                          show build info`)
}
