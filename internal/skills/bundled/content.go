package bundled

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/pedronauck/agh/internal/frontmatter"
)

const skillFileName = "SKILL.md"

// LoadContent returns the markdown body for one embedded bundled skill.
func LoadContent(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", fmt.Errorf("bundled: skill name is required")
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
