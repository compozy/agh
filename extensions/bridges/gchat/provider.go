package main

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/bridgesdk"
	extensioncontract "github.com/pedronauck/agh/internal/extension/contract"
	"github.com/pedronauck/agh/internal/subprocess"
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

	rpcCodeNotInitialized = -32003
)

var gchatTokenURLEnv = strings.Join([]string{"AGH", "BRIDGE", "GCHAT", "TOKEN", "URL"}, "_")

var reactionMessagePattern = regexp.MustCompile(`^(spaces/[^/]+/messages/[^/]+)/reactions/[^/]+$`)

var defaultGoogleX509KeyCache = newGoogleX509KeyCache(
	&http.Client{Timeout: gchatCertFetchTimeout},
	gchatCertCacheFallbackTTL,
	func() time.Time { return time.Now().UTC() },
)

type gchatProvider struct {
	sdk     *bridgesdk.Runtime
	stderr  io.Writer
	env     markerEnv
	now     func() time.Time
	session *bridgesdk.Session

	mu             sync.RWMutex
	lastError      string
	server         *http.Server
	serverAddr     string
	listenAddr     string
	routes         map[string]resolvedInstanceConfig
	deliveries     map[string]deliveryState
	reportedStatus map[string]bridgepkg.BridgeStatus
	apiFactory     func(resolvedInstanceConfig) gchatAPI

	stopCh   chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

type deliveryState struct {
	LastSeq                int64
	RemoteMessageID        string
	ReplaceRemoteMessageID string
}

type gchatProviderConfig struct {
	APIBaseURL string `json:"api_base_url,omitempty"`
	TokenURL   string `json:"oauth_token_url,omitempty"`
	Mode       string `json:"mode,omitempty"`
	Webhook    struct {
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
	Name          string            `json:"name"`
	Text          string            `json:"text,omitempty"`
	ArgumentText  string            `json:"argumentText,omitempty"`
	FormattedText string            `json:"formattedText,omitempty"`
	CreateTime    string            `json:"createTime,omitempty"`
	Sender        gchatUser         `json:"sender"`
	Space         *gchatSpace       `json:"space,omitempty"`
	Thread        *gchatThread      `json:"thread,omitempty"`
	Attachment    []gchatAttachment `json:"attachment,omitempty"`
	Annotations   []gchatAnnotation `json:"annotations,omitempty"`
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
	cfg        resolvedInstanceConfig
	httpClient gchatHTTPDoer

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

type gchatResolvedTarget struct {
	SpaceName  string
	ThreadName string
}

type gchatThreadRef struct {
	SpaceName  string
	ThreadName string
	IsDM       bool
}

func newGChatProvider(stderr io.Writer) (*gchatProvider, error) {
	if stderr == nil {
		stderr = io.Discard
	}

	provider := &gchatProvider{
		stderr:         stderr,
		env:            markerEnvFromProcess(),
		now:            func() time.Time { return time.Now().UTC() },
		routes:         make(map[string]resolvedInstanceConfig),
		deliveries:     make(map[string]deliveryState),
		reportedStatus: make(map[string]bridgepkg.BridgeStatus),
		stopCh:         make(chan struct{}),
	}
	provider.apiFactory = func(cfg resolvedInstanceConfig) gchatAPI {
		return &gchatBotClient{
			cfg: cfg,
			httpClient: &http.Client{
				Timeout: 10 * time.Second,
			},
		}
	}

	sdkRuntime, err := bridgesdk.NewRuntime(bridgesdk.RuntimeConfig{
		ExtensionInfo: subprocess.InitializeExtensionInfo{
			Name:    "gchat",
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

func (p *gchatProvider) serve(stdin io.Reader, stdout io.Writer) error {
	p.reportSideEffectError(
		"write start marker",
		appendMarkerLine(p.env.startsPath, fmt.Sprintf("pid=%d", os.Getpid())),
	)
	return p.sdk.Serve(context.Background(), stdin, stdout)
}

func (p *gchatProvider) handleInitialize(_ context.Context, session *bridgesdk.Session) error {
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

func (p *gchatProvider) afterInitialize(session *bridgesdk.Session) {
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
		if reportErr := p.reportState(
			ctx,
			session,
			cfg.instanceID,
			status,
			degradation,
		); reportErr != nil &&
			ownershipErr == nil {
			ownershipErr = reportErr
		}
	}

	if ownershipErr != nil {
		p.setLastError(ownershipErr)
	} else {
		p.clearLastError()
	}
}

func (p *gchatProvider) handleBridgesDeliver(
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
	if cfg.configError != nil {
		err = cfg.configError
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

	ack, state, err := executeGChatDelivery(
		ctx,
		p.apiFactory(cfg),
		request,
		p.deliveryState(cfg.instanceID, request.Event.DeliveryID),
	)
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
	if err := p.reportReadyIfNeeded(ctx, session, cfg.instanceID); err != nil {
		p.setLastError(err)
	} else {
		p.clearLastError()
	}

	marker.Ack = &ack
	p.reportSideEffectError("write delivery marker", appendJSONLine(p.env.deliveryPath, marker))
	return ack, nil
}

func (p *gchatProvider) healthCheck() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if strings.TrimSpace(p.lastError) == "" {
		return nil
	}
	return errors.New(strings.TrimSpace(p.lastError))
}

func (p *gchatProvider) handleShutdown(
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
	p.mu.Unlock()
	if server != nil {
		if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			p.reportSideEffectError("shutdown gchat webhook server", err)
			p.setLastError(err)
		}
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

func (p *gchatProvider) stop() {
	p.stopOnce.Do(func() {
		close(p.stopCh)
		batchersToClose := make(map[*bridgesdk.InboundBatcher]struct{})
		p.mu.Lock()
		for id := range p.routes {
			cfg := p.routes[id]
			if cfg.batcher != nil {
				batchersToClose[cfg.batcher] = struct{}{}
				cfg.batcher = nil
				p.routes[id] = cfg
			}
		}
		p.mu.Unlock()
		closeGChatInboundBatchers(batchersToClose)
	})
}

func (p *gchatProvider) syncOwnedInstances(
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

func (p *gchatProvider) getOwnedInstance(
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

func (p *gchatProvider) reportState(
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

func (p *gchatProvider) reportReadyIfNeeded(
	ctx context.Context,
	session *bridgesdk.Session,
	bridgeInstanceID string,
) error {
	bridgeInstanceID = strings.TrimSpace(bridgeInstanceID)
	p.mu.RLock()
	status := p.reportedStatus[bridgeInstanceID]
	p.mu.RUnlock()
	if status == bridgepkg.BridgeStatusReady {
		return nil
	}
	return p.reportState(ctx, session, bridgeInstanceID, bridgepkg.BridgeStatusReady, nil)
}

func (p *gchatProvider) ingestBridgeMessage(
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

func (p *gchatProvider) retryHostCall(ctx context.Context, fn func(context.Context) error) error {
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

func (p *gchatProvider) reconcileInstanceConfigs(
	ctx context.Context,
	session *bridgesdk.Session,
	managed []subprocess.InitializeBridgeManagedInstance,
) []resolvedInstanceConfig {
	if len(managed) == 0 {
		p.mu.Lock()
		p.routes = make(map[string]resolvedInstanceConfig)
		p.mu.Unlock()
		return nil
	}

	configs, requestedListen := p.collectGChatConfigs(session, managed)
	p.applyGChatListenErrors(configs, requestedListen)
	nextRoutes := buildGChatRouteMap(configs)
	closeGChatInboundBatchers(p.swapGChatRoutes(nextRoutes, requestedListen))

	for idx := range configs {
		status, degradation, err := p.determineInitialState(ctx, &configs[idx])
		if err != nil {
			p.setLastError(err)
		}
		configs[idx].initialStatus = status
		configs[idx].initialDegradation = degradation
	}
	return configs
}

func (p *gchatProvider) collectGChatConfigs(
	session *bridgesdk.Session,
	managed []subprocess.InitializeBridgeManagedInstance,
) ([]resolvedInstanceConfig, string) {
	configs := make([]resolvedInstanceConfig, 0, len(managed))
	requestedListen := strings.TrimSpace(os.Getenv(gchatListenAddrEnv))
	usedPaths := make(map[string]string, len(managed))

	for _, item := range managed {
		cfg := p.resolveInstanceConfig(session, item)
		requestedListen = applyGChatListenConstraint(&cfg, requestedListen)
		applyGChatWebhookPathConflict(&cfg, usedPaths)
		configs = append(configs, cfg)
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

func applyGChatWebhookPathConflict(cfg *resolvedInstanceConfig, usedPaths map[string]string) {
	if cfg == nil || cfg.webhookPath == "" {
		return
	}
	if owner, ok := usedPaths[cfg.webhookPath]; ok && cfg.configError == nil {
		cfg.configError = fmt.Errorf(
			"gchat: webhook path %q is shared by %q and %q",
			cfg.webhookPath,
			owner,
			cfg.instanceID,
		)
	}
	usedPaths[cfg.webhookPath] = cfg.instanceID
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

func buildGChatRouteMap(configs []resolvedInstanceConfig) map[string]resolvedInstanceConfig {
	nextRoutes := make(map[string]resolvedInstanceConfig, len(configs))
	for idx := range configs {
		cfg := configs[idx]
		nextRoutes[cfg.instanceID] = cfg
	}
	return nextRoutes
}

func (p *gchatProvider) swapGChatRoutes(
	nextRoutes map[string]resolvedInstanceConfig,
	requestedListen string,
) map[*bridgesdk.InboundBatcher]struct{} {
	batchersToClose := make(map[*bridgesdk.InboundBatcher]struct{})
	p.mu.Lock()
	defer p.mu.Unlock()

	existing := p.routes
	for instanceID := range nextRoutes {
		cfg := nextRoutes[instanceID]
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
			firstNonEmpty(strings.TrimSpace(os.Getenv(gchatAPIBaseEnv)), gchatDefaultAPIBaseURL),
		),
		tokenURL: normalizeURL(
			firstNonEmpty(
				strings.TrimSpace(os.Getenv(gchatTokenURLEnv)),
				strings.TrimSpace(credentials.TokenURI),
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
	if err := p.apiFactory(*cfg).ValidateAuth(ctx); err != nil {
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
	p.mu.RLock()
	server := p.server
	currentListen := p.listenAddr
	p.mu.RUnlock()
	if server != nil {
		if currentListen != "" && currentListen != strings.TrimSpace(listenAddr) {
			return fmt.Errorf("gchat: runtime already listening on %q, cannot switch to %q", currentListen, listenAddr)
		}
		return nil
	}

	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", strings.TrimSpace(listenAddr))
	if err != nil {
		return fmt.Errorf("gchat: listen %q: %w", listenAddr, err)
	}

	httpServer := &http.Server{
		Handler:           http.HandlerFunc(p.serveWebhookHTTP),
		ReadHeaderTimeout: gchatWebhookReadHeaderTimeout,
		IdleTimeout:       gchatWebhookIdleTimeout,
	}
	actualAddr := ln.Addr().String()

	p.mu.Lock()
	p.server = httpServer
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
	if item, ok, err := mapDirectMessageEvent(event, cfg.managed, request.ReceivedAt); err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
	} else if ok {
		if cfg.dedup.Mark(item.Envelope.IdempotencyKey) {
			return writeWebhookJSON(w, map[string]any{})
		}
		if !allowGChatDirectMessage(cfg, item.User, item.Direct) {
			return writeWebhookJSON(w, map[string]any{})
		}
		if cfg.batcher != nil {
			if err := cfg.batcher.Enqueue(item.Envelope); err != nil {
				return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
			}
		} else {
			if err := p.dispatchInboundEnvelope(ctx, cfg.instanceID, item.Envelope); err != nil {
				return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
			}
		}
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
		item, mapErr := mapPubSubMessageEvent(notification, cfg.managed, request.ReceivedAt)
		if mapErr != nil {
			return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: mapErr.Error()}
		}
		if cfg.dedup.Mark(item.Envelope.IdempotencyKey) {
			return writeWebhookJSON(w, map[string]any{"success": true})
		}
		if !allowGChatDirectMessage(cfg, item.User, item.Direct) {
			return writeWebhookJSON(w, map[string]any{"success": true})
		}
		if cfg.batcher != nil {
			if err := cfg.batcher.Enqueue(item.Envelope); err != nil {
				return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
			}
		} else {
			if err := p.dispatchInboundEnvelope(ctx, cfg.instanceID, item.Envelope); err != nil {
				return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
			}
		}
	case notification.Reaction != nil:
		item, mapErr := mapPubSubReactionEvent(ctx, p.apiFactory(*cfg), notification, cfg.managed, request.ReceivedAt)
		if mapErr != nil {
			return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: mapErr.Error()}
		}
		if cfg.dedup.Mark(item.Envelope.IdempotencyKey) {
			return writeWebhookJSON(w, map[string]any{"success": true})
		}
		if !allowGChatDirectMessage(cfg, item.User, item.Direct) {
			return writeWebhookJSON(w, map[string]any{"success": true})
		}
		if err := p.dispatchInboundEnvelope(ctx, cfg.instanceID, item.Envelope); err != nil {
			return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
		}
	}
	return writeWebhookJSON(w, map[string]any{"success": true})
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
	if err := p.reportReadyIfNeeded(ctx, session, cfg.instanceID); err != nil {
		p.setLastError(err)
	} else {
		p.clearLastError()
	}
	return nil
}

func (p *gchatProvider) configForInstance(instanceID string) (resolvedInstanceConfig, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	cfg, ok := p.routes[strings.TrimSpace(instanceID)]
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

func (p *gchatProvider) configForPath(path string) (resolvedInstanceConfig, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for instanceID := range p.routes {
		cfg := p.routes[instanceID]
		if cfg.webhookPath == normalizeWebhookPath(path) {
			return cfg, true
		}
	}
	return resolvedInstanceConfig{}, false
}

func (p *gchatProvider) currentSession() *bridgesdk.Session {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.session
}

func (p *gchatProvider) deliveryState(instanceID string, deliveryID string) deliveryState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.deliveries[deliveryStateKey(instanceID, deliveryID)]
}

func (p *gchatProvider) storeDeliveryState(instanceID string, deliveryID string, state deliveryState) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.deliveries[deliveryStateKey(instanceID, deliveryID)] = state
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

func (p *gchatProvider) reportSideEffectError(action string, err error) {
	reportSideEffectError(p.stderr, action, err)
}

func executeGChatDelivery(
	ctx context.Context,
	api gchatAPI,
	request bridgepkg.DeliveryRequest,
	state deliveryState,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	event := request.Event
	if event.Seq <= state.LastSeq {
		return bridgepkg.DeliveryAck{}, state, fmt.Errorf(
			"gchat: out-of-order delivery seq %d after %d",
			event.Seq,
			state.LastSeq,
		)
	}

	switch {
	case isGChatDeleteEvent(event):
		return executeGChatDelete(ctx, api, request, state)
	case shouldPostGChatMessage(event, state, request):
		return executeGChatCreate(ctx, api, event, state)
	default:
		return executeGChatUpdate(ctx, api, request, state)
	}
}

func isGChatDeleteEvent(event bridgepkg.DeliveryEvent) bool {
	return event.Operation.Normalize() == bridgepkg.DeliveryOperationDelete ||
		normalizeDeliveryEventType(event.EventType) == bridgepkg.DeliveryEventTypeDelete
}

func executeGChatDelete(
	ctx context.Context,
	api gchatAPI,
	request bridgepkg.DeliveryRequest,
	state deliveryState,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	event := request.Event
	messageName := gchatRemoteMessageIDFromRequest(request, state)
	if messageName == "" {
		return bridgepkg.DeliveryAck{}, state, errors.New("gchat: delete delivery requires a remote message id")
	}
	if err := api.DeleteMessage(ctx, messageName); err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	state.LastSeq = event.Seq
	state.ReplaceRemoteMessageID = messageName
	ack := bridgepkg.DeliveryAck{
		DeliveryID:             event.DeliveryID,
		Seq:                    event.Seq,
		RemoteMessageID:        messageName,
		ReplaceRemoteMessageID: messageName,
	}
	return ack, state, ack.ValidateFor(event)
}

func executeGChatCreate(
	ctx context.Context,
	api gchatAPI,
	event bridgepkg.DeliveryEvent,
	state deliveryState,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	text, err := gchatDeliveryText(event)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	target, err := resolveGChatDeliveryTarget(event)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	message, err := api.CreateMessage(ctx, gchatCreateMessageRequest{
		SpaceName:  target.SpaceName,
		ThreadName: target.ThreadName,
		Text:       text,
	})
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	if strings.TrimSpace(message.Name) == "" {
		return bridgepkg.DeliveryAck{}, state, &bridgesdk.TransientError{
			Err: errors.New("gchat: create message response omitted name"),
		}
	}
	state.LastSeq = event.Seq
	state.RemoteMessageID = strings.TrimSpace(message.Name)
	if event.Seq > 1 {
		state.ReplaceRemoteMessageID = state.RemoteMessageID
	}
	ack := bridgepkg.DeliveryAck{
		DeliveryID:             event.DeliveryID,
		Seq:                    event.Seq,
		RemoteMessageID:        state.RemoteMessageID,
		ReplaceRemoteMessageID: state.ReplaceRemoteMessageID,
	}
	return ack, state, ack.ValidateFor(event)
}

func executeGChatUpdate(
	ctx context.Context,
	api gchatAPI,
	request bridgepkg.DeliveryRequest,
	state deliveryState,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	event := request.Event
	text, err := gchatDeliveryText(event)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	messageName := gchatRemoteMessageIDFromRequest(request, state)
	if messageName == "" {
		return bridgepkg.DeliveryAck{}, state, errors.New("gchat: edit delivery requires a remote message id")
	}
	updated, err := api.UpdateMessage(ctx, gchatUpdateMessageRequest{
		MessageName: messageName,
		Text:        text,
	})
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	if strings.TrimSpace(updated.Name) == "" {
		updated.Name = messageName
	}
	state.LastSeq = event.Seq
	state.RemoteMessageID = strings.TrimSpace(updated.Name)
	state.ReplaceRemoteMessageID = messageName
	ack := bridgepkg.DeliveryAck{
		DeliveryID:             event.DeliveryID,
		Seq:                    event.Seq,
		RemoteMessageID:        state.RemoteMessageID,
		ReplaceRemoteMessageID: state.ReplaceRemoteMessageID,
	}
	return ack, state, ack.ValidateFor(event)
}

func gchatDeliveryText(event bridgepkg.DeliveryEvent) (string, error) {
	text := strings.TrimSpace(event.Content.Text)
	if text == "" {
		return "", &bridgesdk.PermanentError{Err: errors.New("gchat: text delivery content is required")}
	}
	return text, nil
}

func gchatRemoteMessageIDFromRequest(request bridgepkg.DeliveryRequest, state deliveryState) string {
	messageName := firstNonEmpty(referenceRemoteMessageID(request.Event.Reference), state.RemoteMessageID)
	if messageName == "" && request.Snapshot != nil {
		return strings.TrimSpace(request.Snapshot.RemoteMessageID)
	}
	return messageName
}

func shouldPostGChatMessage(
	event bridgepkg.DeliveryEvent,
	state deliveryState,
	request bridgepkg.DeliveryRequest,
) bool {
	if normalizeDeliveryEventType(event.EventType) == bridgepkg.DeliveryEventTypeStart {
		return true
	}
	if normalizeDeliveryEventType(event.EventType) == bridgepkg.DeliveryEventTypeResume {
		if request.Snapshot == nil {
			return strings.TrimSpace(state.RemoteMessageID) == ""
		}
		return strings.TrimSpace(request.Snapshot.RemoteMessageID) == ""
	}
	return strings.TrimSpace(state.RemoteMessageID) == ""
}

func allowGChatDirectMessage(cfg *resolvedInstanceConfig, user gchatUserIdentity, direct bool) bool {
	if cfg == nil {
		return false
	}
	if !direct {
		return true
	}
	switch cfg.dmPolicy.Normalize() {
	case "", bridgepkg.BridgeDMPolicyOpen:
		return true
	case bridgepkg.BridgeDMPolicyAllowlist:
		return gchatIdentityAllowed(cfg.allowUserIDs, cfg.allowUsernames, user)
	case bridgepkg.BridgeDMPolicyPairing:
		if gchatIdentityAllowed(cfg.pairedUserIDs, cfg.pairedUsernames, user) {
			return true
		}
		return gchatIdentityAllowed(cfg.allowUserIDs, cfg.allowUsernames, user)
	default:
		return false
	}
}

func gchatIdentityAllowed(ids map[string]struct{}, usernames map[string]struct{}, user gchatUserIdentity) bool {
	if len(ids) == 0 && len(usernames) == 0 {
		return false
	}
	if _, ok := ids[normalizeUsername(user.ID)]; ok {
		return true
	}
	if _, ok := usernames[normalizeUsername(firstNonEmpty(user.Username, user.DisplayName))]; ok {
		return true
	}
	return false
}

func mapDirectMessageEvent(
	event gchatEvent,
	managed subprocess.InitializeBridgeManagedInstance,
	receivedAt time.Time,
) (gchatMappedInbound, bool, error) {
	if event.Chat == nil || event.Chat.MessagePayload == nil {
		return gchatMappedInbound{}, false, nil
	}
	message := event.Chat.MessagePayload.Message
	space := event.Chat.MessagePayload.Space
	if isBotUser(message.Sender) {
		return gchatMappedInbound{}, false, nil
	}
	item, err := mapGChatMessage(
		message,
		space,
		managed,
		receivedAt,
		"direct:"+strings.TrimSpace(message.Name),
		"direct_webhook",
	)
	return item, true, err
}

func mapDirectActionEvent(
	event gchatEvent,
	managed subprocess.InitializeBridgeManagedInstance,
	receivedAt time.Time,
) (gchatMappedInbound, bool, error) {
	actionCtx, ok := buildDirectActionContext(event)
	if !ok {
		return gchatMappedInbound{}, false, nil
	}
	if isBotUser(actionCtx.user) {
		return gchatMappedInbound{}, false, nil
	}
	item, err := buildDirectActionItem(event, managed, receivedAt, actionCtx)
	return item, true, err
}

type gchatDirectActionContext struct {
	space           gchatSpace
	message         gchatMessage
	user            gchatUser
	actionID        string
	invokedFunction string
	parameters      map[string]string
}

func buildDirectActionContext(event gchatEvent) (gchatDirectActionContext, bool) {
	if event.Chat == nil {
		return gchatDirectActionContext{}, false
	}
	button := event.Chat.ButtonClickedPayload
	invokedFunction := ""
	parameters := map[string]string(nil)
	if event.CommonEventObject != nil {
		invokedFunction = strings.TrimSpace(event.CommonEventObject.InvokedFunction)
		parameters = event.CommonEventObject.Parameters
	}
	if button == nil && invokedFunction == "" {
		return gchatDirectActionContext{}, false
	}

	ctx := gchatDirectActionContext{
		invokedFunction: invokedFunction,
		parameters:      parameters,
	}
	if button != nil {
		ctx.space = button.Space
		ctx.message = button.Message
		ctx.user = button.User
	}
	if ctx.user.Name == "" && event.Chat.User != nil {
		ctx.user = *event.Chat.User
	}
	ctx.actionID = firstNonEmpty(paramValue(parameters, "actionId"), invokedFunction)
	if ctx.actionID == "" {
		return gchatDirectActionContext{}, false
	}
	return ctx, true
}

func buildDirectActionItem(
	event gchatEvent,
	managed subprocess.InitializeBridgeManagedInstance,
	receivedAt time.Time,
	actionCtx gchatDirectActionContext,
) (gchatMappedInbound, error) {
	direct := isDirectSpace(actionCtx.space)
	threadName := firstNonEmpty(
		threadNameForMessage(actionCtx.message, direct),
		strings.TrimSpace(actionCtx.message.Name),
	)
	envelope := bridgepkg.InboundMessageEnvelope{
		BridgeInstanceID: managed.Instance.ID,
		Scope:            managed.Instance.Scope,
		WorkspaceID:      managed.Instance.WorkspaceID,
		ReceivedAt:       normalizeReceivedAt(receivedAt, event.Chat.EventTime),
		Sender:           gchatSender(actionCtx.user),
		EventFamily:      bridgepkg.InboundEventFamilyAction,
		Action: &bridgepkg.InboundAction{
			ActionID:  actionCtx.actionID,
			MessageID: strings.TrimSpace(actionCtx.message.Name),
			Value:     paramValue(actionCtx.parameters, "value"),
			TriggerID: firstNonEmpty(paramValue(actionCtx.parameters, "triggerId"), actionCtx.invokedFunction),
		},
		IdempotencyKey: fmt.Sprintf(
			"gchat:%s:action:%s:%s",
			managed.Instance.ID,
			actionCtx.actionID,
			strings.TrimSpace(actionCtx.message.Name),
		),
	}
	assignGChatRoute(&envelope, strings.TrimSpace(actionCtx.space.Name), threadName, direct)
	if metadata, err := json.Marshal(map[string]any{
		"source":       "direct_webhook",
		"space_name":   strings.TrimSpace(actionCtx.space.Name),
		"thread_name":  threadName,
		"message_name": strings.TrimSpace(actionCtx.message.Name),
	}); err == nil {
		envelope.ProviderMetadata = metadata
	}
	item := gchatMappedInbound{
		Envelope: envelope,
		Direct:   direct,
		User: gchatUserIdentity{
			ID:          envelope.Sender.ID,
			Username:    envelope.Sender.Username,
			DisplayName: envelope.Sender.DisplayName,
		},
	}
	return item, item.Envelope.Validate()
}

func mapPubSubMessageEvent(
	notification gchatWorkspaceEventNotification,
	managed subprocess.InitializeBridgeManagedInstance,
	receivedAt time.Time,
) (gchatMappedInbound, error) {
	message := notification.Message
	if message == nil {
		return gchatMappedInbound{}, errors.New("gchat: pubsub notification missing message")
	}
	space := gchatSpace{}
	if message.Space != nil {
		space = *message.Space
	}
	if space.Name == "" {
		space.Name = strings.TrimPrefix(strings.TrimSpace(notification.TargetResource), "//chat.googleapis.com/")
	}
	return mapGChatMessage(
		*message,
		space,
		managed,
		normalizeReceivedAt(receivedAt, notification.EventTime),
		"pubsub:"+firstNonEmpty(notification.EventType, strings.TrimSpace(message.Name)),
		"pubsub_workspace_events",
	)
}

func mapPubSubReactionEvent(
	ctx context.Context,
	api gchatAPI,
	notification gchatWorkspaceEventNotification,
	managed subprocess.InitializeBridgeManagedInstance,
	receivedAt time.Time,
) (gchatMappedInbound, error) {
	if notification.Reaction == nil {
		return gchatMappedInbound{}, errors.New("gchat: pubsub notification missing reaction")
	}
	route, err := resolvePubSubReactionRoute(ctx, api, notification)
	if err != nil {
		return gchatMappedInbound{}, err
	}
	item := buildPubSubReactionItem(notification, managed, receivedAt, route)
	return item, item.Envelope.Validate()
}

type gchatPubSubReactionRoute struct {
	messageName string
	spaceName   string
	threadName  string
	direct      bool
	reaction    *gchatReaction
}

func resolvePubSubReactionRoute(
	ctx context.Context,
	api gchatAPI,
	notification gchatWorkspaceEventNotification,
) (gchatPubSubReactionRoute, error) {
	reaction := notification.Reaction
	messageName := extractReactionMessageName(reaction.Name)
	if messageName == "" {
		return gchatPubSubReactionRoute{}, errors.New("gchat: reaction resource omitted message reference")
	}

	spaceName := strings.TrimPrefix(strings.TrimSpace(notification.TargetResource), "//chat.googleapis.com/")
	threadName := messageName
	direct := false
	if api != nil {
		if message, err := api.GetMessage(ctx, messageName); err == nil && message != nil {
			if message.Space != nil && strings.TrimSpace(message.Space.Name) != "" {
				spaceName = strings.TrimSpace(message.Space.Name)
				direct = isDirectSpace(*message.Space)
			}
			threadName = threadNameForMessage(*message, direct)
			if threadName == "" {
				threadName = strings.TrimSpace(message.Name)
			}
		}
	}
	if spaceName == "" {
		parts := strings.Split(messageName, "/")
		if len(parts) >= 2 {
			spaceName = parts[0] + "/" + parts[1]
		}
	}
	if threadName == "" {
		threadName = messageName
	}
	return gchatPubSubReactionRoute{
		messageName: messageName,
		spaceName:   spaceName,
		threadName:  threadName,
		direct:      direct,
		reaction:    reaction,
	}, nil
}

func assignGChatRoute(
	envelope *bridgepkg.InboundMessageEnvelope,
	spaceName string,
	threadName string,
	direct bool,
) {
	if envelope == nil {
		return
	}
	if direct {
		envelope.PeerID = strings.TrimSpace(spaceName)
	} else {
		envelope.GroupID = strings.TrimSpace(spaceName)
	}
	if direct || strings.TrimSpace(threadName) != "" {
		envelope.ThreadID = encodeGChatThreadID(gchatThreadRef{
			SpaceName:  strings.TrimSpace(spaceName),
			ThreadName: strings.TrimSpace(threadName),
			IsDM:       direct,
		})
	}
}

func buildPubSubReactionItem(
	notification gchatWorkspaceEventNotification,
	managed subprocess.InitializeBridgeManagedInstance,
	receivedAt time.Time,
	route gchatPubSubReactionRoute,
) gchatMappedInbound {
	sender := gchatSender(derefUser(route.reaction.User))
	envelope := bridgepkg.InboundMessageEnvelope{
		BridgeInstanceID: managed.Instance.ID,
		Scope:            managed.Instance.Scope,
		WorkspaceID:      managed.Instance.WorkspaceID,
		ReceivedAt:       normalizeReceivedAt(receivedAt, notification.EventTime),
		Sender:           sender,
		EventFamily:      bridgepkg.InboundEventFamilyReaction,
		Reaction: &bridgepkg.InboundReaction{
			MessageID: route.messageName,
			Emoji:     normalizeGChatEmoji(reactionEmoji(route.reaction)),
			RawEmoji:  reactionEmoji(route.reaction),
			Added:     strings.Contains(strings.ToLower(strings.TrimSpace(notification.EventType)), "created"),
		},
		IdempotencyKey: fmt.Sprintf(
			"gchat:%s:reaction:%s:%s",
			managed.Instance.ID,
			strings.TrimSpace(notification.EventType),
			strings.TrimSpace(route.reaction.Name),
		),
	}
	assignGChatRoute(&envelope, route.spaceName, route.threadName, route.direct)
	if metadata, err := json.Marshal(map[string]any{
		"source":        "pubsub_workspace_events",
		"event_type":    strings.TrimSpace(notification.EventType),
		"space_name":    route.spaceName,
		"thread_name":   route.threadName,
		"reaction_name": strings.TrimSpace(route.reaction.Name),
	}); err == nil {
		envelope.ProviderMetadata = metadata
	}

	return gchatMappedInbound{
		Envelope: envelope,
		Direct:   route.direct,
		User:     gchatUserIdentity{ID: sender.ID, Username: sender.Username, DisplayName: sender.DisplayName},
	}
}

func mapGChatMessage(
	message gchatMessage,
	space gchatSpace,
	managed subprocess.InitializeBridgeManagedInstance,
	receivedAt time.Time,
	idempotencyBase string,
	source string,
) (gchatMappedInbound, error) {
	direct := isDirectSpace(space)
	threadName := threadNameForMessage(message, direct)
	threadID := ""
	if direct || threadName != "" {
		threadID = encodeGChatThreadID(gchatThreadRef{
			SpaceName:  strings.TrimSpace(space.Name),
			ThreadName: threadName,
			IsDM:       direct,
		})
	}
	envelope := bridgepkg.InboundMessageEnvelope{
		BridgeInstanceID:  managed.Instance.ID,
		Scope:             managed.Instance.Scope,
		WorkspaceID:       managed.Instance.WorkspaceID,
		PlatformMessageID: strings.TrimSpace(message.Name),
		ReceivedAt:        normalizeReceivedAt(receivedAt, message.CreateTime),
		Sender:            gchatSender(message.Sender),
		Content: bridgepkg.MessageContent{
			Text: normalizeGChatText(message),
		},
		Attachments:    normalizeGChatAttachments(message.Attachment),
		EventFamily:    bridgepkg.InboundEventFamilyMessage,
		IdempotencyKey: fmt.Sprintf("gchat:%s:%s", managed.Instance.ID, strings.TrimSpace(idempotencyBase)),
	}
	if direct {
		envelope.PeerID = strings.TrimSpace(space.Name)
	} else {
		envelope.GroupID = strings.TrimSpace(space.Name)
	}
	envelope.ThreadID = threadID
	if metadata, err := json.Marshal(map[string]any{
		"source":       strings.TrimSpace(source),
		"space_name":   strings.TrimSpace(space.Name),
		"space_type":   firstNonEmpty(strings.TrimSpace(space.SpaceType), strings.TrimSpace(space.Type)),
		"thread_name":  threadName,
		"message_name": strings.TrimSpace(message.Name),
	}); err == nil {
		envelope.ProviderMetadata = metadata
	}

	item := gchatMappedInbound{
		Envelope: envelope,
		Direct:   direct,
		User: gchatUserIdentity{
			ID:          envelope.Sender.ID,
			Username:    envelope.Sender.Username,
			DisplayName: envelope.Sender.DisplayName,
		},
	}
	return item, item.Envelope.Validate()
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

func verifyGChatWebhookBearer(ctx context.Context, req *http.Request, body []byte, cfg *resolvedInstanceConfig) error {
	if cfg == nil {
		return errors.New("gchat: config is required")
	}
	switch detectGChatWebhookShape(body) {
	case gchatModePubSub:
		if !modeUsesPubSubIngress(cfg.mode) {
			return nil
		}
		return verifyPubSubBearerToken(ctx, req, cfg)
	case gchatModeDirect:
		if !modeUsesDirectIngress(cfg.mode) {
			return nil
		}
		return verifyDirectBearerToken(ctx, req, cfg)
	default:
		return errors.New("gchat: unrecognized webhook payload shape")
	}
}

func verifyDirectBearerToken(ctx context.Context, req *http.Request, cfg *resolvedInstanceConfig) error {
	if cfg == nil {
		return errors.New("gchat: config is required")
	}
	tokenString, err := bearerToken(req)
	if err != nil {
		return err
	}
	keys, err := fetchGoogleX509Keys(ctx, cfg.directCertsURL)
	if err != nil {
		return err
	}
	claims := &googleDirectClaims{}
	parsed, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
			return nil, fmt.Errorf("gchat: unsupported signing method %q", token.Method.Alg())
		}
		return keyByTokenHeader(token.Header, keys)
	}, jwt.WithAudience(strings.TrimSpace(cfg.projectNumber)), jwt.WithLeeway(5*time.Minute))
	if err != nil {
		return fmt.Errorf("gchat: invalid direct bearer token: %w", err)
	}
	if !parsed.Valid {
		return errors.New("gchat: invalid direct bearer token")
	}
	if !issuerMatches(claims.Issuer, strings.TrimSpace(cfg.directIssuer)) {
		return fmt.Errorf("gchat: direct bearer issuer %q did not match %q", claims.Issuer, cfg.directIssuer)
	}
	return nil
}

func verifyPubSubBearerToken(ctx context.Context, req *http.Request, cfg *resolvedInstanceConfig) error {
	if cfg == nil {
		return errors.New("gchat: config is required")
	}
	tokenString, err := bearerToken(req)
	if err != nil {
		return err
	}
	keys, err := fetchGoogleX509Keys(ctx, cfg.pubsubCertsURL)
	if err != nil {
		return err
	}
	claims := &googleOIDCClaims{}
	parsed, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
			return nil, fmt.Errorf("gchat: unsupported signing method %q", token.Method.Alg())
		}
		return keyByTokenHeader(token.Header, keys)
	}, jwt.WithAudience(strings.TrimSpace(cfg.pubsubAudience)), jwt.WithLeeway(5*time.Minute))
	if err != nil {
		return fmt.Errorf("gchat: invalid pubsub bearer token: %w", err)
	}
	if !parsed.Valid {
		return errors.New("gchat: invalid pubsub bearer token")
	}
	if !issuerMatches(
		claims.Issuer,
		strings.TrimSpace(cfg.pubsubIssuer),
		"accounts.google.com",
		"https://accounts.google.com",
	) {
		return fmt.Errorf("gchat: pubsub bearer issuer %q did not match expected Google issuer", claims.Issuer)
	}
	if !strings.EqualFold(strings.TrimSpace(claims.Email), strings.TrimSpace(cfg.pubsubServiceAccountEmail)) {
		return fmt.Errorf(
			"gchat: pubsub bearer email %q did not match expected service account %q",
			claims.Email,
			cfg.pubsubServiceAccountEmail,
		)
	}
	if !claims.EmailVerified {
		return fmt.Errorf("gchat: pubsub bearer email %q is not verified", strings.TrimSpace(claims.Email))
	}
	return nil
}

func fetchGoogleX509Keys(ctx context.Context, certsURL string) (map[string]*rsa.PublicKey, error) {
	if strings.TrimSpace(certsURL) == "" {
		return nil, errors.New("gchat: certs url is required")
	}
	return defaultGoogleX509KeyCache.fetch(ctx, certsURL)
}

func newGoogleX509KeyCache(client *http.Client, fallbackTTL time.Duration, now func() time.Time) *googleX509KeyCache {
	if client == nil {
		client = &http.Client{Timeout: gchatCertFetchTimeout}
	}
	if fallbackTTL <= 0 {
		fallbackTTL = gchatCertCacheFallbackTTL
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &googleX509KeyCache{
		client:      client,
		fallbackTTL: fallbackTTL,
		now:         now,
		entries:     make(map[string]googleX509KeyCacheEntry),
	}
}

func (c *googleX509KeyCache) fetch(ctx context.Context, certsURL string) (map[string]*rsa.PublicKey, error) {
	trimmedURL := strings.TrimSpace(certsURL)
	if trimmedURL == "" {
		return nil, errors.New("gchat: certs url is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()
	if entry, ok := c.entries[trimmedURL]; ok && len(entry.keys) > 0 && now.Before(entry.expiresAt) {
		return entry.keys, nil
	}

	keys, expiresAt, err := c.fetchRemote(ctx, trimmedURL)
	if err != nil {
		if entry, ok := c.entries[trimmedURL]; ok && len(entry.keys) > 0 {
			return entry.keys, nil
		}
		return nil, err
	}
	c.entries[trimmedURL] = googleX509KeyCacheEntry{
		keys:      keys,
		expiresAt: expiresAt,
	}
	return keys, nil
}

func (c *googleX509KeyCache) fetchRemote(
	ctx context.Context,
	certsURL string,
) (map[string]*rsa.PublicKey, time.Time, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, certsURL, http.NoBody)
	if err != nil {
		return nil, time.Time{}, err
	}
	if err := validateGChatRequestURL(req); err != nil {
		return nil, time.Time{}, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, time.Time{}, classifyGChatHTTPError(
			resp.StatusCode,
			resp.Header.Get("Retry-After"),
			readResponseBody(resp.Body),
		)
	}
	certs := map[string]string{}
	if err := json.NewDecoder(resp.Body).Decode(&certs); err != nil {
		return nil, time.Time{}, err
	}
	if len(certs) == 0 {
		return nil, time.Time{}, errors.New("gchat: x509 cert document omitted keys")
	}
	keys := make(map[string]*rsa.PublicKey, len(certs))
	for keyID, pemCert := range certs {
		block, _ := pem.Decode([]byte(strings.TrimSpace(pemCert)))
		if block == nil {
			return nil, time.Time{}, fmt.Errorf("gchat: decode x509 cert %q: missing pem block", keyID)
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, time.Time{}, fmt.Errorf("gchat: parse x509 cert %q: %w", keyID, err)
		}
		publicKey, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			return nil, time.Time{}, fmt.Errorf("gchat: x509 cert %q did not contain an rsa public key", keyID)
		}
		keys[strings.TrimSpace(keyID)] = publicKey
	}
	return keys, cacheExpiryFromHeaders(c.now(), resp.Header, c.fallbackTTL), nil
}

func resolveAllowedGoogleURLOverride(
	envOverride string,
	providerOverride string,
	fallback string,
	fieldName string,
	allowedHosts ...string,
) (string, error) {
	if trimmedEnv := strings.TrimSpace(envOverride); trimmedEnv != "" {
		normalized := normalizeURL(trimmedEnv)
		return normalized, validateGChatEndpointURL(normalized)
	}
	if strings.TrimSpace(providerOverride) == "" {
		return normalizeURL(fallback), nil
	}
	normalized := normalizeURL(providerOverride)
	parsed, err := url.Parse(normalized)
	if err != nil || parsed == nil || strings.TrimSpace(parsed.Hostname()) == "" ||
		!strings.EqualFold(parsed.Scheme, "https") {
		return "", fmt.Errorf("gchat: %s must use an allowed Google https host", fieldName)
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	for _, allowedHost := range allowedHosts {
		if host == strings.ToLower(strings.TrimSpace(allowedHost)) {
			return normalized, nil
		}
	}
	return "", fmt.Errorf("gchat: %s host %q is not allowed", fieldName, parsed.Hostname())
}

func cacheExpiryFromHeaders(now time.Time, header http.Header, fallback time.Duration) time.Time {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if fallback <= 0 {
		fallback = gchatCertCacheFallbackTTL
	}
	for directive := range strings.SplitSeq(header.Get("Cache-Control"), ",") {
		part := strings.TrimSpace(directive)
		lower := strings.ToLower(part)
		if !strings.HasPrefix(lower, "max-age=") {
			continue
		}
		seconds, err := strconv.Atoi(strings.TrimSpace(lower[len("max-age="):]))
		if err == nil && seconds > 0 {
			return now.Add(time.Duration(seconds) * time.Second)
		}
	}
	return now.Add(fallback)
}

func bearerToken(req *http.Request) (string, error) {
	if req == nil {
		return "", errors.New("gchat: webhook request is required")
	}
	authz := strings.TrimSpace(req.Header.Get("Authorization"))
	if !strings.HasPrefix(strings.ToLower(authz), "bearer ") {
		return "", errors.New("gchat: bearer authorization is required")
	}
	token := strings.TrimSpace(authz[len("Bearer "):])
	if token == "" {
		return "", errors.New("gchat: bearer token is required")
	}
	return token, nil
}

func keyByTokenHeader(header map[string]any, keys map[string]*rsa.PublicKey) (*rsa.PublicKey, error) {
	keyID := firstNonEmpty(stringHeader(header, "kid"), stringHeader(header, "x5t"))
	if keyID == "" {
		return nil, errors.New("gchat: token header missing key id")
	}
	key, ok := keys[keyID]
	if !ok {
		return nil, fmt.Errorf("gchat: signing key %q not found", keyID)
	}
	return key, nil
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
	if len(parts) < 2 || parts[0] != "gchat" {
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

func (c *gchatBotClient) ValidateAuth(ctx context.Context) error {
	_, err := c.accessToken(ctx)
	return err
}

func (c *gchatBotClient) CreateMessage(ctx context.Context, req gchatCreateMessageRequest) (*gchatSentMessage, error) {
	query := url.Values{}
	body := map[string]any{
		"text": req.Text,
	}
	if strings.TrimSpace(req.ThreadName) != "" {
		body["thread"] = map[string]string{"name": strings.TrimSpace(req.ThreadName)}
		query.Set("messageReplyOption", gchatReplyMode)
	}
	var out gchatSentMessage
	if err := c.callJSON(
		ctx,
		http.MethodPost,
		"/v1/"+strings.TrimPrefix(strings.TrimSpace(req.SpaceName), "/")+"/messages",
		query,
		body,
		&out,
	); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *gchatBotClient) UpdateMessage(ctx context.Context, req gchatUpdateMessageRequest) (*gchatSentMessage, error) {
	query := url.Values{}
	query.Set("updateMask", "text")
	var out gchatSentMessage
	if err := c.callJSON(
		ctx,
		http.MethodPut,
		"/v1/"+strings.TrimPrefix(strings.TrimSpace(req.MessageName), "/"),
		query,
		map[string]any{
			"text": req.Text,
		},
		&out,
	); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *gchatBotClient) DeleteMessage(ctx context.Context, messageName string) error {
	return c.callJSON(
		ctx,
		http.MethodDelete,
		"/v1/"+strings.TrimPrefix(strings.TrimSpace(messageName), "/"),
		nil,
		nil,
		nil,
	)
}

func (c *gchatBotClient) GetMessage(ctx context.Context, messageName string) (*gchatMessage, error) {
	var out gchatMessage
	if err := c.callJSON(
		ctx,
		http.MethodGet,
		"/v1/"+strings.TrimPrefix(strings.TrimSpace(messageName), "/"),
		nil,
		nil,
		&out,
	); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *gchatBotClient) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	if c.cachedToken != "" && c.tokenExpiry.After(time.Now().UTC().Add(30*time.Second)) {
		token := c.cachedToken
		c.mu.Unlock()
		return token, nil
	}
	c.mu.Unlock()

	privateKey, err := parseRSAPrivateKey(c.cfg.credentials.PrivateKey)
	if err != nil {
		return "", &bridgesdk.AuthError{Err: err}
	}
	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"iss":   strings.TrimSpace(c.cfg.credentials.ClientEmail),
		"sub":   strings.TrimSpace(c.cfg.credentials.ClientEmail),
		"scope": gchatBotScope,
		"aud":   strings.TrimSpace(c.cfg.tokenURL),
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(privateKey)
	if err != nil {
		return "", &bridgesdk.AuthError{Err: err}
	}

	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Set("assertion", signed)

	endpoint, err := newValidatedGChatURL(c.cfg.tokenURL)
	if err != nil {
		return "", err
	}
	req, err := newGChatRequest(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.client().Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", classifyGChatHTTPError(resp.StatusCode, resp.Header.Get("Retry-After"), readResponseBody(resp.Body))
	}
	var tokenResp gchatTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}
	if strings.TrimSpace(tokenResp.AccessToken) == "" {
		return "", &bridgesdk.AuthError{Err: errors.New("gchat: token response omitted access token")}
	}
	expiresIn := tokenResp.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	c.mu.Lock()
	c.cachedToken = strings.TrimSpace(tokenResp.AccessToken)
	c.tokenExpiry = time.Now().UTC().Add(time.Duration(expiresIn) * time.Second)
	value := c.cachedToken
	c.mu.Unlock()
	return value, nil
}

func (c *gchatBotClient) callJSON(
	ctx context.Context,
	method string,
	path string,
	query url.Values,
	payload any,
	out any,
) error {
	token, err := c.accessToken(ctx)
	if err != nil {
		return err
	}
	fullURL, err := joinGChatURL(c.cfg.apiBaseURL, path)
	if err != nil {
		return err
	}
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}
	endpoint, err := newValidatedGChatURL(fullURL)
	if err != nil {
		return err
	}
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := newGChatRequest(ctx, method, endpoint, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.client().Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return classifyGChatHTTPError(resp.StatusCode, resp.Header.Get("Retry-After"), readResponseBody(resp.Body))
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		if _, err := io.Copy(io.Discard, resp.Body); err != nil {
			return err
		}
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *gchatBotClient) client() gchatHTTPDoer {
	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return c.httpClient
}

func parseRSAPrivateKey(raw string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(raw)))
	if block == nil {
		return nil, errors.New("gchat: decode private key: missing pem block")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("gchat: parse private key: %w", err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("gchat: private key was not rsa")
	}
	return key, nil
}

func classifyGChatHTTPError(statusCode int, retryAfterHeader string, raw string) error {
	message := strings.TrimSpace(raw)
	envelope := gchatGoogleErrorEnvelope{}
	if json.Unmarshal([]byte(raw), &envelope) == nil {
		if trimmed := strings.TrimSpace(envelope.Error.Message); trimmed != "" {
			message = trimmed
		}
	}
	if message == "" {
		message = fmt.Sprintf("gchat: http %d", statusCode)
	}
	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return &bridgesdk.AuthError{Err: errors.New(message)}
	case http.StatusTooManyRequests:
		return &bridgesdk.RateLimitError{Err: errors.New(message), RetryAfter: parseRetryAfter(retryAfterHeader)}
	case http.StatusRequestTimeout, http.StatusGatewayTimeout:
		return &bridgesdk.TransientError{Err: errors.New(message)}
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusInternalServerError:
		return &bridgesdk.TransientError{Err: errors.New(message)}
	default:
		return &bridgesdk.HTTPError{
			StatusCode: statusCode,
			Message:    message,
			RetryAfter: parseRetryAfter(retryAfterHeader),
		}
	}
}

func readResponseBody(reader io.Reader) string {
	if reader == nil {
		return ""
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}

func parseRetryAfter(header string) time.Duration {
	trimmed := strings.TrimSpace(header)
	if trimmed == "" {
		return 0
	}
	seconds, err := strconv.Atoi(trimmed)
	if err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return 0
}

func stringHeader(header map[string]any, key string) string {
	value, ok := header[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func issuerMatches(issuer string, allowed ...string) bool {
	trimmed := strings.TrimSpace(issuer)
	if trimmed == "" {
		return false
	}
	for _, candidate := range allowed {
		if strings.EqualFold(trimmed, strings.TrimSpace(candidate)) {
			return true
		}
	}
	return false
}

func writeWebhookJSON(w http.ResponseWriter, body any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(body)
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

func joinGChatURL(base string, path string) (string, error) {
	baseURL, err := url.Parse(strings.TrimSpace(base))
	if err != nil {
		return "", err
	}
	ref, err := url.Parse(strings.TrimSpace(path))
	if err != nil {
		return "", err
	}
	return baseURL.ResolveReference(ref).String(), nil
}

func newValidatedGChatURL(value string) (validatedGChatURL, error) {
	normalized := normalizeURL(value)
	if err := validateGChatEndpointURL(normalized); err != nil {
		return "", err
	}
	return validatedGChatURL(normalized), nil
}

func newGChatRequest(
	ctx context.Context,
	method string,
	endpoint validatedGChatURL,
	body io.Reader,
) (*http.Request, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	parsed, err := url.ParseRequestURI(string(endpoint))
	if err != nil {
		return nil, err
	}
	req := &http.Request{
		Method: method,
		URL:    parsed,
		Header: make(http.Header),
	}
	if body != nil {
		req.Body = io.NopCloser(body)
	}
	req = req.WithContext(ctx)
	if err := validateGChatRequestURL(req); err != nil {
		return nil, err
	}
	return req, nil
}

func validateGChatRequestURL(req *http.Request) error {
	if req == nil || req.URL == nil {
		return errors.New("gchat: request url is required")
	}
	return validateGChatEndpointURL(req.URL.String())
}

func validateGChatEndpointURL(value string) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return fmt.Errorf("gchat: parse api url: %w", err)
	}
	if parsed == nil || strings.TrimSpace(parsed.Hostname()) == "" {
		return errors.New("gchat: api url host is required")
	}
	if strings.EqualFold(parsed.Scheme, "https") {
		return nil
	}
	if !strings.EqualFold(parsed.Scheme, "http") {
		return fmt.Errorf("gchat: api url scheme %q is not allowed", parsed.Scheme)
	}

	host := strings.TrimSpace(parsed.Hostname())
	if strings.EqualFold(host, "localhost") {
		return nil
	}
	ip := net.ParseIP(host)
	if ip != nil && ip.IsLoopback() {
		return nil
	}
	return fmt.Errorf("gchat: insecure api host %q is not allowed", host)
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
	if len(result) == 0 {
		return nil
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

func referenceRemoteMessageID(reference *bridgepkg.DeliveryMessageReference) string {
	if reference == nil {
		return ""
	}
	return strings.TrimSpace(reference.RemoteMessageID)
}

func deliveryStateKey(instanceID string, deliveryID string) string {
	return strings.TrimSpace(instanceID) + ":" + strings.TrimSpace(deliveryID)
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
	return rpcErr.Code == rpcCodeNotInitialized ||
		strings.EqualFold(strings.TrimSpace(rpcErr.Message), "Not initialized")
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

func normalizeGChatMode(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func validGChatMode(value string) bool {
	switch normalizeGChatMode(value) {
	case gchatModeDirect, gchatModePubSub, gchatModeHybrid:
		return true
	default:
		return false
	}
}

func modeUsesDirectIngress(mode string) bool {
	switch normalizeGChatMode(mode) {
	case gchatModeDirect, gchatModeHybrid:
		return true
	default:
		return false
	}
}

func modeUsesPubSubIngress(mode string) bool {
	switch normalizeGChatMode(mode) {
	case gchatModePubSub, gchatModeHybrid:
		return true
	default:
		return false
	}
}
