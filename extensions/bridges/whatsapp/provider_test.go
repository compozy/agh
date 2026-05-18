package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/bridgesdk"
	extensioncontract "github.com/pedronauck/agh/internal/extension/contract"
	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
	"github.com/pedronauck/agh/internal/subprocess"
)

func TestMapWhatsAppInboundMessageAndDMPolicy(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	managed := testBridgeRuntime(now, "brg-whatsapp")
	contact := &whatsappContact{WaID: "15551234567"}
	contact.Profile.Name = "Alice Example"

	envelope, err := mapWhatsAppInboundMessage(whatsappInboundMessage{
		ID:        "wamid.abc123",
		From:      "15551234567",
		Timestamp: strconvTime(now),
		Type:      "image",
		Context: &struct {
			From string `json:"from,omitempty"`
			ID   string `json:"id,omitempty"`
		}{
			From: "15557654321",
			ID:   "wamid.parent",
		},
		Image: &struct {
			ID       string `json:"id,omitempty"`
			MIMEType string `json:"mime_type,omitempty"`
			Caption  string `json:"caption,omitempty"`
			SHA256   string `json:"sha256,omitempty"`
		}{
			ID:       "media-1",
			MIMEType: "image/jpeg",
			Caption:  "Need a summary",
		},
	}, contact, &managed, time.Time{}, "123456789")
	if err != nil {
		t.Fatalf("mapWhatsAppInboundMessage() error = %v", err)
	}
	if got, want := envelope.PeerID, "15551234567"; got != want {
		t.Fatalf("envelope.PeerID = %q, want %q", got, want)
	}
	if got := envelope.GroupID; got != "" {
		t.Fatalf("envelope.GroupID = %q, want empty", got)
	}
	if got := envelope.ThreadID; got != "" {
		t.Fatalf("envelope.ThreadID = %q, want empty", got)
	}
	if got, want := envelope.Content.Text, "Need a summary"; got != want {
		t.Fatalf("envelope.Content.Text = %q, want %q", got, want)
	}
	if got, want := envelope.Sender.DisplayName, "Alice Example"; got != want {
		t.Fatalf("envelope.Sender.DisplayName = %q, want %q", got, want)
	}
	if got, want := len(envelope.Attachments), 1; got != want {
		t.Fatalf("len(envelope.Attachments) = %d, want %d", got, want)
	}

	sender := envelope.Sender
	if !allowWhatsAppDirectMessage(
		resolvedInstanceConfig{dmPolicy: bridgepkg.BridgeDMPolicyOpen},
		sender,
	) {
		t.Fatal("allowWhatsAppDirectMessage(open) = false, want true")
	}
	if !allowWhatsAppDirectMessage(resolvedInstanceConfig{
		dmPolicy:       bridgepkg.BridgeDMPolicyAllowlist,
		allowUserIDs:   map[string]struct{}{"15551234567": {}},
		allowUsernames: map[string]struct{}{"alice example": {}},
	}, sender) {
		t.Fatal("allowWhatsAppDirectMessage(allowlist) = false, want true")
	}
	if !allowWhatsAppDirectMessage(resolvedInstanceConfig{
		dmPolicy:        bridgepkg.BridgeDMPolicyPairing,
		pairedUsernames: map[string]struct{}{"alice example": {}},
	}, sender) {
		t.Fatal("allowWhatsAppDirectMessage(pairing) = false, want true")
	}
	if allowWhatsAppDirectMessage(resolvedInstanceConfig{
		dmPolicy: bridgepkg.BridgeDMPolicyAllowlist,
	}, sender) {
		t.Fatal("allowWhatsAppDirectMessage(rejected) = true, want false")
	}
}

func TestVerifyChallengeAndSignature(t *testing.T) {
	t.Parallel()

	body := []byte(whatsappWebhookPayloadForPhone("123456789", "hello"))
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"http://example.test/whatsapp/brg-1",
		strings.NewReader(string(body)),
	)
	req.Header.Set(whatsappSignatureHeader, signWhatsAppBody(body, "top-secret"))
	if err := verifyWhatsAppSignature(context.Background(), req, body, "top-secret"); err != nil {
		t.Fatalf("verifyWhatsAppSignature(valid) error = %v", err)
	}
	if err := verifyWhatsAppSignature(context.Background(), req, body, "wrong"); err == nil {
		t.Fatal("verifyWhatsAppSignature(invalid) error = nil, want non-nil")
	}

	provider, err := newWhatsAppProvider(io.Discard)
	if err != nil {
		t.Fatalf("newWhatsAppProvider() error = %v", err)
	}
	cfg := resolvedInstanceConfig{verifyToken: "verify-me"}

	okReq := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"http://example.test/whatsapp/brg-1?hub.mode=subscribe&hub.verify_token=verify-me&hub.challenge=12345",
		http.NoBody,
	)
	okResp := httptest.NewRecorder()
	provider.handleVerifyChallenge(okResp, okReq, cfg)
	if got, want := okResp.Code, http.StatusOK; got != want {
		t.Fatalf("verify challenge status = %d, want %d", got, want)
	}
	if got, want := strings.TrimSpace(okResp.Body.String()), "12345"; got != want {
		t.Fatalf("verify challenge body = %q, want %q", got, want)
	}

	badReq := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"http://example.test/whatsapp/brg-1?hub.mode=subscribe&hub.verify_token=wrong&hub.challenge=12345",
		http.NoBody,
	)
	badResp := httptest.NewRecorder()
	provider.handleVerifyChallenge(badResp, badReq, cfg)
	if got, want := badResp.Code, http.StatusForbidden; got != want {
		t.Fatalf("verify challenge forbidden status = %d, want %d", got, want)
	}
}

func TestSplitMessage(t *testing.T) {
	t.Parallel()

	short := splitMessage("hello")
	if got, want := len(short), 1; got != want {
		t.Fatalf("len(splitMessage(short)) = %d, want %d", got, want)
	}

	long := strings.Repeat("a", whatsappMessageLimit+10)
	chunks := splitMessage(long)
	if got, want := len(chunks), 2; got != want {
		t.Fatalf("len(splitMessage(long)) = %d, want %d", got, want)
	}
	if got, want := len(chunks[0]), whatsappMessageLimit; got != want {
		t.Fatalf("len(chunks[0]) = %d, want %d", got, want)
	}
	if strings.Join(chunks, "") != long {
		t.Fatalf(
			"splitMessage() lost content: got len %d want len %d",
			len(strings.Join(chunks, "")),
			len(long),
		)
	}
}

func TestExecuteWhatsAppDeliveryPostResumeDeleteAndSplit(t *testing.T) {
	t.Parallel()

	api := &fakeWhatsAppAPI{nextMessageID: 500}
	cfg := resolvedInstanceConfig{
		instanceID:    "brg-1",
		phoneNumberID: "123456789",
	}

	startReq := testDeliveryRequest(
		"brg-1",
		"delivery-1",
		1,
		bridgepkg.DeliveryEventTypeStart,
		false,
		"hello",
	)
	startAck, state, err := executeWhatsAppDelivery(
		context.Background(),
		api,
		cfg,
		startReq,
		deliveryState{},
	)
	if err != nil {
		t.Fatalf("executeWhatsAppDelivery(start) error = %v", err)
	}
	if got, want := startAck.RemoteMessageID, "wamid.500"; got != want {
		t.Fatalf("startAck.RemoteMessageID = %q, want %q", got, want)
	}

	finalReq := testDeliveryRequest(
		"brg-1",
		"delivery-1",
		2,
		bridgepkg.DeliveryEventTypeFinal,
		true,
		"hello world",
	)
	finalAck, state, err := executeWhatsAppDelivery(context.Background(), api, cfg, finalReq, state)
	if err != nil {
		t.Fatalf("executeWhatsAppDelivery(final) error = %v", err)
	}
	if got, want := finalAck.ReplaceRemoteMessageID, startAck.RemoteMessageID; got != want {
		t.Fatalf("finalAck.ReplaceRemoteMessageID = %q, want %q", got, want)
	}

	deleteReq := testDeleteRequest("brg-1", "delivery-1", 3, finalAck.RemoteMessageID)
	if _, _, err := executeWhatsAppDelivery(context.Background(), api, cfg, deleteReq, state); err == nil {
		t.Fatal("executeWhatsAppDelivery(delete) error = nil, want non-nil")
	}

	resumeSnapshotReq := testDeliveryRequest(
		"brg-1",
		"delivery-2",
		1,
		bridgepkg.DeliveryEventTypeResume,
		true,
		"hello",
	)
	resumeSnapshotReq.Event.Resume = &bridgepkg.DeliveryResumeState{
		LatestEventType: bridgepkg.DeliveryEventTypeFinal,
	}
	resumeSnapshotReq.Snapshot = &bridgepkg.DeliverySnapshot{
		DeliveryID:       "delivery-2",
		SessionID:        "sess-1",
		TurnID:           "turn-1",
		BridgeInstanceID: "brg-1",
		RoutingKey:       resumeSnapshotReq.Event.RoutingKey,
		DeliveryTarget:   resumeSnapshotReq.Event.DeliveryTarget,
		LatestSeq:        1,
		LastSentSeq:      1,
		LastAckedSeq:     1,
		LatestEventType:  bridgepkg.DeliveryEventTypeFinal,
		CurrentContent:   bridgepkg.MessageContent{Text: "hello"},
		RemoteMessageID:  "wamid.resume",
		Final:            true,
		UpdatedAt:        time.Date(2026, 4, 15, 12, 5, 0, 0, time.UTC),
	}
	resumeAck, _, err := executeWhatsAppDelivery(
		context.Background(),
		api,
		cfg,
		resumeSnapshotReq,
		deliveryState{},
	)
	if err != nil {
		t.Fatalf("executeWhatsAppDelivery(resume with snapshot remote) error = %v", err)
	}
	if got, want := resumeAck.RemoteMessageID, "wamid.resume"; got != want {
		t.Fatalf("resumeAck.RemoteMessageID = %q, want %q", got, want)
	}

	splitAPI := &fakeWhatsAppAPI{nextMessageID: 900}
	splitReq := testDeliveryRequest(
		"brg-1",
		"delivery-3",
		1,
		bridgepkg.DeliveryEventTypeStart,
		false,
		strings.Repeat("a", whatsappMessageLimit+20),
	)
	splitAck, _, err := executeWhatsAppDelivery(
		context.Background(),
		splitAPI,
		cfg,
		splitReq,
		deliveryState{},
	)
	if err != nil {
		t.Fatalf("executeWhatsAppDelivery(split) error = %v", err)
	}
	if got, want := splitAck.RemoteMessageID, "wamid.901"; got != want {
		t.Fatalf("splitAck.RemoteMessageID = %q, want %q", got, want)
	}
	if got, want := len(splitAPI.requests), 2; got != want {
		t.Fatalf("len(splitAPI.requests) = %d, want %d", got, want)
	}
}

func TestClassifyWhatsAppHTTPError(t *testing.T) {
	t.Parallel()

	rate := classifyWhatsAppHTTPError(
		http.StatusTooManyRequests,
		"5",
		[]byte(`{"error":{"message":"slow down","code":130429}}`),
	)
	var rateErr *bridgesdk.RateLimitError
	if !errors.As(rate, &rateErr) {
		t.Fatalf("classifyWhatsAppHTTPError(rate) = %T, want *RateLimitError", rate)
	}
	if got, want := rateErr.RetryAfter, 5*time.Second; got != want {
		t.Fatalf("rateErr.RetryAfter = %s, want %s", got, want)
	}

	auth := classifyWhatsAppHTTPError(
		http.StatusUnauthorized,
		"",
		[]byte(`{"error":{"message":"invalid token","code":190}}`),
	)
	var authErr *bridgesdk.AuthError
	if !errors.As(auth, &authErr) {
		t.Fatalf("classifyWhatsAppHTTPError(auth) = %T, want *AuthError", auth)
	}

	transient := classifyWhatsAppHTTPError(
		http.StatusBadGateway,
		"",
		[]byte(`{"error":{"message":"upstream failed","code":2}}`),
	)
	var transientErr *bridgesdk.TransientError
	if !errors.As(transient, &transientErr) {
		t.Fatalf("classifyWhatsAppHTTPError(transient) = %T, want *TransientError", transient)
	}

	permanent := classifyWhatsAppHTTPError(
		http.StatusBadRequest,
		"",
		[]byte(`{"error":{"message":"bad request","code":100}}`),
	)
	var httpErr *bridgesdk.HTTPError
	if !errors.As(permanent, &httpErr) {
		t.Fatalf("classifyWhatsAppHTTPError(http) = %T, want *HTTPError", permanent)
	}
	if got, want := httpErr.Message, "bad request"; got != want {
		t.Fatalf("httpErr.Message = %q, want %q", got, want)
	}
}

func TestResolveInstanceConfigAndDetermineInitialState(t *testing.T) {
	env := setProviderTestEnv(t)
	_ = env
	listenAddr := reserveListenAddr(t)

	runtime, hostPeer, cleanup := newRuntimePeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 15, 14, 0, 0, 0, time.UTC)
	managed := testBridgeRuntime(now, "brg-1")
	managed.Instance.DMPolicy = bridgepkg.BridgeDMPolicyPairing
	managed.Instance.ProviderConfig = fmt.Appendf(nil, `{
		"api_base_url":"http://api.example/",
		"api_version":"v99.0",
		"phone_number_id":"123456789",
		"webhook":{"listen_addr":%q,"path":"whatsapp"},
		"batching":{"delay_ms":5,"split_delay_ms":7,"split_threshold":2},
		"dm":{"allow_user_ids":["15551234567"],"allow_usernames":["Alice Example"],"paired_usernames":["Bob"]}
	}`, listenAddr)
	managed.BoundSecrets = []subprocess.InitializeBridgeBoundSecret{
		{BindingName: "access_token", Kind: "token", Value: "access-token"},
		{BindingName: "app_secret", Kind: "token", Value: "app-secret"},
		{BindingName: "verify_token", Kind: "token", Value: "verify-token"},
	}

	mustHandleLifecycle(t, hostPeer, managed)
	if err := hostPeer.Call(context.Background(), "initialize", testInitializeRequest(now, managed), nil); err != nil {
		t.Fatalf("hostPeer.Call(initialize) error = %v", err)
	}
	waitForCondition(t, func() bool {
		runtime.mu.RLock()
		defer runtime.mu.RUnlock()
		return runtime.server != nil &&
			strings.TrimSpace(runtime.serverAddr) != "" &&
			runtime.reportedStatus["brg-1"] != ""
	})

	session := runtime.currentSession()
	if session == nil {
		t.Fatal("runtime.currentSession() = nil, want session")
	}

	cfg := runtime.resolveInstanceConfig(session, managed)
	if cfg.configError != nil {
		t.Fatalf("resolveInstanceConfig() configError = %v, want nil", cfg.configError)
	}
	defer cfg.batcher.Close()

	if got, want := cfg.apiBaseURL, "http://api.example"; got != want {
		t.Fatalf("cfg.apiBaseURL = %q, want %q", got, want)
	}
	if got, want := cfg.apiVersion, "v99.0"; got != want {
		t.Fatalf("cfg.apiVersion = %q, want %q", got, want)
	}
	if got, want := cfg.phoneNumberID, "123456789"; got != want {
		t.Fatalf("cfg.phoneNumberID = %q, want %q", got, want)
	}
	if got, want := cfg.webhookPath, "/whatsapp"; got != want {
		t.Fatalf("cfg.webhookPath = %q, want %q", got, want)
	}
	if got, want := cfg.accessToken, "access-token"; got != want {
		t.Fatalf("cfg.accessToken = %q, want %q", got, want)
	}
	if got, want := cfg.appSecret, "app-secret"; got != want {
		t.Fatalf("cfg.appSecret = %q, want %q", got, want)
	}
	if got, want := cfg.verifyToken, "verify-token"; got != want {
		t.Fatalf("cfg.verifyToken = %q, want %q", got, want)
	}
	if cfg.batcher == nil {
		t.Fatal("cfg.batcher = nil, want batcher")
	}
	if _, ok := cfg.allowUserIDs["15551234567"]; !ok {
		t.Fatalf("cfg.allowUserIDs = %#v, want normalized user id", cfg.allowUserIDs)
	}
	if _, ok := cfg.allowUsernames["alice example"]; !ok {
		t.Fatalf("cfg.allowUsernames = %#v, want normalized username", cfg.allowUsernames)
	}

	status, degradation, err := runtime.determineInitialState(
		context.Background(),
		resolvedInstanceConfig{
			instanceID:  "bad-config",
			configError: errors.New("bad config"),
		},
	)
	if err == nil {
		t.Fatal("determineInitialState(configError) error = nil, want non-nil")
	}
	if got, want := status, bridgepkg.BridgeStatusDegraded; got != want {
		t.Fatalf("status = %q, want %q", got, want)
	}
	if degradation == nil ||
		degradation.Reason != bridgepkg.BridgeDegradationReasonTenantConfigInvalid {
		t.Fatalf("degradation = %#v, want tenant config invalid", degradation)
	}

	status, degradation, err = runtime.determineInitialState(
		context.Background(),
		resolvedInstanceConfig{
			instanceID:    "missing-auth",
			phoneNumberID: "123456789",
			verifyToken:   "verify-token",
			appSecret:     "app-secret",
		},
	)
	if err == nil {
		t.Fatal("determineInitialState(missing auth) error = nil, want non-nil")
	}
	if got, want := status, bridgepkg.BridgeStatusAuthRequired; got != want {
		t.Fatalf("status = %q, want %q", got, want)
	}
	if degradation == nil || degradation.Reason != bridgepkg.BridgeDegradationReasonAuthFailed {
		t.Fatalf("degradation = %#v, want auth failed", degradation)
	}

	runtime.apiFactory = func(cfg resolvedInstanceConfig) whatsappAPI {
		switch cfg.instanceID {
		case "auth":
			return fakeWhatsAppAPIError{err: &bridgesdk.AuthError{Err: errors.New("invalid token")}}
		case "rate":
			return fakeWhatsAppAPIError{
				err: &bridgesdk.RateLimitError{
					Err:        errors.New("slow down"),
					RetryAfter: time.Second,
				},
			}
		default:
			return &fakeWhatsAppAPI{}
		}
	}

	status, degradation, err = runtime.determineInitialState(
		context.Background(),
		resolvedInstanceConfig{
			instanceID:    "auth",
			phoneNumberID: "123456789",
			accessToken:   "access-token",
			appSecret:     "app-secret",
			verifyToken:   "verify-token",
		},
	)
	if err == nil {
		t.Fatal("determineInitialState(auth) error = nil, want non-nil")
	}
	if got, want := status, bridgepkg.BridgeStatusAuthRequired; got != want {
		t.Fatalf("status = %q, want %q", got, want)
	}
	if degradation == nil || degradation.Reason != bridgepkg.BridgeDegradationReasonAuthFailed {
		t.Fatalf("degradation = %#v, want auth failed", degradation)
	}

	status, degradation, err = runtime.determineInitialState(
		context.Background(),
		resolvedInstanceConfig{
			instanceID:    "rate",
			phoneNumberID: "123456789",
			accessToken:   "access-token",
			appSecret:     "app-secret",
			verifyToken:   "verify-token",
		},
	)
	if err == nil {
		t.Fatal("determineInitialState(rate) error = nil, want non-nil")
	}
	if got, want := status, bridgepkg.BridgeStatusDegraded; got != want {
		t.Fatalf("status = %q, want %q", got, want)
	}
	if degradation == nil || degradation.Reason != bridgepkg.BridgeDegradationReasonRateLimited {
		t.Fatalf("degradation = %#v, want rate limited", degradation)
	}
}

func TestRuntimeInitializeStartsServerAndWritesMarkers(t *testing.T) {
	env := setProviderTestEnv(t)
	listenAddr := reserveListenAddr(t)
	mockAPI := newWhatsAppAPIServer(t)
	t.Setenv(whatsappListenAddrEnv, listenAddr)
	t.Setenv(whatsappAPIBaseEnv, mockAPI.URL())

	runtime, hostPeer, cleanup := newRuntimePeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 15, 13, 0, 0, 0, time.UTC)
	managed := []subprocess.InitializeBridgeManagedInstance{
		testBridgeRuntime(now, "brg-1"),
		testBridgeRuntime(now, "brg-2"),
	}
	mustHandleLifecycle(t, hostPeer, managed...)

	if err := hostPeer.Call(
		context.Background(),
		"initialize",
		testInitializeRequest(now, managed...),
		nil,
	); err != nil {
		t.Fatalf("hostPeer.Call(initialize) error = %v", err)
	}

	handshake := waitForJSONFile[initializeMarker](t, env.handshakePath)
	if got, want := handshake.Request.Runtime.Bridge.Provider, "whatsapp"; got != want {
		t.Fatalf("handshake provider = %q, want %q", got, want)
	}
	ownership := waitForJSONFile[ownershipMarker](t, env.ownershipPath)
	if got, want := len(ownership.Fetched), 2; got != want {
		t.Fatalf("len(ownership.Fetched) = %d, want %d", got, want)
	}
	states := waitForJSONLinesFile[stateMarker](
		t,
		env.statePath,
		func(items []stateMarker) bool { return len(items) >= 2 },
	)
	if got, want := states[0].Status.Normalize(), bridgepkg.BridgeStatusReady; got != want {
		t.Fatalf("states[0].Status = %q, want %q", got, want)
	}
	waitForCondition(t, func() bool {
		runtime.mu.RLock()
		defer runtime.mu.RUnlock()
		return strings.TrimSpace(runtime.serverAddr) != ""
	})
}

func TestWebhookIngressRejectsInvalidSignatureAndIngestsMessage(t *testing.T) {
	env := setProviderTestEnv(t)
	listenAddr := reserveListenAddr(t)
	mockAPI := newWhatsAppAPIServer(t)
	t.Setenv(whatsappListenAddrEnv, listenAddr)
	t.Setenv(whatsappAPIBaseEnv, mockAPI.URL())

	runtime, hostPeer, cleanup := newRuntimePeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 15, 13, 5, 0, 0, time.UTC)
	managed := testBridgeRuntime(now, "brg-1")
	mustHandleLifecycle(t, hostPeer, managed)

	var ingested []bridgepkg.InboundMessageEnvelope
	var mu sync.Mutex
	mustHandle(
		t,
		hostPeer,
		string(extensionprotocol.HostAPIMethodBridgesMessagesIngest),
		func(_ context.Context, params json.RawMessage) (any, error) {
			var envelope bridgepkg.InboundMessageEnvelope
			if err := json.Unmarshal(params, &envelope); err != nil {
				return nil, err
			}
			mu.Lock()
			ingested = append(ingested, envelope)
			mu.Unlock()
			return extensioncontract.BridgesMessagesIngestResult{
				SessionID:    "sess-1",
				RouteCreated: true,
				RoutingKey: bridgepkg.RoutingKey{
					Scope:            envelope.Scope,
					WorkspaceID:      envelope.WorkspaceID,
					BridgeInstanceID: envelope.BridgeInstanceID,
					PeerID:           envelope.PeerID,
				},
			}, nil
		},
	)

	if err := hostPeer.Call(context.Background(), "initialize", testInitializeRequest(now, managed), nil); err != nil {
		t.Fatalf("hostPeer.Call(initialize) error = %v", err)
	}
	waitForCondition(t, func() bool {
		runtime.mu.RLock()
		defer runtime.mu.RUnlock()
		return strings.TrimSpace(runtime.serverAddr) != ""
	})

	runtime.mu.RLock()
	serverAddr := runtime.serverAddr
	runtime.mu.RUnlock()
	webhookURL := "http://" + serverAddr + "/whatsapp/brg-1"
	body := whatsappWebhookPayloadForPhone("123456789", "Need a summary")

	invalidReq, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		webhookURL,
		strings.NewReader(body),
	)
	if err != nil {
		t.Fatalf("http.NewRequest(invalid) error = %v", err)
	}
	invalidReq.Header.Set("Content-Type", "application/json")
	invalidReq.Header.Set(whatsappSignatureHeader, signWhatsAppBody([]byte(body), "wrong-secret"))
	invalidResp, err := http.DefaultClient.Do(invalidReq)
	if err != nil {
		t.Fatalf("http.DefaultClient.Do(invalid) error = %v", err)
	}
	defer func() {
		if closeErr := invalidResp.Body.Close(); closeErr != nil {
			t.Fatalf("invalidResp.Body.Close() error = %v", closeErr)
		}
	}()
	if got, want := invalidResp.StatusCode, http.StatusUnauthorized; got != want {
		t.Fatalf("invalid webhook status = %d, want %d", got, want)
	}

	validReq, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		webhookURL,
		strings.NewReader(body),
	)
	if err != nil {
		t.Fatalf("http.NewRequest(valid) error = %v", err)
	}
	validReq.Header.Set("Content-Type", "application/json")
	validReq.Header.Set(whatsappSignatureHeader, signWhatsAppBody([]byte(body), "app-secret"))
	validResp, err := http.DefaultClient.Do(validReq)
	if err != nil {
		t.Fatalf("http.DefaultClient.Do(valid) error = %v", err)
	}
	defer func() {
		if closeErr := validResp.Body.Close(); closeErr != nil {
			t.Fatalf("validResp.Body.Close() error = %v", closeErr)
		}
	}()
	if got, want := validResp.StatusCode, http.StatusOK; got != want {
		t.Fatalf("valid webhook status = %d, want %d", got, want)
	}

	ingests := waitForJSONLinesFile[ingestMarker](
		t,
		env.ingestPath,
		func(items []ingestMarker) bool {
			return len(items) == 1 && strings.TrimSpace(items[0].Result.SessionID) != ""
		},
	)
	if got, want := ingests[0].Envelope.PeerID, "15551234567"; got != want {
		t.Fatalf("ingests[0].Envelope.PeerID = %q, want %q", got, want)
	}
	if got, want := ingests[0].Envelope.Content.Text, "Need a summary"; got != want {
		t.Fatalf("ingests[0].Envelope.Content.Text = %q, want %q", got, want)
	}
	mu.Lock()
	if got, want := len(ingested), 1; got != want {
		t.Fatalf("len(ingested) = %d, want %d", got, want)
	}
	mu.Unlock()

	verifyReq, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"http://"+serverAddr+"/whatsapp/brg-1?hub.mode=subscribe&hub.verify_token=verify-token&hub.challenge=42",
		http.NoBody,
	)
	if err != nil {
		t.Fatalf("http.NewRequestWithContext(verify challenge) error = %v", err)
	}
	verifyResp, err := http.DefaultClient.Do(verifyReq)
	if err != nil {
		t.Fatalf("http.DefaultClient.Do(verify challenge) error = %v", err)
	}
	defer func() {
		if closeErr := verifyResp.Body.Close(); closeErr != nil {
			t.Fatalf("verifyResp.Body.Close() error = %v", closeErr)
		}
	}()
	verifyBody, _ := io.ReadAll(verifyResp.Body)
	if got, want := verifyResp.StatusCode, http.StatusOK; got != want {
		t.Fatalf("verifyResp.StatusCode = %d, want %d", got, want)
	}
	if got, want := strings.TrimSpace(string(verifyBody)), "42"; got != want {
		t.Fatalf("verify challenge body = %q, want %q", got, want)
	}
}

func TestRuntimeDeliveriesCallWhatsAppGraphAPI(t *testing.T) {
	env := setProviderTestEnv(t)
	listenAddr := reserveListenAddr(t)
	mockAPI := newWhatsAppAPIServer(t)
	t.Setenv(whatsappListenAddrEnv, listenAddr)
	t.Setenv(whatsappAPIBaseEnv, mockAPI.URL())

	_, hostPeer, cleanup := newRuntimePeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 15, 13, 10, 0, 0, time.UTC)
	managed := testBridgeRuntime(now, "brg-1")
	mustHandleLifecycle(t, hostPeer, managed)

	if err := hostPeer.Call(context.Background(), "initialize", testInitializeRequest(now, managed), nil); err != nil {
		t.Fatalf("hostPeer.Call(initialize) error = %v", err)
	}
	states := waitForJSONLinesFile[stateMarker](
		t,
		env.statePath,
		func(items []stateMarker) bool { return len(items) >= 1 },
	)
	if got, want := states[len(states)-1].Status.Normalize(), bridgepkg.BridgeStatusReady; got != want {
		t.Fatalf("states[last].Status = %q, want %q", got, want)
	}

	var startAck bridgepkg.DeliveryAck
	if err := hostPeer.Call(
		context.Background(),
		"bridges/deliver",
		testDeliveryRequest("brg-1", "delivery-1", 1, bridgepkg.DeliveryEventTypeStart, false, "hello"),
		&startAck,
	); err != nil {
		t.Fatalf("hostPeer.Call(start delivery) error = %v", err)
	}
	var finalAck bridgepkg.DeliveryAck
	if err := hostPeer.Call(
		context.Background(),
		"bridges/deliver",
		testDeliveryRequest("brg-1", "delivery-1", 2, bridgepkg.DeliveryEventTypeFinal, true, "hello world"),
		&finalAck,
	); err != nil {
		t.Fatalf("hostPeer.Call(final delivery) error = %v", err)
	}

	records := waitForJSONLinesFile[deliveryMarker](
		t,
		env.deliveryPath,
		func(items []deliveryMarker) bool { return len(items) >= 2 },
	)
	if records[0].Ack == nil || records[1].Ack == nil {
		t.Fatalf("delivery markers = %#v, want recorded acks", records)
	}
	if got, want := finalAck.ReplaceRemoteMessageID, startAck.RemoteMessageID; got != want {
		t.Fatalf("finalAck.ReplaceRemoteMessageID = %q, want %q", got, want)
	}

	calls := mockAPI.Calls()
	if got, want := len(calls), 3; got != want {
		t.Fatalf("len(mockAPI calls) = %d, want %d (identity + send + send)", got, want)
	}
	if got, want := calls[0].Path, "/"+whatsappDefaultAPIVersion+"/123456789"; got != want {
		t.Fatalf("calls[0].Path = %q, want %q", got, want)
	}
	if got, want := calls[1].Body["to"], "15551234567"; got != want {
		t.Fatalf("calls[1].Body[to] = %#v, want %q", calls[1].Body["to"], want)
	}
	if got, want := calls[2].Body["type"], "text"; got != want {
		t.Fatalf("calls[2].Body[type] = %#v, want %q", calls[2].Body["type"], want)
	}
}

func TestDispatchInboundBatchAndShutdown(t *testing.T) {
	env := setProviderTestEnv(t)
	listenAddr := reserveListenAddr(t)
	mockAPI := newWhatsAppAPIServer(t)
	t.Setenv(whatsappListenAddrEnv, listenAddr)
	t.Setenv(whatsappAPIBaseEnv, mockAPI.URL())

	runtime, hostPeer, cleanup := newRuntimePeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 15, 13, 12, 0, 0, time.UTC)
	managed := testBridgeRuntime(now, "brg-1")
	mustHandleLifecycle(t, hostPeer, managed)

	var ingested []bridgepkg.InboundMessageEnvelope
	var mu sync.Mutex
	mustHandle(
		t,
		hostPeer,
		string(extensionprotocol.HostAPIMethodBridgesMessagesIngest),
		func(_ context.Context, params json.RawMessage) (any, error) {
			var envelope bridgepkg.InboundMessageEnvelope
			if err := json.Unmarshal(params, &envelope); err != nil {
				return nil, err
			}
			mu.Lock()
			ingested = append(ingested, envelope)
			mu.Unlock()
			return extensioncontract.BridgesMessagesIngestResult{
				SessionID:    "sess-1",
				RouteCreated: true,
				RoutingKey: bridgepkg.RoutingKey{
					Scope:            envelope.Scope,
					WorkspaceID:      envelope.WorkspaceID,
					BridgeInstanceID: envelope.BridgeInstanceID,
					PeerID:           envelope.PeerID,
					ThreadID:         envelope.ThreadID,
					GroupID:          envelope.GroupID,
				},
			}, nil
		},
	)

	if err := hostPeer.Call(context.Background(), "initialize", testInitializeRequest(now, managed), nil); err != nil {
		t.Fatalf("hostPeer.Call(initialize) error = %v", err)
	}
	waitForCondition(t, func() bool {
		_, err := runtime.configForInstance("brg-1")
		return err == nil
	})

	batch := bridgesdk.InboundBatch{
		Key: "batch-key",
		Items: []bridgepkg.InboundMessageEnvelope{
			{
				BridgeInstanceID:  "brg-1",
				Scope:             bridgepkg.ScopeWorkspace,
				WorkspaceID:       "ws-whatsapp",
				PlatformMessageID: "wamid.1",
				ReceivedAt:        now,
				PeerID:            "15551234567",
				Sender: bridgepkg.MessageSender{
					ID:          "15551234567",
					DisplayName: "Alice",
				},
				Content:        bridgepkg.MessageContent{Text: "first"},
				EventFamily:    bridgepkg.InboundEventFamilyMessage,
				IdempotencyKey: "whatsapp:brg-1:wamid.1",
			},
			{
				BridgeInstanceID:  "brg-1",
				Scope:             bridgepkg.ScopeWorkspace,
				WorkspaceID:       "ws-whatsapp",
				PlatformMessageID: "wamid.2",
				ReceivedAt:        now.Add(time.Second),
				PeerID:            "15551234567",
				Sender: bridgepkg.MessageSender{
					ID:          "15551234567",
					DisplayName: "Alice",
				},
				Content:        bridgepkg.MessageContent{Text: "second"},
				EventFamily:    bridgepkg.InboundEventFamilyMessage,
				IdempotencyKey: "whatsapp:brg-1:wamid.2",
			},
		},
		CreatedAt: now,
		UpdatedAt: now.Add(time.Second),
	}
	if err := runtime.dispatchInboundBatch(context.Background(), "brg-1", batch); err != nil {
		t.Fatalf("dispatchInboundBatch() error = %v", err)
	}

	waitForCondition(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(ingested) == 1
	})
	mu.Lock()
	merged := ingested[0]
	mu.Unlock()
	if got, want := merged.Content.Text, "first\nsecond"; got != want {
		t.Fatalf("merged.Content.Text = %q, want %q", got, want)
	}
	if got, want := merged.IdempotencyKey, "whatsapp:brg-1:wamid.1:batch:2"; got != want {
		t.Fatalf("merged.IdempotencyKey = %q, want %q", got, want)
	}

	if err := runtime.handleShutdown(
		context.Background(),
		nil,
		subprocess.ShutdownRequest{DeadlineMS: 50},
	); err != nil {
		t.Fatalf("handleShutdown() error = %v", err)
	}
	lines := waitForNonEmptyLines(t, env.shutdownPath)
	if len(lines) == 0 || !strings.Contains(lines[0], "pid=") {
		t.Fatalf("shutdown marker lines = %#v, want pid entry", lines)
	}
}

func TestHandleWebhookRequestValidationAndBatching(t *testing.T) {
	t.Parallel()

	provider, err := newWhatsAppProvider(io.Discard)
	if err != nil {
		t.Fatalf("newWhatsAppProvider() error = %v", err)
	}

	managed := testBridgeRuntime(time.Date(2026, 4, 15, 13, 20, 0, 0, time.UTC), "brg-1")
	cfg := resolvedInstanceConfig{
		managed:       &managed,
		instanceID:    "brg-1",
		phoneNumberID: "123456789",
		dmPolicy:      bridgepkg.BridgeDMPolicyAllowlist,
		allowUserIDs:  map[string]struct{}{"15551234567": {}},
		dedup:         bridgesdk.NewDedupCache(5*time.Minute, 32),
	}
	var batches []bridgesdk.InboundBatch
	cfg.batcher, err = bridgesdk.NewInboundBatcher(bridgesdk.InboundBatcherConfig{
		Delay: 0,
		Dispatch: func(_ context.Context, batch bridgesdk.InboundBatch) error {
			batches = append(batches, batch)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("NewInboundBatcher() error = %v", err)
	}
	defer cfg.batcher.Close()

	rec := httptest.NewRecorder()
	if err := provider.handleWebhookRequest(rec, nil, cfg, bridgesdk.WebhookRequest{Body: []byte("{")}); err == nil {
		t.Fatal("handleWebhookRequest(invalid payload) error = nil, want non-nil")
	}

	body := []byte(
		`{"object":"whatsapp_business_account","entry":[{"changes":[{"field":"statuses","value":{}},{"field":"messages","value":{"metadata":{"phone_number_id":"123456789"},"contacts":[{"profile":{"name":"Alice Example"},"wa_id":"15551234567"},{"profile":{"name":"Blocked User"},"wa_id":"16667778888"}],"messages":[{"from":"15551234567","id":"wamid.allowed","timestamp":"1775866800","type":"text","text":{"body":"hello"}},{"from":"16667778888","id":"wamid.blocked","timestamp":"1775866801","type":"text","text":{"body":"blocked"}}]}},{"field":"messages","value":{"metadata":{"phone_number_id":"999999999"},"messages":[{"from":"15551234567","id":"wamid.other","timestamp":"1775866802","type":"text","text":{"body":"wrong phone"}}]}}]}]}`,
	)
	req := bridgesdk.WebhookRequest{
		Body:       body,
		ReceivedAt: time.Date(2026, 4, 15, 13, 20, 0, 0, time.UTC),
	}
	rec = httptest.NewRecorder()
	if err := provider.handleWebhookRequest(rec, nil, cfg, req); err != nil {
		t.Fatalf("handleWebhookRequest(valid) error = %v", err)
	}
	if got, want := rec.Code, http.StatusOK; got != want {
		t.Fatalf("handleWebhookRequest(valid) status = %d, want %d", got, want)
	}
	if got, want := strings.TrimSpace(rec.Body.String()), "ok"; got != want {
		t.Fatalf("handleWebhookRequest(valid) body = %q, want %q", got, want)
	}
	if got, want := len(batches), 1; got != want {
		t.Fatalf("len(batches) = %d, want %d", got, want)
	}
	if got, want := len(batches[0].Items), 1; got != want {
		t.Fatalf("len(batches[0].Items) = %d, want %d", got, want)
	}
	if got, want := batches[0].Items[0].PeerID, "15551234567"; got != want {
		t.Fatalf("batches[0].Items[0].PeerID = %q, want %q", got, want)
	}

	rec = httptest.NewRecorder()
	if err := provider.handleWebhookRequest(rec, nil, cfg, req); err != nil {
		t.Fatalf("handleWebhookRequest(duplicate) error = %v", err)
	}
	if got, want := len(batches), 1; got != want {
		t.Fatalf("len(batches) after duplicate = %d, want %d", got, want)
	}
}

func TestRejectMisconfiguredRoutes(t *testing.T) {
	t.Parallel()

	t.Run("ShouldRejectInvalidInstanceDeliveryBeforeAPICall", func(t *testing.T) {
		t.Parallel()

		provider, err := newWhatsAppProvider(io.Discard)
		if err != nil {
			t.Fatalf("newWhatsAppProvider() error = %v", err)
		}

		badConfig := errors.New("whatsapp: provider_config.phone_number_id is required")
		apiCalled := false
		provider.apiFactory = func(resolvedInstanceConfig) whatsappAPI {
			apiCalled = true
			return &fakeWhatsAppAPI{}
		}

		provider.mu.Lock()
		provider.routes["brg-1"] = resolvedInstanceConfig{
			instanceID:  "brg-1",
			configError: badConfig,
		}
		provider.mu.Unlock()

		ack, err := provider.handleBridgesDeliver(
			context.Background(),
			nil,
			testDeliveryRequest(
				"brg-1",
				"delivery-1",
				1,
				bridgepkg.DeliveryEventTypeStart,
				false,
				"hello",
			),
		)
		if err == nil {
			t.Fatal("handleBridgesDeliver() error = nil, want non-nil")
		}
		if !errors.Is(err, errWhatsAppInstanceConfigInvalid) {
			t.Fatalf("handleBridgesDeliver() error = %v, want invalid-config sentinel", err)
		}
		if !errors.Is(err, badConfig) {
			t.Fatalf("handleBridgesDeliver() error = %v, want wrapped config error", err)
		}
		if ack != (bridgepkg.DeliveryAck{}) {
			t.Fatalf("handleBridgesDeliver() ack = %#v, want zero value", ack)
		}
		if apiCalled {
			t.Fatal("handleBridgesDeliver() called Graph API for invalid config")
		}
	})

	t.Run("ShouldFailClosedForDuplicateWebhookPaths", func(t *testing.T) {
		t.Parallel()

		provider, err := newWhatsAppProvider(io.Discard)
		if err != nil {
			t.Fatalf("newWhatsAppProvider() error = %v", err)
		}

		provider.mu.Lock()
		provider.routes["brg-1"] = resolvedInstanceConfig{
			instanceID:  "brg-1",
			webhookPath: "/whatsapp/shared",
			verifyToken: "verify-1",
		}
		provider.routes["brg-2"] = resolvedInstanceConfig{
			instanceID:  "brg-2",
			webhookPath: "/whatsapp/shared",
			verifyToken: "verify-2",
			configError: errors.New("whatsapp: webhook path \"/whatsapp/shared\" is shared"),
		}
		provider.mu.Unlock()

		req := httptest.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			"http://example.test/whatsapp/shared?hub.mode=subscribe&hub.verify_token=verify-1&hub.challenge=42",
			http.NoBody,
		)
		resp := httptest.NewRecorder()
		provider.serveWebhookHTTP(resp, req)
		if got, want := resp.Code, http.StatusNotFound; got != want {
			t.Fatalf("serveWebhookHTTP() status = %d, want %d", got, want)
		}
	})
}

func TestRetryWaitAndHealthHelpers(t *testing.T) {
	t.Parallel()

	runtime, err := newWhatsAppProvider(io.Discard)
	if err != nil {
		t.Fatalf("newWhatsAppProvider() error = %v", err)
	}

	attempts := 0
	err = runtime.retryHostCall(context.Background(), func(context.Context) error {
		attempts++
		if attempts < 3 {
			return subprocess.NewRPCError(rpcCodeNotInitialized, "Not initialized", nil)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("retryHostCall() error = %v", err)
	}
	if got, want := attempts, 3; got != want {
		t.Fatalf("attempts = %d, want %d", got, want)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := runtime.retryHostCall(ctx, func(context.Context) error {
		return subprocess.NewRPCError(rpcCodeNotInitialized, "Not initialized", nil)
	}); !errors.Is(err, context.Canceled) {
		t.Fatalf("retryHostCall(context canceled) error = %v, want %v", err, context.Canceled)
	}

	stopped, err := newWhatsAppProvider(io.Discard)
	if err != nil {
		t.Fatalf("newWhatsAppProvider(stopped) error = %v", err)
	}
	stopped.stop()
	stopErr := subprocess.NewRPCError(rpcCodeNotInitialized, "Not initialized", nil)
	if err := stopped.retryHostCall(
		context.Background(),
		func(context.Context) error { return stopErr },
	); !errors.Is(
		err,
		stopErr,
	) {
		t.Fatalf("retryHostCall(stopped) error = %v, want %v", err, stopErr)
	}

	waitProvider, err := newWhatsAppProvider(io.Discard)
	if err != nil {
		t.Fatalf("newWhatsAppProvider(wait) error = %v", err)
	}
	go func() {
		time.Sleep(20 * time.Millisecond)
		waitProvider.mu.Lock()
		waitProvider.routes["brg-1"] = resolvedInstanceConfig{
			instanceID:  "brg-1",
			webhookPath: "/whatsapp/brg-1",
		}
		waitProvider.mu.Unlock()
	}()
	cfg, err := waitProvider.waitForInstanceConfig("brg-1", 200*time.Millisecond)
	if err != nil {
		t.Fatalf("waitForInstanceConfig() error = %v", err)
	}
	if got, want := cfg.instanceID, "brg-1"; got != want {
		t.Fatalf("cfg.instanceID = %q, want %q", got, want)
	}

	waitProvider.setLastError(errors.New("boom"))
	if err := waitProvider.healthCheck(); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("healthCheck() error = %v, want boom", err)
	}
	waitProvider.clearLastError()
	if err := waitProvider.healthCheck(); err != nil {
		t.Fatalf("healthCheck(clear) error = %v", err)
	}

	if !isNotInitializedRPCError(
		subprocess.NewRPCError(rpcCodeNotInitialized, "Not initialized", nil),
	) {
		t.Fatal("isNotInitializedRPCError() = false, want true")
	}
	if isNotInitializedRPCError(errors.New("boom")) {
		t.Fatal("isNotInitializedRPCError(non-rpc) = true, want false")
	}
}

func TestExtractWhatsAppTextContentAndAttachments(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		message         whatsappInboundMessage
		wantText        string
		wantAttachments int
		validate        func(*testing.T, []bridgepkg.MessageAttachment)
	}{
		{
			name: "text",
			message: whatsappInboundMessage{
				Type: "text",
				Text: &struct {
					Body string `json:"body,omitempty"`
				}{Body: "hello"},
			},
			wantText: "hello",
		},
		{
			name: "image with caption",
			message: whatsappInboundMessage{
				Type: "image",
				Image: &struct {
					ID       string `json:"id,omitempty"`
					MIMEType string `json:"mime_type,omitempty"`
					Caption  string `json:"caption,omitempty"`
					SHA256   string `json:"sha256,omitempty"`
				}{ID: "img-1", MIMEType: "image/jpeg", Caption: "look"},
			},
			wantText:        "look",
			wantAttachments: 1,
			validate: func(t *testing.T, attachments []bridgepkg.MessageAttachment) {
				t.Helper()
				if got, want := attachments[0].ID, "img-1"; got != want {
					t.Fatalf("attachments[0].ID = %q, want %q", got, want)
				}
			},
		},
		{
			name: "document fallback",
			message: whatsappInboundMessage{
				Type: "document",
				Document: &struct {
					ID       string `json:"id,omitempty"`
					MIMEType string `json:"mime_type,omitempty"`
					Caption  string `json:"caption,omitempty"`
					Filename string `json:"filename,omitempty"`
					SHA256   string `json:"sha256,omitempty"`
				}{ID: "doc-1", MIMEType: "application/pdf", Filename: "report.pdf"},
			},
			wantText:        "[Document: report.pdf]",
			wantAttachments: 1,
		},
		{
			name: "audio",
			message: whatsappInboundMessage{
				Type: "audio",
				Audio: &struct {
					ID       string `json:"id,omitempty"`
					MIMEType string `json:"mime_type,omitempty"`
					Voice    bool   `json:"voice,omitempty"`
					SHA256   string `json:"sha256,omitempty"`
				}{ID: "aud-1", MIMEType: "audio/ogg"},
			},
			wantText:        "[Audio message]",
			wantAttachments: 1,
		},
		{
			name: "video fallback",
			message: whatsappInboundMessage{
				Type: "video",
				Video: &struct {
					ID       string `json:"id,omitempty"`
					MIMEType string `json:"mime_type,omitempty"`
					Caption  string `json:"caption,omitempty"`
					SHA256   string `json:"sha256,omitempty"`
				}{ID: "vid-1", MIMEType: "video/mp4"},
			},
			wantText:        "[Video]",
			wantAttachments: 1,
		},
		{
			name: "sticker",
			message: whatsappInboundMessage{
				Type: "sticker",
				Sticker: &struct {
					ID       string `json:"id,omitempty"`
					MIMEType string `json:"mime_type,omitempty"`
					Animated bool   `json:"animated,omitempty"`
					SHA256   string `json:"sha256,omitempty"`
				}{ID: "stk-1", MIMEType: "image/webp"},
			},
			wantText:        "[Sticker]",
			wantAttachments: 1,
		},
		{
			name: "location with fallback map url",
			message: whatsappInboundMessage{
				Type: "location",
				Location: &struct {
					Latitude  float64 `json:"latitude,omitempty"`
					Longitude float64 `json:"longitude,omitempty"`
					Name      string  `json:"name,omitempty"`
					Address   string  `json:"address,omitempty"`
					URL       string  `json:"url,omitempty"`
				}{Latitude: -23.5, Longitude: -46.6, Name: "HQ", Address: "Rua 1"},
			},
			wantText:        "[Location: HQ - Rua 1]",
			wantAttachments: 1,
			validate: func(t *testing.T, attachments []bridgepkg.MessageAttachment) {
				t.Helper()
				if got := attachments[0].URL; !strings.Contains(got, "google.com/maps") {
					t.Fatalf("attachments[0].URL = %q, want google maps fallback", got)
				}
			},
		},
		{
			name: "unsupported",
			message: whatsappInboundMessage{
				Type: "contacts",
			},
			wantText: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got, want := extractWhatsAppTextContent(tc.message), tc.wantText; got != want {
				t.Fatalf("extractWhatsAppTextContent() = %q, want %q", got, want)
			}
			attachments := normalizeWhatsAppAttachments(tc.message)
			if got, want := len(attachments), tc.wantAttachments; got != want {
				t.Fatalf("len(normalizeWhatsAppAttachments()) = %d, want %d", got, want)
			}
			if tc.validate != nil {
				tc.validate(t, attachments)
			}
		})
	}
}

func TestWhatsAppGraphClientMethods(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Header.Get("Authorization"), "Bearer access-token"; got != want {
			t.Fatalf("authorization header = %q, want %q", got, want)
		}

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v99.0/123456789":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "123456789"})
		case r.Method == http.MethodPost && r.URL.Path == "/v99.0/123456789/messages":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("json.Decode(payload) error = %v", err)
			}
			if got, want := payload["to"], "15551234567"; got != want {
				t.Fatalf("payload[to] = %#v, want %q", got, want)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"messages": []map[string]any{{"id": "wamid.graph"}},
			})
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := &whatsappGraphClient{
		baseURL:     server.URL,
		apiVersion:  "v99.0",
		accessToken: "access-token",
		httpClient:  server.Client(),
	}

	phone, err := client.GetPhoneNumber(context.Background(), "123456789")
	if err != nil {
		t.Fatalf("GetPhoneNumber() error = %v", err)
	}
	if got, want := phone.ID, "123456789"; got != want {
		t.Fatalf("phone.ID = %q, want %q", got, want)
	}

	resp, err := client.SendTextMessage(
		context.Background(),
		"123456789",
		whatsappSendMessageRequest{
			MessagingProduct: "whatsapp",
			RecipientType:    "individual",
			To:               "15551234567",
			Type:             "text",
			Text: struct {
				Body       string `json:"body"`
				PreviewURL bool   `json:"preview_url"`
			}{
				Body:       "hello",
				PreviewURL: false,
			},
		},
	)
	if err != nil {
		t.Fatalf("SendTextMessage() error = %v", err)
	}
	if got, want := resp.Messages[0].ID, "wamid.graph"; got != want {
		t.Fatalf("resp.Messages[0].ID = %q, want %q", got, want)
	}
}

func TestMarkerHelpers(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	linePath := filepath.Join(root, "markers", "lines.log")
	jsonPath := filepath.Join(root, "markers", "value.json")
	crashPath := filepath.Join(root, "markers", "crash-once.json")

	if err := appendMarkerLine(linePath, "  first line  "); err != nil {
		t.Fatalf("appendMarkerLine() error = %v", err)
	}
	lines := waitForNonEmptyLines(t, linePath)
	if got, want := lines[0], "first line"; got != want {
		t.Fatalf("lines[0] = %q, want %q", got, want)
	}

	if err := writeJSONFile(jsonPath, map[string]any{"ok": true}); err != nil {
		t.Fatalf("writeJSONFile() error = %v", err)
	}
	payload, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("os.ReadFile(jsonPath) error = %v", err)
	}
	if !strings.Contains(string(payload), `"ok":true`) {
		t.Fatalf("json payload = %q, want ok=true", string(payload))
	}

	if got := shouldCrashOnce(crashPath); !got {
		t.Fatal("shouldCrashOnce(missing) = false, want true")
	}
	if err := writeJSONFile(crashPath, map[string]any{"crashed": true}); err != nil {
		t.Fatalf("writeJSONFile(crashPath) error = %v", err)
	}
	if got := shouldCrashOnce(crashPath); got {
		t.Fatal("shouldCrashOnce(existing) = true, want false")
	}

	var stderr strings.Builder
	reportSideEffectError(&stderr, " test action ", errors.New("boom"))
	if got := stderr.String(); !strings.Contains(got, "whatsapp: test action: boom") {
		t.Fatalf("reportSideEffectError() wrote %q, want action and error", got)
	}
}

func TestRunHelpers(t *testing.T) {
	t.Parallel()

	if err := run([]string{"bad"}, strings.NewReader(""), io.Discard, io.Discard); err == nil {
		t.Fatal("run(unsupported) error = nil, want non-nil")
	}
}

type fakeWhatsAppAPI struct {
	nextMessageID int
	requests      []whatsappSendMessageRequest
}

func (f *fakeWhatsAppAPI) GetPhoneNumber(context.Context, string) (*whatsappPhoneNumber, error) {
	return &whatsappPhoneNumber{ID: "123456789"}, nil
}

func (f *fakeWhatsAppAPI) SendTextMessage(
	_ context.Context,
	_ string,
	req whatsappSendMessageRequest,
) (*whatsappSendMessageResponse, error) {
	f.requests = append(f.requests, req)
	messageID := fmt.Sprintf("wamid.%d", f.nextMessageID)
	f.nextMessageID++
	return &whatsappSendMessageResponse{
		Messages: []struct {
			ID string `json:"id,omitempty"`
		}{
			{ID: messageID},
		},
	}, nil
}

type fakeWhatsAppAPIError struct {
	err error
}

func (f fakeWhatsAppAPIError) GetPhoneNumber(
	context.Context,
	string,
) (*whatsappPhoneNumber, error) {
	return nil, f.err
}

func (f fakeWhatsAppAPIError) SendTextMessage(
	context.Context,
	string,
	whatsappSendMessageRequest,
) (*whatsappSendMessageResponse, error) {
	return nil, f.err
}

type whatsappAPIServer struct {
	server        *httptest.Server
	mu            sync.Mutex
	calls         []whatsappAPICall
	nextMessageID int
}

type whatsappAPICall struct {
	Path string
	Body map[string]any
}

func newWhatsAppAPIServer(t *testing.T) *whatsappAPIServer {
	t.Helper()

	srv := &whatsappAPIServer{nextMessageID: 700}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{}
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}

		srv.mu.Lock()
		srv.calls = append(srv.calls, whatsappAPICall{Path: r.URL.Path, Body: body})
		srv.mu.Unlock()

		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/123456789"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "123456789"})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/123456789/messages"):
			srv.mu.Lock()
			messageID := fmt.Sprintf("wamid.%d", srv.nextMessageID)
			srv.nextMessageID++
			srv.mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{
				"messages": []map[string]any{{"id": messageID}},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"message": "unknown method",
					"code":    http.StatusNotFound,
				},
			})
		}
	}))
	srv.server = server
	t.Cleanup(server.Close)
	return srv
}

func (s *whatsappAPIServer) URL() string {
	return s.server.URL
}

func (s *whatsappAPIServer) Calls() []whatsappAPICall {
	s.mu.Lock()
	defer s.mu.Unlock()
	cloned := make([]whatsappAPICall, len(s.calls))
	copy(cloned, s.calls)
	return cloned
}

func newRuntimePeerPair(t *testing.T) (*whatsappProvider, *bridgesdk.Peer, func()) {
	t.Helper()

	hostConn, runtimeConn := net.Pipe()
	runtime, err := newWhatsAppProvider(io.Discard)
	if err != nil {
		t.Fatalf("newWhatsAppProvider() error = %v", err)
	}

	hostPeer := bridgesdk.NewPeer(hostConn, hostConn)
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 2)
	go func() { errCh <- runtime.serve(runtimeConn, runtimeConn) }()
	go func() { errCh <- hostPeer.Serve(ctx) }()

	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			cancel()
			runtime.stop()
			runtime.mu.RLock()
			server := runtime.server
			runtime.mu.RUnlock()
			if server != nil {
				shutdownCtx, shutdownCancel := context.WithTimeout(
					context.Background(),
					2*time.Second,
				)
				_ = server.Shutdown(shutdownCtx)
				shutdownCancel()
			}
			_ = hostConn.Close()
			_ = runtimeConn.Close()
			for range 2 {
				err := <-errCh
				if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, net.ErrClosed) {
					continue
				}
				if strings.Contains(err.Error(), "closed") {
					continue
				}
				t.Fatalf("runtime peer serve error = %v", err)
			}
			runtime.wg.Wait()
		})
	}

	return runtime, hostPeer, cleanup
}

func mustHandle(t *testing.T, peer *bridgesdk.Peer, method string, handler bridgesdk.RPCHandler) {
	t.Helper()
	if err := peer.Handle(method, handler); err != nil {
		t.Fatalf("peer.Handle(%q) error = %v", method, err)
	}
}

func mustHandleLifecycle(
	t *testing.T,
	peer *bridgesdk.Peer,
	managed ...subprocess.InitializeBridgeManagedInstance,
) {
	t.Helper()

	mustHandle(
		t,
		peer,
		string(extensionprotocol.HostAPIMethodBridgesInstancesList),
		func(context.Context, json.RawMessage) (any, error) {
			instances := make([]bridgepkg.BridgeInstance, 0, len(managed))
			for _, item := range managed {
				instances = append(instances, item.Instance)
			}
			return instances, nil
		},
	)
	mustHandle(
		t,
		peer,
		string(extensionprotocol.HostAPIMethodBridgesInstancesGet),
		func(_ context.Context, params json.RawMessage) (any, error) {
			var payload extensioncontract.BridgeInstanceTargetParams
			if err := json.Unmarshal(params, &payload); err != nil {
				return nil, err
			}
			for _, item := range managed {
				if item.Instance.ID == payload.BridgeInstanceID {
					return item.Instance, nil
				}
			}
			return nil, errors.New("unexpected instance")
		},
	)
	mustHandle(
		t,
		peer,
		string(extensionprotocol.HostAPIMethodBridgesInstancesReportState),
		func(_ context.Context, params json.RawMessage) (any, error) {
			var payload extensioncontract.BridgesInstancesReportStateParams
			if err := json.Unmarshal(params, &payload); err != nil {
				return nil, err
			}
			for _, item := range managed {
				if item.Instance.ID == payload.BridgeInstanceID {
					instance := item.Instance
					instance.Status = payload.Status
					instance.Degradation = payload.Degradation
					return instance, nil
				}
			}
			return nil, errors.New("unexpected state instance")
		},
	)
}

func testBridgeRuntime(
	now time.Time,
	instanceID string,
) subprocess.InitializeBridgeManagedInstance {
	return subprocess.InitializeBridgeManagedInstance{
		Instance: bridgepkg.BridgeInstance{
			ID:            instanceID,
			Scope:         bridgepkg.ScopeWorkspace,
			WorkspaceID:   "ws-whatsapp",
			Platform:      "whatsapp",
			ExtensionName: "whatsapp",
			DisplayName:   "WhatsApp",
			Enabled:       true,
			Status:        bridgepkg.BridgeStatusReady,
			RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
			ProviderConfig: []byte(`{
				"phone_number_id":"123456789"
			}`),
			CreatedAt: now,
			UpdatedAt: now,
		},
		BoundSecrets: []subprocess.InitializeBridgeBoundSecret{
			{BindingName: "access_token", Kind: "token", Value: "access-token"},
			{BindingName: "app_secret", Kind: "token", Value: "app-secret"},
			{BindingName: "verify_token", Kind: "token", Value: "verify-token"},
		},
	}
}

func testInitializeRequest(
	_ time.Time,
	managed ...subprocess.InitializeBridgeManagedInstance,
) subprocess.InitializeRequest {
	return subprocess.InitializeRequest{
		ProtocolVersion:          "1",
		SupportedProtocolVersion: []string{"1"},
		AGHVersion:               "0.5.0",
		SessionNonce:             "nonce-test",
		Extension: subprocess.InitializeExtension{
			Name:       "whatsapp",
			Version:    "0.1.0",
			SourceTier: "user",
		},
		Capabilities: subprocess.InitializeCapabilities{
			Provides: []string{"bridge.adapter"},
			GrantedActions: []extensionprotocol.HostAPIMethod{
				extensionprotocol.HostAPIMethodBridgesInstancesList,
				extensionprotocol.HostAPIMethodBridgesInstancesGet,
				extensionprotocol.HostAPIMethodBridgesInstancesReportState,
				extensionprotocol.HostAPIMethodBridgesMessagesIngest,
			},
			GrantedSecurity: []string{"bridge.read", "bridge.write"},
		},
		Methods: subprocess.InitializeMethods{
			ExtensionServices: []string{"bridges/deliver", "health_check", "shutdown"},
		},
		Runtime: subprocess.InitializeRuntime{
			HealthCheckIntervalMS: 30_000,
			HealthCheckTimeoutMS:  5_000,
			ShutdownTimeoutMS:     5_000,
			DefaultHookTimeoutMS:  5_000,
			Bridge: &subprocess.InitializeBridgeRuntime{
				RuntimeVersion:   subprocess.InitializeBridgeRuntimeVersion1,
				Provider:         "whatsapp",
				Platform:         "whatsapp",
				ManagedInstances: managed,
			},
		},
	}
}

func testDeliveryRequest(
	instanceID string,
	deliveryID string,
	seq int64,
	eventType string,
	final bool,
	text string,
) bridgepkg.DeliveryRequest {
	return bridgepkg.DeliveryRequest{
		Event: bridgepkg.DeliveryEvent{
			DeliveryID:       deliveryID,
			BridgeInstanceID: instanceID,
			RoutingKey: bridgepkg.RoutingKey{
				Scope:            bridgepkg.ScopeWorkspace,
				WorkspaceID:      "ws-whatsapp",
				BridgeInstanceID: instanceID,
				PeerID:           "15551234567",
			},
			DeliveryTarget: bridgepkg.DeliveryTarget{
				BridgeInstanceID: instanceID,
				PeerID:           "15551234567",
				Mode:             bridgepkg.DeliveryModeReply,
			},
			Seq:       seq,
			EventType: eventType,
			Content:   bridgepkg.MessageContent{Text: text},
			Final:     final,
		},
	}
}

func testDeleteRequest(
	instanceID string,
	deliveryID string,
	seq int64,
	remoteMessageID string,
) bridgepkg.DeliveryRequest {
	req := testDeliveryRequest(
		instanceID,
		deliveryID,
		seq,
		bridgepkg.DeliveryEventTypeDelete,
		true,
		"",
	)
	req.Event.Operation = bridgepkg.DeliveryOperationDelete
	req.Event.Reference = &bridgepkg.DeliveryMessageReference{RemoteMessageID: remoteMessageID}
	return req
}

func whatsappWebhookPayloadForPhone(phoneNumberID string, text string) string {
	return fmt.Sprintf(
		`{"object":"whatsapp_business_account","entry":[{"id":"waba-1","changes":[{"field":"messages","value":{"messaging_product":"whatsapp","metadata":{"display_phone_number":"+15551234567","phone_number_id":%q},"contacts":[{"profile":{"name":"Alice Example"},"wa_id":"15551234567"}],"messages":[{"from":"15551234567","id":"wamid.abc123","timestamp":"1775866800","type":"text","text":{"body":%q}}]}}]}]}`,
		phoneNumberID,
		text,
	)
}

func signWhatsAppBody(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func strconvTime(ts time.Time) string {
	return strconv.FormatInt(ts.Unix(), 10)
}

func setProviderTestEnv(t *testing.T) markerEnv {
	t.Helper()

	root := filepath.Join(t.TempDir(), "markers")
	env := markerEnv{
		handshakePath: filepath.Join(root, "handshake.json"),
		ownershipPath: filepath.Join(root, "ownership.json"),
		statePath:     filepath.Join(root, "state.jsonl"),
		deliveryPath:  filepath.Join(root, "delivery.jsonl"),
		ingestPath:    filepath.Join(root, "ingest.jsonl"),
		startsPath:    filepath.Join(root, "starts.log"),
		shutdownPath:  filepath.Join(root, "shutdown.log"),
		crashOncePath: filepath.Join(root, "crash-once.json"),
	}

	t.Setenv(adapterHandshakeEnv, env.handshakePath)
	t.Setenv(adapterOwnershipEnv, env.ownershipPath)
	t.Setenv(adapterStateEnv, env.statePath)
	t.Setenv(adapterDeliveryEnv, env.deliveryPath)
	t.Setenv(adapterIngestEnv, env.ingestPath)
	t.Setenv(adapterStartsEnv, env.startsPath)
	t.Setenv(adapterShutdownEnv, env.shutdownPath)
	t.Setenv(adapterCrashOnceEnv, "")

	return env
}

func reserveListenAddr(t *testing.T) string {
	t.Helper()

	var listenConfig net.ListenConfig
	ln, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listenConfig.Listen() error = %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("ln.Close() error = %v", err)
	}
	return addr
}

func waitForJSONFile[T any](t *testing.T, path string) T {
	t.Helper()

	var item T
	waitForCondition(t, func() bool {
		payload, err := os.ReadFile(path)
		if err != nil {
			return false
		}
		return json.Unmarshal(payload, &item) == nil
	})
	return item
}

func waitForJSONLinesFile[T any](t *testing.T, path string, predicate func([]T) bool) []T {
	t.Helper()

	var items []T
	waitForCondition(t, func() bool {
		payload, err := os.ReadFile(path)
		if err != nil {
			return false
		}
		lines := nonEmptyLines(string(payload))
		decoded := make([]T, 0, len(lines))
		for _, line := range lines {
			var item T
			if err := json.Unmarshal([]byte(line), &item); err != nil {
				return false
			}
			decoded = append(decoded, item)
		}
		items = decoded
		return predicate(items)
	})
	return items
}

func waitForNonEmptyLines(t *testing.T, path string) []string {
	t.Helper()

	var lines []string
	waitForCondition(t, func() bool {
		payload, err := os.ReadFile(path)
		if err != nil {
			return false
		}
		lines = nonEmptyLines(string(payload))
		return len(lines) > 0
	})
	return lines
}

func waitForCondition(t *testing.T, fn func() bool) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition did not succeed before timeout")
}

func nonEmptyLines(input string) []string {
	lines := strings.Split(input, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		filtered = append(filtered, trimmed)
	}
	return filtered
}
