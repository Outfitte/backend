// Package main is the entry point for the Outfitte server.
package main

import (
	"fmt"
	"os"

	"github.com/outfitte/outfitte/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %s\n", err)
		os.Exit(1)
	}
	_ = cfg
}
