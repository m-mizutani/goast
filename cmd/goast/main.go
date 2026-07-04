package main

import (
	"os"
)

// version is injected at build time via -ldflags "-X main.version=...".
// It defaults to "dev" for local builds where no ldflags are supplied.
var version = "dev"

func main() {
	if err := run(os.Args); err != nil {
		os.Exit(1)
	}
}
