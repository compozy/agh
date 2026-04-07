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

	"github.com/pedronauck/agh/internal/filesnap"
	"github.com/pedronauck/agh/internal/frontmatter"
	"gopkg.in/yaml.v3"
)

const (
	skillFileName     = "SKILL.md"
	maxScanDepth      = 4
	maxScanCandidates = 300
)

var (
	errSkillNameRequired = errors.New("skills: skill name is required")
	errScanLimitReached  = errors.New("skills: scan candidate limit reached")
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

	meta, body, err := parseSkillContent(content)
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

// scanDirectory returns every SKILL.md file discovered under dir.
func scanDirectory(dir string) ([]string, error) {
	paths, _, err := scanDirectoryWithSnapshots(dir)
	return paths, err
}

func scanDirectoryWithSnapshots(dir string) ([]string, map[string]filesnap.Snapshot, error) {
	root := strings.TrimSpace(dir)
	if root == "" {
		return nil, nil, errors.New("skills: scan directory root is required")
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, nil, fmt.Errorf("skills: resolve scan root %q: %w", dir, err)
	}

	info, err := os.Stat(absRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []string{}, map[string]filesnap.Snapshot{}, nil
		}
		return nil, nil, fmt.Errorf("skills: stat scan root %q: %w", absRoot, err)
	}
	if !info.IsDir() {
		return nil, nil, fmt.Errorf("skills: scan root %q is not a directory", absRoot)
	}

	paths := make([]string, 0, maxScanCandidates)
	snapshots := make(map[string]filesnap.Snapshot, maxScanCandidates)
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

		snapshot, err := filesnap.FromPath(path)
		if err != nil {
			slog.Warn("skills: skipping unreadable skill file during scan", "path", path, "error", err)
			return nil
		}

		paths = append(paths, path)
		snapshots[path] = snapshot
		if len(paths) >= maxScanCandidates {
			slog.Warn("skills: scan candidate limit reached", "root", absRoot, "limit", maxScanCandidates)
			return errScanLimitReached
		}

		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, errScanLimitReached) {
		return nil, nil, walkErr
	}

	slices.Sort(paths)
	return paths, snapshots, nil
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

func parseSkillContent(content []byte) (SkillMeta, string, error) {
	parts, err := frontmatter.Split(content)
	if err != nil {
		return SkillMeta{}, "", err
	}

	meta, err := decodeSkillMeta(string(parts.Metadata))
	if err != nil {
		return SkillMeta{}, "", fmt.Errorf("decode YAML frontmatter: %w", err)
	}

	return meta, parts.Body, nil
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
