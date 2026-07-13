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
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	bridgepkg "github.com/compozy/agh/internal/bridges"
	"github.com/compozy/agh/internal/bridgesdk"
	"github.com/compozy/agh/internal/subprocess"
)

const (
	providerAghBridgeDeliveryKey = "agh_bridge_delivery"
	providerBridgeInstanceIDKey  = "bridge_instance_id"
	providerChannelIDKey         = "channel_id"
	providerChannelNameKey       = "channel_name"
	providerDeliveryIDKey        = "delivery_id"
	providerMessageKey           = "message"
	providerReactionAddedKey     = "reaction_added"
	providerResponseURLKey       = "response_url"
	providerSlackKey             = "slack"
	providerTeamIDKey            = "team_id"
	providerTriggerIDKey         = "trigger_id"
	providerTypeKey              = "type"
)

const (
	slackListenAddrEnv = "AGH_BRIDGE_SLACK_LISTEN_ADDR"
	slackAPIBaseEnv    = "AGH_BRIDGE_SLACK_API_BASE_URL"

	slackDefaultAPIBaseURL        = "https://slack.com/api"
	slackSignatureVersion         = "v0"
	slackWebhookReadHeaderTimeout = 10 * time.Second
	slackWebhookIdleTimeout       = 2 * time.Minute
)

type slackProvider struct {
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
	apiFactory func(*resolvedInstanceConfig) slackAPI
}

type slackProviderConfig struct {
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
	BotID           string                `json:"bot_id,omitempty"`
	Channel         string                `json:"channel,omitempty"`
	ChannelType     string                `json:"channel_type,omitempty"`
	DeletedTS       string                `json:"deleted_ts,omitempty"`
	Edited          *slackEdit            `json:"edited,omitempty"`
	Files           []slackFile           `json:"files,omitempty"`
	Message         *slackMessageSnapshot `json:"message,omitempty"`
	PreviousMessage *slackMessageSnapshot `json:"previous_message,omitempty"`
	Subtype         string                `json:"subtype,omitempty"`
	Team            string                `json:"team,omitempty"`
	TeamID          string                `json:"team_id,omitempty"`
	Text            string                `json:"text,omitempty"`
	ThreadTS        string                `json:"thread_ts,omitempty"`
	TS              string                `json:"ts,omitempty"`
	Type            string                `json:"type"`
	User            string                `json:"user,omitempty"`
	Username        string                `json:"username,omitempty"`
}

type slackMessageSnapshot struct {
	BotID    string      `json:"bot_id,omitempty"`
	Edited   *slackEdit  `json:"edited,omitempty"`
	Files    []slackFile `json:"files,omitempty"`
	Text     string      `json:"text,omitempty"`
	ThreadTS string      `json:"thread_ts,omitempty"`
	TS       string      `json:"ts,omitempty"`
	Type     string      `json:"type,omitempty"`
	User     string      `json:"user,omitempty"`
	Username string      `json:"username,omitempty"`
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
	SetThreadStatus(context.Context, slackSetThreadStatusRequest) error
	AddReaction(context.Context, slackAddReactionRequest) error
}

type slackDeliveryReconciler interface {
	FindDeliveryMessage(context.Context, slackFindDeliveryMessageRequest) (*slackPostedMessage, error)
}

type slackAuthIdentity struct {
	BotID         string   `json:"bot_id,omitempty"`
	UserID        string   `json:"user_id,omitempty"`
	GrantedScopes []string `json:"-"`
}

type slackPostedMessage struct {
	TS string `json:"ts,omitempty"`
}

type slackPostMessageRequest struct {
	Channel  string                `json:"channel"`
	ThreadTS string                `json:"thread_ts,omitempty"`
	Text     string                `json:"text"`
	Mrkdwn   bool                  `json:"mrkdwn"`
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

//nolint:funlen // Construction keeps the provider's declarative runtime wiring visible in one place.
func newSlackProvider(stderr io.Writer) (*slackProvider, error) {
	if stderr == nil {
		stderr = io.Discard
	}

	provider := &slackProvider{
		stderr:  stderr,
		markers: bridgesdk.NewAdapterMarkers(providerSlackKey, stderr),
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
	provider.apiFactory = func(cfg *resolvedInstanceConfig) slackAPI {
		return &slackBotClient{
			baseURL:  cfg.apiBaseURL,
			botToken: cfg.botToken,
			httpClient: &http.Client{
				Timeout: 10 * time.Second,
			},
		}
	}

	lifecycle, err := bridgesdk.NewProviderLifecycle(bridgesdk.ProviderLifecycleConfig{
		ProviderName: providerSlackKey,
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
		ReadHeaderTimeout: slackWebhookReadHeaderTimeout,
		IdleTimeout:       slackWebhookIdleTimeout,
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
			Name:    providerSlackKey,
			Version: "0.1.0",
			SDKName: "bridgesdk",
		},
		Initialize:  lifecycle.Initialize,
		Deliver:     provider.handleBridgesDeliver,
		Progress:    provider.handleBridgesProgress,
		Check:       provider.handleBridgeCheck,
		HealthCheck: func(context.Context, *bridgesdk.Session) error { return provider.healthCheck() },
		Shutdown:    lifecycle.Shutdown,
		Now:         func() time.Time { return provider.now() },
	})
	if err != nil {
		return nil, err
	}
	provider.sdk = sdkRuntime
	return provider, nil
}

func (p *slackProvider) serve(stdin io.Reader, stdout io.Writer) error {
	return p.lifecycle.Serve(context.Background(), p.sdk, stdin, stdout)
}

func (p *slackProvider) handleBridgesDeliver(
	ctx context.Context,
	session *bridgesdk.Session,
	request bridgepkg.DeliveryRequest,
) (bridgepkg.DeliveryAck, error) {
	marker := bridgesdk.DeliveryMarker{
		PID:     os.Getpid(),
		Request: request,
	}

	cfg, err := p.waitForInstanceConfig(strings.TrimSpace(request.Event.BridgeInstanceID), 500*time.Millisecond)
	if err != nil {
		marker.Error = err.Error()
		p.markers.RecordDelivery(marker)
		p.setLastError(err)
		return bridgepkg.DeliveryAck{}, err
	}

	if p.markers.ShouldCrashOnce() {
		p.markers.RecordDelivery(marker)
		p.markers.RecordCrash(map[string]any{
			"crashed":                   true,
			"pid":                       os.Getpid(),
			providerDeliveryIDKey:       strings.TrimSpace(request.Event.DeliveryID),
			providerBridgeInstanceIDKey: cfg.instanceID,
		})
		os.Exit(23)
	}

	ack, state, err := p.executeTextDeliveryWithProgress(ctx, &cfg, request)
	if err != nil {
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
	if progressCleanupErr == nil {
		p.clearLastError()
	}
	if err := p.lifecycle.Host().ReportReadyIfNeeded(ctx, session, cfg.instanceID); err != nil {
		p.setLastError(err)
	}

	marker.Ack = &ack
	p.markers.RecordDelivery(marker)
	if progressCleanupErr != nil {
		p.recordProgressCleanupError("clear progress after text delivery", progressCleanupErr)
	}
	return ack, nil
}

func (p *slackProvider) stopResources() {
	p.closeAllProgressDispatchers()
	batchersToClose := make(map[*bridgesdk.InboundBatcher]struct{})
	for id, cfg := range p.routes.Snapshot() {
		if cfg.batcher == nil {
			continue
		}
		batchersToClose[cfg.batcher] = struct{}{}
		p.routes.Update(id, func(current resolvedInstanceConfig) resolvedInstanceConfig {
			current.batcher = nil
			return current
		})
	}
	closeInboundBatchers(batchersToClose)
}

func (p *slackProvider) reconcileInstanceConfigs(
	ctx context.Context,
	session *bridgesdk.Session,
	managed []subprocess.InitializeBridgeManagedInstance,
) []resolvedInstanceConfig {
	batchersToClose := make(map[*bridgesdk.InboundBatcher]struct{})
	reconciler := bridgesdk.ManagedConfigReconciler[resolvedInstanceConfig]{
		Routes:   p.routes,
		Resolve:  p.resolveInstanceConfig,
		Prepare:  p.prepareSlackConfigs,
		Finalize: p.finalizeSlackConfigs,
		Identity: func(config resolvedInstanceConfig) string { return config.instanceID },
		Merge: func(prior resolvedInstanceConfig, next resolvedInstanceConfig) resolvedInstanceConfig {
			if prior.batcher != nil && prior.batcher != next.batcher {
				batchersToClose[prior.batcher] = struct{}{}
			}
			return next
		},
		OnRemoved: func(config resolvedInstanceConfig) error {
			if config.batcher != nil {
				batchersToClose[config.batcher] = struct{}{}
			}
			return nil
		},
		OnPublish: func() { closeInboundBatchers(batchersToClose) },
	}
	configs, err := reconciler.Reconcile(ctx, session, managed)
	if err != nil {
		p.setLastError(err)
		return nil
	}
	return configs
}

func (p *slackProvider) finalizeSlackConfigs(
	ctx context.Context,
	_ *bridgesdk.Session,
	configs []resolvedInstanceConfig,
) ([]resolvedInstanceConfig, error) {
	for idx := range configs {
		status, degradation, probeErr := p.determineInitialState(ctx, &configs[idx])
		if probeErr != nil {
			p.setLastError(probeErr)
		}
		configs[idx].initialStatus = status
		configs[idx].initialDegradation = degradation
	}
	return configs, nil
}

func (p *slackProvider) prepareSlackConfigs(
	_ context.Context,
	_ *bridgesdk.Session,
	configs []resolvedInstanceConfig,
) ([]resolvedInstanceConfig, error) {
	if len(configs) == 0 {
		return configs, nil
	}
	requestedListen := strings.TrimSpace(os.Getenv(slackListenAddrEnv))
	usedPaths := make(map[string]int, len(configs))

	for idx := range configs {
		requestedListen = applySlackListenConstraint(&configs[idx], requestedListen)
		applySlackWebhookPathConflict(&configs[idx], usedPaths, configs[:idx])
	}
	p.applySlackListenErrors(configs, requestedListen)
	p.mu.Lock()
	p.listenAddr = requestedListen
	p.mu.Unlock()
	return configs, nil
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

func (p *slackProvider) resolveInstanceConfig(
	session *bridgesdk.Session,
	managed subprocess.InitializeBridgeManagedInstance,
) resolvedInstanceConfig {
	cfg := slackProviderConfig{}
	if len(managed.Instance.ProviderConfig) > 0 {
		if err := json.Unmarshal(managed.Instance.ProviderConfig, &cfg); err != nil {
			return resolvedInstanceConfig{
				managed:     &managed,
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
		firstNonEmpty(strings.TrimSpace(os.Getenv(slackAPIBaseEnv)), slackDefaultAPIBaseURL),
	)

	resolved := resolvedInstanceConfig{
		managed:         &managed,
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
	cfg *resolvedInstanceConfig,
) (bridgepkg.BridgeStatus, *bridgepkg.BridgeDegradation, error) {
	if cfg == nil {
		err := errors.New("slack: resolved instance config is required")
		return bridgepkg.BridgeStatusError, nil, err
	}
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
	if err := p.http.Start(listenAddr); err != nil {
		return fmt.Errorf("slack: %w", err)
	}
	p.markers.RecordListen(p.http.Address())
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
		return p.handleWebhookRequest(w, r, &cfg, request)
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
	cfg *resolvedInstanceConfig,
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
	cfg *resolvedInstanceConfig,
	request bridgesdk.WebhookRequest,
) error {
	values, err := url.ParseQuery(string(request.Body))
	if err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: "invalid slack form payload"}
	}
	if values.Has("command") && !values.Has("payload") {
		mapped, err := mapSlackSlashCommand(values, *cfg.managed, request.ReceivedAt)
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

	mapped, err := mapSlackBlockActions(payload, *cfg.managed, request.ReceivedAt)
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
	cfg *resolvedInstanceConfig,
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
	case providerMessageKey, "app_mention":
		return p.handleSlackMessageJSONEvent(ctx, w, cfg, request, payload)
	case providerReactionAddedKey, "reaction_removed":
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
	cfg *resolvedInstanceConfig,
	request bridgesdk.WebhookRequest,
	payload slackWebhookEnvelope,
) error {
	var event slackMessageEvent
	if err := json.Unmarshal(payload.Event, &event); err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: "invalid slack message event"}
	}
	mapped, ignored, err := mapSlackMessageEvent(
		event,
		*cfg.managed,
		request.ReceivedAt,
		payload.EventID,
		payload.TeamID,
		payload.EventTime,
		nil,
	)
	if err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
	}
	if ignored {
		return writeWebhookOK(w)
	}
	return p.dispatchSlackWebhookEnvelope(
		ctx,
		w,
		cfg,
		mapped,
		slackEventReplyParentMessageID(event),
		shouldBatchSlackInbound(event),
	)
}

func (p *slackProvider) handleSlackReactionJSONEvent(
	ctx context.Context,
	w http.ResponseWriter,
	cfg *resolvedInstanceConfig,
	request bridgesdk.WebhookRequest,
	payload slackWebhookEnvelope,
) error {
	var event slackReactionEvent
	if err := json.Unmarshal(payload.Event, &event); err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: "invalid slack reaction event"}
	}
	mapped, err := mapSlackReactionEvent(event, *cfg.managed, request.ReceivedAt, payload.EventID, payload.TeamID)
	if err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
	}
	return p.dispatchSlackWebhookEnvelope(ctx, w, cfg, mapped, "", false)
}

func (p *slackProvider) dispatchSlackWebhookEnvelope(
	ctx context.Context,
	w http.ResponseWriter,
	cfg *resolvedInstanceConfig,
	mapped slackMappedInbound,
	parentMessageID string,
	allowBatch bool,
) error {
	if cfg.dedup.Seen(mapped.Envelope.IdempotencyKey) {
		return writeWebhookOK(w)
	}
	if !allowSlackDirectMessage(cfg, mapped.User, mapped.Direct) {
		return writeWebhookOK(w)
	}
	if p.parents.EnrichReply(&mapped.Envelope, parentMessageID) {
		if err := mapped.Envelope.Validate(); err != nil {
			return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
		}
	}
	if allowBatch && cfg.batcher != nil {
		if err := cfg.batcher.Enqueue(mapped.Envelope); err != nil {
			return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
		}
		p.parents.RememberEnvelope(mapped.Envelope)
		cfg.dedup.Mark(mapped.Envelope.IdempotencyKey)
		return writeWebhookOK(w)
	}
	if err := p.dispatchInboundEnvelope(ctx, cfg.instanceID, mapped.Envelope); err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
	}
	p.parents.RememberEnvelope(mapped.Envelope)
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

	_, err = p.lifecycle.Host().IngestBridgeMessage(ctx, session, envelope)
	if err != nil {
		return err
	}
	if err := p.lifecycle.Host().ReportReadyIfNeeded(ctx, session, cfg.instanceID); err != nil {
		p.setLastError(err)
	}
	return nil
}

func (p *slackProvider) configForInstance(instanceID string) (resolvedInstanceConfig, error) {
	cfg, ok := p.routes.Get(instanceID)
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

	cfg, ok, err := p.routes.Wait(context.Background(), instanceID, timeout, p.lifecycle.StopChannel())
	if err != nil {
		return resolvedInstanceConfig{}, err
	}
	if !ok {
		return p.configForInstance(instanceID)
	}
	return cfg, nil
}

func (p *slackProvider) configForPath(path string) (resolvedInstanceConfig, bool) {
	configs := p.routes.ByPath(normalizeWebhookPath(path))
	if len(configs) == 0 {
		return resolvedInstanceConfig{}, false
	}
	return configs[0], true
}

func (p *slackProvider) currentSession() *bridgesdk.Session {
	return p.lifecycle.Session()
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
