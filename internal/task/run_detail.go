package task

import (
	"context"
	"strings"
)

// RunDetail returns one persisted run and its task reference when the run is task-backed.
func (m *Service) RunDetail(
	ctx context.Context,
	runID string,
	actor ActorContext,
) (*RunDetailView, error) {
	if err := requireReadAuthority(actor); err != nil {
		return nil, err
	}

	run, err := m.loadRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	taskReference, err := m.runDetailTaskReference(ctx, run)
	if err != nil {
		return nil, err
	}

	session := baseRunSessionRef(run.SessionID)
	summary := RunOperationalSummary{}
	if m.runtimeViews != nil && strings.TrimSpace(run.SessionID) != "" {
		if enriched, runtimeErr := m.runtimeViews.GetSession(ctx, run.SessionID); runtimeErr == nil && enriched != nil {
			session = enriched
		}
		summary = m.bestEffortRunOperationalSummary(ctx, run.SessionID)
	}

	return &RunDetailView{
		Run:     run,
		Task:    taskReference,
		Session: session,
		Summary: summary,
	}, nil
}

func (m *Service) runDetailTaskReference(ctx context.Context, run Run) (*Reference, error) {
	if strings.TrimSpace(run.TaskID) == "" {
		return nil, nil
	}

	taskRecord, err := m.store.GetTask(ctx, run.TaskID)
	if err != nil {
		return nil, err
	}
	dependencies, err := m.store.ListDependencies(ctx, taskRecord.ID)
	if err != nil {
		return nil, err
	}
	runs, err := m.store.ListTaskRuns(ctx, RunQuery{TaskID: taskRecord.ID})
	if err != nil {
		return nil, err
	}
	status, err := m.canonicalTaskStatus(ctx, taskRecord, dependencies, runs)
	if err != nil {
		return nil, err
	}

	reference := taskReferenceFromTask(taskRecord, status)
	return &reference, nil
}
