package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/transcript"
)

// Transcript returns a canonical AI SDK replay transcript for the requested session.
func (m *Manager) Transcript(ctx context.Context, id string) ([]transcript.UIMessage, error) {
	target := strings.TrimSpace(id)
	var err error
	for attempt := range 2 {
		recorder, cleanup, openErr := m.openQueryRecorder(ctx, target)
		if openErr != nil {
			return nil, openErr
		}

		var events []store.SessionEvent
		events, err = recorder.Query(ctx, store.EventQuery{})
		m.logTranscriptCleanupError(ctx, target, cleanup())
		if err == nil {
			return transcript.ToUIMessages(events)
		}
		if errors.Is(err, store.ErrClosed) && attempt == 0 {
			if _, waitErr := m.waitForSessionFinalization(ctx, target); waitErr != nil {
				return nil, fmt.Errorf(
					"session: wait for finalization for %q after closed transcript recorder: %w",
					target,
					waitErr,
				)
			}
			continue
		}
		return nil, fmt.Errorf("session: query transcript events for %q: %w", target, err)
	}
	return nil, fmt.Errorf("session: query transcript events for %q: %w", target, err)
}

func (m *Manager) logTranscriptCleanupError(ctx context.Context, sessionID string, err error) {
	if err == nil {
		return
	}
	logger := m.logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.WarnContext(ctx, "session: transcript cleanup failed", "session_id", sessionID, "error", err)
}
