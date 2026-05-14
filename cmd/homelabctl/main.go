package main

import (
	"fmt"
	"os"

	"github.com/example/go-homelabctl/internal/app"
)

func main() {
	if err := app.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
