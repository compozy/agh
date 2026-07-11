package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	// ErrAgentDefinitionNotFound marks an authored definition that no longer exists.
	ErrAgentDefinitionNotFound = errors.New("config: agent definition not found")
)

// DeleteAgentDefinition removes the complete authored directory for one effective definition.
func DeleteAgentDefinition(sourcePath string) error {
	cleaned := filepath.Clean(strings.TrimSpace(sourcePath))
	if cleaned == "." || !strings.EqualFold(filepath.Base(cleaned), AgentDefinitionFileName) {
		return fmt.Errorf("config: invalid agent definition source path %q", sourcePath)
	}
	info, err := os.Lstat(cleaned)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return errors.Join(
				ErrAgentDefinitionNotFound,
				fmt.Errorf("config: agent definition %q: %w", cleaned, os.ErrNotExist),
			)
		}
		return fmt.Errorf("config: inspect agent definition %q: %w", cleaned, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("config: agent definition %q must be a regular file", cleaned)
	}
	agentDir := filepath.Dir(cleaned)
	if err := os.RemoveAll(agentDir); err != nil {
		return fmt.Errorf("config: remove agent definition directory %q: %w", agentDir, err)
	}
	return nil
}
