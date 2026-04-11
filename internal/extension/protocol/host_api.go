package protocol

// HostAPIMethod identifies one extension -> AGH Host API request.
type HostAPIMethod string

const (
	HostAPIMethodSessionsList             HostAPIMethod = "sessions/list"
	HostAPIMethodSessionsCreate           HostAPIMethod = "sessions/create"
	HostAPIMethodSessionsPrompt           HostAPIMethod = "sessions/prompt"
	HostAPIMethodSessionsStop             HostAPIMethod = "sessions/stop"
	HostAPIMethodSessionsStatus           HostAPIMethod = "sessions/status"
	HostAPIMethodSessionsEvents           HostAPIMethod = "sessions/events"
	HostAPIMethodMemoryRecall             HostAPIMethod = "memory/recall"
	HostAPIMethodMemoryStore              HostAPIMethod = "memory/store"
	HostAPIMethodMemoryForget             HostAPIMethod = "memory/forget"
	HostAPIMethodObserveHealth            HostAPIMethod = "observe/health"
	HostAPIMethodObserveEvents            HostAPIMethod = "observe/events"
	HostAPIMethodSkillsList               HostAPIMethod = "skills/list"
	HostAPIMethodAutomationJobs           HostAPIMethod = "automation/jobs"
	HostAPIMethodAutomationJobsGet        HostAPIMethod = "automation/jobs/get"
	HostAPIMethodAutomationJobsCreate     HostAPIMethod = "automation/jobs/create"
	HostAPIMethodAutomationJobsUpdate     HostAPIMethod = "automation/jobs/update"
	HostAPIMethodAutomationJobsDelete     HostAPIMethod = "automation/jobs/delete"
	HostAPIMethodAutomationJobsTrigger    HostAPIMethod = "automation/jobs/trigger"
	HostAPIMethodAutomationJobsRuns       HostAPIMethod = "automation/jobs/runs"
	HostAPIMethodAutomationTriggers       HostAPIMethod = "automation/triggers"
	HostAPIMethodAutomationTriggersGet    HostAPIMethod = "automation/triggers/get"
	HostAPIMethodAutomationTriggersCreate HostAPIMethod = "automation/triggers/create"
	HostAPIMethodAutomationTriggersUpdate HostAPIMethod = "automation/triggers/update"
	HostAPIMethodAutomationTriggersDelete HostAPIMethod = "automation/triggers/delete"
	HostAPIMethodAutomationTriggersRuns   HostAPIMethod = "automation/triggers/runs"
	HostAPIMethodAutomationTriggersFire   HostAPIMethod = "automation/triggers/fire"
	HostAPIMethodAutomationRuns           HostAPIMethod = "automation/runs"
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
	}
}
