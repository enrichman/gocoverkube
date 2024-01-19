package main

import (
	"fmt"
	"os"

	"github.com/enrichman/gocoverkube/internal/cli"
)

func main() {
	rootCmd := cli.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ error: %s\n", err)
		os.Exit(100)
	}
}
