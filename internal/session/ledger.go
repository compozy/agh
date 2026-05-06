package session

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/store"
)

func (m *Manager) materializeSessionLedger(ctx context.Context, session *Session) error {
	if m == nil || m.ledgerMaterializer == nil || session == nil {
		return nil
	}

	info := session.Info()
	if info == nil {
		return nil
	}

	ledgerCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), defaultLifecycleTimeout)
	defer cancel()

	record := store.SessionLedgerRecord{
		SessionID:    strings.TrimSpace(info.ID),
		WorkspaceID:  strings.TrimSpace(info.WorkspaceID),
		AgentName:    strings.TrimSpace(info.AgentName),
		SessionType:  strings.TrimSpace(string(info.Type)),
		EventsDBPath: strings.TrimSpace(session.DBPath()),
		Lineage:      store.NormalizeSessionLineage(info.ID, info.Lineage),
		StartedAt:    normalizeLedgerTime(info.CreatedAt),
		EndedAt:      normalizeLedgerTime(info.UpdatedAt),
	}
	if err := m.ledgerMaterializer.MaterializeSessionLedger(ledgerCtx, record); err != nil {
		return fmt.Errorf("session: materialize ledger for %q: %w", info.ID, err)
	}
	return nil
}

func normalizeLedgerTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	return value.UTC()
}
