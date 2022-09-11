package main

import (
	"os"
)

func main() {
	if err := run(os.Args); err != nil {
		os.Exit(1)
	}
}
