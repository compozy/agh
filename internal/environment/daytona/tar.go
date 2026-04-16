package daytona

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

var errUnsafeTarPath = errors.New("environment/daytona: unsafe tar path")

type tarStats struct {
	Files int
	Bytes int64
}

type archiveEntry struct {
	path string
	rel  string
	info fs.FileInfo
	link string
}

func writeTar(ctx context.Context, root string, dst io.Writer, excludePatterns []string) (tarStats, error) {
	root = filepath.Clean(root)
	entries, err := collectArchiveEntries(ctx, root, excludePatterns)
	if err != nil {
		return tarStats{}, err
	}
	writer := tar.NewWriter(dst)
	defer writer.Close()
	var stats tarStats
	for _, entry := range entries {
		header, err := tar.FileInfoHeader(entry.info, entry.link)
		if err != nil {
			return tarStats{}, fmt.Errorf("environment/daytona: build tar header for %q: %w", entry.path, err)
		}
		header.Name = entry.rel
		if err := writer.WriteHeader(header); err != nil {
			return tarStats{}, fmt.Errorf("environment/daytona: write tar header for %q: %w", entry.rel, err)
		}
		if entry.info.Mode().IsRegular() {
			written, err := copyArchiveFile(writer, entry)
			if err != nil {
				return tarStats{}, err
			}
			stats.Files++
			stats.Bytes += written
		}
	}
	return stats, nil
}

func collectArchiveEntries(ctx context.Context, root string, excludePatterns []string) ([]archiveEntry, error) {
	var entries []archiveEntry
	err := filepath.WalkDir(root, func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if filePath == root {
			return nil
		}
		rel, err := filepath.Rel(root, filePath)
		if err != nil {
			return fmt.Errorf("environment/daytona: make tar relative path: %w", err)
		}
		rel = filepath.ToSlash(rel)
		if shouldExcludeArchivePath(rel, excludePatterns) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("environment/daytona: stat %q for tar: %w", filePath, err)
		}
		var link string
		if info.Mode()&os.ModeSymlink != 0 {
			link, err = os.Readlink(filePath)
			if err != nil {
				return fmt.Errorf("environment/daytona: read symlink %q: %w", filePath, err)
			}
		}
		entries = append(entries, archiveEntry{path: filePath, rel: rel, info: info, link: link})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func copyArchiveFile(writer io.Writer, entry archiveEntry) (int64, error) {
	file, err := os.Open(entry.path)
	if err != nil {
		return 0, fmt.Errorf("environment/daytona: open %q for tar: %w", entry.path, err)
	}
	written, copyErr := io.Copy(writer, file)
	closeErr := file.Close()
	if copyErr != nil {
		return 0, fmt.Errorf("environment/daytona: write tar file %q: %w", entry.rel, copyErr)
	}
	if closeErr != nil {
		return 0, fmt.Errorf("environment/daytona: close tar source %q: %w", entry.path, closeErr)
	}
	return written, nil
}

func extractTar(root string, src io.Reader) (tarStats, error) {
	root = filepath.Clean(root)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return tarStats{}, fmt.Errorf("environment/daytona: create extract root %q: %w", root, err)
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return tarStats{}, fmt.Errorf("environment/daytona: evaluate extract root %q: %w", root, err)
	}

	reader := tar.NewReader(src)
	var stats tarStats
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return stats, nil
		}
		if err != nil {
			return tarStats{}, fmt.Errorf("environment/daytona: read tar header: %w", err)
		}
		name, err := safeArchiveName(header.Name)
		if err != nil {
			return tarStats{}, err
		}
		target := filepath.Join(realRoot, filepath.FromSlash(name))
		if !isWithinRoot(realRoot, target) {
			return tarStats{}, fmt.Errorf("%w: %q escapes %q", errUnsafeTarPath, header.Name, realRoot)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, modePerm(header.FileInfo().Mode(), 0o755)); err != nil {
				return tarStats{}, fmt.Errorf("environment/daytona: create directory %q: %w", target, err)
			}
		case tar.TypeReg:
			if err := ensureSafeParent(realRoot, target); err != nil {
				return tarStats{}, err
			}
			file, err := os.OpenFile(
				target,
				os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
				modePerm(header.FileInfo().Mode(), 0o600),
			)
			if err != nil {
				return tarStats{}, fmt.Errorf("environment/daytona: create extracted file %q: %w", target, err)
			}
			written, copyErr := io.CopyN(file, reader, header.Size)
			closeErr := file.Close()
			if copyErr != nil {
				return tarStats{}, fmt.Errorf("environment/daytona: write extracted file %q: %w", target, copyErr)
			}
			if closeErr != nil {
				return tarStats{}, fmt.Errorf("environment/daytona: close extracted file %q: %w", target, closeErr)
			}
			stats.Files++
			stats.Bytes += written
		case tar.TypeSymlink:
			if err := ensureSafeParent(realRoot, target); err != nil {
				return tarStats{}, err
			}
			linkTarget, err := safeSymlinkTarget(realRoot, target, header.Linkname)
			if err != nil {
				return tarStats{}, err
			}
			if err := os.RemoveAll(target); err != nil {
				return tarStats{}, fmt.Errorf("environment/daytona: replace symlink %q: %w", target, err)
			}
			if err := os.Symlink(linkTarget, target); err != nil {
				return tarStats{}, fmt.Errorf("environment/daytona: create symlink %q: %w", target, err)
			}
		default:
			return tarStats{}, fmt.Errorf(
				"environment/daytona: unsupported tar entry %q mode %v",
				header.Name,
				header.Typeflag,
			)
		}
	}
}

func safeArchiveName(name string) (string, error) {
	cleaned := path.Clean(strings.TrimSpace(name))
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("%w: empty path", errUnsafeTarPath)
	}
	if path.IsAbs(cleaned) {
		return "", fmt.Errorf("%w: absolute path %q", errUnsafeTarPath, name)
	}
	if slices.Contains(strings.Split(cleaned, "/"), "..") {
		return "", fmt.Errorf("%w: traversal path %q", errUnsafeTarPath, name)
	}
	return cleaned, nil
}

func safeSymlinkTarget(root string, target string, linkName string) (string, error) {
	if strings.TrimSpace(linkName) == "" {
		return "", fmt.Errorf("%w: empty symlink target", errUnsafeTarPath)
	}
	if filepath.IsAbs(linkName) {
		if !isWithinRoot(root, linkName) {
			return "", fmt.Errorf("%w: symlink %q escapes %q", errUnsafeTarPath, linkName, root)
		}
		return linkName, nil
	}
	resolved := filepath.Clean(filepath.Join(filepath.Dir(target), linkName))
	if !isWithinRoot(root, resolved) {
		return "", fmt.Errorf("%w: symlink %q escapes %q", errUnsafeTarPath, linkName, root)
	}
	return linkName, nil
}

func ensureSafeParent(root string, target string) error {
	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("environment/daytona: create parent %q: %w", parent, err)
	}
	realParent, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return fmt.Errorf("environment/daytona: evaluate parent %q: %w", parent, err)
	}
	if !isWithinRoot(root, realParent) {
		return fmt.Errorf("%w: parent %q escapes %q", errUnsafeTarPath, realParent, root)
	}
	if info, err := os.Lstat(target); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: refusing to overwrite symlink %q", errUnsafeTarPath, target)
	}
	return nil
}

func shouldExcludeArchivePath(rel string, excludePatterns []string) bool {
	for part := range strings.SplitSeq(rel, "/") {
		switch part {
		case "node_modules", "dist", "build", "target", ".cache", ".next", ".turbo":
			return true
		}
	}
	for _, pattern := range excludePatterns {
		if archivePatternMatches(pattern, rel) {
			return true
		}
	}
	return false
}

func archivePatternMatches(pattern string, rel string) bool {
	pattern = strings.TrimSpace(filepath.ToSlash(pattern))
	if pattern == "" {
		return false
	}
	rel = strings.TrimSpace(filepath.ToSlash(rel))
	if rel == "" {
		return false
	}
	trimmed := strings.TrimSuffix(pattern, "/")
	if trimmed != "" && (rel == trimmed || strings.HasPrefix(rel, trimmed+"/")) {
		return true
	}
	if matched, err := path.Match(pattern, rel); err == nil && matched {
		return true
	}
	if !strings.Contains(pattern, "/") {
		if matched, err := path.Match(pattern, path.Base(rel)); err == nil && matched {
			return true
		}
	}
	return false
}

func isWithinRoot(root string, target string) bool {
	cleanRoot := filepath.Clean(root)
	cleanTarget := filepath.Clean(target)
	if cleanRoot == cleanTarget {
		return true
	}
	return strings.HasPrefix(cleanTarget, cleanRoot+string(os.PathSeparator))
}

func modePerm(mode fs.FileMode, fallback fs.FileMode) fs.FileMode {
	perm := mode.Perm()
	if perm == 0 {
		return fallback
	}
	return perm
}
