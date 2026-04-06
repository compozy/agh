package skills

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	skillFileName     = "SKILL.md"
	maxScanDepth      = 4
	maxScanCandidates = 300
)

var (
	errFrontmatterMissing      = errors.New("skills: missing YAML frontmatter")
	errFrontmatterUnterminated = errors.New("skills: unterminated YAML frontmatter")
	errSkillNameRequired       = errors.New("skills: skill name is required")
	errScanLimitReached        = errors.New("skills: scan candidate limit reached")
)

var allowedFrontmatterFields = map[string]struct{}{
	"name":        {},
	"description": {},
	"version":     {},
	"metadata":    {},
}

// ParseSkillFile reads and parses a SKILL.md file from disk.
//
// The loader fills the parsed metadata, markdown body, and canonical file
// locations. Callers that know the registry provenance should assign Source
// after parsing.
func ParseSkillFile(path string) (*Skill, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("skills: resolve path %q: %w", path, err)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("skills: read %q: %w", absPath, err)
	}

	meta, body, err := parseFrontmatter(string(content))
	if err != nil {
		return nil, fmt.Errorf("skills: parse %q: %w", absPath, err)
	}
	if meta.Name == "" {
		return nil, fmt.Errorf("skills: parse %q: %w", absPath, errSkillNameRequired)
	}

	skill := &Skill{
		Meta:     meta,
		Content:  body,
		Dir:      filepath.Dir(absPath),
		FilePath: absPath,
		Enabled:  true,
	}
	if skill.Meta.Description == "" {
		slog.Warn("skills: parsed skill without description", "path", absPath, "name", skill.Meta.Name)
	}

	return skill, nil
}

// parseFrontmatter splits YAML frontmatter from the markdown body of a SKILL.md file.
func parseFrontmatter(content string) (SkillMeta, string, error) {
	normalized := normalizeLineEndings(content)
	if !strings.HasPrefix(normalized, "---") {
		return SkillMeta{}, "", errFrontmatterMissing
	}

	openLine, remainder, ok := strings.Cut(normalized, "\n")
	if !ok {
		if normalized == "---" {
			return SkillMeta{}, "", errFrontmatterUnterminated
		}
		return SkillMeta{}, "", errFrontmatterMissing
	}
	if openLine != "---" {
		return SkillMeta{}, "", errFrontmatterMissing
	}

	closeStart, closeEnd, ok := findClosingDelimiter(remainder)
	if !ok {
		return SkillMeta{}, "", errFrontmatterUnterminated
	}

	frontmatter := remainder[:closeStart]
	body := remainder[closeEnd:]
	body = strings.TrimPrefix(body, "\n")

	meta, err := decodeSkillMeta(frontmatter)
	if err != nil {
		return SkillMeta{}, "", fmt.Errorf("decode YAML frontmatter: %w", err)
	}

	return meta, body, nil
}

// scanDirectory returns every SKILL.md file discovered under dir.
func scanDirectory(dir string) ([]string, error) {
	root := strings.TrimSpace(dir)
	if root == "" {
		return nil, errors.New("skills: scan directory root is required")
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("skills: resolve scan root %q: %w", dir, err)
	}

	info, err := os.Stat(absRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("skills: stat scan root %q: %w", absRoot, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("skills: scan root %q is not a directory", absRoot)
	}

	paths := make([]string, 0, maxScanCandidates)
	walkErr := filepath.WalkDir(absRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			slog.Warn("skills: skipping unreadable path during scan", "path", path, "error", walkErr)
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		depth, err := scanDepth(absRoot, path, entry.IsDir())
		if err != nil {
			return fmt.Errorf("skills: determine scan depth for %q: %w", path, err)
		}

		if entry.IsDir() {
			if path != absRoot && shouldSkipDir(entry.Name()) {
				return filepath.SkipDir
			}
			if depth > maxScanDepth {
				return filepath.SkipDir
			}
			return nil
		}

		if depth > maxScanDepth || entry.Name() != skillFileName {
			return nil
		}

		if _, err := snapshotFile(path); err != nil {
			slog.Warn("skills: skipping unreadable skill file during scan", "path", path, "error", err)
			return nil
		}

		paths = append(paths, path)
		if len(paths) >= maxScanCandidates {
			slog.Warn("skills: scan candidate limit reached", "root", absRoot, "limit", maxScanCandidates)
			return errScanLimitReached
		}

		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, errScanLimitReached) {
		return nil, walkErr
	}

	slices.Sort(paths)
	return paths, nil
}

func decodeSkillMeta(frontmatter string) (SkillMeta, error) {
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(frontmatter), &document); err != nil {
		return SkillMeta{}, err
	}

	warnUnknownFields(&document)

	var meta SkillMeta
	if err := yaml.Unmarshal([]byte(frontmatter), &meta); err != nil {
		return SkillMeta{}, err
	}

	meta.Name = strings.TrimSpace(meta.Name)
	meta.Description = strings.TrimSpace(meta.Description)
	meta.Version = strings.TrimSpace(meta.Version)

	return meta, nil
}

func warnUnknownFields(document *yaml.Node) {
	if document == nil || len(document.Content) == 0 {
		return
	}

	root := document.Content[0]
	if root.Kind != yaml.MappingNode {
		return
	}

	for i := 0; i+1 < len(root.Content); i += 2 {
		key := strings.TrimSpace(root.Content[i].Value)
		if _, ok := allowedFrontmatterFields[key]; ok {
			continue
		}

		slog.Warn("skills: unknown frontmatter field", "field", key)
	}
}

func normalizeLineEndings(content string) string {
	return strings.ReplaceAll(content, "\r\n", "\n")
}

func findClosingDelimiter(content string) (int, int, bool) {
	offset := 0
	for offset <= len(content) {
		lineEnd := strings.IndexByte(content[offset:], '\n')
		if lineEnd == -1 {
			if content[offset:] == "---" {
				return offset, len(content), true
			}
			return 0, 0, false
		}

		lineEnd += offset
		if content[offset:lineEnd] == "---" {
			return offset, lineEnd, true
		}

		offset = lineEnd + 1
	}

	return 0, 0, false
}

func scanDepth(root, current string, isDir bool) (int, error) {
	rel, err := filepath.Rel(root, current)
	if err != nil {
		return 0, err
	}
	if rel == "." {
		return 0, nil
	}

	parts := strings.Split(rel, string(filepath.Separator))
	if !isDir {
		return max(len(parts)-1, 0), nil
	}

	return len(parts), nil
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", "node_modules":
		return true
	case ".agh", ".agents":
		return false
	}

	return strings.HasPrefix(name, ".")
}

func snapshotFile(path string) (fileSnapshot, error) {
	info, err := os.Stat(path)
	if err != nil {
		return fileSnapshot{}, err
	}

	return fileSnapshot{
		path:    path,
		modTime: info.ModTime(),
		size:    info.Size(),
	}, nil
}
