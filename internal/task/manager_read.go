package task

import (
	"context"

	"fmt"

	"strings"
)

// GetTask returns one expanded task view after enforcing read authority.
func (m *Service) GetTask(ctx context.Context, id string, actor ActorContext) (*View, error) {
	if err := requireReadAuthority(actor); err != nil {
		return nil, err
	}

	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: task id is required", ErrValidation)
	}

	record, err := m.store.GetTask(ctx, trimmedID)
	if err != nil {
		return nil, err
	}

	children, err := m.listTaskSummaries(ctx, Query{ParentTaskID: trimmedID})
	if err != nil {
		return nil, err
	}
	dependencies, err := m.store.ListDependencies(ctx, trimmedID)
	if err != nil {
		return nil, err
	}
	runs, err := m.store.ListTaskRuns(ctx, RunQuery{TaskID: trimmedID})
	if err != nil {
		return nil, err
	}
	events, err := m.store.ListTaskEvents(ctx, EventQuery{TaskID: trimmedID})
	if err != nil {
		return nil, err
	}

	summary, err := m.enrichTaskSummaryFromState(
		ctx,
		record,
		len(children),
		dependencies,
		runs,
		events,
	)
	if err != nil {
		return nil, err
	}
	dependencyReferences, err := m.buildDependencyReferences(ctx, dependencies)
	if err != nil {
		return nil, err
	}

	view := &View{
		Summary:              summary,
		Task:                 record,
		Children:             children,
		Dependencies:         dependencies,
		DependencyReferences: dependencyReferences,
		Runs:                 runs,
		Events:               events,
	}
	view.Task.Status = summary.Status
	return view, nil
}

// ListTaskRuns returns task runs for one task after enforcing read authority and
// task existence.
func (m *Service) ListTaskRuns(
	ctx context.Context,
	taskID string,
	query RunQuery,
	actor ActorContext,
) ([]Run, error) {
	if err := requireReadAuthority(actor); err != nil {
		return nil, err
	}

	trimmedID := strings.TrimSpace(taskID)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: task id is required", ErrValidation)
	}

	if _, err := m.store.GetTask(ctx, trimmedID); err != nil {
		return nil, err
	}

	normalizedQuery := query
	normalizedQuery.TaskID = trimmedID
	return m.store.ListTaskRuns(ctx, normalizedQuery)
}

// ListTasks returns task summaries that satisfy the supplied query filters
// after enforcing read authority.
func (m *Service) ListTasks(
	ctx context.Context,
	query Query,
	actor ActorContext,
) ([]Summary, error) {
	if err := requireReadAuthority(actor); err != nil {
		return nil, err
	}
	return m.listTaskSummaries(ctx, query)
}

func (m *Service) listTaskSummaries(ctx context.Context, query Query) ([]Summary, error) {
	summaries, err := m.store.ListTasks(ctx, query)
	if err != nil {
		return nil, err
	}

	enriched := make([]Summary, 0, len(summaries))
	for idx := range summaries {
		item, err := m.enrichTaskSummary(ctx, &summaries[idx])
		if err != nil {
			return nil, err
		}
		enriched = append(enriched, item)
	}
	return enriched, nil
}

func (m *Service) enrichTaskSummary(ctx context.Context, summary *Summary) (Summary, error) {
	childCount, err := m.store.CountDirectChildren(ctx, summary.ID)
	if err != nil {
		return Summary{}, err
	}
	dependencies, err := m.store.ListDependencies(ctx, summary.ID)
	if err != nil {
		return Summary{}, err
	}
	runs, err := m.store.ListTaskRuns(ctx, RunQuery{TaskID: summary.ID})
	if err != nil {
		return Summary{}, err
	}
	events, err := m.store.ListTaskEvents(ctx, EventQuery{TaskID: summary.ID, Limit: 1})
	if err != nil {
		return Summary{}, err
	}
	return m.enrichTaskSummaryFromState(
		ctx,
		taskRecordFromSummary(summary),
		childCount,
		dependencies,
		runs,
		events,
	)
}

func (m *Service) enrichTaskSummaryFromState(
	ctx context.Context,
	record Task,
	childCount int,
	dependencies []Dependency,
	runs []Run,
	events []Event,
) (Summary, error) {
	status, err := m.canonicalTaskStatus(ctx, record, dependencies, runs)
	if err != nil {
		return Summary{}, err
	}

	summary := summaryFromTaskRecord(record)
	summary.Status = status
	summary.Draft = status == TaskStatusDraft
	summary.ChildCount = ClampSummaryCount(childCount)
	summary.DependencyCount = ClampSummaryCount(len(dependencies))
	summary.ActiveRun = activeRunSummary(runs, record.MaxAttempts)
	summary.LastActivityAt = latestTaskActivityAt(record, runs, events)
	blockedReasons, err := m.blockedReasonsForSnapshot(ctx, record, dependencies)
	if err != nil {
		return Summary{}, err
	}
	if len(blockedReasons) > 0 {
		summary.BlockedReasons = &blockedReasons
	}
	if err := m.applyEffectivePauseToSummary(ctx, &summary); err != nil {
		return Summary{}, err
	}
	summary.Dependencies, err = m.buildDependencyReferences(ctx, dependencies)
	if err != nil {
		return Summary{}, err
	}
	return summary, nil
}

func (m *Service) buildDependencyReferences(
	ctx context.Context,
	dependencies []Dependency,
) ([]DependencyReference, error) {
	refs := make([]DependencyReference, 0, len(dependencies))
	for _, dependency := range dependencies {
		dependsOn, err := m.taskReference(ctx, dependency.DependsOnTaskID)
		if err != nil {
			return nil, err
		}
		refs = append(refs, DependencyReference{
			TaskID:          dependency.TaskID,
			DependsOnTaskID: dependency.DependsOnTaskID,
			Kind:            dependency.Kind,
			CreatedAt:       dependency.CreatedAt,
			DependsOn:       dependsOn,
		})
	}
	return refs, nil
}
