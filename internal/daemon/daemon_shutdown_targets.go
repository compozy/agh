package daemon

import (
	"context"

	"github.com/compozy/agh/internal/memory"
	"github.com/compozy/agh/internal/memory/consolidation"
	"github.com/compozy/agh/internal/resources"
)

type shutdownTargets struct {
	scheduler           *schedulerRuntime
	coordinator         *coordinatorRuntime
	spawnReaper         *spawnReaper
	tasks               *taskRuntime
	sessions            SessionManager
	network             networkRuntime
	hooks               hookRuntime
	extensions          extensionRuntime
	automation          automationRuntime
	resourceReconcile   resources.ReconcileDriver
	bridges             *bridgeRuntime
	httpServer          Server
	udsServer           Server
	registry            Registry
	lock                *Lock
	closeLogger         func() error
	infoPath            string
	dreamRuntime        *consolidation.Runtime
	memoryExtractor     *daemonMemoryExtractor
	memoryStore         *memory.Store
	localMemoryProvider memoryProviderShutdowner
	modelCatalog        *modelCatalogRuntime
	marketplace         *marketplaceRuntime
	skillsCancel        context.CancelFunc
	skillsDone          chan struct{}
	loopsCancel         context.CancelFunc
	loopsDone           chan struct{}
	goalOutboxCancel    context.CancelFunc
	goalOutboxDone      chan struct{}
	retention           observerRetentionStopper
}
