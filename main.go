package main

import (
	"os"

	"github.com/ZoneCNH/xlib-harness/internal/harness"
)

func main() {
	os.Exit(harness.Run(os.Args[1:], os.Stdout, os.Stderr))
}
