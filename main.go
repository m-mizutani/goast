package main

import (
	"os"

	"github.com/m-mizutani/goast/pkg/cmd"
)

func main() {
	if err := cmd.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
