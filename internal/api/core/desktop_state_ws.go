package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/clientstate"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	desktopStateFrameSub      = "sub"
	desktopStateFrameApply    = "apply"
	desktopStateFramePing     = "ping"
	desktopStateFrameSnapshot = "snapshot"
	desktopStateFrameEvent    = "event"
	desktopStateFrameAck      = "ack"
	desktopStateFrameError    = "error"
	desktopStateFramePong     = "pong"

	desktopStateWriteTimeout    = 10 * time.Second
	desktopStatePingInterval    = 30 * time.Second
	desktopStatePongTimeout     = 60 * time.Second
	desktopStateEvictionTimeout = 2 * time.Second
	desktopStateMutationTimeout = 5 * time.Second
	desktopStateMaxMessageBytes = 4 << 20
)

var desktopStateUpgrader = websocket.Upgrader{HandshakeTimeout: desktopStateWriteTimeout}

// StreamDesktopState upgrades one HTTP or UDS request to the desktop-state WebSocket protocol.
func (h *BaseHandlers) StreamDesktopState(c *gin.Context) {
	if h.DesktopState == nil {
		h.respondDesktopStateError(c, clientstate.ErrClosed, "")
		return
	}
	workspace := desktopStateWorkspace(c)
	if _, err := h.DesktopState.List(c.Request.Context(), workspace, desktopStateDomain); err != nil {
		h.respondDesktopStateError(c, err, "")
		return
	}
	baseCtx, done, ok := h.desktopStateStreams.begin()
	if !ok {
		h.respondDesktopStateError(c, clientstate.ErrClosed, "")
		return
	}
	defer done()

	conn, err := desktopStateUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		if h.Logger != nil {
			h.Logger.Warn(
				"desktop-state websocket upgrade failed",
				"component", "clientstate",
				"workspace", workspace,
				"code", contract.DesktopStateErrorInvalidValue,
				"error", err,
			)
		}
		return
	}
	h.DesktopState.ConnectionOpened()
	defer h.DesktopState.ConnectionClosed()

	socket := newDesktopStateSocket(baseCtx, conn, h.DesktopState, workspace)
	if h.Logger != nil {
		h.Logger.Info(
			"clientstate.ws.connect",
			"component", "clientstate",
			"workspace", workspace,
			"conn_id", socket.id,
		)
	}
	runErr := socket.run()
	if h.Logger != nil {
		closeReason := "normal"
		if !isExpectedDesktopStateSocketError(runErr) {
			closeReason = "error"
		}
		h.Logger.Info(
			"clientstate.ws.disconnect",
			"component", "clientstate",
			"workspace", workspace,
			"conn_id", socket.id,
			"close_reason", closeReason,
		)
		if closeReason == "error" {
			h.Logger.Debug(
				"desktop-state websocket closed with error",
				"component", "clientstate",
				"workspace", workspace,
				"conn_id", socket.id,
				"error", runErr,
			)
		}
	}
}

type desktopStateIncomingFrame struct {
	Op  string                         `json:"op"`
	Req string                         `json:"req"`
	Ops []contract.DesktopStateApplyOp `json:"ops"`
}

type desktopStateSocket struct {
	baseCtx   context.Context
	ctx       context.Context
	cancel    context.CancelFunc
	conn      *websocket.Conn
	service   DesktopStateService
	workspace clientstate.WorkspaceID
	id        string
	outbound  chan any
	evict     chan contract.DesktopStateErrorCode
	events    chan struct{}

	subscription clientstate.Subscription
}

func newDesktopStateSocket(
	baseCtx context.Context,
	conn *websocket.Conn,
	service DesktopStateService,
	workspace clientstate.WorkspaceID,
) *desktopStateSocket {
	ctx, cancel := context.WithCancel(baseCtx)
	return &desktopStateSocket{
		baseCtx: baseCtx, ctx: ctx, cancel: cancel, conn: conn,
		service: service, workspace: workspace, id: uuid.NewString(),
		outbound: make(chan any, clientstate.SubscriptionBufferSize),
		evict:    make(chan contract.DesktopStateErrorCode, 1),
		events:   make(chan struct{}),
	}
}

func (s *desktopStateSocket) run() error {
	if err := s.configureReader(); err != nil {
		return err
	}
	writerDone := make(chan error, 1)
	go func() {
		writerErr := s.writePump()
		closeErr := s.conn.Close()
		if closeErr != nil && !errors.Is(closeErr, net.ErrClosed) {
			closeErr = fmt.Errorf("close desktop-state websocket from writer: %w", closeErr)
		}
		writerDone <- errors.Join(writerErr, closeErr)
	}()

	readErr := s.readPump()
	s.cancel()
	if s.subscription != nil {
		if err := s.subscription.Close(); err != nil {
			readErr = errors.Join(readErr, fmt.Errorf("close desktop-state subscription: %w", err))
		}
		<-s.events
	}
	writerErr := <-writerDone
	closeErr := s.conn.Close()
	if closeErr != nil && !errors.Is(closeErr, context.Canceled) && !errors.Is(closeErr, net.ErrClosed) {
		closeErr = fmt.Errorf("close desktop-state websocket: %w", closeErr)
	}
	return errors.Join(readErr, writerErr, closeErr)
}

func (s *desktopStateSocket) configureReader() error {
	s.conn.SetReadLimit(desktopStateMaxMessageBytes)
	if err := s.conn.SetReadDeadline(time.Now().Add(desktopStatePongTimeout)); err != nil {
		return fmt.Errorf("set desktop-state read deadline: %w", err)
	}
	s.conn.SetPongHandler(func(string) error {
		return s.conn.SetReadDeadline(time.Now().Add(desktopStatePongTimeout))
	})
	return nil
}

func (s *desktopStateSocket) readPump() error {
	subscribed := false
	for {
		messageType, payload, err := s.conn.ReadMessage()
		if err != nil {
			return err
		}
		if messageType != websocket.TextMessage {
			s.sendError("", clientstate.ErrInvalidValue, "")
			continue
		}
		var frame desktopStateIncomingFrame
		if err := json.Unmarshal(payload, &frame); err != nil {
			s.sendError("", fmt.Errorf("%w: %w", clientstate.ErrInvalidValue, err), "")
			continue
		}
		switch strings.TrimSpace(frame.Op) {
		case desktopStateFrameSub:
			if subscribed {
				s.sendError("", clientstate.ErrInvalidValue, "")
				continue
			}
			if err := s.subscribe(); err != nil {
				s.sendError("", err, "")
				continue
			}
			subscribed = true
		case desktopStateFrameApply:
			s.apply(frame)
		case desktopStateFramePing:
			s.enqueue(contract.DesktopStatePongFrame{Op: desktopStateFramePong})
		default:
			s.sendError(frame.Req, clientstate.ErrInvalidValue, "")
		}
	}
}

func (s *desktopStateSocket) subscribe() error {
	subscription, err := s.service.Watch(s.ctx, s.workspace, []string{desktopStateDomain})
	if err != nil {
		return err
	}
	entries, err := desktopStateEntriesFromEngine(subscription.Snapshot())
	if err != nil {
		closeErr := subscription.Close()
		return errors.Join(err, closeErr)
	}
	if subscription.AsOfSeq() > desktopStateMaxSafeID {
		closeErr := subscription.Close()
		return errors.Join(errDesktopStateUnsafeNumber, closeErr)
	}
	if !s.enqueue(contract.DesktopStateSnapshotFrame{
		Op:      desktopStateFrameSnapshot,
		AsOfSeq: contract.DesktopStateSafeNumber(subscription.AsOfSeq()),
		Entries: entries,
	}) {
		return errors.Join(clientstate.ErrSlowConsumer, subscription.Close())
	}
	s.subscription = subscription
	go s.relayEvents(subscription)
	return nil
}

func (s *desktopStateSocket) apply(frame desktopStateIncomingFrame) {
	req := strings.TrimSpace(frame.Req)
	if req == "" {
		s.sendError(req, clientstate.ErrInvalidValue, desktopStateErrorKey(frame.Ops))
		return
	}
	ops, err := desktopStateOpsFromContract(frame.Ops)
	if err != nil {
		s.sendError(req, err, desktopStateErrorKey(frame.Ops))
		return
	}
	operationCtx, cancel := context.WithTimeout(s.baseCtx, desktopStateMutationTimeout)
	entries, err := s.service.Apply(
		operationCtx,
		s.workspace,
		desktopStateDomain,
		ops,
		clientstate.ApplyOptions{Origin: s.id},
	)
	cancel()
	if err != nil {
		s.sendError(req, err, desktopStateErrorKey(frame.Ops))
		return
	}
	if len(entries) != len(frame.Ops) {
		s.sendError(req, unexpectedDesktopStateResultCount(len(frame.Ops), len(entries)), "")
		return
	}
	results := make([]contract.DesktopStateAckResult, 0, len(entries))
	for _, entry := range entries {
		if entry.Rev > desktopStateMaxSafeID || entry.Seq > desktopStateMaxSafeID {
			s.sendError(req, errDesktopStateUnsafeNumber, entry.Key)
			return
		}
		results = append(results, contract.DesktopStateAckResult{
			Key: entry.Key,
			Rev: contract.DesktopStateSafeNumber(entry.Rev), Seq: contract.DesktopStateSafeNumber(entry.Seq),
		})
	}
	s.enqueue(contract.DesktopStateAckFrame{Op: desktopStateFrameAck, Req: req, Results: results})
}

func (s *desktopStateSocket) sendError(req string, err error, key string) {
	_, code := desktopStateErrorStatus(err)
	s.enqueue(contract.DesktopStateErrorFrame{
		Op: desktopStateFrameError, Req: strings.TrimSpace(req), Code: code, Key: key,
	})
}

func isExpectedDesktopStateSocketError(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return true
	}
	return websocket.IsCloseError(
		err,
		websocket.CloseNormalClosure,
		websocket.CloseGoingAway,
		websocket.CloseNoStatusReceived,
	)
}
