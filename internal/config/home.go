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
	// RestartsDirName is the directory used for persisted daemon restart operations.
	RestartsDirName = "restarts"
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
	LogFileName = "agh.log"
	// NetworkAuditFileName is the append-only network audit filename.
	NetworkAuditFileName = "network.audit"
	// AgentDefinitionFileName is the canonical file name for persisted agent definitions.
	AgentDefinitionFileName = "AGENT.md"
	agentDefName            = AgentDefinitionFileName
)

// HomePaths captures the filesystem layout for the AGH home directory.
type HomePaths struct {
	HomeDir          string
	ConfigFile       string
	AgentsDir        string
	SkillsDir        string
	MemoryDir        string
	SessionsDir      string
	RestartsDir      string
	LogsDir          string
	LogFile          string
	NetworkAuditFile string
	DatabaseFile     string
	DaemonSocket     string
	DaemonLock       string
	DaemonInfo       string
}

// ResolveHomeDir resolves the global AGH home directory, honoring AGH_HOME when present.
func ResolveHomeDir() (string, error) {
	return resolveHomeDir(processEnvLookup)
}

func resolveHomeDir(lookup envLookup) (string, error) {
	if lookup != nil {
		if override, ok := lookup("AGH_HOME"); ok && strings.TrimSpace(override) != "" {
			return resolveAbsoluteDir(override)
		}
	}

	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}

	return filepath.Join(userHome, DirName), nil
}

// ResolveHomePaths resolves the canonical AGH home layout.
func ResolveHomePaths() (HomePaths, error) {
	return resolveHomePaths(processEnvLookup)
}

// ResolveHomePathsForWorkspace resolves the canonical AGH home layout while
// honoring AGH_HOME from the supplied workspace .env when the process env omits it.
func ResolveHomePathsForWorkspace(workspaceRoot string) (HomePaths, error) {
	workspaceRoot, err := resolveWorkspaceRoot(workspaceRoot)
	if err != nil {
		return HomePaths{}, err
	}
	lookup := processEnvLookup
	dotenvLookup, err := loadDotEnvLookup(workspaceRoot)
	if err != nil {
		return HomePaths{}, err
	}
	if dotenvLookup != nil {
		lookup = layeredEnvLookup(processEnvLookup, dotenvLookup)
	}
	return resolveHomePaths(lookup)
}

func resolveHomePaths(lookup envLookup) (HomePaths, error) {
	homeDir, err := resolveHomeDir(lookup)
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
		HomeDir:          root,
		ConfigFile:       filepath.Join(root, ConfigName),
		AgentsDir:        filepath.Join(root, AgentsDirName),
		SkillsDir:        filepath.Join(root, SkillsDirName),
		MemoryDir:        filepath.Join(root, MemoryDirName),
		SessionsDir:      filepath.Join(root, SessionsDirName),
		RestartsDir:      filepath.Join(root, RestartsDirName),
		LogsDir:          filepath.Join(root, LogsDirName),
		LogFile:          filepath.Join(root, LogsDirName, LogFileName),
		NetworkAuditFile: filepath.Join(root, LogsDirName, NetworkAuditFileName),
		DatabaseFile:     filepath.Join(root, DatabaseName),
		DaemonSocket:     filepath.Join(root, DaemonSocketName),
		DaemonLock:       filepath.Join(root, DaemonLockName),
		DaemonInfo:       filepath.Join(root, DaemonInfoName),
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
		paths.RestartsDir,
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
