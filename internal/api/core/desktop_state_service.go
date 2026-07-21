package core

import (
	"context"
	"fmt"
	"sync"

	"github.com/compozy/agh/internal/clientstate"
)

// DesktopStateService is the daemon-owned client-state engine exposed by the desktop-state API.
type DesktopStateService interface {
	clientstate.Service
	ConnectionOpened()
	ConnectionClosed()
	RecordOutboundQueueDepth(int)
	RecordSlowConsumerEviction()
}

type desktopStateStreamLifecycle struct {
	mu        sync.Mutex
	base      context.Context
	cancel    context.CancelFunc
	accepting bool
	wg        sync.WaitGroup
}

func newDesktopStateStreamLifecycle() *desktopStateStreamLifecycle {
	lifecycle := &desktopStateStreamLifecycle{}
	lifecycle.reset()
	return lifecycle
}

func (l *desktopStateStreamLifecycle) reset() {
	if l == nil {
		return
	}
	base, cancel := context.WithCancel(context.Background())
	l.mu.Lock()
	if l.cancel != nil {
		l.cancel()
	}
	l.base = base
	l.cancel = cancel
	l.accepting = true
	l.mu.Unlock()
}

func (l *desktopStateStreamLifecycle) begin() (context.Context, func(), bool) {
	if l == nil {
		return nil, nil, false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.accepting || l.base == nil || l.base.Err() != nil {
		return nil, nil, false
	}
	l.wg.Add(1)
	return l.base, l.wg.Done, true
}

func (l *desktopStateStreamLifecycle) shutdown(ctx context.Context) error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	l.accepting = false
	if l.cancel != nil {
		l.cancel()
	}
	l.mu.Unlock()

	done := make(chan struct{})
	go func() {
		l.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("api: wait for desktop-state streams: %w", ctx.Err())
	}
}
