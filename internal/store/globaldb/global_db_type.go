package globaldb

import (
	"context"
	"database/sql"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	taskpkg "github.com/compozy/agh/internal/task"
)

// GlobalDB owns the global session index and observability database.
type GlobalDB struct {
	db     *sql.DB
	path   string
	now    func() time.Time
	closed atomic.Int32

	taskEventObserverMu sync.RWMutex
	taskEventObserver   taskpkg.EventObserver
}

var _ taskpkg.EventCommitObserverStore = (*GlobalDB)(nil)

// SetTaskEventCommitObserver installs the single task service that owns live
// fanout for this database.
func (g *GlobalDB) SetTaskEventCommitObserver(observer taskpkg.EventObserver) {
	if g == nil {
		return
	}
	g.taskEventObserverMu.Lock()
	defer g.taskEventObserverMu.Unlock()
	g.taskEventObserver = observer
}

func (g *GlobalDB) notifyCommittedTaskEvents(ctx context.Context, records []taskpkg.EventRecord) {
	if g == nil || len(records) == 0 {
		return
	}
	g.taskEventObserverMu.RLock()
	observer := g.taskEventObserver
	g.taskEventObserverMu.RUnlock()
	if observer == nil {
		return
	}
	for _, record := range records {
		notifyCommittedTaskEvent(ctx, observer, record)
	}
}

func notifyCommittedTaskEvent(ctx context.Context, observer taskpkg.EventObserver, record taskpkg.EventRecord) {
	defer func() {
		if recovered := recover(); recovered != nil {
			slog.Error(
				"store: task event commit observer panicked",
				"panic", recovered,
				"event_id", record.Event.ID,
				"task_id", record.Event.TaskID,
				"event_type", record.Event.EventType,
			)
		}
	}()
	observer.OnTaskEvent(ctx, record)
}
