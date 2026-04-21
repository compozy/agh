package session

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/transcript"
)

// Transcript returns a canonical AI SDK replay transcript for the requested session.
func (m *Manager) Transcript(ctx context.Context, id string) ([]transcript.UIMessage, error) {
	recorder, cleanup, err := m.openQueryRecorder(ctx, id)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			logger := m.logger
			if logger == nil {
				logger = slog.Default()
			}
			logger.Warn("session: transcript cleanup failed", "session_id", strings.TrimSpace(id), "error", cleanupErr)
		}
	}()

	events, err := recorder.Query(ctx, store.EventQuery{})
	if err != nil {
		return nil, fmt.Errorf("session: query transcript events for %q: %w", strings.TrimSpace(id), err)
	}

	return transcript.ToUIMessages(events)
}
