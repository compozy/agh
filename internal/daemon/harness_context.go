package daemon

import (
	"errors"
	"fmt"
	"strings"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/session"
)

// TurnOrigin identifies the resolved harness origin for one turn.
type TurnOrigin string

const (
	// TurnOriginUser identifies a standard user-driven turn.
	TurnOriginUser TurnOrigin = "user"
	// TurnOriginNetwork identifies a network-originated turn.
	TurnOriginNetwork TurnOrigin = "network"
	// TurnOriginSynthetic identifies a daemon-owned synthetic turn.
	TurnOriginSynthetic TurnOrigin = "synthetic"
)

// SessionClass is the session-side harness axis derived from durable session
// metadata rather than a top-level profile enum.
type SessionClass string

const (
	// SessionClassInteractive identifies user-owned interactive sessions.
	SessionClassInteractive SessionClass = "interactive"
	// SessionClassDream identifies dream consolidation sessions.
	SessionClassDream SessionClass = "dream"
	// SessionClassSystem identifies daemon-owned system sessions.
	SessionClassSystem SessionClass = "system"
	// SessionClassCoordinator identifies daemon-owned workspace coordinator sessions.
	SessionClassCoordinator SessionClass = "coordinator"
	// SessionClassSpawned identifies bounded child worker sessions.
	SessionClassSpawned SessionClass = "spawned"
)

// HarnessPromptSection identifies a startup prompt section managed by harness policy.
type HarnessPromptSection string

const (
	// HarnessPromptSectionSituation injects the bounded AGH situation context.
	HarnessPromptSectionSituation HarnessPromptSection = "situation"
	// HarnessPromptSectionMemory injects durable memory prompt context.
	HarnessPromptSectionMemory HarnessPromptSection = "memory"
	// HarnessPromptSectionSkills injects the active skills catalog prompt context.
	HarnessPromptSectionSkills HarnessPromptSection = "skills"
	// HarnessPromptSectionNetwork injects the bundled AGH network startup section.
	HarnessPromptSectionNetwork HarnessPromptSection = "network"
)

// HarnessAugmenter identifies a prompt input augmenter managed by harness policy.
type HarnessAugmenter string

const (
	// HarnessAugmenterSituation injects fresh bounded AGH situation context.
	HarnessAugmenterSituation HarnessAugmenter = "situation"
	// HarnessAugmenterDurableMemory enables the durable memory recall augmenter.
	HarnessAugmenterDurableMemory HarnessAugmenter = "durable_memory"
)

// ReentryMode identifies how a resolved policy participates in synthetic reentry.
type ReentryMode string

const (
	// ReentryModeNone means the turn is not synthetic reentry.
	ReentryModeNone ReentryMode = "none"
	// ReentryModeSynthetic means the turn is a validated synthetic reentry path.
	ReentryModeSynthetic ReentryMode = "synthetic"
)

// DetachedRunMode identifies detached task-runtime behavior for the resolved policy.
type DetachedRunMode string

const (
	// DetachedRunModeNone means detached task-runtime behavior is not enabled.
	DetachedRunModeNone DetachedRunMode = "none"
	// DetachedRunModeTaskRuntime means detached work reuses the existing task runtime.
	DetachedRunModeTaskRuntime DetachedRunMode = "task_runtime"
)

// ResolutionSurface identifies which runtime seam is consuming the resolver.
type ResolutionSurface string

const (
	// ResolutionSurfaceStartup resolves startup prompt policy.
	ResolutionSurfaceStartup ResolutionSurface = "startup"
	// ResolutionSurfaceTurn resolves live prompt-turn policy.
	ResolutionSurfaceTurn ResolutionSurface = "turn"
)

// HarnessRuntimeSignals captures daemon-owned capability flags available to the resolver.
type HarnessRuntimeSignals struct {
	SituationPromptSectionEnabled bool
	MemoryPromptSectionEnabled    bool
	SkillsPromptSectionEnabled    bool
	SituationAugmenter            bool
	DurableMemoryAugmenter        bool
	SyntheticTurnsEnabled         bool
	DetachedTaskRuntimeEnabled    bool
}

// HarnessSessionInput carries durable session metadata into the resolver.
type HarnessSessionInput struct {
	Type        session.Type
	Channel     string
	WorkspaceID string
	Workspace   string
	AgentName   string
}

// SyntheticTurnMetadata carries validated daemon-only synthetic reentry metadata.
type SyntheticTurnMetadata struct {
	Reason      string
	Trigger     string
	SourceTask  string
	SourceRunID string
}

// DetachedRunMetadata carries detached task-runtime correlation fields.
type DetachedRunMetadata struct {
	TaskID    string
	TaskRunID string
}

// HarnessTurnRequest carries turn-time metadata into the resolver.
type HarnessTurnRequest struct {
	Source     session.TurnSource
	PromptMeta acp.PromptMeta
	Synthetic  *SyntheticTurnMetadata
	Detached   *DetachedRunMetadata
}

// HarnessResolutionInput captures one resolver request.
type HarnessResolutionInput struct {
	Surface ResolutionSurface
	Session HarnessSessionInput
	Turn    HarnessTurnRequest
}

// HarnessSessionContext is the normalized durable session context emitted by the resolver.
type HarnessSessionContext struct {
	Type         session.Type
	SessionClass SessionClass
	Channel      string
	ChannelBound bool
	WorkspaceID  string
	Workspace    string
	AgentName    string
}

// HarnessTurnContext is the normalized turn context emitted by the resolver.
type HarnessTurnContext struct {
	Origin     TurnOrigin
	PromptMeta acp.PromptMeta
	Synthetic  *SyntheticTurnMetadata
	Detached   *DetachedRunMetadata
}

// ResolvedHarnessPolicy is the authoritative daemon-owned harness runtime policy.
type ResolvedHarnessPolicy struct {
	SessionClass      SessionClass
	TurnOrigin        TurnOrigin
	IncludeSections   []HarnessPromptSection
	EnableAugmenters  []HarnessAugmenter
	ReentryMode       ReentryMode
	DetachedRunMode   DetachedRunMode
	DiagnosticLabel   string
	ObservabilityTags map[string]string
}

// ResolvedHarnessContext captures the normalized inputs plus the derived policy.
type ResolvedHarnessContext struct {
	Surface ResolutionSurface
	Session HarnessSessionContext
	Turn    HarnessTurnContext
	Policy  ResolvedHarnessPolicy
}

// HarnessContextResolver derives harness policy from durable session state plus
// turn-origin metadata.
type HarnessContextResolver struct {
	runtime HarnessRuntimeSignals
}

// NewHarnessContextResolver constructs a daemon-owned harness context resolver.
func NewHarnessContextResolver(runtime HarnessRuntimeSignals) *HarnessContextResolver {
	return &HarnessContextResolver{runtime: runtime}
}

// ResolveStartup resolves startup policy from durable session context.
func (r *HarnessContextResolver) ResolveStartup(startup session.StartupPromptContext) (ResolvedHarnessContext, error) {
	return r.Resolve(HarnessResolutionInput{
		Surface: ResolutionSurfaceStartup,
		Session: HarnessSessionInput{
			Type:        startup.SessionType,
			Channel:     startup.Channel,
			WorkspaceID: startup.WorkspaceID,
			Workspace:   startup.Workspace,
			AgentName:   startup.AgentName,
		},
		Turn: HarnessTurnRequest{
			Source: session.TurnSourceUser,
		},
	})
}

// ResolvePrompt resolves one live prompt-turn policy from session metadata.
func (r *HarnessContextResolver) ResolvePrompt(
	info *session.Info,
	source session.TurnSource,
	meta acp.PromptMeta,
) (ResolvedHarnessContext, error) {
	if info == nil {
		return ResolvedHarnessContext{}, errors.New("daemon: harness resolve prompt requires session info")
	}
	return r.Resolve(HarnessResolutionInput{
		Surface: ResolutionSurfaceTurn,
		Session: HarnessSessionInput{
			Type:        info.Type,
			Channel:     info.Channel,
			WorkspaceID: info.WorkspaceID,
			Workspace:   info.Workspace,
			AgentName:   info.AgentName,
		},
		Turn: HarnessTurnRequest{
			Source:     source,
			PromptMeta: meta,
		},
	})
}

// Resolve derives one authoritative harness context from explicit inputs.
func (r *HarnessContextResolver) Resolve(input HarnessResolutionInput) (ResolvedHarnessContext, error) {
	if r == nil {
		return ResolvedHarnessContext{}, errors.New("daemon: harness context resolver is required")
	}

	surface := normalizeResolutionSurface(input.Surface)
	if surface == "" {
		return ResolvedHarnessContext{}, fmt.Errorf("daemon: invalid harness resolution surface %q", input.Surface)
	}

	sessionCtx, err := normalizeHarnessSessionContext(input.Session)
	if err != nil {
		return ResolvedHarnessContext{}, err
	}

	turnCtx, err := r.normalizeHarnessTurnContext(sessionCtx, input.Turn)
	if err != nil {
		return ResolvedHarnessContext{}, err
	}

	policy := ResolvedHarnessPolicy{
		SessionClass:     sessionCtx.SessionClass,
		TurnOrigin:       turnCtx.Origin,
		IncludeSections:  r.resolveSections(sessionCtx),
		EnableAugmenters: r.resolveAugmenters(surface, turnCtx),
		ReentryMode:      r.resolveReentry(turnCtx),
		DetachedRunMode:  r.resolveDetachedRunMode(sessionCtx, turnCtx),
	}
	policy.DiagnosticLabel = buildHarnessDiagnosticLabel(sessionCtx, policy)
	policy.ObservabilityTags = buildHarnessObservabilityTags(surface, sessionCtx, turnCtx, policy)

	return ResolvedHarnessContext{
		Surface: surface,
		Session: sessionCtx,
		Turn:    turnCtx,
		Policy:  policy,
	}, nil
}

func normalizeResolutionSurface(surface ResolutionSurface) ResolutionSurface {
	switch ResolutionSurface(strings.TrimSpace(string(surface))) {
	case ResolutionSurfaceStartup:
		return ResolutionSurfaceStartup
	case "", ResolutionSurfaceTurn:
		return ResolutionSurfaceTurn
	default:
		return ""
	}
}

func normalizeHarnessSessionContext(input HarnessSessionInput) (HarnessSessionContext, error) {
	sessionType := normalizeHarnessSessionType(input.Type)
	if sessionType == "" {
		return HarnessSessionContext{}, fmt.Errorf("daemon: invalid harness session type %q", input.Type)
	}

	sessionClass, err := harnessSessionClassForType(sessionType)
	if err != nil {
		return HarnessSessionContext{}, err
	}

	channel := strings.TrimSpace(input.Channel)
	return HarnessSessionContext{
		Type:         sessionType,
		SessionClass: sessionClass,
		Channel:      channel,
		ChannelBound: channel != "",
		WorkspaceID:  strings.TrimSpace(input.WorkspaceID),
		Workspace:    strings.TrimSpace(input.Workspace),
		AgentName:    strings.TrimSpace(input.AgentName),
	}, nil
}

func normalizeHarnessSessionType(sessionType session.Type) session.Type {
	switch session.Type(strings.TrimSpace(string(sessionType))) {
	case session.SessionTypeUser:
		return session.SessionTypeUser
	case session.SessionTypeDream:
		return session.SessionTypeDream
	case session.SessionTypeSystem:
		return session.SessionTypeSystem
	case session.SessionTypeCoordinator:
		return session.SessionTypeCoordinator
	case session.SessionTypeSpawned:
		return session.SessionTypeSpawned
	default:
		return ""
	}
}

func harnessSessionClassForType(sessionType session.Type) (SessionClass, error) {
	switch sessionType {
	case session.SessionTypeUser:
		return SessionClassInteractive, nil
	case session.SessionTypeDream:
		return SessionClassDream, nil
	case session.SessionTypeSystem:
		return SessionClassSystem, nil
	case session.SessionTypeCoordinator:
		return SessionClassCoordinator, nil
	case session.SessionTypeSpawned:
		return SessionClassSpawned, nil
	default:
		return "", fmt.Errorf("daemon: unsupported harness session type %q", sessionType)
	}
}

func (r *HarnessContextResolver) normalizeHarnessTurnContext(
	sessionCtx HarnessSessionContext,
	request HarnessTurnRequest,
) (HarnessTurnContext, error) {
	normalizedMeta := request.PromptMeta.Normalize()
	origin, err := turnOriginFromSource(request.Source)
	if err != nil {
		return HarnessTurnContext{}, err
	}

	if request.Synthetic != nil && origin != TurnOriginSynthetic {
		return HarnessTurnContext{}, errors.New(
			"daemon: synthetic harness metadata requires the synthetic turn origin",
		)
	}

	detached, err := normalizeDetachedRunMetadata(request.Detached)
	if err != nil {
		return HarnessTurnContext{}, err
	}
	if origin == TurnOriginSynthetic {
		return r.normalizeSyntheticHarnessTurnContext(sessionCtx, normalizedMeta, request.Synthetic, detached)
	}

	normalizedMeta, err = normalizePromptMetaForHarnessTurn(origin, normalizedMeta)
	if err != nil {
		return HarnessTurnContext{}, err
	}
	return HarnessTurnContext{
		Origin:     origin,
		PromptMeta: normalizedMeta,
		Detached:   detached,
	}, nil
}

func normalizePromptMetaForHarnessTurn(
	origin TurnOrigin,
	meta acp.PromptMeta,
) (acp.PromptMeta, error) {
	switch origin {
	case TurnOriginUser:
		return normalizePromptMetaForExpectedTurnSource(origin, meta, acp.PromptTurnSourceUser)
	case TurnOriginNetwork:
		return normalizePromptMetaForExpectedTurnSource(origin, meta, acp.PromptTurnSourceNetwork)
	default:
		return acp.PromptMeta{}, fmt.Errorf("daemon: invalid harness turn origin %q", origin)
	}
}

func normalizePromptMetaForExpectedTurnSource(
	origin TurnOrigin,
	meta acp.PromptMeta,
	expected string,
) (acp.PromptMeta, error) {
	if meta.TurnSource == "" {
		meta.TurnSource = expected
	}
	if meta.TurnSource != expected {
		return acp.PromptMeta{}, fmt.Errorf(
			"daemon: harness turn origin %q does not match prompt metadata turn_source %q",
			origin,
			meta.TurnSource,
		)
	}
	if err := meta.Validate(); err != nil {
		return acp.PromptMeta{}, err
	}
	return meta, nil
}

func (r *HarnessContextResolver) normalizeSyntheticHarnessTurnContext(
	sessionCtx HarnessSessionContext,
	meta acp.PromptMeta,
	syntheticInput *SyntheticTurnMetadata,
	detached *DetachedRunMetadata,
) (HarnessTurnContext, error) {
	if !r.runtime.SyntheticTurnsEnabled {
		return HarnessTurnContext{}, errors.New("daemon: synthetic harness turns are not enabled")
	}
	if sessionCtx.Type != session.SessionTypeSystem {
		return HarnessTurnContext{}, fmt.Errorf(
			"daemon: synthetic harness turns require a system session, got %q",
			sessionCtx.Type,
		)
	}
	if meta.Network != nil {
		return HarnessTurnContext{}, errors.New(
			"daemon: synthetic harness turns cannot include network prompt metadata",
		)
	}
	synthetic, err := normalizeSyntheticTurnMetadata(syntheticInput)
	if err != nil {
		return HarnessTurnContext{}, err
	}
	meta.TurnSource = acp.PromptTurnSourceSynthetic
	return HarnessTurnContext{
		Origin:     TurnOriginSynthetic,
		PromptMeta: meta,
		Synthetic:  synthetic,
		Detached:   detached,
	}, nil
}

func turnOriginFromSource(source session.TurnSource) (TurnOrigin, error) {
	switch session.TurnSource(strings.TrimSpace(string(source))) {
	case "", session.TurnSourceUser:
		return TurnOriginUser, nil
	case session.TurnSourceNetwork:
		return TurnOriginNetwork, nil
	case session.TurnSourceSynthetic:
		return TurnOriginSynthetic, nil
	default:
		return "", fmt.Errorf("daemon: invalid harness turn origin %q", source)
	}
}

func normalizeSyntheticTurnMetadata(input *SyntheticTurnMetadata) (*SyntheticTurnMetadata, error) {
	if input == nil {
		return nil, errors.New("daemon: synthetic harness turns require runtime metadata")
	}

	normalized := &SyntheticTurnMetadata{
		Reason:      strings.TrimSpace(input.Reason),
		Trigger:     strings.TrimSpace(input.Trigger),
		SourceTask:  strings.TrimSpace(input.SourceTask),
		SourceRunID: strings.TrimSpace(input.SourceRunID),
	}
	if normalized.Reason == "" {
		return nil, errors.New("daemon: synthetic harness turns require a reason")
	}
	if normalized.Trigger == "" {
		return nil, errors.New("daemon: synthetic harness turns require a trigger")
	}
	return normalized, nil
}

func normalizeDetachedRunMetadata(input *DetachedRunMetadata) (*DetachedRunMetadata, error) {
	if input == nil {
		return nil, nil
	}

	normalized := &DetachedRunMetadata{
		TaskID:    strings.TrimSpace(input.TaskID),
		TaskRunID: strings.TrimSpace(input.TaskRunID),
	}
	if normalized.TaskID == "" && normalized.TaskRunID == "" {
		return nil, errors.New("daemon: detached harness metadata requires a task id or task run id")
	}
	return normalized, nil
}

func (r *HarnessContextResolver) resolveSections(sessionCtx HarnessSessionContext) []HarnessPromptSection {
	sections := make([]HarnessPromptSection, 0, 4)
	if r.runtime.SituationPromptSectionEnabled {
		sections = append(sections, HarnessPromptSectionSituation)
	}
	if r.runtime.MemoryPromptSectionEnabled {
		sections = append(sections, HarnessPromptSectionMemory)
	}
	if r.runtime.SkillsPromptSectionEnabled {
		sections = append(sections, HarnessPromptSectionSkills)
	}
	if sessionCtx.ChannelBound {
		sections = append(sections, HarnessPromptSectionNetwork)
	}
	return sections
}

func (r *HarnessContextResolver) resolveAugmenters(
	surface ResolutionSurface,
	turnCtx HarnessTurnContext,
) []HarnessAugmenter {
	if surface != ResolutionSurfaceTurn {
		return nil
	}
	if turnCtx.Origin != TurnOriginUser {
		return nil
	}
	augmenters := make([]HarnessAugmenter, 0, 2)
	if r.runtime.SituationAugmenter {
		augmenters = append(augmenters, HarnessAugmenterSituation)
	}
	if r.runtime.DurableMemoryAugmenter {
		augmenters = append(augmenters, HarnessAugmenterDurableMemory)
	}
	return augmenters
}

func (r *HarnessContextResolver) resolveReentry(turnCtx HarnessTurnContext) ReentryMode {
	if turnCtx.Origin == TurnOriginSynthetic {
		return ReentryModeSynthetic
	}
	return ReentryModeNone
}

func (r *HarnessContextResolver) resolveDetachedRunMode(
	sessionCtx HarnessSessionContext,
	turnCtx HarnessTurnContext,
) DetachedRunMode {
	if !r.runtime.DetachedTaskRuntimeEnabled {
		return DetachedRunModeNone
	}
	if turnCtx.Detached == nil {
		return DetachedRunModeNone
	}
	if sessionCtx.SessionClass == SessionClassSystem {
		return DetachedRunModeTaskRuntime
	}
	return DetachedRunModeNone
}

func buildHarnessDiagnosticLabel(
	sessionCtx HarnessSessionContext,
	policy ResolvedHarnessPolicy,
) string {
	parts := []string{string(sessionCtx.SessionClass)}
	if sessionCtx.ChannelBound {
		parts = append(parts, "channel")
	}
	parts = append(parts, string(policy.TurnOrigin))
	if policy.ReentryMode == ReentryModeSynthetic {
		parts = append(parts, "reentry")
	}
	return strings.Join(parts, ".")
}

func buildHarnessObservabilityTags(
	surface ResolutionSurface,
	sessionCtx HarnessSessionContext,
	turnCtx HarnessTurnContext,
	policy ResolvedHarnessPolicy,
) map[string]string {
	tags := map[string]string{
		"harness.surface":          string(surface),
		"harness.session_type":     string(sessionCtx.Type),
		"harness.session_class":    string(policy.SessionClass),
		"harness.turn_origin":      string(policy.TurnOrigin),
		"harness.channel_bound":    boolTag(sessionCtx.ChannelBound),
		"harness.diagnostic_label": policy.DiagnosticLabel,
	}
	if turnCtx.Synthetic != nil {
		tags["harness.synthetic_reason"] = turnCtx.Synthetic.Reason
		tags["harness.synthetic_trigger"] = turnCtx.Synthetic.Trigger
	}
	if turnCtx.Detached != nil {
		if turnCtx.Detached.TaskID != "" {
			tags["harness.task_id"] = turnCtx.Detached.TaskID
		}
		if turnCtx.Detached.TaskRunID != "" {
			tags["harness.task_run_id"] = turnCtx.Detached.TaskRunID
		}
	}
	return tags
}

func boolTag(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
