package globaldb

import (
	"context"
	"database/sql"
	"time"
)

const notificationRollbackTimeout = 5 * time.Second

func notificationRollbackContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(parent), notificationRollbackTimeout)
}

func rollbackNotificationImmediate(parent context.Context, target *error, conn *sql.Conn, action string) {
	rollbackCtx, rollbackCancel := notificationRollbackContext(parent)
	defer rollbackCancel()
	joinCleanupError(target, rollbackImmediate(rollbackCtx, conn, action))
}
