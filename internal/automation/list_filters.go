package automation

import "strings"

func jobMatchesLoopName(job Job, loopName string) bool {
	if !job.IsLoopTarget() || job.LoopTarget == nil {
		return false
	}
	return strings.TrimSpace(job.LoopTarget.LoopName) == strings.TrimSpace(loopName)
}

func triggerMatchesLoopName(trigger Trigger, loopName string) bool {
	if !trigger.IsLoopTarget() || trigger.LoopTarget == nil {
		return false
	}
	return strings.TrimSpace(trigger.LoopTarget.LoopName) == strings.TrimSpace(loopName)
}
