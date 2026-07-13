package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	bridgepkg "github.com/compozy/agh/internal/bridges/contract"
	"github.com/compozy/agh/internal/bridgesdk"
	"github.com/compozy/agh/internal/subprocess"
)

const (
	telegramProviderVersion = "0.1.0"
	providerTelegramKey     = "telegram"
)

const (
	telegramListenAddrEnv = "AGH_BRIDGE_TELEGRAM_LISTEN_ADDR"
	telegramAPIBaseEnv    = "AGH_BRIDGE_TELEGRAM_API_BASE_URL"

	telegramDefaultAPIBaseURL        = "https://api.telegram.org"
	telegramGeneralTopicID           = "1"
	telegramWebhookReadHeaderTimeout = 10 * time.Second
	telegramWebhookIdleTimeout       = 30 * time.Second
)

type telegramProvider struct {
	sdk        *bridgesdk.Runtime
	lifecycle  *bridgesdk.ProviderLifecycle
	markers    *bridgesdk.AdapterMarkers
	http       *bridgesdk.ProviderHTTPServer
	stderr     io.Writer
	now        func() time.Time
	parents    *bridgesdk.ParentMessageCache
	routes     *bridgesdk.RouteTable[resolvedInstanceConfig]
	deliveries *bridgesdk.DeliveryStateStore[deliveryState]

	mu         sync.RWMutex
	lastError  string
	listenAddr string
	apiFactory func(*resolvedInstanceConfig) telegramAPI
}

type telegramProviderConfig struct {
	Webhook struct {
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
	managed            *subprocess.InitializeBridgeManagedInstance
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
	MessageID       int64            `json:"message_id"`
	MessageThreadID int64            `json:"message_thread_id,omitempty"`
	Date            int64            `json:"date"`
	EditDate        int64            `json:"edit_date,omitempty"`
	Chat            telegramChat     `json:"chat"`
	From            telegramUser     `json:"from"`
	SenderChat      *telegramChat    `json:"sender_chat,omitempty"`
	Text            string           `json:"text,omitempty"`
	Caption         string           `json:"caption,omitempty"`
	ReplyToMessage  *telegramMessage `json:"reply_to_message,omitempty"`
}

type telegramChat struct {
	ID       int64  `json:"id"`
	Type     string `json:"type,omitempty"`
	Title    string `json:"title,omitempty"`
	Username string `json:"username,omitempty"`
	IsForum  bool   `json:"is_forum,omitempty"`
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
	Parameters  telegramAPIErrorDetails `json:"parameters"`
}

type telegramAPIErrorDetails struct {
	RetryAfter int `json:"retry_after,omitempty"`
}

type telegramSendMessageRequest struct {
	ChatID          string `json:"chat_id"`
	Text            string `json:"text"`
	MessageThreadID int64  `json:"message_thread_id,omitempty"`
	ParseMode       string `json:"parse_mode,omitempty"`
}

type telegramEditMessageTextRequest struct {
	ChatID    string `json:"chat_id"`
	MessageID int64  `json:"message_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
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
	SendChatAction(context.Context, telegramSendChatActionRequest) error
	SetMessageReaction(context.Context, telegramSetMessageReactionRequest) error
}

//nolint:funlen // Construction keeps the provider's declarative runtime wiring visible in one place.
func newTelegramProvider(stderr io.Writer) (*telegramProvider, error) {
	if stderr == nil {
		stderr = io.Discard
	}

	provider := &telegramProvider{
		stderr:  stderr,
		markers: bridgesdk.NewAdapterMarkers(providerTelegramKey, stderr),
		now:     func() time.Time { return time.Now().UTC() },
		parents: bridgesdk.NewParentMessageCache(0),
		routes: bridgesdk.NewRouteTable(func(config resolvedInstanceConfig) []string {
			if config.configError != nil {
				return nil
			}
			return []string{config.webhookPath}
		}),
		deliveries: bridgesdk.NewDeliveryStateStore[deliveryState](),
	}
	provider.apiFactory = func(cfg *resolvedInstanceConfig) telegramAPI {
		return &telegramBotClient{
			baseURL:  cfg.apiBaseURL,
			botToken: cfg.botToken,
			httpClient: &http.Client{
				Timeout: 10 * time.Second,
			},
			reportResponseCleanup: func(err error) {
				provider.markers.ReportError("clean up Telegram API response", err)
			},
		}
	}

	lifecycle, err := bridgesdk.NewProviderLifecycle(bridgesdk.ProviderLifecycleConfig{
		ProviderName: providerTelegramKey,
		Markers:      provider.markers,
		Reconcile: func(
			ctx context.Context,
			managed []subprocess.InitializeBridgeManagedInstance,
		) ([]bridgesdk.ProviderInitialState, error) {
			configs := provider.reconcileInstanceConfigs(ctx, provider.lifecycle.Session(), managed)
			states := make([]bridgesdk.ProviderInitialState, 0, len(configs))
			for idx := range configs {
				states = append(states, bridgesdk.ProviderInitialState{
					BridgeInstanceID: configs[idx].instanceID,
					Status:           configs[idx].initialStatus,
					Degradation:      configs[idx].initialDegradation,
				})
			}
			return states, nil
		},
		FinalizeInitialize: func(err error) {
			if err != nil {
				provider.setLastError(err)
			}
		},
		OnStop: provider.stopResources,
		ShutdownResources: func(ctx context.Context) error {
			if provider.http == nil {
				return nil
			}
			return provider.http.Shutdown(ctx)
		},
	})
	if err != nil {
		return nil, err
	}
	provider.lifecycle = lifecycle
	providerHTTP, err := bridgesdk.NewProviderHTTPServer(bridgesdk.ProviderHTTPConfig{
		ReadHeaderTimeout: telegramWebhookReadHeaderTimeout,
		IdleTimeout:       telegramWebhookIdleTimeout,
		Handler:           http.HandlerFunc(provider.serveWebhookHTTP),
		Go:                lifecycle.Go,
		OnError:           provider.setLastError,
	})
	if err != nil {
		return nil, err
	}
	provider.http = providerHTTP

	sdkRuntime, err := bridgesdk.NewRuntime(bridgesdk.RuntimeConfig{
		ExtensionInfo: subprocess.InitializeExtensionInfo{
			Name:    providerTelegramKey,
			Version: telegramProviderVersion,
			SDKName: "bridgesdk",
		},
		Initialize:      lifecycle.Initialize,
		Deliver:         provider.handleBridgesDeliver,
		Progress:        provider.handleBridgesProgress,
		Check:           provider.handleBridgeCheck,
		RegisterWebhook: provider.handleBridgeWebhookRegistration,
		HealthCheck:     func(context.Context, *bridgesdk.Session) error { return provider.healthCheck() },
		Shutdown:        lifecycle.Shutdown,
		Now:             func() time.Time { return provider.now() },
	})
	if err != nil {
		return nil, err
	}
	provider.sdk = sdkRuntime
	return provider, nil
}

func (p *telegramProvider) serve(stdin io.Reader, stdout io.Writer) error {
	return p.lifecycle.Serve(context.Background(), p.sdk, stdin, stdout)
}

func (p *telegramProvider) handleBridgesDeliver(
	ctx context.Context,
	session *bridgesdk.Session,
	request bridgepkg.DeliveryRequest,
) (bridgepkg.DeliveryAck, error) {
	marker := bridgesdk.DeliveryMarker{
		PID:     os.Getpid(),
		Request: request,
	}

	cfg, err := p.waitForInstanceConfig(
		strings.TrimSpace(request.Event.BridgeInstanceID),
		500*time.Millisecond,
	)
	if err != nil {
		marker.Error = err.Error()
		p.markers.RecordDelivery(marker)
		p.setLastError(err)
		return bridgepkg.DeliveryAck{}, err
	}

	if p.markers.ShouldCrashOnce() {
		p.markers.RecordDelivery(marker)
		p.markers.RecordCrash(map[string]any{
			"crashed":            true,
			"pid":                os.Getpid(),
			"delivery_id":        strings.TrimSpace(request.Event.DeliveryID),
			"bridge_instance_id": cfg.instanceID,
		})
		os.Exit(23)
	}

	ack, state, err := p.executeTextDeliveryWithProgress(ctx, &cfg, request)
	if err != nil {
		if bridgesdk.IsCommittedMutation(err) {
			p.deliveries.Delete(deliveryStateKey(cfg.instanceID, request.Event.DeliveryID))
			closeProgressDispatcher(state.Progress)
		} else {
			p.storeDeliveryRetryState(cfg.instanceID, request.Event.DeliveryID, state)
		}
		marker.Error = err.Error()
		p.markers.RecordDelivery(marker)
		classified := bridgesdk.ClassifyError(err)
		_, _, reportErr := session.ReportClassifiedError(ctx, cfg.instanceID, classified)
		if reportErr != nil {
			p.setLastError(reportErr)
		} else {
			p.setLastError(err)
		}
		return bridgepkg.DeliveryAck{}, err
	}

	progressCleanupErr := p.completeTextDeliveryProgress(ctx, cfg.instanceID, request, state)
	if err := p.lifecycle.Host().ReportReadyIfNeeded(ctx, session, cfg.instanceID); err != nil {
		p.setLastError(err)
	} else if progressCleanupErr != nil {
		p.recordProgressCleanupError("clear progress after text delivery", progressCleanupErr)
	} else {
		p.clearLastError()
	}

	marker.Ack = &ack
	p.markers.RecordDelivery(marker)
	return ack, nil
}

func (p *telegramProvider) stopResources() {
	p.closeAllProgressDispatchers()
	for id, cfg := range p.routes.Snapshot() {
		if cfg.batcher == nil {
			continue
		}
		cfg.batcher.Close()
		p.routes.Update(id, func(current resolvedInstanceConfig) resolvedInstanceConfig {
			current.batcher = nil
			return current
		})
	}
}

func (p *telegramProvider) reconcileInstanceConfigs(
	ctx context.Context,
	session *bridgesdk.Session,
	managed []subprocess.InitializeBridgeManagedInstance,
) []resolvedInstanceConfig {
	reconciler := bridgesdk.ManagedConfigReconciler[resolvedInstanceConfig]{
		Routes:   p.routes,
		Resolve:  p.resolveInstanceConfig,
		Prepare:  p.prepareTelegramManagedConfigs,
		Finalize: p.populateTelegramInitialStates,
		Identity: func(config resolvedInstanceConfig) string { return config.instanceID },
		Merge: func(prior resolvedInstanceConfig, next resolvedInstanceConfig) resolvedInstanceConfig {
			if prior.batcher != nil && prior.batcher != next.batcher {
				prior.batcher.Close()
			}
			return next
		},
		OnRemoved: func(config resolvedInstanceConfig) error {
			if config.batcher != nil {
				config.batcher.Close()
			}
			return nil
		},
	}
	configs, err := reconciler.Reconcile(ctx, session, managed)
	if err != nil {
		if !errors.Is(err, bridgesdk.ErrProviderStopped) {
			p.setLastError(err)
		}
		return nil
	}
	return configs
}

func (p *telegramProvider) resolveInstanceConfig(
	session *bridgesdk.Session,
	managed subprocess.InitializeBridgeManagedInstance,
) resolvedInstanceConfig {
	cfg, err := decodeTelegramProviderConfig(managed)
	if err != nil {
		return resolvedInstanceConfig{
			managed:    &managed,
			instanceID: managed.Instance.ID,
			configError: fmt.Errorf(
				"telegram: decode provider_config for %q: %w",
				managed.Instance.ID,
				err,
			),
		}
	}

	resolved := buildTelegramResolvedInstance(session, managed, cfg)
	validateTelegramResolvedConfig(&resolved)
	if resolved.configError != nil {
		return resolved
	}
	configureTelegramBatcher(p, cfg, &resolved)
	return resolved
}

func decodeTelegramProviderConfig(managed subprocess.InitializeBridgeManagedInstance) (telegramProviderConfig, error) {
	cfg := telegramProviderConfig{}
	if len(managed.Instance.ProviderConfig) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(managed.Instance.ProviderConfig, &cfg); err != nil {
		return telegramProviderConfig{}, err
	}
	return cfg, nil
}

func buildTelegramResolvedInstance(
	session *bridgesdk.Session,
	managed subprocess.InitializeBridgeManagedInstance,
	cfg telegramProviderConfig,
) resolvedInstanceConfig {
	botToken, _ := session.Cache().BoundSecretValue(managed.Instance.ID, "bot_token")
	webhookSecret, _ := session.Cache().BoundSecretValue(managed.Instance.ID, "webhook_secret")

	resolved := resolvedInstanceConfig{
		managed:    &managed,
		instanceID: strings.TrimSpace(managed.Instance.ID),
		listenAddr: firstNonEmpty(
			cfg.Webhook.ListenAddr,
			strings.TrimSpace(os.Getenv(telegramListenAddrEnv)),
		),
		webhookPath: normalizeWebhookPath(
			firstNonEmpty(cfg.Webhook.Path, "/telegram/"+strings.TrimSpace(managed.Instance.ID)),
		),
		apiBaseURL: normalizeURL(
			firstNonEmpty(
				strings.TrimSpace(os.Getenv(telegramAPIBaseEnv)),
				telegramDefaultAPIBaseURL,
			),
		),
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
	return resolved
}

func validateTelegramResolvedConfig(resolved *resolvedInstanceConfig) {
	if resolved == nil {
		return
	}
	switch {
	case resolved.webhookPath == "":
		resolved.configError = errors.New("telegram: webhook path is required")
	case strings.TrimSpace(resolved.listenAddr) != "" && strings.TrimSpace(resolved.webhookSecret) == "":
		resolved.configError = errors.New("telegram: webhook secret is required when webhook listener is enabled")
	}
}

func configureTelegramBatcher(
	provider *telegramProvider,
	cfg telegramProviderConfig,
	resolved *resolvedInstanceConfig,
) {
	if resolved == nil || cfg.Batching.DelayMS <= 0 {
		return
	}

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
			return provider.dispatchInboundBatch(ctx, resolved.instanceID, batch)
		},
		Now: func() time.Time { return provider.now() },
	})
	if err != nil {
		resolved.configError = err
		return
	}
	resolved.batcher = batcher
}

func (p *telegramProvider) determineInitialState(
	ctx context.Context,
	cfg *resolvedInstanceConfig,
) (bridgepkg.BridgeStatus, *bridgepkg.BridgeDegradation, error) {
	if cfg == nil {
		err := errors.New("telegram: resolved instance config is required")
		return bridgepkg.BridgeStatusError, nil, err
	}
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
	if p.lifecycle.Stopped() {
		return errors.New("telegram: runtime is stopping")
	}
	if err := p.http.Start(listenAddr); err != nil {
		return fmt.Errorf("telegram: %w", err)
	}
	p.markers.RecordListen(p.http.Address())
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
		return p.handleWebhookRequest(w, r, &cfg, request)
	})
	if err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		p.setLastError(err)
		return
	}
	handler.ServeHTTP(w, r)
}

func (p *telegramProvider) handleWebhookRequest(
	w http.ResponseWriter,
	_ *http.Request,
	cfg *resolvedInstanceConfig,
	request bridgesdk.WebhookRequest,
) error {
	update := telegramUpdate{}
	if err := json.Unmarshal(request.Body, &update); err != nil {
		return &bridgesdk.HTTPError{
			StatusCode: http.StatusBadRequest,
			Message:    "invalid telegram webhook payload",
		}
	}
	message := selectTelegramMessage(update)
	if message == nil {
		return writeTelegramWebhookOK(w)
	}
	if isUnsupportedTextlessTelegramEdit(update) {
		return writeTelegramWebhookOK(w)
	}

	envelope, err := mapTelegramUpdate(update, *cfg.managed, request.ReceivedAt, nil)
	if err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
	}
	if cfg.dedup.Mark(envelope.IdempotencyKey) {
		return writeTelegramWebhookOK(w)
	}
	if !allowDirectMessage(cfg, *message) {
		return writeTelegramWebhookOK(w)
	}
	applyTelegramReplyContext(&envelope, *message, p.parents)
	if err := envelope.Validate(); err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
	}

	if cfg.batcher != nil && shouldBatchTelegramInbound(update) {
		if err := cfg.batcher.Enqueue(envelope); err != nil {
			return &bridgesdk.HTTPError{
				StatusCode: http.StatusInternalServerError,
				Message:    err.Error(),
			}
		}
	} else {
		if err := p.dispatchInboundEnvelope(context.Background(), cfg.instanceID, envelope); err != nil {
			return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
		}
	}
	rememberTelegramReplyParent(p.parents, envelope, *message)
	p.parents.RememberEnvelope(envelope)

	return writeTelegramWebhookOK(w)
}

func (p *telegramProvider) dispatchInboundBatch(
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

func (p *telegramProvider) dispatchInboundEnvelope(
	ctx context.Context,
	bridgeInstanceID string,
	envelope bridgepkg.InboundMessageEnvelope,
) error {
	session := p.currentSession()
	if session == nil {
		return errors.New("telegram: runtime session is not initialized")
	}
	cfg, err := p.configForInstance(bridgeInstanceID)
	if err != nil {
		return err
	}

	_, err = p.lifecycle.Host().IngestBridgeMessage(ctx, session, envelope)
	if err != nil {
		return err
	}
	if err := p.lifecycle.Host().ReportReadyIfNeeded(ctx, session, cfg.instanceID); err != nil {
		p.setLastError(err)
	} else {
		p.clearLastError()
	}
	return nil
}

func (p *telegramProvider) configForInstance(instanceID string) (resolvedInstanceConfig, error) {
	cfg, ok := p.routes.Get(instanceID)
	if !ok {
		return resolvedInstanceConfig{}, fmt.Errorf(
			"telegram: delivery targeted unmanaged instance %q",
			instanceID,
		)
	}
	return cfg, nil
}

func (p *telegramProvider) waitForInstanceConfig(
	instanceID string,
	timeout time.Duration,
) (resolvedInstanceConfig, error) {
	if timeout <= 0 {
		return p.configForInstance(instanceID)
	}

	cfg, ok, err := p.routes.Wait(context.Background(), instanceID, timeout, p.lifecycle.StopChannel())
	if err != nil {
		return resolvedInstanceConfig{}, err
	}
	if !ok {
		return p.configForInstance(instanceID)
	}
	return cfg, nil
}

func (p *telegramProvider) configForPath(path string) (resolvedInstanceConfig, bool) {
	configs := p.routes.ByPath(normalizeWebhookPath(path))
	if len(configs) == 0 {
		return resolvedInstanceConfig{}, false
	}
	return configs[0], true
}

func (p *telegramProvider) currentSession() *bridgesdk.Session {
	return p.lifecycle.Session()
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

func (p *telegramProvider) prepareTelegramManagedConfigs(
	_ context.Context,
	_ *bridgesdk.Session,
	configs []resolvedInstanceConfig,
) ([]resolvedInstanceConfig, error) {
	if len(configs) == 0 {
		return configs, nil
	}
	requestedListen := strings.TrimSpace(os.Getenv(telegramListenAddrEnv))
	usedPaths := make(map[string]string, len(configs))

	for idx := range configs {
		requestedListen = updateTelegramRequestedListen(&configs[idx], requestedListen)
		markDuplicateTelegramWebhookPath(&configs[idx], usedPaths)
	}
	if p.lifecycle.Stopped() {
		closeTelegramBatchers(configs)
		return nil, bridgesdk.ErrProviderStopped
	}
	applyTelegramListenErrors(configs, requestedListen, p.startServer)
	p.mu.Lock()
	p.listenAddr = requestedListen
	p.mu.Unlock()
	return configs, nil
}

func updateTelegramRequestedListen(cfg *resolvedInstanceConfig, requestedListen string) string {
	if cfg == nil || cfg.listenAddr == "" {
		return requestedListen
	}
	if requestedListen == "" {
		return cfg.listenAddr
	}
	if requestedListen != cfg.listenAddr && cfg.configError == nil {
		cfg.configError = fmt.Errorf(
			"telegram: instance %q configured incompatible listen_addr %q (runtime uses %q)",
			cfg.instanceID,
			cfg.listenAddr,
			requestedListen,
		)
	}
	return requestedListen
}

func markDuplicateTelegramWebhookPath(cfg *resolvedInstanceConfig, usedPaths map[string]string) {
	if cfg == nil || cfg.webhookPath == "" {
		return
	}
	if owner, ok := usedPaths[cfg.webhookPath]; ok && cfg.configError == nil {
		cfg.configError = fmt.Errorf(
			"telegram: webhook path %q is shared by %q and %q",
			cfg.webhookPath,
			owner,
			cfg.instanceID,
		)
	}
	usedPaths[cfg.webhookPath] = cfg.instanceID
}

func closeTelegramBatchers(configs []resolvedInstanceConfig) {
	for idx := range configs {
		if configs[idx].batcher != nil {
			configs[idx].batcher.Close()
			configs[idx].batcher = nil
		}
	}
}

func applyTelegramListenErrors(
	configs []resolvedInstanceConfig,
	requestedListen string,
	startServer func(string) error,
) {
	if requestedListen == "" {
		for idx := range configs {
			if configs[idx].configError == nil {
				configs[idx].configError = errors.New(
					"telegram: webhook listen address is required",
				)
			}
		}
		return
	}

	if err := startServer(requestedListen); err != nil {
		for idx := range configs {
			if configs[idx].configError == nil {
				configs[idx].configError = err
			}
		}
	}
}

func (p *telegramProvider) populateTelegramInitialStates(
	ctx context.Context,
	_ *bridgesdk.Session,
	configs []resolvedInstanceConfig,
) ([]resolvedInstanceConfig, error) {
	for idx := range configs {
		status, degradation, err := p.determineInitialState(ctx, &configs[idx])
		if err != nil {
			p.setLastError(err)
		}
		configs[idx].initialStatus = status
		configs[idx].initialDegradation = degradation
	}
	return configs, nil
}

func writeTelegramWebhookOK(w http.ResponseWriter) error {
	if w == nil {
		return nil
	}
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("OK"))
	return err
}

func allowDirectMessage(cfg *resolvedInstanceConfig, message telegramMessage) bool {
	if !isDirectChat(message.Chat.Type) {
		return true
	}
	if cfg == nil {
		return false
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

func identityAllowed(
	ids map[string]struct{},
	usernames map[string]struct{},
	user telegramUser,
) bool {
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

func resolveTelegramThreadID(threadID string, chatID string) (int64, error) {
	trimmedThreadID := strings.TrimSpace(threadID)
	if trimmedThreadID == "" {
		return 0, nil
	}
	if trimmedThreadID == telegramGeneralTopicID &&
		strings.HasPrefix(strings.TrimSpace(chatID), "-") {
		return 0, nil
	}
	value, err := strconv.ParseInt(trimmedThreadID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("telegram: invalid thread id %q: %w", trimmedThreadID, err)
	}
	return value, nil
}

func classifyTelegramHTTPError(
	statusCode int,
	response telegramAPIEnvelope[json.RawMessage],
) error {
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
			Err: &bridgesdk.HTTPError{
				StatusCode: statusCode,
				Message:    message,
				RetryAfter: retryAfter,
			},
			RetryAfter: retryAfter,
		}
	}
	if statusCode == 401 || statusCode == 403 {
		return &bridgesdk.AuthError{
			Err: &bridgesdk.HTTPError{StatusCode: statusCode, Message: message},
		}
	}
	httpErr := &bridgesdk.HTTPError{
		StatusCode: statusCode,
		Message:    message,
		RetryAfter: retryAfter,
	}
	if statusCode == http.StatusBadRequest && telegramMarkdownParseFailure(message) {
		return &telegramMarkdownParseError{err: httpErr}
	}
	return httpErr
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

func deliveryStateKey(instanceID string, deliveryID string) string {
	return strings.TrimSpace(instanceID) + ":" + strings.TrimSpace(deliveryID)
}

func optionalTelegramID(value int64) string {
	if value == 0 {
		return ""
	}
	return strconv.FormatInt(value, 10)
}

func normalizeDeliveryEventType(value bridgepkg.DeliveryEventType) bridgepkg.DeliveryEventType {
	return bridgepkg.DeliveryEventType(strings.ToLower(strings.TrimSpace(string(value))))
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
