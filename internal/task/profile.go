package task

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"
)

const (
	// CoordinatorModeInherit uses daemon/workspace coordinator defaults.
	CoordinatorModeInherit CoordinatorMode = "inherit"
	// CoordinatorModeGuided injects task-specific guidance into the existing coordinator.
	CoordinatorModeGuided CoordinatorMode = "guided"

	// WorkerModeInherit uses normal task/run and workspace worker defaults.
	WorkerModeInherit WorkerMode = "inherit"
	// WorkerModeSelect narrows worker selection using the task profile.
	WorkerModeSelect WorkerMode = "select"

	// SandboxModeInherit uses workspace/global sandbox defaults.
	SandboxModeInherit SandboxMode = "inherit"
	// SandboxModeNone disables task-level sandbox selection when config permits it.
	SandboxModeNone SandboxMode = "none"
	// SandboxModeRef selects one named sandbox reference at session start.
	SandboxModeRef SandboxMode = "ref"
)

const (
	defaultCoordinatorGuidanceMaxBytes = 8192
	profileSelectorMaxBytes            = 128
)

// CoordinatorMode identifies task-specific coordinator bootstrap behavior.
type CoordinatorMode string

// WorkerMode identifies how a task narrows worker selection.
type WorkerMode string

// SandboxMode identifies task-level sandbox selection behavior.
type SandboxMode string

// ExecutionProfile is the typed task-owned orchestration selection state.
type ExecutionProfile struct {
	TaskID       string             `json:"task_id"`
	Coordinator  CoordinatorProfile `json:"coordinator"`
	Worker       WorkerProfile      `json:"worker"`
	Review       ReviewProfile      `json:"review"`
	Participants ParticipantPolicy  `json:"participants"`
	Sandbox      SandboxPolicy      `json:"sandbox"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

// CoordinatorProfile supplies optional guidance to the existing coordinator runtime.
type CoordinatorProfile struct {
	Mode      CoordinatorMode `json:"mode"`
	AgentName string          `json:"agent_name,omitempty"`
	Provider  string          `json:"provider,omitempty"`
	Model     string          `json:"model,omitempty"`
	Guidance  string          `json:"guidance,omitempty"`
}

// WorkerProfile narrows eligible task workers without granting runtime authority.
type WorkerProfile struct {
	Mode                  WorkerMode `json:"mode"`
	AgentName             string     `json:"agent_name,omitempty"`
	Provider              string     `json:"provider,omitempty"`
	Model                 string     `json:"model,omitempty"`
	AllowedAgentNames     []string   `json:"allowed_agent_names,omitempty"`
	PreferredAgentNames   []string   `json:"preferred_agent_names,omitempty"`
	RequiredCapabilities  []string   `json:"required_capabilities,omitempty"`
	PreferredCapabilities []string   `json:"preferred_capabilities,omitempty"`
}

// ReviewProfile narrows reviewer execution shape; verdict authority stays in task review APIs.
type ReviewProfile struct {
	AgentName             string   `json:"agent_name,omitempty"`
	Provider              string   `json:"provider,omitempty"`
	Model                 string   `json:"model,omitempty"`
	AllowedAgentNames     []string `json:"allowed_agent_names,omitempty"`
	PreferredAgentNames   []string `json:"preferred_agent_names,omitempty"`
	AllowedChannelIDs     []string `json:"allowed_channel_ids,omitempty"`
	PreferredChannelIDs   []string `json:"preferred_channel_ids,omitempty"`
	AllowedPeerIDs        []string `json:"allowed_peer_ids,omitempty"`
	PreferredPeerIDs      []string `json:"preferred_peer_ids,omitempty"`
	RequiredCapabilities  []string `json:"required_capabilities,omitempty"`
	PreferredCapabilities []string `json:"preferred_capabilities,omitempty"`
}

// ParticipantPolicy is an upper-bound routing policy, not a permission grant.
type ParticipantPolicy struct {
	AllowedChannelIDs     []string `json:"allowed_channel_ids,omitempty"`
	PreferredChannelIDs   []string `json:"preferred_channel_ids,omitempty"`
	AllowedPeerIDs        []string `json:"allowed_peer_ids,omitempty"`
	PreferredPeerIDs      []string `json:"preferred_peer_ids,omitempty"`
	AllowedAgentNames     []string `json:"allowed_agent_names,omitempty"`
	PreferredAgentNames   []string `json:"preferred_agent_names,omitempty"`
	RequiredCapabilities  []string `json:"required_capabilities,omitempty"`
	PreferredCapabilities []string `json:"preferred_capabilities,omitempty"`
}

// SandboxPolicy selects task-level sandbox behavior at session start.
type SandboxPolicy struct {
	Mode       SandboxMode `json:"mode"`
	SandboxRef string      `json:"sandbox_ref,omitempty"`
}

// ExecutionProfileValidationOptions carries config-backed gates without coupling task to config.
type ExecutionProfileValidationOptions struct {
	AllowProviderOverride       bool
	AllowSandboxNone            bool
	AllowSandboxRef             bool
	MaxCoordinatorGuidanceBytes int
}

// DefaultExecutionProfileValidationOptions returns the permissive built-in gates.
func DefaultExecutionProfileValidationOptions() ExecutionProfileValidationOptions {
	return ExecutionProfileValidationOptions{
		AllowProviderOverride:       true,
		AllowSandboxNone:            true,
		AllowSandboxRef:             true,
		MaxCoordinatorGuidanceBytes: defaultCoordinatorGuidanceMaxBytes,
	}
}

// Normalize returns a canonical copy with trimmed fields, default modes, and stable selector sets.
func (p *ExecutionProfile) Normalize(options ExecutionProfileValidationOptions) (ExecutionProfile, error) {
	if p == nil {
		return ExecutionProfile{}, fmt.Errorf("%w: task_execution_profile is required", ErrValidation)
	}
	normalized := *p
	normalized.TaskID = strings.TrimSpace(normalized.TaskID)
	normalized.Coordinator = normalizeCoordinatorProfile(normalized.Coordinator)
	normalized.Worker = normalizeWorkerProfile(normalized.Worker)
	normalized.Review = normalizeReviewProfile(normalized.Review)
	normalized.Participants = normalizeParticipantPolicy(normalized.Participants)
	normalized.Sandbox = normalizeSandboxPolicy(normalized.Sandbox)

	if err := (&normalized).Validate(options); err != nil {
		return ExecutionProfile{}, err
	}
	return normalized, nil
}

// Validate reports whether the profile can be persisted as typed orchestration state.
func (p *ExecutionProfile) Validate(options ExecutionProfileValidationOptions) error {
	if p == nil {
		return fmt.Errorf("%w: task_execution_profile is required", ErrValidation)
	}
	if strings.TrimSpace(p.TaskID) == "" {
		return fmt.Errorf("%w: task_execution_profile.task_id is required", ErrValidation)
	}
	if err := validateCoordinatorProfile(p.Coordinator, options); err != nil {
		return err
	}
	if err := validateWorkerProfile(p.Worker, options); err != nil {
		return err
	}
	if err := validateReviewProfile(p.Review, options); err != nil {
		return err
	}
	if err := validateParticipantPolicy(p.Participants); err != nil {
		return err
	}
	return validateSandboxPolicy(p.Sandbox, options)
}

// Normalize returns the normalized coordinator mode.
func (m CoordinatorMode) Normalize() CoordinatorMode {
	return CoordinatorMode(strings.ToLower(strings.TrimSpace(string(m))))
}

// Normalize returns the normalized worker mode.
func (m WorkerMode) Normalize() WorkerMode {
	return WorkerMode(strings.ToLower(strings.TrimSpace(string(m))))
}

// Normalize returns the normalized sandbox mode.
func (m SandboxMode) Normalize() SandboxMode {
	return SandboxMode(strings.ToLower(strings.TrimSpace(string(m))))
}

func normalizeCoordinatorProfile(profile CoordinatorProfile) CoordinatorProfile {
	profile.Mode = profile.Mode.Normalize()
	if profile.Mode == "" {
		profile.Mode = CoordinatorModeInherit
	}
	profile.AgentName = strings.TrimSpace(profile.AgentName)
	profile.Provider = strings.TrimSpace(profile.Provider)
	profile.Model = strings.TrimSpace(profile.Model)
	profile.Guidance = strings.TrimSpace(profile.Guidance)
	return profile
}

func normalizeWorkerProfile(profile WorkerProfile) WorkerProfile {
	profile.Mode = profile.Mode.Normalize()
	if profile.Mode == "" {
		profile.Mode = WorkerModeInherit
	}
	profile.AgentName = strings.TrimSpace(profile.AgentName)
	profile.Provider = strings.TrimSpace(profile.Provider)
	profile.Model = strings.TrimSpace(profile.Model)
	profile.AllowedAgentNames = normalizeProfileSelectorList(profile.AllowedAgentNames)
	profile.PreferredAgentNames = normalizeProfileSelectorList(profile.PreferredAgentNames)
	profile.RequiredCapabilities = normalizeProfileSelectorList(profile.RequiredCapabilities)
	profile.PreferredCapabilities = normalizeProfileSelectorList(profile.PreferredCapabilities)
	return profile
}

func normalizeReviewProfile(profile ReviewProfile) ReviewProfile {
	profile.AgentName = strings.TrimSpace(profile.AgentName)
	profile.Provider = strings.TrimSpace(profile.Provider)
	profile.Model = strings.TrimSpace(profile.Model)
	profile.AllowedAgentNames = normalizeProfileSelectorList(profile.AllowedAgentNames)
	profile.PreferredAgentNames = normalizeProfileSelectorList(profile.PreferredAgentNames)
	profile.AllowedChannelIDs = normalizeProfileSelectorList(profile.AllowedChannelIDs)
	profile.PreferredChannelIDs = normalizeProfileSelectorList(profile.PreferredChannelIDs)
	profile.AllowedPeerIDs = normalizeProfileSelectorList(profile.AllowedPeerIDs)
	profile.PreferredPeerIDs = normalizeProfileSelectorList(profile.PreferredPeerIDs)
	profile.RequiredCapabilities = normalizeProfileSelectorList(profile.RequiredCapabilities)
	profile.PreferredCapabilities = normalizeProfileSelectorList(profile.PreferredCapabilities)
	return profile
}

func normalizeParticipantPolicy(policy ParticipantPolicy) ParticipantPolicy {
	policy.AllowedChannelIDs = normalizeProfileSelectorList(policy.AllowedChannelIDs)
	policy.PreferredChannelIDs = normalizeProfileSelectorList(policy.PreferredChannelIDs)
	policy.AllowedPeerIDs = normalizeProfileSelectorList(policy.AllowedPeerIDs)
	policy.PreferredPeerIDs = normalizeProfileSelectorList(policy.PreferredPeerIDs)
	policy.AllowedAgentNames = normalizeProfileSelectorList(policy.AllowedAgentNames)
	policy.PreferredAgentNames = normalizeProfileSelectorList(policy.PreferredAgentNames)
	policy.RequiredCapabilities = normalizeProfileSelectorList(policy.RequiredCapabilities)
	policy.PreferredCapabilities = normalizeProfileSelectorList(policy.PreferredCapabilities)
	return policy
}

func normalizeSandboxPolicy(policy SandboxPolicy) SandboxPolicy {
	policy.Mode = policy.Mode.Normalize()
	if policy.Mode == "" {
		policy.Mode = SandboxModeInherit
	}
	policy.SandboxRef = strings.TrimSpace(policy.SandboxRef)
	return policy
}

func normalizeProfileSelectorList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	sort.Strings(normalized)
	return normalized
}

func validateCoordinatorProfile(profile CoordinatorProfile, options ExecutionProfileValidationOptions) error {
	switch profile.Mode.Normalize() {
	case CoordinatorModeInherit, CoordinatorModeGuided:
	default:
		return fmt.Errorf(
			"%w: task_execution_profile.coordinator.mode must be %q or %q: %q",
			ErrValidation,
			CoordinatorModeInherit,
			CoordinatorModeGuided,
			profile.Mode,
		)
	}
	if len(profile.Guidance) > coordinatorGuidanceMaxBytes(options) {
		return fmt.Errorf(
			"%w: task_execution_profile.coordinator.guidance exceeds %d bytes",
			ErrValidation,
			coordinatorGuidanceMaxBytes(options),
		)
	}
	return validateProviderModelGate(
		profile.Provider,
		profile.Model,
		options,
		"task_execution_profile.coordinator",
	)
}

func validateWorkerProfile(profile WorkerProfile, options ExecutionProfileValidationOptions) error {
	switch profile.Mode.Normalize() {
	case WorkerModeInherit, WorkerModeSelect:
	default:
		return fmt.Errorf(
			"%w: task_execution_profile.worker.mode must be %q or %q: %q",
			ErrValidation,
			WorkerModeInherit,
			WorkerModeSelect,
			profile.Mode,
		)
	}
	if err := validateProviderModelGate(
		profile.Provider,
		profile.Model,
		options,
		"task_execution_profile.worker",
	); err != nil {
		return err
	}
	if err := validateProfileSelectorIDs(
		profile.AllowedAgentNames,
		"task_execution_profile.worker.allowed_agent_names",
	); err != nil {
		return err
	}
	if err := validateProfileSelectorIDs(
		profile.PreferredAgentNames,
		"task_execution_profile.worker.preferred_agent_names",
	); err != nil {
		return err
	}
	if profile.AgentName != "" && !profileAgentAllowed(profile.AgentName, profile.AllowedAgentNames) {
		return fmt.Errorf(
			"%w: task_execution_profile.worker.agent_name %q is outside allowed_agent_names",
			ErrValidation,
			profile.AgentName,
		)
	}
	if err := ValidateCapabilityIDs(
		profile.RequiredCapabilities,
		"task_execution_profile.worker.required_capabilities",
	); err != nil {
		return err
	}
	return ValidateCapabilityIDs(
		profile.PreferredCapabilities,
		"task_execution_profile.worker.preferred_capabilities",
	)
}

func validateReviewProfile(profile ReviewProfile, options ExecutionProfileValidationOptions) error {
	if err := validateProviderModelGate(
		profile.Provider,
		profile.Model,
		options,
		"task_execution_profile.review",
	); err != nil {
		return err
	}
	if err := validateAgentSelectors(
		profile.AgentName,
		profile.AllowedAgentNames,
		profile.PreferredAgentNames,
		"review",
	); err != nil {
		return err
	}
	if err := validateChannelSelectors(
		profile.AllowedChannelIDs,
		profile.PreferredChannelIDs,
		"review",
	); err != nil {
		return err
	}
	if err := validatePeerSelectors(
		profile.AllowedPeerIDs,
		profile.PreferredPeerIDs,
		"review",
	); err != nil {
		return err
	}
	return validateCapabilitySelectors(profile.RequiredCapabilities, profile.PreferredCapabilities, "review")
}

func validateParticipantPolicy(policy ParticipantPolicy) error {
	if err := validateAgentSelectors(
		"",
		policy.AllowedAgentNames,
		policy.PreferredAgentNames,
		"participants",
	); err != nil {
		return err
	}
	if err := validateChannelSelectors(
		policy.AllowedChannelIDs,
		policy.PreferredChannelIDs,
		"participants",
	); err != nil {
		return err
	}
	if err := validatePeerSelectors(
		policy.AllowedPeerIDs,
		policy.PreferredPeerIDs,
		"participants",
	); err != nil {
		return err
	}
	return validateCapabilitySelectors(
		policy.RequiredCapabilities,
		policy.PreferredCapabilities,
		"participants",
	)
}

func validateSandboxPolicy(policy SandboxPolicy, options ExecutionProfileValidationOptions) error {
	switch policy.Mode.Normalize() {
	case SandboxModeInherit:
		if policy.SandboxRef != "" {
			return fmt.Errorf("%w: task_execution_profile.sandbox.sandbox_ref must be empty for inherit", ErrValidation)
		}
	case SandboxModeNone:
		if !options.AllowSandboxNone {
			return fmt.Errorf("%w: task_execution_profile.sandbox.mode none is disabled by config", ErrValidation)
		}
		if policy.SandboxRef != "" {
			return fmt.Errorf("%w: task_execution_profile.sandbox.sandbox_ref must be empty for none", ErrValidation)
		}
	case SandboxModeRef:
		if !options.AllowSandboxRef {
			return fmt.Errorf("%w: task_execution_profile.sandbox.mode ref is disabled by config", ErrValidation)
		}
		if policy.SandboxRef == "" {
			return fmt.Errorf("%w: task_execution_profile.sandbox.sandbox_ref is required for ref", ErrValidation)
		}
	default:
		return fmt.Errorf(
			"%w: task_execution_profile.sandbox.mode must be %q, %q, or %q: %q",
			ErrValidation,
			SandboxModeInherit,
			SandboxModeNone,
			SandboxModeRef,
			policy.Mode,
		)
	}
	return validateSelectorAtom(policy.SandboxRef, "task_execution_profile.sandbox.sandbox_ref", true)
}

func validateProviderModelGate(
	provider string,
	model string,
	options ExecutionProfileValidationOptions,
	path string,
) error {
	if (provider != "" || model != "") && !options.AllowProviderOverride {
		return fmt.Errorf("%w: %s provider/model override is disabled by config", ErrValidation, path)
	}
	if err := validateSelectorAtom(provider, nestedPath(path, "provider"), true); err != nil {
		return err
	}
	return validateSelectorAtom(model, nestedPath(path, "model"), true)
}

func validateAgentSelectors(exact string, allowed []string, preferred []string, role string) error {
	base := "task_execution_profile." + role
	if err := validateSelectorAtom(exact, nestedPath(base, "agent_name"), true); err != nil {
		return err
	}
	if err := validateProfileSelectorIDs(allowed, nestedPath(base, "allowed_agent_names")); err != nil {
		return err
	}
	if err := validateProfileSelectorIDs(preferred, nestedPath(base, "preferred_agent_names")); err != nil {
		return err
	}
	if exact != "" && !profileAgentAllowed(exact, allowed) {
		return fmt.Errorf("%w: %s.agent_name %q is outside allowed_agent_names", ErrValidation, base, exact)
	}
	return nil
}

func validateChannelSelectors(allowed []string, preferred []string, role string) error {
	base := "task_execution_profile." + role
	if err := validateProfileSelectorIDs(allowed, nestedPath(base, "allowed_channel_ids")); err != nil {
		return err
	}
	return validateProfileSelectorIDs(preferred, nestedPath(base, "preferred_channel_ids"))
}

func validatePeerSelectors(allowed []string, preferred []string, role string) error {
	base := "task_execution_profile." + role
	if err := validateProfileSelectorIDs(allowed, nestedPath(base, "allowed_peer_ids")); err != nil {
		return err
	}
	return validateProfileSelectorIDs(preferred, nestedPath(base, "preferred_peer_ids"))
}

func validateCapabilitySelectors(required []string, preferred []string, role string) error {
	base := "task_execution_profile." + role
	if err := ValidateCapabilityIDs(required, nestedPath(base, "required_capabilities")); err != nil {
		return err
	}
	return ValidateCapabilityIDs(preferred, nestedPath(base, "preferred_capabilities"))
}

func validateProfileSelectorIDs(values []string, path string) error {
	for idx, value := range values {
		if err := validateSelectorAtom(value, fmt.Sprintf("%s[%d]", path, idx), false); err != nil {
			return err
		}
	}
	return nil
}

func validateSelectorAtom(value string, path string, allowEmpty bool) error {
	if value == "" {
		if allowEmpty {
			return nil
		}
		return fmt.Errorf("%w: %s is required", ErrValidation, path)
	}
	if len(value) > profileSelectorMaxBytes {
		return fmt.Errorf("%w: %s exceeds %d bytes", ErrValidation, path, profileSelectorMaxBytes)
	}
	if strings.ContainsAny(value, "\t\n\r") {
		return fmt.Errorf("%w: %s must not contain control whitespace", ErrValidation, path)
	}
	return nil
}

func profileAgentAllowed(agentName string, allowed []string) bool {
	return len(allowed) == 0 || slices.Contains(allowed, agentName)
}

func coordinatorGuidanceMaxBytes(options ExecutionProfileValidationOptions) int {
	if options.MaxCoordinatorGuidanceBytes > 0 {
		return options.MaxCoordinatorGuidanceBytes
	}
	return defaultCoordinatorGuidanceMaxBytes
}
