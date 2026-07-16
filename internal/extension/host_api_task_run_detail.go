package extensionpkg

import (
	apicontract "github.com/compozy/agh/internal/api/contract"
	taskpkg "github.com/compozy/agh/internal/task"
)

func taskRunDetailPayloadFromView(view *taskpkg.RunDetailView) apicontract.TaskRunDetailPayload {
	if view == nil {
		return apicontract.TaskRunDetailPayload{}
	}

	var task *apicontract.TaskReferencePayload
	if view.Task != nil {
		payload := taskReferencePayloadFromReference(*view.Task)
		task = &payload
	}

	return apicontract.TaskRunDetailPayload{
		Run:     taskRunPayloadFromRun(&view.Run),
		Task:    task,
		Session: taskRunSessionPayloadFromSession(view.Session),
		Summary: taskRunOperationalSummaryPayloadFromSummary(view.Summary),
	}
}
