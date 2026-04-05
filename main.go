package main

import (
	"os"

	"github.com/dimuthu/kube-gpu/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
