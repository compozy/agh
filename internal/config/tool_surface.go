package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	hookspkg "github.com/pedronauck/agh/internal/hooks"
)

// Entry is one flattened, redacted effective config value.
type Entry struct {
	Path     string `json:"path"`
	Value    any    `json:"value"`
	Redacted bool   `json:"redacted"`
}

// DiffEntry describes one redacted effective config difference.
type DiffEntry struct {
	Path           string `json:"path"`
	Before         any    `json:"before,omitempty"`
	After          any    `json:"after,omitempty"`
	BeforeRedacted bool   `json:"before_redacted,omitempty"`
	AfterRedacted  bool   `json:"after_redacted,omitempty"`
}

// ValueKind identifies the TOML scalar shape supported by tool writes.
type ValueKind uint8

const (
	ConfigValueString ValueKind = iota
	ConfigValueBool
	ConfigValueInt
	ConfigValueInt64
	ConfigValueFloat
	ConfigValueDuration
	ConfigValueStringSlice
)

// PathDenial is the config package's path-policy decision.
type PathDenial string

const (
	ConfigPathAllowed         PathDenial = ""
	ConfigPathForbidden       PathDenial = "path_forbidden"
	ConfigPathSecretForbidden PathDenial = "secret_path_forbidden"
	ConfigPathTrustForbidden  PathDenial = "trust_root_forbidden"
)

// PathPolicy captures the deterministic decision for an agent-facing config path.
type PathPolicy struct {
	Segments []string
	Kind     ValueKind
	Redacted bool
	Denial   PathDenial
}

var (
	configToolDurationType = reflect.TypeFor[time.Duration]()

	agentMutableConfigKinds = map[string]ValueKind{
		"defaults.agent":                                    ConfigValueString,
		"defaults.provider":                                 ConfigValueString,
		"defaults.sandbox":                                  ConfigValueString,
		"agents.soul.enabled":                               ConfigValueBool,
		"agents.soul.max_body_bytes":                        ConfigValueInt64,
		"agents.soul.context_projection_bytes":              ConfigValueInt64,
		"agents.heartbeat.enabled":                          ConfigValueBool,
		"agents.heartbeat.max_body_bytes":                   ConfigValueInt64,
		"agents.heartbeat.context_projection_bytes":         ConfigValueInt64,
		"agents.heartbeat.min_interval":                     ConfigValueDuration,
		"agents.heartbeat.default_interval":                 ConfigValueDuration,
		"agents.heartbeat.wake_cooldown":                    ConfigValueDuration,
		"agents.heartbeat.max_wakes_per_cycle":              ConfigValueInt,
		"agents.heartbeat.active_session_only":              ConfigValueBool,
		"agents.heartbeat.allow_active_hours_preferences":   ConfigValueBool,
		"agents.heartbeat.wake_event_retention":             ConfigValueDuration,
		"agents.heartbeat.session_health_stale_after":       ConfigValueDuration,
		"agents.heartbeat.session_health_hook_min_interval": ConfigValueDuration,
		"limits.max_sessions":                               ConfigValueInt,
		"limits.max_concurrent_agents":                      ConfigValueInt,
		"session.limits.timeout":                            ConfigValueDuration,
		"session.supervision.activity_heartbeat_interval":   ConfigValueDuration,
		"session.supervision.progress_notify_interval":      ConfigValueDuration,
		"session.supervision.prompt_deadline":               ConfigValueDuration,
		"session.supervision.inactivity_warning_after":      ConfigValueDuration,
		"session.supervision.inactivity_timeout":            ConfigValueDuration,
		"session.supervision.timeout_cancel_grace":          ConfigValueDuration,
		"memory.enabled":                                    ConfigValueBool,
		"memory.dream.enabled":                              ConfigValueBool,
		"memory.dream.agent":                                ConfigValueString,
		"memory.dream.min_hours":                            ConfigValueFloat,
		"memory.dream.min_sessions":                         ConfigValueInt,
		"memory.dream.check_interval":                       ConfigValueDuration,
		"skills.enabled":                                    ConfigValueBool,
		"skills.disabled_skills":                            ConfigValueStringSlice,
		"skills.poll_interval":                              ConfigValueDuration,
		"automation.enabled":                                ConfigValueBool,
		"automation.timezone":                               ConfigValueString,
		"automation.max_concurrent_jobs":                    ConfigValueInt,
		"network.enabled":                                   ConfigValueBool,
		"network.default_channel":                           ConfigValueString,
		"network.max_payload":                               ConfigValueInt,
		"network.greet_interval":                            ConfigValueInt,
		"network.max_replay_age":                            ConfigValueInt,
		"network.max_queue_depth":                           ConfigValueInt,
		"tools.default_max_result_bytes":                    ConfigValueInt64,
	}
)

// RedactedConfigMap converts config to the same redacted map shape used by operator-facing CLI output.
func RedactedConfigMap(cfg *Config) map[string]any {
	node, ok := configNodeFromValue(reflect.ValueOf(cfg), "")
	if !ok {
		return map[string]any{}
	}
	values, ok := node.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return values
}

// FlattenConfigEntries returns deterministic flattened config entries.
func FlattenConfigEntries(configMap map[string]any) []Entry {
	entries := make([]Entry, 0)
	flattenConfigValue(&entries, "", configMap, false)
	sort.Slice(entries, func(i int, j int) bool {
		return entries[i].Path < entries[j].Path
	})
	return entries
}

// EntryByPath returns one flattened entry.
func EntryByPath(entries []Entry, path string) (Entry, bool) {
	trimmed := strings.TrimSpace(path)
	for _, entry := range entries {
		if entry.Path == trimmed {
			return entry, true
		}
	}
	return Entry{}, false
}

// DiffConfigEntries returns sorted redacted differences between two effective entry sets.
func DiffConfigEntries(before []Entry, after []Entry) []DiffEntry {
	beforeByPath := make(map[string]Entry, len(before))
	for _, entry := range before {
		beforeByPath[entry.Path] = entry
	}
	afterByPath := make(map[string]Entry, len(after))
	for _, entry := range after {
		afterByPath[entry.Path] = entry
	}
	paths := make(map[string]struct{}, len(beforeByPath)+len(afterByPath))
	for path := range beforeByPath {
		paths[path] = struct{}{}
	}
	for path := range afterByPath {
		paths[path] = struct{}{}
	}

	diff := make([]DiffEntry, 0)
	for path := range paths {
		left, hasLeft := beforeByPath[path]
		right, hasRight := afterByPath[path]
		if hasLeft && hasRight && reflect.DeepEqual(left.Value, right.Value) && left.Redacted == right.Redacted {
			continue
		}
		entry := DiffEntry{Path: path}
		if hasLeft {
			entry.Before = left.Value
			entry.BeforeRedacted = left.Redacted
		}
		if hasRight {
			entry.After = right.Value
			entry.AfterRedacted = right.Redacted
		}
		diff = append(diff, entry)
	}
	sort.Slice(diff, func(i int, j int) bool {
		return diff[i].Path < diff[j].Path
	})
	return diff
}

// ParseDottedConfigPath parses a user-facing dotted config path.
func ParseDottedConfigPath(raw string) ([]string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("config: config path is required")
	}
	parts := strings.Split(trimmed, ".")
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return nil, fmt.Errorf("config: config path %q contains an empty segment", trimmed)
		}
	}
	return parts, nil
}

// ClassifyToolConfigPath applies the agent-facing mutable config policy.
func ClassifyToolConfigPath(path []string) (PathPolicy, error) {
	clean, err := normalizeMutationPath(path)
	if err != nil {
		return PathPolicy{}, err
	}
	policy := PathPolicy{Segments: clean, Kind: ConfigValueString}
	if configPathIsSecret(clean) {
		policy.Denial = ConfigPathSecretForbidden
		return policy, nil
	}
	if configPathHasArraySyntax(clean) {
		policy.Denial = ConfigPathForbidden
		return policy, nil
	}
	if configPathIsTrustRoot(clean) {
		policy.Denial = ConfigPathTrustForbidden
		return policy, nil
	}
	joined := strings.Join(clean, ".")
	if kind, ok := agentMutableConfigKinds[joined]; ok {
		policy.Kind = kind
		return policy, nil
	}
	if len(clean) == 3 && clean[0] == "providers" {
		switch clean[2] {
		case "command",
			"default_model",
			"auth_mode",
			"env_policy",
			"home_policy",
			"auth_status_command",
			"auth_login_command":
			policy.Kind = ConfigValueString
			return policy, nil
		case "session_mcp":
			policy.Kind = ConfigValueBool
			return policy, nil
		}
	}
	policy.Denial = ConfigPathForbidden
	return policy, nil
}

// NormalizeToolConfigValue coerces a JSON-decoded tool value into a supported TOML value.
func NormalizeToolConfigValue(kind ValueKind, value any) (any, error) {
	switch kind {
	case ConfigValueString:
		typed, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("config: expected string value, got %T", value)
		}
		return typed, nil
	case ConfigValueBool:
		return coerceConfigBool(value)
	case ConfigValueInt:
		return coerceConfigInt(value)
	case ConfigValueInt64:
		return coerceConfigInt64(value)
	case ConfigValueFloat:
		return coerceConfigFloat(value)
	case ConfigValueDuration:
		raw, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("config: expected duration string, got %T", value)
		}
		trimmed := strings.TrimSpace(raw)
		if _, err := time.ParseDuration(trimmed); err != nil {
			return nil, fmt.Errorf("config: parse duration value %q: %w", raw, err)
		}
		return trimmed, nil
	case ConfigValueStringSlice:
		return coerceConfigStringSlice(value)
	default:
		return nil, fmt.Errorf("config: unsupported config value kind %d", kind)
	}
}

// OverlayHookDeclarations returns config-backed hook declarations from one overlay target.
func OverlayHookDeclarations(target WriteTarget) ([]hookspkg.HookDecl, error) {
	if !target.isConfigTarget() {
		return nil, fmt.Errorf("config: write target %q is not a config overlay", target.Kind())
	}
	overlay, err := loadConfigOverlayFile(target.path)
	if err != nil {
		return nil, err
	}
	decls := make([]hookspkg.HookDecl, 0, len(overlay.Hooks.Declarations))
	for idx := range overlay.Hooks.Declarations {
		raw := &overlay.Hooks.Declarations[idx]
		decl, err := raw.toHookDecl(hookspkg.HookSourceConfig, "")
		if err != nil {
			return nil, fmt.Errorf("hooks.declarations[%d]: %w", idx, err)
		}
		decls = append(decls, decl)
	}
	return decls, nil
}

// HookDeclarationOverlayValues converts a hook declaration to TOML overlay values.
func HookDeclarationOverlayValues(decl hookspkg.HookDecl) map[string]any {
	values := map[string]any{
		"event": string(decl.Event),
	}
	if decl.Enabled != nil {
		values["enabled"] = *decl.Enabled
	}
	if decl.Mode != "" {
		values["mode"] = string(decl.Mode)
	}
	if decl.Required {
		values["required"] = true
	}
	if decl.PrioritySet || decl.Priority != 0 {
		values["priority"] = decl.Priority
	}
	if decl.Timeout > 0 {
		values["timeout"] = decl.Timeout.String()
	}
	if matcher := hookMatcherOverlayValues(decl.Matcher); len(matcher) > 0 {
		values["matcher"] = matcher
	}
	if decl.Command != "" {
		values["command"] = decl.Command
	}
	if len(decl.Args) > 0 {
		values["args"] = append([]string(nil), decl.Args...)
	}
	if len(decl.Env) > 0 {
		values["env"] = mergeStringMaps(nil, decl.Env)
	}
	if decl.ExecutorKind != "" && decl.ExecutorKind != hookspkg.HookExecutorSubprocess {
		values["executor"] = map[string]any{"kind": string(decl.ExecutorKind)}
	}
	return values
}

func configNodeFromValue(value reflect.Value, fieldName string) (any, bool) {
	value, ok := indirectConfigValue(value)
	if !ok {
		return nil, false
	}
	if value.Type() == configToolDurationType {
		return time.Duration(value.Int()).String(), true
	}
	switch value.Kind() {
	case reflect.Struct:
		return configStructNode(value)
	case reflect.Map:
		return configMapNode(value, fieldName)
	case reflect.Slice, reflect.Array:
		return configSequenceNode(value, fieldName)
	default:
		return configScalarNode(value)
	}
}

func indirectConfigValue(value reflect.Value) (reflect.Value, bool) {
	if !value.IsValid() {
		return reflect.Value{}, false
	}
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return reflect.Value{}, false
		}
		value = value.Elem()
	}
	return value, true
}

func configStructNode(value reflect.Value) (any, bool) {
	result := make(map[string]any)
	valueType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := valueType.Field(i)
		if field.PkgPath != "" {
			continue
		}
		name, omitEmpty, ok := tomlFieldName(field)
		if !ok {
			continue
		}
		fieldValue := value.Field(i)
		if omitEmpty && fieldValue.IsZero() {
			continue
		}
		node, hasValue := configNodeFromValue(fieldValue, name)
		if hasValue {
			result[name] = node
		}
	}
	return result, true
}

func configMapNode(value reflect.Value, fieldName string) (any, bool) {
	if value.IsNil() {
		return map[string]any{}, true
	}
	result := make(map[string]any, value.Len())
	for _, key := range sortedReflectMapKeys(value) {
		mapKey := fmt.Sprint(key.Interface())
		if strings.EqualFold(fieldName, "env") || strings.EqualFold(fieldName, "secret_env") {
			result[mapKey] = RedactedValue()
			continue
		}
		node, hasValue := configNodeFromValue(value.MapIndex(key), "")
		if hasValue {
			result[mapKey] = node
		}
	}
	return result, true
}

func sortedReflectMapKeys(value reflect.Value) []reflect.Value {
	keys := value.MapKeys()
	sort.Slice(keys, func(i int, j int) bool {
		return fmt.Sprint(keys[i].Interface()) < fmt.Sprint(keys[j].Interface())
	})
	return keys
}

func configSequenceNode(value reflect.Value, fieldName string) (any, bool) {
	items := make([]any, 0, value.Len())
	for i := 0; i < value.Len(); i++ {
		node, hasValue := configNodeFromValue(value.Index(i), fieldName)
		if hasValue {
			items = append(items, node)
		}
	}
	return items, true
}

func configScalarNode(value reflect.Value) (any, bool) {
	switch value.Kind() {
	case reflect.String:
		return value.String(), true
	case reflect.Bool:
		return value.Bool(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int(), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return value.Uint(), true
	case reflect.Float32, reflect.Float64:
		return value.Float(), true
	default:
		if value.CanInterface() {
			return fmt.Sprint(value.Interface()), true
		}
		return nil, false
	}
}

func tomlFieldName(field reflect.StructField) (string, bool, bool) {
	tag := field.Tag.Get("toml")
	if tag == "-" {
		return "", false, false
	}
	if tag == "" {
		return strings.ToLower(field.Name), false, true
	}
	parts := strings.Split(tag, ",")
	name := strings.TrimSpace(parts[0])
	if name == "" {
		return "", false, false
	}
	omitEmpty := false
	for _, part := range parts[1:] {
		if strings.TrimSpace(part) == "omitempty" {
			omitEmpty = true
			break
		}
	}
	return name, omitEmpty, true
}

func flattenConfigValue(entries *[]Entry, path string, value any, redacted bool) {
	switch typed := value.(type) {
	case map[string]any:
		if len(typed) == 0 {
			if path != "" {
				*entries = append(*entries, Entry{Path: path, Value: map[string]any{}, Redacted: redacted})
			}
			return
		}
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			nextPath := key
			if path != "" {
				nextPath = path + "." + key
			}
			flattenConfigValue(entries, nextPath, typed[key], redacted || key == "env" || key == "secret_env")
		}
	case []any:
		if len(typed) == 0 {
			if path != "" {
				*entries = append(*entries, Entry{Path: path, Value: []any{}, Redacted: redacted})
			}
			return
		}
		for i, item := range typed {
			flattenConfigValue(entries, fmt.Sprintf("%s[%d]", path, i), item, redacted)
		}
	default:
		if path != "" {
			*entries = append(*entries, Entry{Path: path, Value: typed, Redacted: redacted})
		}
	}
}

func configPathHasArraySyntax(path []string) bool {
	for _, segment := range path {
		if strings.ContainsAny(segment, "[]") {
			return true
		}
	}
	return false
}

func configPathIsSecret(path []string) bool {
	for _, segment := range path {
		lower := strings.ToLower(strings.TrimSpace(segment))
		switch lower {
		case "env", "secret_env", "secret_ref", "client_secret_ref", "webhook_secret_ref":
			return true
		}
		if strings.Contains(lower, "secret") ||
			strings.Contains(lower, "token") ||
			strings.Contains(lower, "password") ||
			strings.Contains(lower, "authorization") {
			return true
		}
	}
	return false
}

func configPathIsTrustRoot(path []string) bool {
	if len(path) == 0 {
		return true
	}
	switch path[0] {
	case "daemon", string(MCPServerTransportHTTP), "permissions", "observability", "log",
		"mcp_servers", "sandboxes", "autonomy":
		return true
	case "hooks":
		return true
	case "providers":
		if len(path) >= 3 {
			switch path[2] {
			case "command", "mcp_servers":
				return true
			}
		}
	case "memory":
		return len(path) >= 2 && path[1] == "global_dir"
	case "network":
		return len(path) >= 2 && path[1] == "port"
	case "tools":
		if len(path) >= 2 {
			switch path[1] {
			case "enabled", "hosted_mcp_enabled", "hosted_mcp", "policy":
				return true
			}
		}
	case "skills":
		if len(path) >= 2 {
			switch path[1] {
			case "allowed_marketplace_mcp", "allowed_marketplace_hooks", "marketplace":
				return true
			}
		}
	case "extensions":
		return true
	}
	return false
}

func coerceConfigBool(value any) (bool, error) {
	switch typed := value.(type) {
	case bool:
		return typed, nil
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(typed))
		if err != nil {
			return false, fmt.Errorf("config: parse bool value %q: %w", typed, err)
		}
		return parsed, nil
	default:
		return false, fmt.Errorf("config: expected bool value, got %T", value)
	}
}

func coerceConfigInt(value any) (int, error) {
	int64Value, err := coerceConfigInt64(value)
	if err != nil {
		return 0, err
	}
	converted := int(int64Value)
	if int64(converted) != int64Value {
		return 0, fmt.Errorf("config: integer value %d overflows int", int64Value)
	}
	return converted, nil
}

func coerceConfigInt64(value any) (int64, error) {
	switch typed := value.(type) {
	case int:
		return int64(typed), nil
	case int8:
		return int64(typed), nil
	case int16:
		return int64(typed), nil
	case int32:
		return int64(typed), nil
	case int64:
		return typed, nil
	case float64:
		if math.Trunc(typed) != typed {
			return 0, fmt.Errorf("config: expected integer value, got %v", typed)
		}
		return int64(typed), nil
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("config: parse integer value %q: %w", typed, err)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("config: expected integer value, got %T", value)
	}
}

func coerceConfigFloat(value any) (float64, error) {
	switch typed := value.(type) {
	case float32:
		return float64(typed), nil
	case float64:
		return typed, nil
	case int:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0, fmt.Errorf("config: parse float value %q: %w", typed, err)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("config: expected float value, got %T", value)
	}
}

func coerceConfigStringSlice(value any) ([]string, error) {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...), nil
	case []any:
		values := make([]string, 0, len(typed))
		for idx, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("config: string array item %d has type %T", idx, item)
			}
			values = append(values, text)
		}
		return values, nil
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return []string{}, nil
		}
		if strings.HasPrefix(trimmed, "[") {
			var values []string
			if err := json.Unmarshal([]byte(trimmed), &values); err != nil {
				return nil, fmt.Errorf("config: parse string array %q: %w", typed, err)
			}
			return values, nil
		}
		parts := strings.Split(trimmed, ",")
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			if item := strings.TrimSpace(part); item != "" {
				values = append(values, item)
			}
		}
		return values, nil
	default:
		return nil, fmt.Errorf("config: expected string array value, got %T", value)
	}
}

func hookMatcherOverlayValues(matcher hookspkg.HookMatcher) map[string]any {
	values := map[string]any{}
	addString := func(key string, value string) {
		if strings.TrimSpace(value) != "" {
			values[key] = strings.TrimSpace(value)
		}
	}
	addString("agent_name", matcher.AgentName)
	addString("agent_type", matcher.AgentType)
	addString("workspace_id", matcher.WorkspaceID)
	addString("workspace_root", matcher.WorkspaceRoot)
	addString("session_type", matcher.SessionType)
	addString("input_class", matcher.InputClass)
	addString("acp_event_type", matcher.ACPEventType)
	addString("turn_id", matcher.TurnID)
	addString("tool_id", matcher.ToolID)
	addString("tool_name", matcher.ToolName)
	addString("decision_class", matcher.DecisionClass)
	addString("message_role", matcher.MessageRole)
	addString("message_delta_type", matcher.MessageDeltaType)
	if matcher.NetworkMatcher != nil {
		addString("channel", matcher.Channel)
		addString("surface", matcher.Surface)
		addString("kind", matcher.Kind)
		addString("direction", matcher.Direction)
		addString("work_state", matcher.WorkState)
	}
	if matcher.CompactionMatcher != nil {
		addString("compaction_reason", matcher.Reason)
		addString("compaction_strategy", matcher.Strategy)
	}
	if matcher.ToolReadOnly != nil {
		values["tool_read_only"] = *matcher.ToolReadOnly
	}
	return values
}
