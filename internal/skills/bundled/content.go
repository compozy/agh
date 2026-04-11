package bundled

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/pedronauck/agh/internal/frontmatter"
)

const skillFileName = "SKILL.md"

var (
	// ErrSkillNameRequired reports missing bundled skill identifiers.
	ErrSkillNameRequired = errors.New("bundled: skill name is required")
	// ErrInvalidSkillName reports bundled skill identifiers that are not a single clean path component.
	ErrInvalidSkillName = errors.New("bundled: invalid skill name")
)

// LoadContent returns the markdown body for one embedded bundled skill.
func LoadContent(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", ErrSkillNameRequired
	}
	if !validSkillName(trimmed) {
		return "", fmt.Errorf("%w: %q", ErrInvalidSkillName, name)
	}

	skillPath := path.Join("skills", trimmed, skillFileName)
	content, err := fs.ReadFile(FS(), skillPath)
	if err != nil {
		return "", fmt.Errorf("bundled: read %q: %w", skillPath, err)
	}

	parts, err := frontmatter.Split(content)
	if err != nil {
		return "", fmt.Errorf("bundled: parse %q frontmatter: %w", skillPath, err)
	}

	body := strings.TrimSpace(parts.Body)
	if body == "" {
		return "", fmt.Errorf("bundled: %q body is empty", skillPath)
	}

	return body, nil
}

func validSkillName(name string) bool {
	switch {
	case name == ".", name == "..":
		return false
	case strings.Contains(name, "/"), strings.Contains(name, `\`):
		return false
	default:
		return path.Clean(name) == name
	}
}
