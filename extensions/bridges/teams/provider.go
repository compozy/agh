package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	bridgepkg "github.com/compozy/agh/internal/bridges/contract"
	"github.com/compozy/agh/internal/bridgesdk"
	"github.com/compozy/agh/internal/subprocess"
)

const (
	providerActivityIDKey         = "activity_id"
	providerBaseConversationIDKey = "base_conversation_id"
	providerConversationIDKey     = "conversation_id"
	providerMessageKey            = "message"
	providerServiceURLKey         = "service_url"
	providerTeamsKey              = "teams"
	providerTenantIDKey           = "tenant_id"
)

const (
	teamsListenAddrEnv            = "AGH_BRIDGE_TEAMS_LISTEN_ADDR"
	teamsServiceURLEnv            = "AGH_BRIDGE_TEAMS_SERVICE_URL"
	teamsOpenIDMetadataURLEnv     = "AGH_BRIDGE_TEAMS_OPENID_METADATA_URL"
	teamsTestLoopbackAuthEnv      = "AGH_BRIDGE_TEAMS_ALLOW_LOOPBACK_AUTH_FOR_TESTING"
	teamsDefaultOpenIDMetadata    = "https://login.botframework.com/v1/.well-known/openidconfiguration"
	teamsDefaultServiceURL        = "https://smba.trafficmanager.net/teams/"
	teamsDefaultScope             = "https://api.botframework.com/.default"
	teamsWebhookReadHeaderTimeout = 10 * time.Second
	teamsWebhookIdleTimeout       = 30 * time.Second
	teamsAuthCacheTTL             = 5 * time.Minute
	rpcCodeNotInitialized         = -32003
)

var messageIDStripPattern = regexp.MustCompile(`;messageid=.+$`)

var teamsAuthHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

type teamsProvider struct {
	sdk        *bridgesdk.Runtime
	lifecycle  *bridgesdk.ProviderLifecycle
	markers    *bridgesdk.AdapterMarkers
	http       *bridgesdk.ProviderHTTPServer
	stderr     io.Writer
	now        func() time.Time
	routes     *bridgesdk.RouteTable[resolvedInstanceConfig]
	deliveries *bridgesdk.DeliveryStateStore[deliveryState]

	mu             sync.RWMutex
	lastError      string
	listenAddr     string
	reportedHealth map[string]string
	userContexts   map[string]teamsUserContext
	apiFactory     func(resolvedInstanceConfig) teamsAPI
}

type teamsOpenIDMetadataCacheEntry struct {
	metadata  teamsOpenIDMetadata
	expiresAt time.Time
}

type teamsJWKSCacheEntry struct {
	jwks      teamsJWKS
	expiresAt time.Time
}

var teamsAuthCache = struct {
	mu       sync.Mutex
	metadata map[string]teamsOpenIDMetadataCacheEntry
	jwks     map[string]teamsJWKSCacheEntry
}{
	metadata: make(map[string]teamsOpenIDMetadataCacheEntry),
	jwks:     make(map[string]teamsJWKSCacheEntry),
}

type teamsProviderConfig struct {
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
	serviceURL         string
	appID              string
	appPassword        string
	appTenantID        string
	openIDMetadataURL  string
	tokenURL           string
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

type teamsUserContext struct {
	ServiceURL string
	TenantID   string
}

type teamsActivity struct {
	Type             string                 `json:"type"`
	ID               string                 `json:"id,omitempty"`
	Name             string                 `json:"name,omitempty"`
	Action           string                 `json:"action,omitempty"`
	Text             string                 `json:"text,omitempty"`
	TextFormat       string                 `json:"textFormat,omitempty"`
	Timestamp        string                 `json:"timestamp,omitempty"`
	ServiceURL       string                 `json:"serviceUrl,omitempty"`
	ChannelID        string                 `json:"channelId,omitempty"`
	ReplyToID        string                 `json:"replyToId,omitempty"`
	From             teamsChannelAccount    `json:"from"`
	Recipient        teamsChannelAccount    `json:"recipient"`
	Conversation     teamsConversation      `json:"conversation"`
	Attachments      []teamsAttachment      `json:"attachments,omitempty"`
	Entities         []teamsEntity          `json:"entities,omitempty"`
	ChannelData      teamsChannelData       `json:"channelData"`
	Value            json.RawMessage        `json:"value,omitempty"`
	ReactionsAdded   []teamsMessageReaction `json:"reactionsAdded,omitempty"`
	ReactionsRemoved []teamsMessageReaction `json:"reactionsRemoved,omitempty"`
}

type teamsChannelAccount struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	AADObjectID string `json:"aadObjectId,omitempty"`
}

type teamsConversation struct {
	ID               string `json:"id,omitempty"`
	Name             string `json:"name,omitempty"`
	TenantID         string `json:"tenantId,omitempty"`
	ConversationType string `json:"conversationType,omitempty"`
	IsGroup          bool   `json:"isGroup,omitempty"`
}

type teamsAttachment struct {
	ContentType string          `json:"contentType,omitempty"`
	ContentURL  string          `json:"contentUrl,omitempty"`
	Name        string          `json:"name,omitempty"`
	Content     json.RawMessage `json:"content,omitempty"`
}

type teamsEntity struct {
	Type      string               `json:"type,omitempty"`
	Text      string               `json:"text,omitempty"`
	Mentioned *teamsChannelAccount `json:"mentioned,omitempty"`
}

type teamsChannelData struct {
	Tenant *struct {
		ID string `json:"id,omitempty"`
	} `json:"tenant,omitempty"`
	Channel *struct {
		ID string `json:"id,omitempty"`
	} `json:"channel,omitempty"`
	Team *struct {
		ID         string `json:"id,omitempty"`
		AADGroupID string `json:"aadGroupId,omitempty"`
	} `json:"team,omitempty"`
	EventType string `json:"eventType,omitempty"`
}

type teamsMessageReaction struct {
	Type string `json:"type,omitempty"`
}

type teamsActionValue struct {
	Action *struct {
		Data json.RawMessage `json:"data,omitempty"`
	} `json:"action,omitempty"`
}

type teamsActionPayload struct {
	ActionID string `json:"actionId,omitempty"`
	Value    string `json:"value,omitempty"`
}

type teamsThreadRef struct {
	ConversationID string
	ServiceURL     string
}

type teamsResolvedTarget struct {
	ConversationID string
	ServiceURL     string
	UserID         string
	TenantID       string
	ReplyToID      string
}

type teamsAPI interface {
	ValidateAuth(context.Context) error
	CreateConversation(
		context.Context,
		string,
		teamsCreateConversationRequest,
	) (*teamsConversationResourceResponse, error)
	SendActivity(
		context.Context,
		string,
		string,
		string,
		teamsOutboundActivity,
	) (*teamsResourceResponse, error)
	UpdateActivity(context.Context, string, string, string, teamsOutboundActivity) error
	DeleteActivity(context.Context, string, string, string) error
}

type teamsBotClient struct {
	cfg                   resolvedInstanceConfig
	httpClient            *http.Client
	reportResponseCleanup func(error)

	mu          sync.Mutex
	cachedToken string
	tokenExpiry time.Time
}

type teamsOpenIDMetadata struct {
	Issuer  string `json:"issuer,omitempty"`
	JWKSURI string `json:"jwks_uri,omitempty"`
}

type teamsJWKS struct {
	Keys []teamsJWK `json:"keys"`
}

type teamsJWK struct {
	Kid          string   `json:"kid,omitempty"`
	X5T          string   `json:"x5t,omitempty"`
	Kty          string   `json:"kty,omitempty"`
	N            string   `json:"n,omitempty"`
	E            string   `json:"e,omitempty"`
	Endorsements []string `json:"endorsements,omitempty"`
}

type teamsAuthClaims struct {
	ServiceURL string `json:"serviceUrl,omitempty"`
	jwt.RegisteredClaims
}

type teamsTokenResponse struct {
	AccessToken string `json:"access_token,omitempty"`
	ExpiresIn   int    `json:"expires_in,omitempty"`
}

type teamsCreateConversationRequest struct {
	Bot         teamsChannelAccount   `json:"bot"`
	Members     []teamsChannelAccount `json:"members"`
	IsGroup     bool                  `json:"isGroup"`
	TenantID    string                `json:"tenantId,omitempty"`
	ChannelData map[string]any        `json:"channelData,omitempty"`
}

type teamsConversationResourceResponse struct {
	ID string `json:"id,omitempty"`
}

type teamsOutboundActivity struct {
	Type       string              `json:"type"`
	Text       string              `json:"text,omitempty"`
	TextFormat string              `json:"textFormat,omitempty"`
	From       teamsChannelAccount `json:"from"`
	Recipient  teamsChannelAccount `json:"recipient"`
}

type teamsResourceResponse struct {
	ID string `json:"id,omitempty"`
}

//nolint:funlen // Construction keeps the provider's declarative runtime wiring visible in one place.
func newTeamsProvider(stderr io.Writer) (*teamsProvider, error) {
	if stderr == nil {
		stderr = io.Discard
	}

	provider := &teamsProvider{
		stderr:  stderr,
		markers: bridgesdk.NewAdapterMarkers(providerTeamsKey, stderr),
		now:     func() time.Time { return time.Now().UTC() },
		routes: bridgesdk.NewRouteTable(func(config resolvedInstanceConfig) []string {
			return []string{config.webhookPath}
		}),
		deliveries:     newTeamsDeliveryStateStore(),
		reportedHealth: make(map[string]string),
		userContexts:   make(map[string]teamsUserContext),
	}
	provider.apiFactory = func(cfg resolvedInstanceConfig) teamsAPI {
		return newTeamsBotClient(&cfg, provider.markers)
	}

	lifecycle, err := bridgesdk.NewProviderLifecycle(bridgesdk.ProviderLifecycleConfig{
		ProviderName: providerTeamsKey,
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
		ReadHeaderTimeout: teamsWebhookReadHeaderTimeout,
		IdleTimeout:       teamsWebhookIdleTimeout,
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
			Name:    providerTeamsKey,
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

func (p *teamsProvider) serve(stdin io.Reader, stdout io.Writer) error {
	return p.lifecycle.Serve(context.Background(), p.sdk, stdin, stdout)
}

func (p *teamsProvider) stopResources() {
	p.closeAllTeamsProgressDispatchers()
	batchers := make(map[*bridgesdk.InboundBatcher]struct{})
	for id, cfg := range p.routes.Snapshot() {
		if cfg.batcher == nil {
			continue
		}
		batchers[cfg.batcher] = struct{}{}
		p.routes.Update(id, func(current resolvedInstanceConfig) resolvedInstanceConfig {
			current.batcher = nil
			return current
		})
	}
	for batcher := range batchers {
		batcher.Close()
	}
}

func (p *teamsProvider) reconcileInstanceConfigs(
	ctx context.Context,
	session *bridgesdk.Session,
	managed []subprocess.InitializeBridgeManagedInstance,
) []resolvedInstanceConfig {
	reconciler := bridgesdk.ManagedConfigReconciler[resolvedInstanceConfig]{
		Routes:   p.routes,
		Resolve:  p.resolveInstanceConfig,
		Prepare:  p.prepareTeamsManagedConfigs,
		Finalize: p.populateTeamsInitialStates,
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
			p.lifecycle.Host().ForgetStatus(config.instanceID)
			p.mu.Lock()
			delete(p.reportedHealth, config.instanceID)
			p.mu.Unlock()
			return nil
		},
	}
	configs, err := reconciler.Reconcile(ctx, session, managed)
	if err != nil {
		p.setLastError(err)
		return nil
	}
	return configs
}

func (p *teamsProvider) resolveInstanceConfig(
	session *bridgesdk.Session,
	managed subprocess.InitializeBridgeManagedInstance,
) resolvedInstanceConfig {
	cfg, err := decodeTeamsProviderConfig(managed)
	if err != nil {
		return resolvedInstanceConfig{
			managed:    &managed,
			instanceID: managed.Instance.ID,
			configError: fmt.Errorf(
				"teams: decode provider_config for %q: %w",
				managed.Instance.ID,
				err,
			),
		}
	}

	resolved := buildTeamsResolvedInstance(session, managed, cfg)
	validateTeamsResolvedConfig(&resolved)
	if resolved.configError != nil {
		return resolved
	}
	configureTeamsBatcher(p, cfg, &resolved)
	return resolved
}

func (p *teamsProvider) determineInitialState(
	ctx context.Context,
	cfg resolvedInstanceConfig,
) (bridgepkg.BridgeStatus, *bridgepkg.BridgeDegradation, error) {
	if cfg.configError != nil {
		return bridgepkg.BridgeStatusDegraded, &bridgepkg.BridgeDegradation{
			Reason:  bridgepkg.BridgeDegradationReasonTenantConfigInvalid,
			Message: cfg.configError.Error(),
		}, cfg.configError
	}
	if strings.TrimSpace(cfg.appID) == "" {
		err := errors.New("teams: app_id secret binding is required")
		return bridgepkg.BridgeStatusAuthRequired, &bridgepkg.BridgeDegradation{
			Reason:  bridgepkg.BridgeDegradationReasonAuthFailed,
			Message: err.Error(),
		}, err
	}
	if strings.TrimSpace(cfg.appPassword) == "" {
		err := errors.New("teams: app_password secret binding is required")
		return bridgepkg.BridgeStatusAuthRequired, &bridgepkg.BridgeDegradation{
			Reason:  bridgepkg.BridgeDegradationReasonAuthFailed,
			Message: err.Error(),
		}, err
	}
	if err := p.apiFactory(cfg).ValidateAuth(ctx); err != nil {
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

func (p *teamsProvider) startServer(listenAddr string) error {
	if err := p.http.Start(listenAddr); err != nil {
		return fmt.Errorf("teams: %w", err)
	}
	p.markers.RecordListen(p.http.Address())
	return nil
}

func (p *teamsProvider) serveWebhookHTTP(w http.ResponseWriter, r *http.Request) {
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
			return verifyTeamsAuthorization(ctx, req, body, cfg)
		},
		RequestKey: func(req *http.Request) string {
			return req.RemoteAddr + "|" + cfg.instanceID
		},
		Now: func() time.Time { return p.now() },
	}, func(w http.ResponseWriter, _ *http.Request, request bridgesdk.WebhookRequest) error {
		return p.handleWebhookRequest(w, cfg, request)
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

func (p *teamsProvider) handleWebhookRequest(
	w http.ResponseWriter,
	cfg resolvedInstanceConfig,
	request bridgesdk.WebhookRequest,
) error {
	var activity teamsActivity
	if err := json.Unmarshal(request.Body, &activity); err != nil {
		return &bridgesdk.HTTPError{
			StatusCode: http.StatusBadRequest,
			Message:    "invalid teams activity payload",
		}
	}

	p.storeUserContext(cfg.instanceID, activity)

	items, err := mapTeamsActivity(activity, cfg, request.ReceivedAt)
	if err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
	}
	if len(items) == 0 {
		return writeWebhookOK(w)
	}

	for _, item := range items {
		if cfg.dedup.Mark(item.Envelope.IdempotencyKey) {
			continue
		}
		if !allowTeamsDirectMessage(cfg, item.User, item.Direct) {
			continue
		}
		if cfg.batcher != nil &&
			item.Envelope.EventFamily.Normalize() == bridgepkg.InboundEventFamilyMessage {
			if err := cfg.batcher.Enqueue(item.Envelope); err != nil {
				return &bridgesdk.HTTPError{
					StatusCode: http.StatusInternalServerError,
					Message:    err.Error(),
				}
			}
			continue
		}
		if err := p.dispatchInboundEnvelope(context.Background(), cfg.instanceID, item.Envelope); err != nil {
			return &bridgesdk.HTTPError{
				StatusCode: http.StatusInternalServerError,
				Message:    err.Error(),
			}
		}
	}
	return writeWebhookOK(w)
}

func (p *teamsProvider) dispatchInboundBatch(
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

func (p *teamsProvider) dispatchInboundEnvelope(
	ctx context.Context,
	bridgeInstanceID string,
	envelope bridgepkg.InboundMessageEnvelope,
) error {
	session := p.currentSession()
	if session == nil {
		return errors.New("teams: runtime session is not initialized")
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

func (p *teamsProvider) configForInstance(instanceID string) (resolvedInstanceConfig, error) {
	cfg, ok := p.routes.Get(instanceID)
	if !ok {
		return resolvedInstanceConfig{}, fmt.Errorf(
			"teams: bridge instance %q is not initialized",
			instanceID,
		)
	}
	return cfg, nil
}

func (p *teamsProvider) waitForInstanceConfig(
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

func (p *teamsProvider) configForPath(path string) (resolvedInstanceConfig, bool) {
	configs := p.routes.ByPath(normalizeWebhookPath(path))
	if len(configs) != 1 || configs[0].configError != nil {
		return resolvedInstanceConfig{}, false
	}
	return configs[0], true
}

func (p *teamsProvider) currentSession() *bridgesdk.Session {
	return p.lifecycle.Session()
}

func (p *teamsProvider) setLastError(err error) {
	if err == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastError = err.Error()
}

func (p *teamsProvider) clearLastError() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastError = ""
}

func bridgeHealthMessage(degradation *bridgepkg.BridgeDegradation) string {
	if degradation == nil {
		return ""
	}
	if message := strings.TrimSpace(degradation.Message); message != "" {
		return message
	}
	return strings.TrimSpace(string(degradation.Reason))
}

func (p *teamsProvider) storeUserContext(instanceID string, activity teamsActivity) {
	userID := normalizeTeamsID(activity.From.ID)
	if userID == "" {
		return
	}
	ctx := teamsUserContext{
		ServiceURL: normalizeURL(activity.ServiceURL),
		TenantID:   strings.TrimSpace(extractTeamsTenantID(activity)),
	}
	if ctx.ServiceURL == "" && ctx.TenantID == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	current := p.userContexts[userContextKey(instanceID, userID)]
	if ctx.ServiceURL == "" {
		ctx.ServiceURL = current.ServiceURL
	}
	if ctx.TenantID == "" {
		ctx.TenantID = current.TenantID
	}
	p.userContexts[userContextKey(instanceID, userID)] = ctx
}

func (p *teamsProvider) userContext(instanceID string, userID string) (teamsUserContext, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	ctx, ok := p.userContexts[userContextKey(instanceID, normalizeTeamsID(userID))]
	return ctx, ok
}
