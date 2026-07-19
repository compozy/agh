package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func main() {
	if len(os.Args) < 4 || os.Args[2] != "--" {
		fmt.Fprintln(os.Stderr, "usage: go run ./scripts/air-state-lock <lock-file> -- <command> [args...]")
		os.Exit(2)
	}

	lockPath := filepath.Clean(os.Args[1])
	if lockPath != os.Args[1] || filepath.Base(lockPath) != "dev-owner.lock" {
		fatalf("Air state lock path must be normalized and end with dev-owner.lock")
	}
	// #nosec G703 -- the normalized operator-owned path is constrained to the fixed lock basename.
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		fatalf("open Air state lock: %v", err)
	}
	if err := unix.Flock(int(lockFile.Fd()), unix.LOCK_EX); err != nil {
		closeAndFatal(lockFile, "acquire Air state lock: %v", err)
	}
	if _, err := unix.FcntlInt(lockFile.Fd(), unix.F_SETFD, 0); err != nil {
		closeAndFatal(lockFile, "preserve Air state lock across exec: %v", err)
	}

	command := os.Args[3:]
	commandPath, err := exec.LookPath(command[0])
	if err != nil {
		closeAndFatal(lockFile, "resolve locked command %q: %v", command[0], err)
	}
	if err := unix.Exec(commandPath, command, os.Environ()); err != nil {
		closeAndFatal(lockFile, "execute locked command %q: %v", command[0], err)
	}
}

func closeAndFatal(lockFile *os.File, format string, args ...any) {
	if err := lockFile.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "dev: close Air state lock: %v\n", err)
	}
	fatalf(format, args...)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "dev: "+format+"\n", args...)
	os.Exit(1)
}
