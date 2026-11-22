package main

import (
	"os"

	"github.com/krewire/krewire/pkg/ai/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
