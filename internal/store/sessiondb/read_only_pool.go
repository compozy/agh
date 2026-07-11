package sessiondb

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/transcript"
)

const defaultReadOnlyPoolTTL = 30 * time.Second

// ReadOnlyPoolOpener opens the recorder stored behind a pooled read-only lease.
type ReadOnlyPoolOpener func(ctx context.Context, sessionID string, path string) (store.EventRecorder, error)

// ReadOnlyPoolConfig customizes read-only recorder pooling.
type ReadOnlyPoolConfig struct {
	TTL  time.Duration
	Now  func() time.Time
	Open ReadOnlyPoolOpener
}

// ReadOnlyPool reuses short-lived read-only session database handles for hot inactive sessions.
type ReadOnlyPool struct {
	mu      sync.Mutex
	ttl     time.Duration
	now     func() time.Time
	open    ReadOnlyPoolOpener
	entries map[readOnlyPoolKey]*readOnlyPoolEntry
	closed  bool
}

type readOnlyPoolKey struct {
	sessionID string
	path      string
}

type readOnlyPoolEntry struct {
	recorder  store.EventRecorder
	refs      int
	expiresAt time.Time
}

// NewReadOnlyPool constructs a read-only session recorder pool.
func NewReadOnlyPool(config ReadOnlyPoolConfig) *ReadOnlyPool {
	ttl := config.TTL
	if ttl <= 0 {
		ttl = defaultReadOnlyPoolTTL
	}
	now := config.Now
	if now == nil {
		now = func() time.Time {
			return time.Now().UTC()
		}
	}
	open := config.Open
	if open == nil {
		open = func(ctx context.Context, sessionID string, path string) (store.EventRecorder, error) {
			return OpenSessionDBReadOnly(ctx, sessionID, path)
		}
	}
	return &ReadOnlyPool{
		ttl:     ttl,
		now:     now,
		open:    open,
		entries: make(map[readOnlyPoolKey]*readOnlyPoolEntry),
	}
}

// Open returns a lease for a session-keyed read-only recorder.
func (p *ReadOnlyPool) Open(ctx context.Context, sessionID string, path string) (store.EventRecorder, error) {
	if p == nil {
		return nil, errors.New("store: read-only pool is required")
	}
	if ctx == nil {
		return nil, errors.New("store: open read-only pool context is required")
	}
	key, err := normalizeReadOnlyPoolKey(sessionID, path)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil, errors.New("store: read-only pool is closed")
	}
	if err := p.closeExpiredLocked(ctx, p.now()); err != nil {
		return nil, err
	}
	if entry := p.entries[key]; entry != nil {
		entry.refs++
		entry.expiresAt = time.Time{}
		return newReadOnlyPoolLease(p, key, entry), nil
	}

	recorder, err := p.open(ctx, key.sessionID, key.path)
	if err != nil {
		return nil, err
	}
	entry := &readOnlyPoolEntry{recorder: recorder, refs: 1}
	p.entries[key] = entry
	return newReadOnlyPoolLease(p, key, entry), nil
}

// CloseExpired closes idle handles whose TTL has elapsed.
func (p *ReadOnlyPool) CloseExpired(ctx context.Context) error {
	if p == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("store: close expired read-only pool context is required")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.closeExpiredLocked(ctx, p.now())
}

// Close closes every pooled handle and rejects future opens.
func (p *ReadOnlyPool) Close(ctx context.Context) error {
	if p == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("store: close read-only pool context is required")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	var closeErr error
	for key, entry := range p.entries {
		delete(p.entries, key)
		if entry == nil || entry.recorder == nil {
			continue
		}
		if err := entry.recorder.Close(ctx); err != nil {
			closeErr = errors.Join(closeErr, fmt.Errorf("store: close pooled read-only recorder: %w", err))
		}
	}
	return closeErr
}

func (p *ReadOnlyPool) release(key readOnlyPoolKey, entry *readOnlyPoolEntry) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	current := p.entries[key]
	if current == nil || current != entry {
		return nil
	}
	if current.refs > 0 {
		current.refs--
	}
	if current.refs == 0 {
		current.expiresAt = p.now().Add(p.ttl)
	}
	return nil
}

func (p *ReadOnlyPool) closeExpiredLocked(ctx context.Context, now time.Time) error {
	var closeErr error
	for key, entry := range p.entries {
		if entry == nil || entry.refs > 0 || entry.expiresAt.IsZero() || entry.expiresAt.After(now) {
			continue
		}
		delete(p.entries, key)
		if entry.recorder == nil {
			continue
		}
		if err := entry.recorder.Close(ctx); err != nil {
			closeErr = errors.Join(closeErr, fmt.Errorf("store: close expired read-only recorder: %w", err))
		}
	}
	return closeErr
}

func normalizeReadOnlyPoolKey(sessionID string, path string) (readOnlyPoolKey, error) {
	cleanSessionID := strings.TrimSpace(sessionID)
	if cleanSessionID == "" {
		return readOnlyPoolKey{}, errors.New("store: read-only pool session id is required")
	}
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return readOnlyPoolKey{}, errors.New("store: read-only pool path is required")
	}
	return readOnlyPoolKey{
		sessionID: cleanSessionID,
		path:      filepath.Clean(cleanPath),
	}, nil
}

type readOnlyPoolLease struct {
	pool  *ReadOnlyPool
	key   readOnlyPoolKey
	entry *readOnlyPoolEntry
	once  sync.Once
	err   error
}

var _ store.EventRecorder = (*readOnlyPoolLease)(nil)
var _ transcript.Reader = (*readOnlyPoolLease)(nil)

func newReadOnlyPoolLease(
	pool *ReadOnlyPool,
	key readOnlyPoolKey,
	entry *readOnlyPoolEntry,
) *readOnlyPoolLease {
	return &readOnlyPoolLease{
		pool:  pool,
		key:   key,
		entry: entry,
	}
}

func (l *readOnlyPoolLease) Record(context.Context, store.SessionEvent) error {
	return ErrReadOnlyRecordEvents
}

func (l *readOnlyPoolLease) RecordTokenUsage(context.Context, store.TokenUsage) error {
	return ErrReadOnlyRecordTokenUsage
}

func (l *readOnlyPoolLease) Query(
	ctx context.Context,
	query store.EventQuery,
) ([]store.SessionEvent, error) {
	if l == nil || l.entry == nil || l.entry.recorder == nil {
		return nil, errors.New("store: read-only pool lease recorder is required")
	}
	return l.entry.recorder.Query(ctx, query)
}

func (l *readOnlyPoolLease) History(
	ctx context.Context,
	query store.EventQuery,
) ([]store.TurnHistory, error) {
	if l == nil || l.entry == nil || l.entry.recorder == nil {
		return nil, errors.New("store: read-only pool lease recorder is required")
	}
	return l.entry.recorder.History(ctx, query)
}

func (l *readOnlyPoolLease) TranscriptPage(
	ctx context.Context,
	query transcript.PageQuery,
) (transcript.Page, error) {
	if l == nil || l.entry == nil || l.entry.recorder == nil {
		return transcript.Page{}, errors.New("store: read-only pool lease recorder is required")
	}
	reader, ok := l.entry.recorder.(transcript.Reader)
	if !ok {
		return transcript.Page{}, errors.New("store: pooled recorder has no transcript projection")
	}
	return reader.TranscriptPage(ctx, query)
}

func (l *readOnlyPoolLease) TranscriptChanges(
	ctx context.Context,
	query transcript.ChangeQuery,
) (transcript.ChangePage, error) {
	if l == nil || l.entry == nil || l.entry.recorder == nil {
		return transcript.ChangePage{}, errors.New("store: read-only pool lease recorder is required")
	}
	reader, ok := l.entry.recorder.(transcript.Reader)
	if !ok {
		return transcript.ChangePage{}, errors.New("store: pooled recorder has no transcript projection")
	}
	return reader.TranscriptChanges(ctx, query)
}

func (l *readOnlyPoolLease) Close(context.Context) error {
	if l == nil || l.pool == nil || l.entry == nil {
		return nil
	}
	l.once.Do(func() {
		l.err = l.pool.release(l.key, l.entry)
	})
	return l.err
}
