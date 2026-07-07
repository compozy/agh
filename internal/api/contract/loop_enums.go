package contract

// LoopRunStatusValues returns the closed public loop run status vocabulary.
func LoopRunStatusValues() []string {
	return []string{
		string(LoopRunStatusQueued),
		string(LoopRunStatusRunning),
		string(LoopRunStatusWatching),
		string(LoopRunStatusNeedsApproval),
		string(LoopRunStatusPaused),
		string(LoopRunStatusDone),
		string(LoopRunStatusNoOp),
		string(LoopRunStatusBlocked),
		string(LoopRunStatusFailed),
		string(LoopRunStatusExhausted),
		string(LoopRunStatusStalled),
	}
}

// LoopRunLiveStatusValues returns the non-terminal loop run statuses.
func LoopRunLiveStatusValues() []string {
	return []string{
		string(LoopRunStatusQueued),
		string(LoopRunStatusRunning),
		string(LoopRunStatusWatching),
		string(LoopRunStatusNeedsApproval),
		string(LoopRunStatusPaused),
	}
}

// LoopRunTerminalStatusValues returns the terminal loop run statuses.
func LoopRunTerminalStatusValues() []string {
	return []string{
		string(LoopRunStatusDone),
		string(LoopRunStatusNoOp),
		string(LoopRunStatusBlocked),
		string(LoopRunStatusFailed),
		string(LoopRunStatusExhausted),
		string(LoopRunStatusStalled),
	}
}

// LoopRunEventKindValues returns the closed public loop run event vocabulary.
func LoopRunEventKindValues() []string {
	return []string{
		string(LoopRunEventNodeRunning),
		string(LoopRunEventNodeSucceeded),
		string(LoopRunEventNodeFailed),
		string(LoopRunEventGateVerdict),
		string(LoopRunEventGenerationStarted),
		string(LoopRunEventChannelMsg),
		string(LoopRunEventTokenTick),
		string(LoopRunEventNeedsApproval),
		string(LoopRunEventStatusChanged),
	}
}

// LoopRunLifecycleEventKindValues returns event kinds that mutate durable run state.
func LoopRunLifecycleEventKindValues() []string {
	return []string{
		string(LoopRunEventStatusChanged),
		string(LoopRunEventNodeRunning),
		string(LoopRunEventNodeSucceeded),
		string(LoopRunEventNodeFailed),
		string(LoopRunEventGateVerdict),
		string(LoopRunEventGenerationStarted),
		string(LoopRunEventNeedsApproval),
	}
}
