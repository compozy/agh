package memory

import (
	"fmt"
	"path/filepath"
)

// ParseHeader decodes and validates memory frontmatter from a raw document.
func ParseHeader(content []byte) (MemoryHeader, error) {
	var header MemoryHeader

	if _, err := parseFrontmatter(content, &header); err != nil {
		return MemoryHeader{}, fmt.Errorf("memory: parse frontmatter: %w", fmt.Errorf("%w: %v", ErrValidation, err))
	}
	if err := header.Validate(); err != nil {
		return MemoryHeader{}, fmt.Errorf("memory: validate frontmatter: %w", fmt.Errorf("%w: %v", ErrValidation, err))
	}

	return header, nil
}

// ConsolidationLockPath returns the canonical lock path for a global memory directory.
func ConsolidationLockPath(globalDir string) string {
	return filepath.Join(cleanDirPath(globalDir), consolidationLockName)
}
