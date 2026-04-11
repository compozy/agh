package protocol

// HostAPIMethod identifies one extension -> AGH Host API request.
type HostAPIMethod string

const (
	HostAPIMethodSessionsList   HostAPIMethod = "sessions/list"
	HostAPIMethodSessionsCreate HostAPIMethod = "sessions/create"
	HostAPIMethodSessionsPrompt HostAPIMethod = "sessions/prompt"
	HostAPIMethodSessionsStop   HostAPIMethod = "sessions/stop"
	HostAPIMethodSessionsStatus HostAPIMethod = "sessions/status"
	HostAPIMethodSessionsEvents HostAPIMethod = "sessions/events"
	HostAPIMethodMemoryRecall   HostAPIMethod = "memory/recall"
	HostAPIMethodMemoryStore    HostAPIMethod = "memory/store"
	HostAPIMethodMemoryForget   HostAPIMethod = "memory/forget"
	HostAPIMethodObserveHealth  HostAPIMethod = "observe/health"
	HostAPIMethodObserveEvents  HostAPIMethod = "observe/events"
	HostAPIMethodSkillsList     HostAPIMethod = "skills/list"
)
