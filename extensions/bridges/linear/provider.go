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
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/bridgesdk"
	extensioncontract "github.com/pedronauck/agh/internal/extension/contract"
	"github.com/pedronauck/agh/internal/subprocess"
)

const (
	linearListenAddrEnv = "AGH_BRIDGE_LINEAR_LISTEN_ADDR"
	linearAPIBaseEnv    = "AGH_BRIDGE_LINEAR_API_BASE_URL"

	linearDefaultAPIBaseURL  = "https://api.linear.app"
	linearDefaultWebhookPath = "/linear"
	linearOAuthPathSuffix    = "/oauth/token"

	linearModeComments      = "comments"
	linearModeAgentSessions = "agent_sessions"

	linearAuthModeAPIKey = "api_key"
	linearAuthModeOAuth  = "oauth"

	linearWebhookSkew              = time.Minute
	linearWebhookReadHeaderTimeout = 10 * time.Second
	linearWebhookIdleTimeout       = 30 * time.Second
	linearWebhookIngressTimeout    = 10 * time.Second

	rpcCodeNotInitialized = -32003
)

var (
	linearCommentSessionThreadPattern = regexp.MustCompile(`^linear:([^:]+):c:([^:]+):s:([^:]+)$`)
	linearIssueSessionThreadPattern   = regexp.MustCompile(`^linear:([^:]+):s:([^:]+)$`)
	linearCommentThreadPattern        = regexp.MustCompile(`^linear:([^:]+):c:([^:]+)$`)
	linearIssueThreadPattern          = regexp.MustCompile(`^linear:([^:]+)$`)
)

type linearProvider struct {
	sdk     *bridgesdk.Runtime
	stderr  io.Writer
	env     markerEnv
	now     func() time.Time
	session *bridgesdk.Session

	mu                    sync.RWMutex
	lastError             string
	server                *http.Server
	serverAddr            string
	listenAddr            string
	routes                map[string]resolvedInstanceConfig
	deliveries            map[string]deliveryState
	reportedStatus        map[string]bridgepkg.BridgeStatus
	apiFactory            func(resolvedInstanceConfig) linearAPI
	rateLimiter           *bridgesdk.FixedWindowRateLimiter
	inFlight              *bridgesdk.InFlightLimiter
	webhookIngressTimeout time.Duration

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

type linearProviderConfig struct {
	APIBaseURL     string `json:"api_base_url,omitempty"`
	OAuthTokenURL  string `json:"oauth_token_url,omitempty"`
	OrganizationID string `json:"organization_id,omitempty"`
	Mode           string `json:"mode,omitempty"`
	AuthMode       string `json:"auth_mode,omitempty"`
	Webhook        struct {
		ListenAddr string `json:"listen_addr,omitempty"`
		Path       string `json:"path,omitempty"`
	} `json:"webhook"`
}

type resolvedInstanceConfig struct {
	managed            *subprocess.InitializeBridgeManagedInstance
	instanceID         string
	organizationID     string
	mode               string
	authMode           string
	listenAddr         string
	webhookPath        string
	apiBaseURL         string
	oauthTokenURL      string
	webhookSecret      string
	apiKey             string
	clientID           string
	clientSecret       string
	botUserID          string
	botDisplayName     string
	dedup              *bridgesdk.DedupCache
	configError        error
	initialDegradation *bridgepkg.BridgeDegradation
	initialStatus      bridgepkg.BridgeStatus
	oauthTokenCache    *linearOAuthTokenCache
}

type linearThreadRef struct {
	IssueID        string
	RootCommentID  string
	AgentSessionID string
}

type linearWebhookEnvelope struct {
	Type             string `json:"type,omitempty"`
	Action           string `json:"action,omitempty"`
	OrganizationID   string `json:"organizationId,omitempty"`
	WebhookID        string `json:"webhookId,omitempty"`
	WebhookTimestamp int64  `json:"webhookTimestamp,omitempty"`
}

type linearActor struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Email     string `json:"email,omitempty"`
	AvatarURL string `json:"avatarUrl,omitempty"`
	URL       string `json:"url,omitempty"`
	Type      string `json:"type,omitempty"`
}

type linearCommentData struct {
	ID        string      `json:"id,omitempty"`
	Body      string      `json:"body,omitempty"`
	IssueID   string      `json:"issueId,omitempty"`
	UserID    string      `json:"userId,omitempty"`
	User      linearActor `json:"user"`
	CreatedAt string      `json:"createdAt,omitempty"`
	UpdatedAt string      `json:"updatedAt,omitempty"`
	ParentID  string      `json:"parentId,omitempty"`
}

type linearCommentWebhookPayload struct {
	Type             string            `json:"type,omitempty"`
	Action           string            `json:"action,omitempty"`
	CreatedAt        string            `json:"createdAt,omitempty"`
	OrganizationID   string            `json:"organizationId,omitempty"`
	URL              string            `json:"url,omitempty"`
	WebhookID        string            `json:"webhookId,omitempty"`
	WebhookTimestamp int64             `json:"webhookTimestamp,omitempty"`
	Data             linearCommentData `json:"data"`
	Actor            linearActor       `json:"actor"`
}

type linearAgentActivityPayload struct {
	ID        string `json:"id,omitempty"`
	Body      string `json:"body,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
	Content   struct {
		Type string `json:"type,omitempty"`
		Body string `json:"body,omitempty"`
	} `json:"content"`
}

type linearSessionComment struct {
	ID     string `json:"id,omitempty"`
	Body   string `json:"body,omitempty"`
	UserID string `json:"userId,omitempty"`
}

type linearAgentSession struct {
	ID              string                `json:"id,omitempty"`
	AppUserID       string                `json:"appUserId,omitempty"`
	IssueID         string                `json:"issueId,omitempty"`
	CommentID       string                `json:"commentId,omitempty"`
	SourceCommentID string                `json:"sourceCommentId,omitempty"`
	Comment         *linearSessionComment `json:"comment"`
	Creator         *linearActor          `json:"creator"`
}

type linearAgentSessionWebhookPayload struct {
	Type             string                      `json:"type,omitempty"`
	Action           string                      `json:"action,omitempty"`
	CreatedAt        string                      `json:"createdAt,omitempty"`
	AppUserID        string                      `json:"appUserId,omitempty"`
	OAuthClientID    string                      `json:"oauthClientId,omitempty"`
	OrganizationID   string                      `json:"organizationId,omitempty"`
	WebhookID        string                      `json:"webhookId,omitempty"`
	WebhookTimestamp int64                       `json:"webhookTimestamp,omitempty"`
	PromptContext    string                      `json:"promptContext,omitempty"`
	AgentSession     linearAgentSession          `json:"agentSession"`
	AgentActivity    *linearAgentActivityPayload `json:"agentActivity"`
	Actor            linearActor                 `json:"actor"`
}

func newLinearProvider(stderr io.Writer) (*linearProvider, error) {
	if stderr == nil {
		stderr = io.Discard
	}

	provider := &linearProvider{
		stderr:                stderr,
		env:                   markerEnvFromProcess(),
		now:                   func() time.Time { return time.Now().UTC() },
		routes:                make(map[string]resolvedInstanceConfig),
		deliveries:            make(map[string]deliveryState),
		reportedStatus:        make(map[string]bridgepkg.BridgeStatus),
		rateLimiter:           bridgesdk.NewFixedWindowRateLimiter(300, time.Minute),
		inFlight:              bridgesdk.NewInFlightLimiter(48),
		webhookIngressTimeout: linearWebhookIngressTimeout,
		stopCh:                make(chan struct{}),
	}
	provider.apiFactory = func(cfg resolvedInstanceConfig) linearAPI {
		return &linearClient{
			cfg: cfg,
			httpClient: &http.Client{
				Timeout: 10 * time.Second,
			},
			now: func() time.Time { return provider.now() },
		}
	}

	sdkRuntime, err := bridgesdk.NewRuntime(bridgesdk.RuntimeConfig{
		ExtensionInfo: subprocess.InitializeExtensionInfo{
			Name:    "linear",
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

func (p *linearProvider) serve(stdin io.Reader, stdout io.Writer) error {
	p.reportSideEffectError(
		"write start marker",
		appendMarkerLine(p.env.startsPath, fmt.Sprintf("pid=%d", os.Getpid())),
	)
	return p.sdk.Serve(context.Background(), stdin, stdout)
}

func (p *linearProvider) handleInitialize(_ context.Context, session *bridgesdk.Session) error {
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

func (p *linearProvider) afterInitialize(session *bridgesdk.Session) {
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

func (p *linearProvider) handleBridgesDeliver(
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
	ack, state, err := executeLinearDelivery(
		ctx,
		api,
		cfg,
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

func (p *linearProvider) healthCheck() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if strings.TrimSpace(p.lastError) == "" {
		return nil
	}
	return errors.New(strings.TrimSpace(p.lastError))
}

func (p *linearProvider) handleShutdown(
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
		if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			shutdownErr = fmt.Errorf("linear: shutdown webhook server: %w", err)
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

func (p *linearProvider) stop() {
	p.stopOnce.Do(func() {
		close(p.stopCh)
	})
}

func (p *linearProvider) syncOwnedInstances(
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

func (p *linearProvider) getOwnedInstance(
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

func (p *linearProvider) reportState(
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

func (p *linearProvider) reportReadyIfNeeded(
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

func (p *linearProvider) ingestBridgeMessage(
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

func (p *linearProvider) retryHostCall(ctx context.Context, fn func(context.Context) error) error {
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

func (p *linearProvider) reconcileInstanceConfigs(
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

	configs := make([]resolvedInstanceConfig, 0, len(managed))
	requestedListen := strings.TrimSpace(os.Getenv(linearListenAddrEnv))
	seenOwnership := make(map[string]string, len(managed))

	for _, item := range managed {
		cfg := p.resolveInstanceConfig(session, item)
		if cfg.listenAddr != "" {
			if requestedListen == "" {
				requestedListen = cfg.listenAddr
			} else if requestedListen != cfg.listenAddr && cfg.configError == nil {
				cfg.configError = fmt.Errorf(
					"linear: instance %q configured incompatible listen_addr %q (runtime uses %q)",
					cfg.instanceID,
					cfg.listenAddr,
					requestedListen,
				)
			}
		}
		ownershipKey := cfg.ownershipKey()
		if owner, ok := seenOwnership[ownershipKey]; ok && ownershipKey != "" && cfg.configError == nil {
			cfg.configError = fmt.Errorf(
				"linear: organization %q mode %q already belongs to %q and cannot also belong to %q",
				cfg.organizationID,
				cfg.mode,
				owner,
				cfg.instanceID,
			)
		}
		if ownershipKey != "" {
			seenOwnership[ownershipKey] = cfg.instanceID
		}
		configs = append(configs, cfg)
	}

	if requestedListen == "" {
		for idx := range configs {
			if configs[idx].configError == nil {
				configs[idx].configError = errors.New("linear: webhook listen address is required")
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
	for idx := range configs {
		updated, status, degradation, err := p.determineInitialState(ctx, configs[idx])
		if err != nil {
			p.setLastError(err)
		}
		updated.initialStatus = status
		updated.initialDegradation = degradation
		configs[idx] = updated
		nextRoutes[updated.instanceID] = updated
	}

	p.mu.Lock()
	p.routes = nextRoutes
	p.listenAddr = requestedListen
	p.mu.Unlock()

	return configs
}

func (p *linearProvider) resolveInstanceConfig(
	session *bridgesdk.Session,
	managed subprocess.InitializeBridgeManagedInstance,
) resolvedInstanceConfig {
	webhookSecret, _ := session.Cache().BoundSecretValue(managed.Instance.ID, "webhook_secret")
	apiKey, _ := session.Cache().BoundSecretValue(managed.Instance.ID, "api_key")
	clientID, _ := session.Cache().BoundSecretValue(managed.Instance.ID, "client_id")
	clientSecret, _ := session.Cache().BoundSecretValue(managed.Instance.ID, "client_secret")

	return resolveLinearInstanceConfig(
		managed,
		instanceSecretValues{
			webhookSecret: webhookSecret,
			apiKey:        apiKey,
			clientID:      clientID,
			clientSecret:  clientSecret,
		},
		resolveLinearEnv{
			listenAddr: strings.TrimSpace(os.Getenv(linearListenAddrEnv)),
			apiBaseURL: strings.TrimSpace(os.Getenv(linearAPIBaseEnv)),
			tokenURL:   strings.TrimSpace(os.Getenv(linearOAuthTokenURLEnvName())),
		},
	)
}

type instanceSecretValues struct {
	webhookSecret string
	apiKey        string
	clientID      string
	clientSecret  string
}

type resolveLinearEnv struct {
	listenAddr string
	apiBaseURL string
	tokenURL   string
}

func resolveLinearInstanceConfig(
	managed subprocess.InitializeBridgeManagedInstance,
	secrets instanceSecretValues,
	env resolveLinearEnv,
) resolvedInstanceConfig {
	cfg := linearProviderConfig{}
	if len(managed.Instance.ProviderConfig) > 0 {
		if err := json.Unmarshal(managed.Instance.ProviderConfig, &cfg); err != nil {
			return resolvedInstanceConfig{
				managed:         &managed,
				instanceID:      strings.TrimSpace(managed.Instance.ID),
				configError:     fmt.Errorf("linear: decode provider_config for %q: %w", managed.Instance.ID, err),
				dedup:           bridgesdk.NewDedupCache(5*time.Minute, 4000),
				oauthTokenCache: &linearOAuthTokenCache{},
			}
		}
	}

	resolved := resolvedInstanceConfig{
		managed:         &managed,
		instanceID:      strings.TrimSpace(managed.Instance.ID),
		organizationID:  strings.TrimSpace(cfg.OrganizationID),
		mode:            normalizeLinearMode(cfg.Mode),
		authMode:        normalizeLinearAuthMode(cfg.AuthMode),
		listenAddr:      firstNonEmpty(cfg.Webhook.ListenAddr, env.listenAddr),
		webhookPath:     normalizeWebhookPath(firstNonEmpty(cfg.Webhook.Path, linearDefaultWebhookPath)),
		apiBaseURL:      normalizeURL(firstNonEmpty(cfg.APIBaseURL, env.apiBaseURL, linearDefaultAPIBaseURL)),
		oauthTokenURL:   normalizeURL(firstNonEmpty(cfg.OAuthTokenURL, env.tokenURL, linearDefaultOAuthTokenURL())),
		webhookSecret:   strings.TrimSpace(secrets.webhookSecret),
		apiKey:          strings.TrimSpace(secrets.apiKey),
		clientID:        strings.TrimSpace(secrets.clientID),
		clientSecret:    strings.TrimSpace(secrets.clientSecret),
		dedup:           bridgesdk.NewDedupCache(5*time.Minute, 4000),
		oauthTokenCache: &linearOAuthTokenCache{},
	}

	switch {
	case resolved.organizationID == "":
		resolved.configError = errors.New("linear: provider_config.organization_id is required")
	case resolved.mode == "":
		resolved.configError = errors.New("linear: provider_config.mode is required")
	case resolved.mode != linearModeComments && resolved.mode != linearModeAgentSessions:
		resolved.configError = fmt.Errorf("linear: unsupported provider_config.mode %q", cfg.Mode)
	case resolved.authMode == "":
		resolved.configError = errors.New("linear: provider_config.auth_mode is required")
	case resolved.authMode != linearAuthModeAPIKey && resolved.authMode != linearAuthModeOAuth:
		resolved.configError = fmt.Errorf("linear: unsupported provider_config.auth_mode %q", cfg.AuthMode)
	case resolved.webhookPath == "":
		resolved.configError = errors.New("linear: webhook path is required")
	case resolved.apiBaseURL == "":
		resolved.configError = errors.New("linear: api base url is required")
	case !validLinearCredentialedURL(resolved.apiBaseURL):
		resolved.configError = fmt.Errorf("linear: api base url %q is invalid", resolved.apiBaseURL)
	case resolved.authMode == linearAuthModeOAuth && resolved.oauthTokenURL == "":
		resolved.configError = errors.New("linear: oauth token url is required for oauth auth_mode")
	case resolved.authMode == linearAuthModeOAuth && !validLinearCredentialedURL(resolved.oauthTokenURL):
		resolved.configError = fmt.Errorf("linear: oauth token url %q is invalid", resolved.oauthTokenURL)
	}

	return resolved
}

func (p *linearProvider) determineInitialState(
	ctx context.Context,
	cfg resolvedInstanceConfig,
) (resolvedInstanceConfig, bridgepkg.BridgeStatus, *bridgepkg.BridgeDegradation, error) {
	if cfg.configError != nil {
		return cfg, bridgepkg.BridgeStatusDegraded, &bridgepkg.BridgeDegradation{
			Reason:  bridgepkg.BridgeDegradationReasonTenantConfigInvalid,
			Message: cfg.configError.Error(),
		}, cfg.configError
	}
	if strings.TrimSpace(cfg.webhookSecret) == "" {
		err := errors.New("linear: webhook_secret secret binding is required")
		return cfg, bridgepkg.BridgeStatusAuthRequired, &bridgepkg.BridgeDegradation{
			Reason:  bridgepkg.BridgeDegradationReasonAuthFailed,
			Message: err.Error(),
		}, err
	}

	switch cfg.authMode {
	case linearAuthModeAPIKey:
		if strings.TrimSpace(cfg.apiKey) == "" {
			err := errors.New("linear: api_key secret binding is required for api_key auth_mode")
			return cfg, bridgepkg.BridgeStatusAuthRequired, &bridgepkg.BridgeDegradation{
				Reason:  bridgepkg.BridgeDegradationReasonAuthFailed,
				Message: err.Error(),
			}, err
		}
	case linearAuthModeOAuth:
		if strings.TrimSpace(cfg.clientID) == "" || strings.TrimSpace(cfg.clientSecret) == "" {
			err := errors.New("linear: client_id and client_secret secret bindings are required for oauth auth_mode")
			return cfg, bridgepkg.BridgeStatusAuthRequired, &bridgepkg.BridgeDegradation{
				Reason:  bridgepkg.BridgeDegradationReasonAuthFailed,
				Message: err.Error(),
			}, err
		}
	}

	viewer, err := p.apiFactory(cfg).ValidateAuth(ctx)
	if err != nil {
		classified := bridgesdk.ClassifyError(err)
		recovery := classified.Recovery()
		status := recovery.Status
		if status == "" {
			status = bridgepkg.BridgeStatusError
		}
		if recovery.Degradation != nil {
			return cfg, status, recovery.Degradation, err
		}
		return cfg, status, &bridgepkg.BridgeDegradation{
			Reason:  bridgepkg.BridgeDegradationReasonProviderTimeout,
			Message: classified.Message,
		}, err
	}

	if viewer != nil {
		if strings.TrimSpace(cfg.organizationID) != "" && strings.TrimSpace(viewer.OrganizationID) != "" &&
			!strings.EqualFold(strings.TrimSpace(cfg.organizationID), strings.TrimSpace(viewer.OrganizationID)) {
			err := fmt.Errorf(
				"linear: provider_config.organization_id %q does not match authenticated organization %q",
				cfg.organizationID,
				viewer.OrganizationID,
			)
			return cfg, bridgepkg.BridgeStatusDegraded, &bridgepkg.BridgeDegradation{
				Reason:  bridgepkg.BridgeDegradationReasonTenantConfigInvalid,
				Message: err.Error(),
			}, err
		}
		cfg.botUserID = strings.TrimSpace(viewer.ID)
		cfg.botDisplayName = strings.TrimSpace(viewer.DisplayName)
	}
	return cfg, bridgepkg.BridgeStatusReady, nil, nil
}

func (p *linearProvider) startServer(listenAddr string) error {
	p.mu.RLock()
	server := p.server
	currentListen := p.listenAddr
	p.mu.RUnlock()
	if server != nil {
		if currentListen != "" && currentListen != strings.TrimSpace(listenAddr) {
			return fmt.Errorf("linear: runtime already listening on %q, cannot switch to %q", currentListen, listenAddr)
		}
		return nil
	}

	ln, err := listenLinearWebhook(strings.TrimSpace(listenAddr))
	if err != nil {
		return fmt.Errorf("linear: listen %q: %w", listenAddr, err)
	}

	httpServer := &http.Server{
		Handler:           http.HandlerFunc(p.serveWebhookHTTP),
		ReadHeaderTimeout: linearWebhookReadHeaderTimeout,
		IdleTimeout:       linearWebhookIdleTimeout,
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

func (p *linearProvider) serveWebhookHTTP(w http.ResponseWriter, r *http.Request) {
	candidates := p.configsForPath(r.URL.Path)
	if len(candidates) == 0 {
		http.NotFound(w, r)
		return
	}

	handler, err := bridgesdk.NewWebhookHandler(bridgesdk.WebhookGuardConfig{
		AllowedMethods:      []string{http.MethodPost},
		AllowedContentTypes: []string{"application/json"},
		MaxBodyBytes:        1 << 20,
		RateLimiter:         p.rateLimiter,
		InFlightLimiter:     p.inFlight,
		VerifySignature: func(_ context.Context, req *http.Request, body []byte) error {
			scopedCandidates, err := selectLinearWebhookSignatureCandidates(body, candidates)
			if err != nil {
				return err
			}
			return verifyLinearWebhookSignature(req, body, scopedCandidates)
		},
		RequestKey: func(req *http.Request) string {
			return req.RemoteAddr + "|" + normalizeWebhookPath(req.URL.Path)
		},
		Now: func() time.Time { return p.now() },
	}, func(w http.ResponseWriter, r *http.Request, request bridgesdk.WebhookRequest) error {
		return p.handleWebhookRequest(w, r, candidates, request)
	})
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		p.setLastError(err)
		return
	}
	handler.ServeHTTP(w, r)
}

func (p *linearProvider) handleWebhookRequest(
	w http.ResponseWriter,
	r *http.Request,
	candidates []resolvedInstanceConfig,
	request bridgesdk.WebhookRequest,
) error {
	eventType, err := decodeLinearWebhookEnvelopeType(request.Body, request.ReceivedAt)
	if err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: "invalid linear webhook payload"}
	}
	ctx, cancel := p.webhookIngressContext(r)
	defer cancel()

	switch strings.TrimSpace(eventType) {
	case "Comment":
		return p.handleLinearCommentWebhook(ctx, w, candidates, request)
	case "AgentSessionEvent":
		return p.handleLinearAgentSessionWebhook(ctx, w, candidates, request)
	default:
		return writeWebhookText(w, http.StatusOK, "ok")
	}
}

func (p *linearProvider) webhookIngressContext(r *http.Request) (context.Context, context.CancelFunc) {
	base := context.Background()
	if r != nil {
		base = context.WithoutCancel(r.Context())
	}
	timeout := linearWebhookIngressTimeout
	if p != nil && p.webhookIngressTimeout > 0 {
		timeout = p.webhookIngressTimeout
	}
	return context.WithTimeout(base, timeout)
}

func (p *linearProvider) dispatchInboundEnvelope(
	ctx context.Context,
	bridgeInstanceID string,
	envelope bridgepkg.InboundMessageEnvelope,
) error {
	session := p.currentSession()
	if session == nil {
		return errors.New("linear: runtime session is not initialized")
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

func (p *linearProvider) configForInstance(instanceID string) (resolvedInstanceConfig, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	cfg, ok := p.routes[strings.TrimSpace(instanceID)]
	if !ok {
		return resolvedInstanceConfig{}, fmt.Errorf("linear: unmanaged bridge instance %q", instanceID)
	}
	return cfg, nil
}

func (p *linearProvider) waitForInstanceConfig(
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

func (p *linearProvider) configsForPath(path string) []resolvedInstanceConfig {
	normalizedPath := normalizeWebhookPath(path)
	p.mu.RLock()
	defer p.mu.RUnlock()

	configs := make([]resolvedInstanceConfig, 0, len(p.routes))
	for _, cfg := range p.routes {
		if cfg.webhookPath == normalizedPath {
			configs = append(configs, cfg)
		}
	}
	return configs
}

func (p *linearProvider) currentSession() *bridgesdk.Session {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.session
}

func (p *linearProvider) deliveryState(instanceID string, deliveryID string) deliveryState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.deliveries[deliveryStateKey(instanceID, deliveryID)]
}

func (p *linearProvider) storeDeliveryState(instanceID string, deliveryID string, state deliveryState) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.deliveries[deliveryStateKey(instanceID, deliveryID)] = state
}

func (p *linearProvider) setLastError(err error) {
	if err == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastError = err.Error()
}

func (p *linearProvider) clearLastError() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastError = ""
}

func (p *linearProvider) reportSideEffectError(action string, err error) {
	reportSideEffectError(p.stderr, action, err)
}

func executeLinearDelivery(
	ctx context.Context,
	api linearAPI,
	cfg resolvedInstanceConfig,
	request bridgepkg.DeliveryRequest,
	state deliveryState,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	event := request.Event
	if event.Seq <= state.LastSeq {
		return bridgepkg.DeliveryAck{}, state, fmt.Errorf(
			"linear: out-of-order delivery seq %d after %d",
			event.Seq,
			state.LastSeq,
		)
	}

	switch cfg.mode {
	case linearModeComments:
		return executeLinearCommentDelivery(ctx, api, request, state)
	case linearModeAgentSessions:
		return executeLinearAgentSessionDelivery(ctx, api, request, state)
	default:
		return bridgepkg.DeliveryAck{}, state, &bridgesdk.PermanentError{
			Err: fmt.Errorf("linear: unsupported runtime mode %q", cfg.mode),
		}
	}
}

func executeLinearCommentDelivery(
	ctx context.Context,
	api linearAPI,
	request bridgepkg.DeliveryRequest,
	state deliveryState,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	event := request.Event

	if event.Operation.Normalize() == bridgepkg.DeliveryOperationDelete ||
		normalizeDeliveryEventType(event.EventType) == bridgepkg.DeliveryEventTypeDelete {
		remoteID := resolveLinearRemoteMessageID(event.Reference, state, request.Snapshot)
		if strings.TrimSpace(remoteID) == "" {
			return bridgepkg.DeliveryAck{}, state, &bridgesdk.PermanentError{
				Err: errors.New("linear: delete requires a remote message id"),
			}
		}
		if err := api.DeleteComment(ctx, remoteID); err != nil {
			return bridgepkg.DeliveryAck{}, state, err
		}
		next := state
		next.LastSeq = event.Seq
		next.LastContent = ""
		ack := bridgepkg.DeliveryAck{
			DeliveryID:      event.DeliveryID,
			Seq:             event.Seq,
			RemoteMessageID: remoteID,
		}
		return ack, next, nil
	}

	thread, err := decodeLinearThreadID(
		firstNonEmpty(
			event.DeliveryTarget.ThreadID,
			event.RoutingKey.ThreadID,
			issueThreadIDFromGroup(event.DeliveryTarget.GroupID, event.RoutingKey.GroupID),
		),
	)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, &bridgesdk.PermanentError{Err: err}
	}
	body := event.Content.Text
	remoteID := resolveLinearRemoteMessageID(event.Reference, state, request.Snapshot)

	var comment *linearComment
	switch {
	case shouldSkipLinearCommentDelivery(event, remoteID, body):
		comment = nil
	case strings.TrimSpace(remoteID) == "":
		comment, err = api.CreateComment(ctx, thread.IssueID, body, thread.RootCommentID)
	default:
		comment, err = api.UpdateComment(ctx, remoteID, body)
	}
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	if comment != nil {
		remoteID = comment.ID
	}

	next := state
	next.LastSeq = event.Seq
	next.LastContent = body
	next.ReplaceRemoteMessageID = state.RemoteMessageID
	next.RemoteMessageID = strings.TrimSpace(remoteID)

	ack := bridgepkg.DeliveryAck{
		DeliveryID:             event.DeliveryID,
		Seq:                    event.Seq,
		RemoteMessageID:        strings.TrimSpace(remoteID),
		ReplaceRemoteMessageID: strings.TrimSpace(state.RemoteMessageID),
	}
	return ack, next, nil
}

func executeLinearAgentSessionDelivery(
	ctx context.Context,
	api linearAPI,
	request bridgepkg.DeliveryRequest,
	state deliveryState,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	event := request.Event
	if err := validateLinearAgentSessionDelivery(event); err != nil {
		return bridgepkg.DeliveryAck{}, state, &bridgesdk.PermanentError{Err: err}
	}

	thread, err := decodeLinearAgentSessionThread(event)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, &bridgesdk.PermanentError{Err: err}
	}

	remoteID := resolveLinearRemoteMessageID(event.Reference, state, request.Snapshot)
	if ack, next, ok := resumeLinearAgentSessionDelivery(event, state, remoteID); ok {
		return ack, next, nil
	}

	content := event.Content.Text
	delta := computeLinearAppendDelta(state.LastContent, content)
	if normalizeDeliveryEventType(event.EventType) == bridgepkg.DeliveryEventTypeResume &&
		strings.TrimSpace(remoteID) == "" {
		delta = content
	}

	next := state
	next.LastSeq = event.Seq
	next.LastContent = content
	next.ReplaceRemoteMessageID = state.RemoteMessageID
	next.RemoteMessageID = remoteID

	if delta == "" {
		ack := bridgepkg.DeliveryAck{
			DeliveryID:             event.DeliveryID,
			Seq:                    event.Seq,
			RemoteMessageID:        remoteID,
			ReplaceRemoteMessageID: firstNonEmpty(state.RemoteMessageID, remoteID),
		}
		return ack, next, nil
	}

	activity, err := api.CreateAgentActivity(ctx, thread.AgentSessionID, delta)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	remoteID = strings.TrimSpace(activity.ID)
	if activity.SourceComment != nil && strings.TrimSpace(activity.SourceComment.ID) != "" {
		remoteID = strings.TrimSpace(activity.SourceComment.ID)
	}
	next.RemoteMessageID = remoteID

	ack := bridgepkg.DeliveryAck{
		DeliveryID:             event.DeliveryID,
		Seq:                    event.Seq,
		RemoteMessageID:        remoteID,
		ReplaceRemoteMessageID: strings.TrimSpace(state.RemoteMessageID),
	}
	return ack, next, nil
}

type linearMappedInbound struct {
	Envelope bridgepkg.InboundMessageEnvelope
}

func mapLinearCommentCreated(
	payload linearCommentWebhookPayload,
	managed subprocess.InitializeBridgeManagedInstance,
	receivedAt time.Time,
) (linearMappedInbound, bool, error) {
	comment := payload.Data
	if strings.TrimSpace(comment.IssueID) == "" {
		return linearMappedInbound{}, true, nil
	}
	commentID := strings.TrimSpace(comment.ID)
	if commentID == "" {
		return linearMappedInbound{}, false, errors.New("linear: comment webhook data.id is required")
	}
	rootCommentID := firstNonEmpty(comment.ParentID, commentID)
	threadID := encodeLinearThreadID(linearThreadRef{
		IssueID:       strings.TrimSpace(comment.IssueID),
		RootCommentID: strings.TrimSpace(rootCommentID),
	})
	providerMetadata, err := json.Marshal(map[string]any{
		"organization_id": payload.OrganizationID,
		"issue_id":        strings.TrimSpace(comment.IssueID),
		"comment_id":      commentID,
		"root_comment_id": strings.TrimSpace(rootCommentID),
		"mode":            linearModeComments,
		"url":             strings.TrimSpace(payload.URL),
	})
	if err != nil {
		return linearMappedInbound{}, false, err
	}

	envelope := bridgepkg.InboundMessageEnvelope{
		BridgeInstanceID:  managed.Instance.ID,
		Scope:             managed.Instance.Scope,
		WorkspaceID:       managed.Instance.WorkspaceID,
		GroupID:           strings.TrimSpace(comment.IssueID),
		ThreadID:          threadID,
		PlatformMessageID: commentID,
		ReceivedAt:        receivedAt,
		Sender: bridgepkg.MessageSender{
			ID:          firstNonEmpty(comment.User.ID, comment.UserID, payload.Actor.ID),
			Username:    linearUserName(firstNonEmpty(comment.User.URL, payload.Actor.URL)),
			DisplayName: firstNonEmpty(comment.User.Name, payload.Actor.Name),
		},
		Content: bridgepkg.MessageContent{
			Text: strings.TrimSpace(comment.Body),
		},
		EventFamily:      bridgepkg.InboundEventFamilyMessage,
		ProviderMetadata: providerMetadata,
		IdempotencyKey:   firstNonEmpty(payload.WebhookID, commentID),
	}
	return linearMappedInbound{Envelope: envelope}, false, envelope.Validate()
}

func mapLinearAgentSessionEvent(
	payload linearAgentSessionWebhookPayload,
	managed *subprocess.InitializeBridgeManagedInstance,
	receivedAt time.Time,
	botUserID string,
) (linearMappedInbound, bool, error) {
	if managed == nil {
		return linearMappedInbound{}, false, errors.New("linear: managed bridge instance is required")
	}
	action := strings.TrimSpace(payload.Action)
	if !shouldProcessLinearAgentSessionAction(action) {
		return linearMappedInbound{}, true, nil
	}

	sessionID := strings.TrimSpace(payload.AgentSession.ID)
	issueID := strings.TrimSpace(payload.AgentSession.IssueID)
	rootCommentID := strings.TrimSpace(
		firstNonEmpty(payload.AgentSession.CommentID, payload.AgentSession.SourceCommentID),
	)
	if issueID == "" || sessionID == "" || rootCommentID == "" {
		return linearMappedInbound{}, false, errors.New(
			"linear: agent session webhook is missing issue, session, or root comment identity",
		)
	}
	if isUnexpectedLinearBotUser(botUserID, payload) {
		return linearMappedInbound{}, true, nil
	}

	messageID, text, sender, err := mapLinearAgentSessionMessage(action, payload)
	if err != nil {
		return linearMappedInbound{}, false, err
	}
	if messageID == "" {
		return linearMappedInbound{}, false, errors.New("linear: agent session webhook message id is required")
	}

	threadID := encodeLinearThreadID(linearThreadRef{
		IssueID:        issueID,
		RootCommentID:  rootCommentID,
		AgentSessionID: sessionID,
	})
	providerMetadata, err := json.Marshal(map[string]any{
		"organization_id":  payload.OrganizationID,
		"issue_id":         issueID,
		"root_comment_id":  rootCommentID,
		"agent_session_id": sessionID,
		"prompt_context":   strings.TrimSpace(payload.PromptContext),
		"mode":             linearModeAgentSessions,
		"action":           action,
	})
	if err != nil {
		return linearMappedInbound{}, false, err
	}

	envelope := bridgepkg.InboundMessageEnvelope{
		BridgeInstanceID:  managed.Instance.ID,
		Scope:             managed.Instance.Scope,
		WorkspaceID:       managed.Instance.WorkspaceID,
		GroupID:           issueID,
		ThreadID:          threadID,
		PlatformMessageID: messageID,
		ReceivedAt:        receivedAt,
		Sender:            sender,
		Content:           bridgepkg.MessageContent{Text: text},
		EventFamily:       bridgepkg.InboundEventFamilyMessage,
		ProviderMetadata:  providerMetadata,
		IdempotencyKey:    firstNonEmpty(payload.WebhookID, sessionID+":"+action+":"+messageID),
	}
	return linearMappedInbound{Envelope: envelope}, false, envelope.Validate()
}

func verifyLinearWebhookSignature(req *http.Request, body []byte, candidates []resolvedInstanceConfig) error {
	if req == nil {
		return errors.New("linear: webhook request is required")
	}
	signature := strings.TrimSpace(req.Header.Get("linear-signature"))
	if signature == "" {
		return errors.New("linear: webhook signature is required")
	}
	for _, cfg := range candidates {
		secret := strings.TrimSpace(cfg.webhookSecret)
		if secret == "" {
			continue
		}
		if linearSignature(secret, body) == signature {
			return nil
		}
	}
	return errors.New("linear: invalid webhook signature")
}

func selectLinearWebhookSignatureCandidates(
	body []byte,
	candidates []resolvedInstanceConfig,
) ([]resolvedInstanceConfig, error) {
	envelope := linearWebhookEnvelope{}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return candidates, nil
	}

	mode, ok := linearWebhookModeForType(envelope.Type)
	organizationID := strings.TrimSpace(envelope.OrganizationID)
	if !ok || organizationID == "" {
		return candidates, nil
	}

	cfg, found, err := selectLinearConfig(candidates, organizationID, mode)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.New("linear: invalid webhook signature")
	}
	return []resolvedInstanceConfig{cfg}, nil
}

func linearWebhookModeForType(eventType string) (string, bool) {
	switch strings.TrimSpace(eventType) {
	case "Comment":
		return linearModeComments, true
	case "AgentSessionEvent":
		return linearModeAgentSessions, true
	default:
		return "", false
	}
}

func validateLinearWebhookTimestamp(timestampMS int64, receivedAt time.Time) error {
	if timestampMS <= 0 {
		return nil
	}
	when := time.UnixMilli(timestampMS).UTC()
	if when.Before(receivedAt.UTC().Add(-linearWebhookSkew)) {
		return errors.New("linear: webhook timestamp is stale")
	}
	return nil
}

func linearCommentIsSelf(cfg resolvedInstanceConfig, payload linearCommentWebhookPayload) bool {
	commentUserID := strings.TrimSpace(firstNonEmpty(payload.Data.User.ID, payload.Data.UserID, payload.Actor.ID))
	if commentUserID == "" || strings.TrimSpace(cfg.botUserID) == "" {
		return false
	}
	return commentUserID == strings.TrimSpace(cfg.botUserID)
}

func selectLinearConfig(
	candidates []resolvedInstanceConfig,
	organizationID string,
	mode string,
) (resolvedInstanceConfig, bool, error) {
	selected := resolvedInstanceConfig{}
	found := false
	for _, cfg := range candidates {
		if strings.TrimSpace(cfg.organizationID) != strings.TrimSpace(organizationID) ||
			strings.TrimSpace(cfg.mode) != strings.TrimSpace(mode) {
			continue
		}
		if found {
			return resolvedInstanceConfig{}, false, fmt.Errorf(
				"linear: multiple managed instances matched organization %q mode %q",
				organizationID,
				mode,
			)
		}
		selected = cfg
		found = true
	}
	return selected, found, nil
}

func (c resolvedInstanceConfig) ownershipKey() string {
	if strings.TrimSpace(c.organizationID) == "" || strings.TrimSpace(c.mode) == "" {
		return ""
	}
	return strings.TrimSpace(c.organizationID) + "|" + strings.TrimSpace(c.mode)
}

func (c resolvedInstanceConfig) graphqlURL() string {
	base := strings.TrimRight(strings.TrimSpace(c.apiBaseURL), "/")
	if strings.HasSuffix(base, "/graphql") {
		return base
	}
	return base + "/graphql"
}

func listenLinearWebhook(listenAddr string) (net.Listener, error) {
	var listenConfig net.ListenConfig
	return listenConfig.Listen(context.Background(), "tcp", strings.TrimSpace(listenAddr))
}

func linearDefaultOAuthTokenURL() string {
	return strings.TrimRight(linearDefaultAPIBaseURL, "/") + linearOAuthPathSuffix
}

func linearOAuthTokenURLEnvName() string {
	return strings.Join([]string{"AGH", "BRIDGE", "LINEAR", "TOKEN", "URL"}, "_")
}

func validLinearCredentialedURL(value string) bool {
	parsed, err := url.Parse(normalizeURL(value))
	if err != nil || parsed.Host == "" {
		return false
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	switch parsed.Scheme {
	case "https":
		return host == "api.linear.app"
	case "http":
		return isLoopbackBridgeHost(host)
	default:
		return false
	}
}

func isLoopbackBridgeHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func managedInstancesToInstances(managed []subprocess.InitializeBridgeManagedInstance) []bridgepkg.BridgeInstance {
	instances := make([]bridgepkg.BridgeInstance, 0, len(managed))
	for _, item := range managed {
		instances = append(instances, item.Instance)
	}
	return instances
}

func cloneDegradation(value *bridgepkg.BridgeDegradation) *bridgepkg.BridgeDegradation {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func deliveryStateKey(instanceID string, deliveryID string) string {
	return strings.TrimSpace(instanceID) + "|" + strings.TrimSpace(deliveryID)
}

func normalizeLinearMode(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "comments", "comment":
		return linearModeComments
	case "agent_sessions", "agent-sessions", "agent_session", "agent-session":
		return linearModeAgentSessions
	default:
		return normalized
	}
}

func normalizeLinearAuthMode(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "api_key", "api-key":
		return linearAuthModeAPIKey
	case "oauth":
		return linearAuthModeOAuth
	default:
		return normalized
	}
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func writeWebhookText(w http.ResponseWriter, statusCode int, body string) error {
	if w == nil {
		return nil
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(statusCode)
	_, err := io.WriteString(w, body)
	return err
}

func decodeLinearWebhookEnvelopeType(body []byte, receivedAt time.Time) (string, error) {
	envelope := linearWebhookEnvelope{}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return "", err
	}
	if err := validateLinearWebhookTimestamp(envelope.WebhookTimestamp, receivedAt); err != nil {
		return "", err
	}
	return strings.TrimSpace(envelope.Type), nil
}

func (p *linearProvider) handleLinearCommentWebhook(
	ctx context.Context,
	w http.ResponseWriter,
	candidates []resolvedInstanceConfig,
	request bridgesdk.WebhookRequest,
) error {
	payload := linearCommentWebhookPayload{}
	if err := json.Unmarshal(request.Body, &payload); err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: "invalid linear comment payload"}
	}

	cfg, ok, err := selectLinearConfig(candidates, payload.OrganizationID, linearModeComments)
	if err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
	}
	if !ok {
		return writeWebhookText(w, http.StatusOK, "ignored")
	}
	if strings.TrimSpace(payload.Action) != "create" {
		return writeWebhookText(w, http.StatusOK, "ok")
	}

	mapped, ignored, err := mapLinearCommentCreated(payload, *cfg.managed, request.ReceivedAt)
	if err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
	}
	if ignored || cfg.dedup.Mark(mapped.Envelope.IdempotencyKey) || linearCommentIsSelf(cfg, payload) {
		return writeWebhookText(w, http.StatusOK, "ok")
	}
	if err := p.dispatchInboundEnvelope(ctx, cfg.instanceID, mapped.Envelope); err != nil {
		return linearWebhookDispatchHTTPError(err)
	}
	return writeWebhookText(w, http.StatusOK, "ok")
}

func (p *linearProvider) handleLinearAgentSessionWebhook(
	ctx context.Context,
	w http.ResponseWriter,
	candidates []resolvedInstanceConfig,
	request bridgesdk.WebhookRequest,
) error {
	payload := linearAgentSessionWebhookPayload{}
	if err := json.Unmarshal(request.Body, &payload); err != nil {
		return &bridgesdk.HTTPError{
			StatusCode: http.StatusBadRequest,
			Message:    "invalid linear agent session payload",
		}
	}

	cfg, ok, err := selectLinearConfig(candidates, payload.OrganizationID, linearModeAgentSessions)
	if err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
	}
	if !ok {
		return writeWebhookText(w, http.StatusOK, "ignored")
	}

	mapped, ignored, err := mapLinearAgentSessionEvent(payload, cfg.managed, request.ReceivedAt, cfg.botUserID)
	if err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
	}
	if ignored || cfg.dedup.Mark(mapped.Envelope.IdempotencyKey) {
		return writeWebhookText(w, http.StatusOK, "ok")
	}
	if err := p.dispatchInboundEnvelope(ctx, cfg.instanceID, mapped.Envelope); err != nil {
		return linearWebhookDispatchHTTPError(err)
	}
	return writeWebhookText(w, http.StatusOK, "ok")
}

func linearWebhookDispatchHTTPError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return &bridgesdk.HTTPError{
			StatusCode: http.StatusGatewayTimeout,
			Message:    "linear: webhook ingestion timed out",
		}
	case errors.Is(err, context.Canceled):
		return &bridgesdk.HTTPError{
			StatusCode: http.StatusServiceUnavailable,
			Message:    "linear: webhook ingestion canceled",
		}
	default:
		return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
	}
}

func linearSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(secret)))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func normalizeDeliveryEventType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func resolveLinearRemoteMessageID(
	reference *bridgepkg.DeliveryMessageReference,
	state deliveryState,
	snapshot *bridgepkg.DeliverySnapshot,
) string {
	if reference != nil && strings.TrimSpace(reference.RemoteMessageID) != "" {
		return strings.TrimSpace(reference.RemoteMessageID)
	}
	if strings.TrimSpace(state.RemoteMessageID) != "" {
		return strings.TrimSpace(state.RemoteMessageID)
	}
	if snapshot != nil {
		return strings.TrimSpace(snapshot.RemoteMessageID)
	}
	return ""
}

func computeLinearAppendDelta(previous string, current string) string {
	if previous == "" {
		return current
	}
	if strings.HasPrefix(current, previous) {
		return current[len(previous):]
	}
	return current
}

func shouldSkipLinearCommentDelivery(event bridgepkg.DeliveryEvent, _ string, body string) bool {
	return normalizeDeliveryEventType(event.EventType) == bridgepkg.DeliveryEventTypeResume &&
		strings.TrimSpace(body) == ""
}

func validateLinearAgentSessionDelivery(event bridgepkg.DeliveryEvent) error {
	if event.Operation.Normalize() == bridgepkg.DeliveryOperationDelete ||
		normalizeDeliveryEventType(event.EventType) == bridgepkg.DeliveryEventTypeDelete {
		return errors.New("linear: agent session activities are append-only and cannot be deleted")
	}
	if event.Operation.Normalize() == bridgepkg.DeliveryOperationEdit {
		return errors.New("linear: agent session activities are append-only and cannot be edited")
	}
	return nil
}

func decodeLinearAgentSessionThread(event bridgepkg.DeliveryEvent) (linearThreadRef, error) {
	thread, err := decodeLinearThreadID(firstNonEmpty(event.DeliveryTarget.ThreadID, event.RoutingKey.ThreadID))
	if err != nil {
		return linearThreadRef{}, err
	}
	if strings.TrimSpace(thread.AgentSessionID) == "" {
		return linearThreadRef{}, errors.New("linear: agent_sessions mode requires an agent session thread id")
	}
	return thread, nil
}

func resumeLinearAgentSessionDelivery(
	event bridgepkg.DeliveryEvent,
	state deliveryState,
	remoteID string,
) (bridgepkg.DeliveryAck, deliveryState, bool) {
	if normalizeDeliveryEventType(event.EventType) != bridgepkg.DeliveryEventTypeResume ||
		strings.TrimSpace(remoteID) == "" {
		return bridgepkg.DeliveryAck{}, state, false
	}

	next := state
	next.LastSeq = event.Seq
	next.LastContent = event.Content.Text
	next.RemoteMessageID = remoteID
	ack := bridgepkg.DeliveryAck{
		DeliveryID:      event.DeliveryID,
		Seq:             event.Seq,
		RemoteMessageID: remoteID,
	}
	return ack, next, true
}

func shouldProcessLinearAgentSessionAction(action string) bool {
	switch action {
	case "created", "prompted":
		return true
	default:
		return false
	}
}

func isUnexpectedLinearBotUser(botUserID string, payload linearAgentSessionWebhookPayload) bool {
	appUserID := strings.TrimSpace(firstNonEmpty(payload.AgentSession.AppUserID, payload.AppUserID))
	if strings.TrimSpace(botUserID) == "" || appUserID == "" {
		return false
	}
	return appUserID != strings.TrimSpace(botUserID)
}

func mapLinearAgentSessionMessage(
	action string,
	payload linearAgentSessionWebhookPayload,
) (string, string, bridgepkg.MessageSender, error) {
	switch action {
	case "created":
		if payload.AgentSession.Comment == nil {
			return "", "", bridgepkg.MessageSender{}, errors.New(
				"linear: created agent session event is missing comment payload",
			)
		}
		return strings.TrimSpace(payload.AgentSession.Comment.ID),
			strings.TrimSpace(payload.AgentSession.Comment.Body),
			bridgepkg.MessageSender{
				ID:          firstNonEmpty(actorID(payload.AgentSession.Creator), payload.Actor.ID),
				Username:    linearUserName(actorURL(payload.AgentSession.Creator)),
				DisplayName: firstNonEmpty(actorName(payload.AgentSession.Creator), payload.Actor.Name),
			},
			nil
	case "prompted":
		if payload.AgentActivity == nil {
			return "", "", bridgepkg.MessageSender{}, errors.New(
				"linear: prompted agent session event is missing agentActivity",
			)
		}
		return strings.TrimSpace(firstNonEmpty(payload.AgentSession.SourceCommentID, payload.AgentSession.CommentID)),
			strings.TrimSpace(firstNonEmpty(payload.AgentActivity.Content.Body, payload.AgentActivity.Body)),
			bridgepkg.MessageSender{
				ID:          strings.TrimSpace(payload.Actor.ID),
				Username:    linearUserName(payload.Actor.URL),
				DisplayName: strings.TrimSpace(payload.Actor.Name),
			},
			nil
	default:
		return "", "", bridgepkg.MessageSender{}, errors.New("linear: unsupported agent session action")
	}
}

func encodeLinearThreadID(ref linearThreadRef) string {
	issueID := strings.TrimSpace(ref.IssueID)
	if strings.TrimSpace(ref.AgentSessionID) != "" {
		if strings.TrimSpace(ref.RootCommentID) != "" {
			return "linear:" + issueID + ":c:" + strings.TrimSpace(
				ref.RootCommentID,
			) + ":s:" + strings.TrimSpace(
				ref.AgentSessionID,
			)
		}
		return "linear:" + issueID + ":s:" + strings.TrimSpace(ref.AgentSessionID)
	}
	if strings.TrimSpace(ref.RootCommentID) != "" {
		return "linear:" + issueID + ":c:" + strings.TrimSpace(ref.RootCommentID)
	}
	return "linear:" + issueID
}

func decodeLinearThreadID(threadID string) (linearThreadRef, error) {
	trimmed := strings.TrimSpace(threadID)
	if trimmed == "" {
		return linearThreadRef{}, errors.New("linear: thread id is required")
	}

	if matches := linearCommentSessionThreadPattern.FindStringSubmatch(trimmed); len(matches) == 4 {
		return linearThreadRef{
			IssueID:        matches[1],
			RootCommentID:  matches[2],
			AgentSessionID: matches[3],
		}, nil
	}
	if matches := linearIssueSessionThreadPattern.FindStringSubmatch(trimmed); len(matches) == 3 {
		return linearThreadRef{
			IssueID:        matches[1],
			AgentSessionID: matches[2],
		}, nil
	}
	if matches := linearCommentThreadPattern.FindStringSubmatch(trimmed); len(matches) == 3 {
		return linearThreadRef{
			IssueID:       matches[1],
			RootCommentID: matches[2],
		}, nil
	}
	if matches := linearIssueThreadPattern.FindStringSubmatch(trimmed); len(matches) == 2 {
		return linearThreadRef{IssueID: matches[1]}, nil
	}
	return linearThreadRef{}, fmt.Errorf("linear: invalid thread id %q", trimmed)
}

func issueThreadIDFromGroup(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return "linear:" + trimmed
		}
	}
	return ""
}

func linearUserName(profileURL string) string {
	url := strings.TrimSpace(profileURL)
	if url == "" {
		return ""
	}
	parts := strings.Split(url, "/profiles/")
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func actorID(actor *linearActor) string {
	if actor == nil {
		return ""
	}
	return strings.TrimSpace(actor.ID)
}

func actorName(actor *linearActor) string {
	if actor == nil {
		return ""
	}
	return strings.TrimSpace(actor.Name)
}

func actorURL(actor *linearActor) string {
	if actor == nil {
		return ""
	}
	return strings.TrimSpace(actor.URL)
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
