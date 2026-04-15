package extension

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
)

var (
	// ErrBundleInvalid reports invalid extension bundle resources.
	ErrBundleInvalid = errors.New("extension: invalid bundle")
)

// BundleSpec declares one team/product package shipped by an extension.
type BundleSpec struct {
	Name        string          `toml:"name" json:"name"`
	Description string          `toml:"description,omitempty" json:"description,omitempty"`
	Profiles    []BundleProfile `toml:"profiles" json:"profiles"`
}

// BundleProfile declares one activatable resource profile for a bundle.
type BundleProfile struct {
	Name        string               `toml:"name" json:"name"`
	Description string               `toml:"description,omitempty" json:"description,omitempty"`
	Channels    BundleChannelsConfig `toml:"channels" json:"channels"`
	Jobs        []BundleJob          `toml:"jobs,omitempty" json:"jobs,omitempty"`
	Triggers    []BundleTrigger      `toml:"triggers,omitempty" json:"triggers,omitempty"`
	Bridges     []BundleBridgePreset `toml:"bridges,omitempty" json:"bridges,omitempty"`
}

// BundleChannelsConfig declares the canonical channels packaged by a profile.
type BundleChannelsConfig struct {
	Primary string          `toml:"primary,omitempty" json:"primary,omitempty"`
	Items   []BundleChannel `toml:"items,omitempty" json:"items,omitempty"`
}

// BundleChannel describes one declared network channel bundled by a profile.
type BundleChannel struct {
	Name        string `toml:"name" json:"name"`
	Description string `toml:"description,omitempty" json:"description,omitempty"`
}

// BundleJob declares one package-managed automation job template.
type BundleJob struct {
	Name      string                        `toml:"name" json:"name"`
	AgentName string                        `toml:"agent" json:"agent"`
	Prompt    string                        `toml:"prompt" json:"prompt"`
	Schedule  automationpkg.ScheduleSpec    `toml:"schedule" json:"schedule"`
	Task      *automationpkg.JobTaskConfig  `toml:"task,omitempty" json:"task,omitempty"`
	Enabled   bool                          `toml:"enabled" json:"enabled"`
	Retry     automationpkg.RetryConfig     `toml:"retry,omitempty" json:"retry,omitempty"`
	FireLimit automationpkg.FireLimitConfig `toml:"fire_limit,omitempty" json:"fire_limit,omitempty"`
}

// BundleTrigger declares one package-managed automation trigger template.
type BundleTrigger struct {
	Name         string                        `toml:"name" json:"name"`
	AgentName    string                        `toml:"agent" json:"agent"`
	Prompt       string                        `toml:"prompt" json:"prompt"`
	Event        string                        `toml:"event" json:"event"`
	Filter       map[string]string             `toml:"filter,omitempty" json:"filter,omitempty"`
	Enabled      bool                          `toml:"enabled" json:"enabled"`
	Retry        automationpkg.RetryConfig     `toml:"retry,omitempty" json:"retry,omitempty"`
	FireLimit    automationpkg.FireLimitConfig `toml:"fire_limit,omitempty" json:"fire_limit,omitempty"`
	EndpointSlug string                        `toml:"endpoint_slug,omitempty" json:"endpoint_slug,omitempty"`
}

// BundleBridgePreset declares one package-managed bridge instance template.
type BundleBridgePreset struct {
	Name             string                   `toml:"name" json:"name"`
	ExtensionName    string                   `toml:"extension_name,omitempty" json:"extension_name,omitempty"`
	Platform         string                   `toml:"platform,omitempty" json:"platform,omitempty"`
	DisplayName      string                   `toml:"display_name" json:"display_name"`
	RoutingPolicy    bridgepkg.RoutingPolicy  `toml:"routing_policy" json:"routing_policy"`
	DeliveryDefaults json.RawMessage          `toml:"delivery_defaults,omitempty" json:"delivery_defaults,omitempty"`
	SecretSlots      []BundleBridgeSecretSlot `toml:"secret_slots,omitempty" json:"secret_slots,omitempty"`
}

// BundleBridgeSecretSlot declares one required bridge secret binding.
type BundleBridgeSecretSlot struct {
	Name        string `toml:"name" json:"name"`
	Kind        string `toml:"kind" json:"kind"`
	Description string `toml:"description,omitempty" json:"description,omitempty"`
}

type bundleDocument struct {
	Bundle bundleRawSpec `toml:"bundle" json:"bundle"`

	Name        string             `toml:"name" json:"name"`
	Description string             `toml:"description,omitempty" json:"description,omitempty"`
	Profiles    []bundleRawProfile `toml:"profiles" json:"profiles"`
}

type bundleRawSpec struct {
	Name        string             `toml:"name" json:"name"`
	Description string             `toml:"description,omitempty" json:"description,omitempty"`
	Profiles    []bundleRawProfile `toml:"profiles" json:"profiles"`
}

type bundleRawProfile struct {
	Name        string               `toml:"name" json:"name"`
	Description string               `toml:"description,omitempty" json:"description,omitempty"`
	Channels    BundleChannelsConfig `toml:"channels" json:"channels"`
	Jobs        []bundleRawJob       `toml:"jobs,omitempty" json:"jobs,omitempty"`
	Triggers    []bundleRawTrigger   `toml:"triggers,omitempty" json:"triggers,omitempty"`
	Bridges     []BundleBridgePreset `toml:"bridges,omitempty" json:"bridges,omitempty"`
}

type bundleRawJob struct {
	Name      string                        `toml:"name" json:"name"`
	AgentName string                        `toml:"agent" json:"agent"`
	Prompt    string                        `toml:"prompt" json:"prompt"`
	Schedule  automationpkg.ScheduleSpec    `toml:"schedule" json:"schedule"`
	Task      *automationpkg.JobTaskConfig  `toml:"task,omitempty" json:"task,omitempty"`
	Enabled   *bool                         `toml:"enabled,omitempty" json:"enabled,omitempty"`
	Retry     automationpkg.RetryConfig     `toml:"retry,omitempty" json:"retry,omitempty"`
	FireLimit automationpkg.FireLimitConfig `toml:"fire_limit,omitempty" json:"fire_limit,omitempty"`
}

type bundleRawTrigger struct {
	Name         string                        `toml:"name" json:"name"`
	AgentName    string                        `toml:"agent" json:"agent"`
	Prompt       string                        `toml:"prompt" json:"prompt"`
	Event        string                        `toml:"event" json:"event"`
	Filter       map[string]string             `toml:"filter,omitempty" json:"filter,omitempty"`
	Enabled      *bool                         `toml:"enabled,omitempty" json:"enabled,omitempty"`
	Retry        automationpkg.RetryConfig     `toml:"retry,omitempty" json:"retry,omitempty"`
	FireLimit    automationpkg.FireLimitConfig `toml:"fire_limit,omitempty" json:"fire_limit,omitempty"`
	EndpointSlug string                        `toml:"endpoint_slug,omitempty" json:"endpoint_slug,omitempty"`
}

// LoadBundleSpecs resolves and validates bundle resources declared by a manifest.
func LoadBundleSpecs(rootDir string, manifest *Manifest) ([]BundleSpec, error) {
	if manifest == nil || len(manifest.Resources.Bundles) == 0 {
		return nil, nil
	}

	loaded := make(map[string]BundleSpec)
	for _, resourcePath := range manifest.Resources.Bundles {
		resourceRoot, err := resolveResourcePath(rootDir, resourcePath)
		if err != nil {
			return nil, err
		}
		files, err := collectBundleFiles(resourceRoot)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			spec, err := loadBundleSpecAtPath(file)
			if err != nil {
				return nil, err
			}
			if err := spec.Validate(manifest); err != nil {
				return nil, err
			}
			key := bundleLookupKey(spec.Name)
			if _, exists := loaded[key]; exists {
				return nil, fmt.Errorf("%w: duplicate bundle %q", ErrBundleInvalid, spec.Name)
			}
			loaded[key] = spec
		}
	}

	bundles := make([]BundleSpec, 0, len(loaded))
	for _, key := range sortedKeys(loaded) {
		bundles = append(bundles, loaded[key])
	}
	return bundles, nil
}

// Validate ensures the bundle spec is internally consistent for the owning manifest.
func (b BundleSpec) Validate(manifest *Manifest) error {
	name := strings.TrimSpace(b.Name)
	if name == "" {
		return fmt.Errorf("%w: bundle.name is required", ErrBundleInvalid)
	}
	if len(b.Profiles) == 0 {
		return fmt.Errorf("%w: bundle %q must declare at least one profile", ErrBundleInvalid, name)
	}

	seenProfiles := make(map[string]struct{}, len(b.Profiles))
	for idx, profile := range b.Profiles {
		profileName := strings.TrimSpace(profile.Name)
		if profileName == "" {
			return fmt.Errorf("%w: bundle %q profile[%d].name is required", ErrBundleInvalid, name, idx)
		}
		profileKey := bundleLookupKey(profileName)
		if _, exists := seenProfiles[profileKey]; exists {
			return fmt.Errorf("%w: bundle %q profile %q is duplicated", ErrBundleInvalid, name, profileName)
		}
		seenProfiles[profileKey] = struct{}{}
		if err := profile.Validate(name, manifest); err != nil {
			return err
		}
	}
	return nil
}

// Validate ensures one bundle profile is internally consistent.
func (p BundleProfile) Validate(bundleName string, manifest *Manifest) error {
	channelNames := make(map[string]struct{}, len(p.Channels.Items))
	for idx, item := range p.Channels.Items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			return fmt.Errorf("%w: bundle %q profile %q channels[%d].name is required", ErrBundleInvalid, bundleName, p.Name, idx)
		}
		if _, exists := channelNames[name]; exists {
			return fmt.Errorf("%w: bundle %q profile %q channel %q is duplicated", ErrBundleInvalid, bundleName, p.Name, name)
		}
		channelNames[name] = struct{}{}
	}

	primary := strings.TrimSpace(p.Channels.Primary)
	switch {
	case len(channelNames) > 0 && primary == "":
		return fmt.Errorf("%w: bundle %q profile %q must declare channels.primary", ErrBundleInvalid, bundleName, p.Name)
	case primary != "":
		if _, ok := channelNames[primary]; !ok {
			return fmt.Errorf("%w: bundle %q profile %q primary channel %q is not declared", ErrBundleInvalid, bundleName, p.Name, primary)
		}
	}

	seenJobs := make(map[string]struct{}, len(p.Jobs))
	for _, job := range p.Jobs {
		jobName := strings.TrimSpace(job.Name)
		if jobName == "" {
			return fmt.Errorf("%w: bundle %q profile %q job.name is required", ErrBundleInvalid, bundleName, p.Name)
		}
		if _, exists := seenJobs[jobName]; exists {
			return fmt.Errorf("%w: bundle %q profile %q job %q is duplicated", ErrBundleInvalid, bundleName, p.Name, jobName)
		}
		seenJobs[jobName] = struct{}{}
		if err := job.Validate(bundleName, p.Name, channelNames); err != nil {
			return err
		}
	}

	seenTriggers := make(map[string]struct{}, len(p.Triggers))
	for _, trigger := range p.Triggers {
		triggerName := strings.TrimSpace(trigger.Name)
		if triggerName == "" {
			return fmt.Errorf("%w: bundle %q profile %q trigger.name is required", ErrBundleInvalid, bundleName, p.Name)
		}
		if _, exists := seenTriggers[triggerName]; exists {
			return fmt.Errorf("%w: bundle %q profile %q trigger %q is duplicated", ErrBundleInvalid, bundleName, p.Name, triggerName)
		}
		seenTriggers[triggerName] = struct{}{}
		if err := trigger.Validate(bundleName, p.Name); err != nil {
			return err
		}
	}

	seenBridges := make(map[string]struct{}, len(p.Bridges))
	for _, bridge := range p.Bridges {
		bridgeName := strings.TrimSpace(bridge.Name)
		if bridgeName == "" {
			return fmt.Errorf("%w: bundle %q profile %q bridge.name is required", ErrBundleInvalid, bundleName, p.Name)
		}
		if _, exists := seenBridges[bridgeName]; exists {
			return fmt.Errorf("%w: bundle %q profile %q bridge %q is duplicated", ErrBundleInvalid, bundleName, p.Name, bridgeName)
		}
		seenBridges[bridgeName] = struct{}{}
		if err := bridge.Validate(bundleName, p.Name, manifest); err != nil {
			return err
		}
	}
	return nil
}

// Validate ensures one bundle job is internally consistent.
func (j BundleJob) Validate(bundleName string, profileName string, channelNames map[string]struct{}) error {
	job := automationpkg.Job{
		ID:        "bundle-validation",
		Scope:     automationpkg.AutomationScopeGlobal,
		Name:      strings.TrimSpace(j.Name),
		AgentName: strings.TrimSpace(j.AgentName),
		Prompt:    strings.TrimSpace(j.Prompt),
		Schedule:  &j.Schedule,
		Task:      cloneBundleTaskConfig(j.Task),
		Enabled:   j.Enabled,
		Retry:     j.Retry,
		FireLimit: j.FireLimit,
		Source:    automationpkg.JobSourcePackage,
	}
	if err := job.Validate("bundle.jobs"); err != nil {
		return fmt.Errorf("%w: bundle %q profile %q job %q: %w", ErrBundleInvalid, bundleName, profileName, j.Name, err)
	}
	if j.Task != nil {
		channel := strings.TrimSpace(j.Task.NetworkChannel)
		if channel != "" {
			if _, ok := channelNames[channel]; !ok {
				return fmt.Errorf("%w: bundle %q profile %q job %q references undeclared channel %q", ErrBundleInvalid, bundleName, profileName, j.Name, channel)
			}
		}
	}
	return nil
}

// Validate ensures one bundle trigger is internally consistent.
func (t BundleTrigger) Validate(bundleName string, profileName string) error {
	trigger := automationpkg.Trigger{
		ID:           "bundle-validation",
		Scope:        automationpkg.AutomationScopeGlobal,
		Name:         strings.TrimSpace(t.Name),
		AgentName:    strings.TrimSpace(t.AgentName),
		Prompt:       strings.TrimSpace(t.Prompt),
		Event:        strings.TrimSpace(t.Event),
		Filter:       cloneStringMap(t.Filter),
		Enabled:      t.Enabled,
		Retry:        t.Retry,
		FireLimit:    t.FireLimit,
		Source:       automationpkg.JobSourcePackage,
		EndpointSlug: strings.TrimSpace(t.EndpointSlug),
	}
	if err := trigger.Validate("bundle.triggers"); err != nil {
		return fmt.Errorf("%w: bundle %q profile %q trigger %q: %w", ErrBundleInvalid, bundleName, profileName, t.Name, err)
	}
	return nil
}

// Validate ensures one bundle bridge preset is internally consistent.
func (b BundleBridgePreset) Validate(bundleName string, profileName string, manifest *Manifest) error {
	displayName := strings.TrimSpace(b.DisplayName)
	if displayName == "" {
		return fmt.Errorf("%w: bundle %q profile %q bridge %q display_name is required", ErrBundleInvalid, bundleName, profileName, b.Name)
	}
	if err := b.RoutingPolicy.Validate(); err != nil {
		return fmt.Errorf("%w: bundle %q profile %q bridge %q routing_policy: %w", ErrBundleInvalid, bundleName, profileName, b.Name, err)
	}
	trimmedDeliveryDefaults := strings.TrimSpace(string(b.DeliveryDefaults))
	if trimmedDeliveryDefaults != "" && !json.Valid([]byte(trimmedDeliveryDefaults)) {
		return fmt.Errorf("%w: bundle %q profile %q bridge %q delivery_defaults: invalid JSON", ErrBundleInvalid, bundleName, profileName, b.Name)
	}
	for _, slot := range b.SecretSlots {
		if strings.TrimSpace(slot.Name) == "" {
			return fmt.Errorf("%w: bundle %q profile %q bridge %q secret_slots.name is required", ErrBundleInvalid, bundleName, profileName, b.Name)
		}
		if strings.TrimSpace(slot.Kind) == "" {
			return fmt.Errorf("%w: bundle %q profile %q bridge %q secret slot %q kind is required", ErrBundleInvalid, bundleName, profileName, b.Name, slot.Name)
		}
	}

	if strings.TrimSpace(b.ExtensionName) == "" && manifest != nil && strings.TrimSpace(b.Platform) == "" {
		if !providesCapability(manifest.Capabilities.Provides, "bridge.adapter") {
			return fmt.Errorf("%w: bundle %q profile %q bridge %q must declare extension_name or platform", ErrBundleInvalid, bundleName, profileName, b.Name)
		}
	}
	return nil
}

func loadBundleSpecAtPath(path string) (BundleSpec, error) {
	switch strings.ToLower(filepath.Ext(strings.TrimSpace(path))) {
	case ".toml":
		return loadBundleTOML(path)
	case ".json":
		return loadBundleJSON(path)
	default:
		return BundleSpec{}, fmt.Errorf("%w: unsupported bundle path %q", ErrBundleInvalid, path)
	}
}

func bundleLookupKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func loadBundleTOML(path string) (BundleSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return BundleSpec{}, fmt.Errorf("extension: read bundle %q: %w", path, err)
	}

	var doc bundleDocument
	if _, err := toml.Decode(string(data), &doc); err != nil {
		return BundleSpec{}, fmt.Errorf("extension: decode bundle %q: %w", path, err)
	}
	return doc.toBundleSpec()
}

func loadBundleJSON(path string) (BundleSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return BundleSpec{}, fmt.Errorf("extension: read bundle %q: %w", path, err)
	}

	var doc bundleDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return BundleSpec{}, fmt.Errorf("extension: decode bundle %q: %w", path, err)
	}
	return doc.toBundleSpec()
}

func (d bundleDocument) toBundleSpec() (BundleSpec, error) {
	name, err := mergeManifestValue("bundle.name", d.Name, d.Bundle.Name)
	if err != nil {
		return BundleSpec{}, err
	}
	description, err := mergeManifestValue("bundle.description", d.Description, d.Bundle.Description)
	if err != nil {
		return BundleSpec{}, err
	}

	profiles := d.Profiles
	if len(profiles) == 0 {
		profiles = d.Bundle.Profiles
	}
	if len(d.Profiles) > 0 && len(d.Bundle.Profiles) > 0 {
		return BundleSpec{}, fmt.Errorf("%w: conflicting root and bundle profiles", ErrBundleInvalid)
	}

	spec := BundleSpec{
		Name:        strings.TrimSpace(name),
		Description: strings.TrimSpace(description),
		Profiles:    make([]BundleProfile, 0, len(profiles)),
	}
	for _, profile := range profiles {
		spec.Profiles = append(spec.Profiles, profile.toBundleProfile())
	}
	return spec, nil
}

func (p bundleRawProfile) toBundleProfile() BundleProfile {
	profile := BundleProfile{
		Name:        strings.TrimSpace(p.Name),
		Description: strings.TrimSpace(p.Description),
		Channels: BundleChannelsConfig{
			Primary: strings.TrimSpace(p.Channels.Primary),
			Items:   normalizeBundleChannels(p.Channels.Items),
		},
		Jobs:     make([]BundleJob, 0, len(p.Jobs)),
		Triggers: make([]BundleTrigger, 0, len(p.Triggers)),
		Bridges:  normalizeBundleBridges(p.Bridges),
	}
	for _, job := range p.Jobs {
		profile.Jobs = append(profile.Jobs, job.toBundleJob())
	}
	for _, trigger := range p.Triggers {
		profile.Triggers = append(profile.Triggers, trigger.toBundleTrigger())
	}
	return profile
}

func (j bundleRawJob) toBundleJob() BundleJob {
	job := BundleJob{
		Name:      strings.TrimSpace(j.Name),
		AgentName: strings.TrimSpace(j.AgentName),
		Prompt:    strings.TrimSpace(j.Prompt),
		Schedule:  j.Schedule,
		Task:      cloneBundleTaskConfig(j.Task),
		Enabled:   true,
		Retry:     j.Retry,
		FireLimit: j.FireLimit,
	}
	if j.Enabled != nil {
		job.Enabled = *j.Enabled
	}
	return job
}

func (t bundleRawTrigger) toBundleTrigger() BundleTrigger {
	trigger := BundleTrigger{
		Name:         strings.TrimSpace(t.Name),
		AgentName:    strings.TrimSpace(t.AgentName),
		Prompt:       strings.TrimSpace(t.Prompt),
		Event:        strings.TrimSpace(t.Event),
		Filter:       cloneStringMap(t.Filter),
		Enabled:      true,
		Retry:        t.Retry,
		FireLimit:    t.FireLimit,
		EndpointSlug: strings.TrimSpace(t.EndpointSlug),
	}
	if t.Enabled != nil {
		trigger.Enabled = *t.Enabled
	}
	return trigger
}

func collectBundleFiles(root string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("extension: stat bundle resource %q: %w", root, err)
	}
	if !info.IsDir() {
		if isBundleFile(root) {
			return []string{root}, nil
		}
		return nil, fmt.Errorf("%w: unsupported bundle resource %q", ErrBundleInvalid, root)
	}

	files := make([]string, 0)
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if isBundleFile(path) {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("extension: collect bundle files from %q: %w", root, err)
	}

	slices.Sort(files)
	return files, nil
}

func isBundleFile(path string) bool {
	switch strings.ToLower(filepath.Ext(strings.TrimSpace(path))) {
	case ".toml", ".json":
		return true
	default:
		return false
	}
}

func normalizeBundleChannels(values []BundleChannel) []BundleChannel {
	if len(values) == 0 {
		return nil
	}
	normalized := make([]BundleChannel, 0, len(values))
	for _, value := range values {
		normalized = append(normalized, BundleChannel{
			Name:        strings.TrimSpace(value.Name),
			Description: strings.TrimSpace(value.Description),
		})
	}
	return normalized
}

func normalizeBundleBridges(values []BundleBridgePreset) []BundleBridgePreset {
	if len(values) == 0 {
		return nil
	}
	normalized := make([]BundleBridgePreset, 0, len(values))
	for _, value := range values {
		next := value
		next.Name = strings.TrimSpace(next.Name)
		next.ExtensionName = strings.TrimSpace(next.ExtensionName)
		next.Platform = strings.TrimSpace(next.Platform)
		next.DisplayName = strings.TrimSpace(next.DisplayName)
		next.DeliveryDefaults = cloneRawMessage(next.DeliveryDefaults)
		next.SecretSlots = slices.Clone(next.SecretSlots)
		for idx := range next.SecretSlots {
			next.SecretSlots[idx].Name = strings.TrimSpace(next.SecretSlots[idx].Name)
			next.SecretSlots[idx].Kind = strings.TrimSpace(next.SecretSlots[idx].Kind)
			next.SecretSlots[idx].Description = strings.TrimSpace(next.SecretSlots[idx].Description)
		}
		normalized = append(normalized, next)
	}
	return normalized
}

func cloneBundleTaskConfig(config *automationpkg.JobTaskConfig) *automationpkg.JobTaskConfig {
	if config == nil {
		return nil
	}
	cloned := *config
	if config.Owner != nil {
		owner := *config.Owner
		cloned.Owner = &owner
	}
	return &cloned
}
