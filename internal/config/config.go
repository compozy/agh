// Package config loads and validates AGH configuration.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	// DirName is the AGH directory name used for both the global home and workspace overlays.
	DirName = ".agh"
	// ConfigName is the standard TOML configuration filename.
	ConfigName = "config.toml"
)

// DaemonConfig controls the daemon-local socket settings.
type DaemonConfig struct {
	Socket string `toml:"socket"`
}

// HTTPConfig controls the HTTP server bind address.
type HTTPConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

// DefaultsConfig holds global runtime defaults.
type DefaultsConfig struct {
	Agent    string `toml:"agent"`
	Provider string `toml:"provider,omitempty"`
}

// LimitsConfig defines runtime safety bounds.
type LimitsConfig struct {
	MaxSessions         int `toml:"max_sessions"`
	MaxConcurrentAgents int `toml:"max_concurrent_agents"`
}

// PermissionMode is the static permission policy applied by the daemon.
type PermissionMode string

const (
	// DefaultAgentName is the bootstrap agent name used across the system.
	DefaultAgentName                          = "general"
	PermissionModeDenyAll      PermissionMode = "deny-all"
	PermissionModeApproveReads PermissionMode = "approve-reads"
	PermissionModeApproveAll   PermissionMode = "approve-all"
)

// PermissionsConfig defines the global default permission policy.
type PermissionsConfig struct {
	Mode PermissionMode `toml:"mode"`
}

// ObservabilityConfig controls global event retention settings.
type ObservabilityConfig struct {
	Enabled        bool                          `toml:"enabled"`
	RetentionDays  int                           `toml:"retention_days"`
	MaxGlobalBytes int64                         `toml:"max_global_bytes"`
	Transcripts    ObservabilityTranscriptConfig `toml:"transcripts"`
}

// ObservabilityTranscriptConfig configures transcript capture and retention.
type ObservabilityTranscriptConfig struct {
	Enabled            bool  `toml:"enabled"`
	SegmentBytes       int   `toml:"segment_bytes"`
	MaxBytesPerSession int64 `toml:"max_bytes_per_session"`
}

// LogConfig controls structured logging.
type LogConfig struct {
	Level string `toml:"level"`
}

// MemoryConfig controls persistent memory features.
type MemoryConfig struct {
	Enabled   bool        `toml:"enabled"`
	GlobalDir string      `toml:"global_dir,omitempty"`
	Dream     DreamConfig `toml:"dream"`
}

// DreamConfig controls background dream consolidation.
type DreamConfig struct {
	Enabled       bool          `toml:"enabled"`
	Agent         string        `toml:"agent"`
	MinHours      float64       `toml:"min_hours"`
	MinSessions   int           `toml:"min_sessions"`
	CheckInterval time.Duration `toml:"check_interval"`
}

// MarketplaceConfig controls the external skill registry used by CLI skill commands.
type MarketplaceConfig struct {
	Registry string `toml:"registry"`
	BaseURL  string `toml:"base_url,omitempty"`
}

// SkillsConfig controls skill loading and discovery.
type SkillsConfig struct {
	Enabled                 bool              `toml:"enabled"`
	DisabledSkills          []string          `toml:"disabled_skills,omitempty"`
	PollInterval            time.Duration     `toml:"poll_interval"`
	AllowedMarketplaceMCP   []string          `toml:"allowed_marketplace_mcp,omitempty"`
	AllowedMarketplaceHooks []string          `toml:"allowed_marketplace_hooks,omitempty"`
	Marketplace             MarketplaceConfig `toml:"marketplace,omitempty"`
}

// Config is the fully merged AGH configuration.
type Config struct {
	Daemon        DaemonConfig              `toml:"daemon"`
	HTTP          HTTPConfig                `toml:"http"`
	Defaults      DefaultsConfig            `toml:"defaults"`
	Limits        LimitsConfig              `toml:"limits"`
	Permissions   PermissionsConfig         `toml:"permissions"`
	Providers     map[string]ProviderConfig `toml:"providers"`
	Observability ObservabilityConfig       `toml:"observability"`
	Log           LogConfig                 `toml:"log"`
	Memory        MemoryConfig              `toml:"memory"`
	Skills        SkillsConfig              `toml:"skills"`
	Hooks         HooksConfig               `toml:"hooks"`
}

type loadOptions struct {
	workspaceRoot string
	skipDotEnv    bool
	skipValidate  bool
}

// LoadOption customizes configuration loading.
type LoadOption func(*loadOptions)

// WithWorkspaceRoot loads the optional workspace overlay from `<root>/.agh/config.toml`.
// When omitted, Load applies only the built-in defaults and the global AGH home config.
func WithWorkspaceRoot(root string) LoadOption {
	return func(opts *loadOptions) {
		opts.workspaceRoot = root
	}
}

// WithoutDotEnv disables automatic `.env` loading during config load.
func WithoutDotEnv() LoadOption {
	return func(opts *loadOptions) {
		opts.skipDotEnv = true
	}
}

// WithoutValidation returns the merged config without validating it.
func WithoutValidation() LoadOption {
	return func(opts *loadOptions) {
		opts.skipValidate = true
	}
}

// Load reads the default config, the optional global config, and the optional workspace overlay.
// Workspace overlays are loaded only when WithWorkspaceRoot supplies an explicit root.
func Load(opts ...LoadOption) (Config, error) {
	options := loadOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}

	workspaceRoot, err := resolveWorkspaceRoot(options.workspaceRoot)
	if err != nil {
		return Config{}, err
	}

	if !options.skipDotEnv {
		if err := loadDotEnv(workspaceRoot); err != nil {
			return Config{}, err
		}
	}

	homePaths, err := ResolveHomePaths()
	if err != nil {
		return Config{}, err
	}

	return loadWithHome(homePaths, workspaceRoot, options.skipValidate)
}

// LoadForHome reads the default config, the optional global config, and the optional workspace
// overlay using the supplied AGH home layout instead of the ambient process home.
func LoadForHome(homePaths HomePaths, opts ...LoadOption) (Config, error) {
	options := loadOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}

	workspaceRoot, err := resolveWorkspaceRoot(options.workspaceRoot)
	if err != nil {
		return Config{}, err
	}

	if !options.skipDotEnv {
		if err := loadDotEnv(workspaceRoot); err != nil {
			return Config{}, err
		}
	}

	return loadWithHome(homePaths, workspaceRoot, options.skipValidate)
}

func loadWithHome(homePaths HomePaths, workspaceRoot string, skipValidate bool) (Config, error) {
	cfg := DefaultWithHome(homePaths)
	if err := ApplyConfigOverlayFile(homePaths.ConfigFile, &cfg); err != nil {
		return Config{}, fmt.Errorf("load global config: %w", err)
	}
	if workspaceRoot != "" {
		if err := ApplyConfigOverlayFile(workspaceConfigFile(workspaceRoot), &cfg); err != nil {
			return Config{}, fmt.Errorf("load workspace config: %w", err)
		}
	}
	if err := normalizeConfigPaths(&cfg); err != nil {
		return Config{}, err
	}

	if !skipValidate {
		if err := cfg.Validate(); err != nil {
			return Config{}, fmt.Errorf("validate config: %w", err)
		}
	}

	return cfg, nil
}

// Default returns the built-in default configuration for the resolved AGH home.
func Default() (Config, error) {
	homePaths, err := ResolveHomePaths()
	if err != nil {
		return Config{}, err
	}

	return DefaultWithHome(homePaths), nil
}

// DefaultWithHome returns the built-in default configuration for the supplied AGH home.
func DefaultWithHome(homePaths HomePaths) Config {
	return Config{
		Daemon: DaemonConfig{
			Socket: homePaths.DaemonSocket,
		},
		HTTP: HTTPConfig{
			Host: "localhost",
			Port: 2123,
		},
		Defaults: DefaultsConfig{
			Agent: DefaultAgentName,
		},
		Limits: LimitsConfig{
			MaxSessions:         10,
			MaxConcurrentAgents: 20,
		},
		Permissions: PermissionsConfig{
			Mode: PermissionModeApproveAll,
		},
		Providers: map[string]ProviderConfig{},
		Observability: ObservabilityConfig{
			Enabled:        true,
			RetentionDays:  7,
			MaxGlobalBytes: 1 << 30,
			Transcripts: ObservabilityTranscriptConfig{
				Enabled:            true,
				SegmentBytes:       1 << 20,
				MaxBytesPerSession: 256 << 20,
			},
		},
		Log: LogConfig{
			Level: "info",
		},
		Memory: MemoryConfig{
			Enabled:   true,
			GlobalDir: homePaths.MemoryDir,
			Dream: DreamConfig{
				Enabled:       true,
				Agent:         DefaultAgentName,
				MinHours:      24,
				MinSessions:   3,
				CheckInterval: 30 * time.Minute,
			},
		},
		Skills: SkillsConfig{
			Enabled:      true,
			PollInterval: 3 * time.Second,
		},
	}
}

// Validate ensures the loaded configuration is internally consistent.
func (c Config) Validate() error {
	if err := c.Daemon.Validate(); err != nil {
		return err
	}
	if err := c.HTTP.Validate(); err != nil {
		return err
	}
	if err := c.Defaults.Validate(); err != nil {
		return err
	}
	if err := c.Limits.Validate(); err != nil {
		return err
	}
	if err := c.Permissions.Validate(); err != nil {
		return err
	}
	if err := c.Observability.Validate(); err != nil {
		return err
	}
	if err := c.Log.Validate(); err != nil {
		return err
	}
	if err := c.Memory.Validate(); err != nil {
		return err
	}
	if err := c.Skills.Validate(); err != nil {
		return err
	}
	if err := c.Hooks.Validate(); err != nil {
		return fmt.Errorf("validate hooks config: %w", err)
	}

	for name := range c.Providers {
		if _, err := c.ResolveProvider(name); err != nil {
			return err
		}
	}
	if provider := strings.TrimSpace(c.Defaults.Provider); provider != "" {
		if _, err := c.ResolveProvider(provider); err != nil {
			return err
		}
	}

	return nil
}

// Validate ensures the daemon config contains a socket path.
func (c DaemonConfig) Validate() error {
	if strings.TrimSpace(c.Socket) == "" {
		return errors.New("daemon.socket is required")
	}

	return nil
}

// Validate ensures the HTTP bind settings are valid.
func (c HTTPConfig) Validate() error {
	if strings.TrimSpace(c.Host) == "" {
		return errors.New("http.host is required")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("http.port must be between 1 and 65535: %d", c.Port)
	}

	return nil
}

// Validate ensures the default agent setting is present.
func (c DefaultsConfig) Validate() error {
	if strings.TrimSpace(c.Agent) == "" {
		return errors.New("defaults.agent is required")
	}

	return nil
}

// Validate ensures the configured limits are positive.
func (c LimitsConfig) Validate() error {
	switch {
	case c.MaxSessions <= 0:
		return fmt.Errorf("limits.max_sessions must be positive: %d", c.MaxSessions)
	case c.MaxConcurrentAgents <= 0:
		return fmt.Errorf("limits.max_concurrent_agents must be positive: %d", c.MaxConcurrentAgents)
	default:
		return nil
	}
}

// Validate ensures the permission mode is supported.
func (c PermissionsConfig) Validate() error {
	return c.Mode.Validate("permissions.mode")
}

// Validate ensures the permission mode is supported.
func (m PermissionMode) Validate(path string) error {
	switch m {
	case PermissionModeDenyAll, PermissionModeApproveReads, PermissionModeApproveAll:
		return nil
	default:
		return fmt.Errorf("%s must be one of %q, %q, %q: %q", path, PermissionModeDenyAll, PermissionModeApproveReads, PermissionModeApproveAll, m)
	}
}

// Validate ensures observability settings are sensible.
func (c ObservabilityConfig) Validate() error {
	if c.RetentionDays <= 0 {
		return fmt.Errorf("observability.retention_days must be positive: %d", c.RetentionDays)
	}
	if c.MaxGlobalBytes <= 0 {
		return fmt.Errorf("observability.max_global_bytes must be positive: %d", c.MaxGlobalBytes)
	}

	return c.Transcripts.Validate()
}

// Validate ensures transcript retention settings are sensible.
func (c ObservabilityTranscriptConfig) Validate() error {
	if c.SegmentBytes <= 0 {
		return fmt.Errorf("observability.transcripts.segment_bytes must be positive: %d", c.SegmentBytes)
	}
	if c.MaxBytesPerSession <= 0 {
		return fmt.Errorf("observability.transcripts.max_bytes_per_session must be positive: %d", c.MaxBytesPerSession)
	}

	return nil
}

// Validate ensures the log level is supported.
func (c LogConfig) Validate() error {
	switch strings.ToLower(strings.TrimSpace(c.Level)) {
	case "debug", "info", "warn", "error":
		return nil
	default:
		return fmt.Errorf("log.level must be one of %q, %q, %q, %q: %q", "debug", "info", "warn", "error", c.Level)
	}
}

// Validate ensures the memory configuration is internally consistent.
func (c MemoryConfig) Validate() error {
	return c.Dream.Validate()
}

// Validate ensures the skills configuration is internally consistent.
func (c SkillsConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.PollInterval <= 0 {
		return fmt.Errorf("skills.poll_interval must be positive: %s", c.PollInterval)
	}
	if err := c.Marketplace.Validate(); err != nil {
		return err
	}

	return nil
}

// Validate ensures the marketplace configuration is internally consistent when configured.
func (c MarketplaceConfig) Validate() error {
	registry := strings.TrimSpace(c.Registry)
	baseURL := strings.TrimSpace(c.BaseURL)
	if registry == "" && baseURL == "" {
		return nil
	}
	if registry == "" {
		return errors.New("skills.marketplace.registry is required")
	}
	if baseURL != "" {
		parsed, err := url.Parse(baseURL)
		if err != nil {
			return fmt.Errorf("skills.marketplace.base_url is invalid: %w", err)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return fmt.Errorf("skills.marketplace.base_url must use http or https: %q", c.BaseURL)
		}
		if strings.TrimSpace(parsed.Host) == "" {
			return fmt.Errorf("skills.marketplace.base_url must include a host: %q", c.BaseURL)
		}
	}

	switch strings.ToLower(registry) {
	case "clawhub":
		return nil
	default:
		return fmt.Errorf("skills.marketplace.registry must be %q: %q", "clawhub", c.Registry)
	}
}

// Validate ensures the dream configuration is internally consistent.
func (c DreamConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if strings.TrimSpace(c.Agent) == "" {
		return errors.New("memory.dream.agent is required")
	}
	if c.MinHours <= 0 {
		return fmt.Errorf("memory.dream.min_hours must be positive: %v", c.MinHours)
	}
	if c.MinSessions <= 0 {
		return fmt.Errorf("memory.dream.min_sessions must be positive: %d", c.MinSessions)
	}
	if c.CheckInterval <= 0 {
		return fmt.Errorf("memory.dream.check_interval must be positive: %s", c.CheckInterval)
	}
	return nil
}

func normalizeConfigPaths(cfg *Config) error {
	if cfg == nil {
		return errors.New("config is required")
	}

	socket, err := expandUserPath(cfg.Daemon.Socket)
	if err != nil {
		return fmt.Errorf("expand daemon.socket: %w", err)
	}
	cfg.Daemon.Socket = socket

	if strings.TrimSpace(cfg.Memory.GlobalDir) != "" {
		memoryDir, err := expandUserPath(cfg.Memory.GlobalDir)
		if err != nil {
			return fmt.Errorf("expand memory.global_dir: %w", err)
		}
		cfg.Memory.GlobalDir = memoryDir
	}

	return nil
}

func resolveWorkspaceRoot(root string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", nil
	}

	return resolveAbsoluteDir(root)
}

func workspaceConfigFile(root string) string {
	return filepath.Join(root, DirName, ConfigName)
}

func loadDotEnv(workspaceRoot string) error {
	if strings.TrimSpace(workspaceRoot) == "" {
		return nil
	}

	path := filepath.Join(workspaceRoot, ".env")
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat .env file %q: %w", path, err)
	}

	if err := godotenv.Load(path); err != nil {
		return fmt.Errorf("load .env file %q: %w", path, err)
	}

	return nil
}
