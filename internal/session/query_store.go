package session

import (
	"context"
	"time"

	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/store/sessiondb"
)

const defaultReadOnlyQueryStoreTTL = 30 * time.Second

func newDefaultQueryStoreOpener() StoreOpener {
	pool := sessiondb.NewReadOnlyPool(sessiondb.ReadOnlyPoolConfig{
		TTL: defaultReadOnlyQueryStoreTTL,
		Open: func(ctx context.Context, sessionID string, path string) (store.EventRecorder, error) {
			return sessiondb.OpenSessionDBReadOnly(ctx, sessionID, path)
		},
	})
	return pool.Open
}
