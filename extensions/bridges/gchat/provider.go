package main

import (
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
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
	providerAudKey        = "aud"
	providerExpKey        = "exp"
	providerGchatKey      = "gchat"
	providerIatKey        = "iat"
	providerIssKey        = "iss"
	providerNameKey       = "name"
	providerSourceKey     = "source"
	providerSpaceNameKey  = "space_name"
	providerSuccessKey    = "success"
	providerTextKey       = "text"
	providerThreadNameKey = "thread_name"
)

const (
	gchatListenAddrEnv  = "AGH_BRIDGE_GCHAT_LISTEN_ADDR"
	gchatAPIBaseEnv     = "AGH_BRIDGE_GCHAT_API_BASE_URL"
	gchatDirectCertsEnv = "AGH_BRIDGE_GCHAT_DIRECT_CERTS_URL"
	gchatPubSubCertsEnv = "AGH_BRIDGE_GCHAT_PUBSUB_CERTS_URL"

	gchatDefaultAPIBaseURL      = "https://chat.googleapis.com"
	gchatDefaultAuthEndpointURL = "https://oauth2.googleapis.com/token"
	gchatDefaultDirectCertsURL  = "https://www.googleapis.com/service_accounts/v1/metadata/x509/" +
		"chat@system.gserviceaccount.com"
	gchatDefaultPubSubCertsURL    = "https://www.googleapis.com/oauth2/v1/certs"
	gchatDefaultDirectIssuer      = "chat@system.gserviceaccount.com"
	gchatDefaultPubSubIssuerURL   = "https://accounts.google.com"
	gchatBotScope                 = "https://www.googleapis.com/auth/chat.bot"
	gchatWebhookReadHeaderTimeout = 10 * time.Second
	gchatWebhookIdleTimeout       = 2 * time.Minute
	gchatCertFetchTimeout         = 5 * time.Second
	gchatCertCacheFallbackTTL     = 5 * time.Minute

	gchatModeDirect = "direct"
	gchatModePubSub = "pubsub"
	gchatModeHybrid = "hybrid"

	gchatReplyMode = "REPLY_MESSAGE_FALLBACK_TO_NEW_THREAD"
)

var gchatTokenURLEnv = strings.Join([]string{"AGH", "BRIDGE", "GCHAT", "TOKEN", "URL"}, "_")

var reactionMessagePattern = regexp.MustCompile(`^(spaces/[^/]+/messages/[^/]+)/reactions/[^/]+$`)

var defaultGoogleX509KeyCache = newGoogleX509KeyCache(
	&http.Client{Timeout: gchatCertFetchTimeout},
	gchatCertCacheFallbackTTL,
	func() time.Time { return time.Now().UTC() },
)

type gchatProvider struct {
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
	apiFactory func(*resolvedInstanceConfig) gchatAPI
	apiClients map[string]cachedGChatAPIClient
}

type cachedGChatAPIClient struct {
	key    string
	client *gchatBotClient
}

type gchatProviderConfig struct {
	Mode    string `json:"mode,omitempty"`
	Webhook struct {
		ListenAddr string `json:"listen_addr,omitempty"`
		Path       string `json:"path,omitempty"`
	} `json:"webhook"`
	Verification struct {
		DirectCertsURL       string `json:"direct_certs_url,omitempty"`
		DirectIssuer         string `json:"direct_issuer,omitempty"`
		PubSubAudience       string `json:"pubsub_audience,omitempty"`
		PubSubCertsURL       string `json:"pubsub_certs_url,omitempty"`
		PubSubIssuer         string `json:"pubsub_issuer,omitempty"`
		PubSubServiceAccount string `json:"pubsub_service_account_email,omitempty"`
	} `json:"verification"`
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

type serviceAccountCredentials struct {
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
	ProjectID   string `json:"project_id,omitempty"`
	TokenURI    string `json:"token_uri,omitempty"`
}

type googleX509KeyCache struct {
	mu          sync.Mutex
	client      *http.Client
	fallbackTTL time.Duration
	now         func() time.Time
	entries     map[string]googleX509KeyCacheEntry
}

type googleX509KeyCacheEntry struct {
	keys      map[string]*rsa.PublicKey
	expiresAt time.Time
}

type resolvedInstanceConfig struct {
	managed                   subprocess.InitializeBridgeManagedInstance
	instanceID                string
	listenAddr                string
	webhookPath               string
	apiBaseURL                string
	tokenURL                  string
	mode                      string
	credentials               serviceAccountCredentials
	projectNumber             string
	directIssuer              string
	directCertsURL            string
	pubsubAudience            string
	pubsubIssuer              string
	pubsubCertsURL            string
	pubsubServiceAccountEmail string
	dmPolicy                  bridgepkg.BridgeDMPolicy
	allowUserIDs              map[string]struct{}
	allowUsernames            map[string]struct{}
	pairedUserIDs             map[string]struct{}
	pairedUsernames           map[string]struct{}
	dedup                     *bridgesdk.DedupCache
	rateLimiter               *bridgesdk.FixedWindowRateLimiter
	inFlightLimiter           *bridgesdk.InFlightLimiter
	batcher                   *bridgesdk.InboundBatcher
	configError               error
	initialDegradation        *bridgepkg.BridgeDegradation
	initialStatus             bridgepkg.BridgeStatus
}

type gchatWebhookProbe struct {
	Subscription string           `json:"subscription,omitempty"`
	Message      gchatPubSubInner `json:"message"`
	Chat         *json.RawMessage `json:"chat,omitempty"`
}

type gchatPubSubPushMessage struct {
	Message      gchatPubSubInner `json:"message"`
	Subscription string           `json:"subscription,omitempty"`
}

type gchatPubSubInner struct {
	Data        string            `json:"data,omitempty"`
	MessageID   string            `json:"messageId,omitempty"`
	PublishTime string            `json:"publishTime,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
}

type gchatWorkspaceEventNotification struct {
	Subscription   string         `json:"subscription"`
	TargetResource string         `json:"target_resource"`
	EventType      string         `json:"event_type"`
	EventTime      string         `json:"event_time"`
	Message        *gchatMessage  `json:"message,omitempty"`
	Reaction       *gchatReaction `json:"reaction,omitempty"`
}

type gchatEvent struct {
	Chat *struct {
		User           *gchatUser `json:"user,omitempty"`
		EventTime      string     `json:"eventTime,omitempty"`
		MessagePayload *struct {
			Space   gchatSpace   `json:"space"`
			Message gchatMessage `json:"message"`
		} `json:"messagePayload,omitempty"`
		AddedToSpacePayload *struct {
			Space gchatSpace `json:"space"`
		} `json:"addedToSpacePayload,omitempty"`
		RemovedFromSpacePayload *struct {
			Space gchatSpace `json:"space"`
		} `json:"removedFromSpacePayload,omitempty"`
		ButtonClickedPayload *struct {
			Space   gchatSpace   `json:"space"`
			Message gchatMessage `json:"message"`
			User    gchatUser    `json:"user"`
		} `json:"buttonClickedPayload,omitempty"`
	} `json:"chat,omitempty"`
	CommonEventObject *struct {
		InvokedFunction string            `json:"invokedFunction,omitempty"`
		Parameters      map[string]string `json:"parameters,omitempty"`
	} `json:"commonEventObject,omitempty"`
}

type gchatMessage struct {
	Name                  string                      `json:"name"`
	Text                  string                      `json:"text,omitempty"`
	ArgumentText          string                      `json:"argumentText,omitempty"`
	FormattedText         string                      `json:"formattedText,omitempty"`
	CreateTime            string                      `json:"createTime,omitempty"`
	Sender                gchatUser                   `json:"sender"`
	Space                 *gchatSpace                 `json:"space,omitempty"`
	Thread                *gchatThread                `json:"thread,omitempty"`
	Attachment            []gchatAttachment           `json:"attachment,omitempty"`
	Annotations           []gchatAnnotation           `json:"annotations,omitempty"`
	QuotedMessageMetadata *gchatQuotedMessageMetadata `json:"quotedMessageMetadata,omitempty"`
}

type gchatQuotedMessageMetadata struct {
	Name                  string                      `json:"name,omitempty"`
	LastUpdateTime        string                      `json:"lastUpdateTime,omitempty"`
	QuoteType             string                      `json:"quoteType,omitempty"`
	QuotedMessageSnapshot *gchatQuotedMessageSnapshot `json:"quotedMessageSnapshot,omitempty"`
}

type gchatQuotedMessageSnapshot struct {
	Text          string `json:"text,omitempty"`
	FormattedText string `json:"formattedText,omitempty"`
	Sender        string `json:"sender,omitempty"`
}

type gchatSpace struct {
	Name                string `json:"name"`
	Type                string `json:"type,omitempty"`
	SpaceType           string `json:"spaceType,omitempty"`
	DisplayName         string `json:"displayName,omitempty"`
	SingleUserBotDM     bool   `json:"singleUserBotDm,omitempty"`
	SpaceThreadingState string `json:"spaceThreadingState,omitempty"`
}

type gchatThread struct {
	Name string `json:"name,omitempty"`
}

type gchatUser struct {
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Type        string `json:"type,omitempty"`
	Email       string `json:"email,omitempty"`
}

type gchatAttachment struct {
	Name        string `json:"name,omitempty"`
	ContentName string `json:"contentName,omitempty"`
	ContentType string `json:"contentType,omitempty"`
	DownloadURI string `json:"downloadUri,omitempty"`
}

type gchatAnnotation struct {
	Type        string `json:"type,omitempty"`
	StartIndex  int    `json:"startIndex,omitempty"`
	Length      int    `json:"length,omitempty"`
	UserMention *struct {
		User gchatUser `json:"user"`
		Type string    `json:"type,omitempty"`
	} `json:"userMention,omitempty"`
}

type gchatReaction struct {
	Name  string `json:"name,omitempty"`
	Emoji *struct {
		Unicode string `json:"unicode,omitempty"`
	} `json:"emoji,omitempty"`
	User *gchatUser `json:"user,omitempty"`
}

type gchatUserIdentity struct {
	ID          string
	Username    string
	DisplayName string
}

type gchatMappedInbound struct {
	Envelope bridgepkg.InboundMessageEnvelope
	Direct   bool
	User     gchatUserIdentity
}

type gchatAPI interface {
	ValidateAuth(context.Context) error
	CreateMessage(context.Context, gchatCreateMessageRequest) (*gchatSentMessage, error)
	UpdateMessage(context.Context, gchatUpdateMessageRequest) (*gchatSentMessage, error)
	DeleteMessage(context.Context, string) error
	GetMessage(context.Context, string) (*gchatMessage, error)
}

type gchatHTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type validatedGChatURL string

type gchatBotClient struct {
	cfg                   resolvedInstanceConfig
	httpClient            gchatHTTPDoer
	reportResponseCleanup func(error)

	mu          sync.Mutex
	cachedToken string
	tokenExpiry time.Time
}

type gchatCreateMessageRequest struct {
	SpaceName  string
	ThreadName string
	Text       string
}

type gchatUpdateMessageRequest struct {
	MessageName string
	Text        string
}

type gchatSentMessage struct {
	Name   string       `json:"name,omitempty"`
	Thread *gchatThread `json:"thread,omitempty"`
	Space  *gchatSpace  `json:"space,omitempty"`
}

type gchatTokenResponse struct {
	AccessToken string `json:"access_token,omitempty"`
	ExpiresIn   int    `json:"expires_in,omitempty"`
	TokenType   string `json:"token_type,omitempty"`
}

type gchatGoogleErrorEnvelope struct {
	Error struct {
		Code    int    `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
		Status  string `json:"status,omitempty"`
	} `json:"error"`
}

type googleDirectClaims struct {
	jwt.RegisteredClaims
	Email string `json:"email,omitempty"`
}

type googleOIDCClaims struct {
	jwt.RegisteredClaims
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
}

type gchatThreadRef struct {
	SpaceName  string
	ThreadName string
	IsDM       bool
}

//nolint:funlen // Construction keeps the provider's declarative runtime wiring visible in one place.
func newGChatProvider(stderr io.Writer) (*gchatProvider, error) {
	if stderr == nil {
		stderr = io.Discard
	}

	provider := &gchatProvider{
		stderr:  stderr,
		markers: bridgesdk.NewAdapterMarkers(providerGchatKey, stderr),
		now:     func() time.Time { return time.Now().UTC() },
		parents: bridgesdk.NewParentMessageCache(0),
		routes: bridgesdk.NewRouteTable(func(config resolvedInstanceConfig) []string {
			return []string{config.webhookPath}
		}),
		deliveries: bridgesdk.NewDeliveryStateStore[deliveryState](),
		apiClients: make(map[string]cachedGChatAPIClient),
	}
	provider.apiFactory = provider.apiForConfig
	lifecycle, err := bridgesdk.NewProviderLifecycle(bridgesdk.ProviderLifecycleConfig{
		ProviderName: providerGchatKey,
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
		ReadHeaderTimeout: gchatWebhookReadHeaderTimeout,
		IdleTimeout:       gchatWebhookIdleTimeout,
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
			Name:    providerGchatKey,
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

func (p *gchatProvider) apiForConfig(cfg *resolvedInstanceConfig) gchatAPI {
	cacheKey := gchatAPIClientCacheKey(cfg)
	instanceID := strings.TrimSpace(cfg.instanceID)

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.apiClients == nil {
		p.apiClients = make(map[string]cachedGChatAPIClient)
	}
	if cached, ok := p.apiClients[instanceID]; ok && cached.key == cacheKey {
		return cached.client
	}
	client := newGChatBotClient(cfg, p.markers)
	p.apiClients[instanceID] = cachedGChatAPIClient{key: cacheKey, client: client}
	return client
}

func gchatAPIClientCacheKey(cfg *resolvedInstanceConfig) string {
	privateKeyHash := sha256.Sum256([]byte(strings.TrimSpace(cfg.credentials.PrivateKey)))
	return strings.Join([]string{
		strings.TrimSpace(cfg.instanceID),
		normalizeURL(cfg.apiBaseURL),
		normalizeURL(cfg.tokenURL),
		strings.TrimSpace(cfg.credentials.ClientEmail),
		fmt.Sprintf("%x", privateKeyHash),
	}, "\x00")
}

func (p *gchatProvider) serve(stdin io.Reader, stdout io.Writer) error {
	return p.lifecycle.Serve(context.Background(), p.sdk, stdin, stdout)
}

func (p *gchatProvider) stopResources() {
	p.closeAllGChatProgressDispatchers()
	batchers := make(map[*bridgesdk.InboundBatcher]struct{})
	routes := p.routes.Snapshot()
	for id := range routes {
		cfg := routes[id]
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

func (p *gchatProvider) reconcileInstanceConfigs(
	ctx context.Context,
	session *bridgesdk.Session,
	managed []subprocess.InitializeBridgeManagedInstance,
) []resolvedInstanceConfig {
	batchersToClose := make(map[*bridgesdk.InboundBatcher]struct{})
	removedInstanceIDs := make([]string, 0)
	reconciler := bridgesdk.ManagedConfigReconciler[resolvedInstanceConfig]{
		Routes:   p.routes,
		Resolve:  p.resolveInstanceConfig,
		Prepare:  p.prepareGChatConfigs,
		Finalize: p.finalizeGChatConfigs,
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
			removedInstanceIDs = append(removedInstanceIDs, config.instanceID)
			return nil
		},
		OnPublish: func() {
			closeGChatInboundBatchers(batchersToClose)
			p.reconcileGChatAPIClients(p.routes.Snapshot(), removedInstanceIDs)
		},
	}
	configs, err := reconciler.Reconcile(ctx, session, managed)
	if err != nil {
		p.setLastError(err)
		return nil
	}
	return configs
}

func (p *gchatProvider) finalizeGChatConfigs(
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

func (p *gchatProvider) prepareGChatConfigs(
	_ context.Context,
	_ *bridgesdk.Session,
	configs []resolvedInstanceConfig,
) ([]resolvedInstanceConfig, error) {
	if len(configs) == 0 {
		return configs, nil
	}
	configs, requestedListen := prepareGChatConfigConstraints(configs)
	p.applyGChatListenErrors(configs, requestedListen)
	p.mu.Lock()
	p.listenAddr = requestedListen
	p.mu.Unlock()
	return configs, nil
}

func prepareGChatConfigConstraints(
	configs []resolvedInstanceConfig,
) ([]resolvedInstanceConfig, string) {
	requestedListen := strings.TrimSpace(os.Getenv(gchatListenAddrEnv))
	usedPaths := make(map[string]int, len(configs))

	for idx := range configs {
		requestedListen = applyGChatListenConstraint(&configs[idx], requestedListen)
		applyGChatWebhookPathConflict(&configs[idx], configs[:idx], usedPaths)
	}
	return configs, requestedListen
}

func applyGChatListenConstraint(cfg *resolvedInstanceConfig, requestedListen string) string {
	if cfg == nil || cfg.listenAddr == "" {
		return requestedListen
	}
	if requestedListen == "" {
		return cfg.listenAddr
	}
	if requestedListen != cfg.listenAddr && cfg.configError == nil {
		cfg.configError = fmt.Errorf(
			"gchat: instance %q configured incompatible listen_addr %q (runtime uses %q)",
			cfg.instanceID,
			cfg.listenAddr,
			requestedListen,
		)
	}
	return requestedListen
}

func applyGChatWebhookPathConflict(
	cfg *resolvedInstanceConfig,
	configs []resolvedInstanceConfig,
	usedPaths map[string]int,
) {
	if cfg == nil || cfg.webhookPath == "" {
		return
	}
	if ownerIdx, ok := usedPaths[cfg.webhookPath]; ok {
		owner := configs[ownerIdx].instanceID
		conflictErr := fmt.Errorf(
			"gchat: webhook path %q is shared by %q and %q",
			cfg.webhookPath,
			owner,
			cfg.instanceID,
		)
		if configs[ownerIdx].configError == nil {
			configs[ownerIdx].configError = conflictErr
		}
		if cfg.configError == nil {
			cfg.configError = conflictErr
		}
		return
	}
	usedPaths[cfg.webhookPath] = len(configs)
}

func (p *gchatProvider) applyGChatListenErrors(configs []resolvedInstanceConfig, requestedListen string) {
	if requestedListen == "" {
		for idx := range configs {
			if configs[idx].configError == nil {
				configs[idx].configError = errors.New("gchat: webhook listen address is required")
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

func (p *gchatProvider) reconcileGChatAPIClients(
	configs map[string]resolvedInstanceConfig,
	removedInstanceIDs []string,
) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for instanceID := range configs {
		cfg := configs[instanceID]
		if cached, ok := p.apiClients[instanceID]; ok && cached.key != gchatAPIClientCacheKey(&cfg) {
			delete(p.apiClients, instanceID)
		}
	}
	for _, instanceID := range removedInstanceIDs {
		delete(p.apiClients, instanceID)
	}
}

func closeGChatInboundBatchers(batchers map[*bridgesdk.InboundBatcher]struct{}) {
	for batcher := range batchers {
		batcher.Close()
	}
}

func (p *gchatProvider) resolveInstanceConfig(
	session *bridgesdk.Session,
	managed subprocess.InitializeBridgeManagedInstance,
) resolvedInstanceConfig {
	cfg := gchatProviderConfig{}
	if len(managed.Instance.ProviderConfig) > 0 {
		if err := json.Unmarshal(managed.Instance.ProviderConfig, &cfg); err != nil {
			return resolvedInstanceConfig{
				managed:     managed,
				instanceID:  managed.Instance.ID,
				configError: fmt.Errorf("gchat: decode provider_config for %q: %w", managed.Instance.ID, err),
			}
		}
	}

	credentials, err := resolveGChatCredentials(session, managed)
	if err != nil {
		return resolvedInstanceConfig{
			managed:     managed,
			instanceID:  managed.Instance.ID,
			configError: err,
		}
	}
	resolved, err := p.newResolvedGChatConfig(managed, cfg, credentials, session)
	if err != nil {
		resolved.configError = err
		return resolved
	}
	if err := validateResolvedGChatConfig(&resolved, cfg.Mode); err != nil {
		resolved.configError = err
		return resolved
	}
	if err := p.attachGChatBatcher(&resolved, cfg.Batching); err != nil {
		resolved.configError = err
		return resolved
	}
	return resolved
}

func resolveGChatCredentials(
	session *bridgesdk.Session,
	managed subprocess.InitializeBridgeManagedInstance,
) (serviceAccountCredentials, error) {
	credentialsJSON, _ := session.Cache().BoundSecretValue(managed.Instance.ID, "credentials_json")
	credentials := serviceAccountCredentials{}
	if strings.TrimSpace(credentialsJSON) == "" {
		return credentials, nil
	}
	if err := json.Unmarshal([]byte(credentialsJSON), &credentials); err != nil {
		return serviceAccountCredentials{}, fmt.Errorf(
			"gchat: decode credentials_json for %q: %w",
			managed.Instance.ID,
			err,
		)
	}
	return credentials, nil
}

func (p *gchatProvider) newResolvedGChatConfig(
	managed subprocess.InitializeBridgeManagedInstance,
	cfg gchatProviderConfig,
	credentials serviceAccountCredentials,
	session *bridgesdk.Session,
) (resolvedInstanceConfig, error) {
	projectNumber, _ := session.Cache().BoundSecretValue(managed.Instance.ID, "project_number")
	directCertsURL, pubsubCertsURL, err := resolveGChatVerificationURLs(cfg)
	if err != nil {
		return resolvedInstanceConfig{
			managed:    managed,
			instanceID: strings.TrimSpace(managed.Instance.ID),
		}, err
	}

	mode := normalizeGChatMode(cfg.Mode)
	if mode == "" {
		mode = gchatModeDirect
	}

	resolved := resolvedInstanceConfig{
		managed:    managed,
		instanceID: strings.TrimSpace(managed.Instance.ID),
		listenAddr: firstNonEmpty(
			cfg.Webhook.ListenAddr,
			strings.TrimSpace(os.Getenv(gchatListenAddrEnv)),
		),
		webhookPath: normalizeWebhookPath(
			firstNonEmpty(cfg.Webhook.Path, "/gchat/"+strings.TrimSpace(managed.Instance.ID)),
		),
		apiBaseURL: normalizeURL(
			firstNonEmpty(
				strings.TrimSpace(os.Getenv(gchatAPIBaseEnv)),
				gchatDefaultAPIBaseURL,
			),
		),
		tokenURL: normalizeURL(
			firstNonEmpty(
				strings.TrimSpace(os.Getenv(gchatTokenURLEnv)),
				gchatDefaultAuthEndpointURL,
			),
		),
		mode:                      mode,
		credentials:               credentials,
		projectNumber:             strings.TrimSpace(projectNumber),
		directIssuer:              firstNonEmpty(cfg.Verification.DirectIssuer, gchatDefaultDirectIssuer),
		directCertsURL:            directCertsURL,
		pubsubAudience:            strings.TrimSpace(cfg.Verification.PubSubAudience),
		pubsubIssuer:              firstNonEmpty(cfg.Verification.PubSubIssuer, gchatDefaultPubSubIssuerURL),
		pubsubCertsURL:            pubsubCertsURL,
		pubsubServiceAccountEmail: strings.TrimSpace(cfg.Verification.PubSubServiceAccount),
		dmPolicy:                  managed.Instance.DMPolicy.Normalize(),
		allowUserIDs:              buildIdentitySet(cfg.DM.AllowUserIDs),
		allowUsernames:            buildIdentitySet(cfg.DM.AllowUsernames),
		pairedUserIDs:             buildIdentitySet(cfg.DM.PairedUserIDs),
		pairedUsernames:           buildIdentitySet(cfg.DM.PairedUsernames),
		dedup:                     bridgesdk.NewDedupCache(5*time.Minute, 4000),
		rateLimiter:               bridgesdk.NewFixedWindowRateLimiter(200, time.Minute),
		inFlightLimiter:           bridgesdk.NewInFlightLimiter(24),
	}
	if resolved.dmPolicy == "" {
		resolved.dmPolicy = bridgepkg.BridgeDMPolicyOpen
	}
	return resolved, nil
}

func resolveGChatVerificationURLs(cfg gchatProviderConfig) (string, string, error) {
	directCertsURL, directErr := resolveAllowedGoogleURLOverride(
		strings.TrimSpace(os.Getenv(gchatDirectCertsEnv)),
		cfg.Verification.DirectCertsURL,
		gchatDefaultDirectCertsURL,
		"provider_config.verification.direct_certs_url",
		"www.googleapis.com",
	)
	if directErr != nil {
		return "", "", directErr
	}
	pubsubCertsURL, pubsubErr := resolveAllowedGoogleURLOverride(
		strings.TrimSpace(os.Getenv(gchatPubSubCertsEnv)),
		cfg.Verification.PubSubCertsURL,
		gchatDefaultPubSubCertsURL,
		"provider_config.verification.pubsub_certs_url",
		"www.googleapis.com",
	)
	if pubsubErr != nil {
		return "", "", pubsubErr
	}
	return directCertsURL, pubsubCertsURL, nil
}

func validateResolvedGChatConfig(resolved *resolvedInstanceConfig, configuredMode string) error {
	if resolved == nil {
		return errors.New("gchat: resolved config is required")
	}
	apiBaseErr := validateGChatEndpointURL(resolved.apiBaseURL)
	tokenURLErr := validateGChatEndpointURL(resolved.tokenURL)
	switch {
	case resolved.webhookPath == "":
		return errors.New("gchat: webhook path is required")
	case resolved.apiBaseURL == "":
		return errors.New("gchat: api base url is required")
	case resolved.tokenURL == "":
		return errors.New("gchat: oauth token url is required")
	case apiBaseErr != nil:
		return apiBaseErr
	case tokenURLErr != nil:
		return tokenURLErr
	case !validGChatMode(resolved.mode):
		return fmt.Errorf("gchat: unsupported provider_config.mode %q", configuredMode)
	case modeUsesDirectIngress(resolved.mode) && strings.TrimSpace(resolved.projectNumber) == "":
		return fmt.Errorf("gchat: project_number secret binding is required for mode %q", resolved.mode)
	case modeUsesDirectIngress(resolved.mode) && resolved.directCertsURL == "":
		return errors.New("gchat: direct certs url is required")
	case modeUsesPubSubIngress(resolved.mode) && resolved.pubsubAudience == "":
		return fmt.Errorf(
			"gchat: provider_config.verification.pubsub_audience is required for mode %q",
			resolved.mode,
		)
	case modeUsesPubSubIngress(resolved.mode) && resolved.pubsubServiceAccountEmail == "":
		return fmt.Errorf(
			"gchat: provider_config.verification.pubsub_service_account_email is required for mode %q",
			resolved.mode,
		)
	case modeUsesPubSubIngress(resolved.mode) && resolved.pubsubCertsURL == "":
		return errors.New("gchat: pubsub certs url is required")
	default:
		return nil
	}
}

func (p *gchatProvider) attachGChatBatcher(
	resolved *resolvedInstanceConfig,
	cfg struct {
		DelayMS        int `json:"delay_ms,omitempty"`
		SplitDelayMS   int `json:"split_delay_ms,omitempty"`
		SplitThreshold int `json:"split_threshold,omitempty"`
	},
) error {
	if resolved == nil || cfg.DelayMS <= 0 {
		return nil
	}
	batcher, err := bridgesdk.NewInboundBatcher(bridgesdk.InboundBatcherConfig{
		Context: context.Background(),
		Delay:   time.Duration(cfg.DelayMS) * time.Millisecond,
		SplitDelay: func() time.Duration {
			if cfg.SplitDelayMS <= 0 {
				return time.Duration(cfg.DelayMS) * time.Millisecond
			}
			return time.Duration(cfg.SplitDelayMS) * time.Millisecond
		}(),
		SplitThreshold: cfg.SplitThreshold,
		Dispatch: func(ctx context.Context, batch bridgesdk.InboundBatch) error {
			return p.dispatchInboundBatch(ctx, resolved.instanceID, batch)
		},
		Now: func() time.Time { return p.now() },
	})
	if err != nil {
		return err
	}
	resolved.batcher = batcher
	return nil
}

func (p *gchatProvider) determineInitialState(
	ctx context.Context,
	cfg *resolvedInstanceConfig,
) (bridgepkg.BridgeStatus, *bridgepkg.BridgeDegradation, error) {
	if cfg == nil {
		return bridgepkg.BridgeStatusError, nil, errors.New("gchat: config is required")
	}
	if cfg.configError != nil {
		return bridgepkg.BridgeStatusDegraded, &bridgepkg.BridgeDegradation{
			Reason:  bridgepkg.BridgeDegradationReasonTenantConfigInvalid,
			Message: cfg.configError.Error(),
		}, cfg.configError
	}
	if strings.TrimSpace(cfg.credentials.ClientEmail) == "" || strings.TrimSpace(cfg.credentials.PrivateKey) == "" {
		err := errors.New("gchat: credentials_json secret binding is required")
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

func (p *gchatProvider) startServer(listenAddr string) error {
	if err := p.http.Start(listenAddr); err != nil {
		return fmt.Errorf("gchat: %w", err)
	}
	p.markers.RecordListen(p.http.Address())
	return nil
}

func (p *gchatProvider) serveWebhookHTTP(w http.ResponseWriter, r *http.Request) {
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
			return verifyGChatWebhookBearer(ctx, req, body, &cfg)
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

func (p *gchatProvider) handleWebhookRequest(
	w http.ResponseWriter,
	r *http.Request,
	cfg *resolvedInstanceConfig,
	request bridgesdk.WebhookRequest,
) error {
	ctx := context.Background()
	if r != nil && r.Context() != nil {
		ctx = r.Context()
	}
	shape := detectGChatWebhookShape(request.Body)
	switch shape {
	case gchatModePubSub:
		if !modeUsesPubSubIngress(cfg.mode) {
			return writeWebhookJSON(w, map[string]any{"ignored": true})
		}
		return p.handlePubSubWebhook(ctx, w, cfg, request)
	case gchatModeDirect:
		if !modeUsesDirectIngress(cfg.mode) {
			return writeWebhookJSON(w, map[string]any{"ignored": true})
		}
		return p.handleDirectWebhook(ctx, w, cfg, request)
	default:
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: "invalid google chat webhook payload"}
	}
}

func (p *gchatProvider) handleDirectWebhook(
	ctx context.Context,
	w http.ResponseWriter,
	cfg *resolvedInstanceConfig,
	request bridgesdk.WebhookRequest,
) error {
	event := gchatEvent{}
	if err := json.Unmarshal(request.Body, &event); err != nil {
		return &bridgesdk.HTTPError{
			StatusCode: http.StatusBadRequest,
			Message:    "invalid google chat direct webhook payload",
		}
	}

	if event.Chat == nil {
		return writeWebhookJSON(w, map[string]any{})
	}
	if item, ok, err := mapDirectActionEvent(event, cfg.managed, request.ReceivedAt); err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
	} else if ok {
		if cfg.dedup.Mark(item.Envelope.IdempotencyKey) {
			return writeWebhookJSON(w, map[string]any{})
		}
		if allowGChatDirectMessage(cfg, item.User, item.Direct) {
			if err := p.dispatchInboundEnvelope(ctx, cfg.instanceID, item.Envelope); err != nil {
				return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
			}
		}
		return writeWebhookJSON(w, map[string]any{})
	}
	if item, ok, err := mapDirectMessageEvent(event, cfg.managed, request.ReceivedAt, nil); err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
	} else if ok {
		if cfg.dedup.Mark(item.Envelope.IdempotencyKey) {
			return writeWebhookJSON(w, map[string]any{})
		}
		if !allowGChatDirectMessage(cfg, item.User, item.Direct) {
			return writeWebhookJSON(w, map[string]any{})
		}
		message := event.Chat.MessagePayload.Message
		applyGChatReplyContext(&item.Envelope, message, p.parents)
		if err := item.Envelope.Validate(); err != nil {
			return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
		}
		if cfg.batcher != nil && shouldBatchGChatMessage(event.Chat.MessagePayload.Message) {
			if err := cfg.batcher.Enqueue(item.Envelope); err != nil {
				return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
			}
		} else {
			if err := p.dispatchInboundEnvelope(ctx, cfg.instanceID, item.Envelope); err != nil {
				return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
			}
		}
		rememberGChatQuotedParent(p.parents, item.Envelope, message)
		p.parents.RememberEnvelope(item.Envelope)
	}
	return writeWebhookJSON(w, map[string]any{})
}

func (p *gchatProvider) handlePubSubWebhook(
	ctx context.Context,
	w http.ResponseWriter,
	cfg *resolvedInstanceConfig,
	request bridgesdk.WebhookRequest,
) error {
	push := gchatPubSubPushMessage{}
	if err := json.Unmarshal(request.Body, &push); err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: "invalid google chat pubsub payload"}
	}
	notification, err := decodePubSubMessage(push)
	if err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
	}

	switch {
	case notification.Message != nil:
		item, mapErr := mapPubSubMessageEvent(notification, cfg.managed, request.ReceivedAt, nil)
		if mapErr != nil {
			return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: mapErr.Error()}
		}
		if cfg.dedup.Mark(item.Envelope.IdempotencyKey) {
			return writeWebhookJSON(w, map[string]any{providerSuccessKey: true})
		}
		if !allowGChatDirectMessage(cfg, item.User, item.Direct) {
			return writeWebhookJSON(w, map[string]any{providerSuccessKey: true})
		}
		applyGChatReplyContext(&item.Envelope, *notification.Message, p.parents)
		if err := item.Envelope.Validate(); err != nil {
			return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
		}
		if cfg.batcher != nil && shouldBatchGChatMessage(*notification.Message) {
			if err := cfg.batcher.Enqueue(item.Envelope); err != nil {
				return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
			}
		} else {
			if err := p.dispatchInboundEnvelope(ctx, cfg.instanceID, item.Envelope); err != nil {
				return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
			}
		}
		rememberGChatQuotedParent(p.parents, item.Envelope, *notification.Message)
		p.parents.RememberEnvelope(item.Envelope)
	case notification.Reaction != nil:
		item, mapErr := mapPubSubReactionEvent(ctx, p.apiFactory(cfg), notification, cfg.managed, request.ReceivedAt)
		if mapErr != nil {
			return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: mapErr.Error()}
		}
		if cfg.dedup.Mark(item.Envelope.IdempotencyKey) {
			return writeWebhookJSON(w, map[string]any{providerSuccessKey: true})
		}
		if !allowGChatDirectMessage(cfg, item.User, item.Direct) {
			return writeWebhookJSON(w, map[string]any{providerSuccessKey: true})
		}
		if err := p.dispatchInboundEnvelope(ctx, cfg.instanceID, item.Envelope); err != nil {
			return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
		}
	}
	return writeWebhookJSON(w, map[string]any{providerSuccessKey: true})
}

func (p *gchatProvider) dispatchInboundBatch(
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
		attachments := make([]bridgepkg.MessageAttachment, 0)
		for _, item := range batch.Items {
			if text := strings.TrimSpace(item.Content.Text); text != "" {
				parts = append(parts, text)
			}
			attachments = append(attachments, item.Attachments...)
		}
		merged.Content.Text = strings.Join(parts, "\n")
		merged.Attachments = attachments
		merged.IdempotencyKey = fmt.Sprintf("%s:batch:%d", merged.IdempotencyKey, len(batch.Items))
	}
	return p.dispatchInboundEnvelope(ctx, bridgeInstanceID, merged)
}

func (p *gchatProvider) dispatchInboundEnvelope(
	ctx context.Context,
	bridgeInstanceID string,
	envelope bridgepkg.InboundMessageEnvelope,
) error {
	session := p.currentSession()
	if session == nil {
		return errors.New("gchat: runtime session is not initialized")
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

func (p *gchatProvider) configForInstance(instanceID string) (resolvedInstanceConfig, error) {
	cfg, ok := p.routes.Get(instanceID)
	if !ok {
		return resolvedInstanceConfig{}, fmt.Errorf("gchat: delivery targeted unmanaged instance %q", instanceID)
	}
	return cfg, nil
}

func (p *gchatProvider) waitForInstanceConfig(
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

func (p *gchatProvider) configForPath(path string) (resolvedInstanceConfig, bool) {
	normalizedPath := normalizeWebhookPath(path)
	configs := p.routes.ByPath(normalizedPath)
	if len(configs) != 1 || configs[0].configError != nil {
		return resolvedInstanceConfig{}, false
	}
	return configs[0], true
}

func (p *gchatProvider) currentSession() *bridgesdk.Session {
	return p.lifecycle.Session()
}

func (p *gchatProvider) setLastError(err error) {
	if err == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastError = err.Error()
}

func (p *gchatProvider) clearLastError() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastError = ""
}

func resolveGChatDeliveryTarget(event bridgepkg.DeliveryEvent) (gchatResolvedTarget, error) {
	if thread := firstNonEmpty(
		strings.TrimSpace(event.DeliveryTarget.ThreadID),
		strings.TrimSpace(event.RoutingKey.ThreadID),
	); thread != "" {
		if decoded, err := decodeGChatThreadID(thread); err == nil && strings.TrimSpace(decoded.SpaceName) != "" {
			return gchatResolvedTarget{
				SpaceName:  strings.TrimSpace(decoded.SpaceName),
				ThreadName: strings.TrimSpace(decoded.ThreadName),
			}, nil
		}
	}

	spaceName := firstNonEmpty(
		strings.TrimSpace(event.DeliveryTarget.PeerID),
		strings.TrimSpace(event.DeliveryTarget.GroupID),
		strings.TrimSpace(event.RoutingKey.PeerID),
		strings.TrimSpace(event.RoutingKey.GroupID),
	)
	if spaceName == "" {
		return gchatResolvedTarget{}, errors.New("gchat: delivery target requires peer_id or group_id")
	}
	return gchatResolvedTarget{SpaceName: spaceName}, nil
}

func decodePubSubMessage(push gchatPubSubPushMessage) (gchatWorkspaceEventNotification, error) {
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(push.Message.Data))
	if err != nil {
		return gchatWorkspaceEventNotification{}, fmt.Errorf("gchat: decode pubsub payload: %w", err)
	}
	payload := struct {
		Message  *gchatMessage  `json:"message,omitempty"`
		Reaction *gchatReaction `json:"reaction,omitempty"`
	}{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return gchatWorkspaceEventNotification{}, fmt.Errorf("gchat: decode pubsub notification payload: %w", err)
	}
	attributes := push.Message.Attributes
	return gchatWorkspaceEventNotification{
		Subscription:   strings.TrimSpace(push.Subscription),
		TargetResource: strings.TrimSpace(attributes["ce-subject"]),
		EventType:      strings.TrimSpace(attributes["ce-type"]),
		EventTime:      firstNonEmpty(attributes["ce-time"], push.Message.PublishTime),
		Message:        payload.Message,
		Reaction:       payload.Reaction,
	}, nil
}

func detectGChatWebhookShape(body []byte) string {
	probe := gchatWebhookProbe{}
	if err := json.Unmarshal(body, &probe); err != nil {
		return ""
	}
	if strings.TrimSpace(probe.Subscription) != "" && strings.TrimSpace(probe.Message.Data) != "" {
		return gchatModePubSub
	}
	if probe.Chat != nil {
		return gchatModeDirect
	}
	return ""
}

func normalizeReceivedAt(fallback time.Time, value string) time.Time {
	if strings.TrimSpace(value) == "" {
		if fallback.IsZero() {
			return time.Now().UTC()
		}
		return fallback.UTC()
	}
	if parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value)); err == nil {
		return parsed.UTC()
	}
	if fallback.IsZero() {
		return time.Now().UTC()
	}
	return fallback.UTC()
}

func normalizeGChatText(message gchatMessage) string {
	text := firstNonEmpty(message.ArgumentText, message.Text, message.FormattedText)
	return strings.TrimSpace(text)
}

func normalizeGChatAttachments(items []gchatAttachment) []bridgepkg.MessageAttachment {
	attachments := make([]bridgepkg.MessageAttachment, 0, len(items))
	for _, item := range items {
		attachment := bridgepkg.MessageAttachment{
			ID:       strings.TrimSpace(item.Name),
			Name:     strings.TrimSpace(firstNonEmpty(item.ContentName, item.Name)),
			MIMEType: strings.TrimSpace(item.ContentType),
			URL:      strings.TrimSpace(item.DownloadURI),
		}
		if attachment.ID == "" && attachment.Name == "" && attachment.MIMEType == "" && attachment.URL == "" {
			continue
		}
		attachments = append(attachments, attachment)
	}
	if len(attachments) == 0 {
		return nil
	}
	return attachments
}

func gchatSender(user gchatUser) bridgepkg.MessageSender {
	displayName := strings.TrimSpace(user.DisplayName)
	username := normalizeUsername(firstNonEmpty(strings.TrimSpace(user.Email), displayName))
	if username == "" {
		username = normalizeUsername(strings.TrimPrefix(strings.TrimSpace(user.Name), "users/"))
	}
	return bridgepkg.MessageSender{
		ID:          strings.TrimSpace(user.Name),
		Username:    username,
		DisplayName: displayName,
	}
}

func isDirectSpace(space gchatSpace) bool {
	return strings.EqualFold(strings.TrimSpace(space.Type), "DM") ||
		strings.EqualFold(strings.TrimSpace(space.SpaceType), "DIRECT_MESSAGE")
}

func isBotUser(user gchatUser) bool {
	return strings.EqualFold(strings.TrimSpace(user.Type), "BOT")
}

func threadNameForMessage(message gchatMessage, direct bool) string {
	if direct {
		return ""
	}
	if message.Thread != nil && strings.TrimSpace(message.Thread.Name) != "" {
		return strings.TrimSpace(message.Thread.Name)
	}
	return strings.TrimSpace(message.Name)
}

func paramValue(params map[string]string, key string) string {
	if len(params) == 0 {
		return ""
	}
	return strings.TrimSpace(params[key])
}

func derefUser(user *gchatUser) gchatUser {
	if user == nil {
		return gchatUser{}
	}
	return *user
}

func reactionEmoji(reaction *gchatReaction) string {
	if reaction == nil || reaction.Emoji == nil {
		return ""
	}
	return strings.TrimSpace(reaction.Emoji.Unicode)
}

func normalizeGChatEmoji(value string) string {
	return strings.TrimSpace(value)
}

func extractReactionMessageName(reactionName string) string {
	matches := reactionMessagePattern.FindStringSubmatch(strings.TrimSpace(reactionName))
	if len(matches) != 2 {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

func encodeGChatThreadID(ref gchatThreadRef) string {
	space := strings.TrimSpace(ref.SpaceName)
	thread := strings.TrimSpace(ref.ThreadName)
	if space == "" {
		return ""
	}
	encodedThread := ""
	if thread != "" {
		encodedThread = ":" + base64.RawURLEncoding.EncodeToString([]byte(thread))
	}
	dmSuffix := ""
	if ref.IsDM {
		dmSuffix = ":dm"
	}
	return "gchat:" + space + encodedThread + dmSuffix
}

func decodeGChatThreadID(value string) (gchatThreadRef, error) {
	trimmed := strings.TrimSpace(value)
	isDM := strings.HasSuffix(trimmed, ":dm")
	if isDM {
		trimmed = strings.TrimSuffix(trimmed, ":dm")
	}
	parts := strings.Split(trimmed, ":")
	if len(parts) < 2 || parts[0] != providerGchatKey {
		return gchatThreadRef{}, errors.New("gchat: invalid thread id")
	}
	ref := gchatThreadRef{
		SpaceName: strings.TrimSpace(parts[1]),
		IsDM:      isDM,
	}
	if len(parts) > 2 && strings.TrimSpace(parts[2]) != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(parts[2]))
		if err != nil {
			return gchatThreadRef{}, err
		}
		ref.ThreadName = string(decoded)
	}
	return ref, nil
}
