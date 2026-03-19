package main

import (
	"os"

	"github.com/cyperx84/threadgroups/internal/cli"
)

func main() {
	if err := cli.RootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
