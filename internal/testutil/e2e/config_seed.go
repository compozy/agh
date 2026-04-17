package e2e

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/goccy/go-yaml"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/testutil"
)

// AgentSeed defines one persisted AGENT.md fixture.
type AgentSeed struct {
	Name        string
	Provider    string
	Command     string
	Model       string
	Permissions string
	Tools       []string
	MCPServers  []aghconfig.MCPServer
	Prompt      string
}

// ConfigSeedOptions configures the seeded daemon runtime config.
type ConfigSeedOptions struct {
	Host               string
	HTTPPort           int
	SocketPath         string
	DefaultAgent       string
	DefaultProvider    string
	DefaultEnvironment string
	PermissionMode     aghconfig.PermissionMode
	Providers          map[string]aghconfig.ProviderConfig
	Environments       map[string]aghconfig.EnvironmentProfile
	AgentDefs          []AgentSeed
	Mutate             func(*aghconfig.Config)
}

// WorkspaceSeedOptions configures the seeded workspace root.
type WorkspaceSeedOptions struct {
	Root  string
	Files map[string]string
}

type configSeedFile struct {
	Daemon       *configSeedDaemonSection                `toml:"daemon,omitempty"`
	HTTP         *configSeedHTTPSection                  `toml:"http,omitempty"`
	Defaults     *configSeedDefaultsSection              `toml:"defaults,omitempty"`
	Permissions  *configSeedPermissionsSection           `toml:"permissions,omitempty"`
	Network      *aghconfig.NetworkConfig                `toml:"network,omitempty"`
	Providers    map[string]aghconfig.ProviderConfig     `toml:"providers,omitempty"`
	Environments map[string]aghconfig.EnvironmentProfile `toml:"environments,omitempty"`
}

type configSeedDaemonSection struct {
	Socket string `toml:"socket"`
}

type configSeedHTTPSection struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

type configSeedDefaultsSection struct {
	Agent       string `toml:"agent,omitempty"`
	Provider    string `toml:"provider,omitempty"`
	Environment string `toml:"environment,omitempty"`
}

type configSeedPermissionsSection struct {
	Mode aghconfig.PermissionMode `toml:"mode,omitempty"`
}

// NewHomePaths creates an isolated AGH home layout for one test run.
func NewHomePaths(t testing.TB) aghconfig.HomePaths {
	t.Helper()

	homePaths, err := aghconfig.ResolveHomePathsFrom(t.TempDir())
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	if err := aghconfig.EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}
	return homePaths
}

// SeedConfig writes a minimal config overlay and any requested agent definitions.
func SeedConfig(t testing.TB, homePaths aghconfig.HomePaths, opts ConfigSeedOptions) aghconfig.Config {
	t.Helper()

	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.HTTP.Host = defaultString(opts.Host, "127.0.0.1")
	if opts.HTTPPort > 0 {
		cfg.HTTP.Port = opts.HTTPPort
	} else {
		cfg.HTTP.Port = testutil.FreeTCPPort(t)
	}
	cfg.Daemon.Socket = defaultString(opts.SocketPath, shortSocketPath(t))
	if trimmed := strings.TrimSpace(opts.DefaultAgent); trimmed != "" {
		cfg.Defaults.Agent = trimmed
	}
	if trimmed := strings.TrimSpace(opts.DefaultProvider); trimmed != "" {
		cfg.Defaults.Provider = trimmed
	}
	if trimmed := strings.TrimSpace(opts.DefaultEnvironment); trimmed != "" {
		cfg.Defaults.Environment = trimmed
	}
	if opts.PermissionMode != "" {
		cfg.Permissions.Mode = opts.PermissionMode
	}
	if len(opts.Providers) > 0 {
		cfg.Providers = cloneProviders(opts.Providers)
	}
	if len(opts.Environments) > 0 {
		cfg.Environments = cloneEnvironmentProfiles(opts.Environments)
	}
	if opts.Mutate != nil {
		opts.Mutate(&cfg)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("seed config validate error = %v", err)
	}

	overlay := configSeedFile{
		Daemon: &configSeedDaemonSection{
			Socket: cfg.Daemon.Socket,
		},
		HTTP: &configSeedHTTPSection{
			Host: cfg.HTTP.Host,
			Port: cfg.HTTP.Port,
		},
		Defaults: &configSeedDefaultsSection{
			Agent:       cfg.Defaults.Agent,
			Provider:    cfg.Defaults.Provider,
			Environment: cfg.Defaults.Environment,
		},
		Network:      &cfg.Network,
		Providers:    cloneProviders(cfg.Providers),
		Environments: cloneEnvironmentProfiles(cfg.Environments),
	}
	if cfg.Permissions.Mode != "" {
		overlay.Permissions = &configSeedPermissionsSection{Mode: cfg.Permissions.Mode}
	}

	file, err := os.Create(homePaths.ConfigFile)
	if err != nil {
		t.Fatalf("os.Create(%q) error = %v", homePaths.ConfigFile, err)
	}
	if err := toml.NewEncoder(file).Encode(overlay); err != nil {
		if closeErr := file.Close(); closeErr != nil {
			t.Fatalf("toml encode config %q error = %v (close error = %v)", homePaths.ConfigFile, err, closeErr)
		}
		t.Fatalf("toml encode config %q error = %v", homePaths.ConfigFile, err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close config %q error = %v", homePaths.ConfigFile, err)
	}

	for _, agent := range opts.AgentDefs {
		WriteAgentDef(t, homePaths, agent)
	}

	return cfg
}

// SeedWorkspace creates an isolated workspace root and any requested files.
func SeedWorkspace(t testing.TB, opts WorkspaceSeedOptions) string {
	t.Helper()

	root := strings.TrimSpace(opts.Root)
	if root == "" {
		root = t.TempDir()
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", root, err)
	}

	for relativePath, contents := range opts.Files {
		targetPath, err := seedWorkspaceTargetPath(root, relativePath)
		if err != nil {
			t.Fatalf("seed workspace path %q error = %v", relativePath, err)
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			t.Fatalf("os.MkdirAll(%q) error = %v", filepath.Dir(targetPath), err)
		}
		if err := os.WriteFile(targetPath, []byte(contents), 0o600); err != nil {
			t.Fatalf("os.WriteFile(%q) error = %v", targetPath, err)
		}
	}

	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err == nil {
		return canonicalRoot
	}
	return root
}

func seedWorkspaceTargetPath(root string, relativePath string) (string, error) {
	trimmedRoot := strings.TrimSpace(root)
	if trimmedRoot == "" {
		return "", fmt.Errorf("workspace root is required")
	}
	cleanedRoot := filepath.Clean(trimmedRoot)

	targetPath := filepath.Clean(filepath.Join(cleanedRoot, relativePath))
	relativeTarget, err := filepath.Rel(cleanedRoot, targetPath)
	if err != nil {
		return "", fmt.Errorf("rel workspace target %q: %w", targetPath, err)
	}
	if relativeTarget == "." {
		return "", fmt.Errorf("workspace path %q must reference a file", relativePath)
	}
	if relativeTarget == ".." || strings.HasPrefix(relativeTarget, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("workspace path %q escapes root %q", relativePath, cleanedRoot)
	}

	return targetPath, nil
}

func cloneEnvironmentProfiles(
	profiles map[string]aghconfig.EnvironmentProfile,
) map[string]aghconfig.EnvironmentProfile {
	if len(profiles) == 0 {
		return nil
	}
	cloned := make(map[string]aghconfig.EnvironmentProfile, len(profiles))
	for name, profile := range profiles {
		next := profile
		next.Env = maps.Clone(profile.Env)
		next.Network.AllowList = append([]string(nil), profile.Network.AllowList...)
		next.Network.DenyList = append([]string(nil), profile.Network.DenyList...)
		cloned[name] = next
	}
	return cloned
}

// WriteAgentDef persists one AGENT.md fixture under the supplied home.
func WriteAgentDef(t testing.TB, homePaths aghconfig.HomePaths, seed AgentSeed) {
	t.Helper()

	name := strings.TrimSpace(seed.Name)
	if name == "" {
		t.Fatal("agent seed name is required")
	}

	path := filepath.Join(homePaths.AgentsDir, name, "AGENT.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}

	content, err := renderSeedAgentDef(seed)
	if err != nil {
		t.Fatalf("render seed agent def %q error = %v", name, err)
	}

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
}

type seedAgentDefFrontmatter struct {
	Name        string                 `yaml:"name"`
	Provider    string                 `yaml:"provider,omitempty"`
	Command     string                 `yaml:"command,omitempty"`
	Model       string                 `yaml:"model,omitempty"`
	Permissions string                 `yaml:"permissions,omitempty"`
	Tools       []string               `yaml:"tools,omitempty"`
	MCPServers  []seedAgentMCPServerFM `yaml:"mcp_servers,omitempty"`
}

type seedAgentMCPServerFM struct {
	Name    string            `yaml:"name"`
	Command string            `yaml:"command"`
	Args    []string          `yaml:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
}

func renderSeedAgentDef(seed AgentSeed) (string, error) {
	name := strings.TrimSpace(seed.Name)
	if name == "" {
		return "", fmt.Errorf("agent seed name is required")
	}

	metadata := seedAgentDefFrontmatter{
		Name:        name,
		Provider:    strings.TrimSpace(seed.Provider),
		Command:     strings.TrimSpace(seed.Command),
		Model:       strings.TrimSpace(seed.Model),
		Permissions: strings.TrimSpace(seed.Permissions),
		Tools:       trimSeedValues(seed.Tools),
		MCPServers:  make([]seedAgentMCPServerFM, 0, len(seed.MCPServers)),
	}
	for _, server := range seed.MCPServers {
		metadata.MCPServers = append(metadata.MCPServers, seedAgentMCPServerFM{
			Name:    strings.TrimSpace(server.Name),
			Command: strings.TrimSpace(server.Command),
			Args:    append([]string(nil), server.Args...),
			Env:     maps.Clone(server.Env),
		})
	}

	frontmatterBytes, err := yaml.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("marshal agent frontmatter: %w", err)
	}

	var builder strings.Builder
	builder.WriteString("---\n")
	builder.Write(frontmatterBytes)
	builder.WriteString("---\n\n")
	builder.WriteString(defaultString(seed.Prompt, "You are "+name+"."))
	builder.WriteString("\n")
	return builder.String(), nil
}

func trimSeedValues(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		trimmed = append(trimmed, strings.TrimSpace(value))
	}
	return trimmed
}

func shortSocketPath(t testing.TB) string {
	t.Helper()

	path := filepath.Join(
		os.TempDir(),
		fmt.Sprintf("agh-e2e-%d-%d.sock", os.Getpid(), time.Now().UTC().UnixNano()),
	)
	t.Cleanup(func() {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Logf("remove socket %q error = %v", path, err)
		}
	})
	return path
}

func cloneProviders(in map[string]aghconfig.ProviderConfig) map[string]aghconfig.ProviderConfig {
	if len(in) == 0 {
		return map[string]aghconfig.ProviderConfig{}
	}
	out := make(map[string]aghconfig.ProviderConfig, len(in))
	maps.Copy(out, in)
	return out
}

func defaultString(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func sortStrings(values []string) {
	if len(values) < 2 {
		return
	}
	for i := 0; i < len(values)-1; i++ {
		for j := i + 1; j < len(values); j++ {
			if values[j] < values[i] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}
}
