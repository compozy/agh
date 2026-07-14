package daemon

import (
	"strings"

	taskpkg "github.com/compozy/agh/internal/task"
)

type taskRunListInput struct {
	TaskID               string `json:"task_id"`
	Status               string `json:"status,omitempty"`
	SessionID            string `json:"session_id,omitempty"`
	ParticipationChannel string `json:"participation_channel,omitempty"`
	Limit                int    `json:"limit,omitempty"`
}

func (i taskRunListInput) query() taskpkg.RunQuery {
	limit := i.Limit
	if strings.TrimSpace(i.ParticipationChannel) != "" {
		limit = 0
	}
	return taskpkg.RunQuery{
		TaskID:    strings.TrimSpace(i.TaskID),
		Status:    taskpkg.ParseRunStatus(i.Status),
		SessionID: strings.TrimSpace(i.SessionID),
		Limit:     limit,
	}
}

func (i taskRunListInput) filterRuns(runs []taskpkg.Run) []taskpkg.Run {
	channel := strings.TrimSpace(i.ParticipationChannel)
	if channel != "" {
		filtered := make([]taskpkg.Run, 0, len(runs))
		for index := range runs {
			if strings.TrimSpace(runs[index].NetworkSpecSnapshot().ChannelID) == channel {
				filtered = append(filtered, runs[index])
			}
		}
		runs = filtered
	}
	if i.Limit > 0 && len(runs) > i.Limit {
		return runs[:i.Limit]
	}
	return runs
}
