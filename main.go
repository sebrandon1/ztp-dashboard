package main

import (
	"os"

	"github.com/sebrandon1/ztp-dashboard/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
