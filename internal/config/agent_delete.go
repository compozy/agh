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

// DeleteAgentDefinition removes one authored directory beneath an authorized agents root.
func DeleteAgentDefinition(agentsRoot string, sourcePath string) (retErr error) {
	cleanRoot := filepath.Clean(strings.TrimSpace(agentsRoot))
	cleaned := filepath.Clean(strings.TrimSpace(sourcePath))
	if cleanRoot == "." ||
		cleaned == "." ||
		!filepath.IsAbs(cleanRoot) ||
		!filepath.IsAbs(cleaned) ||
		filepath.Base(cleanRoot) != AgentsDirName ||
		!strings.EqualFold(filepath.Base(cleaned), AgentDefinitionFileName) {
		return fmt.Errorf("config: invalid agent definition source path %q", sourcePath)
	}
	agentDir := filepath.Dir(cleaned)
	agentName := filepath.Base(agentDir)
	expectedPath := filepath.Join(cleanRoot, agentName, AgentDefinitionFileName)
	if agentName == "." || agentName == AgentsDirName || cleaned != expectedPath {
		return fmt.Errorf("config: invalid agent definition delete target %q", sourcePath)
	}
	rootInfo, err := os.Lstat(cleanRoot)
	if err != nil {
		return fmt.Errorf("config: inspect agents root %q: %w", cleanRoot, err)
	}
	if !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("config: agents root %q must be a real directory", cleanRoot)
	}
	root, err := os.OpenRoot(cleanRoot)
	if err != nil {
		return fmt.Errorf("config: open agents root %q: %w", cleanRoot, err)
	}
	defer func() {
		if closeErr := root.Close(); closeErr != nil {
			retErr = errors.Join(retErr, fmt.Errorf("config: close agents root %q: %w", cleanRoot, closeErr))
		}
	}()

	agentInfo, err := root.Lstat(agentName)
	if errors.Is(err, os.ErrNotExist) {
		return agentDefinitionNotFoundError(cleaned)
	}
	if err != nil {
		return fmt.Errorf("config: inspect agent directory %q: %w", agentDir, err)
	}
	if !agentInfo.IsDir() || agentInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("config: agent directory %q must be a real directory", agentDir)
	}
	definitionPath := filepath.Join(agentName, AgentDefinitionFileName)
	definitionInfo, err := root.Lstat(definitionPath)
	if errors.Is(err, os.ErrNotExist) {
		return agentDefinitionNotFoundError(cleaned)
	}
	if err != nil {
		return fmt.Errorf("config: inspect agent definition %q: %w", cleaned, err)
	}
	if !definitionInfo.Mode().IsRegular() {
		return fmt.Errorf("config: agent definition %q must be a regular file", cleaned)
	}
	if err := root.RemoveAll(agentName); err != nil {
		return fmt.Errorf("config: remove agent definition directory %q: %w", agentDir, err)
	}
	return nil
}

func agentDefinitionNotFoundError(sourcePath string) error {
	return errors.Join(
		ErrAgentDefinitionNotFound,
		fmt.Errorf("config: agent definition %q: %w", sourcePath, os.ErrNotExist),
	)
}
