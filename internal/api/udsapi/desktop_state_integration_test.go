//go:build integration

package udsapi

// Invariant: the daemon-owned desktop-state engine has one ordered, workspace-isolated
// contract across HTTP, UDS, WebSocket, CLI, deletion, and restart boundaries.
// Owning layer: daemon-wired public transports. Canonical suite: desktop_state_integration_test.go.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	e2etest "github.com/compozy/agh/internal/testutil/e2e"
	"github.com/gorilla/websocket"
)

const desktopStateIntegrationTimeout = 15 * time.Second

func TestDesktopStateDaemonWiringIntegration(t *testing.T) {
	runtimeHarness := e2etest.StartRuntimeHarness(t, e2etest.RuntimeHarnessOptions{})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	primaryWorkspace := runtimeHarness.WorkspaceID
	primaryRoot := runtimeHarness.WorkspaceRoot

	t.Run("Should publish HTTP writes to an HTTP WebSocket subscriber (IT-001)", func(t *testing.T) {
		conn, snapshot := openDesktopStateHTTPSubscriber(t, runtimeHarness, primaryWorkspace)
		entry := putDesktopState(
			t,
			ctx,
			runtimeHarness.HTTPClient,
			runtimeHarness.HTTPBaseURL,
			primaryWorkspace,
			"win:app:tasks",
			map[string]any{"v": float64(1), "source": "http"},
		)
		event := readDesktopStateFrame[contract.DesktopStateEventFrame](t, conn, "event")
		if event.Entry.Key != entry.Key || event.Entry.Rev != entry.Rev ||
			event.Entry.Seq != entry.Seq || !reflect.DeepEqual(event.Entry.Value, entry.Value) {
			t.Fatalf("event.Entry = %#v, want %#v", event.Entry, entry)
		}
		if event.Origin != "" {
			t.Fatalf("event.Origin = %q, want empty HTTP origin", event.Origin)
		}
		if event.Entry.Seq <= snapshot.AsOfSeq {
			t.Fatalf("event seq = %d, want greater than snapshot fence %d", event.Entry.Seq, snapshot.AsOfSeq)
		}
	})

	t.Run("Should expose WebSocket apply through HTTP and suppress only the sender echo (IT-002)", func(t *testing.T) {
		writer, _ := openDesktopStateHTTPSubscriber(t, runtimeHarness, primaryWorkspace)
		observer, _ := openDesktopStateHTTPSubscriber(t, runtimeHarness, primaryWorkspace)
		value := map[string]any{"v": float64(1), "source": "websocket"}
		writeDesktopStateFrame(t, writer, contract.DesktopStateApplyFrame{
			Op: "apply", Req: "it-002",
			Ops: []contract.DesktopStateApplyOp{{
				Kind: contract.DesktopStateOpPut, Key: "win:app:ws", Value: &value,
			}},
		})
		ack := readDesktopStateFrame[contract.DesktopStateAckFrame](t, writer, "ack")
		if ack.Req != "it-002" || len(ack.Results) != 1 || ack.Results[0].Key != "win:app:ws" {
			t.Fatalf("ack = %#v, want correlated one-key result", ack)
		}
		event := readDesktopStateFrame[contract.DesktopStateEventFrame](t, observer, "event")
		if event.Entry.Key != "win:app:ws" || event.Origin == "" {
			t.Fatalf("observer event = %#v, want remote WebSocket origin", event)
		}
		got := getDesktopState(
			t,
			ctx,
			runtimeHarness.HTTPClient,
			runtimeHarness.HTTPBaseURL,
			primaryWorkspace,
			"win:app:ws",
		)
		if got.Rev != ack.Results[0].Rev || got.Seq != ack.Results[0].Seq ||
			!reflect.DeepEqual(got.Value, value) {
			t.Fatalf("HTTP GET = %#v, ack=%#v value=%#v", got, ack.Results[0], value)
		}
		writeDesktopStateFrame(t, writer, contract.DesktopStatePingFrame{Op: "ping"})
		readDesktopStateFrame[contract.DesktopStatePongFrame](t, writer, "pong")
	})

	var secondaryWorkspace string
	t.Run("Should isolate subscribers by workspace (IT-003)", func(t *testing.T) {
		secondary, err := runtimeHarness.ResolveWorkspace(ctx, t.TempDir())
		if err != nil {
			t.Fatalf("ResolveWorkspace(secondary) error = %v", err)
		}
		secondaryWorkspace = secondary.ID
		primaryConn, _ := openDesktopStateHTTPSubscriber(t, runtimeHarness, primaryWorkspace)
		secondaryConn, _ := openDesktopStateHTTPSubscriber(t, runtimeHarness, secondaryWorkspace)
		putDesktopState(
			t,
			ctx,
			runtimeHarness.HTTPClient,
			runtimeHarness.HTTPBaseURL,
			primaryWorkspace,
			"isolation:w1",
			map[string]any{"workspace": "w1"},
		)
		primaryEvent := readDesktopStateFrame[contract.DesktopStateEventFrame](t, primaryConn, "event")
		if primaryEvent.Entry.Key != "isolation:w1" {
			t.Fatalf("primary event key = %q, want isolation:w1", primaryEvent.Entry.Key)
		}
		writeDesktopStateFrame(t, secondaryConn, contract.DesktopStatePingFrame{Op: "ping"})
		readDesktopStateFrame[contract.DesktopStatePongFrame](t, secondaryConn, "pong")
		list := listDesktopState(
			t,
			ctx,
			runtimeHarness.HTTPClient,
			runtimeHarness.HTTPBaseURL,
			secondaryWorkspace,
		)
		if len(list.Entries) != 0 {
			t.Fatalf("secondary entries = %#v, want empty isolated workspace", list.Entries)
		}
	})

	t.Run("Should purge deleted workspace state and close only its socket (IT-004)", func(t *testing.T) {
		deleting, err := runtimeHarness.ResolveWorkspace(ctx, t.TempDir())
		if err != nil {
			t.Fatalf("ResolveWorkspace(deleting) error = %v", err)
		}
		deletingConn, _ := openDesktopStateHTTPSubscriber(t, runtimeHarness, deleting.ID)
		unaffectedConn, _ := openDesktopStateHTTPSubscriber(t, runtimeHarness, secondaryWorkspace)
		putDesktopState(
			t,
			ctx,
			runtimeHarness.HTTPClient,
			runtimeHarness.HTTPBaseURL,
			deleting.ID,
			"delete:me",
			map[string]any{"v": float64(1)},
		)
		readDesktopStateFrame[contract.DesktopStateEventFrame](t, deletingConn, "event")
		stdout, stderr, err := runtimeHarness.CLI.Run(ctx, "workspace", "remove", deleting.ID, "-o", "json")
		if err != nil {
			t.Fatalf("workspace remove error = %v; stdout=%s stderr=%s", err, stdout, stderr)
		}
		response := requestDesktopState(
			t,
			ctx,
			runtimeHarness.HTTPClient,
			http.MethodGet,
			runtimeHarness.HTTPBaseURL+desktopStatePath(deleting.ID)+"/delete:me",
			nil,
		)
		if response.status != http.StatusNotFound {
			t.Fatalf("GET deleted workspace status = %d, want %d; body=%s", response.status, http.StatusNotFound, response.body)
		}
		var payload contract.DesktopStateErrorPayload
		decodeDesktopStateJSON(t, response.body, &payload)
		if payload.Code != contract.DesktopStateErrorWorkspace {
			t.Fatalf("GET deleted workspace code = %q, want %q", payload.Code, contract.DesktopStateErrorWorkspace)
		}
		if err := deletingConn.SetReadDeadline(time.Now().Add(desktopStateIntegrationTimeout)); err != nil {
			t.Fatalf("SetReadDeadline(deleting socket) error = %v", err)
		}
		if _, _, err := deletingConn.ReadMessage(); err == nil {
			t.Fatal("deleted workspace socket remained open")
		}
		writeDesktopStateFrame(t, unaffectedConn, contract.DesktopStatePingFrame{Op: "ping"})
		readDesktopStateFrame[contract.DesktopStatePongFrame](t, unaffectedConn, "pong")
	})

	t.Run("Should preserve CLI agent-operability parity with HTTP and WebSocket (IT-006)", func(t *testing.T) {
		conn, _ := openDesktopStateHTTPSubscriber(t, runtimeHarness, primaryWorkspace)
		stdout, stderr, err := runtimeHarness.CLI.Run(
			ctx,
			"desktop-state", "set",
			"--workspace", primaryWorkspace,
			"--key", "cli:parity",
			"--value", `{"v":1,"source":"cli"}`,
			"-o", "json",
		)
		if err != nil {
			t.Fatalf("desktop-state set error = %v; stdout=%s stderr=%s", err, stdout, stderr)
		}
		var setEntry contract.DesktopStateEntry
		decodeDesktopStateJSON(t, []byte(stdout), &setEntry)
		event := readDesktopStateFrame[contract.DesktopStateEventFrame](t, conn, "event")
		if !reflect.DeepEqual(event.Entry, setEntry) || event.Origin != "" {
			t.Fatalf("CLI event = %#v, want entry %#v with empty origin", event, setEntry)
		}
		stdout, stderr, err = runtimeHarness.CLI.Run(
			ctx,
			"desktop-state", "get",
			"--workspace", primaryWorkspace,
			"--key", "cli:parity",
			"-o", "json",
		)
		if err != nil {
			t.Fatalf("desktop-state get error = %v; stdout=%s stderr=%s", err, stdout, stderr)
		}
		httpResponse := requestDesktopState(
			t,
			ctx,
			runtimeHarness.HTTPClient,
			http.MethodGet,
			runtimeHarness.HTTPBaseURL+desktopStatePath(primaryWorkspace)+"/cli:parity",
			nil,
		)
		if httpResponse.status != http.StatusOK {
			t.Fatalf("HTTP GET status = %d, want %d; body=%s", httpResponse.status, http.StatusOK, httpResponse.body)
		}
		var cliEntry, httpEntry contract.DesktopStateEntry
		decodeDesktopStateJSON(t, []byte(stdout), &cliEntry)
		decodeDesktopStateJSON(t, httpResponse.body, &httpEntry)
		if !reflect.DeepEqual(cliEntry, httpEntry) {
			t.Fatalf("CLI GET = %#v, HTTP GET = %#v", cliEntry, httpEntry)
		}
	})

	t.Run("Should return byte-identical CRUD and apply bodies over HTTP and UDS (IT-007)", func(t *testing.T) {
		putDesktopState(
			t,
			ctx,
			runtimeHarness.HTTPClient,
			runtimeHarness.HTTPBaseURL,
			primaryWorkspace,
			"transport:parity",
			map[string]any{"v": float64(1)},
		)
		cases := []struct {
			name       string
			method     string
			path       string
			body       []byte
			wantStatus int
		}{
			{
				name: "get", method: http.MethodGet,
				path: desktopStatePath(primaryWorkspace) + "/transport:parity", wantStatus: http.StatusOK,
			},
			{
				name: "list", method: http.MethodGet,
				path: desktopStatePath(primaryWorkspace), wantStatus: http.StatusOK,
			},
			{
				name: "put conflict", method: http.MethodPut,
				path: desktopStatePath(primaryWorkspace) + "/transport:parity",
				body: desktopStateJSON(t, contract.DesktopStatePutRequest{
					Value: map[string]any{"v": float64(2)}, IfRev: desktopStateRevision(999),
				}),
				wantStatus: http.StatusConflict,
			},
			{
				name: "apply invalid", method: http.MethodPost,
				path:       desktopStatePath(primaryWorkspace) + "/apply",
				body:       desktopStateJSON(t, contract.DesktopStateApplyRequest{}),
				wantStatus: http.StatusUnprocessableEntity,
			},
			{
				name: "delete missing", method: http.MethodDelete,
				path: desktopStatePath(primaryWorkspace) + "/transport:missing", wantStatus: http.StatusNotFound,
			},
		}
		for _, tt := range cases {
			t.Run("Should match "+tt.name, func(t *testing.T) {
				httpResponse := requestDesktopState(
					t, ctx, runtimeHarness.HTTPClient, tt.method,
					runtimeHarness.HTTPBaseURL+tt.path, tt.body,
				)
				udsResponse := requestDesktopState(
					t, ctx, runtimeHarness.UDSClient, tt.method,
					runtimeHarness.UDSBaseURL+tt.path, tt.body,
				)
				if httpResponse.status != tt.wantStatus || udsResponse.status != tt.wantStatus {
					t.Fatalf(
						"statuses = HTTP %d UDS %d, want %d; HTTP=%s UDS=%s",
						httpResponse.status,
						udsResponse.status,
						tt.wantStatus,
						httpResponse.body,
						udsResponse.body,
					)
				}
				if !bytes.Equal(httpResponse.body, udsResponse.body) {
					t.Fatalf("HTTP body = %s, UDS body = %s", httpResponse.body, udsResponse.body)
				}
			})
		}
	})

	t.Run("Should preserve snapshot, event order, and errors over UDS watch (IT-008)", func(t *testing.T) {
		httpConn, httpSnapshot := openDesktopStateHTTPSubscriber(t, runtimeHarness, primaryWorkspace)
		udsConn, udsSnapshot := openDesktopStateUDSSubscriber(t, runtimeHarness, primaryWorkspace)
		if !reflect.DeepEqual(httpSnapshot, udsSnapshot) {
			t.Fatalf("HTTP snapshot = %#v, UDS snapshot = %#v", httpSnapshot, udsSnapshot)
		}
		invalidValue := map[string]any{"v": float64(1)}
		invalid := contract.DesktopStateApplyFrame{
			Op: "apply", Req: "it-008-invalid",
			Ops: []contract.DesktopStateApplyOp{{
				Kind: contract.DesktopStateOpPut, Key: "invalid key", Value: &invalidValue,
			}},
		}
		writeDesktopStateFrame(t, httpConn, invalid)
		writeDesktopStateFrame(t, udsConn, invalid)
		httpError := readDesktopStateFrame[contract.DesktopStateErrorFrame](t, httpConn, "error")
		udsError := readDesktopStateFrame[contract.DesktopStateErrorFrame](t, udsConn, "error")
		if !reflect.DeepEqual(httpError, udsError) || httpError.Code != contract.DesktopStateErrorInvalidKey {
			t.Fatalf("HTTP error = %#v, UDS error = %#v", httpError, udsError)
		}
		for index := range 3 {
			putDesktopState(
				t,
				ctx,
				runtimeHarness.HTTPClient,
				runtimeHarness.HTTPBaseURL,
				primaryWorkspace,
				"watch:parity:"+strconv.Itoa(index),
				map[string]any{"i": float64(index)},
			)
			httpEvent := readDesktopStateFrame[contract.DesktopStateEventFrame](t, httpConn, "event")
			udsEvent := readDesktopStateFrame[contract.DesktopStateEventFrame](t, udsConn, "event")
			if !reflect.DeepEqual(httpEvent, udsEvent) {
				t.Fatalf("HTTP event = %#v, UDS event = %#v", httpEvent, udsEvent)
			}
		}
	})

	t.Run("Should expose one identical total commit order to HTTP and UDS subscribers (IT-009)", func(t *testing.T) {
		httpSubscriber, httpSnapshot := openDesktopStateHTTPSubscriber(t, runtimeHarness, primaryWorkspace)
		udsSubscriber, udsSnapshot := openDesktopStateUDSSubscriber(t, runtimeHarness, primaryWorkspace)
		if httpSnapshot.AsOfSeq != udsSnapshot.AsOfSeq {
			t.Fatalf("snapshot fences = HTTP %d UDS %d, want identical", httpSnapshot.AsOfSeq, udsSnapshot.AsOfSeq)
		}
		writer := openDesktopStateHTTPSocket(t, runtimeHarness, primaryWorkspace)
		httpEvents := make(chan desktopStateEventCollection, 1)
		udsEvents := make(chan desktopStateEventCollection, 1)
		go collectDesktopStateEvents(httpSubscriber, 100, httpEvents)
		go collectDesktopStateEvents(udsSubscriber, 100, udsEvents)

		for index := range 100 {
			key := fmt.Sprintf("load:%03d", index)
			value := map[string]any{"i": float64(index)}
			if index%2 == 0 {
				putDesktopState(
					t,
					ctx,
					runtimeHarness.HTTPClient,
					runtimeHarness.HTTPBaseURL,
					primaryWorkspace,
					key,
					value,
				)
				continue
			}
			req := "it-009-" + strconv.Itoa(index)
			writeDesktopStateFrame(t, writer, contract.DesktopStateApplyFrame{
				Op: "apply", Req: req,
				Ops: []contract.DesktopStateApplyOp{{
					Kind: contract.DesktopStateOpPut, Key: key, Value: &value,
				}},
			})
			ack := readDesktopStateFrame[contract.DesktopStateAckFrame](t, writer, "ack")
			if ack.Req != req || len(ack.Results) != 1 || ack.Results[0].Key != key {
				t.Fatalf("ack = %#v, want req %q key %q", ack, req, key)
			}
		}

		httpResult := <-httpEvents
		udsResult := <-udsEvents
		if httpResult.err != nil || udsResult.err != nil {
			t.Fatalf("event collection errors = HTTP %v UDS %v", httpResult.err, udsResult.err)
		}
		if !reflect.DeepEqual(httpResult.events, udsResult.events) {
			t.Fatalf("HTTP and UDS event streams differ")
		}
		for index, event := range httpResult.events {
			wantSeq := uint64(httpSnapshot.AsOfSeq) + uint64(index) + 1
			if uint64(event.Entry.Seq) != wantSeq {
				t.Fatalf("event %d seq = %d, want %d", index, event.Entry.Seq, wantSeq)
			}
		}
		list := listDesktopState(
			t,
			ctx,
			runtimeHarness.HTTPClient,
			runtimeHarness.HTTPBaseURL,
			primaryWorkspace,
		)
		finalByKey := make(map[string]contract.DesktopStateEntry, 100)
		for _, entry := range list.Entries {
			if strings.HasPrefix(entry.Key, "load:") {
				finalByKey[entry.Key] = entry
			}
		}
		if len(finalByKey) != 100 {
			t.Fatalf("final load entries = %d, want 100", len(finalByKey))
		}
		for _, event := range httpResult.events {
			if got := finalByKey[event.Entry.Key]; !reflect.DeepEqual(got, event.Entry) {
				t.Fatalf("final entry for %q = %#v, want event %#v", event.Entry.Key, got, event.Entry)
			}
		}
	})

	t.Run("Should preserve entries and revisions through a real daemon restart (IT-005)", func(t *testing.T) {
		want := putDesktopState(
			t,
			ctx,
			runtimeHarness.HTTPClient,
			runtimeHarness.HTTPBaseURL,
			primaryWorkspace,
			"restart:durable",
			map[string]any{"v": float64(1), "durable": true},
		)
		stopCtx, stopCancel := context.WithTimeout(context.Background(), desktopStateIntegrationTimeout)
		defer stopCancel()
		if err := runtimeHarness.Stop(stopCtx); err != nil {
			t.Fatalf("Stop(before restart) error = %v", err)
		}
		restarted := e2etest.StartRuntimeHarness(t, e2etest.RuntimeHarnessOptions{
			BinaryPath: runtimeHarness.BinaryPath,
			HomePaths:  runtimeHarness.HomePaths,
			Workspace:  e2etest.WorkspaceSeedOptions{Root: primaryRoot},
		})
		if restarted.WorkspaceID != primaryWorkspace {
			t.Fatalf("restarted workspace id = %q, want %q", restarted.WorkspaceID, primaryWorkspace)
		}
		got := getDesktopState(
			t,
			ctx,
			restarted.HTTPClient,
			restarted.HTTPBaseURL,
			primaryWorkspace,
			"restart:durable",
		)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("entry after restart = %#v, want %#v", got, want)
		}
	})
}

type desktopStateResponse struct {
	status int
	body   []byte
}

type desktopStateEventCollection struct {
	events []contract.DesktopStateEventFrame
	err    error
}

func desktopStatePath(workspace string) string {
	return "/api/workspaces/" + url.PathEscape(workspace) + "/desktop-state"
}

func desktopStateRevision(value uint64) *contract.DesktopStateSafeNumber {
	revision := contract.DesktopStateSafeNumber(value)
	return &revision
}

func openDesktopStateHTTPSubscriber(
	t *testing.T,
	runtimeHarness *e2etest.RuntimeHarness,
	workspace string,
) (*websocket.Conn, contract.DesktopStateSnapshotFrame) {
	t.Helper()
	conn := openDesktopStateHTTPSocket(t, runtimeHarness, workspace)
	writeDesktopStateFrame(t, conn, contract.DesktopStateSubscribeFrame{Op: "sub"})
	return conn, readDesktopStateFrame[contract.DesktopStateSnapshotFrame](t, conn, "snapshot")
}

func openDesktopStateUDSSubscriber(
	t *testing.T,
	runtimeHarness *e2etest.RuntimeHarness,
	workspace string,
) (*websocket.Conn, contract.DesktopStateSnapshotFrame) {
	t.Helper()
	dialer := websocket.Dialer{
		HandshakeTimeout: desktopStateIntegrationTimeout,
		NetDialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var netDialer net.Dialer
			return netDialer.DialContext(ctx, "unix", runtimeHarness.Config.Daemon.Socket)
		},
	}
	conn := dialDesktopStateSocket(
		t,
		&dialer,
		"ws://unix"+desktopStatePath(workspace)+"/stream",
	)
	writeDesktopStateFrame(t, conn, contract.DesktopStateSubscribeFrame{Op: "sub"})
	return conn, readDesktopStateFrame[contract.DesktopStateSnapshotFrame](t, conn, "snapshot")
}

func openDesktopStateHTTPSocket(
	t *testing.T,
	runtimeHarness *e2etest.RuntimeHarness,
	workspace string,
) *websocket.Conn {
	t.Helper()
	dialer := websocket.Dialer{HandshakeTimeout: desktopStateIntegrationTimeout}
	websocketBase := "ws" + strings.TrimPrefix(runtimeHarness.HTTPBaseURL, "http")
	return dialDesktopStateSocket(
		t,
		&dialer,
		websocketBase+desktopStatePath(workspace)+"/stream",
	)
}

func dialDesktopStateSocket(
	t *testing.T,
	dialer *websocket.Dialer,
	target string,
) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), desktopStateIntegrationTimeout)
	defer cancel()
	conn, response, err := dialer.DialContext(ctx, target, nil)
	if err != nil {
		if response != nil {
			body, readErr := io.ReadAll(response.Body)
			closeErr := response.Body.Close()
			err = errors.Join(err, readErr, closeErr)
			t.Fatalf("DialContext(%q) error = %v; body=%s", target, err, body)
		}
		t.Fatalf("DialContext(%q) error = %v", target, err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("websocket.Close(%q) error = %v", target, err)
		}
	})
	return conn
}

func writeDesktopStateFrame(t *testing.T, conn *websocket.Conn, frame any) {
	t.Helper()
	if err := conn.SetWriteDeadline(time.Now().Add(desktopStateIntegrationTimeout)); err != nil {
		t.Fatalf("SetWriteDeadline() error = %v", err)
	}
	if err := conn.WriteJSON(frame); err != nil {
		t.Fatalf("WriteJSON(%T) error = %v", frame, err)
	}
}

func readDesktopStateFrame[T any](t *testing.T, conn *websocket.Conn, wantOp string) T {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(desktopStateIntegrationTimeout)); err != nil {
		t.Fatalf("SetReadDeadline() error = %v", err)
	}
	_, payload, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage(%q) error = %v", wantOp, err)
	}
	var envelope struct {
		Op string `json:"op"`
	}
	decodeDesktopStateJSON(t, payload, &envelope)
	if envelope.Op != wantOp {
		t.Fatalf("frame op = %q, want %q; payload=%s", envelope.Op, wantOp, payload)
	}
	var frame T
	decodeDesktopStateJSON(t, payload, &frame)
	return frame
}

func collectDesktopStateEvents(
	conn *websocket.Conn,
	count int,
	result chan<- desktopStateEventCollection,
) {
	collected := desktopStateEventCollection{events: make([]contract.DesktopStateEventFrame, 0, count)}
	if err := conn.SetReadDeadline(time.Now().Add(desktopStateIntegrationTimeout)); err != nil {
		collected.err = fmt.Errorf("set event collection read deadline: %w", err)
		result <- collected
		return
	}
	for range count {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			collected.err = fmt.Errorf("read event collection frame: %w", err)
			result <- collected
			return
		}
		var event contract.DesktopStateEventFrame
		if err := json.Unmarshal(payload, &event); err != nil {
			collected.err = fmt.Errorf("decode event collection frame: %w", err)
			result <- collected
			return
		}
		if event.Op != "event" {
			collected.err = fmt.Errorf("event collection op = %q, want event", event.Op)
			result <- collected
			return
		}
		collected.events = append(collected.events, event)
	}
	result <- collected
}

func putDesktopState(
	t *testing.T,
	ctx context.Context,
	client *http.Client,
	baseURL string,
	workspace string,
	key string,
	value map[string]any,
) contract.DesktopStateEntry {
	t.Helper()
	response := requestDesktopState(
		t,
		ctx,
		client,
		http.MethodPut,
		baseURL+desktopStatePath(workspace)+"/"+url.PathEscape(key),
		desktopStateJSON(t, contract.DesktopStatePutRequest{Value: value}),
	)
	if response.status != http.StatusOK {
		t.Fatalf("PUT %q status = %d, want %d; body=%s", key, response.status, http.StatusOK, response.body)
	}
	var entry contract.DesktopStateEntry
	decodeDesktopStateJSON(t, response.body, &entry)
	return entry
}

func getDesktopState(
	t *testing.T,
	ctx context.Context,
	client *http.Client,
	baseURL string,
	workspace string,
	key string,
) contract.DesktopStateEntry {
	t.Helper()
	response := requestDesktopState(
		t,
		ctx,
		client,
		http.MethodGet,
		baseURL+desktopStatePath(workspace)+"/"+url.PathEscape(key),
		nil,
	)
	if response.status != http.StatusOK {
		t.Fatalf("GET %q status = %d, want %d; body=%s", key, response.status, http.StatusOK, response.body)
	}
	var entry contract.DesktopStateEntry
	decodeDesktopStateJSON(t, response.body, &entry)
	return entry
}

func listDesktopState(
	t *testing.T,
	ctx context.Context,
	client *http.Client,
	baseURL string,
	workspace string,
) contract.DesktopStateListResponse {
	t.Helper()
	response := requestDesktopState(
		t,
		ctx,
		client,
		http.MethodGet,
		baseURL+desktopStatePath(workspace),
		nil,
	)
	if response.status != http.StatusOK {
		t.Fatalf("LIST status = %d, want %d; body=%s", response.status, http.StatusOK, response.body)
	}
	var list contract.DesktopStateListResponse
	decodeDesktopStateJSON(t, response.body, &list)
	return list
}

func requestDesktopState(
	t *testing.T,
	ctx context.Context,
	client *http.Client,
	method string,
	target string,
	body []byte,
) desktopStateResponse {
	t.Helper()
	request, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("http.NewRequestWithContext(%s %s) error = %v", method, target, err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("client.Do(%s %s) error = %v", method, target, err)
	}
	payload, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		t.Fatalf("read/close response for %s %s error = %v", method, target, err)
	}
	return desktopStateResponse{status: response.StatusCode, body: payload}
}

func desktopStateJSON(t *testing.T, value any) []byte {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal(%T) error = %v", value, err)
	}
	return payload
}

func decodeDesktopStateJSON(t *testing.T, payload []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(payload, target); err != nil {
		t.Fatalf("json.Unmarshal(%T) error = %v; payload=%s", target, err, payload)
	}
}
