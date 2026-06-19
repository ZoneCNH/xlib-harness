package main

import (
	"os"
	"testing"
)

func TestMainDispatchesToHarness(t *testing.T) {
	oldArgs := os.Args
	oldExit := exit
	t.Cleanup(func() {
		os.Args = oldArgs
		exit = oldExit
	})

	got := -1
	exit = func(code int) {
		got = code
	}
	os.Args = []string{"xlib-harness", "help"}

	main()

	if got != 0 {
		t.Fatalf("exit code = %d, want 0", got)
	}
}
