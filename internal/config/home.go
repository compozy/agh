package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// AgentsDirName is the directory used for persisted agent definitions.
	AgentsDirName = "agents"
	// SkillsDirName is the directory used for persisted user skills.
	SkillsDirName = "skills"
	// MemoryDirName is the directory used for persistent memory files.
	MemoryDirName = "memory"
	// SessionsDirName is the directory used for persisted session state.
	SessionsDirName = "sessions"
	// LogsDirName is the directory used for structured logs.
	LogsDirName = "logs"
	// DatabaseName is the global database filename.
	DatabaseName = "agh.db"
	// DaemonSocketName is the daemon UDS filename.
	DaemonSocketName = "daemon.sock"
	// DaemonLockName is the daemon file-lock name.
	DaemonLockName = "daemon.lock"
	// DaemonInfoName is the daemon metadata filename.
	DaemonInfoName = "daemon.json"
	// LogFileName is the structured daemon log filename.
	LogFileName  = "agh.log"
	agentDefName = "AGENT.md"
)

// HomePaths captures the filesystem layout for the AGH home directory.
type HomePaths struct {
	HomeDir      string
	ConfigFile   string
	AgentsDir    string
	SkillsDir    string
	MemoryDir    string
	SessionsDir  string
	LogsDir      string
	LogFile      string
	DatabaseFile string
	DaemonSocket string
	DaemonLock   string
	DaemonInfo   string
}

// ResolveHomeDir resolves the global AGH home directory, honoring AGH_HOME when present.
func ResolveHomeDir() (string, error) {
	if override := strings.TrimSpace(os.Getenv("AGH_HOME")); override != "" {
		return resolveAbsoluteDir(override)
	}

	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}

	return filepath.Join(userHome, DirName), nil
}

// ResolveHomePaths resolves the canonical AGH home layout.
func ResolveHomePaths() (HomePaths, error) {
	homeDir, err := ResolveHomeDir()
	if err != nil {
		return HomePaths{}, err
	}

	return ResolveHomePathsFrom(homeDir)
}

// ResolveHomePathsFrom resolves the canonical AGH home layout from an explicit directory.
func ResolveHomePathsFrom(homeDir string) (HomePaths, error) {
	root, err := resolveAbsoluteDir(homeDir)
	if err != nil {
		return HomePaths{}, err
	}

	return HomePaths{
		HomeDir:      root,
		ConfigFile:   filepath.Join(root, ConfigName),
		AgentsDir:    filepath.Join(root, AgentsDirName),
		SkillsDir:    filepath.Join(root, SkillsDirName),
		MemoryDir:    filepath.Join(root, MemoryDirName),
		SessionsDir:  filepath.Join(root, SessionsDirName),
		LogsDir:      filepath.Join(root, LogsDirName),
		LogFile:      filepath.Join(root, LogsDirName, LogFileName),
		DatabaseFile: filepath.Join(root, DatabaseName),
		DaemonSocket: filepath.Join(root, DaemonSocketName),
		DaemonLock:   filepath.Join(root, DaemonLockName),
		DaemonInfo:   filepath.Join(root, DaemonInfoName),
	}, nil
}

// EnsureHomeLayout creates the directories required by the AGH home layout.
func EnsureHomeLayout(paths HomePaths) error {
	for _, dir := range []string{
		paths.HomeDir,
		paths.AgentsDir,
		paths.SkillsDir,
		paths.MemoryDir,
		paths.SessionsDir,
		paths.LogsDir,
	} {
		if strings.TrimSpace(dir) == "" {
			return errors.New("config: home path is required")
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create agh directory %q: %w", dir, err)
		}
	}

	return nil
}

func resolveAbsoluteDir(path string) (string, error) {
	absPath, err := ResolvePath(path)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(absPath) == "" {
		return "", errors.New("config: path is required")
	}
	return absPath, nil
}

// ResolvePath expands `~`-prefixed paths and returns an absolute path.
func ResolvePath(path string) (string, error) {
	expanded, err := expandUserPath(path)
	if err != nil {
		return "", err
	}

	clean := strings.TrimSpace(expanded)
	if clean == "" {
		return "", nil
	}

	absPath, err := filepath.Abs(clean)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path %q: %w", path, err)
	}

	return absPath, nil
}

// ResolveUserAgentsSkillsDir resolves the user-level `.agents/skills` directory.
func ResolveUserAgentsSkillsDir(getenv func(string) string) (string, error) {
	if getenv != nil {
		if home := strings.TrimSpace(getenv("HOME")); home != "" {
			resolvedHome, err := ResolvePath(home)
			if err != nil {
				return "", fmt.Errorf("config: resolve HOME for user agent skills: %w", err)
			}
			return filepath.Join(resolvedHome, ".agents", "skills"), nil
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("config: resolve user home for agent skills: %w", err)
	}

	resolvedHome, err := ResolvePath(home)
	if err != nil {
		return "", fmt.Errorf("config: resolve user home for agent skills: %w", err)
	}

	return filepath.Join(resolvedHome, ".agents", "skills"), nil
}

func expandUserPath(path string) (string, error) {
	clean := strings.TrimSpace(path)
	if clean == "" {
		return "", nil
	}
	if clean == "~" {
		return os.UserHomeDir()
	}
	if !strings.HasPrefix(clean, "~/") {
		return clean, nil
	}

	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}

	return filepath.Join(userHome, clean[2:]), nil
}
