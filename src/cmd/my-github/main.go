package main

import (
	"fmt"
	"io"
	"os"

	"github.com/end-2/my-cli/src/cmd/my-github/internal/app"
)

var Version = "dev"

func main() {
	if err := execute(os.Stdin, os.Stdout, os.Stderr, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func execute(stdin io.Reader, stdout, stderr io.Writer, args []string) error {
	return app.Execute(stdin, stdout, stderr, args, Version)
}
