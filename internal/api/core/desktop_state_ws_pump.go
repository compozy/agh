package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/clientstate"
	"github.com/gorilla/websocket"
)

func (s *desktopStateSocket) relayEvents(subscription clientstate.Subscription) {
	defer close(s.events)
	for {
		select {
		case <-s.ctx.Done():
			return
		case event, ok := <-subscription.Events():
			if !ok {
				if errors.Is(subscription.Err(), clientstate.ErrSlowConsumer) {
					s.requestEviction(contract.DesktopStateErrorSlowConsumer)
				} else {
					s.cancel()
				}
				return
			}
			if event.Origin == s.id {
				continue
			}
			entry, err := desktopStateEntryFromEngine(event.Entry)
			if err != nil {
				s.sendError("", err, event.Entry.Key)
				return
			}
			if !s.enqueue(contract.DesktopStateEventFrame{
				Op: desktopStateFrameEvent, Entry: entry, Origin: event.Origin,
			}) {
				return
			}
		}
	}
}

func (s *desktopStateSocket) enqueue(frame any) bool {
	select {
	case <-s.ctx.Done():
		return false
	case s.outbound <- frame:
		if s.service != nil {
			s.service.RecordOutboundQueueDepth(len(s.outbound))
		}
		return true
	default:
		if s.service != nil {
			s.service.RecordSlowConsumerEviction()
		}
		s.requestEviction(contract.DesktopStateErrorSlowConsumer)
		return false
	}
}

func (s *desktopStateSocket) requestEviction(code contract.DesktopStateErrorCode) {
	select {
	case s.evict <- code:
	default:
	}
}

func (s *desktopStateSocket) writePump() error {
	ticker := time.NewTicker(desktopStatePingInterval)
	defer ticker.Stop()
	for {
		select {
		case code := <-s.evict:
			return s.writeEviction(code)
		default:
		}
		select {
		case code := <-s.evict:
			return s.writeEviction(code)
		case frame := <-s.outbound:
			if err := s.conn.SetWriteDeadline(time.Now().Add(desktopStateWriteTimeout)); err != nil {
				return fmt.Errorf("set desktop-state write deadline: %w", err)
			}
			if err := s.conn.WriteJSON(frame); err != nil {
				return fmt.Errorf("write desktop-state frame: %w", err)
			}
		case <-ticker.C:
			deadline := time.Now().Add(desktopStateWriteTimeout)
			if err := s.conn.SetWriteDeadline(deadline); err != nil {
				return fmt.Errorf("set desktop-state ping deadline: %w", err)
			}
			if err := s.conn.WriteControl(websocket.PingMessage, nil, deadline); err != nil {
				return fmt.Errorf("write desktop-state ping: %w", err)
			}
		case <-s.ctx.Done():
			return s.writeClose(websocket.CloseGoingAway, "daemon shutdown")
		}
	}
}

func (s *desktopStateSocket) writeEviction(code contract.DesktopStateErrorCode) error {
	deadline := time.Now().Add(desktopStateEvictionTimeout)
	if err := s.conn.SetWriteDeadline(deadline); err != nil {
		return fmt.Errorf("set desktop-state eviction deadline: %w", err)
	}
	frameErr := s.conn.WriteJSON(contract.DesktopStateErrorFrame{
		Op: desktopStateFrameError, Code: code,
	})
	closeErr := s.writeClose(websocket.ClosePolicyViolation, string(code))
	if frameErr != nil {
		frameErr = fmt.Errorf("write desktop-state eviction frame: %w", frameErr)
	}
	return errors.Join(frameErr, closeErr)
}

func (s *desktopStateSocket) writeClose(code int, reason string) error {
	deadline := time.Now().Add(desktopStateEvictionTimeout)
	err := s.conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(code, reason),
		deadline,
	)
	if err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("write desktop-state close: %w", err)
	}
	return nil
}
