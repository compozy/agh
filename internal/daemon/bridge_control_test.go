// Suite: daemon bridge control runtime
// Invariant: disabled instances resolve fresh secrets without lifecycle mutation and stay locked through control reap.
// Boundary IN: daemon bridge runtime control transport.
// Boundary OUT: subprocess restrictions, owned by extension manager tests.
package daemon

import (
	"context"
	"errors"
	gostdruntime "runtime"
	"strings"
	"sync"
	"testing"
	"time"

	bridgepkg "github.com/compozy/agh/internal/bridges"
	bridgecontract "github.com/compozy/agh/internal/bridges/contract"
	"github.com/compozy/agh/internal/subprocess"
	"github.com/compozy/agh/internal/testutil"
)

func TestBridgeRuntimeResolveBridgeControlRuntimeKeepsDisabledState(t *testing.T) {
	t.Run("Should resolve fresh secrets without changing disabled lifecycle state", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		now := time.Date(2026, 7, 12, 13, 0, 0, 0, time.UTC)
		resolver := &recordingBridgeSecretResolver{values: map[string]string{"bot_token": "secret-value"}}
		runtime := newBridgeRuntime(db, discardLogger(), func() time.Time { return now }, resolver)
		instance := mustCreateDaemonBridgeInstance(t, runtime, bridgepkg.CreateInstanceRequest{
			ID:            "brg-control-disabled",
			Scope:         bridgepkg.ScopeGlobal,
			Platform:      "telegram",
			ExtensionName: "telegram-adapter",
			DisplayName:   "Telegram",
			Enabled:       false,
			Status:        bridgepkg.BridgeStatusDisabled,
			RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
		})
		if err := db.PutBridgeSecretBinding(testutil.Context(t), bridgepkg.BridgeSecretBinding{
			BridgeInstanceID: instance.ID,
			BindingName:      "bot_token",
			SecretRef:        "vault:bridges/brg-control-disabled/bot_token",
			Kind:             "token",
			CreatedAt:        now,
			UpdatedAt:        now,
		}); err != nil {
			t.Fatalf("PutBridgeSecretBinding() error = %v", err)
		}

		launch, err := runtime.ResolveBridgeControlRuntime(
			testutil.Context(t),
			instance.ExtensionName,
			instance.ID,
			bridgepkg.ControlMethodCheck,
		)
		if err != nil {
			t.Fatalf("ResolveBridgeControlRuntime() error = %v", err)
		}
		if launch.Purpose != subprocess.BridgeRuntimePurposeControl {
			t.Fatalf("Purpose = %q, want control", launch.Purpose)
		}
		got := launch.AllowedMethods
		want := []string{string(bridgepkg.ControlMethodCheck)}
		if len(got) != 1 || got[0] != want[0] {
			t.Fatalf("AllowedMethods = %#v, want %#v", got, want)
		}
		managed, err := launch.SingleManagedInstance()
		if err != nil {
			t.Fatalf("SingleManagedInstance() error = %v", err)
		}
		if managed.Instance.Enabled || managed.Instance.Status != bridgecontract.BridgeStatusDisabled {
			t.Fatalf(
				"managed instance state = enabled %v status %q, want disabled",
				managed.Instance.Enabled,
				managed.Instance.Status,
			)
		}
		if len(managed.BoundSecrets) != 1 || managed.BoundSecrets[0].Value != "secret-value" {
			t.Fatalf("BoundSecrets = %#v, want freshly resolved bot_token", managed.BoundSecrets)
		}
		persisted, err := runtime.GetInstance(testutil.Context(t), instance.ID)
		if err != nil {
			t.Fatalf("GetInstance() error = %v", err)
		}
		if persisted.Enabled || persisted.Status != bridgepkg.BridgeStatusDisabled {
			t.Fatalf(
				"persisted state = enabled %v status %q, want unchanged disabled",
				persisted.Enabled,
				persisted.Status,
			)
		}
	})
}

func TestBridgeRuntimeCheckBridgeDoesNotChangeLifecycleState(t *testing.T) {
	t.Run("Should return provider checks while the persisted instance remains disabled", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		now := time.Date(2026, 7, 12, 13, 5, 0, 0, time.UTC)
		runtime := newBridgeRuntime(db, discardLogger(), func() time.Time { return now }, nil)
		instance := mustCreateDaemonBridgeInstance(t, runtime, bridgepkg.CreateInstanceRequest{
			ID:            "brg-control-read-only",
			Scope:         bridgepkg.ScopeGlobal,
			Platform:      "telegram",
			ExtensionName: "telegram-adapter",
			DisplayName:   "Telegram",
			Enabled:       false,
			Status:        bridgepkg.BridgeStatusDisabled,
			RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
		})
		transport := newBlockingBridgeControlExtensionRuntime()
		close(transport.release)
		runtime.setExtensionRuntime(transport)

		response, err := runtime.CheckBridge(
			testutil.Context(t),
			instance.ExtensionName,
			bridgepkg.BridgeCheckRequest{BridgeInstanceID: instance.ID},
		)
		if err != nil {
			t.Fatalf("CheckBridge() error = %v", err)
		}
		if len(response.Checks) != 1 || response.Checks[0].Status != bridgepkg.BridgeCheckStatusPass {
			t.Fatalf("CheckBridge() response = %#v", response)
		}

		persisted, err := runtime.GetInstance(testutil.Context(t), instance.ID)
		if err != nil {
			t.Fatalf("GetInstance() error = %v", err)
		}
		if persisted.Enabled || persisted.Status != bridgepkg.BridgeStatusDisabled {
			t.Fatalf(
				"persisted state = enabled %v status %q, want unchanged disabled",
				persisted.Enabled,
				persisted.Status,
			)
		}
	})
}

func TestBridgeRuntimeControlCallHoldsLifecycleLockThroughTransportCleanup(t *testing.T) {
	t.Run("Should hold the lifecycle lock through nested resolution and transport cleanup", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		now := time.Date(2026, 7, 12, 13, 10, 0, 0, time.UTC)
		runtime := newBridgeRuntime(db, discardLogger(), func() time.Time { return now }, nil)
		instance := mustCreateDaemonBridgeInstance(t, runtime, bridgepkg.CreateInstanceRequest{
			ID:            "brg-control-lock",
			Scope:         bridgepkg.ScopeGlobal,
			Platform:      "telegram",
			ExtensionName: "telegram-adapter",
			DisplayName:   "Telegram",
			Enabled:       false,
			Status:        bridgepkg.BridgeStatusDisabled,
			RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
		})
		transport := newBlockingBridgeControlExtensionRuntime()
		transport.resolve = func(
			ctx context.Context,
			extensionName string,
			request bridgepkg.BridgeCheckRequest,
		) error {
			_, err := runtime.ResolveBridgeControlRuntime(
				ctx,
				extensionName,
				request.BridgeInstanceID,
				bridgepkg.ControlMethodCheck,
			)
			return err
		}
		runtime.setExtensionRuntime(transport)

		checkErr := make(chan error, 1)
		go func() {
			_, err := runtime.CheckBridge(
				testutil.Context(t),
				instance.ExtensionName,
				bridgepkg.BridgeCheckRequest{BridgeInstanceID: instance.ID},
			)
			checkErr <- err
		}()
		select {
		case <-transport.entered:
		case <-time.After(time.Second):
			t.Fatal("control transport did not start")
		}

		putDone := make(chan error, 1)
		putAttempted := make(chan struct{})
		go func() {
			close(putAttempted)
			putDone <- runtime.PutSecretBinding(testutil.Context(t), bridgepkg.BridgeSecretBinding{
				BridgeInstanceID: instance.ID,
				BindingName:      "bot_token",
				SecretRef:        "vault:bridges/brg-control-lock/bot_token",
				Kind:             "token",
				CreatedAt:        now,
				UpdatedAt:        now,
			}, nil)
		}()
		<-putAttempted
		select {
		case err := <-putDone:
			t.Fatalf("PutSecretBinding() completed before control cleanup: %v", err)
		case <-time.After(30 * time.Millisecond):
		}

		close(transport.release)
		if err := <-checkErr; err != nil {
			t.Fatalf("CheckBridge() error = %v", err)
		}
		if err := <-putDone; err != nil {
			t.Fatalf("PutSecretBinding() error = %v", err)
		}
	})
}

func TestBridgeRuntimeControlLockWaitHonorsCancellation(t *testing.T) {
	t.Run("Should cancel a queued call without entering the provider transport", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		runtime := newBridgeRuntime(db, discardLogger(), time.Now, nil)
		instance := mustCreateDaemonBridgeInstance(t, runtime, bridgepkg.CreateInstanceRequest{
			ID:            "brg-control-cancel",
			Scope:         bridgepkg.ScopeGlobal,
			Platform:      "telegram",
			ExtensionName: "telegram-adapter",
			DisplayName:   "Telegram",
			Enabled:       false,
			Status:        bridgepkg.BridgeStatusDisabled,
			RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
		})
		transport := newBlockingBridgeControlExtensionRuntime()
		defer closeBridgeControlRelease(transport.release)
		runtime.setExtensionRuntime(transport)

		firstDone := make(chan error, 1)
		go func() {
			_, err := runtime.CheckBridge(
				testutil.Context(t),
				instance.ExtensionName,
				bridgepkg.BridgeCheckRequest{BridgeInstanceID: instance.ID},
			)
			firstDone <- err
		}()
		select {
		case <-transport.entered:
		case <-time.After(time.Second):
			t.Fatal("first control transport did not start")
		}

		queuedCtx, cancel := context.WithCancel(testutil.Context(t))
		secondDone := make(chan error, 1)
		go func() {
			_, err := runtime.CheckBridge(
				queuedCtx,
				instance.ExtensionName,
				bridgepkg.BridgeCheckRequest{BridgeInstanceID: instance.ID},
			)
			secondDone <- err
		}()
		waitForBridgeLifecycleRefs(t, runtime, instance.ID, 2)
		cancel()

		select {
		case err := <-secondDone:
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("queued CheckBridge() error = %v, want context canceled", err)
			}
		case <-time.After(time.Second):
			t.Fatal("queued CheckBridge() did not honor cancellation")
		}
		transport.mu.Lock()
		calls := transport.calls
		transport.mu.Unlock()
		if calls != 1 {
			t.Fatalf("provider transport calls = %d, want only the active call", calls)
		}

		closeBridgeControlRelease(transport.release)
		if err := <-firstDone; err != nil {
			t.Fatalf("first CheckBridge() error = %v", err)
		}
	})
}

func TestBridgeRuntimeControlAllowsDifferentInstancesForOneExtension(t *testing.T) {
	t.Run("Should run distinct instances concurrently while keeping per-instance exclusion", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		runtime := newBridgeRuntime(db, discardLogger(), time.Now, nil)
		instances := make([]*bridgepkg.BridgeInstance, 0, 2)
		for _, id := range []string{"brg-control-a", "brg-control-b"} {
			instances = append(instances, mustCreateDaemonBridgeInstance(t, runtime, bridgepkg.CreateInstanceRequest{
				ID:            id,
				Scope:         bridgepkg.ScopeGlobal,
				Platform:      "telegram",
				ExtensionName: "telegram-adapter",
				DisplayName:   id,
				Enabled:       false,
				Status:        bridgepkg.BridgeStatusDisabled,
				RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
			}))
		}
		transport := newBlockingBridgeControlExtensionRuntime()
		defer closeBridgeControlRelease(transport.release)
		runtime.setExtensionRuntime(transport)

		done := make(chan error, len(instances))
		for _, instance := range instances {
			go func() {
				_, err := runtime.CheckBridge(
					testutil.Context(t),
					instance.ExtensionName,
					bridgepkg.BridgeCheckRequest{BridgeInstanceID: instance.ID},
				)
				done <- err
			}()
		}

		entered := make(map[string]struct{}, len(instances))
		for range instances {
			select {
			case id := <-transport.enteredCalls:
				entered[id] = struct{}{}
			case <-time.After(time.Second):
				t.Fatalf("concurrent provider entries = %#v, want both instances", entered)
			}
		}
		if len(entered) != len(instances) {
			t.Fatalf("concurrent provider entries = %#v, want two distinct instances", entered)
		}

		closeBridgeControlRelease(transport.release)
		for range instances {
			if err := <-done; err != nil {
				t.Fatalf("CheckBridge() error = %v", err)
			}
		}
	})
}

func TestBridgeRuntimeControlRejectsExtensionMismatchBeforeTransport(t *testing.T) {
	t.Run("Should reject cross-extension instance access before provider transport", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		runtime := newBridgeRuntime(db, discardLogger(), time.Now, nil)
		instance := mustCreateDaemonBridgeInstance(t, runtime, bridgepkg.CreateInstanceRequest{
			ID:            "brg-control-owner",
			Scope:         bridgepkg.ScopeGlobal,
			Platform:      "telegram",
			ExtensionName: "telegram-adapter",
			DisplayName:   "Telegram",
			Enabled:       false,
			Status:        bridgepkg.BridgeStatusDisabled,
			RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
		})
		transport := newBlockingBridgeControlExtensionRuntime()
		close(transport.release)
		runtime.setExtensionRuntime(transport)

		_, err := runtime.CheckBridge(
			testutil.Context(t),
			"slack-adapter",
			bridgepkg.BridgeCheckRequest{BridgeInstanceID: instance.ID},
		)
		if err == nil || !strings.Contains(err.Error(), "belongs to extension") {
			t.Fatalf("CheckBridge() error = %v, want extension ownership error", err)
		}
		transport.mu.Lock()
		calls := transport.calls
		transport.mu.Unlock()
		if calls != 0 {
			t.Fatalf("control transport calls = %d, want zero", calls)
		}
	})
}

type blockingBridgeControlExtensionRuntime struct {
	fakeExtensionRuntime
	mu           sync.Mutex
	calls        int
	entered      chan struct{}
	enteredCalls chan string
	release      chan struct{}
	once         sync.Once
	resolve      func(context.Context, string, bridgepkg.BridgeCheckRequest) error
}

func newBlockingBridgeControlExtensionRuntime() *blockingBridgeControlExtensionRuntime {
	return &blockingBridgeControlExtensionRuntime{
		entered:      make(chan struct{}),
		enteredCalls: make(chan string, 8),
		release:      make(chan struct{}),
	}
}

func (r *blockingBridgeControlExtensionRuntime) CheckBridge(
	ctx context.Context,
	extensionName string,
	request bridgepkg.BridgeCheckRequest,
) (bridgepkg.BridgeCheckResponse, error) {
	if err := request.Validate(); err != nil {
		return bridgepkg.BridgeCheckResponse{}, err
	}
	if r.resolve != nil {
		if err := r.resolve(ctx, extensionName, request); err != nil {
			return bridgepkg.BridgeCheckResponse{}, err
		}
	}
	r.mu.Lock()
	r.calls++
	r.mu.Unlock()
	r.once.Do(func() { close(r.entered) })
	r.enteredCalls <- request.BridgeInstanceID
	select {
	case <-r.release:
	case <-ctx.Done():
		return bridgepkg.BridgeCheckResponse{}, ctx.Err()
	}
	return bridgepkg.BridgeCheckResponse{
		Checks: []bridgepkg.BridgeCheckRecord{{
			Check: "provider.identity", Status: bridgepkg.BridgeCheckStatusPass,
		}},
	}, nil
}

func waitForBridgeLifecycleRefs(t *testing.T, runtime *bridgeRuntime, instanceID string, want int) {
	t.Helper()

	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	for {
		runtime.lifecycleMu.Lock()
		refs := 0
		if lock := runtime.lifecycleLocks[instanceID]; lock != nil {
			refs = lock.refs
		}
		runtime.lifecycleMu.Unlock()
		if refs >= want {
			return
		}
		select {
		case <-deadline.C:
			t.Fatalf("lifecycle refs for %q = %d, want at least %d", instanceID, refs, want)
		default:
			gostdruntime.Gosched()
		}
	}
}

func closeBridgeControlRelease(release chan struct{}) {
	select {
	case <-release:
	default:
		close(release)
	}
}

func (r *blockingBridgeControlExtensionRuntime) RegisterBridgeWebhook(
	context.Context,
	string,
	bridgepkg.BridgeWebhookRegistrationRequest,
) (bridgepkg.BridgeWebhookRegistrationResponse, error) {
	return bridgepkg.BridgeWebhookRegistrationResponse{
		Status: bridgepkg.BridgeCheckStatusPass,
	}, nil
}
