//go:build windows

package fileutil

// Windows does not provide a portable directory fsync path through the Go stdlib.
// AtomicWriteFile still renames atomically there, but directory metadata durability
// cannot be strengthened the same way we do on Unix.
func syncDir(string) error {
	return nil
}
