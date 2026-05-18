package main

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
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
	teamsListenAddrEnv            = "AGH_BRIDGE_TEAMS_LISTEN_ADDR"
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
	reportedHealth map[string]string
	userContexts   map[string]teamsUserContext
	apiFactory     func(resolvedInstanceConfig) teamsAPI

	stopCh   chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

type deliveryState struct {
	LastSeq                int64
	RemoteMessageID        string
	ReplaceRemoteMessageID string
}

type teamsDeliveryStateLookup func(deliveryID string) (deliveryState, bool)

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
	ServiceURL string `json:"service_url,omitempty"`
	Webhook    struct {
		ListenAddr string `json:"listen_addr,omitempty"`
		Path       string `json:"path,omitempty"`
	} `json:"webhook"`
	Auth struct {
		OpenIDMetadataURL string `json:"openid_metadata_url,omitempty"`
		TokenURL          string `json:"token_url,omitempty"`
	} `json:"auth"`
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
	cfg        resolvedInstanceConfig
	httpClient *http.Client

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

func newTeamsProvider(stderr io.Writer) (*teamsProvider, error) {
	if stderr == nil {
		stderr = io.Discard
	}

	provider := &teamsProvider{
		stderr:         stderr,
		env:            markerEnvFromProcess(),
		now:            func() time.Time { return time.Now().UTC() },
		routes:         make(map[string]resolvedInstanceConfig),
		deliveries:     make(map[string]deliveryState),
		reportedStatus: make(map[string]bridgepkg.BridgeStatus),
		reportedHealth: make(map[string]string),
		userContexts:   make(map[string]teamsUserContext),
		stopCh:         make(chan struct{}),
	}
	provider.apiFactory = func(cfg resolvedInstanceConfig) teamsAPI {
		return &teamsBotClient{
			cfg: cfg,
			httpClient: &http.Client{
				Timeout: 10 * time.Second,
			},
		}
	}

	sdkRuntime, err := bridgesdk.NewRuntime(bridgesdk.RuntimeConfig{
		ExtensionInfo: subprocess.InitializeExtensionInfo{
			Name:    "teams",
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

func (p *teamsProvider) serve(stdin io.Reader, stdout io.Writer) error {
	p.reportSideEffectError(
		"write start marker",
		appendMarkerLine(p.env.startsPath, fmt.Sprintf("pid=%d", os.Getpid())),
	)
	return p.sdk.Serve(context.Background(), stdin, stdout)
}

func (p *teamsProvider) handleInitialize(_ context.Context, session *bridgesdk.Session) error {
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

func (p *teamsProvider) afterInitialize(session *bridgesdk.Session) {
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
	for _, cfg := range configs {
		status := cfg.initialStatus
		degradation := cfg.initialDegradation
		if status == "" {
			status = bridgepkg.BridgeStatusReady
		}
		if stateErr := p.reportState(
			ctx,
			session,
			cfg.instanceID,
			status,
			degradation,
		); stateErr != nil &&
			ownershipErr == nil {
			ownershipErr = stateErr
		}
	}
	if ownershipErr != nil {
		p.setLastError(ownershipErr)
	} else {
		p.clearLastError()
	}
}

func (p *teamsProvider) handleBridgesDeliver(
	ctx context.Context,
	session *bridgesdk.Session,
	request bridgepkg.DeliveryRequest,
) (bridgepkg.DeliveryAck, error) {
	marker := deliveryMarker{
		PID:     os.Getpid(),
		Request: request,
	}

	cfg, err := p.waitForInstanceConfig(
		strings.TrimSpace(request.Event.BridgeInstanceID),
		500*time.Millisecond,
	)
	if err != nil {
		marker.Error = err.Error()
		p.reportSideEffectError(
			"write failed delivery marker",
			appendJSONLine(p.env.deliveryPath, marker),
		)
		p.setLastError(err)
		return bridgepkg.DeliveryAck{}, err
	}

	if shouldCrashOnce(p.env.crashOncePath) {
		p.reportSideEffectError(
			"write pre-crash delivery marker",
			appendJSONLine(p.env.deliveryPath, marker),
		)
		p.reportSideEffectError(
			"write crash marker",
			writeJSONFile(p.env.crashOncePath, map[string]any{
				"crashed":            true,
				"pid":                os.Getpid(),
				"delivery_id":        strings.TrimSpace(request.Event.DeliveryID),
				"bridge_instance_id": cfg.instanceID,
			}),
		)
		os.Exit(23)
	}

	api := p.apiFactory(cfg)
	ack, state, err := executeTeamsDelivery(
		ctx,
		api,
		cfg,
		request,
		p.deliveryState(cfg.instanceID, request.Event.DeliveryID),
		func(deliveryID string) (deliveryState, bool) {
			state := p.deliveryState(cfg.instanceID, deliveryID)
			return state, strings.TrimSpace(state.RemoteMessageID) != ""
		},
		p.userContext,
	)
	if err != nil {
		marker.Error = err.Error()
		p.reportSideEffectError(
			"write failed delivery marker",
			appendJSONLine(p.env.deliveryPath, marker),
		)
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

func (p *teamsProvider) healthCheck() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for instanceID, status := range p.reportedStatus {
		normalized := status.Normalize()
		if normalized == "" || normalized == bridgepkg.BridgeStatusReady {
			continue
		}
		message := strings.TrimSpace(p.reportedHealth[instanceID])
		if message != "" {
			return fmt.Errorf("teams: bridge instance %s is %s: %s", instanceID, normalized, message)
		}
		return fmt.Errorf("teams: bridge instance %s is %s", instanceID, normalized)
	}
	if strings.TrimSpace(p.lastError) == "" {
		return nil
	}
	return errors.New(strings.TrimSpace(p.lastError))
}

func (p *teamsProvider) handleShutdown(
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
	var shutdownErr error
	if server != nil {
		if err := server.Shutdown(shutdownCtx); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			shutdownErr = fmt.Errorf("teams: shutdown webhook server: %w", err)
			p.setLastError(shutdownErr)
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
	return shutdownErr
}

func (p *teamsProvider) stop() {
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

func (p *teamsProvider) syncOwnedInstances(
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

func (p *teamsProvider) getOwnedInstance(
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

func (p *teamsProvider) reportState(
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
		p.reportSideEffectError(
			"write failed state marker",
			appendJSONLine(p.env.statePath, stateMarker{
				BridgeInstanceID: strings.TrimSpace(bridgeInstanceID),
				Status:           status,
				Error:            err.Error(),
			}),
		)
		return err
	}

	p.mu.Lock()
	instanceID := strings.TrimSpace(bridgeInstanceID)
	p.reportedStatus[instanceID] = result.Status.Normalize()
	if health := bridgeHealthMessage(result.Degradation); health != "" {
		p.reportedHealth[instanceID] = health
	} else {
		delete(p.reportedHealth, instanceID)
	}
	p.mu.Unlock()
	p.reportSideEffectError("write state marker", appendJSONLine(p.env.statePath, stateMarker{
		BridgeInstanceID: result.ID,
		Status:           result.Status,
		Instance:         *result,
	}))
	return nil
}

func (p *teamsProvider) reportReadyIfNeeded(
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

func (p *teamsProvider) ingestBridgeMessage(
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

func (p *teamsProvider) retryHostCall(ctx context.Context, fn func(context.Context) error) error {
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

func (p *teamsProvider) reconcileInstanceConfigs(
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

	configs, requestedListen := p.resolveTeamsManagedConfigs(session, managed)
	applyTeamsListenErrors(configs, requestedListen, p.startServer)
	p.swapTeamsRoutes(configs, requestedListen)
	p.populateTeamsInitialStates(ctx, configs)
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
	p.mu.RLock()
	server := p.server
	currentListen := p.listenAddr
	p.mu.RUnlock()
	if server != nil {
		if currentListen != "" && currentListen != strings.TrimSpace(listenAddr) {
			return fmt.Errorf(
				"teams: runtime already listening on %q, cannot switch to %q",
				currentListen,
				listenAddr,
			)
		}
		return nil
	}

	ln, err := listenTeamsWebhook(strings.TrimSpace(listenAddr))
	if err != nil {
		return fmt.Errorf("teams: listen %q: %w", listenAddr, err)
	}

	httpServer := &http.Server{
		Handler:           http.HandlerFunc(p.serveWebhookHTTP),
		ReadHeaderTimeout: teamsWebhookReadHeaderTimeout,
		IdleTimeout:       teamsWebhookIdleTimeout,
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
		if serveErr := httpServer.Serve(ln); serveErr != nil &&
			!errors.Is(serveErr, http.ErrServerClosed) {
			p.setLastError(serveErr)
		}
	})
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

	result, err := p.ingestBridgeMessage(ctx, session, envelope)
	if err != nil {
		p.reportSideEffectError(
			"write failed ingest marker",
			appendJSONLine(p.env.ingestPath, ingestMarker{
				Envelope: envelope,
				Error:    err.Error(),
			}),
		)
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

func (p *teamsProvider) configForInstance(instanceID string) (resolvedInstanceConfig, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	cfg, ok := p.routes[strings.TrimSpace(instanceID)]
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
	deadline := p.now().Add(timeout)
	for {
		cfg, err := p.configForInstance(instanceID)
		if err == nil {
			return cfg, nil
		}
		if timeout <= 0 || !p.now().Before(deadline) {
			return resolvedInstanceConfig{}, err
		}
		select {
		case <-time.After(10 * time.Millisecond):
		case <-p.stopCh:
			return resolvedInstanceConfig{}, err
		}
	}
}

func (p *teamsProvider) configForPath(path string) (resolvedInstanceConfig, bool) {
	normalizedPath := normalizeWebhookPath(path)
	p.mu.RLock()
	defer p.mu.RUnlock()
	var match resolvedInstanceConfig
	found := false
	for _, cfg := range p.routes {
		if cfg.webhookPath != normalizedPath || cfg.configError != nil {
			continue
		}
		if found {
			return resolvedInstanceConfig{}, false
		}
		match = cfg
		found = true
	}
	return match, found
}

func (p *teamsProvider) currentSession() *bridgesdk.Session {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.session
}

func (p *teamsProvider) deliveryState(instanceID string, deliveryID string) deliveryState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.deliveries[deliveryStateKey(instanceID, deliveryID)]
}

func (p *teamsProvider) storeDeliveryState(
	instanceID string,
	deliveryID string,
	state deliveryState,
) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.deliveries[deliveryStateKey(instanceID, deliveryID)] = state
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

func (p *teamsProvider) reportSideEffectError(action string, err error) {
	reportSideEffectError(p.stderr, action, err)
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

type mappedTeamsInbound struct {
	Envelope bridgepkg.InboundMessageEnvelope
	Direct   bool
	User     teamsUserIdentity
}

type teamsUserIdentity struct {
	ID          string
	Username    string
	DisplayName string
}

type teamsInboundContext struct {
	receivedAt         time.Time
	serviceURL         string
	conversationID     string
	direct             bool
	baseConversationID string
	threadID           string
	user               teamsUserIdentity
}

func mapTeamsActivity(
	activity teamsActivity,
	cfg resolvedInstanceConfig,
	receivedAt time.Time,
) ([]mappedTeamsInbound, error) {
	switch strings.TrimSpace(activity.Type) {
	case "message":
		if payload, ok := decodeTeamsMessageAction(activity.Value); ok {
			item, err := mapTeamsActionActivity(activity, cfg, payload, receivedAt, "message")
			if err != nil {
				return nil, err
			}
			if item.Envelope.BridgeInstanceID == "" {
				return nil, nil
			}
			return []mappedTeamsInbound{item}, nil
		}
		item, ignored, err := mapTeamsMessageActivity(activity, cfg, receivedAt)
		if err != nil {
			return nil, err
		}
		if ignored {
			return nil, nil
		}
		return []mappedTeamsInbound{item}, nil
	case "invoke":
		payload, ok := decodeTeamsInvokeAction(activity.Value)
		if !ok {
			return nil, nil
		}
		item, err := mapTeamsActionActivity(activity, cfg, payload, receivedAt, "invoke")
		if err != nil {
			return nil, err
		}
		return []mappedTeamsInbound{item}, nil
	case "messageReaction":
		return mapTeamsReactionActivity(activity, cfg, receivedAt)
	case "conversationUpdate", "installationUpdate":
		return nil, nil
	default:
		return nil, nil
	}
}

func mapTeamsMessageActivity(
	activity teamsActivity,
	cfg resolvedInstanceConfig,
	receivedAt time.Time,
) (mappedTeamsInbound, bool, error) {
	conversationID := strings.TrimSpace(activity.Conversation.ID)
	if conversationID == "" || strings.TrimSpace(activity.ID) == "" {
		return mappedTeamsInbound{}, false, errors.New(
			"teams: message activity requires conversation.id and id",
		)
	}
	if isTeamsMessageFromSelf(activity, cfg) {
		return mappedTeamsInbound{}, true, nil
	}
	inboundContext, err := resolveTeamsInboundContext(activity, cfg, receivedAt, "message activity")
	if err != nil {
		return mappedTeamsInbound{}, false, err
	}
	envelope := buildTeamsMessageEnvelope(cfg, inboundContext, activity)
	if err := envelope.Validate(); err != nil {
		return mappedTeamsInbound{}, false, err
	}
	return mappedTeamsInbound{Envelope: envelope, Direct: inboundContext.direct, User: inboundContext.user}, false, nil
}

func mapTeamsActionActivity(
	activity teamsActivity,
	cfg resolvedInstanceConfig,
	payload teamsActionPayload,
	receivedAt time.Time,
	source string,
) (mappedTeamsInbound, error) {
	if strings.TrimSpace(payload.ActionID) == "" {
		return mappedTeamsInbound{}, errors.New("teams: action activity requires actionId")
	}
	inboundContext, err := resolveTeamsInboundContext(activity, cfg, receivedAt, "action activity")
	if err != nil {
		return mappedTeamsInbound{}, err
	}
	messageID := firstNonEmpty(
		strings.TrimSpace(activity.ReplyToID),
		messageIDFromConversationID(inboundContext.conversationID),
		strings.TrimSpace(activity.ID),
	)
	envelope := buildTeamsActionEnvelope(cfg, inboundContext, activity, payload, source, messageID)
	if err := envelope.Validate(); err != nil {
		return mappedTeamsInbound{}, err
	}
	return mappedTeamsInbound{
		Envelope: envelope,
		Direct:   inboundContext.direct,
		User:     inboundContext.user,
	}, nil
}

func mapTeamsReactionActivity(
	activity teamsActivity,
	cfg resolvedInstanceConfig,
	receivedAt time.Time,
) ([]mappedTeamsInbound, error) {
	inboundContext, err := resolveTeamsInboundContext(
		activity,
		cfg,
		receivedAt,
		"reaction activity",
	)
	if err != nil {
		return nil, err
	}
	messageID := firstNonEmpty(
		messageIDFromConversationID(inboundContext.conversationID),
		strings.TrimSpace(activity.ReplyToID),
		strings.TrimSpace(activity.ID),
	)
	if messageID == "" {
		return nil, errors.New("teams: reaction activity requires a message identifier")
	}
	items := make(
		[]mappedTeamsInbound,
		0,
		len(activity.ReactionsAdded)+len(activity.ReactionsRemoved),
	)
	for _, reaction := range activity.ReactionsAdded {
		item, ok, err := mapTeamsReactionItem(
			cfg,
			inboundContext,
			activity,
			messageID,
			reaction,
			true,
		)
		if err != nil {
			return nil, err
		}
		if ok {
			items = append(items, item)
		}
	}
	for _, reaction := range activity.ReactionsRemoved {
		item, ok, err := mapTeamsReactionItem(
			cfg,
			inboundContext,
			activity,
			messageID,
			reaction,
			false,
		)
		if err != nil {
			return nil, err
		}
		if ok {
			items = append(items, item)
		}
	}
	return items, nil
}

func executeTeamsDelivery(
	ctx context.Context,
	api teamsAPI,
	cfg resolvedInstanceConfig,
	request bridgepkg.DeliveryRequest,
	state deliveryState,
	referenceStateLookup teamsDeliveryStateLookup,
	userContextLookup func(string, string) (teamsUserContext, bool),
) (bridgepkg.DeliveryAck, deliveryState, error) {
	if err := request.Validate(); err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}

	event := request.Event
	if event.EventType != bridgepkg.DeliveryEventTypeResume && event.Seq <= state.LastSeq {
		return bridgepkg.DeliveryAck{}, state, fmt.Errorf(
			"teams: out-of-order delivery seq %d after %d",
			event.Seq,
			state.LastSeq,
		)
	}
	if event.EventType == bridgepkg.DeliveryEventTypeResume && request.Snapshot != nil {
		state.LastSeq = request.Snapshot.LastAckedSeq
		state.RemoteMessageID = strings.TrimSpace(request.Snapshot.RemoteMessageID)
		state.ReplaceRemoteMessageID = strings.TrimSpace(request.Snapshot.ReplaceRemoteMessageID)
	}

	switch {
	case isTeamsDeleteDelivery(event):
		return executeTeamsDeleteDelivery(ctx, api, event, request.Snapshot, state, referenceStateLookup)
	case shouldPostTeamsMessage(event, state, request):
		return executeTeamsPostDelivery(ctx, api, cfg, event, state, userContextLookup)
	default:
		return executeTeamsEditDelivery(ctx, api, event, request.Snapshot, state, referenceStateLookup)
	}
}

func (p *teamsProvider) resolveTeamsManagedConfigs(
	session *bridgesdk.Session,
	managed []subprocess.InitializeBridgeManagedInstance,
) ([]resolvedInstanceConfig, string) {
	configs := make([]resolvedInstanceConfig, 0, len(managed))
	requestedListen := strings.TrimSpace(os.Getenv(teamsListenAddrEnv))
	usedPaths := make(map[string]int, len(managed))

	for _, item := range managed {
		cfg := p.resolveInstanceConfig(session, item)
		requestedListen = updateTeamsRequestedListen(&cfg, requestedListen)
		configs = append(configs, cfg)
		markDuplicateTeamsWebhookPath(configs, len(configs)-1, usedPaths)
	}
	return configs, requestedListen
}

func updateTeamsRequestedListen(cfg *resolvedInstanceConfig, requestedListen string) string {
	if cfg == nil || cfg.listenAddr == "" {
		return requestedListen
	}
	if requestedListen == "" {
		return cfg.listenAddr
	}
	if requestedListen != cfg.listenAddr && cfg.configError == nil {
		cfg.configError = fmt.Errorf(
			"teams: instance %q configured incompatible listen_addr %q (runtime uses %q)",
			cfg.instanceID,
			cfg.listenAddr,
			requestedListen,
		)
	}
	return requestedListen
}

func markDuplicateTeamsWebhookPath(
	configs []resolvedInstanceConfig,
	idx int,
	usedPaths map[string]int,
) {
	if idx < 0 || idx >= len(configs) {
		return
	}
	cfg := &configs[idx]
	if cfg.webhookPath == "" {
		return
	}
	if ownerIdx, ok := usedPaths[cfg.webhookPath]; ok {
		ownerID := strings.TrimSpace(configs[ownerIdx].instanceID)
		collisionErr := fmt.Errorf(
			"teams: webhook path %q is shared by %q and %q",
			cfg.webhookPath,
			ownerID,
			cfg.instanceID,
		)
		configs[ownerIdx].configError = joinTeamsConfigError(configs[ownerIdx].configError, collisionErr)
		cfg.configError = joinTeamsConfigError(cfg.configError, collisionErr)
		return
	}
	usedPaths[cfg.webhookPath] = idx
}

func joinTeamsConfigError(existing error, next error) error {
	if next == nil {
		return existing
	}
	if existing == nil {
		return next
	}
	return errors.Join(existing, next)
}

func applyTeamsListenErrors(
	configs []resolvedInstanceConfig,
	requestedListen string,
	startServer func(string) error,
) {
	if requestedListen == "" {
		for idx := range configs {
			if configs[idx].configError == nil {
				configs[idx].configError = errors.New("teams: webhook listen address is required")
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

func (p *teamsProvider) swapTeamsRoutes(configs []resolvedInstanceConfig, requestedListen string) {
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
		delete(p.reportedStatus, instanceID)
		delete(p.reportedHealth, instanceID)
	}
	p.routes = nextRoutes
	p.listenAddr = requestedListen
	p.mu.Unlock()
}

func (p *teamsProvider) populateTeamsInitialStates(
	ctx context.Context,
	configs []resolvedInstanceConfig,
) {
	for idx := range configs {
		status, degradation, err := p.determineInitialState(ctx, configs[idx])
		if err != nil {
			p.setLastError(err)
		}
		configs[idx].initialStatus = status
		configs[idx].initialDegradation = degradation
	}
}

func decodeTeamsProviderConfig(
	managed subprocess.InitializeBridgeManagedInstance,
) (teamsProviderConfig, error) {
	cfg := teamsProviderConfig{}
	if len(managed.Instance.ProviderConfig) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(managed.Instance.ProviderConfig, &cfg); err != nil {
		return teamsProviderConfig{}, err
	}
	return cfg, nil
}

func buildTeamsResolvedInstance(
	session *bridgesdk.Session,
	managed subprocess.InitializeBridgeManagedInstance,
	cfg teamsProviderConfig,
) resolvedInstanceConfig {
	appID, _ := session.Cache().BoundSecretValue(managed.Instance.ID, "app_id")
	appPassword, _ := session.Cache().BoundSecretValue(managed.Instance.ID, "app_password")
	appTenantID, _ := session.Cache().BoundSecretValue(managed.Instance.ID, "app_tenant_id")

	resolved := resolvedInstanceConfig{
		managed:    &managed,
		instanceID: strings.TrimSpace(managed.Instance.ID),
		listenAddr: firstNonEmpty(
			cfg.Webhook.ListenAddr,
			strings.TrimSpace(os.Getenv(teamsListenAddrEnv)),
		),
		webhookPath: normalizeWebhookPath(
			firstNonEmpty(cfg.Webhook.Path, "/teams/"+strings.TrimSpace(managed.Instance.ID)),
		),
		serviceURL: normalizeURL(firstNonEmpty(cfg.ServiceURL, teamsDefaultServiceURL)),
		openIDMetadataURL: normalizeURL(
			firstNonEmpty(
				cfg.Auth.OpenIDMetadataURL,
				strings.TrimSpace(os.Getenv(teamsOpenIDMetadataURLEnv)),
				teamsDefaultOpenIDMetadata,
			),
		),
		tokenURL: normalizeURL(
			firstNonEmpty(
				cfg.Auth.TokenURL,
				strings.TrimSpace(os.Getenv(teamsOAuthTokenURLEnvName())),
				defaultTeamsTokenURL(strings.TrimSpace(appTenantID)),
			),
		),
		appID:           strings.TrimSpace(appID),
		appPassword:     strings.TrimSpace(appPassword),
		appTenantID:     strings.TrimSpace(appTenantID),
		dmPolicy:        managed.Instance.DMPolicy.Normalize(),
		allowUserIDs:    buildTeamsIDSet(cfg.DM.AllowUserIDs),
		allowUsernames:  buildTeamsUsernameSet(cfg.DM.AllowUsernames),
		pairedUserIDs:   buildTeamsIDSet(cfg.DM.PairedUserIDs),
		pairedUsernames: buildTeamsUsernameSet(cfg.DM.PairedUsernames),
		dedup:           bridgesdk.NewDedupCache(5*time.Minute, 4000),
		rateLimiter:     bridgesdk.NewFixedWindowRateLimiter(200, time.Minute),
		inFlightLimiter: bridgesdk.NewInFlightLimiter(24),
	}
	if resolved.dmPolicy == "" {
		resolved.dmPolicy = bridgepkg.BridgeDMPolicyOpen
	}
	return resolved
}

func validateTeamsResolvedConfig(resolved *resolvedInstanceConfig) {
	if resolved == nil {
		return
	}
	switch {
	case resolved.webhookPath == "":
		resolved.configError = errors.New("teams: webhook path is required")
	case resolved.serviceURL == "":
		resolved.configError = errors.New("teams: provider_config.service_url is required")
	case !validTeamsServiceURL(resolved.serviceURL):
		resolved.configError = fmt.Errorf(
			"teams: provider_config.service_url %q is invalid",
			resolved.serviceURL,
		)
	case resolved.openIDMetadataURL == "":
		resolved.configError = errors.New("teams: openid metadata url is required")
	case !validTeamsCredentialedURL(resolved.openIDMetadataURL):
		resolved.configError = fmt.Errorf("teams: openid metadata url %q is invalid", resolved.openIDMetadataURL)
	case resolved.tokenURL == "":
		resolved.configError = errors.New("teams: token url is required")
	case !validTeamsCredentialedURL(resolved.tokenURL):
		resolved.configError = fmt.Errorf("teams: token url %q is invalid", resolved.tokenURL)
	case resolved.appTenantID != "" && !looksLikeTenantID(resolved.appTenantID):
		resolved.configError = fmt.Errorf(
			"teams: app_tenant_id %q is malformed",
			resolved.appTenantID,
		)
	}
}

func configureTeamsBatcher(
	provider *teamsProvider,
	cfg teamsProviderConfig,
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

func resolveTeamsInboundContext(
	activity teamsActivity,
	cfg resolvedInstanceConfig,
	receivedAt time.Time,
	kind string,
) (teamsInboundContext, error) {
	serviceURL := firstNonEmpty(normalizeURL(activity.ServiceURL), cfg.serviceURL)
	if serviceURL == "" {
		return teamsInboundContext{}, fmt.Errorf("teams: %s requires serviceUrl", kind)
	}
	if receivedAt.IsZero() {
		receivedAt = time.Now().UTC()
	}
	if parsed := parseTeamsTimestamp(activity.Timestamp); !parsed.IsZero() {
		receivedAt = parsed
	}
	conversationID := strings.TrimSpace(activity.Conversation.ID)
	if conversationID == "" {
		return teamsInboundContext{}, fmt.Errorf("teams: %s requires conversation.id", kind)
	}

	return teamsInboundContext{
		receivedAt:         receivedAt,
		serviceURL:         serviceURL,
		conversationID:     conversationID,
		direct:             isTeamsDirectConversation(activity.Conversation),
		baseConversationID: baseTeamsConversationID(conversationID),
		threadID: encodeTeamsThreadID(teamsThreadRef{
			ConversationID: conversationID,
			ServiceURL:     serviceURL,
		}),
		user: teamsUserIdentity{
			ID:       normalizeTeamsID(activity.From.ID),
			Username: normalizeTeamsUsername(activity.From.Name),
			DisplayName: firstNonEmpty(
				strings.TrimSpace(activity.From.Name),
				normalizeTeamsID(activity.From.ID),
			),
		},
	}, nil
}

func buildTeamsActionEnvelope(
	cfg resolvedInstanceConfig,
	inboundContext teamsInboundContext,
	activity teamsActivity,
	payload teamsActionPayload,
	source string,
	messageID string,
) bridgepkg.InboundMessageEnvelope {
	envelope := bridgepkg.InboundMessageEnvelope{
		BridgeInstanceID: cfg.instanceID,
		Scope:            cfg.managed.Instance.Scope,
		WorkspaceID:      cfg.managed.Instance.WorkspaceID,
		ReceivedAt:       inboundContext.receivedAt,
		Sender: bridgepkg.MessageSender{
			ID:          inboundContext.user.ID,
			Username:    inboundContext.user.Username,
			DisplayName: inboundContext.user.DisplayName,
		},
		EventFamily: bridgepkg.InboundEventFamilyAction,
		Action: &bridgepkg.InboundAction{
			ActionID:  strings.TrimSpace(payload.ActionID),
			MessageID: messageID,
			Value:     strings.TrimSpace(payload.Value),
		},
		IdempotencyKey: firstNonEmpty(
			strings.TrimSpace(activity.ID),
			fmt.Sprintf(
				"teams:%s:action:%s:%s",
				cfg.instanceID,
				messageID,
				strings.TrimSpace(payload.ActionID),
			),
		),
		ThreadID: inboundContext.threadID,
	}
	if inboundContext.direct {
		envelope.PeerID = inboundContext.baseConversationID
	} else {
		envelope.GroupID = inboundContext.baseConversationID
	}
	metadata, err := json.Marshal(map[string]any{
		"activity_id":          strings.TrimSpace(activity.ID),
		"action_id":            strings.TrimSpace(payload.ActionID),
		"base_conversation_id": inboundContext.baseConversationID,
		"conversation_id":      inboundContext.conversationID,
		"message_id":           messageID,
		"service_url":          inboundContext.serviceURL,
		"source":               source,
		"tenant_id":            extractTeamsTenantID(activity),
	})
	if err == nil {
		envelope.ProviderMetadata = metadata
	}
	return envelope
}

func buildTeamsMessageEnvelope(
	cfg resolvedInstanceConfig,
	inboundContext teamsInboundContext,
	activity teamsActivity,
) bridgepkg.InboundMessageEnvelope {
	envelope := bridgepkg.InboundMessageEnvelope{
		BridgeInstanceID:  cfg.instanceID,
		Scope:             cfg.managed.Instance.Scope,
		WorkspaceID:       cfg.managed.Instance.WorkspaceID,
		PlatformMessageID: strings.TrimSpace(activity.ID),
		ReceivedAt:        inboundContext.receivedAt,
		Sender: bridgepkg.MessageSender{
			ID:          inboundContext.user.ID,
			Username:    inboundContext.user.Username,
			DisplayName: inboundContext.user.DisplayName,
		},
		Content: bridgepkg.MessageContent{
			Text: normalizeTeamsText(activity.Text),
		},
		Attachments: normalizeTeamsAttachments(activity.Attachments),
		EventFamily: bridgepkg.InboundEventFamilyMessage,
		IdempotencyKey: fmt.Sprintf(
			"teams:%s:message:%s",
			cfg.instanceID,
			strings.TrimSpace(activity.ID),
		),
		ThreadID: inboundContext.threadID,
	}
	if inboundContext.direct {
		envelope.PeerID = inboundContext.baseConversationID
	} else {
		envelope.GroupID = inboundContext.baseConversationID
	}
	metadata, err := json.Marshal(map[string]any{
		"activity_id":          strings.TrimSpace(activity.ID),
		"base_conversation_id": inboundContext.baseConversationID,
		"channel_id":           strings.TrimSpace(activity.ChannelID),
		"conversation_id":      inboundContext.conversationID,
		"conversation_type":    strings.TrimSpace(activity.Conversation.ConversationType),
		"reply_to_id":          strings.TrimSpace(activity.ReplyToID),
		"service_url":          inboundContext.serviceURL,
		"tenant_id":            extractTeamsTenantID(activity),
		"type":                 strings.TrimSpace(activity.Type),
	})
	if err == nil {
		envelope.ProviderMetadata = metadata
	}
	return envelope
}

func mapTeamsReactionItem(
	cfg resolvedInstanceConfig,
	inboundContext teamsInboundContext,
	activity teamsActivity,
	messageID string,
	reaction teamsMessageReaction,
	added bool,
) (mappedTeamsInbound, bool, error) {
	raw := strings.TrimSpace(reaction.Type)
	if raw == "" {
		return mappedTeamsInbound{}, false, nil
	}

	envelope := bridgepkg.InboundMessageEnvelope{
		BridgeInstanceID: cfg.instanceID,
		Scope:            cfg.managed.Instance.Scope,
		WorkspaceID:      cfg.managed.Instance.WorkspaceID,
		ReceivedAt:       inboundContext.receivedAt,
		Sender: bridgepkg.MessageSender{
			ID:          inboundContext.user.ID,
			Username:    inboundContext.user.Username,
			DisplayName: inboundContext.user.DisplayName,
		},
		EventFamily: bridgepkg.InboundEventFamilyReaction,
		Reaction: &bridgepkg.InboundReaction{
			MessageID: messageID,
			Emoji:     normalizeTeamsEmoji(raw),
			RawEmoji:  raw,
			Added:     added,
		},
		IdempotencyKey: fmt.Sprintf(
			"teams:%s:reaction:%s:%s:%t",
			cfg.instanceID,
			messageID,
			raw,
			added,
		),
		ThreadID: inboundContext.threadID,
	}
	if inboundContext.direct {
		envelope.PeerID = inboundContext.baseConversationID
	} else {
		envelope.GroupID = inboundContext.baseConversationID
	}
	metadata, err := json.Marshal(map[string]any{
		"activity_id":          strings.TrimSpace(activity.ID),
		"base_conversation_id": inboundContext.baseConversationID,
		"conversation_id":      inboundContext.conversationID,
		"message_id":           messageID,
		"service_url":          inboundContext.serviceURL,
		"tenant_id":            extractTeamsTenantID(activity),
		"type":                 strings.TrimSpace(activity.Type),
	})
	if err == nil {
		envelope.ProviderMetadata = metadata
	}
	if err := envelope.Validate(); err != nil {
		return mappedTeamsInbound{}, false, err
	}
	return mappedTeamsInbound{
		Envelope: envelope,
		Direct:   inboundContext.direct,
		User:     inboundContext.user,
	}, true, nil
}

func isTeamsDeleteDelivery(event bridgepkg.DeliveryEvent) bool {
	return event.Operation.Normalize() == bridgepkg.DeliveryOperationDelete ||
		normalizeDeliveryEventType(event.EventType) == bridgepkg.DeliveryEventTypeDelete
}

func executeTeamsDeleteDelivery(
	ctx context.Context,
	api teamsAPI,
	event bridgepkg.DeliveryEvent,
	snapshot *bridgepkg.DeliverySnapshot,
	state deliveryState,
	referenceStateLookup teamsDeliveryStateLookup,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	remoteID := resolveTeamsReferencedRemoteMessageID(event.Reference, snapshot, state, referenceStateLookup)
	if remoteID == "" {
		return bridgepkg.DeliveryAck{}, state, errors.New(
			"teams: delete delivery requires a remote message id",
		)
	}
	ref, err := decodeRemoteMessageID(remoteID)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	if err := api.DeleteActivity(ctx, ref.ServiceURL, ref.ConversationID, ref.ActivityID); err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}

	ack := newTeamsDeliveryAck(event, remoteID, firstNonEmpty(state.RemoteMessageID, remoteID))
	state.LastSeq = event.Seq
	state.RemoteMessageID = remoteID
	state.ReplaceRemoteMessageID = ack.ReplaceRemoteMessageID
	return ack, state, ack.ValidateFor(event)
}

func executeTeamsPostDelivery(
	ctx context.Context,
	api teamsAPI,
	cfg resolvedInstanceConfig,
	event bridgepkg.DeliveryEvent,
	state deliveryState,
	userContextLookup func(string, string) (teamsUserContext, bool),
) (bridgepkg.DeliveryAck, deliveryState, error) {
	target, err := resolveTeamsDeliveryTarget(cfg, event, userContextLookup)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}

	conversationID := target.ConversationID
	serviceURL := target.ServiceURL
	if conversationID == "" {
		createReq := teamsCreateConversationRequest{
			Bot:      teamsChannelAccount{ID: cfg.appID},
			Members:  []teamsChannelAccount{{ID: target.UserID}},
			IsGroup:  false,
			TenantID: target.TenantID,
			ChannelData: map[string]any{
				"tenant": map[string]any{"id": target.TenantID},
			},
		}
		created, err := api.CreateConversation(ctx, serviceURL, createReq)
		if err != nil {
			return bridgepkg.DeliveryAck{}, state, err
		}
		if created == nil || strings.TrimSpace(created.ID) == "" {
			return bridgepkg.DeliveryAck{}, state, &bridgesdk.TransientError{
				Err: errors.New("teams: create conversation response omitted id"),
			}
		}
		conversationID = strings.TrimSpace(created.ID)
	}

	baseConversationID, replyToID := splitTeamsConversationTarget(
		firstNonEmpty(conversationID, target.ConversationID),
	)
	if target.ReplyToID != "" {
		replyToID = target.ReplyToID
	}
	sent, err := api.SendActivity(
		ctx,
		serviceURL,
		baseConversationID,
		replyToID,
		teamsOutboundActivity{
			Type:       "message",
			Text:       strings.TrimSpace(event.Content.Text),
			TextFormat: "markdown",
		},
	)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	if sent == nil || strings.TrimSpace(sent.ID) == "" {
		return bridgepkg.DeliveryAck{}, state, &bridgesdk.TransientError{
			Err: errors.New("teams: send activity response omitted id"),
		}
	}

	remoteID := encodeRemoteMessageID(teamsRemoteMessageRef{
		ConversationID: baseConversationID,
		ServiceURL:     serviceURL,
		ActivityID:     strings.TrimSpace(sent.ID),
	})
	ack := newTeamsDeliveryAck(event, remoteID, state.RemoteMessageID)
	state.LastSeq = event.Seq
	state.ReplaceRemoteMessageID = state.RemoteMessageID
	state.RemoteMessageID = remoteID
	return ack, state, ack.ValidateFor(event)
}

func executeTeamsEditDelivery(
	ctx context.Context,
	api teamsAPI,
	event bridgepkg.DeliveryEvent,
	snapshot *bridgepkg.DeliverySnapshot,
	state deliveryState,
	referenceStateLookup teamsDeliveryStateLookup,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	remoteID := resolveTeamsReferencedRemoteMessageID(event.Reference, snapshot, state, referenceStateLookup)
	if remoteID == "" {
		return bridgepkg.DeliveryAck{}, state, errors.New(
			"teams: edit delivery requires a remote message id",
		)
	}
	ref, err := decodeRemoteMessageID(remoteID)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	if err := api.UpdateActivity(ctx, ref.ServiceURL, ref.ConversationID, ref.ActivityID, teamsOutboundActivity{
		Type:       "message",
		Text:       strings.TrimSpace(event.Content.Text),
		TextFormat: "markdown",
	}); err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}

	ack := newTeamsDeliveryAck(event, remoteID, firstNonEmpty(state.RemoteMessageID, remoteID))
	state.LastSeq = event.Seq
	state.RemoteMessageID = remoteID
	state.ReplaceRemoteMessageID = ack.ReplaceRemoteMessageID
	return ack, state, ack.ValidateFor(event)
}

func resolveTeamsReferencedRemoteMessageID(
	reference *bridgepkg.DeliveryMessageReference,
	snapshot *bridgepkg.DeliverySnapshot,
	state deliveryState,
	referenceStateLookup teamsDeliveryStateLookup,
) string {
	if remoteID := referenceRemoteMessageID(reference); remoteID != "" {
		return remoteID
	}
	if deliveryID := referenceDeliveryID(reference); deliveryID != "" && referenceStateLookup != nil {
		if referencedState, ok := referenceStateLookup(deliveryID); ok {
			if remoteID := strings.TrimSpace(referencedState.RemoteMessageID); remoteID != "" {
				return remoteID
			}
		}
	}
	if remoteID := strings.TrimSpace(state.RemoteMessageID); remoteID != "" {
		return remoteID
	}
	if snapshot != nil {
		return strings.TrimSpace(snapshot.RemoteMessageID)
	}
	return ""
}

func newTeamsDeliveryAck(
	event bridgepkg.DeliveryEvent,
	remoteMessageID string,
	replaceRemoteMessageID string,
) bridgepkg.DeliveryAck {
	ack := bridgepkg.DeliveryAck{
		DeliveryID:      event.DeliveryID,
		Seq:             event.Seq,
		RemoteMessageID: remoteMessageID,
	}
	if strings.TrimSpace(replaceRemoteMessageID) != "" {
		ack.ReplaceRemoteMessageID = strings.TrimSpace(replaceRemoteMessageID)
	}
	return ack
}

func shouldPostTeamsMessage(
	event bridgepkg.DeliveryEvent,
	state deliveryState,
	request bridgepkg.DeliveryRequest,
) bool {
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

func allowTeamsDirectMessage(cfg resolvedInstanceConfig, user teamsUserIdentity, direct bool) bool {
	if !direct {
		return true
	}
	switch cfg.dmPolicy.Normalize() {
	case "", bridgepkg.BridgeDMPolicyOpen:
		return true
	case bridgepkg.BridgeDMPolicyAllowlist:
		return teamsIdentityAllowed(cfg.allowUserIDs, cfg.allowUsernames, user)
	case bridgepkg.BridgeDMPolicyPairing:
		if teamsIdentityAllowed(cfg.pairedUserIDs, cfg.pairedUsernames, user) {
			return true
		}
		return teamsIdentityAllowed(cfg.allowUserIDs, cfg.allowUsernames, user)
	default:
		return false
	}
}

func teamsIdentityAllowed(
	ids map[string]struct{},
	usernames map[string]struct{},
	user teamsUserIdentity,
) bool {
	if len(ids) == 0 && len(usernames) == 0 {
		return false
	}
	if _, ok := ids[normalizeTeamsID(user.ID)]; ok {
		return true
	}
	if _, ok := usernames[normalizeTeamsUsername(firstNonEmpty(user.Username, user.DisplayName))]; ok {
		return true
	}
	return false
}

func parseTeamsBearerToken(header string) (string, error) {
	authz := strings.TrimSpace(header)
	if !strings.HasPrefix(strings.ToLower(authz), "bearer ") {
		return "", errors.New("teams: bearer authorization is required")
	}
	tokenString := strings.TrimSpace(authz[len("Bearer "):])
	if tokenString == "" {
		return "", errors.New("teams: bearer token is required")
	}
	return tokenString, nil
}

func decodeTeamsAuthorizationProbe(body []byte) (struct {
	ServiceURL string `json:"serviceUrl,omitempty"`
	ChannelID  string `json:"channelId,omitempty"`
}, error) {
	probe := struct {
		ServiceURL string `json:"serviceUrl,omitempty"`
		ChannelID  string `json:"channelId,omitempty"`
	}{}
	if err := json.Unmarshal(body, &probe); err != nil {
		return probe, errors.New("teams: webhook payload is not valid json")
	}
	return probe, nil
}

func teamsAuthorizationServiceURL(probeServiceURL string, defaultServiceURL string) (string, error) {
	serviceURL := normalizeURL(probeServiceURL)
	if serviceURL == "" {
		serviceURL = defaultServiceURL
	}
	if serviceURL == "" {
		return "", errors.New("teams: serviceUrl is required for token validation")
	}
	return serviceURL, nil
}

func verifyTeamsAuthorization(
	ctx context.Context,
	req *http.Request,
	body []byte,
	cfg resolvedInstanceConfig,
) error {
	if req == nil {
		return errors.New("teams: webhook request is required")
	}
	tokenString, err := parseTeamsBearerToken(req.Header.Get("Authorization"))
	if err != nil {
		return err
	}
	probe, err := decodeTeamsAuthorizationProbe(body)
	if err != nil {
		return err
	}
	serviceURL, err := teamsAuthorizationServiceURL(probe.ServiceURL, cfg.serviceURL)
	if err != nil {
		return err
	}

	metadata, err := fetchTeamsOpenIDMetadata(ctx, cfg.openIDMetadataURL)
	if err != nil {
		return err
	}
	jwks, err := fetchTeamsJWKS(ctx, metadata.JWKSURI)
	if err != nil {
		return err
	}
	claims := &teamsAuthClaims{}
	parsed, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
				return nil, fmt.Errorf("teams: unsupported signing method %q", token.Method.Alg())
			}
			keyID := firstNonEmpty(
				stringHeader(token.Header, "kid"),
				stringHeader(token.Header, "x5t"),
			)
			if keyID == "" {
				return nil, errors.New("teams: token header missing key id")
			}
			jwk, err := jwks.keyByID(keyID)
			if err != nil {
				if refreshed, refreshErr := refreshTeamsJWKS(ctx, metadata.JWKSURI); refreshErr == nil {
					jwks = refreshed
					jwk, err = jwks.keyByID(keyID)
				}
			}
			if err != nil {
				return nil, err
			}
			if err := jwk.validateEndorsement(
				firstNonEmpty(strings.TrimSpace(probe.ChannelID), "msteams"),
			); err != nil {
				return nil, err
			}
			return jwk.publicKey()
		},
		jwt.WithAudience(strings.TrimSpace(cfg.appID)),
		jwt.WithIssuer(
			firstNonEmpty(strings.TrimSpace(metadata.Issuer), "https://api.botframework.com"),
		),
		jwt.WithLeeway(5*time.Minute),
	)
	if err != nil {
		return fmt.Errorf("teams: invalid bearer token: %w", err)
	}
	if !parsed.Valid {
		return errors.New("teams: invalid bearer token")
	}
	return validateTeamsAuthorizationClaims(claims, serviceURL)
}

func validateTeamsAuthorizationClaims(claims *teamsAuthClaims, serviceURL string) error {
	if normalizeURL(claims.ServiceURL) != serviceURL {
		return fmt.Errorf(
			"teams: token serviceUrl %q did not match activity serviceUrl %q",
			claims.ServiceURL,
			serviceURL,
		)
	}
	return nil
}

func fetchTeamsOpenIDMetadata(
	ctx context.Context,
	metadataURL string,
) (*teamsOpenIDMetadata, error) {
	if strings.TrimSpace(metadataURL) == "" {
		return nil, errors.New("teams: openid metadata url is required")
	}
	endpoint, err := validatedTeamsCredentialedURL(metadataURL, "openid metadata url")
	if err != nil {
		return nil, err
	}
	if cached := cachedTeamsOpenIDMetadata(endpoint); cached != nil {
		return cached, nil
	}
	metadata, err := fetchTeamsOpenIDMetadataUncached(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	storeTeamsOpenIDMetadata(endpoint, metadata)
	return cloneTeamsOpenIDMetadata(metadata), nil
}

func fetchTeamsOpenIDMetadataUncached(
	ctx context.Context,
	endpoint string,
) (*teamsOpenIDMetadata, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, err
	}
	resp, err := teamsAuthHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, classifyTeamsHTTPError(
			resp.StatusCode,
			resp.Header.Get("Retry-After"),
			readResponseBody(resp.Body),
		)
	}
	var metadata teamsOpenIDMetadata
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, err
	}
	if strings.TrimSpace(metadata.JWKSURI) == "" {
		return nil, errors.New("teams: openid metadata jwks_uri is required")
	}
	if !validTeamsCredentialedURL(metadata.JWKSURI) {
		return nil, fmt.Errorf("teams: openid metadata jwks_uri %q is invalid", metadata.JWKSURI)
	}
	return &metadata, nil
}

func fetchTeamsJWKS(ctx context.Context, jwksURL string) (*teamsJWKS, error) {
	endpoint, err := validatedTeamsCredentialedURL(jwksURL, "jwks url")
	if err != nil {
		return nil, err
	}
	if cached := cachedTeamsJWKS(endpoint); cached != nil {
		return cached, nil
	}
	keys, err := fetchTeamsJWKSUncached(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	storeTeamsJWKS(endpoint, keys)
	return cloneTeamsJWKS(keys), nil
}

func refreshTeamsJWKS(ctx context.Context, jwksURL string) (*teamsJWKS, error) {
	endpoint, err := validatedTeamsCredentialedURL(jwksURL, "jwks url")
	if err != nil {
		return nil, err
	}
	keys, err := fetchTeamsJWKSUncached(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	storeTeamsJWKS(endpoint, keys)
	return cloneTeamsJWKS(keys), nil
}

func fetchTeamsJWKSUncached(ctx context.Context, endpoint string) (*teamsJWKS, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, err
	}
	resp, err := teamsAuthHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, classifyTeamsHTTPError(
			resp.StatusCode,
			resp.Header.Get("Retry-After"),
			readResponseBody(resp.Body),
		)
	}
	var keys teamsJWKS
	if err := json.NewDecoder(resp.Body).Decode(&keys); err != nil {
		return nil, err
	}
	if len(keys.Keys) == 0 {
		return nil, errors.New("teams: jwks document omitted signing keys")
	}
	return &keys, nil
}

func cachedTeamsOpenIDMetadata(endpoint string) *teamsOpenIDMetadata {
	teamsAuthCache.mu.Lock()
	defer teamsAuthCache.mu.Unlock()
	entry, ok := teamsAuthCache.metadata[endpoint]
	if !ok || time.Now().After(entry.expiresAt) {
		if ok {
			delete(teamsAuthCache.metadata, endpoint)
		}
		return nil
	}
	return cloneTeamsOpenIDMetadata(&entry.metadata)
}

func storeTeamsOpenIDMetadata(endpoint string, metadata *teamsOpenIDMetadata) {
	if metadata == nil {
		return
	}
	teamsAuthCache.mu.Lock()
	defer teamsAuthCache.mu.Unlock()
	teamsAuthCache.metadata[endpoint] = teamsOpenIDMetadataCacheEntry{
		metadata:  *cloneTeamsOpenIDMetadata(metadata),
		expiresAt: time.Now().Add(teamsAuthCacheTTL),
	}
}

func cachedTeamsJWKS(endpoint string) *teamsJWKS {
	teamsAuthCache.mu.Lock()
	defer teamsAuthCache.mu.Unlock()
	entry, ok := teamsAuthCache.jwks[endpoint]
	if !ok || time.Now().After(entry.expiresAt) {
		if ok {
			delete(teamsAuthCache.jwks, endpoint)
		}
		return nil
	}
	return cloneTeamsJWKS(&entry.jwks)
}

func storeTeamsJWKS(endpoint string, keys *teamsJWKS) {
	if keys == nil {
		return
	}
	teamsAuthCache.mu.Lock()
	defer teamsAuthCache.mu.Unlock()
	teamsAuthCache.jwks[endpoint] = teamsJWKSCacheEntry{
		jwks:      *cloneTeamsJWKS(keys),
		expiresAt: time.Now().Add(teamsAuthCacheTTL),
	}
}

func cloneTeamsOpenIDMetadata(metadata *teamsOpenIDMetadata) *teamsOpenIDMetadata {
	if metadata == nil {
		return nil
	}
	clone := *metadata
	return &clone
}

func cloneTeamsJWKS(keys *teamsJWKS) *teamsJWKS {
	if keys == nil {
		return nil
	}
	clone := teamsJWKS{Keys: make([]teamsJWK, len(keys.Keys))}
	copy(clone.Keys, keys.Keys)
	for idx := range clone.Keys {
		clone.Keys[idx].Endorsements = append([]string(nil), keys.Keys[idx].Endorsements...)
	}
	return &clone
}

func teamsCredentialedHTTPClient(base *http.Client) *http.Client {
	if base == nil {
		return &http.Client{
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}

	client := *base
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &client
}

func (k teamsJWKS) keyByID(keyID string) (*teamsJWK, error) {
	for idx := range k.Keys {
		if strings.TrimSpace(k.Keys[idx].Kid) == keyID ||
			strings.TrimSpace(k.Keys[idx].X5T) == keyID {
			return &k.Keys[idx], nil
		}
	}
	return nil, fmt.Errorf("teams: jwk %q not found", keyID)
}

func (k teamsJWK) validateEndorsement(channelID string) error {
	if len(k.Endorsements) == 0 {
		return nil
	}
	for _, endorsement := range k.Endorsements {
		if strings.EqualFold(strings.TrimSpace(endorsement), strings.TrimSpace(channelID)) {
			return nil
		}
	}
	return fmt.Errorf("teams: jwk is not endorsed for channel %q", channelID)
}

func (k teamsJWK) publicKey() (*rsa.PublicKey, error) {
	if strings.TrimSpace(k.Kty) != "" && !strings.EqualFold(strings.TrimSpace(k.Kty), "RSA") {
		return nil, fmt.Errorf("teams: unsupported jwk kty %q", k.Kty)
	}
	modulusBytes, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(k.N))
	if err != nil {
		return nil, err
	}
	exponentBytes, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(k.E))
	if err != nil {
		return nil, err
	}
	exponent := 0
	for _, b := range exponentBytes {
		exponent = exponent<<8 + int(b)
	}
	if exponent == 0 {
		return nil, errors.New("teams: jwk exponent is invalid")
	}
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(modulusBytes),
		E: exponent,
	}, nil
}

func (c *teamsBotClient) ValidateAuth(ctx context.Context) error {
	_, err := c.accessToken(ctx)
	return err
}

func (c *teamsBotClient) CreateConversation(
	ctx context.Context,
	serviceURL string,
	req teamsCreateConversationRequest,
) (*teamsConversationResourceResponse, error) {
	var out teamsConversationResourceResponse
	if err := c.callJSON(ctx, http.MethodPost, serviceURL, "/v3/conversations", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *teamsBotClient) SendActivity(
	ctx context.Context,
	serviceURL string,
	conversationID string,
	replyToID string,
	activity teamsOutboundActivity,
) (*teamsResourceResponse, error) {
	path := "/v3/conversations/" + url.PathEscape(strings.TrimSpace(conversationID)) + "/activities"
	if strings.TrimSpace(replyToID) != "" {
		path = "/v3/conversations/" + url.PathEscape(
			strings.TrimSpace(conversationID),
		) + "/activities/" + url.PathEscape(
			strings.TrimSpace(replyToID),
		)
	}
	var out teamsResourceResponse
	if err := c.callJSON(ctx, http.MethodPost, serviceURL, path, activity, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *teamsBotClient) UpdateActivity(
	ctx context.Context,
	serviceURL string,
	conversationID string,
	activityID string,
	activity teamsOutboundActivity,
) error {
	return c.callJSON(
		ctx,
		http.MethodPut,
		serviceURL,
		"/v3/conversations/"+url.PathEscape(
			strings.TrimSpace(conversationID),
		)+"/activities/"+url.PathEscape(
			strings.TrimSpace(activityID),
		),
		activity,
		nil,
	)
}

func (c *teamsBotClient) DeleteActivity(
	ctx context.Context,
	serviceURL string,
	conversationID string,
	activityID string,
) error {
	return c.callJSON(
		ctx,
		http.MethodDelete,
		serviceURL,
		"/v3/conversations/"+url.PathEscape(
			strings.TrimSpace(conversationID),
		)+"/activities/"+url.PathEscape(
			strings.TrimSpace(activityID),
		),
		nil,
		nil,
	)
}

func (c *teamsBotClient) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	if c.cachedToken != "" && c.tokenExpiry.After(time.Now().UTC().Add(30*time.Second)) {
		token := c.cachedToken
		c.mu.Unlock()
		return token, nil
	}
	c.mu.Unlock()

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", c.cfg.appID)
	form.Set("client_secret", c.cfg.appPassword)
	form.Set("scope", teamsDefaultScope)

	tokenURL, err := validatedTeamsCredentialedURL(c.cfg.tokenURL, "token url")
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		tokenURL,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := teamsCredentialedHTTPClient(c.httpClient).Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", classifyTeamsHTTPError(
			resp.StatusCode,
			resp.Header.Get("Retry-After"),
			readResponseBody(resp.Body),
		)
	}
	var tokenResp teamsTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}
	if strings.TrimSpace(tokenResp.AccessToken) == "" {
		return "", &bridgesdk.AuthError{
			Err: errors.New("teams: token response omitted access token"),
		}
	}
	expiresIn := tokenResp.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	c.mu.Lock()
	c.cachedToken = strings.TrimSpace(tokenResp.AccessToken)
	c.tokenExpiry = time.Now().UTC().Add(time.Duration(expiresIn) * time.Second)
	token := c.cachedToken
	c.mu.Unlock()
	return token, nil
}

func (c *teamsBotClient) callJSON(
	ctx context.Context,
	method string,
	serviceURL string,
	path string,
	payload any,
	out any,
) error {
	token, err := c.accessToken(ctx)
	if err != nil {
		return err
	}
	base := strings.TrimRight(normalizeURL(serviceURL), "/")
	if base == "" {
		return errors.New("teams: service url is required")
	}
	fullURL := base + path
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return classifyTeamsHTTPError(
			resp.StatusCode,
			resp.Header.Get("Retry-After"),
			readResponseBody(resp.Body),
		)
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		if _, err := io.Copy(io.Discard, resp.Body); err != nil {
			return fmt.Errorf("teams: drain response body: %w", err)
		}
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

type teamsRemoteMessageRef struct {
	ConversationID string
	ServiceURL     string
	ActivityID     string
}

func resolveTeamsDeliveryTarget(
	cfg resolvedInstanceConfig,
	event bridgepkg.DeliveryEvent,
	userContextLookup func(string, string) (teamsUserContext, bool),
) (teamsResolvedTarget, error) {
	if thread := firstNonEmpty(
		strings.TrimSpace(event.DeliveryTarget.ThreadID),
		strings.TrimSpace(event.RoutingKey.ThreadID),
	); thread != "" {
		if decoded, err := decodeTeamsThreadID(thread); err == nil {
			baseConversationID, replyToID := splitTeamsConversationTarget(decoded.ConversationID)
			return teamsResolvedTarget{
				ConversationID: baseConversationID,
				ServiceURL:     firstNonEmpty(decoded.ServiceURL, cfg.serviceURL),
				ReplyToID:      replyToID,
			}, nil
		}
	}

	targetID := firstNonEmpty(
		strings.TrimSpace(event.DeliveryTarget.PeerID),
		strings.TrimSpace(event.DeliveryTarget.GroupID),
		strings.TrimSpace(event.RoutingKey.PeerID),
		strings.TrimSpace(event.RoutingKey.GroupID),
	)
	if targetID == "" {
		return teamsResolvedTarget{}, errors.New(
			"teams: delivery target requires peer_id or group_id",
		)
	}

	if looksLikeTeamsUserID(targetID) {
		ctx, ok := userContextLookup(cfg.instanceID, targetID)
		serviceURL := cfg.serviceURL
		tenantID := cfg.appTenantID
		if ok {
			serviceURL = firstNonEmpty(ctx.ServiceURL, serviceURL)
			tenantID = firstNonEmpty(ctx.TenantID, tenantID)
		}
		if tenantID == "" {
			return teamsResolvedTarget{}, &bridgesdk.PermanentError{
				Err: errors.New("teams: tenant ID not found for proactive DM target"),
			}
		}
		if serviceURL == "" {
			return teamsResolvedTarget{}, &bridgesdk.PermanentError{
				Err: errors.New("teams: service URL not found for proactive DM target"),
			}
		}
		return teamsResolvedTarget{
			ServiceURL: serviceURL,
			UserID:     normalizeTeamsID(targetID),
			TenantID:   tenantID,
		}, nil
	}

	baseConversationID, replyToID := splitTeamsConversationTarget(targetID)
	return teamsResolvedTarget{
		ConversationID: baseConversationID,
		ServiceURL:     cfg.serviceURL,
		ReplyToID:      replyToID,
	}, nil
}

func decodeTeamsMessageAction(raw json.RawMessage) (teamsActionPayload, bool) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return teamsActionPayload{}, false
	}
	var payload teamsActionPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return teamsActionPayload{}, false
	}
	if strings.TrimSpace(payload.ActionID) == "" {
		return teamsActionPayload{}, false
	}
	return payload, true
}

func decodeTeamsInvokeAction(raw json.RawMessage) (teamsActionPayload, bool) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return teamsActionPayload{}, false
	}
	var wrapper teamsActionValue
	if err := json.Unmarshal(raw, &wrapper); err != nil || wrapper.Action == nil {
		return teamsActionPayload{}, false
	}
	var payload teamsActionPayload
	if err := json.Unmarshal(wrapper.Action.Data, &payload); err != nil {
		return teamsActionPayload{}, false
	}
	if strings.TrimSpace(payload.ActionID) == "" {
		return teamsActionPayload{}, false
	}
	return payload, true
}

func normalizeTeamsAttachments(items []teamsAttachment) []bridgepkg.MessageAttachment {
	attachments := make([]bridgepkg.MessageAttachment, 0, len(items))
	for _, item := range items {
		contentType := strings.TrimSpace(item.ContentType)
		if contentType == "application/vnd.microsoft.card.adaptive" ||
			(contentType == "text/html" && strings.TrimSpace(item.ContentURL) == "") {
			continue
		}
		attachment := bridgepkg.MessageAttachment{
			Name:     strings.TrimSpace(item.Name),
			MIMEType: contentType,
			URL:      strings.TrimSpace(item.ContentURL),
		}
		if attachment.Name == "" && attachment.URL == "" && attachment.MIMEType == "" {
			continue
		}
		attachments = append(attachments, attachment)
	}
	if len(attachments) == 0 {
		return nil
	}
	return attachments
}

func normalizeTeamsText(text string) string {
	return strings.TrimSpace(text)
}

func normalizeTeamsEmoji(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeTeamsID(value string) string {
	return strings.TrimSpace(value)
}

func normalizeTeamsUsername(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isTeamsDirectConversation(conversation teamsConversation) bool {
	conversationID := strings.TrimSpace(conversation.ID)
	if conversationID == "" {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(conversation.ConversationType), "channel") {
		return false
	}
	return !strings.HasPrefix(conversationID, "19:")
}

func isTeamsMessageFromSelf(activity teamsActivity, cfg resolvedInstanceConfig) bool {
	sender := normalizeTeamsID(activity.From.ID)
	recipient := normalizeTeamsID(activity.Recipient.ID)
	return sender != "" && recipient != "" && sender == recipient ||
		recipient == strings.TrimSpace(cfg.appID)
}

func extractTeamsTenantID(activity teamsActivity) string {
	if tenantID := strings.TrimSpace(activity.Conversation.TenantID); tenantID != "" {
		return tenantID
	}
	if activity.ChannelData.Tenant != nil {
		return strings.TrimSpace(activity.ChannelData.Tenant.ID)
	}
	return ""
}

func messageIDFromConversationID(conversationID string) string {
	parts := strings.SplitN(conversationID, ";messageid=", 2)
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func baseTeamsConversationID(conversationID string) string {
	return strings.TrimSpace(
		messageIDStripPattern.ReplaceAllString(strings.TrimSpace(conversationID), ""),
	)
}

func splitTeamsConversationTarget(conversationID string) (string, string) {
	return baseTeamsConversationID(conversationID), messageIDFromConversationID(conversationID)
}

func encodeTeamsThreadID(ref teamsThreadRef) string {
	encodedConversationID := base64.RawURLEncoding.EncodeToString(
		[]byte(strings.TrimSpace(ref.ConversationID)),
	)
	encodedServiceURL := base64.RawURLEncoding.EncodeToString([]byte(normalizeURL(ref.ServiceURL)))
	return "teams:" + encodedConversationID + ":" + encodedServiceURL
}

func decodeTeamsThreadID(value string) (teamsThreadRef, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 3 || parts[0] != "teams" {
		return teamsThreadRef{}, errors.New("teams: invalid thread id")
	}
	conversationID, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return teamsThreadRef{}, err
	}
	serviceURL, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return teamsThreadRef{}, err
	}
	return teamsThreadRef{
		ConversationID: string(conversationID),
		ServiceURL:     string(serviceURL),
	}, nil
}

func encodeRemoteMessageID(ref teamsRemoteMessageRef) string {
	conversationID := base64.RawURLEncoding.EncodeToString(
		[]byte(strings.TrimSpace(ref.ConversationID)),
	)
	serviceURL := base64.RawURLEncoding.EncodeToString([]byte(normalizeURL(ref.ServiceURL)))
	activityID := base64.RawURLEncoding.EncodeToString([]byte(strings.TrimSpace(ref.ActivityID)))
	return strings.Join([]string{"teamsmsg", conversationID, serviceURL, activityID}, ":")
}

func decodeRemoteMessageID(value string) (teamsRemoteMessageRef, error) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 4 || parts[0] != "teamsmsg" {
		return teamsRemoteMessageRef{}, errors.New("teams: invalid remote message id")
	}
	conversationID, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return teamsRemoteMessageRef{}, err
	}
	serviceURL, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return teamsRemoteMessageRef{}, err
	}
	activityID, err := base64.RawURLEncoding.DecodeString(parts[3])
	if err != nil {
		return teamsRemoteMessageRef{}, err
	}
	if strings.TrimSpace(string(conversationID)) == "" ||
		strings.TrimSpace(string(serviceURL)) == "" ||
		strings.TrimSpace(string(activityID)) == "" {
		return teamsRemoteMessageRef{}, errors.New("teams: remote message id is incomplete")
	}
	return teamsRemoteMessageRef{
		ConversationID: strings.TrimSpace(string(conversationID)),
		ServiceURL:     normalizeURL(string(serviceURL)),
		ActivityID:     strings.TrimSpace(string(activityID)),
	}, nil
}

func referenceRemoteMessageID(reference *bridgepkg.DeliveryMessageReference) string {
	if reference == nil {
		return ""
	}
	return strings.TrimSpace(reference.RemoteMessageID)
}

func referenceDeliveryID(reference *bridgepkg.DeliveryMessageReference) string {
	if reference == nil {
		return ""
	}
	return strings.TrimSpace(reference.DeliveryID)
}

func classifyTeamsHTTPError(statusCode int, retryAfterHeader string, raw string) error {
	message := strings.TrimSpace(raw)
	if message == "" {
		message = fmt.Sprintf("teams: http %d", statusCode)
	}
	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return &bridgesdk.AuthError{Err: errors.New(message)}
	case http.StatusTooManyRequests:
		return &bridgesdk.RateLimitError{
			Err:        errors.New(message),
			RetryAfter: parseRetryAfter(retryAfterHeader),
		}
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

func defaultTeamsTokenURL(tenantID string) string {
	authority := "botframework.com"
	if trimmed := strings.TrimSpace(tenantID); trimmed != "" {
		authority = trimmed
	}
	return "https://login.microsoftonline.com/" + authority + "/oauth2/v2.0/token"
}

func teamsOAuthTokenURLEnvName() string {
	return strings.Join([]string{"AGH", "BRIDGE", "TEAMS", "TOKEN", "URL"}, "_")
}

func listenTeamsWebhook(listenAddr string) (net.Listener, error) {
	var listenConfig net.ListenConfig
	return listenConfig.Listen(context.Background(), "tcp", strings.TrimSpace(listenAddr))
}

func looksLikeTenantID(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	if strings.EqualFold(trimmed, "botframework.com") {
		return true
	}
	parts := strings.Split(trimmed, "-")
	if len(parts) != 5 {
		return false
	}
	return !slices.Contains(parts, "")
}

func validTeamsServiceURL(value string) bool {
	parsed, err := url.Parse(normalizeURL(value))
	if err != nil || parsed.Host == "" {
		return false
	}
	if parsed.Scheme == "https" {
		return true
	}
	if parsed.Scheme != "http" {
		return false
	}
	host := strings.TrimSpace(parsed.Hostname())
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func validatedTeamsCredentialedURL(value string, label string) (string, error) {
	normalized := normalizeURL(value)
	if !validTeamsCredentialedURL(normalized) {
		return "", fmt.Errorf("teams: %s %q is invalid", label, normalized)
	}
	return normalized, nil
}

func validTeamsCredentialedURL(value string) bool {
	parsed, err := url.Parse(normalizeURL(value))
	if err != nil || parsed.Host == "" {
		return false
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	switch parsed.Scheme {
	case "https":
		return host == "login.botframework.com" || host == "login.microsoftonline.com"
	case "http":
		return teamsLoopbackCredentialedURLsEnabledForTesting() && isLoopbackTeamsHost(host)
	default:
		return false
	}
}

func teamsLoopbackCredentialedURLsEnabledForTesting() bool {
	return strings.TrimSpace(os.Getenv(teamsTestLoopbackAuthEnv)) == "1"
}

func isLoopbackTeamsHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func looksLikeTeamsUserID(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	if strings.HasPrefix(trimmed, "29:") || strings.HasPrefix(trimmed, "8:orgid:") ||
		strings.HasPrefix(trimmed, "8:teamsvisitor:") {
		return true
	}
	if strings.HasPrefix(trimmed, "19:") || strings.Contains(trimmed, "@thread") {
		return false
	}
	return strings.Contains(trimmed, ":")
}

func parseTeamsTimestamp(value string) time.Time {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}

func buildTeamsIDSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		if normalized := normalizeTeamsID(value); normalized != "" {
			out[normalized] = struct{}{}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func buildTeamsUsernameSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		if normalized := normalizeTeamsUsername(value); normalized != "" {
			out[normalized] = struct{}{}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func managedInstancesToInstances(
	items []subprocess.InitializeBridgeManagedInstance,
) []bridgepkg.BridgeInstance {
	if len(items) == 0 {
		return nil
	}
	out := make([]bridgepkg.BridgeInstance, 0, len(items))
	for _, item := range items {
		out = append(out, item.Instance)
	}
	return out
}

func deliveryStateKey(instanceID string, deliveryID string) string {
	return strings.TrimSpace(instanceID) + ":" + strings.TrimSpace(deliveryID)
}

func userContextKey(instanceID string, userID string) string {
	return strings.TrimSpace(instanceID) + ":" + normalizeTeamsID(userID)
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
	return strings.TrimRight(strings.TrimSpace(value), "/")
}

func normalizeDeliveryEventType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
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
	_, err := w.Write([]byte("ok"))
	return err
}

func isNotInitializedRPCError(err error) bool {
	var rpcErr *subprocess.RPCError
	if !errors.As(err, &rpcErr) {
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
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
