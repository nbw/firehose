package main

import (
	"os"

	"github.com/nbw/firehose/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
