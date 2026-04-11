package protocol

import (
	"slices"
	"strings"
)

// HostAPIMethod identifies one extension -> AGH Host API request.
type HostAPIMethod string

const (
	// CapabilityProvideMemoryBackend is the provide surface for daemon-managed memory backends.
	CapabilityProvideMemoryBackend = "memory.backend"
	// CapabilityProvideChannelAdapter is the provide surface for channel-capable adapter extensions.
	CapabilityProvideChannelAdapter = "channel.adapter"
)

// ExtensionServiceMethod identifies one AGH -> extension capability service request.
type ExtensionServiceMethod string

const (
	ExtensionServiceMethodMemoryStore     ExtensionServiceMethod = "memory/store"
	ExtensionServiceMethodMemoryRecall    ExtensionServiceMethod = "memory/recall"
	ExtensionServiceMethodMemoryForget    ExtensionServiceMethod = "memory/forget"
	ExtensionServiceMethodChannelsDeliver ExtensionServiceMethod = "channels/deliver"
)

const (
	HostAPIMethodSessionsList                 HostAPIMethod = "sessions/list"
	HostAPIMethodSessionsCreate               HostAPIMethod = "sessions/create"
	HostAPIMethodSessionsPrompt               HostAPIMethod = "sessions/prompt"
	HostAPIMethodSessionsStop                 HostAPIMethod = "sessions/stop"
	HostAPIMethodSessionsStatus               HostAPIMethod = "sessions/status"
	HostAPIMethodSessionsEvents               HostAPIMethod = "sessions/events"
	HostAPIMethodMemoryRecall                 HostAPIMethod = "memory/recall"
	HostAPIMethodMemoryStore                  HostAPIMethod = "memory/store"
	HostAPIMethodMemoryForget                 HostAPIMethod = "memory/forget"
	HostAPIMethodObserveHealth                HostAPIMethod = "observe/health"
	HostAPIMethodObserveEvents                HostAPIMethod = "observe/events"
	HostAPIMethodSkillsList                   HostAPIMethod = "skills/list"
	HostAPIMethodAutomationJobs               HostAPIMethod = "automation/jobs"
	HostAPIMethodAutomationJobsGet            HostAPIMethod = "automation/jobs/get"
	HostAPIMethodAutomationJobsCreate         HostAPIMethod = "automation/jobs/create"
	HostAPIMethodAutomationJobsUpdate         HostAPIMethod = "automation/jobs/update"
	HostAPIMethodAutomationJobsDelete         HostAPIMethod = "automation/jobs/delete"
	HostAPIMethodAutomationJobsTrigger        HostAPIMethod = "automation/jobs/trigger"
	HostAPIMethodAutomationJobsRuns           HostAPIMethod = "automation/jobs/runs"
	HostAPIMethodAutomationTriggers           HostAPIMethod = "automation/triggers"
	HostAPIMethodAutomationTriggersGet        HostAPIMethod = "automation/triggers/get"
	HostAPIMethodAutomationTriggersCreate     HostAPIMethod = "automation/triggers/create"
	HostAPIMethodAutomationTriggersUpdate     HostAPIMethod = "automation/triggers/update"
	HostAPIMethodAutomationTriggersDelete     HostAPIMethod = "automation/triggers/delete"
	HostAPIMethodAutomationTriggersRuns       HostAPIMethod = "automation/triggers/runs"
	HostAPIMethodAutomationTriggersFire       HostAPIMethod = "automation/triggers/fire"
	HostAPIMethodAutomationRuns               HostAPIMethod = "automation/runs"
	HostAPIMethodChannelsMessagesIngest       HostAPIMethod = "channels/messages/ingest"
	HostAPIMethodChannelsInstancesGet         HostAPIMethod = "channels/instances/get"
	HostAPIMethodChannelsInstancesReportState HostAPIMethod = "channels/instances/report_state"
)

// AllHostAPIMethods returns the canonical Host API method registry in wire order.
func AllHostAPIMethods() []HostAPIMethod {
	return []HostAPIMethod{
		HostAPIMethodSessionsList,
		HostAPIMethodSessionsCreate,
		HostAPIMethodSessionsPrompt,
		HostAPIMethodSessionsStop,
		HostAPIMethodSessionsStatus,
		HostAPIMethodSessionsEvents,
		HostAPIMethodMemoryRecall,
		HostAPIMethodMemoryStore,
		HostAPIMethodMemoryForget,
		HostAPIMethodObserveHealth,
		HostAPIMethodObserveEvents,
		HostAPIMethodSkillsList,
		HostAPIMethodAutomationJobs,
		HostAPIMethodAutomationJobsGet,
		HostAPIMethodAutomationJobsCreate,
		HostAPIMethodAutomationJobsUpdate,
		HostAPIMethodAutomationJobsDelete,
		HostAPIMethodAutomationJobsTrigger,
		HostAPIMethodAutomationJobsRuns,
		HostAPIMethodAutomationTriggers,
		HostAPIMethodAutomationTriggersGet,
		HostAPIMethodAutomationTriggersCreate,
		HostAPIMethodAutomationTriggersUpdate,
		HostAPIMethodAutomationTriggersDelete,
		HostAPIMethodAutomationTriggersRuns,
		HostAPIMethodAutomationTriggersFire,
		HostAPIMethodAutomationRuns,
		HostAPIMethodChannelsMessagesIngest,
		HostAPIMethodChannelsInstancesGet,
		HostAPIMethodChannelsInstancesReportState,
	}
}

var capabilityServiceMethods = map[string][]ExtensionServiceMethod{
	CapabilityProvideMemoryBackend: {
		ExtensionServiceMethodMemoryStore,
		ExtensionServiceMethodMemoryRecall,
		ExtensionServiceMethodMemoryForget,
	},
	CapabilityProvideChannelAdapter: {
		ExtensionServiceMethodChannelsDeliver,
	},
}

// CapabilityServiceMethods returns the negotiated AGH -> extension service methods
// enabled by the declared provide surfaces.
func CapabilityServiceMethods(provides []string) []string {
	if len(provides) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	methods := make([]string, 0)
	for _, provide := range normalizeUniqueStrings(provides) {
		for _, method := range capabilityServiceMethods[provide] {
			name := strings.TrimSpace(string(method))
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			methods = append(methods, name)
		}
	}
	if len(methods) == 0 {
		return nil
	}
	slices.Sort(methods)
	return methods
}

func normalizeUniqueStrings(values []string) []string {
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
	if len(normalized) == 0 {
		return nil
	}
	slices.Sort(normalized)
	return normalized
}
