// Package heartbeat resolves optional HEARTBEAT.md wake-policy artifacts.
package heartbeat

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/goccy/go-yaml"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/diagnostics"
	"github.com/pedronauck/agh/internal/frontmatter"
)

const (
	// FileName is the canonical authored wake-policy filename.
	FileName = "HEARTBEAT.md"

	schemaVersion      = 1
	digestPrefix       = "agh.heartbeat.v1\n"
	configDigestPrefix = "agh.heartbeat.config.v1\n"
	diagnosticError    = "error"
	diagnosticWarning  = "warning"
)

var (
	// ErrInvalid reports a HEARTBEAT.md file that exists but cannot be accepted.
	ErrInvalid = errors.New("heartbeat: invalid HEARTBEAT.md")
	// ErrPathEscape reports a HEARTBEAT.md path outside its configured workspace root.
	ErrPathEscape = errors.New("heartbeat: path escapes workspace root")
)

// ResolveRequest describes a HEARTBEAT.md file located beside an AGENT.md file.
type ResolveRequest struct {
	AgentPath     string
	WorkspaceRoot string
	Config        aghconfig.HeartbeatConfig
}

// ParseRequest describes in-memory HEARTBEAT.md content to validate and project.
type ParseRequest struct {
	SourcePath    string
	WorkspaceRoot string
	Content       []byte
	Config        aghconfig.HeartbeatConfig
}

// ResolvedPolicy is the normalized wake policy later runtime surfaces consume.
type ResolvedPolicy struct {
	Enabled          bool
	Present          bool
	Active           bool
	Valid            bool
	SourcePath       string
	Digest           string
	ConfigDigest     string
	SchemaVersion    int
	Summary          string
	GuidanceMarkdown string
	Frontmatter      Frontmatter
	Preferences      Preferences
	ConfigProvenance ConfigProvenance
	Prompt           PromptContribution
	Status           StatusData
	Diagnostics      []Diagnostic
}

// Frontmatter is the allowlisted strict HEARTBEAT.md metadata.
type Frontmatter struct {
	Version     int                    `json:"version"`
	Enabled     bool                   `json:"enabled"`
	Summary     string                 `json:"summary,omitempty"`
	Preferences FrontmatterPreferences `json:"preferences"`
	Context     ContextProjection      `json:"context"`
}

// FrontmatterPreferences captures authored preference hints before config bounds.
type FrontmatterPreferences struct {
	MinInterval  string       `json:"min_interval,omitempty"`
	ActiveHours  []TimeWindow `json:"active_hours,omitempty"`
	QuietWindows []TimeWindow `json:"quiet_windows,omitempty"`
}

// Preferences captures config-bound resolved wake-policy hints.
type Preferences struct {
	MinInterval  time.Duration     `json:"min_interval"`
	ActiveHours  []TimeWindow      `json:"active_hours,omitempty"`
	QuietWindows []TimeWindow      `json:"quiet_windows,omitempty"`
	Context      ContextProjection `json:"context"`
}

// ContextProjection captures authored context projection hints.
type ContextProjection struct {
	Include []string `json:"include,omitempty"`
}

// TimeWindow is one local wall-clock active or quiet window.
type TimeWindow struct {
	Timezone string `json:"timezone"`
	Start    string `json:"start"`
	End      string `json:"end"`
}

// ConfigSubset is the canonical config authority subset included in digests.
type ConfigSubset struct {
	Enabled                      bool   `json:"enabled"`
	MaxBodyBytes                 int64  `json:"max_body_bytes"`
	ContextProjectionBytes       int64  `json:"context_projection_bytes"`
	MinInterval                  string `json:"min_interval"`
	DefaultInterval              string `json:"default_interval"`
	WakeCooldown                 string `json:"wake_cooldown"`
	MaxWakesPerCycle             int    `json:"max_wakes_per_cycle"`
	ActiveSessionOnly            bool   `json:"active_session_only"`
	AllowActiveHoursPreferences  bool   `json:"allow_active_hours_preferences"`
	WakeEventRetention           string `json:"wake_event_retention"`
	SessionHealthStaleAfter      string `json:"session_health_stale_after"`
	SessionHealthHookMinInterval string `json:"session_health_hook_min_interval"`
}

// ConfigProvenance records the resolved config subset that shaped a policy.
type ConfigProvenance struct {
	Digest string       `json:"digest"`
	Subset ConfigSubset `json:"subset"`
}

// PromptContribution is the bounded synthetic-wake prompt contribution.
type PromptContribution struct {
	Active           bool              `json:"active"`
	Digest           string            `json:"digest,omitempty"`
	ConfigDigest     string            `json:"config_digest,omitempty"`
	SourcePath       string            `json:"source_path,omitempty"`
	Summary          string            `json:"summary,omitempty"`
	GuidanceMarkdown string            `json:"guidance_markdown,omitempty"`
	Preferences      Preferences       `json:"preferences"`
	Truncated        bool              `json:"truncated"`
	MaxBytes         int64             `json:"max_bytes"`
	MaxBodyBytes     int64             `json:"max_body_bytes"`
	Diagnostics      []Diagnostic      `json:"diagnostics,omitempty"`
	Context          ContextProjection `json:"context"`
}

// StatusData is the compact inspect/status read model for policy diagnostics.
type StatusData struct {
	Enabled                      bool               `json:"enabled"`
	Present                      bool               `json:"present"`
	Active                       bool               `json:"active"`
	Valid                        bool               `json:"valid"`
	SourcePath                   string             `json:"source_path,omitempty"`
	Digest                       string             `json:"digest,omitempty"`
	ConfigDigest                 string             `json:"config_digest,omitempty"`
	SchemaVersion                int                `json:"schema_version"`
	Summary                      string             `json:"summary,omitempty"`
	Preferences                  Preferences        `json:"preferences"`
	Diagnostics                  []Diagnostic       `json:"diagnostics,omitempty"`
	MaxBodyBytes                 int64              `json:"max_body_bytes"`
	ContextProjectionBytes       int64              `json:"context_projection_bytes"`
	ActiveSessionOnly            bool               `json:"active_session_only"`
	AllowActiveHoursPreferences  bool               `json:"allow_active_hours_preferences"`
	WakeCooldown                 time.Duration      `json:"wake_cooldown"`
	MaxWakesPerCycle             int                `json:"max_wakes_per_cycle"`
	WakeEventRetention           time.Duration      `json:"wake_event_retention"`
	SessionHealthStaleAfter      time.Duration      `json:"session_health_stale_after"`
	SessionHealthHookMinInterval time.Duration      `json:"session_health_hook_min_interval"`
	ConfigProvenance             ConfigProvenance   `json:"config_provenance"`
	Prompt                       PromptContribution `json:"prompt"`
}

// Diagnostic describes a closed, redacted HEARTBEAT.md validation problem.
type Diagnostic struct {
	Code       string `json:"code"`
	Severity   string `json:"severity"`
	Message    string `json:"message"`
	Field      string `json:"field,omitempty"`
	Section    string `json:"section,omitempty"`
	SourcePath string `json:"source_path,omitempty"`
	Line       int    `json:"line,omitempty"`
	Column     int    `json:"column,omitempty"`
}

// DiagnosticError carries structured diagnostics for invalid authored content.
type DiagnosticError struct {
	Diagnostics []Diagnostic
	cause       error
}

func (e *DiagnosticError) Error() string {
	if e == nil || len(e.Diagnostics) == 0 {
		return ErrInvalid.Error()
	}
	first := e.Diagnostics[0]
	return fmt.Sprintf("%s: %s", first.Code, first.Message)
}

// Unwrap exposes the validation sentinel for errors.Is callers.
func (e *DiagnosticError) Unwrap() error {
	if e == nil || e.cause == nil {
		return ErrInvalid
	}
	return e.cause
}

// Resolve reads and resolves HEARTBEAT.md beside the provided agent path.
func Resolve(ctx context.Context, req ResolveRequest) (ResolvedPolicy, error) {
	if err := req.Config.Validate(); err != nil {
		return ResolvedPolicy{}, err
	}
	if err := checkContext(ctx); err != nil {
		return ResolvedPolicy{}, err
	}
	provenance, err := ConfigProvenanceFor(req.Config)
	if err != nil {
		return ResolvedPolicy{}, err
	}

	heartbeatPath, pathErr := heartbeatPathForAgent(req.AgentPath)
	safePath, diagnostic := safeSourcePath(heartbeatPath, req.WorkspaceRoot)
	result := emptyResult(req.Config, provenance, safePath)
	if pathErr != nil {
		diag := diagnosticForError("heartbeat_invalid_source_path", safePath, pathErr, ErrInvalid)
		diag.Message = "HEARTBEAT.md source path is required"
		return resultWithDiagnostics(&result, []Diagnostic{diag})
	}
	if diagnostic != nil {
		return resultWithDiagnostics(&result, []Diagnostic{*diagnostic})
	}

	content, err := os.ReadFile(heartbeatPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return result, nil
		}
		diag := diagnosticForError("heartbeat_parser_io", safePath, err, err)
		diag.Message = "HEARTBEAT.md could not be read"
		return resultWithDiagnostics(&result, []Diagnostic{diag})
	}
	if err := checkContext(ctx); err != nil {
		return ResolvedPolicy{}, err
	}

	return Parse(ctx, ParseRequest{
		SourcePath:    heartbeatPath,
		WorkspaceRoot: req.WorkspaceRoot,
		Content:       content,
		Config:        req.Config,
	})
}

// Parse validates and normalizes in-memory HEARTBEAT.md content.
func Parse(ctx context.Context, req ParseRequest) (ResolvedPolicy, error) {
	if err := req.Config.Validate(); err != nil {
		return ResolvedPolicy{}, err
	}
	if err := checkContext(ctx); err != nil {
		return ResolvedPolicy{}, err
	}
	provenance, err := ConfigProvenanceFor(req.Config)
	if err != nil {
		return ResolvedPolicy{}, err
	}

	safePath, diagnostic := safeSourcePath(req.SourcePath, req.WorkspaceRoot)
	result := emptyResult(req.Config, provenance, safePath)
	if diagnostic != nil {
		return resultWithDiagnostics(&result, []Diagnostic{*diagnostic})
	}
	result.Present = true
	result.Status.Present = true

	front, body, bodyLineOffset, parseDiagnostics := parseDocument(req.Content, req.Config, safePath)
	if len(parseDiagnostics) > 0 {
		return resultWithDiagnostics(&result, parseDiagnostics)
	}

	preferences, preferenceDiagnostics := resolvePreferences(front, req.Config, safePath)
	if hasErrorDiagnostics(preferenceDiagnostics) {
		return resultWithDiagnostics(&result, preferenceDiagnostics)
	}
	bodyDiagnostics := validateBodyAuthorityClaims(body, safePath, bodyLineOffset)
	if len(bodyDiagnostics) > 0 {
		return resultWithDiagnostics(&result, append(preferenceDiagnostics, bodyDiagnostics...))
	}

	digest, err := digestPolicy(front, body, provenance.Subset)
	if err != nil {
		diag := diagnosticForError("heartbeat_digest_failed", safePath, err, ErrInvalid)
		diag.Message = "HEARTBEAT.md digest could not be computed"
		return resultWithDiagnostics(&result, []Diagnostic{diag})
	}

	active := req.Config.Enabled && front.Enabled
	result.Enabled = req.Config.Enabled
	result.Present = true
	result.Active = active
	result.Valid = true
	result.SourcePath = safePath
	result.Digest = digest
	result.ConfigDigest = provenance.Digest
	result.SchemaVersion = schemaVersion
	result.Summary = front.Summary
	result.GuidanceMarkdown = body
	result.Frontmatter = front
	result.Preferences = preferences
	result.Diagnostics = sanitizeDiagnostics(preferenceDiagnostics)
	result.Prompt = promptContribution(req.Config, &result)
	result.Status = statusData(req.Config, &result)
	return result, nil
}

// ConfigProvenanceFor returns the canonical config subset and digest used by Heartbeat.
func ConfigProvenanceFor(cfg aghconfig.HeartbeatConfig) (ConfigProvenance, error) {
	if err := cfg.Validate(); err != nil {
		return ConfigProvenance{}, err
	}
	subset := ConfigSubset{
		Enabled:                      cfg.Enabled,
		MaxBodyBytes:                 cfg.MaxBodyBytes,
		ContextProjectionBytes:       cfg.ContextProjectionBytes,
		MinInterval:                  cfg.MinInterval.String(),
		DefaultInterval:              cfg.DefaultInterval.String(),
		WakeCooldown:                 cfg.WakeCooldown.String(),
		MaxWakesPerCycle:             cfg.MaxWakesPerCycle,
		ActiveSessionOnly:            cfg.ActiveSessionOnly,
		AllowActiveHoursPreferences:  cfg.AllowActiveHoursPreferences,
		WakeEventRetention:           cfg.WakeEventRetention.String(),
		SessionHealthStaleAfter:      cfg.SessionHealthStaleAfter.String(),
		SessionHealthHookMinInterval: cfg.SessionHealthHookMinInterval.String(),
	}
	encoded, err := json.Marshal(subset)
	if err != nil {
		return ConfigProvenance{}, fmt.Errorf("marshal heartbeat config subset: %w", err)
	}
	sum := sha256.Sum256([]byte(configDigestPrefix + string(encoded)))
	return ConfigProvenance{
		Digest: "sha256:" + hex.EncodeToString(sum[:]),
		Subset: subset,
	}, nil
}

// AllowsAt evaluates active-hours allow windows minus quiet-window deny windows.
func (p Preferences) AllowsAt(now time.Time) (bool, error) {
	activeAllowed := len(p.ActiveHours) == 0
	for _, window := range p.ActiveHours {
		contains, err := window.Contains(now)
		if err != nil {
			return false, err
		}
		if contains {
			activeAllowed = true
			break
		}
	}
	if !activeAllowed {
		return false, nil
	}
	for _, window := range p.QuietWindows {
		contains, err := window.Contains(now)
		if err != nil {
			return false, err
		}
		if contains {
			return false, nil
		}
	}
	return true, nil
}

// Contains reports whether now falls inside this local wall-clock window.
func (w TimeWindow) Contains(now time.Time) (bool, error) {
	location, err := time.LoadLocation(strings.TrimSpace(w.Timezone))
	if err != nil {
		return false, fmt.Errorf("load heartbeat time window timezone %q: %w", w.Timezone, err)
	}
	start, err := parseClock(w.Start)
	if err != nil {
		return false, err
	}
	end, err := parseClock(w.End)
	if err != nil {
		return false, err
	}
	local := now.In(location)
	current := local.Hour()*60 + local.Minute()
	startMinutes := start.Hour()*60 + start.Minute()
	endMinutes := end.Hour()*60 + end.Minute()
	if endMinutes <= startMinutes {
		return current >= startMinutes || current < endMinutes, nil
	}
	return current >= startMinutes && current < endMinutes, nil
}

func parseDocument(
	content []byte,
	cfg aghconfig.HeartbeatConfig,
	sourcePath string,
) (Frontmatter, string, int, []Diagnostic) {
	normalized := normalizeLineEndings(content)
	front := defaultFrontmatter()
	parts, err := frontmatter.Split(normalized)
	if err != nil {
		if errors.Is(err, frontmatter.ErrMissing) {
			body := normalizeBody(string(normalized))
			if int64(len([]byte(body))) > cfg.MaxBodyBytes {
				message := fmt.Sprintf(
					"HEARTBEAT.md body exceeds agents.heartbeat.max_body_bytes (%d)",
					cfg.MaxBodyBytes,
				)
				return Frontmatter{}, "", 1, []Diagnostic{{
					Code:       "heartbeat_oversized_body",
					Severity:   diagnosticError,
					Message:    message,
					SourcePath: sourcePath,
					Line:       1,
					Column:     1,
				}}
			}
			return front, body, 1, nil
		}
		diag := diagnosticForError("heartbeat_malformed_frontmatter", sourcePath, err, ErrInvalid)
		diag.Message = "HEARTBEAT.md frontmatter is unterminated or malformed"
		diag.Line = 1
		diag.Column = 1
		return Frontmatter{}, "", 1, []Diagnostic{diag}
	}

	if int64(len(parts.Metadata)) > cfg.MaxBodyBytes {
		message := fmt.Sprintf(
			"HEARTBEAT.md frontmatter exceeds agents.heartbeat.max_body_bytes (%d)",
			cfg.MaxBodyBytes,
		)
		return Frontmatter{}, "", 1, []Diagnostic{{
			Code:       "heartbeat_oversized_frontmatter",
			Severity:   diagnosticError,
			Message:    message,
			SourcePath: sourcePath,
			Line:       2,
			Column:     1,
		}}
	}
	body := normalizeBody(parts.Body)
	if int64(len([]byte(body))) > cfg.MaxBodyBytes {
		message := fmt.Sprintf(
			"HEARTBEAT.md body exceeds agents.heartbeat.max_body_bytes (%d)",
			cfg.MaxBodyBytes,
		)
		return Frontmatter{}, "", 1, []Diagnostic{{
			Code:       "heartbeat_oversized_body",
			Severity:   diagnosticError,
			Message:    message,
			SourcePath: sourcePath,
			Line:       bodyStartLine(parts.Metadata),
			Column:     1,
		}}
	}

	parsedFront, diagnosticsList := parseFrontmatter(parts.Metadata, sourcePath)
	if len(diagnosticsList) > 0 {
		return Frontmatter{}, "", 1, diagnosticsList
	}
	return parsedFront, body, bodyStartLine(parts.Metadata), nil
}

func parseFrontmatter(metadata []byte, sourcePath string) (Frontmatter, []Diagnostic) {
	front := defaultFrontmatter()
	if strings.TrimSpace(string(metadata)) == "" {
		return front, nil
	}

	var raw map[string]any
	if err := yaml.UnmarshalWithOptions(metadata, &raw, yaml.Strict()); err != nil {
		diag := diagnosticForError("heartbeat_malformed_frontmatter", sourcePath, err, ErrInvalid)
		diag.Message = "HEARTBEAT.md frontmatter is not valid YAML"
		diag.Line = 2
		diag.Column = 1
		return Frontmatter{}, []Diagnostic{diag}
	}

	diagnosticsList := make([]Diagnostic, 0)
	keys := sortedKeys(raw)
	for _, key := range keys {
		value := raw[key]
		locationLine, locationColumn := frontmatterKeyLocation(metadata, key)
		if !isAllowedField(key) {
			diagnosticsList = append(diagnosticsList, unsupportedFieldDiagnostic(
				sourcePath,
				key,
				key,
				locationLine,
				locationColumn,
			))
			continue
		}
		if err := assignAllowedField(&front, key, value); err != nil {
			diagnosticsList = append(diagnosticsList, Diagnostic{
				Code:       "heartbeat_invalid_field_type",
				Severity:   diagnosticError,
				Field:      key,
				Message:    diagnostics.Redact(err.Error()),
				SourcePath: sourcePath,
				Line:       locationLine,
				Column:     locationColumn,
			})
		}
	}
	if len(diagnosticsList) > 0 {
		return Frontmatter{}, diagnosticsList
	}
	return front, nil
}

func assignAllowedField(front *Frontmatter, key string, value any) error {
	switch key {
	case "version":
		version, err := scalarInt(value)
		if err != nil {
			return fmt.Errorf("HEARTBEAT.md frontmatter field %q must be version number 1", key)
		}
		if version != schemaVersion {
			return fmt.Errorf("HEARTBEAT.md frontmatter field %q must be version 1", key)
		}
		front.Version = version
	case "enabled":
		enabled, err := boolOnly(value)
		if err != nil {
			return fmt.Errorf("HEARTBEAT.md frontmatter field %q must be a boolean", key)
		}
		front.Enabled = enabled
	case "summary":
		summary, err := stringOnly(value)
		if err != nil {
			return fmt.Errorf("HEARTBEAT.md frontmatter field %q must be a string", key)
		}
		front.Summary = summary
	case "preferences":
		preferences, err := parsePreferences(value)
		if err != nil {
			return err
		}
		front.Preferences = preferences
	case "context":
		contextProjection, err := parseContextProjection(value)
		if err != nil {
			return err
		}
		front.Context = contextProjection
	}
	return nil
}

func parsePreferences(value any) (FrontmatterPreferences, error) {
	raw, ok := value.(map[string]any)
	if !ok {
		return FrontmatterPreferences{}, errors.New("HEARTBEAT.md frontmatter field \"preferences\" must be a mapping")
	}
	var parsed FrontmatterPreferences
	for _, key := range sortedKeys(raw) {
		switch key {
		case "min_interval":
			minInterval, err := stringOnly(raw[key])
			if err != nil {
				return FrontmatterPreferences{}, errors.New(
					"HEARTBEAT.md frontmatter field \"preferences.min_interval\" must be a duration string",
				)
			}
			parsed.MinInterval = minInterval
		case "active_hours":
			windows, err := parseTimeWindows(raw[key], "preferences.active_hours")
			if err != nil {
				return FrontmatterPreferences{}, err
			}
			parsed.ActiveHours = windows
		case "quiet_windows":
			windows, err := parseTimeWindows(raw[key], "preferences.quiet_windows")
			if err != nil {
				return FrontmatterPreferences{}, err
			}
			parsed.QuietWindows = windows
		default:
			if owner := forbiddenOwner(key); owner != "" {
				return FrontmatterPreferences{}, fmt.Errorf(
					"HEARTBEAT.md frontmatter field %q belongs to %s",
					"preferences."+key,
					owner,
				)
			}
			return FrontmatterPreferences{}, fmt.Errorf(
				"HEARTBEAT.md frontmatter field %q is not supported",
				"preferences."+key,
			)
		}
	}
	return parsed, nil
}

func parseContextProjection(value any) (ContextProjection, error) {
	raw, ok := value.(map[string]any)
	if !ok {
		return ContextProjection{}, errors.New("HEARTBEAT.md frontmatter field \"context\" must be a mapping")
	}
	var projection ContextProjection
	for _, key := range sortedKeys(raw) {
		switch key {
		case "include":
			values, err := stringList(raw[key], "context.include")
			if err != nil {
				return ContextProjection{}, err
			}
			projection.Include = values
		default:
			if owner := forbiddenOwner(key); owner != "" {
				return ContextProjection{}, fmt.Errorf(
					"HEARTBEAT.md frontmatter field %q belongs to %s",
					"context."+key,
					owner,
				)
			}
			return ContextProjection{}, fmt.Errorf(
				"HEARTBEAT.md frontmatter field %q is not supported",
				"context."+key,
			)
		}
	}
	return projection, nil
}

func parseTimeWindows(value any, field string) ([]TimeWindow, error) {
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("HEARTBEAT.md frontmatter field %q must be a list", field)
	}
	windows := make([]TimeWindow, 0, len(items))
	for idx, item := range items {
		raw, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("HEARTBEAT.md frontmatter field %q[%d] must be a mapping", field, idx)
		}
		var window TimeWindow
		for _, key := range sortedKeys(raw) {
			value := raw[key]
			text, err := stringOnly(value)
			if err != nil {
				return nil, fmt.Errorf("HEARTBEAT.md frontmatter field %q[%d].%s must be a string", field, idx, key)
			}
			switch key {
			case "timezone":
				window.Timezone = text
			case "start":
				window.Start = text
			case "end":
				window.End = text
			default:
				return nil, fmt.Errorf("HEARTBEAT.md frontmatter field %q[%d].%s is not supported", field, idx, key)
			}
		}
		windows = append(windows, window)
	}
	return windows, nil
}

func resolvePreferences(
	front Frontmatter,
	cfg aghconfig.HeartbeatConfig,
	sourcePath string,
) (Preferences, []Diagnostic) {
	preferences := Preferences{
		MinInterval: cfg.DefaultInterval,
		Context:     front.Context,
	}
	diagnosticsList := make([]Diagnostic, 0)
	if strings.TrimSpace(front.Preferences.MinInterval) != "" {
		interval, err := time.ParseDuration(strings.TrimSpace(front.Preferences.MinInterval))
		if err != nil {
			return Preferences{}, []Diagnostic{{
				Code:       "heartbeat_invalid_min_interval",
				Severity:   diagnosticError,
				Field:      "preferences.min_interval",
				Message:    diagnostics.RedactAndBound(fmt.Sprintf("parse min_interval: %v", err), 300),
				SourcePath: sourcePath,
				Line:       1,
				Column:     1,
			}}
		}
		preferences.MinInterval = interval
	}
	if preferences.MinInterval < cfg.MinInterval {
		diagnosticsList = append(diagnosticsList, Diagnostic{
			Code:     "heartbeat_preference_clamped",
			Severity: diagnosticWarning,
			Field:    "preferences.min_interval",
			Message: fmt.Sprintf(
				"HEARTBEAT.md min_interval was clamped to agents.heartbeat.min_interval (%s)",
				cfg.MinInterval,
			),
			SourcePath: sourcePath,
		})
		preferences.MinInterval = cfg.MinInterval
	}

	hasTimePreferences := len(front.Preferences.ActiveHours) > 0 || len(front.Preferences.QuietWindows) > 0
	if hasTimePreferences && !cfg.AllowActiveHoursPreferences {
		diagnosticsList = append(diagnosticsList, Diagnostic{
			Code:       "heartbeat_preference_ignored",
			Severity:   diagnosticWarning,
			Field:      "preferences.active_hours",
			Message:    "HEARTBEAT.md active-hours preferences are disabled by agents.heartbeat.allow_active_hours_preferences",
			SourcePath: sourcePath,
		})
		return preferences, diagnosticsList
	}

	activeHours, activeDiagnostics := validateTimeWindows(
		front.Preferences.ActiveHours,
		"preferences.active_hours",
		sourcePath,
	)
	quietWindows, quietDiagnostics := validateTimeWindows(
		front.Preferences.QuietWindows,
		"preferences.quiet_windows",
		sourcePath,
	)
	diagnosticsList = append(diagnosticsList, activeDiagnostics...)
	diagnosticsList = append(diagnosticsList, quietDiagnostics...)
	if hasErrorDiagnostics(diagnosticsList) {
		return Preferences{}, diagnosticsList
	}
	preferences.ActiveHours = activeHours
	preferences.QuietWindows = quietWindows
	return preferences, diagnosticsList
}

func validateTimeWindows(windows []TimeWindow, field string, sourcePath string) ([]TimeWindow, []Diagnostic) {
	validated := make([]TimeWindow, 0, len(windows))
	diagnosticsList := make([]Diagnostic, 0)
	for idx, window := range windows {
		normalized := TimeWindow{
			Timezone: strings.TrimSpace(window.Timezone),
			Start:    strings.TrimSpace(window.Start),
			End:      strings.TrimSpace(window.End),
		}
		if normalized.Timezone == "" {
			diagnosticsList = append(diagnosticsList, timeWindowDiagnostic(
				"heartbeat_invalid_timezone",
				field,
				idx,
				"time window timezone is required",
				sourcePath,
			))
			continue
		}
		if _, err := time.LoadLocation(normalized.Timezone); err != nil {
			diagnosticsList = append(diagnosticsList, timeWindowDiagnostic(
				"heartbeat_invalid_timezone",
				field,
				idx,
				fmt.Sprintf("time window timezone %q is not an IANA timezone", normalized.Timezone),
				sourcePath,
			))
			continue
		}
		if _, err := parseClock(normalized.Start); err != nil {
			diagnosticsList = append(diagnosticsList, timeWindowDiagnostic(
				"heartbeat_invalid_time_window",
				field,
				idx,
				fmt.Sprintf("time window start %q must use HH:MM", normalized.Start),
				sourcePath,
			))
			continue
		}
		if _, err := parseClock(normalized.End); err != nil {
			diagnosticsList = append(diagnosticsList, timeWindowDiagnostic(
				"heartbeat_invalid_time_window",
				field,
				idx,
				fmt.Sprintf("time window end %q must use HH:MM", normalized.End),
				sourcePath,
			))
			continue
		}
		validated = append(validated, normalized)
	}
	return validated, diagnosticsList
}

func validateBodyAuthorityClaims(body string, sourcePath string, bodyLineOffset int) []Diagnostic {
	lines := strings.Split(body, "\n")
	diagnosticsList := make([]Diagnostic, 0)
	for idx, line := range lines {
		if section, ok := markdownHeadingKey(line); ok {
			if owner := forbiddenOwner(section); owner != "" {
				message := fmt.Sprintf("HEARTBEAT.md section %q belongs to %s", section, owner)
				diagnosticsList = append(diagnosticsList, Diagnostic{
					Code:       "heartbeat_reserved_section",
					Severity:   diagnosticError,
					Section:    section,
					Message:    diagnostics.Redact(message),
					SourcePath: sourcePath,
					Line:       bodyLineOffset + idx,
					Column:     1,
				})
			}
			continue
		}
		field, ok := bodyDeclarationKey(line)
		if !ok {
			continue
		}
		if owner := forbiddenOwner(field); owner != "" {
			diagnosticsList = append(diagnosticsList, Diagnostic{
				Code:       "heartbeat_reserved_body_field",
				Severity:   diagnosticError,
				Field:      field,
				Message:    diagnostics.Redact(fmt.Sprintf("HEARTBEAT.md body field %q belongs to %s", field, owner)),
				SourcePath: sourcePath,
				Line:       bodyLineOffset + idx,
				Column:     1,
			})
		}
	}
	return diagnosticsList
}

func digestPolicy(front Frontmatter, body string, subset ConfigSubset) (string, error) {
	canonicalFront, err := json.Marshal(front)
	if err != nil {
		return "", fmt.Errorf("marshal canonical heartbeat frontmatter: %w", err)
	}
	canonicalConfig, err := json.Marshal(subset)
	if err != nil {
		return "", fmt.Errorf("marshal canonical heartbeat config subset: %w", err)
	}
	sum := sha256.Sum256([]byte(digestPrefix + string(canonicalFront) + "\n" + body + "\n" + string(canonicalConfig)))
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func promptContribution(cfg aghconfig.HeartbeatConfig, policy *ResolvedPolicy) PromptContribution {
	contribution := PromptContribution{
		Active:           policy.Active,
		Digest:           policy.Digest,
		ConfigDigest:     policy.ConfigDigest,
		SourcePath:       policy.SourcePath,
		Summary:          policy.Summary,
		GuidanceMarkdown: policy.GuidanceMarkdown,
		Preferences:      policy.Preferences,
		MaxBytes:         cfg.ContextProjectionBytes,
		MaxBodyBytes:     cfg.MaxBodyBytes,
		Diagnostics:      cloneDiagnostics(policy.Diagnostics),
		Context:          policy.Preferences.Context,
	}
	if projectionWithinBudget(contribution, cfg.ContextProjectionBytes) {
		return contribution
	}
	contribution.Truncated = true
	contribution.GuidanceMarkdown = truncateUTF8(contribution.GuidanceMarkdown, int(cfg.ContextProjectionBytes/2))
	for len(contribution.Diagnostics) > 0 && !projectionWithinBudget(contribution, cfg.ContextProjectionBytes) {
		contribution.Diagnostics = contribution.Diagnostics[:len(contribution.Diagnostics)-1]
	}
	for len(contribution.Context.Include) > 0 && !projectionWithinBudget(contribution, cfg.ContextProjectionBytes) {
		contribution.Context.Include = contribution.Context.Include[:len(contribution.Context.Include)-1]
		contribution.Preferences.Context = contribution.Context
	}
	if !projectionWithinBudget(contribution, cfg.ContextProjectionBytes) {
		contribution.Summary = truncateUTF8(contribution.Summary, int(cfg.ContextProjectionBytes/4))
	}
	if !projectionWithinBudget(contribution, cfg.ContextProjectionBytes) {
		contribution.GuidanceMarkdown = ""
	}
	return contribution
}

func projectionWithinBudget(contribution PromptContribution, maxBytes int64) bool {
	if maxBytes <= 0 {
		return false
	}
	data, err := json.Marshal(contribution)
	if err != nil {
		return false
	}
	return int64(len(data)) <= maxBytes
}

func statusData(cfg aghconfig.HeartbeatConfig, policy *ResolvedPolicy) StatusData {
	return StatusData{
		Enabled:                      cfg.Enabled,
		Present:                      policy.Present,
		Active:                       policy.Active,
		Valid:                        policy.Valid,
		SourcePath:                   policy.SourcePath,
		Digest:                       policy.Digest,
		ConfigDigest:                 policy.ConfigDigest,
		SchemaVersion:                schemaVersion,
		Summary:                      policy.Summary,
		Preferences:                  policy.Preferences,
		Diagnostics:                  cloneDiagnostics(policy.Diagnostics),
		MaxBodyBytes:                 cfg.MaxBodyBytes,
		ContextProjectionBytes:       cfg.ContextProjectionBytes,
		ActiveSessionOnly:            cfg.ActiveSessionOnly,
		AllowActiveHoursPreferences:  cfg.AllowActiveHoursPreferences,
		WakeCooldown:                 cfg.WakeCooldown,
		MaxWakesPerCycle:             cfg.MaxWakesPerCycle,
		WakeEventRetention:           cfg.WakeEventRetention,
		SessionHealthStaleAfter:      cfg.SessionHealthStaleAfter,
		SessionHealthHookMinInterval: cfg.SessionHealthHookMinInterval,
		ConfigProvenance:             policy.ConfigProvenance,
		Prompt:                       policy.Prompt,
	}
}

func resultWithDiagnostics(result *ResolvedPolicy, list []Diagnostic) (ResolvedPolicy, error) {
	if result == nil {
		result = &ResolvedPolicy{}
	}
	result.Valid = false
	result.Active = false
	result.Diagnostics = sanitizeDiagnostics(list)
	result.Prompt.Active = false
	result.Prompt.Diagnostics = cloneDiagnostics(result.Diagnostics)
	result.Status.Valid = false
	result.Status.Active = false
	result.Status.Diagnostics = cloneDiagnostics(result.Diagnostics)
	result.Status.Prompt = result.Prompt
	return *result, &DiagnosticError{Diagnostics: cloneDiagnostics(result.Diagnostics), cause: ErrInvalid}
}

func emptyResult(cfg aghconfig.HeartbeatConfig, provenance ConfigProvenance, sourcePath string) ResolvedPolicy {
	preferences := Preferences{MinInterval: cfg.DefaultInterval}
	prompt := PromptContribution{
		Active:       false,
		ConfigDigest: provenance.Digest,
		SourcePath:   sourcePath,
		Preferences:  preferences,
		MaxBytes:     cfg.ContextProjectionBytes,
		MaxBodyBytes: cfg.MaxBodyBytes,
	}
	status := StatusData{
		Enabled:                      cfg.Enabled,
		Present:                      false,
		Active:                       false,
		Valid:                        true,
		SourcePath:                   sourcePath,
		ConfigDigest:                 provenance.Digest,
		SchemaVersion:                schemaVersion,
		Preferences:                  preferences,
		MaxBodyBytes:                 cfg.MaxBodyBytes,
		ContextProjectionBytes:       cfg.ContextProjectionBytes,
		ActiveSessionOnly:            cfg.ActiveSessionOnly,
		AllowActiveHoursPreferences:  cfg.AllowActiveHoursPreferences,
		WakeCooldown:                 cfg.WakeCooldown,
		MaxWakesPerCycle:             cfg.MaxWakesPerCycle,
		WakeEventRetention:           cfg.WakeEventRetention,
		SessionHealthStaleAfter:      cfg.SessionHealthStaleAfter,
		SessionHealthHookMinInterval: cfg.SessionHealthHookMinInterval,
		ConfigProvenance:             provenance,
		Prompt:                       prompt,
	}
	return ResolvedPolicy{
		Enabled:          cfg.Enabled,
		Present:          false,
		Active:           false,
		Valid:            true,
		SourcePath:       sourcePath,
		ConfigDigest:     provenance.Digest,
		SchemaVersion:    schemaVersion,
		Preferences:      preferences,
		ConfigProvenance: provenance,
		Prompt:           prompt,
		Status:           status,
	}
}

func defaultFrontmatter() Frontmatter {
	return Frontmatter{
		Version: schemaVersion,
		Enabled: true,
	}
}

func diagnosticForError(code string, sourcePath string, err error, cause error) Diagnostic {
	message := ""
	if err != nil {
		message = diagnostics.RedactAndBound(err.Error(), 300)
	}
	if strings.TrimSpace(message) == "" && cause != nil {
		message = cause.Error()
	}
	return Diagnostic{
		Code:       code,
		Severity:   diagnosticError,
		Message:    message,
		SourcePath: sourcePath,
	}
}

func unsupportedFieldDiagnostic(
	sourcePath string,
	field string,
	key string,
	line int,
	column int,
) Diagnostic {
	code := "heartbeat_unsupported_field"
	message := fmt.Sprintf("HEARTBEAT.md frontmatter field %q is not supported", field)
	if owner := forbiddenOwner(key); owner != "" {
		code = "heartbeat_forbidden_field"
		message = fmt.Sprintf("HEARTBEAT.md frontmatter field %q belongs to %s", field, owner)
	}
	return Diagnostic{
		Code:       code,
		Severity:   diagnosticError,
		Field:      field,
		Message:    diagnostics.Redact(message),
		SourcePath: sourcePath,
		Line:       line,
		Column:     column,
	}
}

func timeWindowDiagnostic(code string, field string, idx int, message string, sourcePath string) Diagnostic {
	return Diagnostic{
		Code:       code,
		Severity:   diagnosticError,
		Field:      fmt.Sprintf("%s[%d]", field, idx),
		Message:    diagnostics.Redact(message),
		SourcePath: sourcePath,
	}
}

func sanitizeDiagnostics(list []Diagnostic) []Diagnostic {
	if len(list) == 0 {
		return nil
	}
	sanitized := make([]Diagnostic, 0, len(list))
	for _, diag := range list {
		diag.Code = strings.TrimSpace(diag.Code)
		diag.Severity = strings.TrimSpace(diag.Severity)
		if diag.Severity == "" {
			diag.Severity = diagnosticError
		}
		diag.Field = strings.TrimSpace(diag.Field)
		diag.Section = strings.TrimSpace(diag.Section)
		diag.Message = diagnostics.RedactAndBound(diag.Message, 300)
		diag.SourcePath = safePathWithoutRoot(diag.SourcePath)
		sanitized = append(sanitized, diag)
	}
	return sanitized
}

func cloneDiagnostics(list []Diagnostic) []Diagnostic {
	if len(list) == 0 {
		return nil
	}
	cloned := make([]Diagnostic, len(list))
	copy(cloned, list)
	return cloned
}

func hasErrorDiagnostics(list []Diagnostic) bool {
	for _, diag := range list {
		if diag.Severity == "" || diag.Severity == diagnosticError {
			return true
		}
	}
	return false
}

func safeSourcePath(sourcePath string, workspaceRoot string) (string, *Diagnostic) {
	trimmed := strings.TrimSpace(sourcePath)
	if trimmed == "" {
		return "", nil
	}
	if strings.ContainsRune(trimmed, 0) {
		return FileName, &Diagnostic{
			Code:       "heartbeat_path_escape",
			Severity:   diagnosticError,
			Message:    "HEARTBEAT.md path contains an invalid NUL byte",
			SourcePath: FileName,
		}
	}

	cleanSource := filepath.Clean(trimmed)
	if strings.TrimSpace(workspaceRoot) == "" {
		return safePathWithoutRoot(cleanSource), nil
	}

	absRoot, err := filepath.Abs(filepath.Clean(workspaceRoot))
	if err != nil {
		return safePathWithoutRoot(cleanSource), &Diagnostic{
			Code:       "heartbeat_path_escape",
			Severity:   diagnosticError,
			Message:    diagnostics.RedactAndBound(fmt.Sprintf("resolve workspace root: %v", err), 300),
			SourcePath: safePathWithoutRoot(cleanSource),
		}
	}
	sourceForRoot := cleanSource
	if !filepath.IsAbs(sourceForRoot) {
		sourceForRoot = filepath.Join(absRoot, sourceForRoot)
	}
	absSource, err := filepath.Abs(sourceForRoot)
	if err != nil {
		return safePathWithoutRoot(cleanSource), &Diagnostic{
			Code:       "heartbeat_path_escape",
			Severity:   diagnosticError,
			Message:    diagnostics.RedactAndBound(fmt.Sprintf("resolve HEARTBEAT.md path: %v", err), 300),
			SourcePath: safePathWithoutRoot(cleanSource),
		}
	}

	safePath, within := relativePathWithinRoot(absRoot, absSource)
	if !within {
		return safePath, &Diagnostic{
			Code:       "heartbeat_path_escape",
			Severity:   diagnosticError,
			Message:    "HEARTBEAT.md path must stay inside the workspace root",
			SourcePath: safePath,
		}
	}
	if resolvedRoot, rootErr := filepath.EvalSymlinks(absRoot); rootErr == nil {
		if resolvedSource, sourceErr := filepath.EvalSymlinks(absSource); sourceErr == nil {
			safeResolved, resolvedWithin := relativePathWithinRoot(resolvedRoot, resolvedSource)
			if !resolvedWithin {
				return safePath, &Diagnostic{
					Code:       "heartbeat_path_escape",
					Severity:   diagnosticError,
					Message:    "HEARTBEAT.md symlink target must stay inside the workspace root",
					SourcePath: safeResolved,
				}
			}
		}
	}
	return safePath, nil
}

func relativePathWithinRoot(root string, target string) (string, bool) {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return safePathWithoutRoot(target), false
	}
	if rel == "." {
		return ".", true
	}
	if strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." || filepath.IsAbs(rel) {
		return safePathWithoutRoot(target), false
	}
	return filepath.ToSlash(rel), true
}

func safePathWithoutRoot(path string) string {
	slashed := filepath.ToSlash(filepath.Clean(path))
	parts := strings.Split(slashed, "/")
	for idx := 0; idx < len(parts)-2; idx++ {
		if parts[idx] == ".agh" && parts[idx+1] == "agents" {
			return strings.Join(parts[idx:], "/")
		}
	}
	if len(parts) >= 2 {
		return strings.Join(parts[len(parts)-2:], "/")
	}
	if slashed == "." {
		return FileName
	}
	return strings.TrimPrefix(slashed, "/")
}

func heartbeatPathForAgent(agentPath string) (string, error) {
	trimmed := strings.TrimSpace(agentPath)
	if trimmed == "" {
		return "", errors.New("agent path is required")
	}
	cleaned := filepath.Clean(trimmed)
	if strings.EqualFold(filepath.Base(cleaned), "AGENT.md") {
		return filepath.Join(filepath.Dir(cleaned), FileName), nil
	}
	return filepath.Join(cleaned, FileName), nil
}

func isAllowedField(key string) bool {
	switch key {
	case "version", "enabled", "summary", "preferences", "context":
		return true
	default:
		return false
	}
}

func forbiddenOwner(key string) string {
	switch normalizeKey(key) {
	case "session", "session_health", "session_liveness", "liveness", "health",
		"supervision", "session_supervision", "activity", "activity_heartbeat",
		"activity_heartbeat_interval", "inactivity_warning_after", "inactivity_timeout":
		return "[session.supervision] and daemon session health"
	case "scheduler", "schedule", "schedules", "cadence", "interval", "every",
		"default_interval", "sweep", "wake_loop", "run_loop", "loop", "cron":
		return "scheduler config"
	case "task", "tasks", "task_runs", "task_run", "claim_next_run", "claimnextrun",
		"claim", "claim_token", "claim_token_hash", "ownership", "owner", "queue", "queues":
		return "task runtime and ClaimNextRun"
	case "lease", "leases", "lease_duration", "task_lease", "lease_heartbeat",
		"heartbeat_run_lease", "heartbeatrunlease", "heartbeat_at":
		return "task lease heartbeat"
	case "network", "greet", "greet_interval", "presence", "peer_presence", "peers",
		"channels", "channel":
		return "AGH Network greet presence"
	case "provider", "providers", "model", "command", "tools", "toolsets", "deny_tools",
		"permissions", "capabilities", "capability", "hooks", "mcp_servers", "env", "config":
		return "agent definition or runtime config"
	default:
		return ""
	}
}

func markdownHeadingKey(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "#") {
		return "", false
	}
	level := 0
	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}
	if level == 0 || level >= len(trimmed) || !unicode.IsSpace(rune(trimmed[level])) {
		return "", false
	}
	title := strings.TrimSpace(trimmed[level:])
	title = strings.Trim(title, "# ")
	if title == "" {
		return "", false
	}
	return normalizeKey(title), true
}

func bodyDeclarationKey(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimLeft(trimmed, "-*+0123456789. \t")
	before, _, hasColon := strings.Cut(trimmed, ":")
	if !hasColon {
		var hasEquals bool
		before, _, hasEquals = strings.Cut(trimmed, "=")
		if !hasEquals {
			return "", false
		}
	}
	key := normalizeKey(before)
	if key == "" {
		return "", false
	}
	return key, true
}

func normalizeKey(value string) string {
	var builder strings.Builder
	lastUnderscore := false
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.Trim(builder.String(), "_")
}

func sortedKeys(raw map[string]any) []string {
	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func frontmatterKeyLocation(metadata []byte, key string) (int, int) {
	lines := strings.Split(string(metadata), "\n")
	for idx, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		col := len(line) - len(trimmed) + 1
		before, _, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		if strings.TrimSpace(before) == key {
			return idx + 2, col
		}
	}
	return 2, 1
}

func bodyStartLine(metadata []byte) int {
	if len(metadata) == 0 {
		return 3
	}
	return 3 + strings.Count(string(metadata), "\n")
}

func normalizeLineEndings(content []byte) []byte {
	replaced := strings.ReplaceAll(string(content), "\r\n", "\n")
	replaced = strings.ReplaceAll(replaced, "\r", "\n")
	return []byte(replaced)
}

func normalizeBody(body string) string {
	normalized := strings.ReplaceAll(body, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	return strings.TrimRightFunc(normalized, unicode.IsSpace)
}

func scalarInt(value any) (int, error) {
	switch typed := value.(type) {
	case int:
		return typed, nil
	case int64:
		return int(typed), nil
	case uint64:
		if typed > uint64(^uint(0)>>1) {
			return 0, errors.New("integer overflows int")
		}
		return int(typed), nil
	case float64:
		if typed != float64(int(typed)) {
			return 0, errors.New("number must be an integer")
		}
		return int(typed), nil
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil {
			return 0, fmt.Errorf("parse integer string: %w", err)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("expected integer, got %T", value)
	}
}

func boolOnly(value any) (bool, error) {
	typed, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("expected bool, got %T", value)
	}
	return typed, nil
}

func stringOnly(value any) (string, error) {
	typed, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("expected string, got %T", value)
	}
	return strings.TrimSpace(typed), nil
}

func stringList(value any, field string) ([]string, error) {
	values, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("HEARTBEAT.md frontmatter field %q must be a list of strings", field)
	}
	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for idx, item := range values {
		text, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("HEARTBEAT.md frontmatter field %q[%d] must be a string", field, idx)
		}
		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized, nil
}

func parseClock(value string) (time.Time, error) {
	parsed, err := time.Parse("15:04", strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, fmt.Errorf("parse HH:MM clock %q: %w", value, err)
	}
	return parsed, nil
}

func truncateUTF8(value string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(value) <= maxBytes {
		return value
	}
	var builder strings.Builder
	for _, r := range value {
		if builder.Len()+len(string(r)) > maxBytes {
			break
		}
		builder.WriteRune(r)
	}
	return strings.TrimSpace(builder.String())
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
