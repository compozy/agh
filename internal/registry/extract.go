package registry

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	// DefaultMaxDecompressedSize caps the total extracted archive size.
	DefaultMaxDecompressedSize int64 = 500 * 1024 * 1024
	// DefaultMaxFileCount caps the number of archive entries processed.
	DefaultMaxFileCount = 10000
)

var (
	errArchiveTooLarge     = errors.New("registry: archive exceeds max decompressed size")
	errArchiveTooManyFiles = errors.New("registry: archive exceeds max file count")
)

type extractLimits struct {
	maxDecompressedSize int64
	maxFileCount        int
}

func (l extractLimits) normalized() extractLimits {
	if l.maxDecompressedSize <= 0 {
		l.maxDecompressedSize = DefaultMaxDecompressedSize
	}
	if l.maxFileCount <= 0 {
		l.maxFileCount = DefaultMaxFileCount
	}
	return l
}

// ExtractArchive extracts a tar.gz archive into destRoot using the default
// decompressed-size and file-count limits.
func ExtractArchive(reader io.Reader, destRoot string) error {
	return extractArchive(reader, destRoot, extractLimits{})
}

func extractArchive(reader io.Reader, destRoot string, limits extractLimits) (err error) {
	if strings.TrimSpace(destRoot) == "" {
		return errors.New("destination root is required")
	}
	if err := os.MkdirAll(destRoot, 0o755); err != nil {
		return fmt.Errorf("create destination root %q: %w", destRoot, err)
	}

	limits = limits.normalized()

	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("open gzip stream: %w", err)
	}
	defer func() {
		if closeErr := gzipReader.Close(); closeErr != nil {
			if err == nil {
				err = fmt.Errorf("close gzip stream: %w", closeErr)
			} else {
				err = errors.Join(err, fmt.Errorf("close gzip stream: %w", closeErr))
			}
		}
	}()

	tarReader := tar.NewReader(gzipReader)
	entryCount := 0
	var totalExtracted int64

	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read tar entry: %w", err)
		}

		entryCount++
		if entryCount > limits.maxFileCount {
			return fmt.Errorf("%w: limit=%d", errArchiveTooManyFiles, limits.maxFileCount)
		}

		entryName, err := CleanArchiveEntryPath(header.Name)
		if err != nil {
			return err
		}
		targetPath, err := PathWithinRoot(destRoot, filepath.FromSlash(entryName))
		if err != nil {
			return fmt.Errorf("resolve archive entry %q: %w", header.Name, err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := ensureExtractionPathSafe(destRoot, targetPath); err != nil {
				return fmt.Errorf("validate archive directory %q: %w", targetPath, err)
			}

			dirMode := archiveDirMode(header)
			if err := os.MkdirAll(targetPath, dirMode); err != nil {
				return fmt.Errorf("create archive directory %q: %w", targetPath, err)
			}
			if err := os.Chmod(targetPath, dirMode); err != nil {
				return fmt.Errorf("set archive directory mode %q: %w", targetPath, err)
			}
		case tar.TypeReg, 0:
			parentDir := filepath.Dir(targetPath)
			if err := ensureExtractionPathSafe(destRoot, parentDir); err != nil {
				return fmt.Errorf("validate archive parent %q: %w", parentDir, err)
			}
			if err := os.MkdirAll(parentDir, 0o755); err != nil {
				return fmt.Errorf("create archive parent %q: %w", parentDir, err)
			}

			if err := ensureExtractionPathSafe(destRoot, targetPath); err != nil {
				return fmt.Errorf("validate archive file %q: %w", targetPath, err)
			}

			fileMode := archiveFileMode(header)
			file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fileMode)
			if err != nil {
				return fmt.Errorf("create archive file %q: %w", targetPath, err)
			}

			counter := &countingLimitWriter{
				total: &totalExtracted,
				limit: limits.maxDecompressedSize,
			}
			teeReader := io.TeeReader(tarReader, counter)
			if _, err := io.Copy(file, teeReader); err != nil {
				writeErr := fmt.Errorf("write archive file %q: %w", targetPath, err)
				if closeErr := file.Close(); closeErr != nil {
					writeErr = errors.Join(writeErr, fmt.Errorf("close archive file %q after write failure: %w", targetPath, closeErr))
				}
				_ = os.Remove(targetPath)
				return writeErr
			}
			if err := file.Close(); err != nil {
				return fmt.Errorf("close archive file %q: %w", targetPath, err)
			}
			if err := os.Chmod(targetPath, fileMode); err != nil {
				return fmt.Errorf("set archive file mode %q: %w", targetPath, err)
			}
		default:
			return fmt.Errorf("unsupported archive entry type %d for %q", header.Typeflag, header.Name)
		}
	}
}

type countingLimitWriter struct {
	total *int64
	limit int64
}

func (w *countingLimitWriter) Write(p []byte) (int, error) {
	if w.total == nil {
		return 0, errors.New("total counter is required")
	}
	if w.limit <= 0 {
		*w.total += int64(len(p))
		return len(p), nil
	}

	remaining := w.limit - *w.total
	if remaining <= 0 {
		return 0, errArchiveTooLarge
	}
	if int64(len(p)) > remaining {
		*w.total = w.limit
		return int(remaining), errArchiveTooLarge
	}

	*w.total += int64(len(p))
	return len(p), nil
}

// MoveInstalledDir moves an extracted package directory into its final location.
// If replaceExisting is true, the current target is atomically backed up and
// restored on failure.
func MoveInstalledDir(extractedDir string, targetDir string, replaceExisting bool) error {
	if !replaceExisting {
		if _, err := os.Stat(targetDir); err == nil {
			return fmt.Errorf("package %q already exists at %s", filepath.Base(targetDir), targetDir)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("registry: inspect target directory %q: %w", targetDir, err)
		}

		if err := os.Rename(extractedDir, targetDir); err != nil {
			return fmt.Errorf("registry: install package into %q: %w", targetDir, err)
		}
		return nil
	}

	if _, err := os.Stat(targetDir); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("registry: inspect target directory %q: %w", targetDir, err)
		}
		if err := os.Rename(extractedDir, targetDir); err != nil {
			return fmt.Errorf("registry: install updated package into %q: %w", targetDir, err)
		}
		return nil
	}

	backupDir := fmt.Sprintf("%s.backup-%d", targetDir, time.Now().UTC().UnixNano())
	if err := os.Rename(targetDir, backupDir); err != nil {
		return fmt.Errorf("registry: stage existing package backup %q: %w", targetDir, err)
	}

	if err := os.Rename(extractedDir, targetDir); err != nil {
		revertErr := os.Rename(backupDir, targetDir)
		if revertErr != nil {
			return errors.Join(
				fmt.Errorf("registry: install updated package into %q: %w", targetDir, err),
				fmt.Errorf("registry: restore original package from %q: %w", backupDir, revertErr),
			)
		}
		return fmt.Errorf("registry: install updated package into %q: %w", targetDir, err)
	}

	if err := os.RemoveAll(backupDir); err != nil {
		return fmt.Errorf("registry: remove backup package directory %q: %w", backupDir, err)
	}
	return nil
}

// CleanArchiveEntryPath normalizes one archive entry and rejects absolute or
// escaping paths.
func CleanArchiveEntryPath(entry string) (string, error) {
	cleaned := path.Clean(strings.TrimSpace(strings.ReplaceAll(entry, "\\", "/")))
	switch {
	case cleaned == ".", cleaned == "":
		return "", errors.New("archive entry path is required")
	case strings.HasPrefix(cleaned, "/"):
		return "", fmt.Errorf("archive entry %q must be relative", entry)
	case cleaned == "..", strings.HasPrefix(cleaned, "../"):
		return "", fmt.Errorf("archive entry %q escapes the extraction root", entry)
	default:
		return cleaned, nil
	}
}

// PathWithinRoot resolves a child path and guarantees it stays under root.
func PathWithinRoot(root string, child string) (string, error) {
	absRoot, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return "", fmt.Errorf("resolve root %q: %w", root, err)
	}
	targetPath := filepath.Join(absRoot, child)
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return "", fmt.Errorf("resolve target %q: %w", targetPath, err)
	}
	relative, err := filepath.Rel(absRoot, absTarget)
	if err != nil {
		return "", fmt.Errorf("resolve target %q within %q: %w", absTarget, absRoot, err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("path must stay within the root directory")
	}
	return absTarget, nil
}

func archiveDirMode(header *tar.Header) os.FileMode {
	if header == nil {
		return 0o755
	}

	mode := os.FileMode(header.Mode) & os.ModePerm
	if mode == 0 {
		return 0o755
	}
	return mode
}

func archiveFileMode(header *tar.Header) os.FileMode {
	if header == nil {
		return 0o644
	}

	mode := os.FileMode(header.Mode) & os.ModePerm
	if mode == 0 {
		return 0o644
	}
	return mode
}

func ensureExtractionPathSafe(root string, targetPath string) error {
	absRoot, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return fmt.Errorf("resolve extraction root %q: %w", root, err)
	}
	absTarget, err := filepath.Abs(strings.TrimSpace(targetPath))
	if err != nil {
		return fmt.Errorf("resolve extraction target %q: %w", targetPath, err)
	}

	relative, err := filepath.Rel(absRoot, absTarget)
	if err != nil {
		return fmt.Errorf("resolve extraction target %q within %q: %w", absTarget, absRoot, err)
	}

	if err := ensureExtractionComponent(absRoot, true); err != nil {
		return err
	}
	if relative == "." {
		return nil
	}

	current := absRoot
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "" || component == "." {
			continue
		}

		current = filepath.Join(current, component)
		if err := ensureExtractionComponent(current, current != absTarget); err != nil {
			return err
		}
	}

	return nil
}

func ensureExtractionComponent(path string, mustBeDir bool) error {
	info, err := os.Lstat(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil
	case err != nil:
		return fmt.Errorf("inspect extraction path %q: %w", path, err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("extraction path %q traverses symlink", path)
	}
	if mustBeDir && !info.IsDir() {
		return fmt.Errorf("extraction path %q is not a directory", path)
	}
	return nil
}
