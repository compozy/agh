package daemon

import (
	"context"
	"sync"
	"time"

	looppkg "github.com/compozy/agh/internal/loop"
	taskpkg "github.com/compozy/agh/internal/task"
)

const defaultWatchEventsGapScanPageSize = 100

type loopWatchEventsGapWakeStore interface {
	EnqueueWatchEventsGapWakesPage(
		context.Context,
		taskpkg.Origin,
		time.Time,
		looppkg.ParkedWatchEventScanCursor,
		int,
	) ([]taskpkg.Run, looppkg.ParkedWatchEventScanCursor, error)
}

type loopWatchEventsGapScanState struct {
	mu     sync.Mutex
	cursor looppkg.ParkedWatchEventScanCursor
}

func newLoopWatchEventsGapScanState() *loopWatchEventsGapScanState {
	return &loopWatchEventsGapScanState{}
}

func (s schedulerTaskSource) enqueueWatchEventsGapWakes(
	ctx context.Context,
	origin taskpkg.Origin,
	now time.Time,
) error {
	if s.watchEventsGapScan == nil {
		return nil
	}
	gapWakes, ok := s.store.(loopWatchEventsGapWakeStore)
	if !ok {
		return nil
	}
	s.watchEventsGapScan.mu.Lock()
	defer s.watchEventsGapScan.mu.Unlock()
	_, next, err := gapWakes.EnqueueWatchEventsGapWakesPage(
		ctx,
		origin,
		now,
		s.watchEventsGapScan.cursor,
		defaultWatchEventsGapScanPageSize,
	)
	s.watchEventsGapScan.cursor = next
	return err
}
