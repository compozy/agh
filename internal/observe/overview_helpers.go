package observe

import (
	"context"
	"fmt"
	"time"

	"github.com/compozy/agh/internal/store"
)

const (
	daysPerWeek = 7
	hoursPerDay = 24

	freshnessStatusCurrent = "current"
	freshnessStatusEmpty   = "empty"
	freshnessStatusStale   = "stale"
)

func (o *Observer) overviewPulse(
	ctx context.Context,
	overviewStore OverviewStore,
	query OverviewQuery,
	now time.Time,
) (OverviewPulse, error) {
	since := store.LocalDayStart(now, overviewPulseWindowDays-1)

	counted, err := overviewStore.CountEventsByHourWeekday(ctx, store.OverviewSinceQuery{
		WorkspaceID: query.WorkspaceID,
		Since:       since,
	})
	if err != nil {
		return OverviewPulse{}, fmt.Errorf("observe: query pulse buckets: %w", err)
	}

	pulse := OverviewPulse{Buckets: zeroFilledPulseBuckets(counted)}
	pulse.Busiest = busiestPulseBucket(pulse.Buckets)

	longest, err := overviewStore.LongestUserSessionSince(ctx, store.OverviewSinceQuery{
		WorkspaceID: query.WorkspaceID,
		Since:       since,
	})
	if err != nil {
		return OverviewPulse{}, fmt.Errorf("observe: query longest session: %w", err)
	}
	if longest != nil && longest.RuntimeSeconds > 0 {
		pulse.LongestSession = &OverviewLongestSession{
			SessionID:       longest.SessionID,
			AgentName:       longest.AgentName,
			DurationSeconds: longest.RuntimeSeconds,
			Date:            store.LocalDay(longest.StartedAt),
		}
	}
	return pulse, nil
}

func zeroFilledPulseBuckets(counted []store.EventHourWeekdayBucket) []store.EventHourWeekdayBucket {
	buckets := make([]store.EventHourWeekdayBucket, 0, daysPerWeek*hoursPerDay)
	events := make(map[[2]int]int, len(counted))
	for _, bucket := range counted {
		if bucket.Weekday < 0 || bucket.Weekday >= daysPerWeek || bucket.Hour < 0 || bucket.Hour >= hoursPerDay {
			continue
		}
		events[[2]int{bucket.Weekday, bucket.Hour}] = bucket.Events
	}
	for weekday := range daysPerWeek {
		for hour := range hoursPerDay {
			buckets = append(buckets, store.EventHourWeekdayBucket{
				Weekday: weekday,
				Hour:    hour,
				Events:  events[[2]int{weekday, hour}],
			})
		}
	}
	return buckets
}

func busiestPulseBucket(buckets []store.EventHourWeekdayBucket) *store.EventHourWeekdayBucket {
	var busiest *store.EventHourWeekdayBucket
	for index := range buckets {
		if buckets[index].Events <= 0 {
			continue
		}
		if busiest == nil || buckets[index].Events > busiest.Events {
			busiest = &buckets[index]
		}
	}
	if busiest == nil {
		return nil
	}
	found := *busiest
	return &found
}

func (o *Observer) overviewFreshness(
	ctx context.Context,
	overviewStore OverviewStore,
	workspaceID string,
	now time.Time,
) (TaskDashboardFreshness, error) {
	latest, err := overviewStore.LatestEventSummaryAt(ctx, workspaceID)
	if err != nil {
		return TaskDashboardFreshness{}, fmt.Errorf("observe: query latest activity: %w", err)
	}

	hasLiveWork := false
	if o.sessionSource != nil {
		hasLiveWork = len(o.sessionSource.List()) > 0
	}

	staleAfter := max(o.taskDashboardConfig.staleAfter, 0)
	observedAt := now.UTC()
	age := safeSince(observedAt, latest)

	freshness := TaskDashboardFreshness{
		ObservedAt:       observedAt,
		LatestActivityAt: latest,
		AgeMilli:         age.Milliseconds(),
		StaleAfterMilli:  staleAfter.Milliseconds(),
		HasLiveWork:      hasLiveWork,
		Status:           freshnessStatusCurrent,
	}
	switch {
	case latest.IsZero():
		freshness.Status = freshnessStatusEmpty
	case hasLiveWork && staleAfter > 0 && age > staleAfter:
		freshness.Status = freshnessStatusStale
		freshness.Stale = true
	}
	return freshness, nil
}
