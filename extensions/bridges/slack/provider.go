package main

import (
	"bytes"
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
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/bridgesdk"
	extensioncontract "github.com/pedronauck/agh/internal/extension/contract"
	"github.com/pedronauck/agh/internal/subprocess"
)

const (
	slackListenAddrEnv = "AGH_BRIDGE_SLACK_LISTEN_ADDR"
	slackAPIBaseEnv    = "AGH_BRIDGE_SLACK_API_BASE_URL"

	slackDefaultAPIBaseURL        = "https://slack.com/api"
	slackSignatureVersion         = "v0"
	slackWebhookReadHeaderTimeout = 10 * time.Second
	slackWebhookIdleTimeout       = 2 * time.Minute

	rpcCodeNotInitialized = -32003
)

type slackProvider struct {
	sdk     *bridgesdk.Runtime
	stderr  io.Writer
	env     markerEnv
	now     func() time.Time
	session *bridgesdk.Session

	mu             sync.RWMutex
	lastError      string
	server         *http.Server
	serverListener net.Listener
	serverAddr     string
	listenAddr     string
	routes         map[string]resolvedInstanceConfig
	deliveries     map[string]deliveryState
	reportedStatus map[string]bridgepkg.BridgeStatus
	apiFactory     func(resolvedInstanceConfig) slackAPI

	stopCh   chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

type deliveryState struct {
	LastSeq                int64
	RemoteMessageID        string
	ReplaceRemoteMessageID string
}

type slackProviderConfig struct {
	APIBaseURL string `json:"api_base_url,omitempty"`
	Webhook    struct {
		ListenAddr string `json:"listen_addr,omitempty"`
		Path       string `json:"path,omitempty"`
	} `json:"webhook"`
	Batching struct {
		DelayMS        int `json:"delay_ms,omitempty"`
		SplitDelayMS   int `json:"split_delay_ms,omitempty"`
		SplitThreshold int `json:"split_threshold,omitempty"`
	} `json:"batching"`
	DM struct {
		AllowUserIDs    []string `json:"allow_user_ids,omitempty"`
		AllowUsernames  []string `json:"allow_usernames,omitempty"`
		PairedUserIDs   []string `json:"paired_user_ids,omitempty"`
		PairedUsernames []string `json:"paired_usernames,omitempty"`
	} `json:"dm"`
}

type resolvedInstanceConfig struct {
	managed            subprocess.InitializeBridgeManagedInstance
	instanceID         string
	listenAddr         string
	webhookPath        string
	apiBaseURL         string
	botToken           string
	signingSecret      string
	dmPolicy           bridgepkg.BridgeDMPolicy
	allowUserIDs       map[string]struct{}
	allowUsernames     map[string]struct{}
	pairedUserIDs      map[string]struct{}
	pairedUsernames    map[string]struct{}
	dedup              *bridgesdk.DedupCache
	rateLimiter        *bridgesdk.FixedWindowRateLimiter
	inFlightLimiter    *bridgesdk.InFlightLimiter
	batcher            *bridgesdk.InboundBatcher
	configError        error
	initialDegradation *bridgepkg.BridgeDegradation
	initialStatus      bridgepkg.BridgeStatus
}

type slackWebhookEnvelope struct {
	Challenge string          `json:"challenge,omitempty"`
	Event     json.RawMessage `json:"event,omitempty"`
	EventID   string          `json:"event_id,omitempty"`
	EventTime int64           `json:"event_time,omitempty"`
	TeamID    string          `json:"team_id,omitempty"`
	Type      string          `json:"type"`
}

type slackEventTypePayload struct {
	Type string `json:"type"`
}

type slackMessageEvent struct {
	BotID       string      `json:"bot_id,omitempty"`
	Channel     string      `json:"channel,omitempty"`
	ChannelType string      `json:"channel_type,omitempty"`
	Edited      *slackEdit  `json:"edited,omitempty"`
	Files       []slackFile `json:"files,omitempty"`
	Subtype     string      `json:"subtype,omitempty"`
	Team        string      `json:"team,omitempty"`
	TeamID      string      `json:"team_id,omitempty"`
	Text        string      `json:"text,omitempty"`
	ThreadTS    string      `json:"thread_ts,omitempty"`
	TS          string      `json:"ts,omitempty"`
	Type        string      `json:"type"`
	User        string      `json:"user,omitempty"`
	Username    string      `json:"username,omitempty"`
}

type slackEdit struct {
	TS string `json:"ts,omitempty"`
}

type slackFile struct {
	ID         string `json:"id,omitempty"`
	MIMEType   string `json:"mimetype,omitempty"`
	Name       string `json:"name,omitempty"`
	URLPrivate string `json:"url_private,omitempty"`
}

type slackReactionEvent struct {
	EventTS  string            `json:"event_ts,omitempty"`
	Item     slackReactionItem `json:"item"`
	ItemUser string            `json:"item_user,omitempty"`
	Reaction string            `json:"reaction,omitempty"`
	Type     string            `json:"type"`
	User     string            `json:"user,omitempty"`
}

type slackReactionItem struct {
	Channel string `json:"channel,omitempty"`
	TS      string `json:"ts,omitempty"`
	Type    string `json:"type,omitempty"`
}

type slackBlockActionsPayload struct {
	Actions []slackBlockAction `json:"actions"`
	Channel struct {
		ID string `json:"id,omitempty"`
	} `json:"channel"`
	Container struct {
		Type        string `json:"type,omitempty"`
		MessageTS   string `json:"message_ts,omitempty"`
		ChannelID   string `json:"channel_id,omitempty"`
		IsEphemeral bool   `json:"is_ephemeral,omitempty"`
		ThreadTS    string `json:"thread_ts,omitempty"`
	} `json:"container"`
	Message struct {
		TS       string `json:"ts,omitempty"`
		ThreadTS string `json:"thread_ts,omitempty"`
	} `json:"message"`
	ResponseURL string `json:"response_url,omitempty"`
	TriggerID   string `json:"trigger_id,omitempty"`
	Type        string `json:"type"`
	User        struct {
		ID       string `json:"id,omitempty"`
		Name     string `json:"name,omitempty"`
		Username string `json:"username,omitempty"`
	} `json:"user"`
}

type slackBlockAction struct {
	ActionID       string `json:"action_id,omitempty"`
	ActionTS       string `json:"action_ts,omitempty"`
	BlockID        string `json:"block_id,omitempty"`
	Type           string `json:"type,omitempty"`
	Value          string `json:"value,omitempty"`
	SelectedOption *struct {
		Value string `json:"value,omitempty"`
	} `json:"selected_option,omitempty"`
}

type slackMappedInbound struct {
	Envelope bridgepkg.InboundMessageEnvelope
	Direct   bool
	User     slackUserIdentity
}

type slackUserIdentity struct {
	ID          string
	Username    string
	DisplayName string
}

type slackAPI interface {
	AuthTest(context.Context) (*slackAuthIdentity, error)
	PostMessage(context.Context, slackPostMessageRequest) (*slackPostedMessage, error)
	UpdateMessage(context.Context, slackUpdateMessageRequest) error
	DeleteMessage(context.Context, slackDeleteMessageRequest) error
}

type slackDeliveryReconciler interface {
	FindDeliveryMessage(context.Context, slackFindDeliveryMessageRequest) (*slackPostedMessage, error)
}

type slackAuthIdentity struct {
	BotID  string `json:"bot_id,omitempty"`
	UserID string `json:"user_id,omitempty"`
}

type slackPostedMessage struct {
	TS string `json:"ts,omitempty"`
}

type slackPostMessageRequest struct {
	Channel  string                `json:"channel"`
	ThreadTS string                `json:"thread_ts,omitempty"`
	Text     string                `json:"text"`
	Metadata *slackMessageMetadata `json:"metadata,omitempty"`
}

type slackUpdateMessageRequest struct {
	Channel string `json:"channel"`
	TS      string `json:"ts"`
	Text    string `json:"text"`
}

type slackDeleteMessageRequest struct {
	Channel string `json:"channel"`
	TS      string `json:"ts"`
}

type slackFindDeliveryMessageRequest struct {
	Channel          string
	ThreadTS         string
	DeliveryID       string
	BridgeInstanceID string
}

func (r slackFindDeliveryMessageRequest) Validate() error {
	if strings.TrimSpace(r.Channel) == "" {
		return errors.New("slack: delivery reconciliation requires channel")
	}
	if strings.TrimSpace(r.DeliveryID) == "" {
		return errors.New("slack: delivery reconciliation requires delivery id")
	}
	if strings.TrimSpace(r.BridgeInstanceID) == "" {
		return errors.New("slack: delivery reconciliation requires bridge instance id")
	}
	return nil
}

type slackMessageMetadata struct {
	EventType    string                      `json:"event_type"`
	EventPayload slackMessageMetadataPayload `json:"event_payload"`
}

type slackMessageMetadataPayload struct {
	BridgeInstanceID string `json:"bridge_instance_id"`
	DeliveryID       string `json:"delivery_id"`
}

type slackConversationMessagesRequest struct {
	Channel   string `json:"channel"`
	Cursor    string `json:"cursor,omitempty"`
	Inclusive bool   `json:"inclusive,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	TS        string `json:"ts,omitempty"`
}

type slackConversationMessagesResponse struct {
	HasMore          bool                       `json:"has_more,omitempty"`
	Messages         []slackConversationMessage `json:"messages,omitempty"`
	ResponseMetadata *slackResponseMetadata     `json:"response_metadata,omitempty"`
}

type slackConversationMessage struct {
	TS       string                `json:"ts,omitempty"`
	Metadata *slackMessageMetadata `json:"metadata,omitempty"`
}

type slackResponseMetadata struct {
	NextCursor string `json:"next_cursor,omitempty"`
}

type slackAPIEnvelope struct {
	BotID  string `json:"bot_id,omitempty"`
	Error  string `json:"error,omitempty"`
	OK     bool   `json:"ok"`
	TS     string `json:"ts,omitempty"`
	UserID string `json:"user_id,omitempty"`
}

type slackBotClient struct {
	baseURL    string
	botToken   string
	httpClient *http.Client
}

func newSlackProvider(stderr io.Writer) (*slackProvider, error) {
	if stderr == nil {
		stderr = io.Discard
	}

	provider := &slackProvider{
		stderr:         stderr,
		env:            markerEnvFromProcess(),
		now:            func() time.Time { return time.Now().UTC() },
		routes:         make(map[string]resolvedInstanceConfig),
		deliveries:     make(map[string]deliveryState),
		reportedStatus: make(map[string]bridgepkg.BridgeStatus),
		stopCh:         make(chan struct{}),
	}
	provider.apiFactory = func(cfg resolvedInstanceConfig) slackAPI {
		return &slackBotClient{
			baseURL:  cfg.apiBaseURL,
			botToken: cfg.botToken,
			httpClient: &http.Client{
				Timeout: 10 * time.Second,
			},
		}
	}

	sdkRuntime, err := bridgesdk.NewRuntime(bridgesdk.RuntimeConfig{
		ExtensionInfo: subprocess.InitializeExtensionInfo{
			Name:    "slack",
			Version: "0.1.0",
			SDKName: "bridgesdk",
		},
		Initialize:  provider.handleInitialize,
		Deliver:     provider.handleBridgesDeliver,
		HealthCheck: func(context.Context, *bridgesdk.Session) error { return provider.healthCheck() },
		Shutdown:    provider.handleShutdown,
		Now:         func() time.Time { return provider.now() },
	})
	if err != nil {
		return nil, err
	}
	provider.sdk = sdkRuntime
	return provider, nil
}

func (p *slackProvider) serve(stdin io.Reader, stdout io.Writer) error {
	p.reportSideEffectError(
		"write start marker",
		appendMarkerLine(p.env.startsPath, fmt.Sprintf("pid=%d", os.Getpid())),
	)
	return p.sdk.Serve(context.Background(), stdin, stdout)
}

func (p *slackProvider) handleInitialize(_ context.Context, session *bridgesdk.Session) error {
	p.mu.Lock()
	p.session = session
	p.mu.Unlock()

	marker := initializeMarker{
		Request:  session.InitializeRequest(),
		Response: session.InitializeResponse(),
	}
	p.reportSideEffectError("write initialize marker", writeJSONFile(p.env.handshakePath, marker))
	p.clearLastError()

	p.wg.Go(func() {
		p.afterInitialize(session)
	})

	return nil
}

func (p *slackProvider) afterInitialize(session *bridgesdk.Session) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	listed, err := p.syncOwnedInstances(ctx, session)
	ownershipErr := err
	fetched := make([]bridgepkg.BridgeInstance, 0, len(listed))
	if ownershipErr == nil {
		for _, managed := range listed {
			instance, getErr := p.getOwnedInstance(ctx, session, managed.Instance.ID)
			if getErr != nil {
				ownershipErr = getErr
				break
			}
			fetched = append(fetched, *instance)
		}
	}
	if len(listed) == 0 {
		listed = session.Cache().List()
	}

	ownership := ownershipMarker{
		Listed:  managedInstancesToInstances(listed),
		Fetched: fetched,
	}
	if ownershipErr != nil {
		ownership.Error = ownershipErr.Error()
	}
	p.reportSideEffectError("write ownership marker", writeJSONFile(p.env.ownershipPath, ownership))

	configs := p.reconcileInstanceConfigs(ctx, session, listed)
	for idx := range configs {
		cfg := configs[idx]
		status := cfg.initialStatus
		degradation := cfg.initialDegradation
		if status == "" {
			status = bridgepkg.BridgeStatusReady
		}
		if err := p.reportState(
			ctx,
			session,
			cfg.instanceID,
			status,
			degradation,
		); err != nil &&
			ownershipErr == nil {
			ownershipErr = err
		}
	}

	if ownershipErr != nil {
		p.setLastError(ownershipErr)
	} else {
		p.clearLastError()
	}
}

func (p *slackProvider) handleBridgesDeliver(
	ctx context.Context,
	session *bridgesdk.Session,
	request bridgepkg.DeliveryRequest,
) (bridgepkg.DeliveryAck, error) {
	marker := deliveryMarker{
		PID:     os.Getpid(),
		Request: request,
	}

	cfg, err := p.waitForInstanceConfig(strings.TrimSpace(request.Event.BridgeInstanceID), 500*time.Millisecond)
	if err != nil {
		marker.Error = err.Error()
		p.reportSideEffectError("write failed delivery marker", appendJSONLine(p.env.deliveryPath, marker))
		p.setLastError(err)
		return bridgepkg.DeliveryAck{}, err
	}

	if shouldCrashOnce(p.env.crashOncePath) {
		p.reportSideEffectError("write pre-crash delivery marker", appendJSONLine(p.env.deliveryPath, marker))
		p.reportSideEffectError("write crash marker", writeJSONFile(p.env.crashOncePath, map[string]any{
			"crashed":            true,
			"pid":                os.Getpid(),
			"delivery_id":        strings.TrimSpace(request.Event.DeliveryID),
			"bridge_instance_id": cfg.instanceID,
		}))
		os.Exit(23)
	}

	api := p.apiFactory(cfg)
	ack, state, err := executeDelivery(ctx, api, request, p.deliveryState(cfg.instanceID, request.Event.DeliveryID))
	if err != nil {
		marker.Error = err.Error()
		p.reportSideEffectError("write failed delivery marker", appendJSONLine(p.env.deliveryPath, marker))
		classified := bridgesdk.ClassifyError(err)
		_, _, reportErr := session.ReportClassifiedError(ctx, cfg.instanceID, classified)
		if reportErr != nil {
			p.setLastError(reportErr)
		} else {
			p.setLastError(err)
		}
		return bridgepkg.DeliveryAck{}, err
	}

	p.storeDeliveryState(cfg.instanceID, request.Event.DeliveryID, state)
	p.reportReadyIfNeeded(ctx, session, cfg.instanceID)

	marker.Ack = &ack
	p.reportSideEffectError("write delivery marker", appendJSONLine(p.env.deliveryPath, marker))
	p.clearLastError()
	return ack, nil
}

func (p *slackProvider) healthCheck() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if strings.TrimSpace(p.lastError) == "" {
		return nil
	}
	return errors.New(strings.TrimSpace(p.lastError))
}

func (p *slackProvider) handleShutdown(
	_ context.Context,
	_ *bridgesdk.Session,
	request subprocess.ShutdownRequest,
) error {
	p.stop()

	shutdownCtx := context.Background()
	if request.DeadlineMS > 0 {
		var cancel context.CancelFunc
		shutdownCtx, cancel = context.WithTimeout(
			context.Background(),
			time.Duration(request.DeadlineMS)*time.Millisecond,
		)
		defer cancel()
	}

	p.mu.Lock()
	server := p.server
	listener := p.serverListener
	p.server = nil
	p.serverListener = nil
	p.mu.Unlock()
	if listener != nil {
		_ = listener.Close()
	}
	if server != nil {
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
		}
		_ = server.Close()
	}

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-shutdownCtx.Done():
	}

	p.reportSideEffectError(
		"write shutdown marker",
		appendMarkerLine(p.env.shutdownPath, fmt.Sprintf("pid=%d", os.Getpid())),
	)
	return nil
}

func (p *slackProvider) stop() {
	p.stopOnce.Do(func() {
		close(p.stopCh)
		batchersToClose := make(map[*bridgesdk.InboundBatcher]struct{})
		p.mu.Lock()
		for id, cfg := range p.routes {
			if cfg.batcher != nil {
				batchersToClose[cfg.batcher] = struct{}{}
				cfg.batcher = nil
				p.routes[id] = cfg
			}
		}
		p.mu.Unlock()
		closeInboundBatchers(batchersToClose)
	})
}

func (p *slackProvider) syncOwnedInstances(
	ctx context.Context,
	session *bridgesdk.Session,
) ([]subprocess.InitializeBridgeManagedInstance, error) {
	var result []subprocess.InitializeBridgeManagedInstance
	err := p.retryHostCall(ctx, func(callCtx context.Context) error {
		items, callErr := session.SyncInstances(callCtx)
		if callErr == nil {
			result = items
		}
		return callErr
	})
	return result, err
}

func (p *slackProvider) getOwnedInstance(
	ctx context.Context,
	session *bridgesdk.Session,
	bridgeInstanceID string,
) (*bridgepkg.BridgeInstance, error) {
	var result *bridgepkg.BridgeInstance
	err := p.retryHostCall(ctx, func(callCtx context.Context) error {
		instance, callErr := session.HostAPI().GetBridgeInstance(callCtx, bridgeInstanceID)
		if callErr == nil {
			result = instance
		}
		return callErr
	})
	return result, err
}

func (p *slackProvider) reportState(
	ctx context.Context,
	session *bridgesdk.Session,
	bridgeInstanceID string,
	status bridgepkg.BridgeStatus,
	degradation *bridgepkg.BridgeDegradation,
) error {
	var result *bridgepkg.BridgeInstance
	err := p.retryHostCall(ctx, func(callCtx context.Context) error {
		instance, callErr := session.HostAPI().
			ReportBridgeInstanceState(callCtx, extensioncontract.BridgesInstancesReportStateParams{
				BridgeInstanceID: strings.TrimSpace(bridgeInstanceID),
				Status:           status,
				Degradation:      cloneDegradation(degradation),
			})
		if callErr == nil {
			result = instance
		}
		return callErr
	})
	if err != nil {
		p.reportSideEffectError("write failed state marker", appendJSONLine(p.env.statePath, stateMarker{
			BridgeInstanceID: strings.TrimSpace(bridgeInstanceID),
			Status:           status,
			Error:            err.Error(),
		}))
		return err
	}

	p.mu.Lock()
	p.reportedStatus[strings.TrimSpace(bridgeInstanceID)] = result.Status.Normalize()
	p.mu.Unlock()
	p.reportSideEffectError("write state marker", appendJSONLine(p.env.statePath, stateMarker{
		BridgeInstanceID: result.ID,
		Status:           result.Status,
		Instance:         *result,
	}))
	return nil
}

func (p *slackProvider) reportReadyIfNeeded(ctx context.Context, session *bridgesdk.Session, bridgeInstanceID string) {
	p.mu.RLock()
	status := p.reportedStatus[strings.TrimSpace(bridgeInstanceID)]
	p.mu.RUnlock()
	if status == bridgepkg.BridgeStatusReady {
		return
	}
	if err := p.reportState(ctx, session, bridgeInstanceID, bridgepkg.BridgeStatusReady, nil); err != nil {
		p.setLastError(err)
	}
}

func (p *slackProvider) ingestBridgeMessage(
	ctx context.Context,
	session *bridgesdk.Session,
	envelope bridgepkg.InboundMessageEnvelope,
) (*extensioncontract.BridgesMessagesIngestResult, error) {
	var result *extensioncontract.BridgesMessagesIngestResult
	err := p.retryHostCall(ctx, func(callCtx context.Context) error {
		ingestResult, callErr := session.HostAPI().IngestBridgeMessage(callCtx, envelope)
		if callErr == nil {
			result = ingestResult
		}
		return callErr
	})
	return result, err
}

func (p *slackProvider) retryHostCall(ctx context.Context, fn func(context.Context) error) error {
	if ctx == nil {
		ctx = context.Background()
	}

	delay := 10 * time.Millisecond
	var lastErr error
	for range 6 {
		err := fn(ctx)
		if err == nil {
			return nil
		}
		if !isNotInitializedRPCError(err) {
			return err
		}
		lastErr = err

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-p.stopCh:
			if !timer.Stop() {
				<-timer.C
			}
			return err
		case <-timer.C:
		}

		if delay < 100*time.Millisecond {
			delay *= 2
			if delay > 100*time.Millisecond {
				delay = 100 * time.Millisecond
			}
		}
	}

	if lastErr != nil {
		return lastErr
	}
	return nil
}

func (p *slackProvider) reconcileInstanceConfigs(
	ctx context.Context,
	session *bridgesdk.Session,
	managed []subprocess.InitializeBridgeManagedInstance,
) []resolvedInstanceConfig {
	if len(managed) == 0 {
		batchersToClose := make(map[*bridgesdk.InboundBatcher]struct{})
		p.mu.Lock()
		for _, cfg := range p.routes {
			if cfg.batcher != nil {
				batchersToClose[cfg.batcher] = struct{}{}
			}
		}
		p.routes = make(map[string]resolvedInstanceConfig)
		p.mu.Unlock()
		closeInboundBatchers(batchersToClose)
		return nil
	}

	configs, requestedListen := p.collectSlackConfigs(session, managed)
	p.applySlackListenErrors(configs, requestedListen)
	nextRoutes := buildSlackRouteMap(configs)
	closeInboundBatchers(p.swapSlackRoutes(nextRoutes, requestedListen))

	for idx := range configs {
		status, degradation, err := p.determineInitialState(ctx, configs[idx])
		if err != nil {
			p.setLastError(err)
		}
		configs[idx].initialStatus = status
		configs[idx].initialDegradation = degradation
	}

	return configs
}

func (p *slackProvider) collectSlackConfigs(
	session *bridgesdk.Session,
	managed []subprocess.InitializeBridgeManagedInstance,
) ([]resolvedInstanceConfig, string) {
	configs := make([]resolvedInstanceConfig, 0, len(managed))
	requestedListen := strings.TrimSpace(os.Getenv(slackListenAddrEnv))
	usedPaths := make(map[string]int, len(managed))

	for _, item := range managed {
		cfg := p.resolveInstanceConfig(session, item)
		requestedListen = applySlackListenConstraint(&cfg, requestedListen)
		applySlackWebhookPathConflict(&cfg, usedPaths, configs)
		configs = append(configs, cfg)
	}

	return configs, requestedListen
}

func applySlackListenConstraint(cfg *resolvedInstanceConfig, requestedListen string) string {
	if cfg == nil || cfg.listenAddr == "" {
		return requestedListen
	}
	if requestedListen == "" {
		return cfg.listenAddr
	}
	if requestedListen != cfg.listenAddr && cfg.configError == nil {
		cfg.configError = fmt.Errorf(
			"slack: instance %q configured incompatible listen_addr %q (runtime uses %q)",
			cfg.instanceID,
			cfg.listenAddr,
			requestedListen,
		)
	}
	return requestedListen
}

func applySlackWebhookPathConflict(
	cfg *resolvedInstanceConfig,
	usedPaths map[string]int,
	configs []resolvedInstanceConfig,
) {
	if cfg == nil || cfg.webhookPath == "" {
		return
	}
	if ownerIdx, ok := usedPaths[cfg.webhookPath]; ok {
		ownerID := ""
		if ownerIdx >= 0 && ownerIdx < len(configs) {
			ownerID = configs[ownerIdx].instanceID
		}
		conflictErr := fmt.Errorf(
			"slack: webhook path %q is shared by %q and %q",
			cfg.webhookPath,
			ownerID,
			cfg.instanceID,
		)
		if ownerIdx >= 0 && ownerIdx < len(configs) && configs[ownerIdx].configError == nil {
			configs[ownerIdx].configError = conflictErr
		}
		if cfg.configError == nil {
			cfg.configError = conflictErr
		}
		return
	}
	usedPaths[cfg.webhookPath] = len(configs)
}

func (p *slackProvider) applySlackListenErrors(configs []resolvedInstanceConfig, requestedListen string) {
	if requestedListen == "" {
		for idx := range configs {
			if configs[idx].configError == nil {
				configs[idx].configError = errors.New("slack: webhook listen address is required")
			}
		}
		return
	}
	if err := p.startServer(requestedListen); err != nil {
		for idx := range configs {
			if configs[idx].configError == nil {
				configs[idx].configError = err
			}
		}
	}
}

func buildSlackRouteMap(configs []resolvedInstanceConfig) map[string]resolvedInstanceConfig {
	nextRoutes := make(map[string]resolvedInstanceConfig, len(configs))
	for idx := range configs {
		cfg := configs[idx]
		nextRoutes[cfg.instanceID] = cfg
	}
	return nextRoutes
}

func (p *slackProvider) swapSlackRoutes(
	nextRoutes map[string]resolvedInstanceConfig,
	requestedListen string,
) map[*bridgesdk.InboundBatcher]struct{} {
	batchersToClose := make(map[*bridgesdk.InboundBatcher]struct{})
	p.mu.Lock()
	defer p.mu.Unlock()

	existing := p.routes
	for instanceID, cfg := range nextRoutes {
		if prior, ok := existing[instanceID]; ok && prior.batcher != nil && prior.batcher != cfg.batcher {
			batchersToClose[prior.batcher] = struct{}{}
		}
	}
	for instanceID := range existing {
		prior := existing[instanceID]
		if _, ok := nextRoutes[instanceID]; ok {
			continue
		}
		if prior.batcher != nil {
			batchersToClose[prior.batcher] = struct{}{}
		}
	}
	p.routes = nextRoutes
	p.listenAddr = requestedListen
	return batchersToClose
}

func (p *slackProvider) resolveInstanceConfig(
	session *bridgesdk.Session,
	managed subprocess.InitializeBridgeManagedInstance,
) resolvedInstanceConfig {
	cfg := slackProviderConfig{}
	if len(managed.Instance.ProviderConfig) > 0 {
		if err := json.Unmarshal(managed.Instance.ProviderConfig, &cfg); err != nil {
			return resolvedInstanceConfig{
				managed:     managed,
				instanceID:  managed.Instance.ID,
				configError: fmt.Errorf("slack: decode provider_config for %q: %w", managed.Instance.ID, err),
			}
		}
	}

	botToken, _ := session.Cache().BoundSecretValue(managed.Instance.ID, "bot_token")
	signingSecret, _ := session.Cache().BoundSecretValue(managed.Instance.ID, "signing_secret")
	listenAddr := firstNonEmpty(cfg.Webhook.ListenAddr, strings.TrimSpace(os.Getenv(slackListenAddrEnv)))
	webhookPath := normalizeWebhookPath(
		firstNonEmpty(cfg.Webhook.Path, "/slack/"+strings.TrimSpace(managed.Instance.ID)),
	)
	apiBaseURL := normalizeURL(
		firstNonEmpty(cfg.APIBaseURL, strings.TrimSpace(os.Getenv(slackAPIBaseEnv)), slackDefaultAPIBaseURL),
	)

	resolved := resolvedInstanceConfig{
		managed:         managed,
		instanceID:      strings.TrimSpace(managed.Instance.ID),
		listenAddr:      listenAddr,
		webhookPath:     webhookPath,
		apiBaseURL:      apiBaseURL,
		botToken:        strings.TrimSpace(botToken),
		signingSecret:   strings.TrimSpace(signingSecret),
		dmPolicy:        managed.Instance.DMPolicy.Normalize(),
		allowUserIDs:    buildSlackIDSet(cfg.DM.AllowUserIDs),
		allowUsernames:  buildSlackUsernameSet(cfg.DM.AllowUsernames),
		pairedUserIDs:   buildSlackIDSet(cfg.DM.PairedUserIDs),
		pairedUsernames: buildSlackUsernameSet(cfg.DM.PairedUsernames),
		dedup:           bridgesdk.NewDedupCache(5*time.Minute, 4000),
		rateLimiter:     bridgesdk.NewFixedWindowRateLimiter(200, time.Minute),
		inFlightLimiter: bridgesdk.NewInFlightLimiter(24),
	}
	if resolved.dmPolicy == "" {
		resolved.dmPolicy = bridgepkg.BridgeDMPolicyOpen
	}
	if resolved.webhookPath == "" {
		resolved.configError = errors.New("slack: webhook path is required")
		return resolved
	}

	if cfg.Batching.DelayMS > 0 {
		batcher, err := bridgesdk.NewInboundBatcher(bridgesdk.InboundBatcherConfig{
			Context: context.Background(),
			Delay:   time.Duration(cfg.Batching.DelayMS) * time.Millisecond,
			SplitDelay: func() time.Duration {
				if cfg.Batching.SplitDelayMS <= 0 {
					return time.Duration(cfg.Batching.DelayMS) * time.Millisecond
				}
				return time.Duration(cfg.Batching.SplitDelayMS) * time.Millisecond
			}(),
			SplitThreshold: cfg.Batching.SplitThreshold,
			Dispatch: func(ctx context.Context, batch bridgesdk.InboundBatch) error {
				return p.dispatchInboundBatch(ctx, resolved.instanceID, batch)
			},
			Now: func() time.Time { return p.now() },
		})
		if err != nil {
			resolved.configError = err
			return resolved
		}
		resolved.batcher = batcher
	}

	return resolved
}

func (p *slackProvider) determineInitialState(
	ctx context.Context,
	cfg resolvedInstanceConfig,
) (bridgepkg.BridgeStatus, *bridgepkg.BridgeDegradation, error) {
	if cfg.configError != nil {
		return bridgepkg.BridgeStatusDegraded, &bridgepkg.BridgeDegradation{
			Reason:  bridgepkg.BridgeDegradationReasonTenantConfigInvalid,
			Message: cfg.configError.Error(),
		}, cfg.configError
	}
	if strings.TrimSpace(cfg.botToken) == "" {
		err := errors.New("slack: bot_token secret binding is required")
		return bridgepkg.BridgeStatusAuthRequired, &bridgepkg.BridgeDegradation{
			Reason:  bridgepkg.BridgeDegradationReasonAuthFailed,
			Message: err.Error(),
		}, err
	}
	if strings.TrimSpace(cfg.signingSecret) == "" {
		err := errors.New("slack: signing_secret secret binding is required")
		return bridgepkg.BridgeStatusAuthRequired, &bridgepkg.BridgeDegradation{
			Reason:  bridgepkg.BridgeDegradationReasonAuthFailed,
			Message: err.Error(),
		}, err
	}
	_, err := p.apiFactory(cfg).AuthTest(ctx)
	if err != nil {
		classified := bridgesdk.ClassifyError(err)
		recovery := classified.Recovery()
		status := recovery.Status
		if status == "" {
			status = bridgepkg.BridgeStatusError
		}
		if recovery.Degradation != nil {
			return status, recovery.Degradation, err
		}
		return status, &bridgepkg.BridgeDegradation{
			Reason:  bridgepkg.BridgeDegradationReasonProviderTimeout,
			Message: classified.Message,
		}, err
	}
	return bridgepkg.BridgeStatusReady, nil, nil
}

func (p *slackProvider) startServer(listenAddr string) error {
	p.mu.RLock()
	server := p.server
	currentListen := p.listenAddr
	p.mu.RUnlock()
	if server != nil {
		if currentListen != "" && currentListen != strings.TrimSpace(listenAddr) {
			return fmt.Errorf("slack: runtime already listening on %q, cannot switch to %q", currentListen, listenAddr)
		}
		return nil
	}

	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", strings.TrimSpace(listenAddr))
	if err != nil {
		return fmt.Errorf("slack: listen %q: %w", listenAddr, err)
	}

	httpServer := &http.Server{
		Handler:           http.HandlerFunc(p.serveWebhookHTTP),
		ReadHeaderTimeout: slackWebhookReadHeaderTimeout,
		IdleTimeout:       slackWebhookIdleTimeout,
	}

	actualAddr := ln.Addr().String()
	p.mu.Lock()
	p.server = httpServer
	p.serverListener = ln
	p.serverAddr = actualAddr
	p.listenAddr = strings.TrimSpace(listenAddr)
	p.mu.Unlock()

	p.reportSideEffectError(
		"write start marker",
		appendMarkerLine(p.env.startsPath, fmt.Sprintf("listen=%s", actualAddr)),
	)

	p.wg.Go(func() {
		if serveErr := httpServer.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			p.setLastError(serveErr)
		}
	})

	return nil
}

func (p *slackProvider) serveWebhookHTTP(w http.ResponseWriter, r *http.Request) {
	cfg, ok := p.configForPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	handler, err := bridgesdk.NewWebhookHandler(bridgesdk.WebhookGuardConfig{
		AllowedMethods:      []string{http.MethodPost},
		AllowedContentTypes: []string{"application/json", "application/x-www-form-urlencoded"},
		MaxBodyBytes:        1 << 20,
		RateLimiter:         cfg.rateLimiter,
		InFlightLimiter:     cfg.inFlightLimiter,
		VerifySignature: func(ctx context.Context, req *http.Request, body []byte) error {
			return verifySlackSignature(ctx, req, body, cfg.signingSecret, p.now())
		},
		RequestKey: func(req *http.Request) string {
			return req.RemoteAddr + "|" + cfg.instanceID
		},
		Now: func() time.Time { return p.now() },
	}, func(w http.ResponseWriter, r *http.Request, request bridgesdk.WebhookRequest) error {
		return p.handleWebhookRequest(w, r, cfg, request)
	})
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		p.setLastError(err)
		return
	}
	handler.ServeHTTP(w, r)
}

func (p *slackProvider) handleWebhookRequest(
	w http.ResponseWriter,
	r *http.Request,
	cfg resolvedInstanceConfig,
	request bridgesdk.WebhookRequest,
) error {
	contentType := strings.TrimSpace(r.Header.Get("Content-Type"))
	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		return p.handleFormWebhook(r.Context(), w, cfg, request)
	}
	return p.handleJSONWebhook(r.Context(), w, cfg, request)
}

func (p *slackProvider) handleFormWebhook(
	ctx context.Context,
	w http.ResponseWriter,
	cfg resolvedInstanceConfig,
	request bridgesdk.WebhookRequest,
) error {
	values, err := url.ParseQuery(string(request.Body))
	if err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: "invalid slack form payload"}
	}
	if values.Has("command") && !values.Has("payload") {
		mapped, err := mapSlackSlashCommand(values, cfg.managed, request.ReceivedAt)
		if err != nil {
			return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
		}
		if cfg.dedup.Seen(mapped.Envelope.IdempotencyKey) {
			return writeWebhookOK(w)
		}
		if !allowSlackDirectMessage(cfg, mapped.User, mapped.Direct) {
			return writeWebhookOK(w)
		}
		if err := p.dispatchInboundEnvelope(ctx, cfg.instanceID, mapped.Envelope); err != nil {
			return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
		}
		cfg.dedup.Mark(mapped.Envelope.IdempotencyKey)
		return writeWebhookOK(w)
	}

	payloadStr := strings.TrimSpace(values.Get("payload"))
	if payloadStr == "" {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: "missing slack interactive payload"}
	}
	var payload slackBlockActionsPayload
	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: "invalid slack interactive payload"}
	}
	if strings.TrimSpace(payload.Type) != "block_actions" {
		return writeWebhookOK(w)
	}

	mapped, err := mapSlackBlockActions(payload, cfg.managed, request.ReceivedAt)
	if err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
	}
	for _, item := range mapped {
		if cfg.dedup.Seen(item.Envelope.IdempotencyKey) {
			continue
		}
		if !allowSlackDirectMessage(cfg, item.User, item.Direct) {
			continue
		}
		if err := p.dispatchInboundEnvelope(ctx, cfg.instanceID, item.Envelope); err != nil {
			return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
		}
		cfg.dedup.Mark(item.Envelope.IdempotencyKey)
	}
	return writeWebhookOK(w)
}

func (p *slackProvider) handleJSONWebhook(
	ctx context.Context,
	w http.ResponseWriter,
	cfg resolvedInstanceConfig,
	request bridgesdk.WebhookRequest,
) error {
	var payload slackWebhookEnvelope
	if err := json.Unmarshal(request.Body, &payload); err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: "invalid slack webhook payload"}
	}
	if handled, err := handleSlackJSONHandshake(w, payload); handled || err != nil {
		return err
	}
	if len(payload.Event) == 0 {
		return writeWebhookOK(w)
	}

	var eventType slackEventTypePayload
	if err := json.Unmarshal(payload.Event, &eventType); err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: "invalid slack event payload"}
	}

	switch strings.TrimSpace(eventType.Type) {
	case "message", "app_mention":
		return p.handleSlackMessageJSONEvent(ctx, w, cfg, request, payload)
	case "reaction_added", "reaction_removed":
		return p.handleSlackReactionJSONEvent(ctx, w, cfg, request, payload)
	default:
		return writeWebhookOK(w)
	}
}

func handleSlackJSONHandshake(w http.ResponseWriter, payload slackWebhookEnvelope) (bool, error) {
	switch strings.TrimSpace(payload.Type) {
	case "url_verification":
		if strings.TrimSpace(payload.Challenge) == "" {
			return true, &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: "missing slack challenge"}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		return true, json.NewEncoder(w).Encode(map[string]string{"challenge": payload.Challenge})
	case "event_callback":
		return false, nil
	default:
		return true, writeWebhookOK(w)
	}
}

func (p *slackProvider) handleSlackMessageJSONEvent(
	ctx context.Context,
	w http.ResponseWriter,
	cfg resolvedInstanceConfig,
	request bridgesdk.WebhookRequest,
	payload slackWebhookEnvelope,
) error {
	var event slackMessageEvent
	if err := json.Unmarshal(payload.Event, &event); err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: "invalid slack message event"}
	}
	mapped, ignored, err := mapSlackMessageEvent(
		event,
		cfg.managed,
		request.ReceivedAt,
		payload.EventID,
		payload.TeamID,
		payload.EventTime,
	)
	if err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
	}
	if ignored {
		return writeWebhookOK(w)
	}
	return p.dispatchSlackWebhookEnvelope(ctx, w, cfg, mapped, true)
}

func (p *slackProvider) handleSlackReactionJSONEvent(
	ctx context.Context,
	w http.ResponseWriter,
	cfg resolvedInstanceConfig,
	request bridgesdk.WebhookRequest,
	payload slackWebhookEnvelope,
) error {
	var event slackReactionEvent
	if err := json.Unmarshal(payload.Event, &event); err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: "invalid slack reaction event"}
	}
	mapped, err := mapSlackReactionEvent(event, cfg.managed, request.ReceivedAt, payload.EventID, payload.TeamID)
	if err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
	}
	return p.dispatchSlackWebhookEnvelope(ctx, w, cfg, mapped, false)
}

func (p *slackProvider) dispatchSlackWebhookEnvelope(
	ctx context.Context,
	w http.ResponseWriter,
	cfg resolvedInstanceConfig,
	mapped slackMappedInbound,
	allowBatch bool,
) error {
	if cfg.dedup.Seen(mapped.Envelope.IdempotencyKey) {
		return writeWebhookOK(w)
	}
	if !allowSlackDirectMessage(cfg, mapped.User, mapped.Direct) {
		return writeWebhookOK(w)
	}
	if allowBatch && cfg.batcher != nil {
		if err := cfg.batcher.Enqueue(mapped.Envelope); err != nil {
			return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
		}
		cfg.dedup.Mark(mapped.Envelope.IdempotencyKey)
		return writeWebhookOK(w)
	}
	if err := p.dispatchInboundEnvelope(ctx, cfg.instanceID, mapped.Envelope); err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
	}
	cfg.dedup.Mark(mapped.Envelope.IdempotencyKey)
	return writeWebhookOK(w)
}

func (p *slackProvider) dispatchInboundBatch(
	ctx context.Context,
	bridgeInstanceID string,
	batch bridgesdk.InboundBatch,
) error {
	if len(batch.Items) == 0 {
		return nil
	}
	merged := batch.Items[0]
	if len(batch.Items) > 1 {
		parts := make([]string, 0, len(batch.Items))
		for _, item := range batch.Items {
			if text := strings.TrimSpace(item.Content.Text); text != "" {
				parts = append(parts, text)
			}
		}
		merged.Content.Text = strings.Join(parts, "\n")
		merged.IdempotencyKey = fmt.Sprintf("%s:batch:%d", merged.IdempotencyKey, len(batch.Items))
	}
	return p.dispatchInboundEnvelope(ctx, bridgeInstanceID, merged)
}

func (p *slackProvider) dispatchInboundEnvelope(
	ctx context.Context,
	bridgeInstanceID string,
	envelope bridgepkg.InboundMessageEnvelope,
) error {
	session := p.currentSession()
	if session == nil {
		return errors.New("slack: runtime session is not initialized")
	}
	cfg, err := p.configForInstance(bridgeInstanceID)
	if err != nil {
		return err
	}

	result, err := p.ingestBridgeMessage(ctx, session, envelope)
	if err != nil {
		p.reportSideEffectError("write failed ingest marker", appendJSONLine(p.env.ingestPath, ingestMarker{
			Envelope: envelope,
			Error:    err.Error(),
		}))
		return err
	}
	p.reportSideEffectError("write ingest marker", appendJSONLine(p.env.ingestPath, ingestMarker{
		Envelope: envelope,
		Result:   *result,
	}))
	p.reportReadyIfNeeded(ctx, session, cfg.instanceID)
	return nil
}

func (p *slackProvider) configForInstance(instanceID string) (resolvedInstanceConfig, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	cfg, ok := p.routes[strings.TrimSpace(instanceID)]
	if !ok {
		return resolvedInstanceConfig{}, fmt.Errorf("slack: delivery targeted unmanaged instance %q", instanceID)
	}
	return cfg, nil
}

func (p *slackProvider) waitForInstanceConfig(
	instanceID string,
	timeout time.Duration,
) (resolvedInstanceConfig, error) {
	if timeout <= 0 {
		return p.configForInstance(instanceID)
	}

	deadline := time.Now().Add(timeout)
	for {
		cfg, err := p.configForInstance(instanceID)
		if err == nil {
			return cfg, nil
		}
		if time.Now().After(deadline) {
			return resolvedInstanceConfig{}, err
		}

		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-p.stopCh:
			if !timer.Stop() {
				<-timer.C
			}
			return resolvedInstanceConfig{}, err
		case <-timer.C:
		}
	}
}

func (p *slackProvider) configForPath(path string) (resolvedInstanceConfig, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, cfg := range p.routes {
		if cfg.configError != nil {
			continue
		}
		if cfg.webhookPath == normalizeWebhookPath(path) {
			return cfg, true
		}
	}
	return resolvedInstanceConfig{}, false
}

func (p *slackProvider) currentSession() *bridgesdk.Session {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.session
}

func (p *slackProvider) deliveryState(instanceID string, deliveryID string) deliveryState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.deliveries[deliveryStateKey(instanceID, deliveryID)]
}

func (p *slackProvider) storeDeliveryState(instanceID string, deliveryID string, state deliveryState) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.deliveries[deliveryStateKey(instanceID, deliveryID)] = state
}

func closeInboundBatchers(batchers map[*bridgesdk.InboundBatcher]struct{}) {
	for batcher := range batchers {
		batcher.Close()
	}
}

func (p *slackProvider) setLastError(err error) {
	if err == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastError = err.Error()
}

func (p *slackProvider) clearLastError() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastError = ""
}

func (p *slackProvider) reportSideEffectError(action string, err error) {
	reportSideEffectError(p.stderr, action, err)
}

func executeDelivery(
	ctx context.Context,
	api slackAPI,
	request bridgepkg.DeliveryRequest,
	state deliveryState,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	if err := request.Validate(); err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}

	event := request.Event
	if event.EventType != bridgepkg.DeliveryEventTypeResume && event.Seq <= state.LastSeq {
		return bridgepkg.DeliveryAck{}, state, fmt.Errorf(
			"slack: out-of-order delivery seq %d after %d",
			event.Seq,
			state.LastSeq,
		)
	}
	if event.EventType == bridgepkg.DeliveryEventTypeResume && request.Snapshot != nil {
		state.LastSeq = request.Snapshot.LastAckedSeq
		state.RemoteMessageID = strings.TrimSpace(request.Snapshot.RemoteMessageID)
		state.ReplaceRemoteMessageID = strings.TrimSpace(request.Snapshot.ReplaceRemoteMessageID)
	}

	channelID, threadTS, err := resolveDeliveryTarget(event)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}

	switch {
	case isSlackDeleteEvent(event):
		return executeSlackDelete(ctx, api, request, state)
	case shouldPostNewMessage(event, state, request):
		return executeSlackCreate(ctx, api, request, state, channelID, threadTS)
	default:
		return executeSlackUpdate(ctx, api, request, state)
	}
}

func isSlackDeleteEvent(event bridgepkg.DeliveryEvent) bool {
	return event.Operation.Normalize() == bridgepkg.DeliveryOperationDelete ||
		normalizeDeliveryEventType(event.EventType) == bridgepkg.DeliveryEventTypeDelete
}

func executeSlackDelete(
	ctx context.Context,
	api slackAPI,
	request bridgepkg.DeliveryRequest,
	state deliveryState,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	event := request.Event
	remoteID := slackRemoteMessageIDFromRequest(request, state)
	if remoteID == "" {
		return bridgepkg.DeliveryAck{}, state, errors.New("slack: delete delivery requires a remote message id")
	}
	channel, ts, err := decodeRemoteMessageID(remoteID)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	if err := api.DeleteMessage(ctx, slackDeleteMessageRequest{Channel: channel, TS: ts}); err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	ack := bridgepkg.DeliveryAck{
		DeliveryID:             event.DeliveryID,
		Seq:                    event.Seq,
		RemoteMessageID:        remoteID,
		ReplaceRemoteMessageID: firstNonEmpty(state.RemoteMessageID, remoteID),
	}
	state.LastSeq = event.Seq
	state.RemoteMessageID = remoteID
	state.ReplaceRemoteMessageID = ack.ReplaceRemoteMessageID
	return ack, state, ack.ValidateFor(event)
}

func executeSlackCreate(
	ctx context.Context,
	api slackAPI,
	request bridgepkg.DeliveryRequest,
	state deliveryState,
	channelID string,
	threadTS string,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	event := request.Event
	if normalizeDeliveryEventType(event.EventType) == bridgepkg.DeliveryEventTypeResume {
		matched, err := reconcileSlackDelivery(ctx, api, event, channelID, threadTS)
		if err != nil {
			return bridgepkg.DeliveryAck{}, state, err
		}
		if matched != nil {
			return slackCreateAck(event, state, channelID, matched.TS)
		}
	}

	sent, err := api.PostMessage(ctx, slackPostMessageRequest{
		Channel:  channelID,
		ThreadTS: threadTS,
		Text:     event.Content.Text,
		Metadata: slackDeliveryMetadata(event),
	})
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	return slackCreateAck(event, state, channelID, sent.TS)
}

func reconcileSlackDelivery(
	ctx context.Context,
	api slackAPI,
	event bridgepkg.DeliveryEvent,
	channelID string,
	threadTS string,
) (*slackPostedMessage, error) {
	reconciler, ok := api.(slackDeliveryReconciler)
	if !ok {
		return nil, nil
	}
	return reconciler.FindDeliveryMessage(ctx, slackFindDeliveryMessageRequest{
		Channel:          channelID,
		ThreadTS:         threadTS,
		DeliveryID:       event.DeliveryID,
		BridgeInstanceID: event.BridgeInstanceID,
	})
}

func slackCreateAck(
	event bridgepkg.DeliveryEvent,
	state deliveryState,
	channelID string,
	ts string,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	remoteID := encodeRemoteMessageID(channelID, ts)
	ack := bridgepkg.DeliveryAck{
		DeliveryID:      event.DeliveryID,
		Seq:             event.Seq,
		RemoteMessageID: remoteID,
	}
	state.LastSeq = event.Seq
	state.ReplaceRemoteMessageID = state.RemoteMessageID
	state.RemoteMessageID = remoteID
	if state.ReplaceRemoteMessageID != "" {
		ack.ReplaceRemoteMessageID = state.ReplaceRemoteMessageID
	}
	return ack, state, ack.ValidateFor(event)
}

func slackDeliveryMetadata(event bridgepkg.DeliveryEvent) *slackMessageMetadata {
	return &slackMessageMetadata{
		EventType: "agh_bridge_delivery",
		EventPayload: slackMessageMetadataPayload{
			BridgeInstanceID: strings.TrimSpace(event.BridgeInstanceID),
			DeliveryID:       strings.TrimSpace(event.DeliveryID),
		},
	}
}

func executeSlackUpdate(
	ctx context.Context,
	api slackAPI,
	request bridgepkg.DeliveryRequest,
	state deliveryState,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	event := request.Event
	remoteID := slackRemoteMessageIDFromRequest(request, state)
	if remoteID == "" {
		return bridgepkg.DeliveryAck{}, state, errors.New("slack: edit delivery requires a remote message id")
	}
	channel, ts, err := decodeRemoteMessageID(remoteID)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	if err := api.UpdateMessage(ctx, slackUpdateMessageRequest{
		Channel: channel,
		TS:      ts,
		Text:    event.Content.Text,
	}); err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	ack := bridgepkg.DeliveryAck{
		DeliveryID:             event.DeliveryID,
		Seq:                    event.Seq,
		RemoteMessageID:        remoteID,
		ReplaceRemoteMessageID: firstNonEmpty(state.RemoteMessageID, remoteID),
	}
	state.LastSeq = event.Seq
	state.RemoteMessageID = remoteID
	state.ReplaceRemoteMessageID = ack.ReplaceRemoteMessageID
	return ack, state, ack.ValidateFor(event)
}

func slackRemoteMessageIDFromRequest(request bridgepkg.DeliveryRequest, state deliveryState) string {
	remoteID := firstNonEmpty(referenceRemoteMessageID(request.Event.Reference), state.RemoteMessageID)
	if remoteID == "" && request.Snapshot != nil {
		return strings.TrimSpace(request.Snapshot.RemoteMessageID)
	}
	return remoteID
}

func shouldPostNewMessage(event bridgepkg.DeliveryEvent, state deliveryState, request bridgepkg.DeliveryRequest) bool {
	if normalizeDeliveryEventType(event.EventType) == bridgepkg.DeliveryEventTypeStart {
		return true
	}
	if normalizeDeliveryEventType(event.EventType) == bridgepkg.DeliveryEventTypeResume {
		if request.Snapshot == nil {
			return state.RemoteMessageID == ""
		}
		return strings.TrimSpace(request.Snapshot.RemoteMessageID) == ""
	}
	return strings.TrimSpace(state.RemoteMessageID) == ""
}

func mapSlackMessageEvent(
	event slackMessageEvent,
	managed subprocess.InitializeBridgeManagedInstance,
	receivedAt time.Time,
	eventID string,
	teamID string,
	eventTime int64,
) (slackMappedInbound, bool, error) {
	if strings.TrimSpace(event.Channel) == "" || strings.TrimSpace(event.TS) == "" {
		return slackMappedInbound{}, false, errors.New("slack: message event requires channel and ts")
	}
	if isIgnoredSlackMessageEvent(event) {
		return slackMappedInbound{}, true, nil
	}
	if receivedAt.IsZero() {
		receivedAt = time.Now().UTC()
	}
	if eventTime > 0 {
		receivedAt = time.Unix(eventTime, 0).UTC()
	}

	direct := isSlackDirectConversation(event.ChannelType, event.Channel)
	threadID := inboundSlackThreadID(direct, event.TS, event.ThreadTS)
	user := slackUserIdentity{
		ID:          normalizeSlackUserID(event.User),
		Username:    normalizeUsername(event.Username),
		DisplayName: firstNonEmpty(strings.TrimSpace(event.Username), normalizeSlackUserID(event.User)),
	}
	envelope := bridgepkg.InboundMessageEnvelope{
		BridgeInstanceID:  managed.Instance.ID,
		Scope:             managed.Instance.Scope,
		WorkspaceID:       managed.Instance.WorkspaceID,
		PlatformMessageID: strings.TrimSpace(event.TS),
		ReceivedAt:        receivedAt,
		Sender: bridgepkg.MessageSender{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
		},
		Content: bridgepkg.MessageContent{
			Text: strings.TrimSpace(event.Text),
		},
		Attachments: normalizeSlackAttachments(event.Files),
		EventFamily: bridgepkg.InboundEventFamilyMessage,
		IdempotencyKey: firstNonEmpty(
			strings.TrimSpace(eventID),
			fmt.Sprintf(
				"slack:%s:message:%s:%s",
				managed.Instance.ID,
				strings.TrimSpace(event.Channel),
				strings.TrimSpace(event.TS),
			),
		),
	}
	if direct {
		envelope.PeerID = strings.TrimSpace(event.Channel)
		envelope.ThreadID = threadID
	} else {
		envelope.GroupID = strings.TrimSpace(event.Channel)
		envelope.ThreadID = threadID
	}
	metadata, err := json.Marshal(map[string]any{
		"channel_id":   strings.TrimSpace(event.Channel),
		"channel_type": strings.TrimSpace(event.ChannelType),
		"event_id":     strings.TrimSpace(eventID),
		"subtype":      strings.TrimSpace(event.Subtype),
		"team_id":      firstNonEmpty(strings.TrimSpace(event.TeamID), strings.TrimSpace(teamID)),
		"thread_ts":    strings.TrimSpace(event.ThreadTS),
		"ts":           strings.TrimSpace(event.TS),
		"type":         strings.TrimSpace(event.Type),
	})
	if err == nil {
		envelope.ProviderMetadata = metadata
	}
	if err := envelope.Validate(); err != nil {
		return slackMappedInbound{}, false, err
	}
	return slackMappedInbound{Envelope: envelope, Direct: direct, User: user}, false, nil
}

func mapSlackSlashCommand(
	values url.Values,
	managed subprocess.InitializeBridgeManagedInstance,
	receivedAt time.Time,
) (slackMappedInbound, error) {
	if receivedAt.IsZero() {
		receivedAt = time.Now().UTC()
	}
	command := strings.TrimSpace(values.Get("command"))
	channelID := strings.TrimSpace(values.Get("channel_id"))
	userID := normalizeSlackUserID(values.Get("user_id"))
	if command == "" || channelID == "" || userID == "" {
		return slackMappedInbound{}, errors.New("slack: slash command requires command, channel_id, and user_id")
	}
	direct := isSlackSlashCommandDirect(values.Get("channel_name"), channelID)
	user := slackUserIdentity{
		ID:          userID,
		Username:    normalizeUsername(values.Get("user_name")),
		DisplayName: firstNonEmpty(normalizeUsername(values.Get("user_name")), userID),
	}

	envelope := bridgepkg.InboundMessageEnvelope{
		BridgeInstanceID: managed.Instance.ID,
		Scope:            managed.Instance.Scope,
		WorkspaceID:      managed.Instance.WorkspaceID,
		ReceivedAt:       receivedAt,
		Sender: bridgepkg.MessageSender{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
		},
		EventFamily: bridgepkg.InboundEventFamilyCommand,
		Command: &bridgepkg.InboundCommand{
			Command:   command,
			Text:      strings.TrimSpace(values.Get("text")),
			TriggerID: strings.TrimSpace(values.Get("trigger_id")),
		},
		IdempotencyKey: firstNonEmpty(
			strings.TrimSpace(values.Get("trigger_id")),
			fmt.Sprintf("slack:%s:command:%s:%s:%s", managed.Instance.ID, channelID, userID, command),
		),
	}
	if direct {
		envelope.PeerID = channelID
		envelope.ThreadID = slackDirectRootThreadID(channelID)
	} else {
		envelope.GroupID = channelID
		envelope.ThreadID = slackDirectRootThreadID(channelID)
	}
	metadata, err := json.Marshal(map[string]any{
		"channel_id":   channelID,
		"channel_name": strings.TrimSpace(values.Get("channel_name")),
		"response_url": strings.TrimSpace(values.Get("response_url")),
		"team_id":      strings.TrimSpace(values.Get("team_id")),
		"trigger_id":   strings.TrimSpace(values.Get("trigger_id")),
		"type":         "slash_command",
	})
	if err == nil {
		envelope.ProviderMetadata = metadata
	}
	if err := envelope.Validate(); err != nil {
		return slackMappedInbound{}, err
	}
	return slackMappedInbound{Envelope: envelope, Direct: direct, User: user}, nil
}

func mapSlackBlockActions(
	payload slackBlockActionsPayload,
	managed subprocess.InitializeBridgeManagedInstance,
	receivedAt time.Time,
) ([]slackMappedInbound, error) {
	if receivedAt.IsZero() {
		receivedAt = time.Now().UTC()
	}
	ctx, err := newSlackBlockActionContext(payload, managed, receivedAt)
	if err != nil {
		return nil, err
	}
	items := make([]slackMappedInbound, 0, len(payload.Actions))
	for idx := range payload.Actions {
		item, err := buildSlackBlockActionItem(ctx, payload, payload.Actions[idx])
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

type slackBlockActionContext struct {
	managed    subprocess.InitializeBridgeManagedInstance
	receivedAt time.Time
	channelID  string
	messageTS  string
	threadTS   string
	messageID  string
	threadID   string
	direct     bool
	user       slackUserIdentity
}

func newSlackBlockActionContext(
	payload slackBlockActionsPayload,
	managed subprocess.InitializeBridgeManagedInstance,
	receivedAt time.Time,
) (slackBlockActionContext, error) {
	if len(payload.Actions) == 0 {
		return slackBlockActionContext{}, errors.New("slack: block actions payload requires at least one action")
	}
	channelID := firstNonEmpty(strings.TrimSpace(payload.Channel.ID), strings.TrimSpace(payload.Container.ChannelID))
	messageTS := firstNonEmpty(strings.TrimSpace(payload.Message.TS), strings.TrimSpace(payload.Container.MessageTS))
	threadTS := firstNonEmpty(
		strings.TrimSpace(payload.Message.ThreadTS),
		strings.TrimSpace(payload.Container.ThreadTS),
	)
	userID := normalizeSlackUserID(payload.User.ID)
	if channelID == "" || messageTS == "" || userID == "" {
		return slackBlockActionContext{}, errors.New(
			"slack: block actions payload requires channel, message ts, and user id",
		)
	}
	direct := isSlackDirectConversation("", channelID)
	user := slackUserIdentity{
		ID:       userID,
		Username: normalizeUsername(firstNonEmpty(payload.User.Username, payload.User.Name)),
		DisplayName: firstNonEmpty(
			strings.TrimSpace(payload.User.Name),
			strings.TrimSpace(payload.User.Username),
			userID,
		),
	}
	return slackBlockActionContext{
		managed:    managed,
		receivedAt: receivedAt,
		channelID:  channelID,
		messageTS:  messageTS,
		threadTS:   threadTS,
		messageID:  strings.TrimSpace(messageTS),
		threadID:   inboundSlackThreadID(direct, messageTS, threadTS),
		direct:     direct,
		user:       user,
	}, nil
}

func buildSlackBlockActionItem(
	ctx slackBlockActionContext,
	payload slackBlockActionsPayload,
	action slackBlockAction,
) (slackMappedInbound, error) {
	envelope := bridgepkg.InboundMessageEnvelope{
		BridgeInstanceID: ctx.managed.Instance.ID,
		Scope:            ctx.managed.Instance.Scope,
		WorkspaceID:      ctx.managed.Instance.WorkspaceID,
		ReceivedAt:       ctx.receivedAt,
		Sender: bridgepkg.MessageSender{
			ID:          ctx.user.ID,
			Username:    ctx.user.Username,
			DisplayName: ctx.user.DisplayName,
		},
		EventFamily: bridgepkg.InboundEventFamilyAction,
		Action: &bridgepkg.InboundAction{
			ActionID:  strings.TrimSpace(action.ActionID),
			MessageID: ctx.messageID,
			Value:     slackBlockActionValue(action),
			TriggerID: strings.TrimSpace(payload.TriggerID),
		},
		IdempotencyKey: firstNonEmpty(
			strings.TrimSpace(action.ActionTS),
			fmt.Sprintf(
				"slack:%s:action:%s:%s:%s",
				ctx.managed.Instance.ID,
				ctx.messageTS,
				ctx.user.ID,
				strings.TrimSpace(action.ActionID),
			),
		),
	}
	if ctx.direct {
		envelope.PeerID = ctx.channelID
		envelope.ThreadID = ctx.threadID
	} else {
		envelope.GroupID = ctx.channelID
		envelope.ThreadID = ctx.threadID
	}
	if metadata, err := slackBlockActionMetadata(payload, action, ctx); err == nil {
		envelope.ProviderMetadata = metadata
	}
	if err := envelope.Validate(); err != nil {
		return slackMappedInbound{}, err
	}
	return slackMappedInbound{Envelope: envelope, Direct: ctx.direct, User: ctx.user}, nil
}

func slackBlockActionValue(action slackBlockAction) string {
	value := strings.TrimSpace(action.Value)
	if action.SelectedOption != nil && strings.TrimSpace(action.SelectedOption.Value) != "" {
		return strings.TrimSpace(action.SelectedOption.Value)
	}
	return value
}

func slackBlockActionMetadata(
	payload slackBlockActionsPayload,
	action slackBlockAction,
	ctx slackBlockActionContext,
) ([]byte, error) {
	return json.Marshal(map[string]any{
		"action_ts":    strings.TrimSpace(action.ActionTS),
		"block_id":     strings.TrimSpace(action.BlockID),
		"channel_id":   ctx.channelID,
		"container":    strings.TrimSpace(payload.Container.Type),
		"is_ephemeral": payload.Container.IsEphemeral,
		"message_ts":   ctx.messageTS,
		"response_url": strings.TrimSpace(payload.ResponseURL),
		"thread_ts":    strings.TrimSpace(ctx.threadTS),
		"trigger_id":   strings.TrimSpace(payload.TriggerID),
		"type":         strings.TrimSpace(action.Type),
	})
}

func mapSlackReactionEvent(
	event slackReactionEvent,
	managed subprocess.InitializeBridgeManagedInstance,
	receivedAt time.Time,
	eventID string,
	teamID string,
) (slackMappedInbound, error) {
	if strings.TrimSpace(event.Item.Type) != "message" {
		return slackMappedInbound{}, errors.New("slack: reaction event item.type must be message")
	}
	if strings.TrimSpace(event.Item.Channel) == "" || strings.TrimSpace(event.Item.TS) == "" ||
		strings.TrimSpace(event.Reaction) == "" ||
		strings.TrimSpace(event.User) == "" {
		return slackMappedInbound{}, errors.New(
			"slack: reaction event requires item channel, item ts, reaction, and user",
		)
	}
	receivedAt = slackReactionReceivedAt(event, receivedAt)
	direct := isSlackDirectConversation("", event.Item.Channel)
	user := slackUserIdentity{
		ID:          normalizeSlackUserID(event.User),
		DisplayName: normalizeSlackUserID(event.User),
	}
	envelope := bridgepkg.InboundMessageEnvelope{
		BridgeInstanceID: managed.Instance.ID,
		Scope:            managed.Instance.Scope,
		WorkspaceID:      managed.Instance.WorkspaceID,
		ReceivedAt:       receivedAt,
		Sender: bridgepkg.MessageSender{
			ID:          user.ID,
			DisplayName: user.DisplayName,
		},
		EventFamily: bridgepkg.InboundEventFamilyReaction,
		Reaction: &bridgepkg.InboundReaction{
			MessageID: strings.TrimSpace(event.Item.TS),
			Emoji:     normalizeSlackEmoji(event.Reaction),
			RawEmoji:  strings.TrimSpace(event.Reaction),
			Added:     strings.TrimSpace(event.Type) == "reaction_added",
		},
		IdempotencyKey: firstNonEmpty(
			strings.TrimSpace(event.EventTS),
			strings.TrimSpace(eventID),
			fmt.Sprintf(
				"slack:%s:reaction:%s:%s:%s:%s",
				managed.Instance.ID,
				strings.TrimSpace(event.Item.Channel),
				strings.TrimSpace(event.Item.TS),
				user.ID,
				strings.TrimSpace(event.Reaction),
			),
		),
	}
	if direct {
		envelope.PeerID = strings.TrimSpace(event.Item.Channel)
		envelope.ThreadID = strings.TrimSpace(event.Item.TS)
	} else {
		envelope.GroupID = strings.TrimSpace(event.Item.Channel)
		envelope.ThreadID = strings.TrimSpace(event.Item.TS)
	}
	metadata, err := json.Marshal(map[string]any{
		"channel_id": strings.TrimSpace(event.Item.Channel),
		"event_id":   strings.TrimSpace(eventID),
		"event_ts":   strings.TrimSpace(event.EventTS),
		"item_user":  strings.TrimSpace(event.ItemUser),
		"message_ts": strings.TrimSpace(event.Item.TS),
		"team_id":    strings.TrimSpace(teamID),
		"type":       strings.TrimSpace(event.Type),
	})
	if err == nil {
		envelope.ProviderMetadata = metadata
	}
	if err := envelope.Validate(); err != nil {
		return slackMappedInbound{}, err
	}
	return slackMappedInbound{Envelope: envelope, Direct: direct, User: user}, nil
}

func slackReactionReceivedAt(event slackReactionEvent, fallback time.Time) time.Time {
	if fallback.IsZero() {
		fallback = time.Now().UTC()
	}
	if strings.TrimSpace(event.EventTS) == "" {
		return fallback
	}
	parsed, err := parseSlackTimestamp(strings.TrimSpace(event.EventTS))
	if err != nil {
		return fallback
	}
	return parsed
}

func allowSlackDirectMessage(cfg resolvedInstanceConfig, user slackUserIdentity, direct bool) bool {
	if !direct {
		return true
	}

	switch cfg.dmPolicy.Normalize() {
	case "", bridgepkg.BridgeDMPolicyOpen:
		return true
	case bridgepkg.BridgeDMPolicyAllowlist:
		return slackIdentityAllowed(cfg.allowUserIDs, cfg.allowUsernames, user)
	case bridgepkg.BridgeDMPolicyPairing:
		if slackIdentityAllowed(cfg.pairedUserIDs, cfg.pairedUsernames, user) {
			return true
		}
		return slackIdentityAllowed(cfg.allowUserIDs, cfg.allowUsernames, user)
	default:
		return false
	}
}

func slackIdentityAllowed(ids map[string]struct{}, usernames map[string]struct{}, user slackUserIdentity) bool {
	if len(ids) == 0 && len(usernames) == 0 {
		return false
	}
	if _, ok := ids[normalizeSlackUserID(user.ID)]; ok {
		return true
	}
	if _, ok := usernames[normalizeUsername(user.Username)]; ok {
		return true
	}
	return false
}

func resolveDeliveryTarget(event bridgepkg.DeliveryEvent) (string, string, error) {
	channelID := firstNonEmpty(
		strings.TrimSpace(event.DeliveryTarget.PeerID),
		strings.TrimSpace(event.DeliveryTarget.GroupID),
		strings.TrimSpace(event.RoutingKey.PeerID),
		strings.TrimSpace(event.RoutingKey.GroupID),
	)
	if channelID == "" {
		return "", "", errors.New("slack: delivery target requires peer_id or group_id")
	}
	threadTS := firstNonEmpty(
		strings.TrimSpace(event.DeliveryTarget.ThreadID),
		strings.TrimSpace(event.RoutingKey.ThreadID),
	)
	return channelID, threadTS, nil
}

func verifySlackSignature(_ context.Context, req *http.Request, body []byte, secret string, now time.Time) error {
	trimmedSecret := strings.TrimSpace(secret)
	if trimmedSecret == "" {
		return errors.New("slack: signing secret is required")
	}
	if req == nil {
		return errors.New("slack: webhook request is required")
	}

	timestamp := strings.TrimSpace(req.Header.Get("X-Slack-Request-Timestamp"))
	signature := strings.TrimSpace(req.Header.Get("X-Slack-Signature"))
	if timestamp == "" || signature == "" {
		return errors.New("slack: missing request signature headers")
	}
	tsValue, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return errors.New("slack: invalid request timestamp")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if delta := now.Unix() - tsValue; delta > 300 || delta < -300 {
		return errors.New("slack: stale request timestamp")
	}

	mac := hmac.New(sha256.New, []byte(trimmedSecret))
	_, _ = mac.Write([]byte(slackSignatureVersion + ":" + timestamp + ":"))
	_, _ = mac.Write(body)
	expected := slackSignatureVersion + "=" + hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return errors.New("slack: invalid request signature")
	}
	return nil
}

func (c *slackBotClient) AuthTest(ctx context.Context) (*slackAuthIdentity, error) {
	var result slackAuthIdentity
	if err := c.call(ctx, "auth.test", map[string]any{}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *slackBotClient) PostMessage(ctx context.Context, req slackPostMessageRequest) (*slackPostedMessage, error) {
	var result slackPostedMessage
	if err := c.call(ctx, "chat.postMessage", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *slackBotClient) FindDeliveryMessage(
	ctx context.Context,
	req slackFindDeliveryMessageRequest,
) (*slackPostedMessage, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	method := "conversations.history"
	payload := slackConversationMessagesRequest{
		Channel: strings.TrimSpace(req.Channel),
		Limit:   100,
	}
	if strings.TrimSpace(req.ThreadTS) != "" {
		method = "conversations.replies"
		payload.TS = strings.TrimSpace(req.ThreadTS)
		payload.Inclusive = true
	}

	for {
		var result slackConversationMessagesResponse
		if err := c.call(ctx, method, payload, &result); err != nil {
			return nil, err
		}
		for idx := range result.Messages {
			message := result.Messages[idx]
			if slackMetadataMatchesDelivery(message.Metadata, req) {
				return &slackPostedMessage{TS: strings.TrimSpace(message.TS)}, nil
			}
		}
		nextCursor := ""
		if result.ResponseMetadata != nil {
			nextCursor = strings.TrimSpace(result.ResponseMetadata.NextCursor)
		}
		if !result.HasMore && nextCursor == "" {
			break
		}
		payload.Cursor = nextCursor
	}
	return nil, nil
}

func (c *slackBotClient) UpdateMessage(ctx context.Context, req slackUpdateMessageRequest) error {
	var result slackPostedMessage
	return c.call(ctx, "chat.update", req, &result)
}

func (c *slackBotClient) DeleteMessage(ctx context.Context, req slackDeleteMessageRequest) error {
	var result json.RawMessage
	return c.call(ctx, "chat.delete", req, &result)
}

func slackMetadataMatchesDelivery(
	metadata *slackMessageMetadata,
	req slackFindDeliveryMessageRequest,
) bool {
	if metadata == nil {
		return false
	}
	if strings.TrimSpace(metadata.EventType) != "agh_bridge_delivery" {
		return false
	}
	return strings.TrimSpace(metadata.EventPayload.DeliveryID) == strings.TrimSpace(req.DeliveryID) &&
		strings.TrimSpace(metadata.EventPayload.BridgeInstanceID) == strings.TrimSpace(req.BridgeInstanceID)
}

func (c *slackBotClient) call(ctx context.Context, method string, payload any, result any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if c == nil {
		return errors.New("slack: api client is required")
	}
	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("slack: marshal %s payload: %w", method, err)
	}
	endpoint := strings.TrimRight(strings.TrimSpace(c.baseURL), "/") + "/" + strings.TrimSpace(method)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("slack: build %s request: %w", method, err)
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(c.botToken))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("slack: read %s response: %w", method, err)
	}

	var envelope slackAPIEnvelope
	if len(bytes.TrimSpace(responseBody)) > 0 {
		if err := json.Unmarshal(responseBody, &envelope); err != nil {
			return fmt.Errorf("slack: decode %s response: %w", method, err)
		}
	}

	retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
	if resp.StatusCode == http.StatusTooManyRequests ||
		strings.EqualFold(strings.TrimSpace(envelope.Error), "ratelimited") {
		return &bridgesdk.RateLimitError{
			Err:        fmt.Errorf("slack api %s rate limited", strings.TrimSpace(method)),
			RetryAfter: retryAfter,
		}
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return classifySlackAPIError(resp.StatusCode, envelope.Error, retryAfter)
	}
	if !envelope.OK {
		return classifySlackAPIError(resp.StatusCode, envelope.Error, retryAfter)
	}
	if result != nil && len(bytes.TrimSpace(responseBody)) > 0 {
		if err := json.Unmarshal(responseBody, result); err != nil {
			return fmt.Errorf("slack: decode %s result: %w", method, err)
		}
	}
	return nil
}

func classifySlackAPIError(status int, errorText string, retryAfter time.Duration) error {
	trimmed := strings.TrimSpace(errorText)
	lowered := strings.ToLower(trimmed)
	switch {
	case status == http.StatusTooManyRequests || lowered == "ratelimited":
		return &bridgesdk.RateLimitError{
			Err:        fmt.Errorf("slack api rate limited: %s", firstNonEmpty(trimmed, "ratelimited")),
			RetryAfter: retryAfter,
		}
	case status == http.StatusUnauthorized, status == http.StatusForbidden,
		lowered == "invalid_auth", lowered == "not_authed", lowered == "account_inactive",
		lowered == "token_revoked", lowered == "missing_scope":
		return &bridgesdk.AuthError{Err: fmt.Errorf("slack api auth failed: %s", firstNonEmpty(trimmed, "auth failed"))}
	case status == http.StatusGatewayTimeout, status == http.StatusRequestTimeout, lowered == "request_timeout":
		return &bridgesdk.HTTPError{
			StatusCode: http.StatusGatewayTimeout,
			Message:    fmt.Sprintf("slack api timeout: %s", firstNonEmpty(trimmed, "request_timeout")),
		}
	case status >= http.StatusInternalServerError,
		lowered == "internal_error",
		lowered == "fatal_error",
		lowered == "service_unavailable":
		return &bridgesdk.TransientError{
			Err: fmt.Errorf("slack api transient failure: %s", firstNonEmpty(trimmed, "service unavailable")),
		}
	case trimmed != "":
		return &bridgesdk.PermanentError{Err: fmt.Errorf("slack api error: %s", trimmed)}
	default:
		return &bridgesdk.HTTPError{
			StatusCode: maxInt(status, http.StatusInternalServerError),
			Message: fmt.Sprintf(
				"slack api request failed with status %d",
				maxInt(status, http.StatusInternalServerError),
			),
		}
	}
}

func isIgnoredSlackMessageEvent(event slackMessageEvent) bool {
	if strings.TrimSpace(event.User) == "" {
		return true
	}
	if strings.TrimSpace(event.BotID) != "" {
		return true
	}
	ignoredSubtypes := map[string]struct{}{
		"bot_message":     {},
		"message_changed": {},
		"message_deleted": {},
		"message_replied": {},
		"channel_join":    {},
		"channel_leave":   {},
		"channel_topic":   {},
		"channel_purpose": {},
		"channel_name":    {},
		"group_join":      {},
		"group_leave":     {},
	}
	_, ignored := ignoredSubtypes[strings.TrimSpace(event.Subtype)]
	return ignored
}

func normalizeSlackAttachments(files []slackFile) []bridgepkg.MessageAttachment {
	if len(files) == 0 {
		return nil
	}
	attachments := make([]bridgepkg.MessageAttachment, 0, len(files))
	for _, file := range files {
		attachments = append(attachments, bridgepkg.MessageAttachment{
			ID:       strings.TrimSpace(file.ID),
			Name:     strings.TrimSpace(file.Name),
			MIMEType: strings.TrimSpace(file.MIMEType),
			URL:      strings.TrimSpace(file.URLPrivate),
		})
	}
	return attachments
}

func isSlackDirectConversation(channelType string, channelID string) bool {
	if strings.EqualFold(strings.TrimSpace(channelType), "im") {
		return true
	}
	return strings.HasPrefix(strings.TrimSpace(channelID), "D")
}

func isSlackSlashCommandDirect(channelName string, channelID string) bool {
	if strings.EqualFold(strings.TrimSpace(channelName), "directmessage") {
		return true
	}
	return isSlackDirectConversation("", channelID)
}

func inboundSlackThreadID(_ bool, ts string, threadTS string) string {
	return firstNonEmpty(strings.TrimSpace(threadTS), strings.TrimSpace(ts))
}

func slackDirectRootThreadID(channelID string) string {
	return strings.TrimSpace(channelID)
}

func parseSlackTimestamp(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, errors.New("slack: timestamp is required")
	}
	parts := strings.SplitN(trimmed, ".", 2)
	seconds, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	nanos := int64(0)
	if len(parts) == 2 && parts[1] != "" {
		fraction := parts[1]
		if len(fraction) > 9 {
			fraction = fraction[:9]
		}
		for len(fraction) < 9 {
			fraction += "0"
		}
		nanos, err = strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return time.Time{}, err
		}
	}
	return time.Unix(seconds, nanos).UTC(), nil
}

func normalizeSlackEmoji(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	return ":" + strings.Trim(trimmed, ":") + ":"
}

func normalizeSlackUserID(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func normalizeUsername(value string) string {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.TrimPrefix(trimmed, "@")
	return strings.ToLower(trimmed)
}

func buildSlackIDSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	ids := make(map[string]struct{}, len(values))
	for _, value := range values {
		if normalized := normalizeSlackUserID(value); normalized != "" {
			ids[normalized] = struct{}{}
		}
	}
	if len(ids) == 0 {
		return nil
	}
	return ids
}

func buildSlackUsernameSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	names := make(map[string]struct{}, len(values))
	for _, value := range values {
		if normalized := normalizeUsername(value); normalized != "" {
			names[normalized] = struct{}{}
		}
	}
	if len(names) == 0 {
		return nil
	}
	return names
}

func managedInstancesToInstances(items []subprocess.InitializeBridgeManagedInstance) []bridgepkg.BridgeInstance {
	if len(items) == 0 {
		return nil
	}
	instances := make([]bridgepkg.BridgeInstance, 0, len(items))
	for _, item := range items {
		instances = append(instances, item.Instance)
	}
	return instances
}

func deliveryStateKey(instanceID string, deliveryID string) string {
	return strings.TrimSpace(instanceID) + ":" + strings.TrimSpace(deliveryID)
}

func encodeRemoteMessageID(channelID string, ts string) string {
	return strings.TrimSpace(channelID) + ":" + strings.TrimSpace(ts)
}

func decodeRemoteMessageID(value string) (string, string, error) {
	trimmed := strings.TrimSpace(value)
	parts := strings.SplitN(trimmed, ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("slack: invalid remote message id %q", value)
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

func referenceRemoteMessageID(reference *bridgepkg.DeliveryMessageReference) string {
	if reference == nil {
		return ""
	}
	return strings.TrimSpace(reference.RemoteMessageID)
}

func parseRetryAfter(value string) time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func writeWebhookOK(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("OK"))
	return err
}

func normalizeWebhookPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	return strings.TrimRight(trimmed, "/")
}

func normalizeURL(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	return strings.TrimRight(trimmed, "/")
}

func normalizeDeliveryEventType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isNotInitializedRPCError(err error) bool {
	var rpcErr *subprocess.RPCError
	if !errors.As(err, &rpcErr) {
		return false
	}
	if rpcErr == nil {
		return false
	}
	return rpcErr.Code == rpcCodeNotInitialized
}

func cloneDegradation(degradation *bridgepkg.BridgeDegradation) *bridgepkg.BridgeDegradation {
	if degradation == nil {
		return nil
	}
	cloned := *degradation
	return &cloned
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func maxInt(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
