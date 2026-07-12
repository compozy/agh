package bridges

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

const (
	defaultProgressPreviewLimit = 160
	defaultProgressEditInterval = 1500 * time.Millisecond
	progressPlatformTelegram    = "telegram"
)

// ProgressMode controls which tool lifecycle events become bridge presentation events.
type ProgressMode string

const (
	ProgressModeOff     ProgressMode = "off"
	ProgressModeNew     ProgressMode = "new"
	ProgressModeAll     ProgressMode = "all"
	ProgressModeVerbose ProgressMode = "verbose"
)

// Normalize returns the canonical progress-mode representation.
func (m ProgressMode) Normalize() ProgressMode {
	return ProgressMode(strings.ToLower(strings.TrimSpace(string(m))))
}

// Validate reports whether the mode belongs to the supported progress taxonomy.
func (m ProgressMode) Validate() error {
	switch m.Normalize() {
	case ProgressModeOff, ProgressModeNew, ProgressModeAll, ProgressModeVerbose:
		return nil
	case "":
		return errors.New("bridges: progress tool_progress is required")
	default:
		return fmt.Errorf("bridges: unsupported progress tool_progress %q", m)
	}
}

// ProgressGrouping controls whether progress lines accumulate or post separately.
type ProgressGrouping string

const (
	ProgressGroupingAccumulate ProgressGrouping = "accumulate"
	ProgressGroupingSeparate   ProgressGrouping = "separate"
)

// Normalize returns the canonical grouping representation.
func (g ProgressGrouping) Normalize() ProgressGrouping {
	return ProgressGrouping(strings.ToLower(strings.TrimSpace(string(g))))
}

// Validate reports whether the grouping belongs to the supported set.
func (g ProgressGrouping) Validate() error {
	switch g.Normalize() {
	case ProgressGroupingAccumulate, ProgressGroupingSeparate:
		return nil
	case "":
		return errors.New("bridges: progress grouping is required")
	default:
		return fmt.Errorf("bridges: unsupported progress grouping %q", g)
	}
}

// ProgressConfig is the typed effective bridge progress configuration.
type ProgressConfig struct {
	ToolProgress ProgressMode     `json:"tool_progress"`
	Grouping     ProgressGrouping `json:"grouping"`
	Typing       bool             `json:"typing"`
	Reactions    bool             `json:"reactions"`
	PreviewLimit int              `json:"-"`
	EditInterval time.Duration    `json:"-"`
}

// Validate reports whether all public progress settings are supported.
func (c ProgressConfig) Validate() error {
	if err := c.ToolProgress.Validate(); err != nil {
		return err
	}
	if err := c.Grouping.Validate(); err != nil {
		return err
	}
	if c.PreviewLimit < 0 {
		return errors.New("bridges: progress preview limit cannot be negative")
	}
	if c.EditInterval < 0 {
		return errors.New("bridges: progress edit interval cannot be negative")
	}
	return nil
}

func (c ProgressConfig) effective() ProgressConfig {
	c.ToolProgress = c.ToolProgress.Normalize()
	c.Grouping = c.Grouping.Normalize()
	if c.PreviewLimit == 0 && c.ToolProgress != ProgressModeVerbose {
		c.PreviewLimit = defaultProgressPreviewLimit
	}
	if c.EditInterval == 0 {
		c.EditInterval = defaultProgressEditInterval
	}
	return c
}

// ResolveProgressConfig applies instance override, then platform, then global defaults.
func ResolveProgressConfig(instance *BridgeInstance, platform string) ProgressConfig {
	if instance != nil {
		if override, ok := progressConfigOverride(instance.DeliveryDefaults); ok {
			return override.effective()
		}
		if strings.TrimSpace(platform) == "" {
			platform = instance.Platform
		}
	}

	config := ProgressConfig{
		ToolProgress: ProgressModeOff,
		Grouping:     ProgressGroupingAccumulate,
	}
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "slack", progressPlatformTelegram, "discord":
		config.ToolProgress = ProgressModeNew
		config.Typing = true
		config.Reactions = true
	}
	return config.effective()
}

func progressConfigOverride(raw json.RawMessage) (ProgressConfig, bool) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return ProgressConfig{}, false
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return ProgressConfig{}, false
	}
	progress, ok := fields["progress"]
	if !ok {
		return ProgressConfig{}, false
	}
	config, err := decodeProgressConfig(progress)
	if err != nil {
		return ProgressConfig{}, false
	}
	return config, true
}

// NormalizeDeliveryDefaultsJSON validates and canonicalizes bridge delivery default JSON.
func NormalizeDeliveryDefaultsJSON(raw json.RawMessage) (json.RawMessage, error) {
	normalized, err := normalizeRawJSON(raw, "bridge instance delivery defaults")
	if err != nil {
		return nil, err
	}
	if len(normalized) == 0 || bytes.Equal(normalized, []byte("null")) {
		return nil, nil
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(normalized, &fields); err != nil {
		return nil, fmt.Errorf("bridges: bridge instance delivery defaults must be a JSON object or null: %w", err)
	}

	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := fields[key]
		if key == "progress" {
			config, err := decodeProgressConfig(value)
			if err != nil {
				return nil, err
			}
			canonical, err := json.Marshal(config)
			if err != nil {
				return nil, fmt.Errorf("bridges: marshal canonical delivery progress config: %w", err)
			}
			fields[key] = canonical
			continue
		}
		text, fieldErr := requireDeliveryDefaultStringField(value, key)
		if fieldErr != nil {
			return nil, fieldErr
		}
		if key == "mode" {
			mode := DeliveryMode(text).Normalize()
			if err := mode.Validate(); err != nil {
				return nil, err
			}
			canonical, err := json.Marshal(mode)
			if err != nil {
				return nil, fmt.Errorf("bridges: marshal canonical delivery mode: %w", err)
			}
			fields[key] = canonical
		}
	}
	canonical, err := json.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("bridges: marshal canonical bridge instance delivery defaults: %w", err)
	}
	return canonical, nil
}

func decodeProgressConfig(raw json.RawMessage) (ProgressConfig, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var config ProgressConfig
	if err := decoder.Decode(&config); err != nil {
		return ProgressConfig{}, fmt.Errorf("bridges: invalid delivery progress config: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return ProgressConfig{}, errors.New("bridges: invalid trailing data in delivery progress config")
	}
	if err := config.Validate(); err != nil {
		return ProgressConfig{}, err
	}
	return config.effective(), nil
}

func requireDeliveryDefaultStringField(raw json.RawMessage, field string) (string, error) {
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return "", fmt.Errorf("bridges: bridge instance delivery defaults field %q must be valid JSON: %w", field, err)
	}
	text, ok := decoded.(string)
	if !ok {
		return "", fmt.Errorf("bridges: bridge instance delivery defaults field %q must be a string", field)
	}
	return text, nil
}
