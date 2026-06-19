package main

import (
	"os"

	"github.com/ZoneCNH/xlib-harness/internal/harness"
)

var exit = os.Exit

func main() {
	exit(harness.Run(os.Args[1:], os.Stdout, os.Stderr))
}
