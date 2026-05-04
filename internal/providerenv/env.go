package providerenv

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	aghconfig "github.com/pedronauck/agh/internal/config"
)

// ApplyHomePolicy updates provider launch environment according to the provider
// home policy. Operator home intentionally leaves the incoming env untouched.
func ApplyHomePolicy(
	homePaths aghconfig.HomePaths,
	providerName string,
	homePolicy aghconfig.ProviderHomePolicy,
	env []string,
) ([]string, error) {
	if homePolicy != aghconfig.ProviderHomePolicyIsolated {
		return env, nil
	}
	trimmedProvider, providerHome, err := isolatedProviderHome(homePaths, providerName)
	if err != nil {
		return nil, err
	}
	for _, dir := range []string{
		providerHome,
		filepath.Join(providerHome, ".config"),
		filepath.Join(providerHome, ".local", "share"),
		filepath.Join(providerHome, ".cache"),
	} {
		if err := EnsurePrivateDir(dir); err != nil {
			return nil, err
		}
	}

	env = SetEnvValue(env, "PROVIDER_HOME", providerHome)
	env = SetEnvValue(env, "HOME", providerHome)
	env = SetEnvValue(env, "XDG_CONFIG_HOME", filepath.Join(providerHome, ".config"))
	env = SetEnvValue(env, "XDG_DATA_HOME", filepath.Join(providerHome, ".local", "share"))
	env = SetEnvValue(env, "XDG_CACHE_HOME", filepath.Join(providerHome, ".cache"))
	return setKnownProviderHomeEnv(trimmedProvider, providerHome, env)
}

// ApplyPiAgentDirPolicy points native Pi auth at the same isolated home used by
// both session launch and provider auth commands.
func ApplyPiAgentDirPolicy(
	homePaths aghconfig.HomePaths,
	providerName string,
	homePolicy aghconfig.ProviderHomePolicy,
	env []string,
) ([]string, error) {
	if homePolicy != aghconfig.ProviderHomePolicyIsolated {
		return env, nil
	}
	_, providerHome, err := isolatedProviderHome(homePaths, providerName)
	if err != nil {
		return nil, err
	}
	agentDir := filepath.Join(providerHome, ".pi", "agent")
	if err := EnsurePrivateDir(agentDir); err != nil {
		return nil, err
	}
	return SetEnvValue(env, "PI_CODING_AGENT_DIR", agentDir), nil
}

// EnsurePrivateDir creates or tightens an AGH-owned provider state directory.
func EnsurePrivateDir(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("create isolated provider directory %q: %w", path, err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		return fmt.Errorf("secure isolated provider directory %q: %w", path, err)
	}
	return nil
}

// SetEnvValue returns env with exactly one KEY=value entry.
func SetEnvValue(env []string, key string, value string) []string {
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" || strings.Contains(trimmedKey, "=") {
		return env
	}
	entry := trimmedKey + "=" + value
	filtered := env[:0]
	for _, current := range env {
		if strings.HasPrefix(current, trimmedKey+"=") {
			continue
		}
		filtered = append(filtered, current)
	}
	return append(filtered, entry)
}

// SafeProviderHomeSegment reports whether a provider name can be used as one
// path segment below $AGH_HOME/providers.
func SafeProviderHomeSegment(value string) bool {
	if value == "" || value == "." || value == ".." {
		return false
	}
	for _, r := range value {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '-' || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func isolatedProviderHome(homePaths aghconfig.HomePaths, providerName string) (string, string, error) {
	trimmedProvider := strings.TrimSpace(providerName)
	if !SafeProviderHomeSegment(trimmedProvider) {
		return "", "", fmt.Errorf("provider %q cannot use isolated home policy", trimmedProvider)
	}
	if strings.TrimSpace(homePaths.HomeDir) == "" {
		return "", "", errors.New("AGH home is required for isolated provider home policy")
	}
	return trimmedProvider, filepath.Join(homePaths.HomeDir, "providers", trimmedProvider), nil
}

func setKnownProviderHomeEnv(providerName string, providerHome string, env []string) ([]string, error) {
	knownDirs := map[string]map[string]string{
		"claude": {
			"CLAUDE_CONFIG_DIR": filepath.Join(providerHome, "claude"),
		},
		"codex": {
			"CODEX_HOME":          filepath.Join(providerHome, "codex"),
			"PROVIDER_CODEX_HOME": filepath.Join(providerHome, "codex"),
		},
		"opencode": {
			"OPENCODE_CONFIG_DIR": filepath.Join(providerHome, "opencode"),
		},
	}
	dirs := knownDirs[providerName]
	if len(dirs) == 0 {
		return env, nil
	}
	for _, dir := range dirs {
		if err := EnsurePrivateDir(dir); err != nil {
			return nil, err
		}
	}
	for key, dir := range dirs {
		env = SetEnvValue(env, key, dir)
	}
	return env, nil
}
