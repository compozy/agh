// Package fileutil provides shared filesystem helpers for AGH components.
package fileutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrInvalidPath reports a path that cannot be represented safely by the filesystem boundary.
var ErrInvalidPath = errors.New("fileutil: invalid path")

// AtomicWriteFile writes content to path via temp-file-and-rename.
// It always syncs the temp file before rename for durability.
func AtomicWriteFile(path string, content []byte, perm os.FileMode) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("%w: path is required", ErrInvalidPath)
	}
	if strings.ContainsRune(path, 0) {
		return fmt.Errorf("%w: path contains NUL byte", ErrInvalidPath)
	}

	dir := filepath.Dir(path)
	tempFile, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("fileutil: create temp file for %q: %w", path, err)
	}

	tempPath := tempFile.Name()
	cleanup := true
	defer func() {
		if cleanup {
			// Best-effort cleanup only; a failed remove does not affect atomic replacement semantics.
			_ = os.Remove(tempPath)
		}
	}()

	if err := writeTempFile(tempFile, tempPath, content, perm); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("fileutil: replace %q: %w", path, err)
	}
	if err := syncDir(dir); err != nil {
		return fmt.Errorf("fileutil: sync parent directory for %q: %w", path, err)
	}

	cleanup = false
	return nil
}

// AtomicRemoveFile removes a file and syncs its parent directory.
func AtomicRemoveFile(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("%w: path is required", ErrInvalidPath)
	}
	if strings.ContainsRune(path, 0) {
		return fmt.Errorf("%w: path contains NUL byte", ErrInvalidPath)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("fileutil: stat %q before remove: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("fileutil: remove %q: target is a directory", path)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("fileutil: remove %q: %w", path, err)
	}
	if err := syncDir(filepath.Dir(path)); err != nil {
		return fmt.Errorf("fileutil: sync parent directory for %q: %w", path, err)
	}
	return nil
}

func writeTempFile(file *os.File, tempPath string, content []byte, perm os.FileMode) error {
	var err error
	if _, err = file.Write(content); err == nil {
		err = file.Chmod(perm)
	}
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("fileutil: prepare temp file %q: %w", tempPath, err)
	}
	return nil
}
