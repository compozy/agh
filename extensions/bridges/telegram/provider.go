package main

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
	telegramListenAddrEnv = "AGH_BRIDGE_TELEGRAM_LISTEN_ADDR"
	telegramAPIBaseEnv    = "AGH_BRIDGE_TELEGRAM_API_BASE_URL"

	telegramDefaultAPIBaseURL = "https://api.telegram.org"
	telegramGeneralTopicID    = "1"

	rpcCodeNotInitialized = -32003
)

type telegramProvider struct {
	sdk     *bridgesdk.Runtime
	stderr  io.Writer
	env     markerEnv
	now     func() time.Time
	session *bridgesdk.Session

	mu             sync.RWMutex
	lastError      string
	server         *http.Server
	listener       net.Listener
	serverAddr     string
	listenAddr     string
	routes         map[string]resolvedInstanceConfig
	deliveries     map[string]deliveryState
	reportedStatus map[string]bridgepkg.BridgeStatus
	apiFactory     func(resolvedInstanceConfig) telegramAPI

	stopCh   chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

type deliveryState struct {
	LastSeq                int64
	LastContent            string
	RemoteMessageID        string
	ReplaceRemoteMessageID string
}

type telegramProviderConfig struct {
	APIBaseURL string `json:"api_base_url,omitempty"`
	Webhook    struct {
		ListenAddr string `json:"listen_addr,omitempty"`
		Path       string `json:"path,omitempty"`
	} `json:"webhook,omitempty"`
	Batching struct {
		DelayMS        int `json:"delay_ms,omitempty"`
		SplitDelayMS   int `json:"split_delay_ms,omitempty"`
		SplitThreshold int `json:"split_threshold,omitempty"`
	} `json:"batching,omitempty"`
	DM struct {
		AllowUserIDs    []string `json:"allow_user_ids,omitempty"`
		AllowUsernames  []string `json:"allow_usernames,omitempty"`
		PairedUserIDs   []string `json:"paired_user_ids,omitempty"`
		PairedUsernames []string `json:"paired_usernames,omitempty"`
	} `json:"dm,omitempty"`
}

type resolvedInstanceConfig struct {
	managed            subprocess.InitializeBridgeManagedInstance
	instanceID         string
	listenAddr         string
	webhookPath        string
	apiBaseURL         string
	botToken           string
	webhookSecret      string
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

type telegramUpdate struct {
	UpdateID          int64            `json:"update_id"`
	Message           *telegramMessage `json:"message,omitempty"`
	EditedMessage     *telegramMessage `json:"edited_message,omitempty"`
	ChannelPost       *telegramMessage `json:"channel_post,omitempty"`
	EditedChannelPost *telegramMessage `json:"edited_channel_post,omitempty"`
}

type telegramMessage struct {
	MessageID       int64        `json:"message_id"`
	MessageThreadID int64        `json:"message_thread_id,omitempty"`
	Date            int64        `json:"date"`
	Chat            telegramChat `json:"chat"`
	From            telegramUser `json:"from"`
	Text            string       `json:"text,omitempty"`
	Caption         string       `json:"caption,omitempty"`
}

type telegramChat struct {
	ID      int64  `json:"id"`
	Type    string `json:"type,omitempty"`
	Title   string `json:"title,omitempty"`
	IsForum bool   `json:"is_forum,omitempty"`
}

type telegramUser struct {
	ID        int64  `json:"id"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

type telegramBotIdentity struct {
	ID       int64  `json:"id"`
	Username string `json:"username,omitempty"`
}

type telegramSentMessage struct {
	MessageID int64 `json:"message_id"`
}

type telegramAPIEnvelope[T any] struct {
	OK          bool                    `json:"ok"`
	Result      T                       `json:"result"`
	Description string                  `json:"description,omitempty"`
	ErrorCode   int                     `json:"error_code,omitempty"`
	Parameters  telegramAPIErrorDetails `json:"parameters,omitempty"`
}

type telegramAPIErrorDetails struct {
	RetryAfter int `json:"retry_after,omitempty"`
}

type telegramSendMessageRequest struct {
	ChatID          string `json:"chat_id"`
	Text            string `json:"text"`
	MessageThreadID int64  `json:"message_thread_id,omitempty"`
}

type telegramEditMessageTextRequest struct {
	ChatID    string `json:"chat_id"`
	MessageID int64  `json:"message_id"`
	Text      string `json:"text"`
}

type telegramDeleteMessageRequest struct {
	ChatID    string `json:"chat_id"`
	MessageID int64  `json:"message_id"`
}

type telegramAPI interface {
	GetMe(context.Context) (*telegramBotIdentity, error)
	SendMessage(context.Context, telegramSendMessageRequest) (*telegramSentMessage, error)
	EditMessageText(context.Context, telegramEditMessageTextRequest) error
	DeleteMessage(context.Context, telegramDeleteMessageRequest) error
}

type telegramBotClient struct {
	baseURL    string
	botToken   string
	httpClient *http.Client
}

func newTelegramProvider(stderr io.Writer) (*telegramProvider, error) {
	if stderr == nil {
		stderr = io.Discard
	}

	provider := &telegramProvider{
		stderr:         stderr,
		env:            markerEnvFromProcess(),
		now:            func() time.Time { return time.Now().UTC() },
		routes:         make(map[string]resolvedInstanceConfig),
		deliveries:     make(map[string]deliveryState),
		reportedStatus: make(map[string]bridgepkg.BridgeStatus),
		stopCh:         make(chan struct{}),
	}
	provider.apiFactory = func(cfg resolvedInstanceConfig) telegramAPI {
		return &telegramBotClient{
			baseURL:  cfg.apiBaseURL,
			botToken: cfg.botToken,
			httpClient: &http.Client{
				Timeout: 10 * time.Second,
			},
		}
	}

	sdkRuntime, err := bridgesdk.NewRuntime(bridgesdk.RuntimeConfig{
		ExtensionInfo: subprocess.InitializeExtensionInfo{
			Name:    "telegram",
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

func (p *telegramProvider) serve(stdin io.Reader, stdout io.Writer) error {
	p.reportSideEffectError("write start marker", appendMarkerLine(p.env.startsPath, fmt.Sprintf("pid=%d", os.Getpid())))
	return p.sdk.Serve(context.Background(), stdin, stdout)
}

func (p *telegramProvider) handleInitialize(_ context.Context, session *bridgesdk.Session) error {
	p.mu.Lock()
	p.session = session
	p.mu.Unlock()

	marker := initializeMarker{
		Request:  session.InitializeRequest(),
		Response: session.InitializeResponse(),
	}
	p.reportSideEffectError("write initialize marker", writeJSONFile(p.env.handshakePath, marker))
	p.clearLastError()

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.afterInitialize(session)
	}()

	return nil
}

func (p *telegramProvider) afterInitialize(session *bridgesdk.Session) {
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

	if p.stopped() {
		return
	}

	configs, reconcileErr := p.reconcileInstanceConfigs(ctx, session, listed)
	if reconcileErr != nil && ownershipErr == nil {
		ownershipErr = reconcileErr
	}
	for _, cfg := range configs {
		if p.stopped() {
			return
		}
		status := cfg.initialStatus
		degradation := cfg.initialDegradation
		if status == "" {
			status = bridgepkg.BridgeStatusReady
		}
		if _, err := p.reportState(ctx, session, cfg.instanceID, status, degradation); err != nil && ownershipErr == nil {
			ownershipErr = err
		}
	}

	if ownershipErr != nil {
		p.setLastError(ownershipErr)
	} else {
		p.clearLastError()
	}
}

func (p *telegramProvider) handleBridgesDeliver(
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
	ack, state, err := executeDelivery(ctx, api, cfg, request, p.deliveryState(cfg.instanceID, request.Event.DeliveryID))
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

func (p *telegramProvider) healthCheck() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if strings.TrimSpace(p.lastError) == "" {
		return nil
	}
	return errors.New(strings.TrimSpace(p.lastError))
}

func (p *telegramProvider) handleShutdown(
	_ context.Context,
	_ *bridgesdk.Session,
	request subprocess.ShutdownRequest,
) error {
	p.stop()

	shutdownCtx := context.Background()
	if request.DeadlineMS > 0 {
		var cancel context.CancelFunc
		shutdownCtx, cancel = context.WithTimeout(context.Background(), time.Duration(request.DeadlineMS)*time.Millisecond)
		defer cancel()
	}

	p.mu.Lock()
	server := p.server
	listener := p.listener
	p.server = nil
	p.listener = nil
	p.serverAddr = ""
	p.mu.Unlock()
	if listener != nil {
		_ = listener.Close()
	}
	if server != nil {
		_ = server.Shutdown(shutdownCtx)
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

	p.reportSideEffectError("write shutdown marker", appendMarkerLine(p.env.shutdownPath, fmt.Sprintf("pid=%d", os.Getpid())))
	return nil
}

func (p *telegramProvider) stop() {
	p.stopOnce.Do(func() {
		close(p.stopCh)
		p.mu.Lock()
		defer p.mu.Unlock()
		for id, cfg := range p.routes {
			if cfg.batcher != nil {
				cfg.batcher.Close()
				cfg.batcher = nil
				p.routes[id] = cfg
			}
		}
	})
}

func (p *telegramProvider) syncOwnedInstances(
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

func (p *telegramProvider) getOwnedInstance(
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

func (p *telegramProvider) reportState(
	ctx context.Context,
	session *bridgesdk.Session,
	bridgeInstanceID string,
	status bridgepkg.BridgeStatus,
	degradation *bridgepkg.BridgeDegradation,
) (*bridgepkg.BridgeInstance, error) {
	var result *bridgepkg.BridgeInstance
	err := p.retryHostCall(ctx, func(callCtx context.Context) error {
		instance, callErr := session.HostAPI().ReportBridgeInstanceState(callCtx, extensioncontract.BridgesInstancesReportStateParams{
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
		return nil, err
	}

	p.mu.Lock()
	p.reportedStatus[strings.TrimSpace(bridgeInstanceID)] = result.Status.Normalize()
	p.mu.Unlock()
	p.reportSideEffectError("write state marker", appendJSONLine(p.env.statePath, stateMarker{
		BridgeInstanceID: result.ID,
		Status:           result.Status,
		Instance:         *result,
	}))
	return result, nil
}

func (p *telegramProvider) reportReadyIfNeeded(ctx context.Context, session *bridgesdk.Session, bridgeInstanceID string) {
	p.mu.RLock()
	status := p.reportedStatus[strings.TrimSpace(bridgeInstanceID)]
	p.mu.RUnlock()
	if status == bridgepkg.BridgeStatusReady {
		return
	}
	_, _ = p.reportState(ctx, session, bridgeInstanceID, bridgepkg.BridgeStatusReady, nil)
}

func (p *telegramProvider) ingestBridgeMessage(
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

func (p *telegramProvider) retryHostCall(ctx context.Context, fn func(context.Context) error) error {
	if ctx == nil {
		ctx = context.Background()
	}

	delay := 10 * time.Millisecond
	var lastErr error
	for attempt := 0; attempt < 6; attempt++ {
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

func (p *telegramProvider) stopped() bool {
	select {
	case <-p.stopCh:
		return true
	default:
		return false
	}
}

func (p *telegramProvider) reconcileInstanceConfigs(
	ctx context.Context,
	session *bridgesdk.Session,
	managed []subprocess.InitializeBridgeManagedInstance,
) ([]resolvedInstanceConfig, error) {
	if len(managed) == 0 {
		p.mu.Lock()
		p.routes = make(map[string]resolvedInstanceConfig)
		p.mu.Unlock()
		return nil, nil
	}

	configs := make([]resolvedInstanceConfig, 0, len(managed))
	requestedListen := strings.TrimSpace(os.Getenv(telegramListenAddrEnv))
	usedPaths := make(map[string]string, len(managed))

	for _, item := range managed {
		cfg := p.resolveInstanceConfig(session, item)
		if cfg.listenAddr != "" {
			if requestedListen == "" {
				requestedListen = cfg.listenAddr
			} else if requestedListen != cfg.listenAddr && cfg.configError == nil {
				cfg.configError = fmt.Errorf("telegram: instance %q configured incompatible listen_addr %q (runtime uses %q)", cfg.instanceID, cfg.listenAddr, requestedListen)
			}
		}
		if owner, ok := usedPaths[cfg.webhookPath]; ok && cfg.webhookPath != "" && cfg.configError == nil {
			cfg.configError = fmt.Errorf("telegram: webhook path %q is shared by %q and %q", cfg.webhookPath, owner, cfg.instanceID)
		}
		if cfg.webhookPath != "" {
			usedPaths[cfg.webhookPath] = cfg.instanceID
		}
		configs = append(configs, cfg)
	}

	if p.stopped() {
		for idx := range configs {
			if configs[idx].batcher != nil {
				configs[idx].batcher.Close()
				configs[idx].batcher = nil
			}
		}
		return nil, nil
	}

	if requestedListen == "" {
		for idx := range configs {
			if configs[idx].configError == nil {
				configs[idx].configError = errors.New("telegram: webhook listen address is required")
			}
		}
	} else if err := p.startServer(requestedListen); err != nil {
		for idx := range configs {
			if configs[idx].configError == nil {
				configs[idx].configError = err
			}
		}
	}

	nextRoutes := make(map[string]resolvedInstanceConfig, len(configs))
	p.mu.Lock()
	existing := p.routes
	for _, cfg := range configs {
		if prior, ok := existing[cfg.instanceID]; ok && prior.batcher != nil && cfg.batcher == nil {
			prior.batcher.Close()
		}
		nextRoutes[cfg.instanceID] = cfg
	}
	for instanceID, prior := range existing {
		if _, ok := nextRoutes[instanceID]; ok {
			continue
		}
		if prior.batcher != nil {
			prior.batcher.Close()
		}
	}
	p.routes = nextRoutes
	p.listenAddr = requestedListen
	p.mu.Unlock()

	for idx := range configs {
		status, degradation, err := p.determineInitialState(ctx, configs[idx])
		if err != nil {
			p.setLastError(err)
		}
		configs[idx].initialStatus = status
		configs[idx].initialDegradation = degradation
	}

	return configs, nil
}

func (p *telegramProvider) resolveInstanceConfig(
	session *bridgesdk.Session,
	managed subprocess.InitializeBridgeManagedInstance,
) resolvedInstanceConfig {
	cfg := telegramProviderConfig{}
	if len(managed.Instance.ProviderConfig) > 0 {
		if err := json.Unmarshal(managed.Instance.ProviderConfig, &cfg); err != nil {
			return resolvedInstanceConfig{
				managed:    managed,
				instanceID: managed.Instance.ID,
				configError: fmt.Errorf(
					"telegram: decode provider_config for %q: %w",
					managed.Instance.ID,
					err,
				),
			}
		}
	}

	botToken, _ := session.Cache().BoundSecretValue(managed.Instance.ID, "bot_token")
	webhookSecret, _ := session.Cache().BoundSecretValue(managed.Instance.ID, "webhook_secret")
	listenAddr := firstNonEmpty(cfg.Webhook.ListenAddr, strings.TrimSpace(os.Getenv(telegramListenAddrEnv)))
	webhookPath := normalizeWebhookPath(firstNonEmpty(cfg.Webhook.Path, "/telegram/"+strings.TrimSpace(managed.Instance.ID)))
	apiBaseURL := normalizeURL(firstNonEmpty(cfg.APIBaseURL, strings.TrimSpace(os.Getenv(telegramAPIBaseEnv)), telegramDefaultAPIBaseURL))

	resolved := resolvedInstanceConfig{
		managed:         managed,
		instanceID:      strings.TrimSpace(managed.Instance.ID),
		listenAddr:      listenAddr,
		webhookPath:     webhookPath,
		apiBaseURL:      apiBaseURL,
		botToken:        strings.TrimSpace(botToken),
		webhookSecret:   strings.TrimSpace(webhookSecret),
		dmPolicy:        managed.Instance.DMPolicy.Normalize(),
		allowUserIDs:    buildIdentitySet(cfg.DM.AllowUserIDs),
		allowUsernames:  buildIdentitySet(cfg.DM.AllowUsernames),
		pairedUserIDs:   buildIdentitySet(cfg.DM.PairedUserIDs),
		pairedUsernames: buildIdentitySet(cfg.DM.PairedUsernames),
		dedup:           bridgesdk.NewDedupCache(5*time.Minute, 2000),
		rateLimiter:     bridgesdk.NewFixedWindowRateLimiter(100, time.Minute),
		inFlightLimiter: bridgesdk.NewInFlightLimiter(16),
	}

	if resolved.dmPolicy == "" {
		resolved.dmPolicy = bridgepkg.BridgeDMPolicyOpen
	}
	if resolved.webhookPath == "" {
		resolved.configError = errors.New("telegram: webhook path is required")
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

func (p *telegramProvider) determineInitialState(
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
		err := errors.New("telegram: bot_token secret binding is required")
		return bridgepkg.BridgeStatusAuthRequired, &bridgepkg.BridgeDegradation{
			Reason:  bridgepkg.BridgeDegradationReasonAuthFailed,
			Message: err.Error(),
		}, err
	}
	_, err := p.apiFactory(cfg).GetMe(ctx)
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

func (p *telegramProvider) startServer(listenAddr string) error {
	p.mu.RLock()
	server := p.server
	currentListen := p.listenAddr
	p.mu.RUnlock()
	if server != nil {
		if currentListen != "" && currentListen != strings.TrimSpace(listenAddr) {
			return fmt.Errorf("telegram: runtime already listening on %q, cannot switch to %q", currentListen, listenAddr)
		}
		return nil
	}

	ln, err := net.Listen("tcp", strings.TrimSpace(listenAddr))
	if err != nil {
		return fmt.Errorf("telegram: listen %q: %w", listenAddr, err)
	}
	if p.stopped() {
		_ = ln.Close()
		return errors.New("telegram: runtime is stopping")
	}

	httpServer := &http.Server{
		Handler: http.HandlerFunc(p.serveWebhookHTTP),
	}

	actualAddr := ln.Addr().String()
	p.mu.Lock()
	p.server = httpServer
	p.listener = ln
	p.serverAddr = actualAddr
	p.listenAddr = strings.TrimSpace(listenAddr)
	p.mu.Unlock()

	p.reportSideEffectError("write start marker", appendMarkerLine(p.env.startsPath, fmt.Sprintf("listen=%s", actualAddr)))

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		if serveErr := httpServer.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			p.setLastError(serveErr)
		}
		p.mu.Lock()
		if p.server == httpServer {
			p.server = nil
			p.listener = nil
			p.serverAddr = ""
		}
		p.mu.Unlock()
	}()

	return nil
}

func (p *telegramProvider) serveWebhookHTTP(w http.ResponseWriter, r *http.Request) {
	cfg, ok := p.configForPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	handler, err := bridgesdk.NewWebhookHandler(bridgesdk.WebhookGuardConfig{
		AllowedMethods:      []string{http.MethodPost},
		AllowedContentTypes: []string{"application/json"},
		MaxBodyBytes:        1 << 20,
		RateLimiter:         cfg.rateLimiter,
		InFlightLimiter:     cfg.inFlightLimiter,
		VerifySignature: func(ctx context.Context, req *http.Request, body []byte) error {
			return verifyWebhookSecret(ctx, req, body, cfg.webhookSecret)
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

func (p *telegramProvider) handleWebhookRequest(
	w http.ResponseWriter,
	_ *http.Request,
	cfg resolvedInstanceConfig,
	request bridgesdk.WebhookRequest,
) error {
	update := telegramUpdate{}
	if err := json.Unmarshal(request.Body, &update); err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: "invalid telegram webhook payload"}
	}
	message := selectTelegramMessage(update)
	if message == nil {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
		return nil
	}

	envelope, err := mapTelegramUpdate(update, cfg.managed, request.ReceivedAt)
	if err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
	}
	if cfg.dedup.Mark(envelope.IdempotencyKey) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
		return nil
	}
	if !allowDirectMessage(cfg, *message) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
		return nil
	}

	if cfg.batcher != nil {
		if err := cfg.batcher.Enqueue(envelope); err != nil {
			return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
		}
	} else {
		if err := p.dispatchInboundEnvelope(context.Background(), cfg.instanceID, envelope); err != nil {
			return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
		}
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
	return nil
}

func (p *telegramProvider) dispatchInboundBatch(ctx context.Context, bridgeInstanceID string, batch bridgesdk.InboundBatch) error {
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

func (p *telegramProvider) dispatchInboundEnvelope(ctx context.Context, bridgeInstanceID string, envelope bridgepkg.InboundMessageEnvelope) error {
	session := p.currentSession()
	if session == nil {
		return errors.New("telegram: runtime session is not initialized")
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

func (p *telegramProvider) configForInstance(instanceID string) (resolvedInstanceConfig, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	cfg, ok := p.routes[strings.TrimSpace(instanceID)]
	if !ok {
		return resolvedInstanceConfig{}, fmt.Errorf("telegram: delivery targeted unmanaged instance %q", instanceID)
	}
	return cfg, nil
}

func (p *telegramProvider) waitForInstanceConfig(instanceID string, timeout time.Duration) (resolvedInstanceConfig, error) {
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

func (p *telegramProvider) configForPath(path string) (resolvedInstanceConfig, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, cfg := range p.routes {
		if cfg.webhookPath == normalizeWebhookPath(path) {
			return cfg, true
		}
	}
	return resolvedInstanceConfig{}, false
}

func (p *telegramProvider) currentSession() *bridgesdk.Session {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.session
}

func (p *telegramProvider) deliveryState(instanceID string, deliveryID string) deliveryState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.deliveries[deliveryStateKey(instanceID, deliveryID)]
}

func (p *telegramProvider) storeDeliveryState(instanceID string, deliveryID string, state deliveryState) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.deliveries[deliveryStateKey(instanceID, deliveryID)] = state
}

func (p *telegramProvider) setLastError(err error) {
	if err == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastError = err.Error()
}

func (p *telegramProvider) clearLastError() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastError = ""
}

func (p *telegramProvider) reportSideEffectError(action string, err error) {
	reportSideEffectError(p.stderr, action, err)
}

func executeDelivery(
	ctx context.Context,
	api telegramAPI,
	cfg resolvedInstanceConfig,
	request bridgepkg.DeliveryRequest,
	state deliveryState,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	if err := request.Validate(); err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}

	event := request.Event
	if event.EventType != bridgepkg.DeliveryEventTypeResume && event.Seq <= state.LastSeq {
		return bridgepkg.DeliveryAck{}, state, fmt.Errorf(
			"telegram: out-of-order delivery seq %d after %d",
			event.Seq,
			state.LastSeq,
		)
	}
	if event.EventType == bridgepkg.DeliveryEventTypeResume && request.Snapshot != nil {
		state.LastSeq = request.Snapshot.LastAckedSeq
		state.LastContent = request.Snapshot.CurrentContent.Text
		state.RemoteMessageID = strings.TrimSpace(request.Snapshot.RemoteMessageID)
		state.ReplaceRemoteMessageID = strings.TrimSpace(request.Snapshot.ReplaceRemoteMessageID)
	}

	targetChatID, targetThreadID, err := resolveDeliveryTarget(event)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}

	switch {
	case event.Operation.Normalize() == bridgepkg.DeliveryOperationDelete || normalizeDeliveryEventType(event.EventType) == bridgepkg.DeliveryEventTypeDelete:
		remoteID := firstNonEmpty(
			referenceRemoteMessageID(event.Reference),
			state.RemoteMessageID,
		)
		if remoteID == "" && request.Snapshot != nil {
			remoteID = strings.TrimSpace(request.Snapshot.RemoteMessageID)
		}
		if remoteID == "" {
			return bridgepkg.DeliveryAck{}, state, errors.New("telegram: delete delivery requires a remote message id")
		}
		chatID, messageID, err := decodeRemoteMessageID(remoteID)
		if err != nil {
			return bridgepkg.DeliveryAck{}, state, err
		}
		if err := api.DeleteMessage(ctx, telegramDeleteMessageRequest{
			ChatID:    chatID,
			MessageID: messageID,
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
		state.LastContent = ""
		state.RemoteMessageID = remoteID
		state.ReplaceRemoteMessageID = ack.ReplaceRemoteMessageID
		return ack, state, ack.ValidateFor(event)
	case shouldPostNewMessage(event, state, request):
		sent, err := api.SendMessage(ctx, telegramSendMessageRequest{
			ChatID:          targetChatID,
			Text:            event.Content.Text,
			MessageThreadID: resolveTelegramThreadID(targetThreadID, targetChatID),
		})
		if err != nil {
			return bridgepkg.DeliveryAck{}, state, err
		}
		remoteID := encodeRemoteMessageID(targetChatID, sent.MessageID)
		ack := bridgepkg.DeliveryAck{
			DeliveryID:      event.DeliveryID,
			Seq:             event.Seq,
			RemoteMessageID: remoteID,
		}
		state.LastSeq = event.Seq
		state.LastContent = event.Content.Text
		state.ReplaceRemoteMessageID = state.RemoteMessageID
		state.RemoteMessageID = remoteID
		if state.ReplaceRemoteMessageID != "" {
			ack.ReplaceRemoteMessageID = state.ReplaceRemoteMessageID
		}
		return ack, state, ack.ValidateFor(event)
	default:
		remoteID := state.RemoteMessageID
		if remoteID == "" && request.Snapshot != nil {
			remoteID = strings.TrimSpace(request.Snapshot.RemoteMessageID)
		}
		if remoteID == "" {
			return bridgepkg.DeliveryAck{}, state, errors.New("telegram: edit delivery requires a remote message id")
		}
		if event.Content.Text == state.LastContent {
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
		chatID, messageID, err := decodeRemoteMessageID(remoteID)
		if err != nil {
			return bridgepkg.DeliveryAck{}, state, err
		}
		if err := api.EditMessageText(ctx, telegramEditMessageTextRequest{
			ChatID:    chatID,
			MessageID: messageID,
			Text:      event.Content.Text,
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
		state.LastContent = event.Content.Text
		state.RemoteMessageID = remoteID
		state.ReplaceRemoteMessageID = ack.ReplaceRemoteMessageID
		return ack, state, ack.ValidateFor(event)
	}
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

func allowDirectMessage(cfg resolvedInstanceConfig, message telegramMessage) bool {
	if !isDirectChat(message.Chat.Type) {
		return true
	}

	switch cfg.dmPolicy.Normalize() {
	case "", bridgepkg.BridgeDMPolicyOpen:
		return true
	case bridgepkg.BridgeDMPolicyAllowlist:
		return identityAllowed(cfg.allowUserIDs, cfg.allowUsernames, message.From)
	case bridgepkg.BridgeDMPolicyPairing:
		if identityAllowed(cfg.pairedUserIDs, cfg.pairedUsernames, message.From) {
			return true
		}
		return identityAllowed(cfg.allowUserIDs, cfg.allowUsernames, message.From)
	default:
		return false
	}
}

func identityAllowed(ids map[string]struct{}, usernames map[string]struct{}, user telegramUser) bool {
	if len(ids) == 0 && len(usernames) == 0 {
		return false
	}
	if _, ok := ids[strings.TrimSpace(strconv.FormatInt(user.ID, 10))]; ok {
		return true
	}
	if _, ok := usernames[normalizeUsername(user.Username)]; ok {
		return true
	}
	return false
}

func mapTelegramUpdate(
	update telegramUpdate,
	managed subprocess.InitializeBridgeManagedInstance,
	receivedAt time.Time,
) (bridgepkg.InboundMessageEnvelope, error) {
	message := selectTelegramMessage(update)
	if message == nil {
		return bridgepkg.InboundMessageEnvelope{}, errors.New("telegram: message update is required")
	}
	if receivedAt.IsZero() {
		receivedAt = time.Now().UTC()
	}
	if message.Date > 0 {
		receivedAt = time.Unix(message.Date, 0).UTC()
	}

	text := strings.TrimSpace(message.Text)
	if text == "" {
		text = strings.TrimSpace(message.Caption)
	}

	senderName := strings.TrimSpace(strings.Join([]string{strings.TrimSpace(message.From.FirstName), strings.TrimSpace(message.From.LastName)}, " "))
	threadID := inboundThreadID(message.Chat, message.MessageThreadID)
	envelope := bridgepkg.InboundMessageEnvelope{
		BridgeInstanceID:  managed.Instance.ID,
		Scope:             managed.Instance.Scope,
		WorkspaceID:       managed.Instance.WorkspaceID,
		PlatformMessageID: strconv.FormatInt(message.MessageID, 10),
		ReceivedAt:        receivedAt,
		Sender: bridgepkg.MessageSender{
			ID:          optionalTelegramID(message.From.ID),
			Username:    normalizeUsername(message.From.Username),
			DisplayName: senderName,
		},
		Content: bridgepkg.MessageContent{
			Text: text,
		},
		EventFamily:    bridgepkg.InboundEventFamilyMessage,
		IdempotencyKey: fmt.Sprintf("telegram:%s:%d", managed.Instance.ID, update.UpdateID),
	}

	if isDirectChat(message.Chat.Type) {
		envelope.PeerID = strconv.FormatInt(message.Chat.ID, 10)
		envelope.ThreadID = threadID
	} else {
		envelope.GroupID = strconv.FormatInt(message.Chat.ID, 10)
		envelope.ThreadID = threadID
	}

	metadata, err := json.Marshal(map[string]any{
		"chat_id":           message.Chat.ID,
		"chat_type":         strings.TrimSpace(message.Chat.Type),
		"is_forum":          message.Chat.IsForum,
		"message_id":        message.MessageID,
		"message_thread_id": message.MessageThreadID,
		"update_id":         update.UpdateID,
	})
	if err == nil {
		envelope.ProviderMetadata = metadata
	}

	return envelope, envelope.Validate()
}

func resolveDeliveryTarget(event bridgepkg.DeliveryEvent) (string, string, error) {
	chatID := firstNonEmpty(
		strings.TrimSpace(event.DeliveryTarget.PeerID),
		strings.TrimSpace(event.DeliveryTarget.GroupID),
		strings.TrimSpace(event.RoutingKey.PeerID),
		strings.TrimSpace(event.RoutingKey.GroupID),
	)
	if chatID == "" {
		return "", "", errors.New("telegram: delivery target requires peer_id or group_id")
	}
	threadID := firstNonEmpty(
		strings.TrimSpace(event.DeliveryTarget.ThreadID),
		strings.TrimSpace(event.RoutingKey.ThreadID),
	)
	return chatID, threadID, nil
}

func resolveTelegramThreadID(threadID string, chatID string) int64 {
	if strings.TrimSpace(threadID) == "" {
		return 0
	}
	if strings.TrimSpace(threadID) == telegramGeneralTopicID && strings.HasPrefix(strings.TrimSpace(chatID), "-") {
		return 0
	}
	value, err := strconv.ParseInt(strings.TrimSpace(threadID), 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func verifyWebhookSecret(_ context.Context, req *http.Request, _ []byte, secret string) error {
	trimmedSecret := strings.TrimSpace(secret)
	if trimmedSecret == "" {
		return nil
	}
	if req == nil {
		return errors.New("telegram: webhook request is required")
	}
	header := strings.TrimSpace(req.Header.Get("X-Telegram-Bot-Api-Secret-Token"))
	if header == "" || header != trimmedSecret {
		return errors.New("telegram: invalid webhook secret")
	}
	return nil
}

func (c *telegramBotClient) GetMe(ctx context.Context) (*telegramBotIdentity, error) {
	var result telegramBotIdentity
	if err := c.call(ctx, "getMe", map[string]any{}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *telegramBotClient) SendMessage(ctx context.Context, req telegramSendMessageRequest) (*telegramSentMessage, error) {
	var result telegramSentMessage
	if err := c.call(ctx, "sendMessage", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *telegramBotClient) EditMessageText(ctx context.Context, req telegramEditMessageTextRequest) error {
	var result json.RawMessage
	return c.call(ctx, "editMessageText", req, &result)
}

func (c *telegramBotClient) DeleteMessage(ctx context.Context, req telegramDeleteMessageRequest) error {
	var result bool
	return c.call(ctx, "deleteMessage", req, &result)
}

func (c *telegramBotClient) call(ctx context.Context, method string, payload any, result any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if c == nil {
		return errors.New("telegram: bot api client is required")
	}
	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("telegram: marshal %s payload: %w", method, err)
	}
	endpoint := strings.TrimRight(strings.TrimSpace(c.baseURL), "/") + "/bot" + strings.TrimSpace(c.botToken) + "/" + strings.TrimSpace(method)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram: build %s request: %w", method, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("telegram: read %s response: %w", method, err)
	}

	if result == nil {
		result = &json.RawMessage{}
	}
	response := telegramAPIEnvelope[json.RawMessage]{}
	if err := json.Unmarshal(raw, &response); err != nil {
		return &bridgesdk.HTTPError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("telegram %s returned invalid JSON: %s", method, strings.TrimSpace(string(raw))),
		}
	}
	if resp.StatusCode >= 400 || !response.OK {
		return classifyTelegramHTTPError(resp.StatusCode, response)
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(response.Result, result); err != nil {
		return fmt.Errorf("telegram: decode %s result: %w", method, err)
	}
	return nil
}

func classifyTelegramHTTPError(statusCode int, response telegramAPIEnvelope[json.RawMessage]) error {
	message := strings.TrimSpace(response.Description)
	if message == "" {
		message = fmt.Sprintf("telegram bot api error %d", maxInt(statusCode, response.ErrorCode))
	}
	retryAfter := time.Duration(response.Parameters.RetryAfter) * time.Second
	if statusCode == 0 {
		statusCode = response.ErrorCode
	}
	if statusCode == 429 {
		return &bridgesdk.RateLimitError{
			Err:        &bridgesdk.HTTPError{StatusCode: statusCode, Message: message, RetryAfter: retryAfter},
			RetryAfter: retryAfter,
		}
	}
	if statusCode == 401 || statusCode == 403 {
		return &bridgesdk.AuthError{
			Err: &bridgesdk.HTTPError{StatusCode: statusCode, Message: message},
		}
	}
	return &bridgesdk.HTTPError{
		StatusCode: statusCode,
		Message:    message,
		RetryAfter: retryAfter,
	}
}

func selectTelegramMessage(update telegramUpdate) *telegramMessage {
	switch {
	case update.Message != nil:
		return update.Message
	case update.EditedMessage != nil:
		return update.EditedMessage
	case update.ChannelPost != nil:
		return update.ChannelPost
	case update.EditedChannelPost != nil:
		return update.EditedChannelPost
	default:
		return nil
	}
}

func inboundThreadID(chat telegramChat, messageThreadID int64) string {
	if isDirectChat(chat.Type) {
		return optionalTelegramID(messageThreadID)
	}
	if chat.IsForum && messageThreadID == 0 {
		return telegramGeneralTopicID
	}
	return optionalTelegramID(messageThreadID)
}

func isDirectChat(chatType string) bool {
	return strings.EqualFold(strings.TrimSpace(chatType), "private")
}

func encodeRemoteMessageID(chatID string, messageID int64) string {
	return strings.TrimSpace(chatID) + ":" + strconv.FormatInt(messageID, 10)
}

func decodeRemoteMessageID(remoteMessageID string) (string, int64, error) {
	trimmed := strings.TrimSpace(remoteMessageID)
	index := strings.LastIndex(trimmed, ":")
	if index <= 0 || index == len(trimmed)-1 {
		return "", 0, fmt.Errorf("telegram: invalid remote message id %q", remoteMessageID)
	}
	messageID, err := strconv.ParseInt(trimmed[index+1:], 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("telegram: parse remote message id %q: %w", remoteMessageID, err)
	}
	return trimmed[:index], messageID, nil
}

func referenceRemoteMessageID(reference *bridgepkg.DeliveryMessageReference) string {
	if reference == nil {
		return ""
	}
	return strings.TrimSpace(reference.RemoteMessageID)
}

func normalizeWebhookPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	return trimmed
}

func normalizeURL(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return trimmed
	}
	return strings.TrimRight(parsed.String(), "/")
}

func buildIdentitySet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := normalizeUsername(value)
		if trimmed == "" {
			continue
		}
		result[trimmed] = struct{}{}
	}
	return result
}

func normalizeUsername(value string) string {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.TrimPrefix(trimmed, "@")
	return strings.ToLower(trimmed)
}

func managedInstancesToInstances(items []subprocess.InitializeBridgeManagedInstance) []bridgepkg.BridgeInstance {
	instances := make([]bridgepkg.BridgeInstance, 0, len(items))
	for _, item := range items {
		instances = append(instances, item.Instance)
	}
	return instances
}

func deliveryStateKey(instanceID string, deliveryID string) string {
	return strings.TrimSpace(instanceID) + ":" + strings.TrimSpace(deliveryID)
}

func optionalTelegramID(value int64) string {
	if value == 0 {
		return ""
	}
	return strconv.FormatInt(value, 10)
}

func normalizeDeliveryEventType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isNotInitializedRPCError(err error) bool {
	if err == nil {
		return false
	}
	var rpcErr *subprocess.RPCError
	if !errors.As(err, &rpcErr) {
		return false
	}
	return rpcErr.Code == rpcCodeNotInitialized || strings.EqualFold(strings.TrimSpace(rpcErr.Message), "Not initialized")
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
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func maxInt(values ...int) int {
	result := 0
	for _, value := range values {
		if value > result {
			result = value
		}
	}
	return result
}
