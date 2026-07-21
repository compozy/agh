package cli

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/compozy/agh/internal/api/contract"
)

func TestDesktopStateCommands(t *testing.T) {
	t.Parallel()

	t.Run("Should set and get the same JSON value (UT-035)", func(t *testing.T) {
		t.Parallel()
		client := newDesktopStateCommandMemoryClient()
		deps := newTestDeps(t, client)

		setOutput, _, err := executeRootCommand(
			t, deps,
			"desktop-state", "set", "--workspace", "w1", "--key", "desktop",
			"--value", `{"a":1}`, "-o", "json",
		)
		if err != nil {
			t.Fatalf("desktop-state set error = %v", err)
		}
		var setEntry contract.DesktopStateEntry
		decodeDesktopStateCommandJSON(t, setOutput, &setEntry)
		if setEntry.Rev == 0 || setEntry.Key != "desktop" {
			t.Fatalf("set entry = %#v, want desktop with a revision", setEntry)
		}

		getOutput, _, err := executeRootCommand(
			t, deps,
			"desktop-state", "get", "--workspace", "w1", "--key", "desktop", "-o", "json",
		)
		if err != nil {
			t.Fatalf("desktop-state get error = %v", err)
		}
		var getEntry contract.DesktopStateEntry
		decodeDesktopStateCommandJSON(t, getOutput, &getEntry)
		if !reflect.DeepEqual(getEntry.Value, setEntry.Value) || getEntry.Rev != setEntry.Rev {
			t.Fatalf("get entry = %#v, want value and rev from %#v", getEntry, setEntry)
		}
	})

	t.Run("Should render a revision conflict as a structured non-zero error (UT-036)", func(t *testing.T) {
		t.Parallel()
		client := &desktopStateCommandStub{
			stubClient: &stubClient{},
			putFn: func(
				_ context.Context,
				_ string,
				key string,
				_ contract.DesktopStatePutRequest,
			) (contract.DesktopStateEntry, error) {
				return contract.DesktopStateEntry{}, newDesktopStateCommandError(
					409, contract.DesktopStateErrorRevConflict, key,
				)
			},
		}

		exitCode, _, stderr := executeRootCommandWithExit(
			t, newTestDeps(t, client),
			"desktop-state", "set", "--workspace", "w1", "--key", "desktop",
			"--value", `{"v":2}`, "--if-rev", "1", "-o", "json",
		)
		if exitCode == 0 {
			t.Fatalf("exit code = 0; stderr=%s", stderr)
		}
		var payload contract.DesktopStateErrorPayload
		decodeDesktopStateCommandJSON(t, stderr, &payload)
		if payload.Code != contract.DesktopStateErrorRevConflict || payload.Key != "desktop" {
			t.Fatalf("error payload = %#v, want revision conflict for desktop", payload)
		}
	})

	t.Run("Should list the server-provided entries as a JSON array (UT-037)", func(t *testing.T) {
		t.Parallel()
		client := &desktopStateCommandStub{
			stubClient: &stubClient{},
			listFn: func(_ context.Context, workspace string) (contract.DesktopStateListResponse, error) {
				if workspace != "w1" {
					t.Fatalf("workspace = %q, want w1", workspace)
				}
				return contract.DesktopStateListResponse{AsOfSeq: 2, Entries: []contract.DesktopStateEntry{
					desktopStateCommandEntry("desktop", 1, 1, map[string]any{"v": float64(1)}),
					desktopStateCommandEntry("win:tasks", 1, 2, map[string]any{"v": float64(1)}),
				}}, nil
			},
		}

		output, _, err := executeRootCommand(
			t, newTestDeps(t, client),
			"desktop-state", "list", "--workspace", "w1", "-o", "json",
		)
		if err != nil {
			t.Fatalf("desktop-state list error = %v", err)
		}
		var entries []contract.DesktopStateEntry
		decodeDesktopStateCommandJSON(t, output, &entries)
		if len(entries) != 2 || entries[0].Key != "desktop" || entries[1].Key != "win:tasks" {
			t.Fatalf("entries = %#v, want desktop and win:tasks array", entries)
		}
	})

	t.Run("Should emit ordered set and delete events as JSON lines (UT-038)", func(t *testing.T) {
		t.Parallel()
		client := &desktopStateCommandStub{
			stubClient: &stubClient{},
			watchFn: func(
				_ context.Context,
				workspace string,
				handler func(contract.DesktopStateEventFrame) error,
			) error {
				if workspace != "w1" {
					t.Fatalf("workspace = %q, want w1", workspace)
				}
				frames := []contract.DesktopStateEventFrame{
					{
						Op: "event", Origin: "writer-a",
						Entry: desktopStateCommandEntry("desktop", 2, 4, map[string]any{"v": float64(2)}),
					},
					{
						Op: "event", Origin: "writer-b",
						Entry: contract.DesktopStateEntry{
							Key: "win:tasks", Rev: 2, Seq: 5, Deleted: true,
							UpdatedAt: desktopStateCommandTime(),
						},
					},
				}
				for _, frame := range frames {
					if err := handler(frame); err != nil {
						return err
					}
				}
				return nil
			},
		}

		output, _, err := executeRootCommand(
			t, newTestDeps(t, client),
			"desktop-state", "watch", "--workspace", "w1", "-o", "jsonl",
		)
		if err != nil {
			t.Fatalf("desktop-state watch error = %v", err)
		}
		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) != 2 {
			t.Fatalf("watch output = %q, want two JSON lines", output)
		}
		var previousSeq contract.DesktopStateSafeNumber
		for _, line := range lines {
			var frame contract.DesktopStateEventFrame
			decodeDesktopStateCommandJSON(t, line, &frame)
			if frame.Op != "event" || frame.Entry.Seq <= previousSeq {
				t.Fatalf("frame = %#v, want event after seq %d", frame, previousSeq)
			}
			previousSeq = frame.Entry.Seq
		}
	})

	t.Run("Should render an absent key as a structured non-zero error (UT-039)", func(t *testing.T) {
		t.Parallel()
		client := &desktopStateCommandStub{
			stubClient: &stubClient{},
			getFn: func(
				_ context.Context,
				_ string,
				key string,
			) (contract.DesktopStateEntry, error) {
				return contract.DesktopStateEntry{}, newDesktopStateCommandError(
					404, contract.DesktopStateErrorNotFound, key,
				)
			},
		}

		exitCode, _, stderr := executeRootCommandWithExit(
			t, newTestDeps(t, client),
			"desktop-state", "get", "--workspace", "w1", "--key", "missing", "-o", "json",
		)
		if exitCode == 0 {
			t.Fatalf("exit code = 0; stderr=%s", stderr)
		}
		var payload contract.DesktopStateErrorPayload
		decodeDesktopStateCommandJSON(t, stderr, &payload)
		if payload.Code != contract.DesktopStateErrorNotFound || payload.Key != "missing" {
			t.Fatalf("error payload = %#v, want not found for missing", payload)
		}
	})

	t.Run("Should pass the requested revision to delete", func(t *testing.T) {
		t.Parallel()
		client := &desktopStateCommandStub{
			stubClient: &stubClient{},
			deleteFn: func(
				_ context.Context,
				workspace string,
				key string,
				ifRev *contract.DesktopStateSafeNumber,
			) error {
				if workspace != "w1" || key != "desktop" || ifRev == nil || *ifRev != 4 {
					t.Fatalf("DeleteDesktopState(%q, %q, %v), want w1 desktop rev 4", workspace, key, ifRev)
				}
				return nil
			},
		}
		output, _, err := executeRootCommand(
			t, newTestDeps(t, client),
			"desktop-state", "delete", "--workspace", "w1", "--key", "desktop",
			"--if-rev", "4", "-o", "json",
		)
		if err != nil {
			t.Fatalf("desktop-state delete error = %v", err)
		}
		var payload struct {
			Deleted bool   `json:"deleted"`
			Key     string `json:"key"`
		}
		decodeDesktopStateCommandJSON(t, output, &payload)
		if !payload.Deleted || payload.Key != "desktop" {
			t.Fatalf("delete output = %#v, want desktop deleted", payload)
		}
	})
}

type desktopStateCommandStub struct {
	*stubClient
	listFn   func(context.Context, string) (contract.DesktopStateListResponse, error)
	getFn    func(context.Context, string, string) (contract.DesktopStateEntry, error)
	putFn    func(context.Context, string, string, contract.DesktopStatePutRequest) (contract.DesktopStateEntry, error)
	applyFn  func(context.Context, string, contract.DesktopStateApplyRequest) (contract.DesktopStateApplyResponse, error)
	deleteFn func(context.Context, string, string, *contract.DesktopStateSafeNumber) error
	watchFn  func(context.Context, string, func(contract.DesktopStateEventFrame) error) error
}

func (s *desktopStateCommandStub) ListDesktopState(
	ctx context.Context,
	workspace string,
) (contract.DesktopStateListResponse, error) {
	if s.listFn == nil {
		return contract.DesktopStateListResponse{}, errors.New("unexpected ListDesktopState call")
	}
	return s.listFn(ctx, workspace)
}

func (s *desktopStateCommandStub) GetDesktopState(
	ctx context.Context,
	workspace string,
	key string,
) (contract.DesktopStateEntry, error) {
	if s.getFn == nil {
		return contract.DesktopStateEntry{}, errors.New("unexpected GetDesktopState call")
	}
	return s.getFn(ctx, workspace, key)
}

func (s *desktopStateCommandStub) PutDesktopState(
	ctx context.Context,
	workspace string,
	key string,
	request contract.DesktopStatePutRequest,
) (contract.DesktopStateEntry, error) {
	if s.putFn == nil {
		return contract.DesktopStateEntry{}, errors.New("unexpected PutDesktopState call")
	}
	return s.putFn(ctx, workspace, key, request)
}

func (s *desktopStateCommandStub) ApplyDesktopState(
	ctx context.Context,
	workspace string,
	request contract.DesktopStateApplyRequest,
) (contract.DesktopStateApplyResponse, error) {
	if s.applyFn == nil {
		return contract.DesktopStateApplyResponse{}, errors.New("unexpected ApplyDesktopState call")
	}
	return s.applyFn(ctx, workspace, request)
}

func (s *desktopStateCommandStub) DeleteDesktopState(
	ctx context.Context,
	workspace string,
	key string,
	ifRev *contract.DesktopStateSafeNumber,
) error {
	if s.deleteFn == nil {
		return errors.New("unexpected DeleteDesktopState call")
	}
	return s.deleteFn(ctx, workspace, key, ifRev)
}

func (s *desktopStateCommandStub) WatchDesktopState(
	ctx context.Context,
	workspace string,
	handler func(contract.DesktopStateEventFrame) error,
) error {
	if s.watchFn == nil {
		return errors.New("unexpected WatchDesktopState call")
	}
	return s.watchFn(ctx, workspace, handler)
}

type desktopStateCommandMemoryClient struct {
	*desktopStateCommandStub
	mu      sync.Mutex
	entries map[string]contract.DesktopStateEntry
	seq     uint64
}

func newDesktopStateCommandMemoryClient() *desktopStateCommandMemoryClient {
	client := &desktopStateCommandMemoryClient{
		desktopStateCommandStub: &desktopStateCommandStub{stubClient: &stubClient{}},
		entries:                 make(map[string]contract.DesktopStateEntry),
	}
	client.listFn = client.list
	client.getFn = client.get
	client.putFn = client.put
	client.deleteFn = client.delete
	return client
}

func (c *desktopStateCommandMemoryClient) list(
	_ context.Context,
	_ string,
) (contract.DesktopStateListResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entries := make([]contract.DesktopStateEntry, 0, len(c.entries))
	for _, entry := range c.entries {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Key < entries[j].Key })
	return contract.DesktopStateListResponse{
		AsOfSeq: contract.DesktopStateSafeNumber(c.seq), Entries: entries,
	}, nil
}

func (c *desktopStateCommandMemoryClient) get(
	_ context.Context,
	_ string,
	key string,
) (contract.DesktopStateEntry, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok {
		return contract.DesktopStateEntry{}, newDesktopStateCommandError(
			404, contract.DesktopStateErrorNotFound, key,
		)
	}
	return entry, nil
}

func (c *desktopStateCommandMemoryClient) put(
	_ context.Context,
	_ string,
	key string,
	request contract.DesktopStatePutRequest,
) (contract.DesktopStateEntry, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	current := c.entries[key]
	if request.IfRev != nil && current.Rev != *request.IfRev {
		return contract.DesktopStateEntry{}, newDesktopStateCommandError(
			409, contract.DesktopStateErrorRevConflict, key,
		)
	}
	c.seq++
	entry := desktopStateCommandEntry(
		key,
		current.Rev+1,
		contract.DesktopStateSafeNumber(c.seq),
		request.Value,
	)
	c.entries[key] = entry
	return entry, nil
}

func (c *desktopStateCommandMemoryClient) delete(
	_ context.Context,
	_ string,
	key string,
	ifRev *contract.DesktopStateSafeNumber,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok {
		return newDesktopStateCommandError(404, contract.DesktopStateErrorNotFound, key)
	}
	if ifRev != nil && entry.Rev != *ifRev {
		return newDesktopStateCommandError(409, contract.DesktopStateErrorRevConflict, key)
	}
	delete(c.entries, key)
	c.seq++
	return nil
}

func newDesktopStateCommandError(
	status int,
	code contract.DesktopStateErrorCode,
	key string,
) error {
	return &desktopStateAPIError{
		statusCode: status,
		payload: contract.DesktopStateErrorPayload{
			Error: string(code), Code: code, Key: key,
		},
	}
}

func desktopStateCommandEntry(
	key string,
	rev contract.DesktopStateSafeNumber,
	seq contract.DesktopStateSafeNumber,
	value map[string]any,
) contract.DesktopStateEntry {
	return contract.DesktopStateEntry{
		Key: key, Value: value, Rev: rev, Seq: seq, UpdatedAt: desktopStateCommandTime(),
	}
}

func desktopStateCommandTime() time.Time {
	return time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
}

func decodeDesktopStateCommandJSON(t *testing.T, value string, target any) {
	t.Helper()
	if err := json.Unmarshal([]byte(strings.TrimSpace(value)), target); err != nil {
		t.Fatalf("decode JSON: %v; value=%q", err, value)
	}
}
