package e2e

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
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
		_ = file.Close()
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
		targetPath := filepath.Join(root, relativePath)
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

	var builder strings.Builder
	builder.WriteString("---\n")
	builder.WriteString("name: " + name + "\n")
	if provider := strings.TrimSpace(seed.Provider); provider != "" {
		builder.WriteString("provider: " + provider + "\n")
	}
	if command := strings.TrimSpace(seed.Command); command != "" {
		builder.WriteString("command: " + command + "\n")
	}
	if model := strings.TrimSpace(seed.Model); model != "" {
		builder.WriteString("model: " + model + "\n")
	}
	if permissions := strings.TrimSpace(seed.Permissions); permissions != "" {
		builder.WriteString("permissions: " + permissions + "\n")
	}
	if len(seed.Tools) > 0 {
		builder.WriteString("tools:\n")
		for _, tool := range seed.Tools {
			builder.WriteString("  - " + strings.TrimSpace(tool) + "\n")
		}
	}
	if len(seed.MCPServers) > 0 {
		builder.WriteString("mcp_servers:\n")
		for _, server := range seed.MCPServers {
			builder.WriteString("  - name: " + strings.TrimSpace(server.Name) + "\n")
			builder.WriteString("    command: " + strings.TrimSpace(server.Command) + "\n")
			if len(server.Args) > 0 {
				builder.WriteString("    args:\n")
				for _, arg := range server.Args {
					builder.WriteString("      - " + arg + "\n")
				}
			}
			if len(server.Env) > 0 {
				builder.WriteString("    env:\n")
				envKeys := make([]string, 0, len(server.Env))
				for key := range server.Env {
					envKeys = append(envKeys, key)
				}
				sortStrings(envKeys)
				for _, key := range envKeys {
					builder.WriteString("      " + key + ": " + server.Env[key] + "\n")
				}
			}
		}
	}
	builder.WriteString("---\n\n")
	builder.WriteString(defaultString(seed.Prompt, "You are "+name+"."))
	builder.WriteString("\n")

	if err := os.WriteFile(path, []byte(builder.String()), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
}

func shortSocketPath(t testing.TB) string {
	t.Helper()

	path := filepath.Join(
		os.TempDir(),
		fmt.Sprintf("agh-e2e-%d-%d.sock", os.Getpid(), time.Now().UTC().UnixNano()),
	)
	t.Cleanup(func() {
		_ = os.Remove(path)
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
