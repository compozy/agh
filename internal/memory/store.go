package memory

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/goccy/go-yaml"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/fileutil"
	"github.com/pedronauck/agh/internal/frontmatter"
)

const (
	indexFilename     = "MEMORY.md"
	maxScanEntries    = 200
	defaultIndexLines = 200
	defaultIndexBytes = 25_000
	dirPerm           = 0o755
	filePerm          = 0o644
	memoryDirName     = "memory"
)

var (
	// ErrValidation marks memory input and metadata validation failures.
	ErrValidation = errors.New("memory: validation error")
)

// Store manages memory files for the global and workspace scopes.
type Store struct {
	globalDir     string
	workspaceDir  string
	maxIndexLines int
	maxIndexBytes int
	logger        *slog.Logger
}

// NewStore constructs a Store for the provided global memory directory.
func NewStore(globalDir string) *Store {
	return &Store{
		globalDir:     cleanDirPath(globalDir),
		maxIndexLines: defaultIndexLines,
		maxIndexBytes: defaultIndexBytes,
		logger:        slog.Default(),
	}
}

// ForWorkspace returns a clone of the store bound to the supplied workspace root.
func (s *Store) ForWorkspace(workspaceRoot string) *Store {
	clone := *s
	clone.workspaceDir = workspaceMemoryDir(workspaceRoot)
	return &clone
}

// EnsureDirs creates the configured memory directories when missing.
func (s *Store) EnsureDirs() error {
	for _, dir := range []string{s.globalDir, s.workspaceDir} {
		if strings.TrimSpace(dir) == "" {
			continue
		}

		if err := os.MkdirAll(dir, dirPerm); err != nil {
			return fmt.Errorf("memory: ensure directory %q: %w", dir, err)
		}
	}

	if strings.TrimSpace(s.globalDir) == "" {
		return wrapValidationError("ensure directory", "global", errors.New("global directory is required"))
	}

	return nil
}

// Read returns the raw file contents for a memory file in the requested scope.
func (s *Store) Read(scope Scope, filename string) ([]byte, error) {
	path, err := s.pathFor(scope, filename)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("memory: read %q: %w", path, err)
	}

	return content, nil
}

// Exists reports whether a memory file exists in the requested scope.
func (s *Store) Exists(scope Scope, filename string) (bool, error) {
	path, err := s.pathFor(scope, filename)
	if err != nil {
		return false, err
	}

	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("memory: stat %q: %w", path, err)
	}

	return true, nil
}

// Write validates the memory frontmatter and persists the raw file contents atomically.
func (s *Store) Write(scope Scope, filename string, content []byte) error {
	var header MemoryHeader
	if _, err := parseFrontmatter(content, &header); err != nil {
		return fmt.Errorf("memory: parse frontmatter %q: %w", filename, fmt.Errorf("%w: %v", ErrValidation, err))
	}
	if err := header.Validate(); err != nil {
		return wrapValidationError("validate frontmatter", filename, err)
	}

	path, err := s.pathFor(scope, filename)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return fmt.Errorf("memory: ensure directory %q: %w", filepath.Dir(path), err)
	}
	if err := fileutil.AtomicWriteFile(path, content, filePerm); err != nil {
		return fmt.Errorf("memory: write %q: %w", path, err)
	}

	return nil
}

// Delete removes a memory file and strips any matching entry from the local MEMORY.md index.
func (s *Store) Delete(scope Scope, filename string) error {
	path, err := s.pathFor(scope, filename)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("memory: delete %q: %w", path, err)
	}
	if filepath.Base(path) == indexFilename {
		return nil
	}
	if err := s.removeIndexEntry(scope, filepath.Base(path)); err != nil {
		return err
	}

	return nil
}

// Scan lists memory headers for a scope, sorted newest-first and capped at 200 files.
func (s *Store) Scan(scope Scope) ([]MemoryHeader, error) {
	dir, err := s.dirForScope(scope)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []MemoryHeader{}, nil
		}
		return nil, fmt.Errorf("memory: scan %q: %w", dir, err)
	}

	headers := make([]MemoryHeader, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || shouldSkipFile(entry.Name()) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			s.warn("memory: skip memory file with unreadable metadata", "scope", scope, "filename", entry.Name(), "error", err)
			continue
		}

		path := filepath.Join(dir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			s.warn("memory: skip unreadable memory file", "scope", scope, "path", path, "error", err)
			continue
		}

		var header MemoryHeader
		if _, err := parseFrontmatter(content, &header); err != nil {
			s.warn("memory: skip malformed memory file", "scope", scope, "path", path, "error", err)
			continue
		}
		if err := header.Validate(); err != nil {
			s.warn("memory: skip invalid memory metadata", "scope", scope, "path", path, "error", err)
			continue
		}

		header.Filename = entry.Name()
		header.FilePath = path
		header.ModTime = info.ModTime()
		headers = append(headers, header)
	}

	sort.SliceStable(headers, func(i, j int) bool {
		if headers[i].ModTime.Equal(headers[j].ModTime) {
			return headers[i].Filename < headers[j].Filename
		}
		return headers[i].ModTime.After(headers[j].ModTime)
	})

	if len(headers) > maxScanEntries {
		headers = headers[:maxScanEntries]
	}

	return headers, nil
}

// LoadIndex reads MEMORY.md for a scope and truncates it to the prompt-safe limits.
func (s *Store) LoadIndex(scope Scope) (content string, truncated bool, err error) {
	dir, err := s.dirForScope(scope)
	if err != nil {
		return "", false, err
	}

	path := filepath.Join(dir, indexFilename)
	indexBytes, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("memory: load index %q: %w", path, err)
	}

	content, truncated = truncateIndex(string(indexBytes), s.maxIndexLines, s.maxIndexBytes)
	if truncated {
		s.warn(
			"memory: truncated memory index",
			"scope", scope,
			"path", path,
			"max_lines", s.maxIndexLines,
			"max_bytes", s.maxIndexBytes,
		)
	}

	return content, truncated, nil
}

func (s *Store) dirForScope(scope Scope) (string, error) {
	normalized := scope.Normalize()
	if err := normalized.Validate(); err != nil {
		return "", wrapValidationError("resolve scope", string(scope), err)
	}

	switch normalized {
	case ScopeGlobal:
		if s.globalDir == "" {
			return "", wrapValidationError("resolve scope", string(scope), errors.New("global directory is required"))
		}
		return s.globalDir, nil
	case ScopeWorkspace:
		if s.workspaceDir == "" {
			return "", wrapValidationError("resolve scope", string(scope), errors.New("workspace directory is required"))
		}
		return s.workspaceDir, nil
	default:
		return "", wrapValidationError("resolve scope", string(scope), fmt.Errorf("unsupported scope %q", scope))
	}
}

func (s *Store) pathFor(scope Scope, filename string) (string, error) {
	dir, err := s.dirForScope(scope)
	if err != nil {
		return "", err
	}

	base, err := cleanFilename(filename)
	if err != nil {
		return "", wrapValidationError("resolve filename", filename, err)
	}

	return filepath.Join(dir, base), nil
}

func (s *Store) removeIndexEntry(scope Scope, filename string) error {
	dir, err := s.dirForScope(scope)
	if err != nil {
		return err
	}

	indexPath := filepath.Join(dir, indexFilename)
	indexContent, err := os.ReadFile(indexPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("memory: load index %q: %w", indexPath, err)
	}

	needle := "(" + filename + ")"
	lines := strings.SplitAfter(string(indexContent), "\n")
	filtered := make([]string, 0, len(lines))
	removed := false
	for _, line := range lines {
		if strings.Contains(line, needle) {
			removed = true
			continue
		}
		filtered = append(filtered, line)
	}
	if !removed {
		return nil
	}

	if err := fileutil.AtomicWriteFile(indexPath, []byte(strings.Join(filtered, "")), filePerm); err != nil {
		return fmt.Errorf("memory: update index %q: %w", indexPath, err)
	}

	return nil
}

func (s *Store) warn(msg string, args ...any) {
	if s.logger != nil {
		s.logger.Warn(msg, args...)
		return
	}

	slog.Warn(msg, args...)
}

func truncateIndex(content string, maxLines int, maxBytes int) (string, bool) {
	if content == "" {
		return "", false
	}
	if maxLines <= 0 || maxBytes <= 0 {
		return "", true
	}

	lines := strings.SplitAfter(content, "\n")
	var builder strings.Builder

	for idx, line := range lines {
		if line == "" && idx == len(lines)-1 {
			continue
		}
		if idx >= maxLines {
			return builder.String(), true
		}
		if builder.Len()+len(line) > maxBytes {
			if builder.Len() == 0 {
				return truncateToUTF8Boundary(line, maxBytes), true
			}
			return builder.String(), true
		}
		builder.WriteString(line)
	}

	return builder.String(), false
}

func truncateToUTF8Boundary(value string, maxBytes int) string {
	if maxBytes <= 0 || value == "" {
		return ""
	}
	if len(value) <= maxBytes {
		return value
	}

	truncated := value[:maxBytes]
	for len(truncated) > 0 && !utf8.ValidString(truncated) {
		truncated = truncated[:len(truncated)-1]
	}

	return truncated
}

func shouldSkipFile(name string) bool {
	return name == indexFilename || strings.HasPrefix(name, ".")
}

func cleanDirPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}

	return filepath.Clean(trimmed)
}

func cleanFilename(filename string) (string, error) {
	trimmed := strings.TrimSpace(filename)
	if trimmed == "" {
		return "", fmt.Errorf("filename is required")
	}
	if trimmed == "." || trimmed == ".." {
		return "", fmt.Errorf("filename %q is invalid", filename)
	}
	if strings.ContainsAny(trimmed, `/\`) {
		return "", fmt.Errorf("filename %q must not include path separators", filename)
	}

	return trimmed, nil
}

func wrapValidationError(operation string, target string, err error) error {
	return fmt.Errorf("memory: %s %q: %w", operation, target, fmt.Errorf("%w: %v", ErrValidation, err))
}

func workspaceMemoryDir(workspaceRoot string) string {
	trimmed := strings.TrimSpace(workspaceRoot)
	if trimmed == "" {
		return ""
	}

	return filepath.Join(filepath.Clean(trimmed), aghconfig.DirName, memoryDirName)
}

func parseFrontmatter(content []byte, dest any) (string, error) {
	return frontmatter.Decode(content, func(data []byte) error {
		if err := yaml.UnmarshalWithOptions(data, dest, yaml.Strict()); err != nil {
			return fmt.Errorf("decode YAML: %w", err)
		}
		return nil
	})
}
