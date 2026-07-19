package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/clientstate"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func TestDesktopStateWebSocketProtocol(t *testing.T) {
	t.Parallel()

	t.Run("Should send one fenced snapshot after subscription (UT-030)", func(t *testing.T) {
		t.Parallel()
		fixture := newDesktopStateWebSocketFixture(t, clientstate.DefaultLimits(), nil)
		seedDesktopState(t, fixture.engine, "z:last", `{"v":1}`)
		seedDesktopState(t, fixture.engine, "a:first", `{"v":1}`)
		conn := fixture.dial(t)

		writeDesktopStateFrame(t, conn, contract.DesktopStateSubscribeFrame{Op: desktopStateFrameSub})
		var snapshot contract.DesktopStateSnapshotFrame
		readDesktopStateFrame(t, conn, &snapshot)
		if snapshot.Op != desktopStateFrameSnapshot || snapshot.AsOfSeq != 2 || len(snapshot.Entries) != 2 {
			t.Fatalf("snapshot = %#v, want one fenced snapshot at seq 2", snapshot)
		}
		if snapshot.Entries[0].Key != "a:first" || snapshot.Entries[1].Key != "z:last" {
			t.Fatalf("snapshot order = %#v, want key order", snapshot.Entries)
		}

		writeDesktopStateFrame(t, conn, contract.DesktopStatePingFrame{Op: desktopStateFramePing})
		var pong contract.DesktopStatePongFrame
		readDesktopStateFrame(t, conn, &pong)
		if pong.Op != desktopStateFramePong {
			t.Fatalf("second server frame = %#v, want pong without duplicate snapshot", pong)
		}
	})

	t.Run("Should ack the sender and publish an origin-tagged event to peers (UT-031)", func(t *testing.T) {
		t.Parallel()
		fixture := newDesktopStateWebSocketFixture(t, clientstate.DefaultLimits(), nil)
		sender := fixture.dial(t)
		peer := fixture.dial(t)
		subscribeDesktopStateSocket(t, sender)
		subscribeDesktopStateSocket(t, peer)

		writeDesktopStateFrame(t, sender, contract.DesktopStateApplyFrame{
			Op: desktopStateFrameApply, Req: "req-1",
			Ops: []contract.DesktopStateApplyOp{{
				Kind: contract.DesktopStateOpPut, Key: "desktop", Value: new(map[string]any{"v": 1}),
			}},
		})
		var ack contract.DesktopStateAckFrame
		readDesktopStateFrame(t, sender, &ack)
		if ack.Op != desktopStateFrameAck || ack.Req != "req-1" || len(ack.Results) != 1 {
			t.Fatalf("ack = %#v, want req-1 with one result", ack)
		}
		if ack.Results[0].Key != "desktop" || ack.Results[0].Rev != 1 || ack.Results[0].Seq != 1 {
			t.Fatalf("ack result = %#v, want desktop rev=1 seq=1", ack.Results[0])
		}

		var event contract.DesktopStateEventFrame
		readDesktopStateFrame(t, peer, &event)
		if event.Op != desktopStateFrameEvent || event.Entry.Key != "desktop" || event.Origin == "" {
			t.Fatalf("peer event = %#v, want origin-tagged desktop event", event)
		}

		writeDesktopStateFrame(t, sender, contract.DesktopStatePingFrame{Op: desktopStateFramePing})
		var pong contract.DesktopStatePongFrame
		readDesktopStateFrame(t, sender, &pong)
		if pong.Op != desktopStateFramePong {
			t.Fatalf("sender frame after ack = %#v, want pong and no echoed event", pong)
		}
	})

	t.Run("Should report an oversized apply and keep the connection usable (UT-032)", func(t *testing.T) {
		t.Parallel()
		fixture := newDesktopStateWebSocketFixture(t, clientstate.Limits{
			MaxValueBytes: 24, MaxKeysPerWorkspace: 32,
		}, nil)
		conn := fixture.dial(t)
		subscribeDesktopStateSocket(t, conn)

		writeDesktopStateFrame(t, conn, contract.DesktopStateApplyFrame{
			Op: desktopStateFrameApply, Req: "too-large",
			Ops: []contract.DesktopStateApplyOp{{
				Kind: contract.DesktopStateOpPut, Key: "desktop",
				Value: new(map[string]any{"payload": strings.Repeat("x", 80)}),
			}},
		})
		var failure contract.DesktopStateErrorFrame
		readDesktopStateFrame(t, conn, &failure)
		if failure.Op != desktopStateFrameError || failure.Req != "too-large" ||
			failure.Code != contract.DesktopStateErrorValueTooLarge {
			t.Fatalf("error frame = %#v, want correlated value-too-large error", failure)
		}

		writeDesktopStateFrame(t, conn, contract.DesktopStatePingFrame{Op: desktopStateFramePing})
		var pong contract.DesktopStatePongFrame
		readDesktopStateFrame(t, conn, &pong)
		if pong.Op != desktopStateFramePong {
			t.Fatalf("frame after recoverable error = %#v, want pong", pong)
		}
	})

	t.Run("Should finish a mutation after the client disconnects (UT-034)", func(t *testing.T) {
		t.Parallel()
		engine := newDesktopStateTestEngine(t, clientstate.DefaultLimits())
		blocking := &blockingDesktopStateService{
			Engine: engine, started: make(chan struct{}), release: make(chan struct{}),
		}
		fixture := newDesktopStateWebSocketFixtureWithEngine(t, engine, blocking)
		conn := fixture.dial(t)
		subscribeDesktopStateSocket(t, conn)

		writeDesktopStateFrame(t, conn, contract.DesktopStateApplyFrame{
			Op: desktopStateFrameApply, Req: "detached",
			Ops: []contract.DesktopStateApplyOp{{
				Kind: contract.DesktopStateOpPut, Key: "desktop", Value: new(map[string]any{"v": 1}),
			}},
		})
		select {
		case <-blocking.started:
		case <-time.After(2 * time.Second):
			t.Fatal("socket apply did not reach the service")
		}
		if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Fatalf("close client websocket: %v", err)
		}
		close(blocking.release)

		deadline := time.Now().Add(2 * time.Second)
		for {
			request, err := http.NewRequestWithContext(
				t.Context(),
				http.MethodGet,
				fixture.server.URL+desktopStateItemURL("desktop"),
				http.NoBody,
			)
			if err != nil {
				t.Fatalf("create committed desktop-state request: %v", err)
			}
			response, err := http.DefaultClient.Do(request)
			if err != nil {
				t.Fatalf("GET committed desktop state: %v", err)
			}
			bodyErr := response.Body.Close()
			if bodyErr != nil {
				t.Fatalf("close GET response: %v", bodyErr)
			}
			if response.StatusCode == http.StatusOK {
				break
			}
			if response.StatusCode != http.StatusNotFound {
				t.Fatalf("GET status = %d, want 200 or pending 404", response.StatusCode)
			}
			if time.Now().After(deadline) {
				t.Fatal("detached socket mutation did not commit")
			}
			time.Sleep(10 * time.Millisecond)
		}
	})
}

func TestDesktopStateWriterPump(t *testing.T) {
	t.Parallel()

	t.Run("Should serialize concurrent acknowledgements and remote events (UT-078)", func(t *testing.T) {
		t.Parallel()
		fixture := newDesktopStateWebSocketFixture(t, clientstate.DefaultLimits(), nil)
		conn := fixture.dial(t)
		subscribeDesktopStateSocket(t, conn)

		const writes = 20
		clientWrites := make(chan error, 1)
		go func() {
			for index := range writes {
				err := conn.WriteJSON(contract.DesktopStateApplyFrame{
					Op: desktopStateFrameApply, Req: fmt.Sprintf("req-%d", index),
					Ops: []contract.DesktopStateApplyOp{{
						Kind:  contract.DesktopStateOpPut,
						Key:   fmt.Sprintf("own:%d", index),
						Value: new(map[string]any{"v": index}),
					}},
				})
				if err != nil {
					clientWrites <- fmt.Errorf("write client apply %d: %w", index, err)
					return
				}
			}
			clientWrites <- nil
		}()
		remoteWrites := make(chan error, 1)
		go func() {
			for index := range writes {
				_, err := fixture.engine.Apply(
					context.Background(), "w1", desktopStateDomain,
					[]clientstate.Op{{
						Kind: clientstate.OpPut, Key: fmt.Sprintf("remote:%d", index),
						Value: fmt.Appendf(nil, `{"v":%d}`, index),
					}},
					clientstate.ApplyOptions{Origin: "remote-writer"},
				)
				if err != nil {
					remoteWrites <- fmt.Errorf("apply remote value %d: %w", index, err)
					return
				}
			}
			remoteWrites <- nil
		}()

		acks := make(map[string]struct{}, writes)
		var events int
		var lastEventSeq contract.DesktopStateSafeNumber
		for len(acks)+events < writes*2 {
			_, payload, err := conn.ReadMessage()
			if err != nil {
				t.Fatalf("read serialized frame: %v", err)
			}
			var envelope struct {
				Op string `json:"op"`
			}
			if err := json.Unmarshal(payload, &envelope); err != nil {
				t.Fatalf("decode frame envelope: %v; payload=%s", err, payload)
			}
			switch envelope.Op {
			case desktopStateFrameAck:
				var ack contract.DesktopStateAckFrame
				if err := json.Unmarshal(payload, &ack); err != nil {
					t.Fatalf("decode ack: %v", err)
				}
				acks[ack.Req] = struct{}{}
			case desktopStateFrameEvent:
				var event contract.DesktopStateEventFrame
				if err := json.Unmarshal(payload, &event); err != nil {
					t.Fatalf("decode event: %v", err)
				}
				if event.Origin != "remote-writer" || event.Entry.Seq <= lastEventSeq {
					t.Fatalf("event = %#v, want remote origin and increasing seq after %d", event, lastEventSeq)
				}
				lastEventSeq = event.Entry.Seq
				events++
			default:
				t.Fatalf("unexpected serialized frame: %s", payload)
			}
		}
		if err := <-clientWrites; err != nil {
			t.Fatal(err)
		}
		if err := <-remoteWrites; err != nil {
			t.Fatal(err)
		}
		if len(acks) != writes || events != writes {
			t.Fatalf("acks=%d events=%d, want %d each", len(acks), events, writes)
		}
	})

	t.Run("Should evict only the socket whose outbound queue is full (UT-033, UT-078)", func(t *testing.T) {
		t.Parallel()
		slowServer, slowClient := newDesktopStateSocketPair(t)
		healthyServer, healthyClient := newDesktopStateSocketPair(t)
		slow := newDesktopStateSocket(context.Background(), slowServer, nil, "w1")
		healthy := newDesktopStateSocket(context.Background(), healthyServer, nil, "w1")
		t.Cleanup(slow.cancel)
		t.Cleanup(healthy.cancel)

		for index := 0; index < cap(slow.outbound); index++ {
			slow.outbound <- contract.DesktopStatePongFrame{Op: desktopStateFramePong}
		}
		if slow.enqueue(contract.DesktopStatePongFrame{Op: desktopStateFramePong}) {
			t.Fatal("enqueue on full outbound queue succeeded")
		}

		slowDone := make(chan error, 1)
		go func() { slowDone <- slow.writePump() }()
		healthyDone := make(chan error, 1)
		go func() { healthyDone <- healthy.writePump() }()
		healthyFrame := contract.DesktopStateEventFrame{
			Op: desktopStateFrameEvent,
			Entry: contract.DesktopStateEntry{
				Key: "desktop", Value: map[string]any{"v": float64(1)}, Rev: 1, Seq: 1,
			},
			Origin: "writer",
		}
		if !healthy.enqueue(healthyFrame) {
			t.Fatal("healthy socket rejected event")
		}
		if err := healthyClient.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			t.Fatalf("set healthy read deadline: %v", err)
		}
		var delivered contract.DesktopStateEventFrame
		readDesktopStateFrame(t, healthyClient, &delivered)
		if delivered.Op != desktopStateFrameEvent || delivered.Entry.Key != "desktop" {
			t.Fatalf("healthy frame = %#v, want desktop event", delivered)
		}

		if err := slowClient.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			t.Fatalf("set slow read deadline: %v", err)
		}
		var eviction contract.DesktopStateErrorFrame
		readDesktopStateFrame(t, slowClient, &eviction)
		if eviction.Op != desktopStateFrameError || eviction.Code != contract.DesktopStateErrorSlowConsumer {
			t.Fatalf("eviction = %#v, want slow-consumer error", eviction)
		}
		_, _, closeErr := slowClient.ReadMessage()
		if !websocket.IsCloseError(closeErr, websocket.ClosePolicyViolation) {
			t.Fatalf("slow socket close error = %v, want policy-violation close", closeErr)
		}
		if err := <-slowDone; err != nil {
			t.Fatalf("slow write pump error = %v", err)
		}
		healthy.cancel()
		if err := <-healthyDone; err != nil && !isExpectedDesktopStateSocketError(err) {
			t.Fatalf("healthy write pump shutdown error = %v", err)
		}
	})
}

func TestDesktopStateShutdownJoinsQueuedMutationUT079(t *testing.T) {
	engine := newDesktopStateTestEngine(t, clientstate.DefaultLimits())
	service := &cancelBlockingDesktopStateService{
		Engine: engine, started: make(chan struct{}), finished: make(chan struct{}),
	}
	fixture := newDesktopStateWebSocketFixtureWithEngine(t, engine, service)
	conn := fixture.dial(t)
	subscribeDesktopStateSocket(t, conn)
	writeDesktopStateFrame(t, conn, contract.DesktopStateApplyFrame{
		Op: desktopStateFrameApply, Req: "queued",
		Ops: []contract.DesktopStateApplyOp{{
			Kind: contract.DesktopStateOpPut, Key: "desktop",
			Value: new(map[string]any{"v": 1}),
		}},
	})
	select {
	case <-service.started:
	case <-time.After(2 * time.Second):
		t.Fatal("queued mutation did not start")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := fixture.handlers.ShutdownDesktopStateStreams(ctx); err != nil {
		t.Fatalf("ShutdownDesktopStateStreams() error = %v", err)
	}
	select {
	case <-service.finished:
	default:
		t.Fatal("shutdown returned before the queued mutation exited")
	}
	if metrics := engine.Metrics(); metrics.ActiveConnections != 0 {
		t.Fatalf("active connections = %d, want 0 after both pumps joined", metrics.ActiveConnections)
	}
	if err := engine.Close(); err != nil {
		t.Fatalf("Engine.Close() error = %v", err)
	}
	if _, err := engine.Get(
		context.Background(),
		"w1",
		desktopStateDomain,
		"desktop",
	); !errors.Is(err, clientstate.ErrClosed) {
		t.Fatalf("Get() after ordered shutdown error = %v, want ErrClosed", err)
	}
}

type desktopStateWebSocketFixture struct {
	server   *httptest.Server
	engine   *clientstate.Engine
	handlers *BaseHandlers
}

func newDesktopStateWebSocketFixture(
	t *testing.T,
	limits clientstate.Limits,
	service DesktopStateService,
) desktopStateWebSocketFixture {
	t.Helper()
	engine := newDesktopStateTestEngine(t, limits)
	if service == nil {
		service = engine
	}
	return newDesktopStateWebSocketFixtureWithEngine(t, engine, service)
}

func newDesktopStateWebSocketFixtureWithEngine(
	t *testing.T,
	engine *clientstate.Engine,
	service DesktopStateService,
) desktopStateWebSocketFixture {
	t.Helper()
	handlers := NewBaseHandlers(&BaseHandlerConfig{DesktopState: service})
	router := gin.New()
	collection := "/api/workspaces/:workspace_id/desktop-state"
	router.GET(collection+"/stream", handlers.StreamDesktopState)
	router.GET(collection+"/:key", handlers.GetDesktopState)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := handlers.ShutdownDesktopStateStreams(ctx); err != nil {
			t.Errorf("ShutdownDesktopStateStreams() error = %v", err)
		}
	})
	return desktopStateWebSocketFixture{server: server, engine: engine, handlers: handlers}
}

func (f desktopStateWebSocketFixture) dial(t *testing.T) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(f.server.URL, "http") + desktopStateCollectionURL() + "/stream"
	conn, response, err := websocket.DefaultDialer.Dial(url, nil)
	closeDesktopStateDialResponse(t, response)
	if err != nil {
		if response != nil {
			t.Fatalf("dial desktop-state websocket: %v; status=%s", err, response.Status)
		}
		t.Fatalf("dial desktop-state websocket: %v", err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("close desktop-state websocket: %v", err)
		}
	})
	return conn
}

func newDesktopStateTestEngine(t *testing.T, limits clientstate.Limits) *clientstate.Engine {
	t.Helper()
	engine, err := clientstate.Open(
		clientstate.DatabasePath(t.TempDir()), desktopStateHandlerResolver{}, limits,
	)
	if err != nil {
		t.Fatalf("clientstate.Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := engine.Close(); err != nil {
			t.Errorf("Engine.Close() error = %v", err)
		}
	})
	return engine
}

type blockingDesktopStateService struct {
	*clientstate.Engine
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

type cancelBlockingDesktopStateService struct {
	*clientstate.Engine
	started  chan struct{}
	finished chan struct{}
	once     sync.Once
}

func (s *cancelBlockingDesktopStateService) Apply(
	ctx context.Context,
	_ clientstate.WorkspaceID,
	_ string,
	_ []clientstate.Op,
	options clientstate.ApplyOptions,
) ([]clientstate.Entry, error) {
	if options.Origin == "" {
		return nil, errors.New("cancel-blocking service only accepts socket mutations")
	}
	s.once.Do(func() { close(s.started) })
	<-ctx.Done()
	close(s.finished)
	return nil, ctx.Err()
}

func (s *blockingDesktopStateService) Apply(
	ctx context.Context,
	workspace clientstate.WorkspaceID,
	domain string,
	ops []clientstate.Op,
	options clientstate.ApplyOptions,
) ([]clientstate.Entry, error) {
	if options.Origin != "" {
		s.once.Do(func() { close(s.started) })
		select {
		case <-s.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return s.Engine.Apply(ctx, workspace, domain, ops, options)
}

func subscribeDesktopStateSocket(t *testing.T, conn *websocket.Conn) contract.DesktopStateSnapshotFrame {
	t.Helper()
	writeDesktopStateFrame(t, conn, contract.DesktopStateSubscribeFrame{Op: desktopStateFrameSub})
	var snapshot contract.DesktopStateSnapshotFrame
	readDesktopStateFrame(t, conn, &snapshot)
	if snapshot.Op != desktopStateFrameSnapshot {
		t.Fatalf("subscription frame = %#v, want snapshot", snapshot)
	}
	return snapshot
}

func writeDesktopStateFrame(t *testing.T, conn *websocket.Conn, frame any) {
	t.Helper()
	if err := conn.WriteJSON(frame); err != nil {
		t.Fatalf("write desktop-state frame: %v", err)
	}
}

func readDesktopStateFrame(t *testing.T, conn *websocket.Conn, target any) {
	t.Helper()
	if err := conn.ReadJSON(target); err != nil {
		t.Fatalf("read desktop-state frame: %v", err)
	}
}

func newDesktopStateSocketPair(t *testing.T) (*websocket.Conn, *websocket.Conn) {
	t.Helper()
	serverConnections := make(chan *websocket.Conn, 1)
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		conn, err := upgrader.Upgrade(writer, request, nil)
		if err != nil {
			t.Errorf("upgrade socket pair: %v", err)
			return
		}
		serverConnections <- conn
	}))
	t.Cleanup(server.Close)
	client, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	closeDesktopStateDialResponse(t, response)
	if err != nil {
		if response != nil {
			t.Fatalf("dial socket pair: %v; status=%s", err, response.Status)
		}
		t.Fatalf("dial socket pair: %v", err)
	}
	serverConn := <-serverConnections
	t.Cleanup(func() {
		if err := client.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("close socket-pair client: %v", err)
		}
		if err := serverConn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("close socket-pair server: %v", err)
		}
	})
	return serverConn, client
}

func closeDesktopStateDialResponse(t *testing.T, response *http.Response) {
	t.Helper()
	if response == nil {
		return
	}
	if err := response.Body.Close(); err != nil {
		t.Fatalf("close websocket handshake response: %v", err)
	}
}
