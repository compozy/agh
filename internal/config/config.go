// Package config loads and validates AGH configuration.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/joho/godotenv"
	automationpkg "github.com/pedronauck/agh/internal/automation/model"
	"github.com/pedronauck/agh/internal/extension/surfaces"
	"github.com/pedronauck/agh/internal/resources"
)

const (
	// DirName is the AGH directory name used for both the global home and workspace overlays.
	DirName = ".agh"
	// ConfigName is the standard TOML configuration filename.
	ConfigName = "config.toml"
	// marketplaceSchemeHTTP is the accepted plaintext marketplace URL scheme.
	marketplaceSchemeHTTP = "http"
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

// SessionConfig defines session-scoped runtime controls.
type SessionConfig struct {
	Limits SessionLimitsConfig `toml:"limits"`
}

// SessionLimitsConfig defines runtime limits applied to every session.
type SessionLimitsConfig struct {
	Timeout time.Duration `toml:"timeout,omitempty"`
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

// ExtensionsMarketplaceConfig controls the external extension registry used by CLI extension commands.
type ExtensionsMarketplaceConfig struct {
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

// ExtensionsConfig controls extension marketplace discovery and install behavior.
type ExtensionsConfig struct {
	Marketplace ExtensionsMarketplaceConfig `toml:"marketplace,omitempty"`
	Resources   ExtensionsResourcesConfig   `toml:"resources,omitempty"`
}

// ExtensionsResourcesConfig controls resource publication policy for extensions.
type ExtensionsResourcesConfig struct {
	AllowedKinds           []resources.ResourceKind          `toml:"allowed_kinds,omitempty"`
	MaxScope               resources.ResourceScopeKind       `toml:"max_scope,omitempty"`
	SnapshotRateLimit      ExtensionsResourceRateLimitConfig `toml:"snapshot_rate_limit,omitempty"`
	OperatorWriteRateLimit ExtensionsResourceRateLimitConfig `toml:"operator_write_rate_limit,omitempty"`
}

// ExtensionsResourceRateLimitConfig controls one resource publication rate-limit bucket.
type ExtensionsResourceRateLimitConfig struct {
	Requests int           `toml:"requests"`
	Window   time.Duration `toml:"window"`
	Queue    int           `toml:"queue"`
}

// NetworkConfig controls the embedded AGH network runtime.
type NetworkConfig struct {
	Enabled        bool   `toml:"enabled"`
	DefaultChannel string `toml:"default_channel"`
	Port           int    `toml:"port"`
	MaxPayload     int    `toml:"max_payload"`
	GreetInterval  int    `toml:"greet_interval"`
	MaxReplayAge   int    `toml:"max_replay_age"`
	MaxQueueDepth  int    `toml:"max_queue_depth"`
}

// Config is the fully merged AGH configuration.
type Config struct {
	Daemon        DaemonConfig              `toml:"daemon"`
	HTTP          HTTPConfig                `toml:"http"`
	Defaults      DefaultsConfig            `toml:"defaults"`
	Limits        LimitsConfig              `toml:"limits"`
	Session       SessionConfig             `toml:"session"`
	Permissions   PermissionsConfig         `toml:"permissions"`
	MCPServers    []MCPServer               `toml:"mcp_servers,omitempty"`
	Providers     map[string]ProviderConfig `toml:"providers"`
	Observability ObservabilityConfig       `toml:"observability"`
	Log           LogConfig                 `toml:"log"`
	Memory        MemoryConfig              `toml:"memory"`
	Skills        SkillsConfig              `toml:"skills"`
	Extensions    ExtensionsConfig          `toml:"extensions"`
	Automation    AutomationConfig          `toml:"automation"`
	Hooks         HooksConfig               `toml:"hooks"`
	Network       NetworkConfig             `toml:"network"`
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

func withoutDotEnv() LoadOption {
	return func(opts *loadOptions) {
		opts.skipDotEnv = true
	}
}

func withoutValidation() LoadOption {
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
	if err := applyConfigMCPSidecarFile(globalMCPJSONFile(homePaths), &cfg); err != nil {
		return Config{}, fmt.Errorf("load global MCP JSON: %w", err)
	}
	if workspaceRoot != "" {
		if err := ApplyConfigOverlayFile(workspaceConfigFile(workspaceRoot), &cfg); err != nil {
			return Config{}, fmt.Errorf("load workspace config: %w", err)
		}
		if err := applyConfigMCPSidecarFile(workspaceMCPJSONFile(workspaceRoot), &cfg); err != nil {
			return Config{}, fmt.Errorf("load workspace MCP JSON: %w", err)
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

func defaultConfig() (Config, error) {
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
		Session: SessionConfig{
			Limits: SessionLimitsConfig{},
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
		Extensions: ExtensionsConfig{},
		Automation: AutomationConfig{
			Enabled:           true,
			Timezone:          automationpkg.DefaultTimezone,
			MaxConcurrentJobs: automationpkg.DefaultMaxConcurrentJobs,
			DefaultFireLimit:  automationpkg.DefaultFireLimitConfig(),
		},
		Network: NetworkConfig{
			Enabled:        false,
			DefaultChannel: "default",
			Port:           -1,
			MaxPayload:     1 << 20,
			GreetInterval:  30,
			MaxReplayAge:   300,
			MaxQueueDepth:  100,
		},
	}
}

// Validate ensures the loaded configuration is internally consistent.
func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config is required")
	}
	if err := c.validateCore(); err != nil {
		return err
	}
	if err := c.validateFeatures(); err != nil {
		return err
	}
	if err := c.validateProviders(); err != nil {
		return err
	}
	return nil
}

func (c *Config) validateCore() error {
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
	if err := c.Session.Validate(); err != nil {
		return err
	}
	if err := c.Permissions.Validate(); err != nil {
		return err
	}
	for i, server := range c.MCPServers {
		if err := server.Validate(fmt.Sprintf("mcp_servers[%d]", i)); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) validateFeatures() error {
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
	if err := c.Extensions.Validate(); err != nil {
		return err
	}
	if err := c.Automation.Validate(); err != nil {
		return fmt.Errorf("validate automation config: %w", err)
	}
	if err := c.Hooks.Validate(); err != nil {
		return fmt.Errorf("validate hooks config: %w", err)
	}
	if err := c.Network.Validate(); err != nil {
		return fmt.Errorf("validate network config: %w", err)
	}
	return nil
}

func (c *Config) validateProviders() error {
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

// Validate ensures session-scoped controls are internally consistent.
func (c SessionConfig) Validate() error {
	return c.Limits.Validate()
}

// Validate ensures session timeout settings are internally consistent.
func (c SessionLimitsConfig) Validate() error {
	if c.Timeout < 0 {
		return fmt.Errorf("session.limits.timeout must be zero or positive: %s", c.Timeout)
	}
	return nil
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
		return fmt.Errorf(
			"%s must be one of %q, %q, %q: %q",
			path,
			PermissionModeDenyAll,
			PermissionModeApproveReads,
			PermissionModeApproveAll,
			m,
		)
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

// Validate ensures the extension marketplace configuration is internally consistent.
func (c ExtensionsConfig) Validate() error {
	if err := c.Marketplace.Validate(); err != nil {
		return err
	}
	return c.Resources.Validate()
}

// Validate ensures the extension resource policy is internally consistent.
func (c ExtensionsResourcesConfig) Validate() error {
	if _, err := surfaces.NormalizeAllowedKinds(c.AllowedKinds); err != nil {
		return fmt.Errorf("extensions.resources.allowed_kinds: %w", err)
	}
	if c.MaxScope != "" {
		if err := c.MaxScope.Validate("extensions.resources.max_scope"); err != nil {
			return err
		}
	}
	if err := c.SnapshotRateLimit.Validate("extensions.resources.snapshot_rate_limit"); err != nil {
		return err
	}
	if err := c.OperatorWriteRateLimit.Validate("extensions.resources.operator_write_rate_limit"); err != nil {
		return err
	}
	return nil
}

// Validate ensures one configured resource rate-limit bucket is internally consistent.
func (c ExtensionsResourceRateLimitConfig) Validate(path string) error {
	if c.Requests == 0 && c.Window == 0 && c.Queue == 0 {
		return nil
	}
	if c.Requests <= 0 {
		return fmt.Errorf("%s.requests must be positive: %d", path, c.Requests)
	}
	if c.Window <= 0 {
		return fmt.Errorf("%s.window must be positive: %s", path, c.Window)
	}
	if c.Queue < 0 {
		return fmt.Errorf("%s.queue must be zero or positive: %d", path, c.Queue)
	}
	return nil
}

var networkChannelPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

const maxNetworkDurationSeconds = int64(1<<63-1) / int64(time.Second)

// Validate ensures the network configuration is internally consistent.
func (c NetworkConfig) Validate() error {
	defaultChannel := strings.TrimSpace(c.DefaultChannel)
	if defaultChannel == "" {
		return errors.New("network.default_channel is required")
	}
	if !networkChannelPattern.MatchString(defaultChannel) {
		return fmt.Errorf("network.default_channel must match %q: %q", networkChannelPattern.String(), c.DefaultChannel)
	}
	if c.Port != -1 && (c.Port <= 0 || c.Port > 65535) {
		return fmt.Errorf("network.port must be -1 or between 1 and 65535: %d", c.Port)
	}
	if c.MaxPayload <= 0 {
		return fmt.Errorf("network.max_payload must be positive: %d", c.MaxPayload)
	}
	if c.MaxPayload > (1<<31 - 1) {
		return fmt.Errorf("network.max_payload must be <= %d: %d", 1<<31-1, c.MaxPayload)
	}
	if c.GreetInterval <= 0 {
		return fmt.Errorf("network.greet_interval must be positive seconds: %d", c.GreetInterval)
	}
	if int64(c.GreetInterval) > maxNetworkDurationSeconds {
		return fmt.Errorf(
			"network.greet_interval must be between 1 and %d seconds: %d",
			maxNetworkDurationSeconds,
			c.GreetInterval,
		)
	}
	if c.MaxReplayAge <= 0 {
		return fmt.Errorf("network.max_replay_age must be positive seconds: %d", c.MaxReplayAge)
	}
	if int64(c.MaxReplayAge) > maxNetworkDurationSeconds {
		return fmt.Errorf(
			"network.max_replay_age must be between 1 and %d seconds: %d",
			maxNetworkDurationSeconds,
			c.MaxReplayAge,
		)
	}
	if c.MaxQueueDepth <= 0 {
		return fmt.Errorf("network.max_queue_depth must be positive: %d", c.MaxQueueDepth)
	}

	return nil
}

// GreetIntervalDuration returns the configured heartbeat interval as a duration.
func (c NetworkConfig) GreetIntervalDuration() time.Duration {
	return time.Duration(c.GreetInterval) * time.Second
}

// MaxReplayAgeDuration returns the configured replay age window as a duration.
func (c NetworkConfig) MaxReplayAgeDuration() time.Duration {
	return time.Duration(c.MaxReplayAge) * time.Second
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
		if parsed.Scheme != marketplaceSchemeHTTP && parsed.Scheme != "https" {
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

// Validate ensures the extension marketplace configuration is internally consistent when configured.
func (c ExtensionsMarketplaceConfig) Validate() error {
	const githubRegistry = "github"

	registry := strings.TrimSpace(c.Registry)
	baseURL := strings.TrimSpace(c.BaseURL)
	if registry == "" && baseURL == "" {
		return nil
	}
	if registry == "" {
		return errors.New("extensions.marketplace.registry is required")
	}
	if baseURL != "" {
		parsed, err := url.Parse(baseURL)
		if err != nil {
			return fmt.Errorf("extensions.marketplace.base_url is invalid: %w", err)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return fmt.Errorf("extensions.marketplace.base_url must use http or https: %q", c.BaseURL)
		}
		if strings.TrimSpace(parsed.Host) == "" {
			return fmt.Errorf("extensions.marketplace.base_url must include a host: %q", c.BaseURL)
		}
		if parsed.Scheme == "http" {
			slog.Warn("config: extensions marketplace base_url uses insecure http scheme", "url", c.BaseURL)
		}
	}

	switch strings.ToLower(registry) {
	case githubRegistry:
		return nil
	default:
		return fmt.Errorf("extensions.marketplace.registry must be %q: %q", githubRegistry, c.Registry)
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

func applyConfigMCPSidecarFile(path string, cfg *Config) error {
	if cfg == nil {
		return errors.New("config is required")
	}

	servers, err := LoadMCPServersJSONFile(path)
	if err != nil {
		return err
	}
	if len(servers) == 0 {
		return nil
	}

	cfg.MCPServers = OverrideMCPServers(cfg.MCPServers, servers)
	return nil
}

func globalMCPJSONFile(homePaths HomePaths) string {
	return filepath.Join(homePaths.HomeDir, MCPJSONName)
}

func workspaceMCPJSONFile(root string) string {
	trimmed := strings.TrimSpace(root)
	if trimmed == "" {
		return ""
	}

	return filepath.Join(trimmed, DirName, MCPJSONName)
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
