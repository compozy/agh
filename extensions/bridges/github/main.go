package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "serve" {
		return runServe(stdin, stdout, stderr)
	}
	return fmt.Errorf("github: unsupported command %q", strings.TrimSpace(args[0]))
}

func runServe(stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	provider, err := newGitHubProvider(stderr)
	if err != nil {
		return err
	}
	return provider.serve(stdin, stdout)
}
