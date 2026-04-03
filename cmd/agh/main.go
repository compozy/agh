// Package main is the entry point for the AGH daemon process.
package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/pedronauck/agh/internal/daemon"
	"github.com/pedronauck/agh/internal/version"
)

type daemonRunner interface {
	Run(ctx context.Context) error
}

var newDaemon = func() (daemonRunner, error) {
	return daemon.New()
}

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	command := "start"
	if len(args) > 0 {
		command = args[0]
	}

	switch command {
	case "start":
		runner, err := newDaemon()
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		if err := runner.Run(ctx); err != nil {
			_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		return 0
	case "version", "--version", "-v":
		_, _ = fmt.Fprintf(stdout, "agh %s\n", version.Version)
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "usage: agh [start|version]\n")
		return 2
	}
}
