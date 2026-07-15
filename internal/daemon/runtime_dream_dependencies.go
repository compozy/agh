package daemon

import (
	"time"

	"github.com/compozy/agh/internal/memory"
	"github.com/compozy/agh/internal/memory/consolidation"
)

func initializeDreamRuntime(state *bootState, sessions SessionManager) {
	if state == nil || state.dreamSvc == nil {
		return
	}
	lockPath := memory.ConsolidationLockPath(state.globalMemoryDir)
	state.dreamRuntime = consolidation.NewRuntime(
		state.cfg.Memory.Dream.Enabled,
		state.dreamSvc,
		consolidation.NewSessionSpawner(
			sessions,
			state.workspaceResolver,
			&state.cfg,
		),
		state.cfg.Memory.Dream.CheckInterval,
		state.logger,
		func() (time.Time, error) {
			return memory.NewConsolidationLock(lockPath).LastConsolidatedAt()
		},
	)
}
