// Package events owns canonical runtime event names and metadata shared by
// producers, logs, notifications, and contract tests.
package events

import (
	"fmt"
	"slices"
	"strings"
)

const (
	ComponentBridge       = "bridge"
	ComponentConfig       = "config"
	ComponentHarness      = "harness"
	ComponentHook         = "hook"
	ComponentMemory       = "memory"
	ComponentNetwork      = "network"
	ComponentExtension    = "extension"
	ComponentProvider     = "provider"
	ComponentScheduler    = "scheduler"
	ComponentSession      = "session"
	ComponentSkill        = "skill"
	ComponentTask         = "task"
	ComponentTranscript   = "transcript"
	ComponentNotification = "notification"
)

// Outcome classifies an event for log filtering and notification policy.
type Outcome string

const (
	OutcomeInfo    Outcome = "info"
	OutcomeSuccess Outcome = "success"
	OutcomeFailure Outcome = "failure"
	OutcomeWarning Outcome = "warning"
)

// Metadata is the canonical registry entry for one event name.
type Metadata struct {
	Name                 string
	Family               string
	Component            string
	Outcome              Outcome
	EmitsToLogs          bool
	NotificationEligible bool
	GlobalScope          bool
}

const (
	ACPUserMessage      = "user_message"
	ACPSyntheticReentry = "synthetic_reentry"
	ACPAgentMessage     = "agent_message"
	ACPThought          = "thought"
	ACPToolCall         = "tool_call"
	ACPToolResult       = "tool_result"
	ACPPlan             = "plan"
	ACPPermission       = "permission"
	ACPUsage            = "usage"
	ACPSystem           = "system"
	ACPRuntimeProgress  = "runtime_progress"
	ACPRuntimeWarning   = "runtime_warning"
	ACPDone             = "done"
	ACPError            = "error"
	SessionStopped      = "session_stopped"
	SessionUnhealthy    = "session.unhealthy"
	SessionHung         = "session.hung"
	SessionRecovered    = "session.recovered"

	TaskCreated                 = "task.created"
	TaskUpdated                 = "task.updated"
	TaskPublished               = "task.published"
	TaskApproved                = "task.approved"
	TaskRejected                = "task.rejected"
	TaskCanceled                = "task.canceled"
	TaskChildCreated            = "task.child_created"
	TaskDependencyAdded         = "task.dependency_added"
	TaskDependencyRemoved       = "task.dependency_removed"
	TaskPaused                  = "task.paused"
	TaskResumed                 = "task.resumed"
	TaskRunEnqueued             = "task.run_enqueued"
	TaskRunClaimed              = "task.run_claimed"
	TaskRunStarting             = "task.run_starting"
	TaskRunSessionBound         = "task.run_session_bound"
	TaskRunStarted              = "task.run_started"
	TaskRunCompleted            = "task.run_completed"
	TaskRunFailed               = "task.run_failed"
	TaskRunCanceled             = "task.run_canceled"
	TaskRunForceStopped         = "task.run_force_stopped"
	TaskRunRecovered            = "task.run_recovered"
	TaskRunRejected             = "task.run_rejected"
	TaskRunLeaseExtended        = "task.run_lease_extended"
	TaskRunLeaseExpired         = "task.run_lease_expired"
	TaskRunReleased             = "task.run_released"
	TaskRunOperatorForcedFail   = "task.run_operator_forced_fail"
	TaskRunOperatorRetry        = "task.run_operator_retry"
	TaskExecutionProfileUpdated = "task.execution_profile_updated"
	TaskExecutionProfileDeleted = "task.execution_profile_deleted"
	TaskRunReviewRequested      = "task.run_review_requested"
	TaskRunReviewBound          = "task.run_review_bound"
	TaskRunReviewRecorded       = "task.run_review_recorded"
	TaskRunReviewApproved       = "task.run_review_approved"
	TaskRunReviewRejected       = "task.run_review_rejected"
	TaskRunReviewBlocked        = "task.run_review_blocked"
	TaskRunReviewError          = "task.run_review_error"
	TaskRunReviewTimeout        = "task.run_review_timeout"
	TaskRunReviewInvalidOutput  = "task.run_review_invalid_output"
	TaskRunReviewRetryEnqueued  = "task.run_review_retry_enqueued"

	SettingsChanged = "settings.changed"

	SkillShadowed   = "skill.shadowed"
	SkillLoadFailed = "skills.load_failed"

	HookDispatchStart    = "hook.dispatch.start"
	HookDispatchComplete = "hook.dispatch.complete"

	HarnessContextResolved         = "harness.context_resolved"
	HarnessSectionSelected         = "harness.section_selected"
	HarnessAugmenterApplied        = "harness.augmenter_applied"
	HarnessAugmenterFailed         = "harness.augmenter_failed"
	HarnessDetachedRunCompleted    = "harness.detached_run_completed"
	HarnessSyntheticReentryEmitted = "harness.synthetic_reentry_emitted"
	HarnessSyntheticReentryDropped = "harness.synthetic_reentry_dropped"

	MemoryWriteCommitted     = "memory.write.committed"
	MemoryWriteRejected      = "memory.write.rejected"
	MemoryWriteShadowed      = "memory.write.shadowed"
	MemoryWriteReindex       = "memory.write.reindex"
	MemoryWriteReverted      = "memory.write.reverted"
	MemoryProviderCollision  = "memory.provider.collision"
	MemoryRecallExecuted     = "memory.recall.executed"
	MemoryRecallSkipped      = "memory.recall.skipped"
	MemoryRecallDropped      = "memory.recall.signal_dropped"
	MemoryRecallFailed       = "memory.recall.signal_update_failed"
	MemoryDecisionsSummary   = "memory.decisions.audit_summarized"
	MemoryDecisionsPruned    = "memory.decisions.pruned"
	MemoryDreamStarted       = "memory.dream.run.started"
	MemoryDreamPromoted      = "memory.dream.run.promoted"
	MemoryDreamFailed        = "memory.dream.run.failed"
	MemoryExtractorStarted   = "memory.extractor.started"
	MemoryExtractorComplete  = "memory.extractor.completed"
	MemoryExtractorFailed    = "memory.extractor.failed"
	MemoryExtractorCoalesced = "memory.extractor.coalesced"
	MemoryExtractorDropped   = "memory.extractor.dropped"
	MemoryDailyRotated       = "memory.daily.rotated"
	MemoryDailyArchived      = "memory.daily.archived"
	MemoryDailyRestored      = "memory.daily.restored"
	MemoryDailyPurged        = "memory.daily.purged"
	MemoryDailyArchivePurged = "memory.daily.archive_purged"
	MemoryProviderEnabled    = "memory.provider.enabled"
	MemoryProviderDisabled   = "memory.provider.disabled"
	MemoryWorkspaceRelocated = "memory.workspace.relocated"
	MemoryWorkspaceRecovered = "memory.workspace.recovered"
	MemoryAgentPurged        = "memory.agent.purged"
	MemoryMigrationApplied   = "memory.migration.applied"

	SchedulerPaused         = "scheduler.paused"
	SchedulerResumed        = "scheduler.resumed"
	SchedulerDrainStarted   = "scheduler.drain_started"
	SchedulerDrainCompleted = "scheduler.drain_completed"

	TranscriptMarkerCreated  = "transcript_marker.created"
	TranscriptMarkerRedacted = "transcript_marker.redacted"

	ProviderAuthRequired          = "provider.auth_required"
	ProviderAuthRecovered         = "provider.auth_recovered"
	ProviderRateLimited           = "provider.rate_limited"
	ProviderPermissionDenied      = "provider.permission_denied"
	ProviderUnavailable           = "provider.unavailable"
	ProviderModelCatalogRefreshed = "provider.model_catalog_refreshed"

	ExtensionInstalled = "extension.installed"
	ExtensionUpdated   = "extension.updated"
	ExtensionRemoved   = "extension.removed"
	ExtensionEnabled   = "extension.enabled"
	ExtensionDisabled  = "extension.disabled"

	BridgeNotificationSuppressed = "bridge_notification_suppressed"
	NetworkPeerJoined            = "network.peer.joined"
	NetworkPeerLeft              = "network.peer.left"

	NotificationPresetCreated        = "notification.preset_created"
	NotificationPresetUpdated        = "notification.preset_updated"
	NotificationPresetDeleted        = "notification.preset_deleted"
	NotificationPresetDispatchFailed = "notification.preset_dispatch_failed"
)

var registryEntries = []Metadata{
	info(ACPUserMessage, "session", ComponentSession),
	info(ACPSyntheticReentry, "session", ComponentSession),
	info(ACPAgentMessage, "session", ComponentSession),
	info(ACPThought, "session", ComponentSession),
	info(ACPToolCall, "session", ComponentSession),
	info(ACPToolResult, "session", ComponentSession),
	info(ACPPlan, "session", ComponentSession),
	info(ACPPermission, "session", ComponentSession),
	info(ACPUsage, "session", ComponentSession),
	info(ACPSystem, "session", ComponentSession),
	info(ACPRuntimeProgress, "session", ComponentSession),
	warning(ACPRuntimeWarning, "session", ComponentSession),
	success(ACPDone, "session", ComponentSession),
	failure(ACPError, "session", ComponentSession),
	info(SessionStopped, "session", ComponentSession),
	notify(warning(SessionUnhealthy, "session", ComponentSession)),
	notify(warning(SessionHung, "session", ComponentSession)),
	notify(success(SessionRecovered, "session", ComponentSession)),

	info(TaskCreated, "task", ComponentTask),
	info(TaskUpdated, "task", ComponentTask),
	info(TaskPublished, "task", ComponentTask),
	success(TaskApproved, "task", ComponentTask),
	warning(TaskRejected, "task", ComponentTask),
	warning(TaskCanceled, "task", ComponentTask),
	info(TaskChildCreated, "task", ComponentTask),
	info(TaskDependencyAdded, "task", ComponentTask),
	info(TaskDependencyRemoved, "task", ComponentTask),
	warning(TaskPaused, "task", ComponentTask),
	info(TaskResumed, "task", ComponentTask),
	info(TaskRunEnqueued, "task", ComponentTask),
	info(TaskRunClaimed, "task", ComponentTask),
	info(TaskRunStarting, "task", ComponentTask),
	info(TaskRunSessionBound, "task", ComponentTask),
	success(TaskRunStarted, "task", ComponentTask),
	notify(success(TaskRunCompleted, "task", ComponentTask)),
	notify(failure(TaskRunFailed, "task", ComponentTask)),
	notify(warning(TaskRunCanceled, "task", ComponentTask)),
	warning(TaskRunForceStopped, "task", ComponentTask),
	warning(TaskRunRecovered, "task", ComponentTask),
	warning(TaskRunRejected, "task", ComponentTask),
	info(TaskRunLeaseExtended, "task", ComponentTask),
	warning(TaskRunLeaseExpired, "task", ComponentTask),
	info(TaskRunReleased, "task", ComponentTask),
	notify(failure(TaskRunOperatorForcedFail, "task", ComponentTask)),
	info(TaskRunOperatorRetry, "task", ComponentTask),
	info(TaskExecutionProfileUpdated, "task", ComponentTask),
	info(TaskExecutionProfileDeleted, "task", ComponentTask),
	info(TaskRunReviewRequested, "task", ComponentTask),
	info(TaskRunReviewBound, "task", ComponentTask),
	info(TaskRunReviewRecorded, "task", ComponentTask),
	notify(success(TaskRunReviewApproved, "task", ComponentTask)),
	notify(failure(TaskRunReviewRejected, "task", ComponentTask)),
	notify(warning(TaskRunReviewBlocked, "task", ComponentTask)),
	notify(failure(TaskRunReviewError, "task", ComponentTask)),
	notify(warning(TaskRunReviewTimeout, "task", ComponentTask)),
	notify(failure(TaskRunReviewInvalidOutput, "task", ComponentTask)),
	info(TaskRunReviewRetryEnqueued, "task", ComponentTask),

	global(info(SettingsChanged, "settings", ComponentConfig)),
	global(warning(SkillShadowed, "skill", ComponentSkill)),
	global(failure(SkillLoadFailed, "skills", ComponentSkill)),
	global(info(HookDispatchStart, "hook.dispatch", ComponentHook)),
	global(info(HookDispatchComplete, "hook.dispatch", ComponentHook)),

	info(HarnessContextResolved, "harness", ComponentHarness),
	info(HarnessSectionSelected, "harness", ComponentHarness),
	info(HarnessAugmenterApplied, "harness", ComponentHarness),
	warning(HarnessAugmenterFailed, "harness", ComponentHarness),
	success(HarnessDetachedRunCompleted, "harness", ComponentHarness),
	info(HarnessSyntheticReentryEmitted, "harness", ComponentHarness),
	warning(HarnessSyntheticReentryDropped, "harness", ComponentHarness),

	success(MemoryWriteCommitted, "memory.write", ComponentMemory),
	warning(MemoryWriteRejected, "memory.write", ComponentMemory),
	warning(MemoryWriteShadowed, "memory.write", ComponentMemory),
	info(MemoryWriteReindex, "memory.write", ComponentMemory),
	warning(MemoryWriteReverted, "memory.write", ComponentMemory),
	success(MemoryRecallExecuted, "memory.recall", ComponentMemory),
	info(MemoryRecallSkipped, "memory.recall", ComponentMemory),
	warning(MemoryRecallDropped, "memory.recall", ComponentMemory),
	failure(MemoryRecallFailed, "memory.recall", ComponentMemory),
	success(MemoryDecisionsSummary, "memory.decisions", ComponentMemory),
	info(MemoryDecisionsPruned, "memory.decisions", ComponentMemory),
	info(MemoryDreamStarted, "memory.dream", ComponentMemory),
	success(MemoryDreamPromoted, "memory.dream", ComponentMemory),
	failure(MemoryDreamFailed, "memory.dream", ComponentMemory),
	info(MemoryExtractorStarted, "memory.extractor", ComponentMemory),
	success(MemoryExtractorComplete, "memory.extractor", ComponentMemory),
	failure(MemoryExtractorFailed, "memory.extractor", ComponentMemory),
	info(MemoryExtractorCoalesced, "memory.extractor", ComponentMemory),
	warning(MemoryExtractorDropped, "memory.extractor", ComponentMemory),
	success(MemoryDailyRotated, "memory.daily", ComponentMemory),
	success(MemoryDailyArchived, "memory.daily", ComponentMemory),
	success(MemoryDailyRestored, "memory.daily", ComponentMemory),
	warning(MemoryDailyPurged, "memory.daily", ComponentMemory),
	warning(MemoryDailyArchivePurged, "memory.daily", ComponentMemory),
	success(MemoryProviderEnabled, "memory.provider", ComponentMemory),
	warning(MemoryProviderDisabled, "memory.provider", ComponentMemory),
	global(warning(MemoryProviderCollision, "memory.provider", ComponentMemory)),
	info(MemoryWorkspaceRelocated, "memory.workspace", ComponentMemory),
	success(MemoryWorkspaceRecovered, "memory.workspace", ComponentMemory),
	warning(MemoryAgentPurged, "memory.agent", ComponentMemory),
	success(MemoryMigrationApplied, "memory.migration", ComponentMemory),

	notify(global(warning(SchedulerPaused, "scheduler", ComponentScheduler))),
	global(info(SchedulerResumed, "scheduler", ComponentScheduler)),
	global(info(SchedulerDrainStarted, "scheduler", ComponentScheduler)),
	global(success(SchedulerDrainCompleted, "scheduler", ComponentScheduler)),

	info(TranscriptMarkerCreated, "transcript_marker", ComponentTranscript),
	warning(TranscriptMarkerRedacted, "transcript_marker", ComponentTranscript),

	notify(global(warning(ProviderAuthRequired, "provider", ComponentProvider))),
	global(success(ProviderAuthRecovered, "provider", ComponentProvider)),
	notify(global(warning(ProviderRateLimited, "provider", ComponentProvider))),
	notify(global(failure(ProviderPermissionDenied, "provider", ComponentProvider))),
	notify(global(failure(ProviderUnavailable, "provider", ComponentProvider))),
	global(success(ProviderModelCatalogRefreshed, "provider", ComponentProvider)),

	notify(global(success(ExtensionInstalled, "extension", ComponentExtension))),
	notify(global(success(ExtensionUpdated, "extension", ComponentExtension))),
	notify(global(warning(ExtensionRemoved, "extension", ComponentExtension))),
	global(success(ExtensionEnabled, "extension", ComponentExtension)),
	global(warning(ExtensionDisabled, "extension", ComponentExtension)),

	global(success(NotificationPresetCreated, "notification.preset", ComponentNotification)),
	global(info(NotificationPresetUpdated, "notification.preset", ComponentNotification)),
	global(warning(NotificationPresetDeleted, "notification.preset", ComponentNotification)),
	global(failure(NotificationPresetDispatchFailed, "notification.preset", ComponentNotification)),
	notify(global(warning(BridgeNotificationSuppressed, "bridge_notification", ComponentNotification))),
	notify(global(success(NetworkPeerJoined, "network.peer", ComponentNetwork))),
	notify(global(warning(NetworkPeerLeft, "network.peer", ComponentNetwork))),
}

var registryByName = mustBuildRegistry(registryEntries)

// All returns all canonical registry entries sorted by event name.
func All() []Metadata {
	entries := append([]Metadata(nil), registryEntries...)
	slices.SortFunc(entries, func(a Metadata, b Metadata) int {
		return strings.Compare(a.Name, b.Name)
	})
	return entries
}

// Lookup returns metadata for a canonical event name.
func Lookup(name string) (Metadata, bool) {
	meta, ok := registryByName[strings.TrimSpace(name)]
	return meta, ok
}

// ComponentFor returns the registered component for an event name.
func ComponentFor(name string) string {
	meta, ok := Lookup(name)
	if !ok {
		return ""
	}
	return meta.Component
}

// OutcomeFor returns the registered outcome for an event name, defaulting to info.
func OutcomeFor(name string) Outcome {
	meta, ok := Lookup(name)
	if !ok {
		return OutcomeInfo
	}
	return meta.Outcome
}

// AllowsGlobalScope reports whether a summary event may be emitted without a session.
func AllowsGlobalScope(name string) bool {
	meta, ok := Lookup(name)
	return ok && meta.GlobalScope
}

// ValidatePublicName rejects deleted or unsupported public event families.
func ValidatePublicName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil
	}
	if strings.HasPrefix(trimmed, "task_run.") {
		return fmt.Errorf("events: %q is not a public event family; use task.run_* events", trimmed)
	}
	return nil
}

// ValidOutcome reports whether value is one of the canonical event outcomes.
func ValidOutcome(value string) bool {
	switch Outcome(strings.TrimSpace(value)) {
	case "", OutcomeInfo, OutcomeSuccess, OutcomeFailure, OutcomeWarning:
		return true
	default:
		return false
	}
}

// ValidComponent reports whether component is present in the registry.
func ValidComponent(component string) bool {
	component = strings.TrimSpace(component)
	if component == "" {
		return true
	}
	for _, meta := range registryEntries {
		if meta.Component == component {
			return true
		}
	}
	return false
}

// NamesForComponent returns canonical event names registered for component.
func NamesForComponent(component string) []string {
	component = strings.TrimSpace(component)
	if component == "" {
		return nil
	}
	names := make([]string, 0)
	for _, meta := range registryEntries {
		if meta.Component == component {
			names = append(names, meta.Name)
		}
	}
	slices.Sort(names)
	return names
}

// NamesForOutcomes returns canonical event names matching any requested outcome.
func NamesForOutcomes(outcomes ...Outcome) []string {
	if len(outcomes) == 0 {
		return nil
	}
	allowed := make(map[Outcome]struct{}, len(outcomes))
	for _, outcome := range outcomes {
		allowed[outcome] = struct{}{}
	}
	names := make([]string, 0)
	for _, meta := range registryEntries {
		if _, ok := allowed[meta.Outcome]; ok {
			names = append(names, meta.Name)
		}
	}
	slices.Sort(names)
	return names
}

func mustBuildRegistry(entries []Metadata) map[string]Metadata {
	registry := make(map[string]Metadata, len(entries))
	for _, entry := range entries {
		name := strings.TrimSpace(entry.Name)
		if name == "" {
			panic("events: registry entry missing name")
		}
		if _, exists := registry[name]; exists {
			panic("events: duplicate registry entry " + name)
		}
		if strings.TrimSpace(entry.Family) == "" {
			panic("events: registry entry missing family for " + name)
		}
		if strings.TrimSpace(entry.Component) == "" {
			panic("events: registry entry missing component for " + name)
		}
		if !ValidOutcome(string(entry.Outcome)) || entry.Outcome == "" {
			panic("events: registry entry has invalid outcome for " + name)
		}
		entry.Name = name
		entry.Family = strings.TrimSpace(entry.Family)
		entry.Component = strings.TrimSpace(entry.Component)
		registry[name] = entry
	}
	return registry
}

func info(name string, family string, component string) Metadata {
	return metadata(name, family, component, OutcomeInfo)
}

func success(name string, family string, component string) Metadata {
	return metadata(name, family, component, OutcomeSuccess)
}

func failure(name string, family string, component string) Metadata {
	return metadata(name, family, component, OutcomeFailure)
}

func warning(name string, family string, component string) Metadata {
	return metadata(name, family, component, OutcomeWarning)
}

func metadata(name string, family string, component string, outcome Outcome) Metadata {
	return Metadata{
		Name:        name,
		Family:      family,
		Component:   component,
		Outcome:     outcome,
		EmitsToLogs: true,
	}
}

func notify(entry Metadata) Metadata {
	entry.NotificationEligible = true
	return entry
}

func global(entry Metadata) Metadata {
	entry.GlobalScope = true
	return entry
}
