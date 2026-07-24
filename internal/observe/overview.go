package observe

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/store/globaldb"
)

// OverviewStore is the aggregate persistence capability behind the home overview.
type OverviewStore interface {
	UpsertTokenUsageDaily(ctx context.Context, update store.TokenUsageDailyUpdate) error
	ListTokenUsageByDay(ctx context.Context, query store.OverviewDayQuery) ([]store.TokenUsageDay, error)
	ListTokenUsageByAgent(ctx context.Context, query store.OverviewDayQuery) ([]store.TokenUsageAgentTotal, error)
	SumTokenUsageCost(ctx context.Context, query store.OverviewDayQuery) ([]store.TokenUsageCostGroup, error)
	CountTaskRunOutcomesByDay(ctx context.Context, query store.OverviewSinceQuery) ([]store.TaskRunOutcomeDay, error)
	CountTasksClosedByDay(ctx context.Context, query store.OverviewSinceQuery) ([]store.TaskClosedDay, error)
	CountEventsByHourWeekday(
		ctx context.Context,
		query store.OverviewSinceQuery,
	) ([]store.EventHourWeekdayBucket, error)
	LatestEventSummaryAt(ctx context.Context, workspaceID string) (time.Time, error)
	CountNetworkMessagesSince(ctx context.Context, query store.OverviewSinceQuery) (int, error)
	CountHookDispatchesSince(ctx context.Context, query store.OverviewSinceQuery) (store.HookDispatchCounts, error)
	LongestUserSessionSince(ctx context.Context, query store.OverviewSinceQuery) (*store.LongestSessionSample, error)
}

var _ OverviewStore = (*globaldb.GlobalDB)(nil)

// QueryObserveOverview composes the workspace-scoped home overview read model.
func (o *Observer) QueryObserveOverview(ctx context.Context, query OverviewQuery) (OverviewView, error) {
	if ctx == nil {
		return OverviewView{}, errors.New("observe: overview context is required")
	}
	if err := query.Validate(); err != nil {
		return OverviewView{}, err
	}
	overviewStore, ok := o.registry.(OverviewStore)
	if !ok {
		return OverviewView{}, errors.New("observe: overview store is not configured")
	}

	now := o.now()
	todayStart := store.LocalDayStart(now, 0)

	attention, err := o.overviewAttention(ctx, query)
	if err != nil {
		return OverviewView{}, err
	}

	outcomeDays, err := overviewStore.CountTaskRunOutcomesByDay(ctx, store.OverviewSinceQuery{
		WorkspaceID: query.WorkspaceID,
		Since:       store.LocalDayStart(now, overviewOutcomeWindowDays-1),
	})
	if err != nil {
		return OverviewView{}, fmt.Errorf("observe: query run outcomes: %w", err)
	}
	closedToday, err := overviewStore.CountTasksClosedByDay(ctx, store.OverviewSinceQuery{
		WorkspaceID: query.WorkspaceID,
		Since:       todayStart,
	})
	if err != nil {
		return OverviewView{}, fmt.Errorf("observe: query tasks closed today: %w", err)
	}

	usage, err := o.overviewUsage(ctx, overviewStore, query, now)
	if err != nil {
		return OverviewView{}, err
	}

	pulse, err := o.overviewPulse(ctx, overviewStore, query, now)
	if err != nil {
		return OverviewView{}, err
	}

	messagesToday, err := overviewStore.CountNetworkMessagesSince(ctx, store.OverviewSinceQuery{
		WorkspaceID: query.WorkspaceID,
		Since:       todayStart,
	})
	if err != nil {
		return OverviewView{}, fmt.Errorf("observe: count network messages today: %w", err)
	}

	hooks, err := overviewStore.CountHookDispatchesSince(ctx, store.OverviewSinceQuery{
		WorkspaceID: query.WorkspaceID,
		Since:       todayStart,
	})
	if err != nil {
		return OverviewView{}, fmt.Errorf("observe: count hook dispatches today: %w", err)
	}

	freshness, err := o.overviewFreshness(ctx, overviewStore, query.WorkspaceID, now)
	if err != nil {
		return OverviewView{}, err
	}

	return OverviewView{
		GeneratedAt: now.UTC(),
		Attention:   attention,
		Today:       overviewToday(outcomeDays, closedToday, store.LocalDay(now)),
		Outcomes:    overviewOutcomes(outcomeDays),
		Usage:       usage,
		Pulse:       pulse,
		Network:     OverviewNetwork{MessagesToday: messagesToday},
		System: OverviewSystem{
			HookRunsToday:     hooks.Runs,
			HookFailuresToday: hooks.Failures,
			RetentionDays:     o.retention.RetentionDays,
		},
		Freshness: freshness,
	}, nil
}

func overviewToday(
	outcomeDays []store.TaskRunOutcomeDay,
	closedToday []store.TaskClosedDay,
	today string,
) OverviewToday {
	summary := OverviewToday{}
	for _, day := range outcomeDays {
		if day.Day != today {
			continue
		}
		summary.RunsCompleted = day.Completed
		summary.RunsFailed = day.Failed
	}
	for _, day := range closedToday {
		if day.Day != today {
			continue
		}
		summary.TasksClosed = day.Closed
	}
	return summary
}

func overviewOutcomes(days []store.TaskRunOutcomeDay) OverviewOutcomes {
	outcomes := OverviewOutcomes{Days: days}
	for _, day := range days {
		outcomes.Completed += day.Completed
		outcomes.Failed += day.Failed
		outcomes.Canceled += day.Canceled
	}
	total := outcomes.Completed + outcomes.Failed + outcomes.Canceled
	if total > 0 {
		outcomes.SuccessPct = 100 * float64(outcomes.Completed) / float64(total)
	}
	return outcomes
}
