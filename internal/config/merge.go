package config

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
)

type configOverlay struct {
	Daemon        daemonOverlay              `toml:"daemon"`
	HTTP          httpOverlay                `toml:"http"`
	Defaults      defaultsOverlay            `toml:"defaults"`
	Limits        limitsOverlay              `toml:"limits"`
	Session       sessionOverlay             `toml:"session"`
	Permissions   permissionsOverlay         `toml:"permissions"`
	MCPServers    []mcpServerOverlay         `toml:"mcp_servers"`
	Providers     map[string]providerOverlay `toml:"providers"`
	Observability observabilityOverlay       `toml:"observability"`
	Log           logOverlay                 `toml:"log"`
	Memory        memoryOverlay              `toml:"memory"`
	Skills        skillsOverlay              `toml:"skills"`
	Automation    automationOverlay          `toml:"automation"`
	Hooks         hooksOverlay               `toml:"hooks"`
}

type daemonOverlay struct {
	Socket *string `toml:"socket"`
}

type httpOverlay struct {
	Host *string `toml:"host"`
	Port *int    `toml:"port"`
}

type defaultsOverlay struct {
	Agent    *string `toml:"agent"`
	Provider *string `toml:"provider"`
}

type limitsOverlay struct {
	MaxSessions         *int `toml:"max_sessions"`
	MaxConcurrentAgents *int `toml:"max_concurrent_agents"`
}

type sessionOverlay struct {
	Limits sessionLimitsOverlay `toml:"limits"`
}

type sessionLimitsOverlay struct {
	Timeout *time.Duration `toml:"timeout"`
}

type permissionsOverlay struct {
	Mode *PermissionMode `toml:"mode"`
}

type providerOverlay struct {
	Command      *string            `toml:"command"`
	DefaultModel *string            `toml:"default_model"`
	APIKeyEnv    *string            `toml:"api_key_env"`
	MCPServers   []mcpServerOverlay `toml:"mcp_servers"`
}

type observabilityOverlay struct {
	Enabled        *bool                           `toml:"enabled"`
	RetentionDays  *int                            `toml:"retention_days"`
	MaxGlobalBytes *int64                          `toml:"max_global_bytes"`
	Transcripts    observabilityTranscriptsOverlay `toml:"transcripts"`
}

type observabilityTranscriptsOverlay struct {
	Enabled            *bool  `toml:"enabled"`
	SegmentBytes       *int   `toml:"segment_bytes"`
	MaxBytesPerSession *int64 `toml:"max_bytes_per_session"`
}

type logOverlay struct {
	Level *string `toml:"level"`
}

type memoryOverlay struct {
	Enabled   *bool        `toml:"enabled"`
	GlobalDir *string      `toml:"global_dir"`
	Dream     dreamOverlay `toml:"dream"`
}

type dreamOverlay struct {
	Enabled       *bool          `toml:"enabled"`
	Agent         *string        `toml:"agent"`
	MinHours      *float64       `toml:"min_hours"`
	MinSessions   *int           `toml:"min_sessions"`
	CheckInterval *time.Duration `toml:"check_interval"`
}

type skillsOverlay struct {
	Enabled                 *bool              `toml:"enabled"`
	DisabledSkills          *[]string          `toml:"disabled_skills"`
	PollInterval            *time.Duration     `toml:"poll_interval"`
	AllowedMarketplaceMCP   *[]string          `toml:"allowed_marketplace_mcp"`
	AllowedMarketplaceHooks *[]string          `toml:"allowed_marketplace_hooks"`
	Marketplace             marketplaceOverlay `toml:"marketplace"`
}

type marketplaceOverlay struct {
	Registry *string `toml:"registry"`
	BaseURL  *string `toml:"base_url"`
}

type hooksOverlay struct {
	Declarations []parsedHookDeclaration `toml:"declarations"`
}

type mcpServerOverlay struct {
	Name    *string            `toml:"name"`
	Command *string            `toml:"command"`
	Args    *[]string          `toml:"args"`
	Env     *map[string]string `toml:"env"`
}

// ApplyConfigOverlayFile deep-merges an optional TOML config file into dst.
func ApplyConfigOverlayFile(path string, dst *Config) error {
	if dst == nil {
		return errors.New("config: destination config is required")
	}

	overlay, err := loadConfigOverlayFile(path)
	if err != nil {
		return err
	}

	return overlay.Apply(dst)
}

func loadConfigOverlayFile(path string) (configOverlay, error) {
	var overlay configOverlay

	contents, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return overlay, nil
		}
		return overlay, fmt.Errorf("read config file %q: %w", path, err)
	}

	meta, err := toml.Decode(string(contents), &overlay)
	if err != nil {
		return overlay, fmt.Errorf("decode config file %q: %w", path, err)
	}

	if undecoded := meta.Undecoded(); len(undecoded) > 0 {
		return overlay, fmt.Errorf("unknown config keys in %q: %s", path, joinTOMLKeys(undecoded))
	}

	return overlay, nil
}

func (o configOverlay) Apply(dst *Config) error {
	o.Daemon.Apply(&dst.Daemon)
	o.HTTP.Apply(&dst.HTTP)
	o.Defaults.Apply(&dst.Defaults)
	o.Limits.Apply(&dst.Limits)
	o.Session.Apply(&dst.Session)
	o.Permissions.Apply(&dst.Permissions)
	if len(o.MCPServers) > 0 {
		dst.MCPServers = applyMCPServerOverlays(dst.MCPServers, o.MCPServers)
	}
	applyProviderOverlays(dst, o.Providers)
	o.Observability.Apply(&dst.Observability)
	o.Log.Apply(&dst.Log)
	o.Memory.Apply(&dst.Memory)
	o.Skills.Apply(&dst.Skills)
	if err := o.Automation.Apply(&dst.Automation); err != nil {
		return err
	}
	return o.Hooks.Apply(&dst.Hooks)
}

func (o daemonOverlay) Apply(dst *DaemonConfig) {
	if o.Socket != nil {
		dst.Socket = *o.Socket
	}
}

func (o httpOverlay) Apply(dst *HTTPConfig) {
	if o.Host != nil {
		dst.Host = *o.Host
	}
	if o.Port != nil {
		dst.Port = *o.Port
	}
}

func (o defaultsOverlay) Apply(dst *DefaultsConfig) {
	if o.Agent != nil {
		dst.Agent = *o.Agent
	}
	if o.Provider != nil {
		dst.Provider = *o.Provider
	}
}

func (o limitsOverlay) Apply(dst *LimitsConfig) {
	if o.MaxSessions != nil {
		dst.MaxSessions = *o.MaxSessions
	}
	if o.MaxConcurrentAgents != nil {
		dst.MaxConcurrentAgents = *o.MaxConcurrentAgents
	}
}

func (o sessionOverlay) Apply(dst *SessionConfig) {
	o.Limits.Apply(&dst.Limits)
}

func (o sessionLimitsOverlay) Apply(dst *SessionLimitsConfig) {
	if o.Timeout != nil {
		dst.Timeout = *o.Timeout
	}
}

func (o permissionsOverlay) Apply(dst *PermissionsConfig) {
	if o.Mode != nil {
		dst.Mode = *o.Mode
	}
}

func (o providerOverlay) Apply(dst *ProviderConfig) {
	if o.Command != nil {
		dst.Command = *o.Command
	}
	if o.DefaultModel != nil {
		dst.DefaultModel = *o.DefaultModel
	}
	if o.APIKeyEnv != nil {
		dst.APIKeyEnv = *o.APIKeyEnv
	}
	if len(o.MCPServers) > 0 {
		dst.MCPServers = applyMCPServerOverlays(dst.MCPServers, o.MCPServers)
	}
}

func (o observabilityOverlay) Apply(dst *ObservabilityConfig) {
	if o.Enabled != nil {
		dst.Enabled = *o.Enabled
	}
	if o.RetentionDays != nil {
		dst.RetentionDays = *o.RetentionDays
	}
	if o.MaxGlobalBytes != nil {
		dst.MaxGlobalBytes = *o.MaxGlobalBytes
	}
	o.Transcripts.Apply(&dst.Transcripts)
}

func (o observabilityTranscriptsOverlay) Apply(dst *ObservabilityTranscriptConfig) {
	if o.Enabled != nil {
		dst.Enabled = *o.Enabled
	}
	if o.SegmentBytes != nil {
		dst.SegmentBytes = *o.SegmentBytes
	}
	if o.MaxBytesPerSession != nil {
		dst.MaxBytesPerSession = *o.MaxBytesPerSession
	}
}

func (o logOverlay) Apply(dst *LogConfig) {
	if o.Level != nil {
		dst.Level = *o.Level
	}
}

func (o memoryOverlay) Apply(dst *MemoryConfig) {
	if o.Enabled != nil {
		dst.Enabled = *o.Enabled
	}
	if o.GlobalDir != nil && strings.TrimSpace(*o.GlobalDir) != "" {
		dst.GlobalDir = *o.GlobalDir
	}
	o.Dream.Apply(&dst.Dream)
}

func (o dreamOverlay) Apply(dst *DreamConfig) {
	if o.Enabled != nil {
		dst.Enabled = *o.Enabled
	}
	if o.Agent != nil {
		dst.Agent = *o.Agent
	}
	if o.MinHours != nil {
		dst.MinHours = *o.MinHours
	}
	if o.MinSessions != nil {
		dst.MinSessions = *o.MinSessions
	}
	if o.CheckInterval != nil {
		dst.CheckInterval = *o.CheckInterval
	}
}

func (o skillsOverlay) Apply(dst *SkillsConfig) {
	if o.Enabled != nil {
		dst.Enabled = *o.Enabled
	}
	if o.DisabledSkills != nil {
		dst.DisabledSkills = append([]string(nil), (*o.DisabledSkills)...)
	}
	if o.PollInterval != nil {
		dst.PollInterval = *o.PollInterval
	}
	if o.AllowedMarketplaceMCP != nil {
		dst.AllowedMarketplaceMCP = append([]string(nil), (*o.AllowedMarketplaceMCP)...)
	}
	if o.AllowedMarketplaceHooks != nil {
		dst.AllowedMarketplaceHooks = append([]string(nil), (*o.AllowedMarketplaceHooks)...)
	}
	o.Marketplace.Apply(&dst.Marketplace)
}

func (o marketplaceOverlay) Apply(dst *MarketplaceConfig) {
	if o.Registry != nil {
		dst.Registry = *o.Registry
	}
	if o.BaseURL != nil {
		dst.BaseURL = *o.BaseURL
	}
}

func (o hooksOverlay) Apply(dst *HooksConfig) error {
	if len(o.Declarations) == 0 {
		return nil
	}

	merged := cloneHookDecls(dst.Declarations)
	index := make(map[string]int, len(merged))
	for i, decl := range merged {
		if name := strings.TrimSpace(decl.Name); name != "" {
			index[name] = i
		}
	}

	for idx, raw := range o.Declarations {
		decl, err := raw.toHookDecl(hookspkg.HookSourceConfig, "")
		if err != nil {
			return fmt.Errorf("hooks.declarations[%d]: %w", idx, err)
		}

		name := strings.TrimSpace(decl.Name)
		if existingIdx, ok := index[name]; ok && name != "" {
			merged[existingIdx] = decl
			continue
		}

		merged = append(merged, decl)
		if name != "" {
			index[name] = len(merged) - 1
		}
	}

	dst.Declarations = merged
	return nil
}

func (o mcpServerOverlay) Apply(dst *MCPServer) {
	if o.Name != nil {
		dst.Name = *o.Name
	}
	if o.Command != nil {
		dst.Command = *o.Command
	}
	if o.Args != nil {
		dst.Args = append([]string(nil), (*o.Args)...)
	}
	if o.Env != nil {
		dst.Env = mergeStringMaps(dst.Env, *o.Env)
	}
}

func applyProviderOverlays(dst *Config, overlays map[string]providerOverlay) {
	if len(overlays) == 0 {
		return
	}
	if dst.Providers == nil {
		dst.Providers = make(map[string]ProviderConfig, len(overlays))
	}

	for name, overlay := range overlays {
		provider := dst.Providers[name]
		overlay.Apply(&provider)
		dst.Providers[name] = provider
	}
}

func applyMCPServerOverlays(base []MCPServer, overlays []mcpServerOverlay) []MCPServer {
	merged := cloneMCPServers(base)
	index := make(map[string]int, len(merged))
	for i, server := range merged {
		if server.Name == "" {
			continue
		}
		index[server.Name] = i
	}

	for _, overlay := range overlays {
		name := ""
		if overlay.Name != nil {
			name = strings.TrimSpace(*overlay.Name)
		}

		if idx, ok := index[name]; ok && name != "" {
			server := merged[idx]
			overlay.Apply(&server)
			merged[idx] = server
			continue
		}

		var server MCPServer
		overlay.Apply(&server)
		merged = append(merged, server)
		if server.Name != "" {
			index[server.Name] = len(merged) - 1
		}
	}

	return merged
}

func joinTOMLKeys(keys []toml.Key) string {
	if len(keys) == 0 {
		return ""
	}

	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, key.String())
	}
	sort.Strings(values)

	return strings.Join(values, ", ")
}
