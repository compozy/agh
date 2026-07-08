package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	eventspkg "github.com/compozy/agh/internal/events"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/transcript"
)

type transcriptCacheEntry struct {
	mu              sync.Mutex
	fold            *transcript.Fold
	entries         []transcript.Entry
	cursor          int64
	watermark       int64
	watermarkReason string
}

const (
	// TranscriptWatermarkReasonBelowWindowMutation marks snapshots that must reset after unseen earlier events changed.
	TranscriptWatermarkReasonBelowWindowMutation = "below_window_mutation"
	// TranscriptWatermarkReasonCacheRebuild marks snapshots after an in-memory transcript cache rebuild.
	TranscriptWatermarkReasonCacheRebuild = "cache_rebuild"
)

// TranscriptWatermark identifies the highest sequence that requires reconnecting clients to replay a snapshot.
type TranscriptWatermark struct {
	Sequence int64
	Reason   string
}

// Transcript returns a sequence-bearing canonical AI SDK replay transcript for the requested session.
func (m *Manager) Transcript(ctx context.Context, id string, query store.EventQuery) ([]transcript.Entry, error) {
	target := strings.TrimSpace(id)
	if err := query.Validate(); err != nil {
		return nil, err
	}
	if !canUseTranscriptCache(query) {
		return m.assembleTranscriptQuery(ctx, target, query)
	}
	return m.cachedTranscriptEntries(ctx, target, query)
}

func canUseTranscriptCache(query store.EventQuery) bool {
	return strings.TrimSpace(query.Type) == "" &&
		strings.TrimSpace(query.AgentName) == "" &&
		strings.TrimSpace(query.TurnID) == "" &&
		query.Since.IsZero()
}

func (m *Manager) assembleTranscriptQuery(
	ctx context.Context,
	id string,
	query store.EventQuery,
) ([]transcript.Entry, error) {
	events, err := m.queryTranscriptEvents(ctx, id, query)
	if err != nil {
		return nil, err
	}
	return transcript.ToUIEntries(events)
}

func (m *Manager) cachedTranscriptEntries(
	ctx context.Context,
	id string,
	query store.EventQuery,
) ([]transcript.Entry, error) {
	target := strings.TrimSpace(id)
	m.transcriptCacheMu.Lock()
	if m.transcriptCache == nil {
		m.transcriptCache = make(map[string]*transcriptCacheEntry)
	}
	cache := m.transcriptCache[target]
	rebuilt := false
	if cache == nil {
		cache = &transcriptCacheEntry{fold: transcript.NewFold()}
		m.transcriptCache[target] = cache
		rebuilt = true
	}
	m.transcriptCacheMu.Unlock()

	cache.mu.Lock()
	eventQuery := store.EventQuery{}
	if !rebuilt {
		eventQuery.AfterSequence = cache.cursor
	}
	events, err := m.queryTranscriptEvents(ctx, target, eventQuery)
	if err != nil {
		cache.mu.Unlock()
		if rebuilt {
			m.dropTranscriptCacheEntry(target, cache)
		}
		return nil, err
	}
	changed, err := cache.fold.Apply(events)
	if err != nil {
		cache.mu.Unlock()
		if rebuilt {
			m.dropTranscriptCacheEntry(target, cache)
		}
		return nil, err
	}
	if maxSequence := maxTranscriptEventSequence(events); maxSequence > cache.cursor {
		cache.cursor = maxSequence
	}
	if rebuilt && cache.cursor > cache.watermark {
		cache.watermark = cache.cursor
		cache.watermarkReason = TranscriptWatermarkReasonCacheRebuild
	}
	if rebuilt || len(changed) > 0 {
		cache.entries = cache.fold.Entries()
	}
	entries := filterTranscriptEntries(cache.entries, query)
	cursor := cache.cursor
	eventCount := len(events)
	cache.mu.Unlock()

	if rebuilt {
		m.logSessionTranscriptCacheRebuilt(ctx, target, entries, cursor, eventCount)
	}
	return entries, nil
}

func (m *Manager) TranscriptWatermark(_ context.Context, id string) TranscriptWatermark {
	target := strings.TrimSpace(id)
	if target == "" {
		return TranscriptWatermark{}
	}
	m.transcriptCacheMu.Lock()
	if m.transcriptCache == nil {
		m.transcriptCacheMu.Unlock()
		return TranscriptWatermark{}
	}
	cache := m.transcriptCache[target]
	m.transcriptCacheMu.Unlock()
	if cache == nil {
		return TranscriptWatermark{}
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return TranscriptWatermark{
		Sequence: cache.watermark,
		Reason:   cache.watermarkReason,
	}
}

func (m *Manager) markTranscriptBelowWindowMutation(sessionID string, sequence int64) {
	target := strings.TrimSpace(sessionID)
	if target == "" {
		return
	}
	m.transcriptCacheMu.Lock()
	if m.transcriptCache == nil {
		m.transcriptCacheMu.Unlock()
		return
	}
	cache := m.transcriptCache[target]
	m.transcriptCacheMu.Unlock()
	if cache == nil {
		return
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	next := sequence
	if next <= 0 || next < cache.cursor {
		next = cache.cursor
	}
	if next > cache.watermark {
		cache.watermark = next
	}
	cache.watermarkReason = TranscriptWatermarkReasonBelowWindowMutation
}

func (m *Manager) queryTranscriptEvents(
	ctx context.Context,
	id string,
	query store.EventQuery,
) (events []store.SessionEvent, err error) {
	target := strings.TrimSpace(id)
	for attempt := range 2 {
		recorder, cleanup, openErr := m.openQueryRecorder(ctx, target)
		if openErr != nil {
			return nil, openErr
		}

		events, err = recorder.Query(ctx, query)
		m.logTranscriptCleanupError(ctx, target, cleanup())
		if err == nil {
			return events, nil
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

func filterTranscriptEntries(
	entries []transcript.Entry,
	query store.EventQuery,
) []transcript.Entry {
	filtered := make([]transcript.Entry, 0, len(entries))
	for _, entry := range entries {
		if query.AfterSequence > 0 && entry.Sequence <= query.AfterSequence {
			continue
		}
		if query.BeforeSequence > 0 && entry.Sequence >= query.BeforeSequence {
			continue
		}
		filtered = append(filtered, entry)
	}
	if query.Limit > 0 && len(filtered) > query.Limit {
		filtered = filtered[len(filtered)-query.Limit:]
	}
	return transcript.CloneEntries(filtered)
}

func maxTranscriptEventSequence(events []store.SessionEvent) int64 {
	var maxSequence int64
	for _, event := range events {
		if event.Sequence > maxSequence {
			maxSequence = event.Sequence
		}
	}
	return maxSequence
}

func (m *Manager) invalidateTranscriptCache(sessionID string) {
	target := strings.TrimSpace(sessionID)
	if target == "" {
		return
	}
	m.transcriptCacheMu.Lock()
	defer m.transcriptCacheMu.Unlock()
	if m.transcriptCache == nil {
		return
	}
	delete(m.transcriptCache, target)
}

func (m *Manager) dropTranscriptCacheEntry(sessionID string, cache *transcriptCacheEntry) {
	m.transcriptCacheMu.Lock()
	defer m.transcriptCacheMu.Unlock()
	if m.transcriptCache[sessionID] == cache {
		delete(m.transcriptCache, sessionID)
	}
}

func (m *Manager) logSessionTranscriptCacheRebuilt(
	ctx context.Context,
	sessionID string,
	entries []transcript.Entry,
	cursor int64,
	eventCount int,
) {
	logger := m.logger
	if logger == nil {
		logger = slog.Default()
	}
	workspaceID := ""
	if info, err := m.Status(ctx, sessionID); err == nil && info != nil {
		workspaceID = info.WorkspaceID
	}
	logger.InfoContext(
		ctx,
		eventspkg.SessionTranscriptCacheRebuilt,
		"workspace_id", workspaceID,
		"session_id", sessionID,
		"cursor", cursor,
		"entry_count", len(entries),
		"event_count", eventCount,
	)
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
