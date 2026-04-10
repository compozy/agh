// Package extension loads and validates declarative extension manifests.
package extension

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/pedronauck/agh/internal/version"
)

const (
	manifestTOMLFileName = "extension.toml"
	manifestJSONFileName = "extension.json"
)

var (
	// ErrManifestNotFound reports that an extension directory does not contain
	// either supported manifest file.
	ErrManifestNotFound = errors.New("extension: manifest not found")
	// ErrManifestInvalid reports that the manifest schema or content is invalid.
	ErrManifestInvalid = errors.New("extension: invalid manifest")
	// ErrManifestIncompatible reports that the manifest requires a newer daemon
	// version than the current build provides.
	ErrManifestIncompatible = errors.New("extension: incompatible manifest")
)

// Manifest describes one extension without executing any extension code.
type Manifest struct {
	Name          string             `toml:"name" json:"name"`
	Version       string             `toml:"version" json:"version"`
	Description   string             `toml:"description,omitempty" json:"description,omitempty"`
	MinAGHVersion string             `toml:"min_agh_version" json:"min_agh_version"`
	Resources     ResourcesConfig    `toml:"resources" json:"resources"`
	Capabilities  CapabilitiesConfig `toml:"capabilities" json:"capabilities"`
	Actions       ActionsConfig      `toml:"actions" json:"actions"`
	Subprocess    SubprocessConfig   `toml:"subprocess" json:"subprocess"`
	Security      SecurityConfig     `toml:"security" json:"security"`
}

// ResourcesConfig declares static assets bundled with an extension.
type ResourcesConfig struct {
	Skills     []string                   `toml:"skills,omitempty" json:"skills,omitempty"`
	Agents     []string                   `toml:"agents,omitempty" json:"agents,omitempty"`
	Hooks      []HookConfig               `toml:"hooks,omitempty" json:"hooks,omitempty"`
	MCPServers map[string]MCPServerConfig `toml:"mcp_servers,omitempty" json:"mcp_servers,omitempty"`
}

// CapabilitiesConfig declares the runtime interfaces the extension provides.
type CapabilitiesConfig struct {
	Provides []string `toml:"provides,omitempty" json:"provides,omitempty"`
}

// ActionsConfig declares Host API methods the extension wants to call.
type ActionsConfig struct {
	Requires []string `toml:"requires,omitempty" json:"requires,omitempty"`
}

// SubprocessConfig describes how to launch and monitor the extension process.
type SubprocessConfig struct {
	Command             string            `toml:"command,omitempty" json:"command,omitempty"`
	Args                []string          `toml:"args,omitempty" json:"args,omitempty"`
	Env                 map[string]string `toml:"env,omitempty" json:"env,omitempty"`
	HealthCheckInterval Duration          `toml:"health_check_interval,omitempty" json:"health_check_interval,omitempty"`
	ShutdownTimeout     Duration          `toml:"shutdown_timeout,omitempty" json:"shutdown_timeout,omitempty"`
}

// SecurityConfig declares the security grants the extension requests.
type SecurityConfig struct {
	Capabilities []string `toml:"capabilities,omitempty" json:"capabilities,omitempty"`
}

// HookConfig mirrors the hook declaration shape accepted from extension manifests.
type HookConfig struct {
	Name     string             `toml:"name" json:"name"`
	Event    string             `toml:"event" json:"event"`
	Mode     string             `toml:"mode,omitempty" json:"mode,omitempty"`
	Required bool               `toml:"required,omitempty" json:"required,omitempty"`
	Priority *int               `toml:"priority,omitempty" json:"priority,omitempty"`
	Timeout  Duration           `toml:"timeout,omitempty" json:"timeout,omitempty"`
	Matcher  HookMatcherConfig  `toml:"matcher,omitempty" json:"matcher,omitempty"`
	Command  string             `toml:"command,omitempty" json:"command,omitempty"`
	Args     []string           `toml:"args,omitempty" json:"args,omitempty"`
	Env      map[string]string  `toml:"env,omitempty" json:"env,omitempty"`
	Executor HookExecutorConfig `toml:"executor,omitempty" json:"executor,omitempty"`
}

// HookExecutorConfig selects the hook execution boundary and command.
type HookExecutorConfig struct {
	Kind    string            `toml:"kind,omitempty" json:"kind,omitempty"`
	Command string            `toml:"command,omitempty" json:"command,omitempty"`
	Args    []string          `toml:"args,omitempty" json:"args,omitempty"`
	Env     map[string]string `toml:"env,omitempty" json:"env,omitempty"`
}

// HookMatcherConfig narrows when a hook is eligible to run.
type HookMatcherConfig struct {
	AgentName          string `toml:"agent_name,omitempty" json:"agent_name,omitempty"`
	AgentType          string `toml:"agent_type,omitempty" json:"agent_type,omitempty"`
	WorkspaceID        string `toml:"workspace_id,omitempty" json:"workspace_id,omitempty"`
	WorkspaceRoot      string `toml:"workspace_root,omitempty" json:"workspace_root,omitempty"`
	SessionType        string `toml:"session_type,omitempty" json:"session_type,omitempty"`
	InputClass         string `toml:"input_class,omitempty" json:"input_class,omitempty"`
	ACPEventType       string `toml:"acp_event_type,omitempty" json:"acp_event_type,omitempty"`
	TurnID             string `toml:"turn_id,omitempty" json:"turn_id,omitempty"`
	ToolName           string `toml:"tool_name,omitempty" json:"tool_name,omitempty"`
	ToolNamespace      string `toml:"tool_namespace,omitempty" json:"tool_namespace,omitempty"`
	ToolReadOnly       *bool  `toml:"tool_read_only,omitempty" json:"tool_read_only,omitempty"`
	DecisionClass      string `toml:"decision_class,omitempty" json:"decision_class,omitempty"`
	MessageRole        string `toml:"message_role,omitempty" json:"message_role,omitempty"`
	MessageDeltaType   string `toml:"message_delta_type,omitempty" json:"message_delta_type,omitempty"`
	CompactionReason   string `toml:"compaction_reason,omitempty" json:"compaction_reason,omitempty"`
	CompactionStrategy string `toml:"compaction_strategy,omitempty" json:"compaction_strategy,omitempty"`
}

// MCPServerConfig declares one MCP server bundled by the extension.
type MCPServerConfig struct {
	Command string            `toml:"command" json:"command"`
	Args    []string          `toml:"args,omitempty" json:"args,omitempty"`
	Env     map[string]string `toml:"env,omitempty" json:"env,omitempty"`
}

// Duration stores time.Duration values while decoding TOML strings and JSON
// strings consistently.
type Duration time.Duration

// ManifestNotFoundError describes a missing manifest directory.
type ManifestNotFoundError struct {
	Dir   string
	Paths []string
}

// ManifestValidationError describes an invalid manifest field.
type ManifestValidationError struct {
	Field   string
	Value   string
	Message string
}

// ManifestCompatibilityError describes a daemon-version compatibility failure.
type ManifestCompatibilityError struct {
	CurrentVersion string
	MinVersion     string
}

type manifestDocument struct {
	Extension     manifestCore       `toml:"extension" json:"extension"`
	Name          string             `toml:"name" json:"name"`
	Version       string             `toml:"version" json:"version"`
	Description   string             `toml:"description,omitempty" json:"description,omitempty"`
	MinAGHVersion string             `toml:"min_agh_version" json:"min_agh_version"`
	Resources     ResourcesConfig    `toml:"resources" json:"resources"`
	Capabilities  CapabilitiesConfig `toml:"capabilities" json:"capabilities"`
	Actions       ActionsConfig      `toml:"actions" json:"actions"`
	Subprocess    SubprocessConfig   `toml:"subprocess" json:"subprocess"`
	Security      SecurityConfig     `toml:"security" json:"security"`
}

type manifestCore struct {
	Name          string `toml:"name" json:"name"`
	Version       string `toml:"version" json:"version"`
	Description   string `toml:"description,omitempty" json:"description,omitempty"`
	MinAGHVersion string `toml:"min_agh_version" json:"min_agh_version"`
}

// LoadManifest reads one extension manifest from dir, preferring TOML over JSON.
func LoadManifest(dir string) (*Manifest, error) {
	manifestDir := strings.TrimSpace(dir)
	if manifestDir == "" {
		return nil, &ManifestValidationError{
			Field:   "dir",
			Message: "directory is required",
		}
	}

	tomlPath := filepath.Join(manifestDir, manifestTOMLFileName)
	if exists, err := fileExists(tomlPath); err != nil {
		return nil, fmt.Errorf("extension: stat %q: %w", tomlPath, err)
	} else if exists {
		return loadManifestTOML(tomlPath)
	}

	jsonPath := filepath.Join(manifestDir, manifestJSONFileName)
	if exists, err := fileExists(jsonPath); err != nil {
		return nil, fmt.Errorf("extension: stat %q: %w", jsonPath, err)
	} else if exists {
		return loadManifestJSON(jsonPath)
	}

	return nil, &ManifestNotFoundError{
		Dir:   manifestDir,
		Paths: []string{tomlPath, jsonPath},
	}
}

// Validate checks the manifest schema and daemon compatibility.
func (m *Manifest) Validate() error {
	if err := requireField("name", m.Name); err != nil {
		return err
	}
	if err := requireField("version", m.Version); err != nil {
		return err
	}
	if err := validateSemanticVersionField("version", m.Version); err != nil {
		return err
	}
	if err := requireField("min_agh_version", m.MinAGHVersion); err != nil {
		return err
	}
	if err := validateSemanticVersionField("min_agh_version", m.MinAGHVersion); err != nil {
		return err
	}
	if err := validateDaemonCompatibility(m.MinAGHVersion); err != nil {
		return err
	}
	if err := validateDottedIdentifiers("capabilities.provides", m.Capabilities.Provides, false); err != nil {
		return err
	}
	if err := validateSlashIdentifiers("actions.requires", m.Actions.Requires); err != nil {
		return err
	}
	if err := validateDottedIdentifiers("security.capabilities", m.Security.Capabilities, true); err != nil {
		return err
	}
	return nil
}

// Error returns the typed missing-manifest message.
func (e *ManifestNotFoundError) Error() string {
	if len(e.Paths) == 0 {
		return fmt.Sprintf("%s in %q", ErrManifestNotFound, e.Dir)
	}
	return fmt.Sprintf("%s in %q (tried %s)", ErrManifestNotFound, e.Dir, strings.Join(e.Paths, ", "))
}

// Is matches sentinel errors for errors.Is.
func (e *ManifestNotFoundError) Is(target error) bool {
	return target == ErrManifestNotFound
}

// Error returns the field-specific validation message.
func (e *ManifestValidationError) Error() string {
	base := fmt.Sprintf("%s field %q", ErrManifestInvalid, e.Field)
	if strings.TrimSpace(e.Value) != "" {
		base = fmt.Sprintf("%s (value %q)", base, e.Value)
	}
	if strings.TrimSpace(e.Message) == "" {
		return base
	}
	return fmt.Sprintf("%s: %s", base, e.Message)
}

// Is matches sentinel errors for errors.Is.
func (e *ManifestValidationError) Is(target error) bool {
	return target == ErrManifestInvalid
}

// Error returns the daemon-version compatibility message.
func (e *ManifestCompatibilityError) Error() string {
	return fmt.Sprintf("%s: current daemon version %q does not satisfy min_agh_version %q", ErrManifestIncompatible, e.CurrentVersion, e.MinVersion)
}

// Is matches sentinel errors for errors.Is.
func (e *ManifestCompatibilityError) Is(target error) bool {
	return target == ErrManifestIncompatible
}

// IsZero reports whether the duration is unset.
func (d Duration) IsZero() bool {
	return time.Duration(d) == 0
}

// String returns the canonical duration string.
func (d Duration) String() string {
	return time.Duration(d).String()
}

// MarshalJSON emits the duration as a quoted duration string.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// MarshalText emits the duration as text.
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

// UnmarshalJSON accepts duration strings and integer nanoseconds.
func (d *Duration) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*d = 0
		return nil
	}

	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		return d.UnmarshalText([]byte(text))
	}

	var nanos int64
	if err := json.Unmarshal(data, &nanos); err == nil {
		*d = Duration(time.Duration(nanos))
		return nil
	}

	return fmt.Errorf("extension: invalid duration %s", string(data))
}

// UnmarshalText parses duration strings like "30s".
func (d *Duration) UnmarshalText(text []byte) error {
	trimmed := strings.TrimSpace(string(text))
	if trimmed == "" {
		*d = 0
		return nil
	}

	parsed, err := time.ParseDuration(trimmed)
	if err != nil {
		return fmt.Errorf("extension: parse duration %q: %w", trimmed, err)
	}
	*d = Duration(parsed)
	return nil
}

func fileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if info.IsDir() {
		return false, fmt.Errorf("%q is a directory", path)
	}
	return true, nil
}

func loadManifestTOML(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("extension: read manifest %q: %w", path, err)
	}

	var doc manifestDocument
	if _, err := toml.Decode(string(data), &doc); err != nil {
		return nil, fmt.Errorf("extension: decode manifest %q: %w", path, err)
	}

	manifest, err := doc.toManifest()
	if err != nil {
		return nil, err
	}
	if err := manifest.Validate(); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func loadManifestJSON(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("extension: read manifest %q: %w", path, err)
	}

	var doc manifestDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("extension: decode manifest %q: %w", path, err)
	}

	manifest, err := doc.toManifest()
	if err != nil {
		return nil, err
	}
	if err := manifest.Validate(); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func (d manifestDocument) toManifest() (Manifest, error) {
	name, err := mergeManifestValue("name", d.Name, d.Extension.Name)
	if err != nil {
		return Manifest{}, err
	}
	versionValue, err := mergeManifestValue("version", d.Version, d.Extension.Version)
	if err != nil {
		return Manifest{}, err
	}
	description, err := mergeManifestValue("description", d.Description, d.Extension.Description)
	if err != nil {
		return Manifest{}, err
	}
	minVersion, err := mergeManifestValue("min_agh_version", d.MinAGHVersion, d.Extension.MinAGHVersion)
	if err != nil {
		return Manifest{}, err
	}

	manifest := Manifest{
		Name:          name,
		Version:       versionValue,
		Description:   description,
		MinAGHVersion: minVersion,
		Resources:     normalizeResourcesConfig(d.Resources),
		Capabilities:  normalizeCapabilitiesConfig(d.Capabilities),
		Actions:       normalizeActionsConfig(d.Actions),
		Subprocess:    normalizeSubprocessConfig(d.Subprocess),
		Security:      normalizeSecurityConfig(d.Security),
	}
	return manifest, nil
}

func mergeManifestValue(field, rootValue, wrappedValue string) (string, error) {
	rootValue = strings.TrimSpace(rootValue)
	wrappedValue = strings.TrimSpace(wrappedValue)
	switch {
	case rootValue == "":
		return wrappedValue, nil
	case wrappedValue == "":
		return rootValue, nil
	case rootValue == wrappedValue:
		return rootValue, nil
	default:
		return "", &ManifestValidationError{
			Field:   field,
			Message: "conflicting root and extension values",
		}
	}
}

func normalizeResourcesConfig(cfg ResourcesConfig) ResourcesConfig {
	return ResourcesConfig{
		Skills:     normalizeStrings(cfg.Skills),
		Agents:     normalizeStrings(cfg.Agents),
		Hooks:      normalizeHooks(cfg.Hooks),
		MCPServers: normalizeMCPServers(cfg.MCPServers),
	}
}

func normalizeCapabilitiesConfig(cfg CapabilitiesConfig) CapabilitiesConfig {
	return CapabilitiesConfig{
		Provides: normalizeStrings(cfg.Provides),
	}
}

func normalizeActionsConfig(cfg ActionsConfig) ActionsConfig {
	return ActionsConfig{
		Requires: normalizeStrings(cfg.Requires),
	}
}

func normalizeSubprocessConfig(cfg SubprocessConfig) SubprocessConfig {
	return SubprocessConfig{
		Command:             strings.TrimSpace(cfg.Command),
		Args:                normalizeStrings(cfg.Args),
		Env:                 normalizeStringMap(cfg.Env),
		HealthCheckInterval: cfg.HealthCheckInterval,
		ShutdownTimeout:     cfg.ShutdownTimeout,
	}
}

func normalizeSecurityConfig(cfg SecurityConfig) SecurityConfig {
	return SecurityConfig{
		Capabilities: normalizeStrings(cfg.Capabilities),
	}
}

func normalizeHooks(src []HookConfig) []HookConfig {
	if len(src) == 0 {
		return nil
	}

	dst := make([]HookConfig, 0, len(src))
	for _, hook := range src {
		dst = append(dst, HookConfig{
			Name:     strings.TrimSpace(hook.Name),
			Event:    strings.TrimSpace(hook.Event),
			Mode:     strings.TrimSpace(hook.Mode),
			Required: hook.Required,
			Priority: cloneIntPointer(hook.Priority),
			Timeout:  hook.Timeout,
			Matcher: HookMatcherConfig{
				AgentName:          strings.TrimSpace(hook.Matcher.AgentName),
				AgentType:          strings.TrimSpace(hook.Matcher.AgentType),
				WorkspaceID:        strings.TrimSpace(hook.Matcher.WorkspaceID),
				WorkspaceRoot:      strings.TrimSpace(hook.Matcher.WorkspaceRoot),
				SessionType:        strings.TrimSpace(hook.Matcher.SessionType),
				InputClass:         strings.TrimSpace(hook.Matcher.InputClass),
				ACPEventType:       strings.TrimSpace(hook.Matcher.ACPEventType),
				TurnID:             strings.TrimSpace(hook.Matcher.TurnID),
				ToolName:           strings.TrimSpace(hook.Matcher.ToolName),
				ToolNamespace:      strings.TrimSpace(hook.Matcher.ToolNamespace),
				ToolReadOnly:       cloneBoolPointer(hook.Matcher.ToolReadOnly),
				DecisionClass:      strings.TrimSpace(hook.Matcher.DecisionClass),
				MessageRole:        strings.TrimSpace(hook.Matcher.MessageRole),
				MessageDeltaType:   strings.TrimSpace(hook.Matcher.MessageDeltaType),
				CompactionReason:   strings.TrimSpace(hook.Matcher.CompactionReason),
				CompactionStrategy: strings.TrimSpace(hook.Matcher.CompactionStrategy),
			},
			Command: strings.TrimSpace(hook.Command),
			Args:    normalizeStrings(hook.Args),
			Env:     normalizeStringMap(hook.Env),
			Executor: HookExecutorConfig{
				Kind:    strings.TrimSpace(hook.Executor.Kind),
				Command: strings.TrimSpace(hook.Executor.Command),
				Args:    normalizeStrings(hook.Executor.Args),
				Env:     normalizeStringMap(hook.Executor.Env),
			},
		})
	}

	return dst
}

func normalizeMCPServers(src map[string]MCPServerConfig) map[string]MCPServerConfig {
	if len(src) == 0 {
		return nil
	}

	dst := make(map[string]MCPServerConfig, len(src))
	for _, name := range sortedMapKeys(src) {
		trimmedName := strings.TrimSpace(name)
		if trimmedName == "" {
			continue
		}

		server := src[name]
		dst[trimmedName] = MCPServerConfig{
			Command: strings.TrimSpace(server.Command),
			Args:    normalizeStrings(server.Args),
			Env:     normalizeStringMap(server.Env),
		}
	}
	if len(dst) == 0 {
		return nil
	}
	return dst
}

func normalizeStrings(src []string) []string {
	if len(src) == 0 {
		return nil
	}

	dst := make([]string, 0, len(src))
	for _, value := range src {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		dst = append(dst, trimmed)
	}
	if len(dst) == 0 {
		return nil
	}
	return dst
}

func normalizeStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}

	dst := make(map[string]string, len(src))
	for _, key := range sortedMapKeys(src) {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		dst[trimmedKey] = strings.TrimSpace(src[key])
	}
	if len(dst) == 0 {
		return nil
	}
	return dst
}

func sortedMapKeys[V any](src map[string]V) []string {
	keys := make([]string, 0, len(src))
	for key := range src {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneBoolPointer(value *bool) *bool {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func requireField(field, value string) error {
	if strings.TrimSpace(value) != "" {
		return nil
	}
	return &ManifestValidationError{
		Field:   field,
		Message: "value is required",
	}
}

func validateSemanticVersionField(field, value string) error {
	if _, ok := parseSemanticVersion(value); ok {
		return nil
	}
	return &ManifestValidationError{
		Field:   field,
		Value:   value,
		Message: "must be a semantic version",
	}
}

func validateDaemonCompatibility(minVersion string) error {
	current := version.Current().Version
	currentVersion, ok := parseSemanticVersion(current)
	if !ok {
		return nil
	}

	requiredVersion, ok := parseSemanticVersion(minVersion)
	if !ok {
		return &ManifestValidationError{
			Field:   "min_agh_version",
			Value:   minVersion,
			Message: "must be a semantic version",
		}
	}

	if compareSemanticVersions(currentVersion, requiredVersion) >= 0 {
		return nil
	}

	return &ManifestCompatibilityError{
		CurrentVersion: current,
		MinVersion:     strings.TrimSpace(minVersion),
	}
}

func validateDottedIdentifiers(field string, values []string, allowWildcards bool) error {
	for idx, value := range values {
		if err := validateSeparatedIdentifier(value, ".", allowWildcards); err != nil {
			return &ManifestValidationError{
				Field:   fmt.Sprintf("%s[%d]", field, idx),
				Value:   value,
				Message: err.Error(),
			}
		}
	}
	return nil
}

func validateSlashIdentifiers(field string, values []string) error {
	for idx, value := range values {
		if err := validateSeparatedIdentifier(value, "/", false); err != nil {
			return &ManifestValidationError{
				Field:   fmt.Sprintf("%s[%d]", field, idx),
				Value:   value,
				Message: err.Error(),
			}
		}
	}
	return nil
}

func validateSeparatedIdentifier(value, separator string, allowWildcards bool) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return errors.New("value is required")
	}
	if allowWildcards && trimmed == "*" {
		return nil
	}

	parts := strings.Split(trimmed, separator)
	if len(parts) < 2 {
		return fmt.Errorf("must use %q-separated identifiers", separator)
	}

	for _, part := range parts {
		if allowWildcards && part == "*" {
			continue
		}
		if !validIdentifierPart(part) {
			return fmt.Errorf("contains invalid identifier segment %q", part)
		}
	}
	return nil
}

func validIdentifierPart(part string) bool {
	if part == "" {
		return false
	}

	for idx, r := range part {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
			if idx == 0 {
				return false
			}
		case r == '_' || r == '-':
			if idx == 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

type semanticVersion struct {
	core       [3]int
	prerelease []prereleaseIdentifier
}

type prereleaseIdentifier struct {
	raw     string
	number  int
	numeric bool
}

func parseSemanticVersion(value string) (semanticVersion, bool) {
	trimmed := normalizeSemanticVersion(value)
	if trimmed == "" {
		return semanticVersion{}, false
	}

	coreAndPrerelease, buildMetadata, hasBuild := strings.Cut(trimmed, "+")
	if hasBuild && !validIdentifierList(buildMetadata, true) {
		return semanticVersion{}, false
	}

	corePart, prereleasePart, hasPrerelease := strings.Cut(coreAndPrerelease, "-")
	core, ok := parseSemanticCore(corePart)
	if !ok {
		return semanticVersion{}, false
	}

	parsed := semanticVersion{core: core}
	if !hasPrerelease {
		return parsed, true
	}

	identifiers, ok := parsePrereleaseIdentifiers(prereleasePart)
	if !ok {
		return semanticVersion{}, false
	}
	parsed.prerelease = identifiers
	return parsed, true
}

func normalizeSemanticVersion(value string) string {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.TrimPrefix(trimmed, "v")
	trimmed = strings.TrimPrefix(trimmed, "V")
	return trimmed
}

func parseSemanticCore(value string) ([3]int, bool) {
	segments := strings.Split(value, ".")
	if len(segments) != 3 {
		return [3]int{}, false
	}

	var core [3]int
	for idx, segment := range segments {
		number, ok := parseNumericVersionPart(segment)
		if !ok {
			return [3]int{}, false
		}
		core[idx] = number
	}
	return core, true
}

func parseNumericVersionPart(value string) (int, bool) {
	if value == "" {
		return 0, false
	}
	if len(value) > 1 && strings.HasPrefix(value, "0") {
		return 0, false
	}
	number, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return number, true
}

func parsePrereleaseIdentifiers(value string) ([]prereleaseIdentifier, bool) {
	if !validIdentifierList(value, false) {
		return nil, false
	}

	segments := strings.Split(value, ".")
	identifiers := make([]prereleaseIdentifier, 0, len(segments))
	for _, segment := range segments {
		number, numeric := parseNumericVersionPart(segment)
		if !numeric && !validPrereleasePart(segment) {
			return nil, false
		}
		identifiers = append(identifiers, prereleaseIdentifier{
			raw:     segment,
			number:  number,
			numeric: numeric,
		})
	}
	return identifiers, true
}

func validIdentifierList(value string, allowLeadingZeroNumeric bool) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}

	for _, segment := range strings.Split(value, ".") {
		if segment == "" {
			return false
		}
		if number, err := strconv.Atoi(segment); err == nil {
			if !allowLeadingZeroNumeric && len(segment) > 1 && strings.HasPrefix(segment, "0") {
				return false
			}
			if number < 0 {
				return false
			}
			continue
		}
		if !validPrereleasePart(segment) {
			return false
		}
	}
	return true
}

func validPrereleasePart(value string) bool {
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-':
		default:
			return false
		}
	}
	return value != ""
}

func compareSemanticVersions(current, required semanticVersion) int {
	for idx := range current.core {
		switch {
		case current.core[idx] < required.core[idx]:
			return -1
		case current.core[idx] > required.core[idx]:
			return 1
		}
	}

	switch {
	case len(current.prerelease) == 0 && len(required.prerelease) == 0:
		return 0
	case len(current.prerelease) == 0:
		return 1
	case len(required.prerelease) == 0:
		return -1
	default:
		return comparePrerelease(current.prerelease, required.prerelease)
	}
}

func comparePrerelease(current, required []prereleaseIdentifier) int {
	limit := len(current)
	if len(required) > limit {
		limit = len(required)
	}

	for idx := 0; idx < limit; idx++ {
		switch {
		case idx >= len(current):
			return -1
		case idx >= len(required):
			return 1
		}

		left := current[idx]
		right := required[idx]
		switch {
		case left.numeric && right.numeric:
			switch {
			case left.number < right.number:
				return -1
			case left.number > right.number:
				return 1
			}
		case left.numeric:
			return -1
		case right.numeric:
			return 1
		default:
			switch {
			case left.raw < right.raw:
				return -1
			case left.raw > right.raw:
				return 1
			}
		}
	}

	return 0
}
